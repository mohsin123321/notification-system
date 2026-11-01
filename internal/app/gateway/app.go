package gateway

import gatewayport "github.com/mohsin123321/notification-system/internal/domain/gateway/port"

// App is the self-contained gateway bounded context. It owns the service,
// handler, and route registration.
type App struct {
	handler *handler
}

// New wires the gateway app. The concrete port.NotifierClient (infra adapter)
// is injected by cmd/gateway and never referenced inside this package.
func New(client gatewayport.NotifierClient) *App {
	svc := NewService(client)
	return &App{handler: newHandler(svc)}
}
