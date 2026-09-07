package main

import (
	"context"
	"log"
	"matching_service/internal/conf"
	"matching_service/internal/di"
	"net"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/logistic/pkg/middleware"
	"github.com/logistic/pkg/tracer"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	cfg        *conf.Config
	container  *di.Container
	shutdown   func(context.Context) error
}

func NewApp(cfg *conf.Config) (*App, error) {
	shutdownTracer, err := tracer.InitTracer("matching_service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+cfg.Server.GrpcPort)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		middleware.ChainForService("matching_service"),
	)

	container, err := di.Injection(grpcServer, cfg)
	if err != nil {
		return nil, err
	}

	return &App{
		grpcServer: grpcServer,
		listener:   lis,
		cfg:        cfg,
		container:  container,
		shutdown:   shutdownTracer,
	}, nil
}

func (a *App) Start() error {
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Stop() {
	log.Println("Stopping Matching Service (gRPC) gracefully...")
	a.grpcServer.GracefulStop()
	a.container.Close()

	if a.shutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer: %v", err)
		}
	}

	log.Println("Matching Service (gRPC) stopped.")
}