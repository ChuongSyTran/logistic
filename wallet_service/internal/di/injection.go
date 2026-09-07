package di

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"wallet_service/ent"
	"wallet_service/internal/adapter/grpcserver"
	"wallet_service/internal/adapter/kafka"
	"wallet_service/internal/adapter/persistence"
	"wallet_service/internal/adapter/search"
	"wallet_service/internal/app"
	"wallet_service/internal/conf"
	"wallet_service/internal/mapper"
	"wallet_service/internal/mapper/generated"

	pb "github.com/logistic/api/logistic/wallet_service/v1"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/logistic/pkg/concurrencyUtils"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

type HoldDepositPayload struct {
	DriverID   string  `json:"driver_id"`
	Amount     float64 `json:"amount"`
	ContractID string  `json:"contract_id"`
}

func Injection(ctx context.Context, grpcServer *grpc.Server, cfg *conf.Config) (func(), error) {
	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	db, err := entsql.Open(dialect.MySQL, cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	entClient := ent.NewClient(ent.Driver(db))

	esAddresses := strings.Split(cfg.ElasticSearch.Addresses, ",")
	if len(esAddresses) == 0 || esAddresses[0] == "" {
		esAddresses = []string{"http://localhost:9200"}
	}

	// Khai kiểu interface ngay từ đầu: nếu khai biến kiểu con trỏ cụ thể rồi gán
	// nil, khi truyền vào tham số interface nó sẽ thành interface KHÁC nil và
	// mọi kiểm tra `== nil` phía sau đều sai.
	var esEngine search.WalletSearchEngine
	esEngine, err = search.NewElasticSearchEngine(esAddresses, cfg.ElasticSearch.Username, cfg.ElasticSearch.Password)
	if err != nil {
		log.Printf("Failed to init elasticsearch: %v. Search will be disabled.", err)
		esEngine = nil
	} else if err := esEngine.EnsureIndices(ctx); err != nil {
		log.Printf("Failed to ensure elasticsearch indices: %v", err)
	}

	var walletMapper mapper.WalletMapper = &generated.WalletMapperImpl{}

	var indexer app.WalletIndexer
	if esEngine != nil {
		indexer = search.NewIndexer(esEngine)
	}

	uow := persistence.NewUnitOfWork(entClient)
	walletRepo := persistence.NewWalletRepo(entClient, walletMapper)
	txRepo := persistence.NewTransactionRepo(entClient, walletMapper)

	walletUseCase := app.NewWalletUseCase(uow, walletRepo, txRepo, indexer)

	walletServer := grpcserver.NewWalletServer(walletUseCase, esEngine, walletMapper)
	pb.RegisterWalletServiceServer(grpcServer, walletServer)

	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"localhost:9092"}
	}

	kafkaConsumer, err := kafka.NewKafkaConsumer(brokers, "wallet-service-group")
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	bufferSize := 100
	msgChan := make(chan []byte, bufferSize)

	holdDepositHandler := func(ctx context.Context, payload []byte) error {
		var event HoldDepositPayload
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Printf("[Wallet] Failed to unmarshal hold_deposit event: %v", err)
			return fmt.Errorf("%w: %v", app.ErrNonRetryable, err)
		}

		driverID, err := uuid.Parse(event.DriverID)
		if err != nil {
			return fmt.Errorf("%w: invalid driver_id %s", app.ErrNonRetryable, event.DriverID)
		}

		amount := int64(event.Amount * 100)

		if err := walletUseCase.HoldDeposit(ctx, driverID, amount, event.ContractID); err != nil {
			log.Printf("[Wallet] HoldDeposit failed: %v", err)
			return fmt.Errorf("%w: %v", app.ErrNonRetryable, err)
		}

		log.Printf("[Wallet] HoldDeposit success: driver=%s amount=%d refID=%s", event.DriverID, amount, event.ContractID)
		return nil
	}

	worker := concurrencyUtils.NewWorker(msgChan, holdDepositHandler)
	worker.Start(ctx)

	go func() {
		err := kafkaConsumer.Consume(ctx, "wallet.hold_deposit", func(c context.Context, bucket []byte) error {
			select {
			case msgChan <- bucket:
			case <-ctx.Done():
			}
			return nil
		})
		if err != nil {
			log.Printf("[Wallet] Kafka consumer stopped: %v", err)
		}
	}()

	log.Println("[Wallet] Kafka consumer started on topic: wallet.hold_deposit")

	cleanup := func() {
		worker.Stop()
		entClient.Close()
		log.Println("[Wallet] Cleanup complete")
	}

	return cleanup, nil
}
