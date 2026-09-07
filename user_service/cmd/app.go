package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"user_service/internal/conf"
	"user_service/internal/di"

	_ "github.com/lib/pq"
	"github.com/logistic/pkg/middleware"
	"github.com/logistic/pkg/tracer"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type App struct {
	container *di.Container
	server    *grpc.Server
	shutdown  func(context.Context) error
	cfg       *conf.Config
}

func NewApp(cfg *conf.Config) (*App, error) {
	shutdownTracer, err := tracer.InitTracer("user_service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		middleware.ChainForService("user_service"),
	)

	container, err := di.Injection(grpcServer, cfg)
	if err != nil {
		return nil, err
	}

	return &App{
		container: container,
		server:    grpcServer,
		shutdown:  shutdownTracer,
		cfg:       cfg,
	}, nil
}

func (a *App) Run() error {
	lis, err := net.Listen("tcp", ":"+a.cfg.Server.GrpcPort)
	if err != nil {
		return err
	}
	log.Printf("UserService is listening on %v", lis.Addr())
	return a.server.Serve(lis)
}

func (a *App) Stop() {
	log.Println("Stopping UserService gracefully...")
	a.server.GracefulStop()
	a.container.Close()

	if a.shutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer: %v", err)
		}
	}
	log.Println("UserService stopped.")
}