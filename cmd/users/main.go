package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	preferenceapp "github.com/mohsin123321/notification-system/internal/app/preference"
	"github.com/mohsin123321/notification-system/internal/infra/memory"
	"github.com/mohsin123321/notification-system/internal/platform/config"
	"github.com/mohsin123321/notification-system/internal/platform/lifecycle"
	"github.com/mohsin123321/notification-system/internal/platform/logging"
	grpcserver "github.com/mohsin123321/notification-system/internal/server/grpc"
	"github.com/mohsin123321/notification-system/internal/server/grpc/interceptors"
)

func main() {
	cfg := config.Default()
	logger := logging.New("info")
	lm := lifecycle.NewManager()

	// Infrastructure (in-memory for early steps).
	prefRepo := memory.NewPreferenceRepo()

	// Self-contained app: owns service and handler.
	app := preferenceapp.New(prefRepo)

	grpcSrv := grpcserver.New(cfg.Users.GRPCAddr,
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRequestID(),
			interceptors.UnaryLogging(logger),
		),
		grpc.ChainStreamInterceptor(
			interceptors.StreamRequestID(),
			interceptors.StreamLogging(logger),
		),
	)

	workers := grpcSrv.RegisterApps(app)
	lm.Register(grpcSrv)
	for _, w := range workers {
		lm.Register(w)
	}

	if err := lm.StartAll(); err != nil {
		log.Fatalf("failed to start users service: %v", err)
	}

	log.Printf("users service listening on %s", cfg.Users.GRPCAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down users service...")
	lm.StopAll()
}
