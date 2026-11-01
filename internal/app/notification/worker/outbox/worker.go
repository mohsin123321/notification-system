package outbox

import (
	"context"
	"log/slog"
	"time"

	notificationport "github.com/mohsin123321/notification-system/internal/domain/notification/port"
)

// Worker polls the outbox table and publishes pending messages to the broker.
// It implements lifecycle.Component.
type Worker struct {
	repo   notificationport.NotificationRepository
	pub    notificationport.Publisher
	done   chan struct{}
	logger *slog.Logger
}

// New creates a new outbox Worker.
func New(repo notificationport.NotificationRepository, pub notificationport.Publisher, logger *slog.Logger) *Worker {
	return &Worker{
		repo:   repo,
		pub:    pub,
		done:   make(chan struct{}),
		logger: logger,
	}
}

// Name implements lifecycle.Component.
func (w *Worker) Name() string { return "outbox-worker" }

// Start implements lifecycle.Component.
func (w *Worker) Start() error {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.flush()
			case <-w.done:
				return
			}
		}
	}()
	return nil
}

// Stop implements lifecycle.Component.
func (w *Worker) Stop() error {
	close(w.done)
	return nil
}

func (w *Worker) flush() {
	ctx := context.Background()
	msgs, err := w.repo.PendingOutbox(ctx, 50)
	if err != nil {
		w.logger.Error("pending outbox error", "error", err)
		return
	}
	for _, msg := range msgs {
		if err := w.pub.Publish(ctx, msg); err != nil {
			w.logger.Error("publish outbox message error", "id", msg.ID, "error", err)
			continue
		}
		if err := w.repo.MarkOutboxPublished(ctx, msg.ID); err != nil {
			w.logger.Error("mark outbox published error", "id", msg.ID, "error", err)
		}
	}
}
