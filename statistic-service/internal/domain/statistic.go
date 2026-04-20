package domain

import "time"

type Attempt struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	ContentID  string     `json:"content_id"`
	TopicIDs   []string   `json:"topic_ids"`
	TagScores  []TagScore `json:"tag_scores"`
	Difficulty int        `json:"difficulty"`
	Answer     string     `json:"answer"`
	IsCorrect  bool       `json:"is_correct"`
	Source     string     `json:"source"`
	CreatedAt  time.Time  `json:"created_at"`
}

type TagScore struct {
	TagID  string  `json:"tag_id"`
	Code   string  `json:"code,omitempty"`
	Name   string  `json:"name,omitempty"`
	Kind   string  `json:"kind,omitempty"`
	Weight float64 `json:"weight"`
}

type TopicStat struct {
	UserID           string    `json:"user_id"`
	TopicID          string    `json:"topic_id"`
	Attempts         int       `json:"attempts"`
	Correct          int       `json:"correct"`
	WeightedAttempts float64   `json:"weighted_attempts"`
	WeightedCorrect  float64   `json:"weighted_correct"`
	Accuracy         float64   `json:"accuracy"`
	Rating           float64   `json:"rating"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TagStat struct {
	UserID           string    `json:"user_id"`
	TagID            string    `json:"tag_id"`
	Code             string    `json:"code,omitempty"`
	Name             string    `json:"name,omitempty"`
	Kind             string    `json:"kind,omitempty"`
	WeightedAttempts float64   `json:"weighted_attempts"`
	WeightedCorrect  float64   `json:"weighted_correct"`
	Mastery          float64   `json:"mastery"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type KnowledgeProfile struct {
	UserID    string               `json:"user_id"`
	Topics    map[string]TopicStat `json:"topics"`
	Tags      map[string]TagStat   `json:"tags"`
	UpdatedAt time.Time            `json:"updated_at"`
}
