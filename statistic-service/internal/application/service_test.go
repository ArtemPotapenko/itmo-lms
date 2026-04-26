package application

import (
	"context"
	"testing"
	"time"

	"itmo-lms/statistic-service/internal/domain"
)

func TestCreateAttemptEnrichesMetadata(t *testing.T) {
	repo := &fakeStatisticRepo{}
	metadata := fakeMetadataProvider{
		topics:     []string{"top_1"},
		tags:       []domain.TagScore{{TagID: "tag_1", Code: "disc", Weight: 0.7}},
		difficulty: 3,
	}
	service := NewService(repo, metadata)

	attempt, err := service.CreateAttempt(context.Background(), domain.Attempt{
		UserID:    "usr_1",
		ContentID: "tsk_1",
		Answer:    "2,3",
		IsCorrect: true,
	})
	if err != nil {
		t.Fatalf("CreateAttempt() error = %v", err)
	}

	if attempt.Source != "practice" {
		t.Fatalf("source = %q, want practice", attempt.Source)
	}
	if len(attempt.TopicIDs) != 1 || attempt.TopicIDs[0] != "top_1" {
		t.Fatalf("topic ids = %v, want [top_1]", attempt.TopicIDs)
	}
	if len(attempt.TagScores) != 1 || attempt.TagScores[0].TagID != "tag_1" {
		t.Fatalf("tag scores = %v, want tag_1", attempt.TagScores)
	}
	if attempt.Difficulty != 3 {
		t.Fatalf("difficulty = %d, want 3", attempt.Difficulty)
	}
	if repo.saved.ContentID != "tsk_1" {
		t.Fatalf("repo saved content id = %q", repo.saved.ContentID)
	}
}

func TestProfilePreservesWeightedTagStats(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeStatisticRepo{
		profile: domain.KnowledgeProfile{
			UserID: "usr_1",
			Topics: map[string]domain.TopicStat{
				"top_1": {UserID: "usr_1", TopicID: "top_1", Attempts: 2, Correct: 1, WeightedAttempts: 5, WeightedCorrect: 3, Accuracy: 0.5, Rating: 6, UpdatedAt: now},
			},
			Tags: map[string]domain.TagStat{
				"tag_1": {UserID: "usr_1", TagID: "tag_1", WeightedAttempts: 1.0, WeightedCorrect: 0.7, Mastery: 0.7, UpdatedAt: now},
			},
			UpdatedAt: now,
		},
	}
	service := NewService(repo, nil)

	profile, err := service.Profile(context.Background(), "usr_1")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	if got := profile.Tags["tag_1"].Mastery; got != 0.7 {
		t.Fatalf("mastery = %v, want 0.7", got)
	}
	if got := profile.Topics["top_1"].Accuracy; got != 0.5 {
		t.Fatalf("topic accuracy = %v, want 0.5", got)
	}
	if got := profile.Topics["top_1"].Rating; got != 6 {
		t.Fatalf("topic rating = %v, want 6", got)
	}
}

func TestCourseCalibrationBuildsRelativeDifficultyAndWeights(t *testing.T) {
	service := NewService(&fakeStatisticRepo{
		courseAttempts: []domain.Attempt{
			{
				ID:         "att_1",
				UserID:     "usr_1",
				CourseID:   "crs_1",
				ContentID:  "tsk_1",
				TopicIDs:   []string{"top_1", "top_2"},
				TagScores:  []domain.TagScore{{TagID: "tag_a", Weight: 0.7}, {TagID: "tag_b", Weight: 0.3}},
				Difficulty: 4,
				IsCorrect:  true,
			},
			{
				ID:         "att_2",
				UserID:     "usr_2",
				CourseID:   "crs_1",
				ContentID:  "tsk_1",
				TopicIDs:   []string{"top_1", "top_2"},
				TagScores:  []domain.TagScore{{TagID: "tag_a", Weight: 0.7}, {TagID: "tag_b", Weight: 0.3}},
				Difficulty: 4,
				IsCorrect:  false,
			},
			{
				ID:         "att_3",
				UserID:     "usr_1",
				CourseID:   "crs_1",
				ContentID:  "tsk_2",
				TopicIDs:   []string{"top_1"},
				TagScores:  []domain.TagScore{{TagID: "tag_c", Weight: 1}},
				Difficulty: 2,
				IsCorrect:  true,
			},
		},
	}, nil)

	calibration, err := service.CourseCalibration(context.Background(), "crs_1")
	if err != nil {
		t.Fatalf("CourseCalibration() error = %v", err)
	}

	task := calibration.TaskCalibrations["tsk_1"]
	if task.AttemptCount != 2 {
		t.Fatalf("attempt count = %d, want 2", task.AttemptCount)
	}
	if task.SuggestedDifficulty <= 0 || task.SuggestedDifficulty > 10 {
		t.Fatalf("suggested difficulty = %v", task.SuggestedDifficulty)
	}
	if len(task.TopicWeights) != 2 || len(task.TagWeights) != 2 {
		t.Fatalf("weights = %+v %+v", task.TopicWeights, task.TagWeights)
	}
}

type fakeMetadataProvider struct {
	topics     []string
	tags       []domain.TagScore
	difficulty int
}

func (f fakeMetadataProvider) ResolveTask(context.Context, string) ([]string, []domain.TagScore, int, error) {
	return f.topics, f.tags, f.difficulty, nil
}

type fakeStatisticRepo struct {
	saved          domain.Attempt
	profile        domain.KnowledgeProfile
	courseAttempts []domain.Attempt
}

func (r *fakeStatisticRepo) AddAttempt(_ context.Context, attempt domain.Attempt) error {
	r.saved = attempt
	return nil
}

func (r *fakeStatisticRepo) ListAttempts(_ context.Context, _ string) ([]domain.Attempt, error) {
	return nil, nil
}

func (r *fakeStatisticRepo) ListCourseAttempts(_ context.Context, _ string) ([]domain.Attempt, error) {
	return r.courseAttempts, nil
}

func (r *fakeStatisticRepo) Profile(_ context.Context, _ string) (domain.KnowledgeProfile, error) {
	return r.profile, nil
}
