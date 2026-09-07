package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"auth_service/internal/conf"
	"auth_service/internal/di"

	"github.com/logistic/pkg/middleware"
	"github.com/logistic/pkg/tracer"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type App struct {
	grpcServer *grpc.Server
	cfg        *conf.Config
	shutdown   func(context.Context) error
}

func NewApp(cfg *conf.Config) (*App, error) {
	shutdownTracer, err := tracer.InitTracer("auth_service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),

		middleware.ChainForService("auth_service"),
	)

	if err := di.Injection(grpcServer, cfg); err != nil {
		return nil, err
	}

	return &App{grpcServer: grpcServer, cfg: cfg, shutdown: shutdownTracer}, nil
}

func (a *App) Start() error {
	listener, err := net.Listen("tcp", ":"+a.cfg.Server.GrpcPort)
	if err != nil {
		return err
	}

	log.Printf("Starting Auth Service (gRPC) on :%s...", a.cfg.Server.GrpcPort)

	go func() {
		if err := a.grpcServer.Serve(listener); err != nil {
			log.Fatalf("Auth Service crashed: %v", err)
		}
	}()
	return nil
}

func (a *App) Stop() {
	log.Println("Stopping Auth Service (gRPC) gracefully...")
	a.grpcServer.GracefulStop()

	if a.shutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer: %v", err)
		}
	}

	log.Println("Auth Service (gRPC) stopped.")
}