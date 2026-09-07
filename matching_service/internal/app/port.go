package app

import (
	"context"

	"matching_service/internal/entity"

	"github.com/google/uuid"
)

// MatchingRepo là driven port quản lý lưu trữ và truy vấn Bid, Ask, MatchContract trong DB.
type MatchingRepo interface {
	CreateBid(ctx context.Context, bid *entity.Bid) error
	CreateAsk(ctx context.Context, ask *entity.Ask) error
	FindAskForBid(ctx context.Context, bid *entity.Bid) ([]entity.Ask, error)
	FindBidForAsk(ctx context.Context, ask *entity.Ask) ([]entity.Bid, error)
	UpdateBid(ctx context.Context, bid *entity.Bid) error
	UpdateAsk(ctx context.Context, ask *entity.Ask) error
	DeleteAsk(ctx context.Context, id uuid.UUID) error
	DeleteBid(ctx context.Context, id uuid.UUID) error
	GetBid(ctx context.Context, id uuid.UUID) (*entity.Bid, error)
	GetAsk(ctx context.Context, id uuid.UUID) (*entity.Ask, error)
	CreateMatchContract(ctx context.Context, contract *entity.MatchContract) error
}

// SpatialEngine là driven port phân vùng địa lý và tìm các vùng lân cận.
type SpatialEngine interface {
	GetZoneId(ctx context.Context, lat, lng float64) (string, error)
	GetNeighborZones(ctx context.Context, zoneID string) ([]string, error)
}

// WalletClient là driven port kết nối sang wallet_service để kiểm tra số dư ví.
type WalletClient interface {
	CheckBalance(ctx context.Context, userID uuid.UUID) (float64, error)
}

// Notifier là driven port gửi thông báo đẩy sự kiện matching đến người dùng.
type Notifier interface {
	NotifyDriverCandidates(ctx context.Context, bid *entity.Bid, asks []entity.Ask) error
	NotifyMatchFound(ctx context.Context, contract *entity.MatchContract, bid *entity.Bid, ask *entity.Ask) error
	NotifyOfferReceived(ctx context.Context, bid *entity.Bid, ask *entity.Ask, price float64) error
	NotifyOfferRejected(ctx context.Context, bid *entity.Bid, ask *entity.Ask, reason string) error
	NotifyCargoSuggested(ctx context.Context, ask *entity.Ask, bids []entity.Bid) error
}

// Header mang metadata của message broker.
type Header struct {
	Key   []byte
	Value []byte
}

// EventMessage là cấu trúc tin nhắn chuẩn gửi qua message broker.
type EventMessage struct {
	Header  *Header
	Topic   string
	Key     string
	Payload any
}

// EventPublisher là driven port phát sự kiện sang message broker (Kafka, NATS).
type EventPublisher interface {
	Publish(ctx context.Context, msg *EventMessage) error
}

// EventHandler xử lý tin nhắn nhận được từ consumer.
type EventHandler func(ctx context.Context, subject string, payload []byte) error

// EventConsumer là port tiêu thụ tin nhắn từ message broker.
type EventConsumer interface {
	Consume(ctx context.Context, topic string, handler EventHandler) error
}

// MatchingEngine là driving (inbound) port định nghĩa toàn bộ usecase ghép cặp đơn hàng và chuyến xe.
type MatchingEngine interface {
	SubmitBid(ctx context.Context, bid *entity.Bid) (*entity.Bid, error)
	SubmitAsk(ctx context.Context, ask *entity.Ask) (*entity.Ask, error)
	SubmitOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID, desiredPrice float64) error
	ProcessOfferQueue(ctx context.Context, bidID uuid.UUID, offerAsk *entity.Ask) error
	AcceptOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID, consensusPrice float64, shipperSignature string) (*entity.MatchContract, error)
	RejectOffer(ctx context.Context, bidID uuid.UUID, askID uuid.UUID) error
	MatchStream() <-chan *entity.MatchContract
}
