package domain

import "context"

type Repository interface {
	SaveJob(context.Context, DocumentJob) error
	UpdateJob(context.Context, DocumentJob) error
	GetJob(context.Context, string) (DocumentJob, bool, error)
}
