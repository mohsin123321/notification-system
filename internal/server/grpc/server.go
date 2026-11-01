package grpcserver

import (
	"net"

	"google.golang.org/grpc"

	"github.com/mohsin123321/notification-system/internal/platform/lifecycle"
)

// App is implemented by each self-contained application that registers its
// gRPC handlers and exposes any background workers it owns.
type App interface {
	Register(srv *grpc.Server)
	Workers() []lifecycle.Component
}

// Server wraps grpc.Server and implements lifecycle.Component.
type Server struct {
	srv  *grpc.Server
	addr string
}

// New creates a new gRPC Server listening on addr.
func New(addr string, opts ...grpc.ServerOption) *Server {
	return &Server{srv: grpc.NewServer(opts...), addr: addr}
}

// RegisterApps calls Register on each app and returns all their workers so the
// caller can hand them to a lifecycle.Manager.
func (s *Server) RegisterApps(apps ...App) []lifecycle.Component {
	var workers []lifecycle.Component
	for _, a := range apps {
		a.Register(s.srv)
		workers = append(workers, a.Workers()...)
	}
	return workers
}

// GRPCServer returns the underlying *grpc.Server so callers can register services.
func (s *Server) GRPCServer() *grpc.Server { return s.srv }

// Name implements lifecycle.Component.
func (s *Server) Name() string { return "grpc-server-" + s.addr }

// Start implements lifecycle.Component.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	go s.srv.Serve(lis) //nolint:errcheck
	return nil
}

// Stop implements lifecycle.Component.
func (s *Server) Stop() error {
	s.srv.GracefulStop()
	return nil
}
