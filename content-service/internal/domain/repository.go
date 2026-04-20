package domain

import "context"

type Repository interface {
	CreateTopic(context.Context, Topic) error
	ListTopics(context.Context) ([]Topic, error)
	CreateTag(context.Context, Tag) error
	ListTags(context.Context) ([]Tag, error)
	GetTag(context.Context, string) (Tag, bool, error)
	CreateTask(context.Context, Task) error
	ListTasks(context.Context, string) ([]Task, error)
	GetTask(context.Context, string) (Task, bool, error)
	CreateTheory(context.Context, Theory) error
	ListTheory(context.Context, string) ([]Theory, error)
	GetTheory(context.Context, string) (Theory, bool, error)
	CreateWork(context.Context, WorkTemplate) error
	ListWorks(context.Context) ([]WorkTemplate, error)
	GetWork(context.Context, string) (WorkTemplate, bool, error)
}
