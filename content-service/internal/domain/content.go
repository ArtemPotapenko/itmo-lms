package domain

import "time"

type Topic struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Title     string    `json:"title"`
	Order     int       `json:"order"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type TaskTag struct {
	TagID  string  `json:"tag_id"`
	Code   string  `json:"code,omitempty"`
	Name   string  `json:"name,omitempty"`
	Kind   string  `json:"kind,omitempty"`
	Weight float64 `json:"weight"`
}

type Task struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	LatexBody     string    `json:"latex_body"`
	TopicIDs      []string  `json:"topic_ids"`
	Tags          []TaskTag `json:"tags"`
	Difficulty    int       `json:"difficulty"`
	CorrectAnswer string    `json:"correct_answer,omitempty"`
	Status        string    `json:"status"`
	AuthorID      string    `json:"author_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Theory struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	LatexBody string    `json:"latex_body"`
	Summary   string    `json:"summary"`
	TopicIDs  []string  `json:"topic_ids"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkItem struct {
	Order     int    `json:"order"`
	Kind      string `json:"kind"`
	ContentID string `json:"content_id"`
	Title     string `json:"title,omitempty"`
}

type WorkTemplate struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Items     []WorkItem `json:"items"`
	Status    string     `json:"status"`
	CreatedBy string     `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type WorkAnswer struct {
	TaskID string `json:"task_id"`
	Answer string `json:"answer"`
}

type WorkCheckResult struct {
	WorkID       string            `json:"work_id"`
	UserID       string            `json:"user_id"`
	CheckedAt    time.Time         `json:"checked_at"`
	Results      []TaskCheckResult `json:"results"`
	TotalTasks   int               `json:"total_tasks"`
	CorrectTasks int               `json:"correct_tasks"`
}

type TaskCheckResult struct {
	TaskID    string    `json:"task_id"`
	Title     string    `json:"title"`
	TopicIDs  []string  `json:"topic_ids"`
	Tags      []TaskTag `json:"tags"`
	Answer    string    `json:"answer"`
	IsCorrect bool      `json:"is_correct"`
}

type TaskScope struct {
	TopicIDs []string        `json:"topic_ids"`
	TagIDs   []string        `json:"tag_ids"`
	Tasks    []ScopedTaskDef `json:"tasks"`
	AuthorID string          `json:"author_id,omitempty"`
	Status   string          `json:"status,omitempty"`
}

type ScopedTaskDef struct {
	Title         string    `json:"title"`
	LatexBody     string    `json:"latex_body"`
	CorrectAnswer string    `json:"correct_answer,omitempty"`
	Difficulty    int       `json:"difficulty,omitempty"`
	TopicIDs      []string  `json:"topic_ids,omitempty"`
	TagIDs        []string  `json:"tag_ids,omitempty"`
	Tags          []TaskTag `json:"tags,omitempty"`
}
