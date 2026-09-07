package di

import (
	"context"
	"fmt"
	"log"
	"strings"

	"notification_service/ent"
	"notification_service/internal/biz"
	"notification_service/internal/conf"
	"notification_service/internal/consumer"
	"notification_service/internal/controller"
	"notification_service/internal/mapper/generated"
	"notification_service/internal/repo"

	pb "github.com/logistic/api/logistic/notification_service/v1"
	"github.com/logistic/pkg/cache"
	"github.com/logistic/pkg/mq"
	"google.golang.org/grpc"
)

type Container struct {
	EntClient *ent.Client
	Cache     *cache.Client
	MQConn    *mq.Connection
	Consumer  *mq.Consumer

	handler *consumer.MatchingConsumer
}

func (c *Container) StartConsumer(ctx context.Context) error {
	if c == nil || c.Consumer == nil || c.handler == nil {
		return nil
	}
	return c.Consumer.Start(ctx, c.handler.Handle)
}

func (c *Container) HasConsumer() bool {
	return c != nil && c.Consumer != nil
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	if c.Consumer != nil {
		if err := c.Consumer.Close(); err != nil {
			log.Printf("[notification_service] closing consumer failed: %v", err)
		}
	}
	if c.MQConn != nil {
		if err := c.MQConn.Close(); err != nil {
			log.Printf("[notification_service] closing rabbitmq failed: %v", err)
		}
	}
	if c.Cache != nil {
		if err := c.Cache.Close(); err != nil {
			log.Printf("[notification_service] closing redis failed: %v", err)
		}
	}
	if c.EntClient != nil {
		if err := c.EntClient.Close(); err != nil {
			log.Printf("[notification_service] closing ent client failed: %v", err)
		}
	}
}

func Injection(grpcServer *grpc.Server, cfg *conf.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("notification_service: config is nil")
	}

	entClient, err := ent.Open(cfg.Database.Driver, cfg.Database.GetDataSource())
	if err != nil {
		return nil, fmt.Errorf("notification_service: mở kết nối Postgres thất bại: %w", err)
	}

	if err := entClient.Schema.Create(context.Background()); err != nil {
		_ = entClient.Close()
		return nil, fmt.Errorf("notification_service: tạo schema thất bại: %w", err)
	}

	seedDefaultTemplates(context.Background(), entClient)

	container := &Container{EntClient: entClient}

	if cfg.Redis.Enabled {
		redisClient, rErr := cache.New(cache.Config{
			Host:     cfg.Redis.Host,
			Port:     cfg.Redis.Port,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
			Prefix:   cfg.Redis.Prefix,
		})
		if rErr != nil {
			log.Printf("[notification_service] Redis không khả dụng (%v) — bộ đếm chưa đọc sẽ đếm thẳng ở DB", rErr)
		} else {
			container.Cache = redisClient
			log.Printf("[notification_service] Redis đã kết nối tại %s:%s (db=%d)",
				cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)
		}
	}

	appMapper := &generated.AppMapperImpl{}
	notifRepo := repo.NewNotificationRepo(entClient, container.Cache, appMapper)
	notifEngine := biz.NewNotificationEngine(notifRepo)
	notifController := controller.NewNotificationController(notifEngine, appMapper)

	pb.RegisterNotificationServiceServer(grpcServer, notifController)

	if cfg.RabbitMQ.Enabled {
		if err := setupConsumer(container, cfg, notifEngine); err != nil {
			log.Printf("[notification_service] NGHIÊM TRỌNG: không dựng được consumer RabbitMQ (%v) — "+
				"sẽ KHÔNG có thông báo mới nào được tạo cho tới khi kết nối lại", err)
		}
	} else {
		log.Printf("[notification_service] Consumer RabbitMQ bị tắt bằng cấu hình")
	}

	return container, nil
}

func setupConsumer(container *Container, cfg *conf.Config, engine biz.NotificationEngine) error {
	conn, err := mq.Connect(mq.Config{
		Host:     cfg.RabbitMQ.Host,
		Port:     cfg.RabbitMQ.Port,
		User:     cfg.RabbitMQ.User,
		Password: cfg.RabbitMQ.Password,
		VHost:    cfg.RabbitMQ.VHost,
	})
	if err != nil {
		return err
	}
	container.MQConn = conn

	bindings := splitAndTrim(cfg.RabbitMQ.BindingKeys)
	mqConsumer, err := mq.NewConsumer(conn, mq.ConsumerConfig{
		Exchange:    cfg.RabbitMQ.Exchange,
		Queue:       cfg.RabbitMQ.Queue,
		BindingKeys: bindings,
		Prefetch:    cfg.RabbitMQ.Prefetch,
	})
	if err != nil {
		return err
	}

	container.Consumer = mqConsumer
	container.handler = consumer.NewMatchingConsumer(engine)

	log.Printf("[notification_service] RabbitMQ đã sẵn sàng: exchange=%s queue=%s bindings=%v",
		cfg.RabbitMQ.Exchange, cfg.RabbitMQ.Queue, bindings)
	return nil
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
