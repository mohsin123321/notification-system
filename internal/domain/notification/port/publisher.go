package port

//go:generate go run go.uber.org/mock/mockgen -source=publisher.go -destination=mock/mock_publisher.go -package=mock

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

type Publisher interface {
	Publish(ctx context.Context, msg pure.OutboxMessage) error
}
