package app

import (
	"context"

	"wallet_service/internal/entity"

	"github.com/google/uuid"
)

// Toàn bộ port của tầng app nằm trong file này. Chúng được khai ở đây — phía
// *dùng* — chứ không phải ở package implement, đúng "accept interfaces, return
// structs" (Learning Go ch7, trang in 162). Nhờ vậy adapter phụ thuộc vào app,
// app không phụ thuộc vào adapter nào.

type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctxTx context.Context) error) error
}

type WalletRepository interface {
	GetWalletForUpdate(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error)
	GetWallet(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error)
	UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64) error
	MarkMessageProcessed(ctx context.Context, messageID string) error
	CreateWallet(ctx context.Context, userID uuid.UUID, userType uint8) (*entity.Wallet, error)
}

type CreateTransactionParam struct {
	WalletID        uuid.UUID
	Amount          int64
	TransactionType uint8
	ReferenceID     string
	Description     string
	Status          uint8
}

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, param *CreateTransactionParam) (*entity.Transaction, error)
}

// WalletIndexer nhận entity, không nhận document của Elasticsearch. Nhờ thế app
// không biết chỉ mục tìm kiếm có hình dạng gì, và đổi ES sang thứ khác không
// đụng vào use case.
type WalletIndexer interface {
	IndexWallet(ctx context.Context, w *entity.Wallet) error
	IndexTransaction(ctx context.Context, t *entity.Transaction) error
}

type Header struct {
	Key   []byte
	Value []byte
}

type EventMessage struct {
	Header  *Header
	Topic   string
	Key     string
	Payload any
}

type EventPublisher interface {
	Publish(ctx context.Context, msg *EventMessage) error
}

type EventConsumer interface {
	Consume(ctx context.Context, topic string, handler func(ctx context.Context, bucket []byte) error) error
}
