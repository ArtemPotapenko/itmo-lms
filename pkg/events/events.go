package events

type AttemptEvaluated struct {
	UserID    string `json:"user_id"`
	ContentID string `json:"content_id"`
	Answer    string `json:"answer"`
	IsCorrect bool   `json:"is_correct"`
	Source    string `json:"source"`
}
