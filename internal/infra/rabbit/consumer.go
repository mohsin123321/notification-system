package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// Consumer implements notification/port.Consumer using RabbitMQ.
type Consumer struct {
	ch     *amqp.Channel
	logger *slog.Logger
}

// NewConsumer creates a Consumer backed by the given AMQP channel.
// Each Consumer should have its own dedicated channel (not shared with publishers).
func NewConsumer(ch *amqp.Channel, logger *slog.Logger) *Consumer {
	return &Consumer{ch: ch, logger: logger}
}

// Consume starts consuming messages from queue, calling handler for each one.
// It sets QoS to 10 messages and runs in the caller's goroutine (blocking until
// the context is cancelled or the channel is closed).
func (c *Consumer) Consume(ctx context.Context, queue string, handler func(context.Context, pure.DeliveryMessage) error) error {
	if err := c.ch.Qos(10, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	deliveries, err := c.ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", queue, err)
	}

	for {
		select {
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			var msg pure.DeliveryMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				c.logger.Error("failed to unmarshal delivery message", "queue", queue, "error", err)
				d.Nack(false, false) //nolint:errcheck
				continue
			}
			if err := handler(ctx, msg); err != nil {
				c.logger.Error("handler error", "queue", queue, "error", err)
				d.Nack(false, true) //nolint:errcheck
				continue
			}
			d.Ack(false) //nolint:errcheck
		case <-ctx.Done():
			return nil
		}
	}
}
