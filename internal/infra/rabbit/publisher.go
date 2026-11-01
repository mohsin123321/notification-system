package rabbit

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// Publisher implements notification/port.Publisher using RabbitMQ.
// It publishes to the "notifications" topic exchange.
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher creates a Publisher backed by the given AMQP channel.
// Each Publisher should have its own dedicated channel (not shared with consumers).
func NewPublisher(ch *amqp.Channel) *Publisher {
	return &Publisher{ch: ch}
}

// Publish sends an outbox message to the notifications exchange.
func (p *Publisher) Publish(ctx context.Context, msg pure.OutboxMessage) error {
	return p.ch.PublishWithContext(ctx, "notifications", msg.RoutingKey, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         msg.Body,
	})
}
