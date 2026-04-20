package kafka

import (
	"net/http"

	"itmo-lms/pkg/events"
	basekafka "itmo-lms/pkg/kafka"
)

type AttemptPublisher struct {
	publisher *basekafka.Publisher
}

func NewAttemptPublisher(publisher *basekafka.Publisher) *AttemptPublisher {
	return &AttemptPublisher{publisher: publisher}
}

func (p *AttemptPublisher) PublishAttempt(r *http.Request, event events.AttemptEvaluated) error {
	return p.publisher.PublishJSON(r.Context(), event.ContentID, event)
}
