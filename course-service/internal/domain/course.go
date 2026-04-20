package domain

import "time"

type Course struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	OwnerID   string    `json:"owner_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CourseMember struct {
	CourseID string `json:"course_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
}

type Assignment struct {
	ID         string    `json:"id"`
	CourseID   string    `json:"course_id"`
	Title      string    `json:"title"`
	WorkID     string    `json:"work_id,omitempty"`
	TaskIDs    []string  `json:"task_ids"`
	DueAt      time.Time `json:"due_at,omitempty"`
	AssignedBy string    `json:"assigned_by"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type Submission struct {
	ID           string             `json:"id"`
	AssignmentID string             `json:"assignment_id"`
	UserID       string             `json:"user_id"`
	Answers      []SubmissionAnswer `json:"answers"`
	Status       string             `json:"status"`
	SubmittedAt  time.Time          `json:"submitted_at"`
	Review       *TeacherReview     `json:"review,omitempty"`
}

type SubmissionAnswer struct {
	ContentID string `json:"content_id"`
	Answer    string `json:"answer"`
	IsCorrect *bool  `json:"is_correct,omitempty"`
}

type TeacherReview struct {
	ReviewerID string    `json:"reviewer_id"`
	Score      *int      `json:"score,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	ReviewedAt time.Time `json:"reviewed_at"`
}
