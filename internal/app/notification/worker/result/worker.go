package result

import (
	"context"
	"log/slog"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
	notifservice "github.com/mohsin123321/notification-system/internal/app/notification/service"
	"github.com/mohsin123321/notification-system/internal/infra/rabbit"
)

// Worker consumes status events from the notification.events exchange and
// updates the notification service (persists status + notifies watchers).
// It implements lifecycle.Component.
type Worker struct {
	consumer *rabbit.ResultConsumer
	svc      *notifservice.Service
	logger   *slog.Logger
	cancel   context.CancelFunc
}

// New creates a result Worker.
func New(consumer *rabbit.ResultConsumer, svc *notifservice.Service, logger *slog.Logger) *Worker {
	return &Worker{consumer: consumer, svc: svc, logger: logger}
}

// Name implements lifecycle.Component.
func (w *Worker) Name() string { return "result-worker" }

// Start implements lifecycle.Component.
func (w *Worker) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go func() {
		if err := w.consumer.Consume(ctx, w.handle); err != nil {
			w.logger.Error("result consumer stopped", "error", err)
		}
	}()
	return nil
}

// Stop implements lifecycle.Component.
func (w *Worker) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

func (w *Worker) handle(ctx context.Context, event pure.StatusEvent) error {
	if err := w.svc.UpdateDeliveryStatus(ctx, event); err != nil {
		w.logger.Error("update delivery status error",
			"notification_id", event.NotificationID,
			"delivery_id", event.DeliveryID,
			"error", err,
		)
		return err
	}
	return nil
}
