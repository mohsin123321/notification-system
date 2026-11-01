package httpserver

import (
	"context"
	"net/http"
)

// App is implemented by each self-contained application that registers its
// HTTP routes onto the shared mux.
type App interface {
	Register(mux *http.ServeMux)
}

// Server wraps http.Server and implements lifecycle.Component.
type Server struct {
	srv *http.Server
	mux *http.ServeMux
}

// New creates a new HTTP Server listening on addr.
func New(addr string) *Server {
	mux := http.NewServeMux()
	return &Server{mux: mux, srv: &http.Server{Addr: addr, Handler: mux}}
}

// RegisterApps calls Register on each app, wiring their routes into the mux.
func (s *Server) RegisterApps(apps ...App) {
	for _, a := range apps {
		a.Register(s.mux)
	}
}

// Name implements lifecycle.Component.
func (s *Server) Name() string { return "http-server" }

// Start implements lifecycle.Component.
func (s *Server) Start() error {
	go s.srv.ListenAndServe() //nolint:errcheck
	return nil
}

// Stop implements lifecycle.Component.
func (s *Server) Stop() error {
	return s.srv.Shutdown(context.Background())
}
