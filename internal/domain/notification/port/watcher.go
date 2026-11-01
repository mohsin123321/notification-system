package port

//go:generate go run go.uber.org/mock/mockgen -source=watcher.go -destination=mock/mock_watcher.go -package=mock

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

type StatusWatcher interface {
	Notify(ctx context.Context, event pure.StatusEvent) error
	Subscribe(ctx context.Context, id pure.NotificationID) (<-chan pure.StatusEvent, error)
	Unsubscribe(ctx context.Context, id pure.NotificationID, ch <-chan pure.StatusEvent)
}
