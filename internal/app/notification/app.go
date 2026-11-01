package notification

import (
	"log/slog"

	"google.golang.org/grpc"

	notifservice "github.com/mohsin123321/notification-system/internal/app/notification/service"
	notificationport "github.com/mohsin123321/notification-system/internal/domain/notification/port"
	notificationv1 "github.com/mohsin123321/notification-system/internal/gen/proto/notification/v1"
	"github.com/mohsin123321/notification-system/internal/platform/lifecycle"
)

// App is the self-contained notification bounded context. It owns its service,
// handler, and any background workers passed at construction time.
type App struct {
	svc     *notifservice.Service
	handler *handler
	workers []lifecycle.Component
}

// New wires the notification service and handler from the given ports. Pass any
// background workers (outbox relay, channel workers, result worker) as the
// variadic tail; they are returned by Workers() so the caller can register them
// with a lifecycle.Manager.
func New(
	repo notificationport.NotificationRepository,
	prefs notificationport.PreferenceClient,
	pub notificationport.Publisher,
	watcher notificationport.StatusWatcher,
	_ *slog.Logger,
	workers ...lifecycle.Component,
) *App {
	svc := notifservice.New(repo, prefs, pub, watcher)
	return &App{
		svc:     svc,
		handler: newHandler(svc),
		workers: workers,
	}
}

// Service returns the underlying service so callers can pass it to workers that
// need it (e.g. the result worker).
func (a *App) Service() *notifservice.Service { return a.svc }

// Register implements grpcserver.App: registers the notification handler with
// the given gRPC server.
func (a *App) Register(srv *grpc.Server) {
	notificationv1.RegisterNotificationServiceServer(srv, a.handler)
}

// Workers implements grpcserver.App: returns the background workers owned by
// this app.
func (a *App) Workers() []lifecycle.Component { return a.workers }
