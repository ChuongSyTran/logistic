package di

import (
	"context"
	"fmt"
	"log"

	"kyc_service/ent"
	"kyc_service/internal/adapter/grpcserver"
	"kyc_service/internal/adapter/persistence"
	"kyc_service/internal/app"
	"kyc_service/internal/conf"
	"kyc_service/internal/mapper"

	pb "github.com/logistic/api/logistic/kyc_service/v1"
	"google.golang.org/grpc"
)

type Container struct {
	EntClient *ent.Client
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	if c.EntClient != nil {
		if err := c.EntClient.Close(); err != nil {
			log.Printf("[kyc_service] closing ent client failed: %v", err)
		}
	}
}

func Injection(grpcServer *grpc.Server, cfg *conf.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kyc_service: config is nil")
	}

	entClient, err := ent.Open(cfg.Database.Driver, cfg.Database.GetDataSource())
	if err != nil {
		return nil, fmt.Errorf("kyc_service: mở kết nối Postgres thất bại: %w", err)
	}

	if err := entClient.Schema.Create(context.Background()); err != nil {
		_ = entClient.Close()
		return nil, fmt.Errorf("kyc_service: tạo schema thất bại: %w", err)
	}

	appMapper := mapper.NewAppMapper()
	kycRepo := persistence.NewKycRepo(entClient)
	kycUseCase := app.NewKycUseCase(kycRepo)
	kycController := grpcserver.NewKycServer(kycUseCase, appMapper)

	pb.RegisterKycServiceServer(grpcServer, kycController)

	return &Container{EntClient: entClient}, nil
}
