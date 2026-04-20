package application

import (
	"context"
	"errors"
	"time"

	"itmo-lms/pkg/platform"
	"itmo-lms/statistic-service/internal/domain"
)

type MetadataProvider interface {
	ResolveTask(context.Context, string) ([]string, []domain.TagScore, int, error)
}

type Service struct {
	repo     domain.Repository
	metadata MetadataProvider
}

func NewService(repo domain.Repository, metadata MetadataProvider) *Service {
	return &Service{repo: repo, metadata: metadata}
}

func (s *Service) CreateAttempt(ctx context.Context, attempt domain.Attempt) (domain.Attempt, error) {
	if attempt.UserID == "" || attempt.ContentID == "" {
		return domain.Attempt{}, errors.New("user_id and content_id are required")
	}
	if attempt.Source == "" {
		attempt.Source = "practice"
	}
	if s.metadata != nil {
		topicIDs, tagScores, difficulty, err := s.metadata.ResolveTask(ctx, attempt.ContentID)
		if err != nil {
			return domain.Attempt{}, err
		}
		attempt.TopicIDs = topicIDs
		attempt.TagScores = tagScores
		attempt.Difficulty = difficulty
	}
	if attempt.Difficulty <= 0 {
		attempt.Difficulty = 1
	}
	attempt.ID = platform.NewID("att")
	attempt.CreatedAt = time.Now().UTC()
	return attempt, s.repo.AddAttempt(ctx, attempt)
}

func (s *Service) ListAttempts(ctx context.Context, userID string) ([]domain.Attempt, error) {
	return s.repo.ListAttempts(ctx, userID)
}

func (s *Service) Profile(ctx context.Context, userID string) (domain.KnowledgeProfile, error) {
	return s.repo.Profile(ctx, userID)
}
