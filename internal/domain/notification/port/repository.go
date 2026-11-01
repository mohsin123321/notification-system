package port

//go:generate go run go.uber.org/mock/mockgen -source=repository.go -destination=mock/mock_repository.go -package=mock

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

type NotificationRepository interface {
	Save(ctx context.Context, n *pure.Notification) error
	GetByID(ctx context.Context, id pure.NotificationID) (*pure.Notification, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*pure.Notification, error)
	UpdateDeliveryStatus(ctx context.Context, d *pure.Delivery) error
	PendingOutbox(ctx context.Context, limit int) ([]pure.OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, id pure.OutboxID) error
}
