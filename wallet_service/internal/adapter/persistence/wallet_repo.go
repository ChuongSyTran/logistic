package persistence

import (
	"context"

	"wallet_service/ent"
	"wallet_service/ent/wallet"
	"wallet_service/internal/entity"
	"wallet_service/internal/mapper"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

// WalletRepo trả về kiểu cụ thể chứ không phải interface — interface tương ứng
// (app.WalletRepository) được khai ở phía dùng nó.
type WalletRepo struct {
	client *ent.Client
	mapper mapper.WalletMapper
}

func NewWalletRepo(client *ent.Client, m mapper.WalletMapper) *WalletRepo {
	return &WalletRepo{client: client, mapper: m}
}

func (r *WalletRepo) MarkMessageProcessed(ctx context.Context, messageID string) error {
	client := clientFrom(ctx, r.client)
	err := client.ProcessedMessage.Create().SetID(messageID).Exec(ctx)
	return translate(err, entity.ErrMessageAlreadyProcessed)
}

func (r *WalletRepo) GetWalletForUpdate(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error) {
	client := clientFrom(ctx, r.client)
	res, err := client.Wallet.Query().
		Where(wallet.UserID(userID)).
		Modify(func(s *sql.Selector) {
			s.ForUpdate()
		}).
		Only(ctx)
	if err != nil {
		return nil, translate(err, entity.ErrWalletNotFound)
	}
	return r.mapper.EntToWalletEntity(res), nil
}

func (r *WalletRepo) GetWallet(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error) {
	client := clientFrom(ctx, r.client)
	res, err := client.Wallet.Query().
		Where(wallet.UserID(userID)).
		Only(ctx)
	if err != nil {
		return nil, translate(err, entity.ErrWalletNotFound)
	}
	return r.mapper.EntToWalletEntity(res), nil
}

func (r *WalletRepo) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64) error {
	client := clientFrom(ctx, r.client)
	err := client.Wallet.UpdateOneID(walletID).
		AddBalance(amount).
		Exec(ctx)
	return translate(err, entity.ErrWalletNotFound)
}

func (r *WalletRepo) CreateWallet(ctx context.Context, userID uuid.UUID, userType uint8) (*entity.Wallet, error) {
	client := clientFrom(ctx, r.client)
	res, err := client.Wallet.Create().
		SetUserID(userID).
		SetUserType(userType).
		Save(ctx)
	if err != nil {
		return nil, translate(err, entity.ErrWalletNotFound)
	}
	return r.mapper.EntToWalletEntity(res), nil
}
