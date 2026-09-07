package di

import (
	"context"
	"log"
	"strings"
	"time"

	"matching_service/internal/biz"
	"matching_service/internal/broker/kafka"
	"matching_service/internal/broker/nats_jetstream"
	brokerrabbit "matching_service/internal/broker/rabbitmq"
	walletclient "matching_service/internal/client/wallet"
	"matching_service/internal/conf"
	"matching_service/internal/controller"
	"matching_service/internal/mapper/generated"
	"matching_service/internal/repo"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	entclient "matching_service/internal/common/ent_client"

	pb "github.com/logistic/api/logistic/matching_service/v1"
	pbwallet "github.com/logistic/api/logistic/wallet_service/v1"
	"github.com/logistic/pkg/mq"
	"github.com/nats-io/nats.go"
)

type Container struct {
	MQConn      *mq.Connection
	MQPublisher *mq.Publisher
	NatsConn    *nats.Conn
}

func (c *Container) Close() {
	if c == nil {
		return
	}
	if c.NatsConn != nil {
		if err := c.NatsConn.Drain(); err != nil {
			log.Printf("[matching_service] draining nats failed: %v", err)
		}
	}
	if c.MQPublisher != nil {
		if err := c.MQPublisher.Close(); err != nil {
			log.Printf("[matching_service] closing rabbitmq publisher failed: %v", err)
		}
	}
	if c.MQConn != nil {
		if err := c.MQConn.Close(); err != nil {
			log.Printf("[matching_service] closing rabbitmq failed: %v", err)
		}
	}
}

const walletProbeTimeout = 3 * time.Second

// grpc.NewClient chỉ dựng client chứ chưa nối nên gần như không bao giờ lỗi; phải
// thăm dò thật mới biết có wallet_service hay không để còn rơi về ví giả lập.
func newWalletClient(addr string) biz.WalletClient {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[matching_service] CẢNH BÁO: địa chỉ wallet_service không dùng được (%v) — chuyển sang ví giả lập", err)
		return biz.NewMockWalletClient()
	}

	ctx, cancel := context.WithTimeout(context.Background(), walletProbeTimeout)
	defer cancel()

	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			log.Printf("[matching_service] wallet_service sẵn sàng tại %s", addr)
			return walletclient.NewGrpcClient(pbwallet.NewWalletServiceClient(conn))
		}
		if !conn.WaitForStateChange(ctx, state) {
			_ = conn.Close()
			log.Printf("[matching_service] CẢNH BÁO: không nối được wallet_service tại %s sau %s — "+
				"chuyển sang ví giả lập, mọi lần kiểm tra số dư sẽ luôn đạt", addr, walletProbeTimeout)
			return biz.NewMockWalletClient()
		}
	}
}

func Injection(grpcServer *grpc.Server, cfg *conf.Config) (*Container, error) {
	if cfg == nil {
		log.Fatalf("[SYSTEM] failed to read config")
	}

	masterClient, salveClient, err := entclient.NewConnection(&cfg.MasterDatabase, &cfg.SlaveDatabase)
	if err != nil {
		return nil, err
	}

	appMapper := &generated.MatchingMapperImpl{}

	matchingRepo := repo.NewMatchingRepo(masterClient, salveClient, appMapper)
	engine := biz.NewGeoHashEngine()

	natsConn, err := nats.Connect("nats://" + cfg.NatConfig.Host + ":" + cfg.NatConfig.Port)
	if err != nil {
		return nil, err
	}
	natsCtx, err := natsConn.JetStream()
	if err != nil {
		return nil, err
	}
	if err := nats_jetstream.EnsureStream(natsCtx); err != nil {
		return nil, err
	}
	natsPub := nats_jetstream.InitPublisher(natsCtx, appMapper)
	natsSub := nats_jetstream.InitSubcriber(natsCtx)

	brokers := strings.Split(cfg.KafkaConfig.Brokers, ",")
	kafkaPub, err := kafka.NewKafkaPublisher(brokers, appMapper)
	if err != nil {
		return nil, err
	}

	container := &Container{NatsConn: natsConn}

	var notifier biz.Notifier = biz.NoopNotifier{}
	if cfg.RabbitMQ.Enabled {
		mqConn, mqErr := mq.Connect(mq.Config{
			Host:     cfg.RabbitMQ.Host,
			Port:     cfg.RabbitMQ.Port,
			User:     cfg.RabbitMQ.User,
			Password: cfg.RabbitMQ.Password,
			VHost:    cfg.RabbitMQ.VHost,
		})
		if mqErr != nil {
			log.Printf("[matching_service] CẢNH BÁO: không kết nối được RabbitMQ (%v) — "+
				"người dùng sẽ KHÔNG nhận được thông báo ghép đơn", mqErr)
		} else {
			publisher, pErr := mq.NewPublisher(mqConn, cfg.RabbitMQ.Exchange, "matching_service")
			if pErr != nil {
				log.Printf("[matching_service] CẢNH BÁO: không tạo được publisher RabbitMQ (%v)", pErr)
				_ = mqConn.Close()
			} else {
				container.MQConn = mqConn
				container.MQPublisher = publisher
				notifier = brokerrabbit.NewNotifier(publisher)
				log.Printf("[matching_service] RabbitMQ sẵn sàng — exchange=%s", cfg.RabbitMQ.Exchange)
			}
		}
	} else {
		log.Printf("[matching_service] Publisher RabbitMQ bị tắt bằng cấu hình")
	}

	walletClient := newWalletClient(cfg.WalletService.GrpcAddr)

	matchingEngine := biz.NewMatchingEngine(matchingRepo, engine, walletClient, kafkaPub, natsPub, notifier)

	if err := nats_jetstream.StartOfferConsumer(context.Background(), natsSub, matchingEngine, appMapper); err != nil {
		return nil, err
	}

	matchingController := controller.NewMatchingController(matchingEngine)
	pb.RegisterMatchingEngineServiceServer(grpcServer, matchingController)

	return container, nil
}
