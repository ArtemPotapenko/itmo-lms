package domain

import "context"

type UserRepository interface {
	Create(context.Context, User) error
	FindByPhone(context.Context, string) (User, bool, error)
	FindByID(context.Context, string) (User, bool, error)
	List(context.Context) ([]User, error)
	SeedAdmin(context.Context, User) error
}
