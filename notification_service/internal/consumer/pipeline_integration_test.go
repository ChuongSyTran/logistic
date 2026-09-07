//go:build integration

package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"notification_service/ent"
	entnotification "notification_service/ent/notification"
	"notification_service/internal/biz"
	"notification_service/internal/mapper/generated"
	"notification_service/internal/repo"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/logistic/pkg/cache"
	"github.com/logistic/pkg/events"
	"github.com/logistic/pkg/mq"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func setupPipeline(t *testing.T) (*ent.Client, *mq.Publisher, *mq.Consumer, biz.NotificationEngine, *cache.Client) {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("IT_PG_HOST", "127.0.0.1"),
		env("IT_PG_PORT", "5432"),
		env("IT_PG_USER", "notif"),
		env("IT_PG_PASSWORD", "notif"),
		env("IT_PG_DB", "notif_test"),
	)
	entClient, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("mở Postgres thất bại: %v", err)
	}
	if err := entClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("tạo schema thất bại: %v", err)
	}

	redisClient, err := cache.New(cache.Config{
		Host:   env("IT_REDIS_HOST", "127.0.0.1"),
		Port:   env("IT_REDIS_PORT", "6379"),
		DB:     9,
		Prefix: "it-notif",
	})
	if err != nil {
		t.Fatalf("kết nối Redis thất bại: %v", err)
	}

	conn, err := mq.Connect(mq.Config{
		Host:     env("IT_MQ_HOST", "127.0.0.1"),
		Port:     env("IT_MQ_PORT", "5672"),
		User:     env("IT_MQ_USER", "guest"),
		Password: env("IT_MQ_PASSWORD", "guest"),
	})
	if err != nil {
		t.Fatalf("kết nối RabbitMQ thất bại: %v", err)
	}

	exchange := "it.logistic.events." + uuid.NewString()[:8]
	queue := "it.notification.events." + uuid.NewString()[:8]

	publisher, err := mq.NewPublisher(conn, exchange, "matching_service")
	if err != nil {
		t.Fatalf("tạo publisher thất bại: %v", err)
	}

	consumer, err := mq.NewConsumer(conn, mq.ConsumerConfig{
		Exchange:    exchange,
		Queue:       queue,
		BindingKeys: []string{"matching.#"},
		Prefetch:    10,
	})
	if err != nil {
		t.Fatalf("tạo consumer thất bại: %v", err)
	}

	appMapper := &generated.AppMapperImpl{}
	engine := biz.NewNotificationEngine(repo.NewNotificationRepo(entClient, redisClient, appMapper))

	t.Cleanup(func() {
		_ = consumer.Close()
		_ = publisher.Close()
		_ = conn.Close()
		_ = redisClient.Close()
		_ = entClient.Close()
	})

	return entClient, publisher, consumer, engine, redisClient
}

func publishMatchFound(t *testing.T, pub *mq.Publisher, eventID string, payload events.MatchFound) {
	t.Helper()

	blob, _ := json.Marshal(payload)
	var data map[string]any
	if err := json.Unmarshal(blob, &data); err != nil {
		t.Fatalf("chuyển payload sang map thất bại: %v", err)
	}

	env := events.Envelope{
		EventID:    eventID,
		EventType:  events.RoutingKeyMatchFound,
		OccurredAt: time.Now().UTC(),
		Source:     "matching_service",
		Data:       data,
	}

	if err := pub.Publish(context.Background(), events.RoutingKeyMatchFound, eventID, env); err != nil {
		t.Fatalf("publish thất bại: %v", err)
	}
}

func TestPipelineEndToEnd(t *testing.T) {
	entClient, publisher, consumer, engine, redisClient := setupPipeline(t)

	shipperID := uuid.New()
	driverID := uuid.New()
	contractID := uuid.NewString()

	handler := NewMatchingConsumer(engine)
	done := make(chan struct{})
	processed := 0

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		_ = consumer.Start(ctx, func(c context.Context, d mq.Delivery) error {
			err := handler.Handle(c, d)
			processed++
			if processed == 1 {
				close(done)
			}
			return err
		})
	}()

	time.Sleep(500 * time.Millisecond)

	publishMatchFound(t, publisher, uuid.NewString(), events.MatchFound{
		ContractID:       contractID,
		BidID:            uuid.NewString(),
		AskID:            uuid.NewString(),
		ShipperID:        shipperID.String(),
		DriverID:         driverID.String(),
		VehicleID:        uuid.NewString(),
		ConsensusPrice:   3_200_000,
		ConsensusDeposit: 320_000,
	})

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("hết thời gian chờ consumer xử lý message")
	}

	time.Sleep(300 * time.Millisecond)

	notifs, err := entClient.Notification.Query().
		Where(entnotification.RefIDEQ(contractID)).
		All(context.Background())
	if err != nil {
		t.Fatalf("truy vấn thông báo thất bại: %v", err)
	}

	if len(notifs) != 2 {
		t.Fatalf("mong đợi 2 thông báo (chủ hàng + tài xế), nhận %d", len(notifs))
	}

	recipients := map[uuid.UUID]string{}
	for _, n := range notifs {
		recipients[n.UserID] = string(n.RecipientRole)
	}
	if recipients[shipperID] != "shipper" {
		t.Errorf("chủ hàng %s có vai trò %q", shipperID, recipients[shipperID])
	}
	if recipients[driverID] != "driver" {
		t.Errorf("tài xế %s có vai trò %q", driverID, recipients[driverID])
	}

	unread, err := engine.GetUnreadCount(context.Background(), shipperID)
	if err != nil {
		t.Fatalf("đếm chưa đọc thất bại: %v", err)
	}
	if unread != 1 {
		t.Errorf("chủ hàng có %d thông báo chưa đọc, mong đợi 1", unread)
	}

	_ = redisClient
}

func TestPipelineIsIdempotent(t *testing.T) {
	entClient, publisher, consumer, engine, _ := setupPipeline(t)

	contractID := uuid.NewString()
	eventID := uuid.NewString()

	handler := NewMatchingConsumer(engine)
	processedTwice := make(chan struct{})
	count := 0

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		_ = consumer.Start(ctx, func(c context.Context, d mq.Delivery) error {
			err := handler.Handle(c, d)
			count++
			if count == 2 {
				close(processedTwice)
			}
			return err
		})
	}()

	time.Sleep(500 * time.Millisecond)

	payload := events.MatchFound{
		ContractID:     contractID,
		BidID:          uuid.NewString(),
		AskID:          uuid.NewString(),
		ShipperID:      uuid.NewString(),
		DriverID:       uuid.NewString(),
		ConsensusPrice: 1_000_000,
	}

	publishMatchFound(t, publisher, eventID, payload)
	publishMatchFound(t, publisher, eventID, payload)

	select {
	case <-processedTwice:
	case <-ctx.Done():
		t.Fatal("hết thời gian chờ consumer xử lý hai message")
	}

	time.Sleep(300 * time.Millisecond)

	notifs, err := entClient.Notification.Query().
		Where(entnotification.RefIDEQ(contractID)).
		All(context.Background())
	if err != nil {
		t.Fatalf("truy vấn thông báo thất bại: %v", err)
	}

	if len(notifs) != 2 {
		t.Fatalf("gửi trùng phải cho ra 2 thông báo, nhận %d — bảng processed_events không chặn được message lặp",
			len(notifs))
	}
}