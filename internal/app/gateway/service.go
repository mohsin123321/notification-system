package gateway

import (
	"context"

	gatewayport "github.com/mohsin123321/notification-system/internal/domain/gateway/port"
	"github.com/mohsin123321/notification-system/internal/domain/gateway/pure"
)

// Service is the gateway use-case layer. It sits between the HTTP handler and
// the outbound gRPC client, giving a place for gateway-level logic (validation,
// enrichment, rate limiting) without coupling the handler to infra.
type Service struct {
	client gatewayport.NotifierClient
}

func NewService(client gatewayport.NotifierClient) *Service {
	return &Service{client: client}
}

func (s *Service) Send(ctx context.Context, userID, notifType, idempotencyKey string, payload []byte) (*pure.SendResult, error) {
	return s.client.Send(ctx, userID, notifType, idempotencyKey, payload)
}

func (s *Service) Get(ctx context.Context, id string) (*pure.NotificationDetail, error) {
	return s.client.Get(ctx, id)
}
