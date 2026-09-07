package di

import (
	"context"
	"fmt"
	"log"

	"vehicle_service/ent"
	"vehicle_service/internal/adapter/grpcserver"
	"vehicle_service/internal/adapter/persistence"
	"vehicle_service/internal/app"
	"vehicle_service/internal/conf"
	"vehicle_service/internal/mapper/generated"

	pb "github.com/logistic/api/logistic/vehicle_service/v1"
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
			log.Printf("[vehicle_service] closing redis failed: %v", err)
		}
	}
	if c.EntClient != nil {
		if err := c.EntClient.Close(); err != nil {
			log.Printf("[vehicle_service] closing ent client failed: %v", err)
		}
	}
}

func Injection(grpcServer *grpc.Server, cfg *conf.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("vehicle_service: config is nil")
	}

	entClient, err := ent.Open(cfg.Database.Driver, cfg.Database.GetDataSource())
	if err != nil {
		return nil, fmt.Errorf("vehicle_service: mở kết nối Postgres thất bại: %w", err)
	}

	if err := entClient.Schema.Create(context.Background()); err != nil {
		_ = entClient.Close()
		return nil, fmt.Errorf("vehicle_service: tạo schema thất bại: %w", err)
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
			log.Printf("[vehicle_service] CẢNH BÁO: Redis không khả dụng (%v) — "+
				"tìm xe gần đây sẽ chạy đường dự phòng chậm hơn", err)
			redisClient = nil
		} else {
			log.Printf("[vehicle_service] Redis đã kết nối tại %s:%s (db=%d, prefix=%q) — chỉ mục GEO sẵn sàng",
				cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB, cfg.Redis.Prefix)
		}
	} else {
		log.Printf("[vehicle_service] Redis bị tắt bằng cấu hình — dùng đường dự phòng Postgres")
	}

	appMapper := &generated.AppMapperImpl{}
	vehicleRepo := persistence.NewVehicleRepo(entClient, redisClient, appMapper)
	vehicleEngine := app.NewVehicleEngine(vehicleRepo)
	vehicleServer := grpcserver.NewVehicleServer(vehicleEngine, appMapper)

	pb.RegisterVehicleServiceServer(grpcServer, vehicleServer)

	return &Container{EntClient: entClient, Cache: redisClient}, nil
}