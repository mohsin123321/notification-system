package inapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	notificationport "github.com/mohsin123321/notification-system/internal/domain/notification/port"
	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
	templatepure "github.com/mohsin123321/notification-system/internal/domain/template/pure"
)

// Worker consumes in-app delivery messages and publishes them via MQTT using
// the built-in amq.topic exchange in RabbitMQ.
// It implements lifecycle.Component.
type Worker struct {
	consumer notificationport.Consumer
	amqpCh   *amqp.Channel
	logger   *slog.Logger
	done     chan struct{}
}

// New creates a new in-app Worker.
func New(consumer notificationport.Consumer, amqpCh *amqp.Channel, logger *slog.Logger) *Worker {
	return &Worker{
		consumer: consumer,
		amqpCh:   amqpCh,
		logger:   logger,
		done:     make(chan struct{}),
	}
}

// Name implements lifecycle.Component.
func (w *Worker) Name() string { return "inapp-worker" }

// Start implements lifecycle.Component.
func (w *Worker) Start() error {
	go w.consumer.Consume(context.Background(), "q.inapp", w.handle) //nolint:errcheck
	return nil
}

// Stop implements lifecycle.Component.
func (w *Worker) Stop() error {
	close(w.done)
	return nil
}

func (w *Worker) handle(ctx context.Context, msg pure.DeliveryMessage) error {
	body, _ := templatepure.Render(string(msg.Type), "inapp", msg.Payload)

	// Publish the notification to the device via the built-in amq.topic exchange.
	// RabbitMQ MQTT plugin routes messages from amq.topic to subscribed MQTT clients.
	mqttTopic := fmt.Sprintf("users.%s.notifications", string(msg.UserID))
	notifPayload, _ := json.Marshal(map[string]any{
		"notification_id": msg.NotificationID,
		"type":            msg.Type,
		"body":            body,
	})

	if err := w.amqpCh.PublishWithContext(ctx, "amq.topic", mqttTopic, false, false,
		amqp.Publishing{
			Body:        notifPayload,
			ContentType: "application/json",
		}); err != nil {
		return fmt.Errorf("publish to amq.topic: %w", err)
	}

	// Publish badge count update (stubbed as a simple counter).
	badgeTopic := fmt.Sprintf("users.%s.badge", string(msg.UserID))
	badgePayload, _ := json.Marshal(map[string]any{"increment": 1})
	if err := w.amqpCh.PublishWithContext(ctx, "amq.topic", badgeTopic, true, false,
		amqp.Publishing{
			Body:        badgePayload,
			ContentType: "application/json",
		}); err != nil {
		w.logger.Warn("failed to publish badge update", "error", err)
	}

	w.logger.Info("in-app notification sent",
		"notification_id", msg.NotificationID,
		"user_id", msg.UserID,
		"topic", mqttTopic,
	)

	return w.publishResult(ctx, msg, pure.StatusSent, "")
}

func (w *Worker) publishResult(ctx context.Context, msg pure.DeliveryMessage, status pure.DeliveryStatus, errStr string) error {
	event := pure.StatusEvent{
		NotificationID: msg.NotificationID,
		DeliveryID:     msg.DeliveryID,
		Channel:        msg.Channel,
		Status:         status,
		Error:          errStr,
		UpdatedAt:      time.Now(),
	}
	body, _ := json.Marshal(event)
	return w.amqpCh.PublishWithContext(ctx, "notification.events",
		fmt.Sprintf("result.inapp.%s", msg.NotificationID),
		false, false,
		amqp.Publishing{
			Body:         body,
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
		})
}
