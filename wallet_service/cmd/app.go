package main

import (
	"context"
	"log"
	"net"
	"time"
	"wallet_service/internal/adapter/grpcserver"
	"wallet_service/internal/conf"
	"wallet_service/internal/di"

	"github.com/logistic/pkg/tracer"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	cleanup    func()
	shutdown   func(context.Context) error
}

func NewApp(ctx context.Context, cfg *conf.Config) (*App, error) {
	shutdownTracer, err := tracer.InitTracer("wallet_service", cfg.Telemetry.OtlpEndpoint)
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+cfg.Server.GrpcPort)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpcserver.ErrorHandlerInterceptor(),
		),
	)

	cleanup, err := di.Injection(ctx, grpcServer, cfg)
	if err != nil {
		return nil, err
	}

	return &App{
		grpcServer: grpcServer,
		listener:   lis,
		cleanup:    cleanup,
		shutdown:   shutdownTracer,
	}, nil
}

func (a *App) Start() error {
	log.Printf("[Wallet] gRPC server listening on %s", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Stop() {
	log.Println("[Wallet] Stopping gracefully...")
	a.grpcServer.GracefulStop()

	if a.cleanup != nil {
		a.cleanup()
	}

	if a.shutdown != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer: %v", err)
		}
	}

	log.Println("[Wallet] Stopped.")
}
