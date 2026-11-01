package port

import (
	"context"

	"github.com/mohsin123321/notification-system/internal/domain/gateway/pure"
)

// NotifierClient is the outbound port for calling the notification service.
// The concrete adapter lives in infra/grpc/gatewayclient.
type NotifierClient interface {
	Send(ctx context.Context, userID, notifType, idempotencyKey string, payload []byte) (*pure.SendResult, error)
	Get(ctx context.Context, id string) (*pure.NotificationDetail, error)
}
