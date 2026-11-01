package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	notificationapp "github.com/mohsin123321/notification-system/internal/app/notification"
	"github.com/mohsin123321/notification-system/internal/infra/memory"
	"github.com/mohsin123321/notification-system/internal/platform/config"
	"github.com/mohsin123321/notification-system/internal/platform/lifecycle"
	"github.com/mohsin123321/notification-system/internal/platform/logging"
	grpcserver "github.com/mohsin123321/notification-system/internal/server/grpc"
	"github.com/mohsin123321/notification-system/internal/server/grpc/interceptors"
)

func main() {
	cfg, err := config.LoadConfigFromFlag()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger := logging.New("info")
	lm := lifecycle.NewManager()

	// Infrastructure (in-memory for early steps; swap for postgres/rabbit later).
	repo := memory.NewRepository()
	prefClient := memory.NewPreferenceClient()

	// Self-contained app: owns service, handler, and any workers.
	app := notificationapp.New(repo, prefClient, nil, nil, logger)

	grpcSrv := grpcserver.New(cfg.Notifier.GRPCAddr,
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
		log.Fatalf("failed to start notifier: %v", err)
	}

	log.Printf("notifier listening on %s", cfg.Notifier.GRPCAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down notifier...")
	lm.StopAll()
}
