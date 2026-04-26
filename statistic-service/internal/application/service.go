package application

import (
	"context"
	"encoding/json"
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
	cache    Cache
	cacheTTL time.Duration
}

type Cache interface {
	Get(context.Context, string) ([]byte, bool, error)
	SetEX(context.Context, string, time.Duration, []byte) error
	Delete(context.Context, ...string) error
}

func NewService(repo domain.Repository, metadata MetadataProvider, cache Cache, cacheTTL time.Duration) *Service {
	if cacheTTL <= 0 {
		cacheTTL = 2 * time.Hour
	}
	return &Service{repo: repo, metadata: metadata, cache: cache, cacheTTL: cacheTTL}
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
	if err := s.repo.AddAttempt(ctx, attempt); err != nil {
		return domain.Attempt{}, err
	}
	s.invalidateCaches(ctx, attempt.UserID, attempt.CourseID)
	return attempt, nil
}

func (s *Service) ListAttempts(ctx context.Context, userID string) ([]domain.Attempt, error) {
	return s.repo.ListAttempts(ctx, userID)
}

func (s *Service) Profile(ctx context.Context, userID string) (domain.KnowledgeProfile, error) {
	if userID == "" {
		return domain.KnowledgeProfile{}, errors.New("user_id is required")
	}
	key := "profile:" + userID
	if cached, ok, err := s.cacheGetJSON(ctx, key); err == nil && ok {
		var profile domain.KnowledgeProfile
		if json.Unmarshal(cached, &profile) == nil {
			return profile, nil
		}
	}
	profile, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return domain.KnowledgeProfile{}, err
	}
	_ = s.cacheSetJSON(ctx, key, profile)
	return profile, nil
}

func (s *Service) CourseCalibration(ctx context.Context, courseID string) (domain.CourseCalibration, error) {
	if courseID == "" {
		return domain.CourseCalibration{}, errors.New("course_id is required")
	}
	key := "course-calibration:" + courseID
	if cached, ok, err := s.cacheGetJSON(ctx, key); err == nil && ok {
		var calibration domain.CourseCalibration
		if json.Unmarshal(cached, &calibration) == nil {
			return calibration, nil
		}
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
		calibration := domain.CourseCalibration{CourseID: courseID, TaskCalibrations: map[string]domain.TaskCalibration{}, UpdatedAt: now}
		_ = s.cacheSetJSON(ctx, key, calibration)
		return calibration, nil
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
	courseUserTopicRatings := userTopicRatings(attempts)
	courseUserTagMasteries := userTagMasteries(attempts)

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
			TopicWeights:        calibrationTopicWeights(items, courseUserTopicRatings),
			TagWeights:          calibrationTagWeights(items, courseUserTagMasteries),
		}
	}
	calibration := domain.CourseCalibration{CourseID: courseID, TaskCalibrations: taskCalibrations, UpdatedAt: now}
	_ = s.cacheSetJSON(ctx, key, calibration)
	return calibration, nil
}

func calibrationTopicWeights(items []domain.Attempt, courseUserRatings map[string]map[string]float64) []domain.CalibrationWeight {
	if len(items) == 0 {
		return nil
	}
	scores := map[string]float64{}
	for _, topicID := range items[0].TopicIDs {
		strongSolved := 0.0
		strongTotal := 0.0
		weakSolved := 0.0
		weakTotal := 0.0
		for _, attempt := range items {
			if courseUserRatings[attempt.UserID][topicID] >= 7 {
				strongTotal++
				if attempt.IsCorrect {
					strongSolved++
				}
				continue
			}
			weakTotal++
			if attempt.IsCorrect {
				weakSolved++
			}
		}
		scores[topicID] = discriminativeScore(strongSolved, strongTotal, weakSolved, weakTotal, 1)
	}
	return normalizeScores(scores)
}

func calibrationTagWeights(items []domain.Attempt, courseUserMastery map[string]map[string]float64) []domain.CalibrationWeight {
	if len(items) == 0 {
		return nil
	}
	scores := map[string]float64{}
	for _, tag := range items[0].TagScores {
		strongSolved := 0.0
		strongTotal := 0.0
		weakSolved := 0.0
		weakTotal := 0.0
		for _, attempt := range items {
			if courseUserMastery[attempt.UserID][tag.TagID] >= 0.65 {
				strongTotal++
				if attempt.IsCorrect {
					strongSolved++
				}
				continue
			}
			weakTotal++
			if attempt.IsCorrect {
				weakSolved++
			}
		}
		base := tag.Weight
		if base <= 0 {
			base = 1
		}
		scores[tag.TagID] = discriminativeScore(strongSolved, strongTotal, weakSolved, weakTotal, base)
	}
	return normalizeScores(scores)
}

func discriminativeScore(strongSolved, strongTotal, weakSolved, weakTotal, baseWeight float64) float64 {
	const (
		minScore   = 0.05
		minSamples = 6.0
	)
	if baseWeight <= 0 {
		baseWeight = 1
	}
	strongRate := safeRate(strongSolved, strongTotal)
	weakRate := safeRate(weakSolved, weakTotal)
	signal := strongRate - weakRate
	if signal < 0 {
		signal = 0
	}
	confidence := clamp((strongTotal+weakTotal)/minSamples, 0, 1)
	return minScore + confidence*signal + (1-confidence)*baseWeight
}

func safeRate(solved, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return solved / total
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

func (s *Service) cacheGetJSON(ctx context.Context, key string) ([]byte, bool, error) {
	if s.cache == nil {
		return nil, false, nil
	}
	return s.cache.Get(ctx, key)
}

func (s *Service) cacheSetJSON(ctx context.Context, key string, value any) error {
	if s.cache == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.cache.SetEX(ctx, key, s.cacheTTL, raw)
}

func (s *Service) invalidateCaches(ctx context.Context, userID, courseID string) {
	if s.cache == nil {
		return
	}
	keys := make([]string, 0, 2)
	if userID != "" {
		keys = append(keys, "profile:"+userID)
	}
	if courseID != "" {
		keys = append(keys, "course-calibration:"+courseID)
	}
	_ = s.cache.Delete(ctx, keys...)
}
