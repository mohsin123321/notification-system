package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// ResultConsumer consumes status events from the notification.events exchange.
// It creates an exclusive, auto-deleted queue per instance.
type ResultConsumer struct {
	ch     *amqp.Channel
	logger *slog.Logger
}

// NewResultConsumer creates a ResultConsumer using a dedicated AMQP channel.
func NewResultConsumer(ch *amqp.Channel, logger *slog.Logger) *ResultConsumer {
	return &ResultConsumer{ch: ch, logger: logger}
}

// Consume creates an exclusive queue bound to notification.events and calls
// handler for every StatusEvent received.
func (rc *ResultConsumer) Consume(ctx context.Context, handler func(context.Context, pure.StatusEvent) error) error {
	queueName := fmt.Sprintf("q.result.%s", uuid.New().String())

	q, err := rc.ch.QueueDeclare(queueName, false, true, true, false, nil)
	if err != nil {
		return fmt.Errorf("declare result queue: %w", err)
	}

	if err := rc.ch.QueueBind(q.Name, "result.#", "notification.events", false, nil); err != nil {
		return fmt.Errorf("bind result queue: %w", err)
	}

	if err := rc.ch.Qos(10, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	deliveries, err := rc.ch.Consume(q.Name, "", false, true, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume result queue: %w", err)
	}

	for {
		select {
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			var event pure.StatusEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				rc.logger.Error("failed to unmarshal status event", "error", err)
				d.Nack(false, false) //nolint:errcheck
				continue
			}
			if err := handler(ctx, event); err != nil {
				rc.logger.Error("result handler error", "error", err)
				d.Nack(false, true) //nolint:errcheck
				continue
			}
			d.Ack(false) //nolint:errcheck
		case <-ctx.Done():
			return nil
		}
	}
}
