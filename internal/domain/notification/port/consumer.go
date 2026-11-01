package port

//go:generate go run go.uber.org/mock/mockgen -source=consumer.go -destination=mock/mock_consumer.go -package=mock

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

type Consumer interface {
	Consume(ctx context.Context, queue string, handler func(ctx context.Context, msg pure.DeliveryMessage) error) error
}
