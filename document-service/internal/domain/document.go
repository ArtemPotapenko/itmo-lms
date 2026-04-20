package domain

import "time"

type CompileRequest struct {
	Title  string         `json:"title"`
	Tasks  []DocumentTask `json:"tasks"`
	Format string         `json:"format"`
}

type DocumentTask struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	LatexBody string `json:"latex_body"`
}

type DocumentJob struct {
	ID          string         `json:"id"`
	Format      string         `json:"format"`
	Status      string         `json:"status"`
	Files       []DocumentFile `json:"files"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt time.Time      `json:"completed_at,omitempty"`
}

type DocumentFile struct {
	ID         string    `json:"id"`
	JobID      string    `json:"job_id"`
	Kind       string    `json:"kind"`
	StorageKey string    `json:"storage_key"`
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	Checksum   string    `json:"checksum"`
	CreatedAt  time.Time `json:"created_at"`
}
