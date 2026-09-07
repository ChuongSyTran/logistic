package di

import (
	"context"
	"fmt"
	"log"

	"user_service/ent"
	"user_service/internal/adapter/grpcserver"
	"user_service/internal/adapter/persistence"
	"user_service/internal/app"
	"user_service/internal/conf"
	"user_service/internal/mapper/generated"

	pb "github.com/logistic/api/logistic/user_service/v1"
	"github.com/logistic/pkg/cache"
	"google.golang.org/grpc"
)

type Container struct {
	EntClient *ent.Client
	Cache     *cache.Client
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	if c.Cache != nil {
		if err := c.Cache.Close(); err != nil {
			log.Printf("[user_service] closing redis failed: %v", err)
		}
	}
	if c.EntClient != nil {
		if err := c.EntClient.Close(); err != nil {
			log.Printf("[user_service] closing ent client failed: %v", err)
		}
	}
}

func Injection(grpcServer *grpc.Server, cfg *conf.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("user_service: config is nil")
	}

	entClient, err := ent.Open(cfg.Database.Driver, cfg.Database.GetDataSource())
	if err != nil {
		return nil, fmt.Errorf("user_service: mở kết nối Postgres thất bại: %w", err)
	}

	if err := entClient.Schema.Create(context.Background()); err != nil {
		_ = entClient.Close()
		return nil, fmt.Errorf("user_service: tạo schema thất bại: %w", err)
	}

	var redisClient *cache.Client
	if cfg.Redis.Enabled {
		redisClient, err = cache.New(cache.Config{
			Host:     cfg.Redis.Host,
			Port:     cfg.Redis.Port,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
			Prefix:   cfg.Redis.Prefix,
		})
		if err != nil {
			log.Printf("[user_service] Redis không khả dụng (%v) — chạy tiếp KHÔNG cache", err)
			redisClient = nil
		} else {
			log.Printf("[user_service] Redis đã kết nối tại %s:%s (db=%d, prefix=%q)",
				cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB, cfg.Redis.Prefix)
		}
	} else {
		log.Printf("[user_service] Redis bị tắt bằng cấu hình — chạy KHÔNG cache")
	}

	appMapper := &generated.AppMapperImpl{}
	userRepo := persistence.NewUserRepo(entClient, redisClient, appMapper)
	userEngine := app.NewUserEngine(userRepo)
	userController := grpcserver.NewUserServer(userEngine, appMapper)

	pb.RegisterUserServiceServer(grpcServer, userController)

	return &Container{EntClient: entClient, Cache: redisClient}, nil
}
