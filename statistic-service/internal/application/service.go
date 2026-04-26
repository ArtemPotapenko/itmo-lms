package application

import (
	"context"
	"errors"
	"sort"
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

func (s *Service) CourseCalibration(ctx context.Context, courseID string) (domain.CourseCalibration, error) {
	if courseID == "" {
		return domain.CourseCalibration{}, errors.New("course_id is required")
	}
	attempts, err := s.repo.ListCourseAttempts(ctx, courseID)
	if err != nil {
		return domain.CourseCalibration{}, err
	}
	now := time.Now().UTC()
	byTask := map[string][]domain.Attempt{}
	taskSuccessRates := map[string]float64{}
	for _, attempt := range attempts {
		if attempt.ContentID == "" {
			continue
		}
		byTask[attempt.ContentID] = append(byTask[attempt.ContentID], attempt)
	}
	if len(byTask) == 0 {
		return domain.CourseCalibration{CourseID: courseID, TaskCalibrations: map[string]domain.TaskCalibration{}, UpdatedAt: now}, nil
	}
	averageSuccess := 0.0
	for taskID, items := range byTask {
		correct := 0
		for _, item := range items {
			if item.IsCorrect {
				correct++
			}
		}
		rate := float64(correct) / float64(len(items))
		taskSuccessRates[taskID] = rate
		averageSuccess += rate
	}
	averageSuccess /= float64(len(byTask))

	taskCalibrations := make(map[string]domain.TaskCalibration, len(byTask))
	for taskID, items := range byTask {
		baseDifficulty := items[0].Difficulty
		if baseDifficulty <= 0 {
			baseDifficulty = 1
		}
		taskCalibrations[taskID] = domain.TaskCalibration{
			CourseID:            courseID,
			ContentID:           taskID,
			AttemptCount:        len(items),
			SuccessRate:         round4(taskSuccessRates[taskID]),
			CourseAverageRate:   round4(averageSuccess),
			BaseDifficulty:      baseDifficulty,
			SuggestedDifficulty: round4(clamp(float64(baseDifficulty)*(1+(averageSuccess-taskSuccessRates[taskID])), 1, 10)),
			TopicWeights:        calibrationTopicWeights(items),
			TagWeights:          calibrationTagWeights(items),
		}
	}
	return domain.CourseCalibration{CourseID: courseID, TaskCalibrations: taskCalibrations, UpdatedAt: now}, nil
}

func calibrationTopicWeights(items []domain.Attempt) []domain.CalibrationWeight {
	if len(items) == 0 {
		return nil
	}
	userRatings := userTopicRatings(items)
	scores := map[string]float64{}
	for _, topicID := range items[0].TopicIDs {
		solved := 0.0
		total := 0.0
		for _, attempt := range items {
			if userRatings[attempt.UserID][topicID] >= 7 {
				total++
				if attempt.IsCorrect {
					solved++
				}
			}
		}
		if total == 0 {
			scores[topicID] = 1
			continue
		}
		scores[topicID] = solved / total
	}
	return normalizeScores(scores)
}

func calibrationTagWeights(items []domain.Attempt) []domain.CalibrationWeight {
	if len(items) == 0 {
		return nil
	}
	userMastery := userTagMasteries(items)
	scores := map[string]float64{}
	for _, tag := range items[0].TagScores {
		solved := 0.0
		total := 0.0
		for _, attempt := range items {
			if userMastery[attempt.UserID][tag.TagID] >= 0.65 {
				total++
				if attempt.IsCorrect {
					solved++
				}
			}
		}
		if total == 0 {
			if tag.Weight > 0 {
				scores[tag.TagID] = tag.Weight
			} else {
				scores[tag.TagID] = 1
			}
			continue
		}
		scores[tag.TagID] = solved / total
	}
	return normalizeScores(scores)
}

func userTopicRatings(items []domain.Attempt) map[string]map[string]float64 {
	type agg struct{ correct, attempts float64 }
	raw := map[string]map[string]agg{}
	for _, attempt := range items {
		perUser := raw[attempt.UserID]
		if perUser == nil {
			perUser = map[string]agg{}
			raw[attempt.UserID] = perUser
		}
		weight := float64(attempt.Difficulty)
		if weight <= 0 {
			weight = 1
		}
		for _, topicID := range attempt.TopicIDs {
			entry := perUser[topicID]
			entry.attempts += weight
			if attempt.IsCorrect {
				entry.correct += weight
			}
			perUser[topicID] = entry
		}
	}
	out := map[string]map[string]float64{}
	for userID, perUser := range raw {
		out[userID] = map[string]float64{}
		for topicID, entry := range perUser {
			if entry.attempts > 0 {
				out[userID][topicID] = 10 * entry.correct / entry.attempts
			}
		}
	}
	return out
}

func userTagMasteries(items []domain.Attempt) map[string]map[string]float64 {
	type agg struct{ correct, attempts float64 }
	raw := map[string]map[string]agg{}
	for _, attempt := range items {
		perUser := raw[attempt.UserID]
		if perUser == nil {
			perUser = map[string]agg{}
			raw[attempt.UserID] = perUser
		}
		for _, tag := range attempt.TagScores {
			entry := perUser[tag.TagID]
			entry.attempts += tag.Weight
			if attempt.IsCorrect {
				entry.correct += tag.Weight
			}
			perUser[tag.TagID] = entry
		}
	}
	out := map[string]map[string]float64{}
	for userID, perUser := range raw {
		out[userID] = map[string]float64{}
		for tagID, entry := range perUser {
			if entry.attempts > 0 {
				out[userID][tagID] = entry.correct / entry.attempts
			}
		}
	}
	return out
}

func normalizeScores(scores map[string]float64) []domain.CalibrationWeight {
	keys := make([]string, 0, len(scores))
	total := 0.0
	for key, score := range scores {
		if score <= 0 {
			score = 0.0001
			scores[key] = score
		}
		keys = append(keys, key)
		total += score
	}
	sort.Strings(keys)
	out := make([]domain.CalibrationWeight, 0, len(keys))
	for _, key := range keys {
		out = append(out, domain.CalibrationWeight{ID: key, Weight: round4(scores[key] / total)})
	}
	return out
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func round4(value float64) float64 {
	const scale = 10000
	return float64(int(value*scale+0.5)) / scale
}
