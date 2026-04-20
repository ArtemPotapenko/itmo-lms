package kafka

import (
	"context"
	"encoding/json"

	"itmo-lms/pkg/events"
	"itmo-lms/statistic-service/internal/application"
	"itmo-lms/statistic-service/internal/domain"
)

type AttemptConsumer struct {
	consumer consumer
	service  *application.Service
}

type consumer interface {
	Consume(context.Context, func(context.Context, []byte) error) error
}

func NewAttemptConsumer(consumer consumer, service *application.Service) *AttemptConsumer {
	return &AttemptConsumer{consumer: consumer, service: service}
}

func (c *AttemptConsumer) Consume(ctx context.Context) error {
	return c.consumer.Consume(ctx, func(ctx context.Context, raw []byte) error {
		var event events.AttemptEvaluated
		if err := json.Unmarshal(raw, &event); err != nil {
			return err
		}
		_, err := c.service.CreateAttempt(ctx, domain.Attempt{
			UserID:    event.UserID,
			ContentID: event.ContentID,
			Answer:    event.Answer,
			IsCorrect: event.IsCorrect,
			Source:    event.Source,
		})
		return err
	})
}
