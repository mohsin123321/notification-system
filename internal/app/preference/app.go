package preference

import (
	"google.golang.org/grpc"

	prefservice "github.com/mohsin123321/notification-system/internal/app/preference/service"
	preferenceport "github.com/mohsin123321/notification-system/internal/domain/preference/port"
	usersv1 "github.com/mohsin123321/notification-system/internal/gen/proto/users/v1"
	"github.com/mohsin123321/notification-system/internal/platform/lifecycle"
)

// App is the self-contained preference bounded context. It owns its service
// and handler; it has no background workers.
type App struct {
	handler *handler
}

// New wires the preference service and handler from the given repo.
func New(repo preferenceport.PreferenceRepository) *App {
	svc := prefservice.New(repo)
	return &App{handler: newHandler(svc)}
}

// Register implements grpcserver.App: registers the users handler with the
// given gRPC server.
func (a *App) Register(srv *grpc.Server) {
	usersv1.RegisterUsersServiceServer(srv, a.handler)
}

// Workers implements grpcserver.App: preference app has no background workers.
func (a *App) Workers() []lifecycle.Component { return nil }
