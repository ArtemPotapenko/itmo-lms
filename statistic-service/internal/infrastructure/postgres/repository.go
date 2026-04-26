package postgres

import (
	"context"
	"database/sql"
	"time"

	"itmo-lms/pkg/postgres"
	"itmo-lms/statistic-service/internal/domain"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) AddAttempt(ctx context.Context, attempt domain.Attempt) error {
	_, err := r.db.ExecContext(ctx, `insert into attempts(id, user_id, course_id, content_id, topic_ids, tag_scores, difficulty, answer, is_correct, source, created_at) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		attempt.ID, attempt.UserID, attempt.CourseID, attempt.ContentID, postgres.Marshal(attempt.TopicIDs), postgres.Marshal(attempt.TagScores), attempt.Difficulty, attempt.Answer, attempt.IsCorrect, attempt.Source, attempt.CreatedAt)
	return err
}

func (r *Repository) ListAttempts(ctx context.Context, userID string) ([]domain.Attempt, error) {
	rows, err := r.db.QueryContext(ctx, `select id, user_id, course_id, content_id, topic_ids, tag_scores, difficulty, answer, is_correct, source, created_at from attempts where user_id=$1 order by created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Attempt
	for rows.Next() {
		var item domain.Attempt
		var topicRaw, tagRaw []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.CourseID, &item.ContentID, &topicRaw, &tagRaw, &item.Difficulty, &item.Answer, &item.IsCorrect, &item.Source, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.TopicIDs = postgres.Unmarshal[[]string](topicRaw)
		item.TagScores = postgres.Unmarshal[[]domain.TagScore](tagRaw)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListCourseAttempts(ctx context.Context, courseID string) ([]domain.Attempt, error) {
	rows, err := r.db.QueryContext(ctx, `select id, user_id, course_id, content_id, topic_ids, tag_scores, difficulty, answer, is_correct, source, created_at from attempts where course_id=$1 order by created_at`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Attempt
	for rows.Next() {
		var item domain.Attempt
		var topicRaw, tagRaw []byte
		if err := rows.Scan(&item.ID, &item.UserID, &item.CourseID, &item.ContentID, &topicRaw, &tagRaw, &item.Difficulty, &item.Answer, &item.IsCorrect, &item.Source, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.TopicIDs = postgres.Unmarshal[[]string](topicRaw)
		item.TagScores = postgres.Unmarshal[[]domain.TagScore](tagRaw)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Profile(ctx context.Context, userID string) (domain.KnowledgeProfile, error) {
	attempts, err := r.ListAttempts(ctx, userID)
	if err != nil {
		return domain.KnowledgeProfile{}, err
	}
	topics := map[string]domain.TopicStat{}
	tags := map[string]domain.TagStat{}
	now := time.Now().UTC()
	for _, attempt := range attempts {
		difficultyWeight := float64(attempt.Difficulty)
		if difficultyWeight <= 0 {
			difficultyWeight = 1
		}
		for _, topicID := range attempt.TopicIDs {
			stat := topics[topicID]
			stat.UserID = userID
			stat.TopicID = topicID
			stat.Attempts++
			stat.WeightedAttempts += difficultyWeight
			if attempt.IsCorrect {
				stat.Correct++
				stat.WeightedCorrect += difficultyWeight
			}
			stat.Accuracy = float64(stat.Correct) / float64(stat.Attempts)
			if stat.WeightedAttempts > 0 {
				stat.Rating = 10 * stat.WeightedCorrect / stat.WeightedAttempts
			}
			stat.UpdatedAt = now
			topics[topicID] = stat
		}
		for _, score := range attempt.TagScores {
			stat := tags[score.TagID]
			stat.UserID = userID
			stat.TagID = score.TagID
			stat.Code = score.Code
			stat.Name = score.Name
			stat.Kind = score.Kind
			stat.WeightedAttempts += score.Weight
			if attempt.IsCorrect {
				stat.WeightedCorrect += score.Weight
			}
			if stat.WeightedAttempts > 0 {
				stat.Mastery = stat.WeightedCorrect / stat.WeightedAttempts
			}
			stat.UpdatedAt = now
			tags[score.TagID] = stat
		}
	}
	return domain.KnowledgeProfile{UserID: userID, Topics: topics, Tags: tags, UpdatedAt: now}, nil
}
