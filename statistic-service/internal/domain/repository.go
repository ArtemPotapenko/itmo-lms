package domain

import "context"

type Repository interface {
	AddAttempt(context.Context, Attempt) error
	ListAttempts(context.Context, string) ([]Attempt, error)
	Profile(context.Context, string) (KnowledgeProfile, error)
}
