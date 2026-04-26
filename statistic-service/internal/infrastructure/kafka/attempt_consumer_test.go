package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"itmo-lms/pkg/events"
	"itmo-lms/statistic-service/internal/application"
	"itmo-lms/statistic-service/internal/domain"
)

func TestAttemptConsumerConsumesEventAndCreatesAttempt(t *testing.T) {
	repo := &consumerRepo{}
	service := application.NewService(repo, consumerMetadata{})

	raw, err := json.Marshal(events.AttemptEvaluated{
		UserID:    "usr_1",
		CourseID:  "crs_1",
		ContentID: "tsk_1",
		Answer:    "2,3",
		IsCorrect: true,
		Source:    "workbook",
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	consumer := NewAttemptConsumer(&fakeConsumer{messages: [][]byte{raw}}, service)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = consumer.Consume(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("Consume() error = %v", err)
	}

	attempts := repo.Attempts()
	if len(attempts) != 1 {
		t.Fatalf("saved attempts = %d, want 1", len(attempts))
	}
	attempt := attempts[0]
	if attempt.UserID != "usr_1" || attempt.ContentID != "tsk_1" || attempt.Source != "workbook" || !attempt.IsCorrect {
		t.Fatalf("attempt = %+v", attempt)
	}
	if attempt.CourseID != "crs_1" {
		t.Fatalf("course id = %q, want crs_1", attempt.CourseID)
	}
	if len(attempt.TopicIDs) != 1 || attempt.TopicIDs[0] != "top_1" {
		t.Fatalf("attempt topic ids = %+v", attempt.TopicIDs)
	}
	if len(attempt.TagScores) != 1 || attempt.TagScores[0].TagID != "tag_1" || attempt.TagScores[0].Weight != 0.7 {
		t.Fatalf("attempt tag scores = %+v", attempt.TagScores)
	}
}

func TestAttemptConsumerSkipsInvalidPayloadAndDoesNotCommit(t *testing.T) {
	repo := &consumerRepo{}
	service := application.NewService(repo, consumerMetadata{})
	base := &fakeConsumer{messages: [][]byte{[]byte("{invalid")}}

	consumer := NewAttemptConsumer(base, service)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := consumer.Consume(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("Consume() error = %v", err)
	}

	if len(repo.Attempts()) != 0 {
		t.Fatalf("attempts = %+v, want none", repo.Attempts())
	}
	if base.handled != 1 {
		t.Fatalf("handled messages = %d, want 1", base.handled)
	}
}

type fakeConsumer struct {
	messages [][]byte
	handled  int
}

func (c *fakeConsumer) Consume(ctx context.Context, handler func(context.Context, []byte) error) error {
	for _, message := range c.messages {
		c.handled++
		_ = handler(ctx, message)
	}
	<-ctx.Done()
	return ctx.Err()
}

type consumerMetadata struct{}

func (consumerMetadata) ResolveTask(context.Context, string) ([]string, []domain.TagScore, int, error) {
	return []string{"top_1"}, []domain.TagScore{{TagID: "tag_1", Code: "disc", Weight: 0.7}}, 3, nil
}

type consumerRepo struct {
	mu       sync.Mutex
	attempts []domain.Attempt
}

func (r *consumerRepo) AddAttempt(_ context.Context, attempt domain.Attempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts = append(r.attempts, attempt)
	return nil
}

func (r *consumerRepo) ListAttempts(_ context.Context, _ string) ([]domain.Attempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Attempt(nil), r.attempts...), nil
}

func (r *consumerRepo) ListCourseAttempts(_ context.Context, courseID string) ([]domain.Attempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Attempt, 0)
	for _, attempt := range r.attempts {
		if attempt.CourseID == courseID {
			out = append(out, attempt)
		}
	}
	return out, nil
}

func (r *consumerRepo) Profile(context.Context, string) (domain.KnowledgeProfile, error) {
	return domain.KnowledgeProfile{}, nil
}

func (r *consumerRepo) Attempts() []domain.Attempt {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Attempt(nil), r.attempts...)
}
