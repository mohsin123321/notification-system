package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	gatewayapp "github.com/mohsin123321/notification-system/internal/app/gateway"
	"github.com/mohsin123321/notification-system/internal/infra/grpc/gatewayclient"
	"github.com/mohsin123321/notification-system/internal/platform/config"
	"github.com/mohsin123321/notification-system/internal/platform/lifecycle"
	httpserver "github.com/mohsin123321/notification-system/internal/server/http"
)

func main() {
	cfg := config.Default()
	lm := lifecycle.NewManager()

	client, err := gatewayclient.New(cfg.Notifier.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to create gateway client: %v", err)
	}

	app := gatewayapp.New(client)

	httpSrv := httpserver.New(cfg.Gateway.Addr)
	httpSrv.RegisterApps(app)
	lm.Register(httpSrv)

	if err := lm.StartAll(); err != nil {
		log.Fatalf("failed to start gateway: %v", err)
	}

	log.Printf("gateway listening on %s", cfg.Gateway.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gateway...")
	lm.StopAll()
}
