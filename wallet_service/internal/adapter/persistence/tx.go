package persistence

import (
	"context"
	"errors"
	"fmt"

	"wallet_service/ent"
)

type keyTx struct{}

type UnitOfWork struct {
	entClient *ent.Client
}

func NewUnitOfWork(entClient *ent.Client) *UnitOfWork {
	return &UnitOfWork{entClient: entClient}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(ctxTx context.Context) error) (err error) {
	tx, err := u.entClient.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if errorRollBack := tx.Rollback(); errorRollBack != nil {
				err = errors.Join(err, fmt.Errorf("rollback err: %w", errorRollBack))
			}
		} else {
			if errorCommit := tx.Commit(); errorCommit != nil {
				err = fmt.Errorf("commit err: %w", errorCommit)
			}
		}
	}()

	ctxTx := context.WithValue(ctx, keyTx{}, tx.Client())
	err = fn(ctxTx)
	return
}

// clientFrom lấy *ent.Client của transaction đang mở trong context; không có thì
// dùng client thường. Không export vì chỉ repo trong package này mới cần.
func clientFrom(ctx context.Context, client *ent.Client) *ent.Client {
	clientAny := ctx.Value(keyTx{})
	if clientTx, ok := clientAny.(*ent.Client); ok {
		return clientTx
	}
	return client
}
