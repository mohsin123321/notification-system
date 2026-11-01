package sms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	notificationport "github.com/mohsin123321/notification-system/internal/domain/notification/port"
	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
	templatepure "github.com/mohsin123321/notification-system/internal/domain/template/pure"
)

const maxAttempts = 3

// Worker consumes SMS delivery messages and simulates sending them.
// It implements lifecycle.Component.
type Worker struct {
	consumer    notificationport.Consumer
	amqpCh      *amqp.Channel
	logger      *slog.Logger
	failureRate float64
	done        chan struct{}
}

// New creates a new SMS Worker.
func New(consumer notificationport.Consumer, amqpCh *amqp.Channel, logger *slog.Logger, failureRate float64) *Worker {
	return &Worker{
		consumer:    consumer,
		amqpCh:      amqpCh,
		logger:      logger,
		failureRate: failureRate,
		done:        make(chan struct{}),
	}
}

// Name implements lifecycle.Component.
func (w *Worker) Name() string { return "sms-worker" }

// Start implements lifecycle.Component.
func (w *Worker) Start() error {
	go w.consumer.Consume(context.Background(), "q.sms", w.handle) //nolint:errcheck
	return nil
}

// Stop implements lifecycle.Component.
func (w *Worker) Stop() error {
	close(w.done)
	return nil
}

func (w *Worker) handle(ctx context.Context, msg pure.DeliveryMessage) error {
	body, _ := templatepure.Render(string(msg.Type), "sms", msg.Payload)

	//nolint:gosec
	if rand.Float64() < w.failureRate {
		return w.handleFailure(ctx, msg, errors.New("simulated SMS provider failure"))
	}

	preview := body
	if len(preview) > 50 {
		preview = preview[:50]
	}
	w.logger.Info("sent SMS", "notification_id", msg.NotificationID, "body_preview", preview)

	return w.publishResult(ctx, msg, pure.StatusSent, "")
}

func (w *Worker) handleFailure(ctx context.Context, msg pure.DeliveryMessage, sendErr error) error {
	if msg.Attempt >= maxAttempts {
		return w.publishResult(ctx, msg, pure.StatusDead, sendErr.Error())
	}

	retryMsg := msg
	retryMsg.Attempt++
	body, _ := json.Marshal(retryMsg)
	return w.amqpCh.PublishWithContext(ctx, "", "q.sms.retry", false, false,
		amqp.Publishing{
			Body:         body,
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
		})
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
		fmt.Sprintf("result.sms.%s", msg.NotificationID),
		false, false,
		amqp.Publishing{
			Body:         body,
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
		})
}
