package persistence

import (
	"context"

	"wallet_service/ent"
	"wallet_service/internal/app"
	"wallet_service/internal/entity"
	"wallet_service/internal/mapper"
)

type TransactionRepo struct {
	client *ent.Client
	mapper mapper.WalletMapper
}

func NewTransactionRepo(client *ent.Client, m mapper.WalletMapper) *TransactionRepo {
	return &TransactionRepo{client: client, mapper: m}
}

func (r *TransactionRepo) CreateTransaction(ctx context.Context, param *app.CreateTransactionParam) (*entity.Transaction, error) {
	client := clientFrom(ctx, r.client)
	res, err := client.Transaction.Create().
		SetWalletID(param.WalletID).
		SetAmount(param.Amount).
		SetTransactionType(param.TransactionType).
		SetReferenceID(param.ReferenceID).
		SetDescription(param.Description).
		SetStatus(param.Status).
		Save(ctx)
	if err != nil {
		return nil, translate(err, entity.ErrWalletNotFound)
	}
	return r.mapper.EntToTransactionEntity(res), nil
}
