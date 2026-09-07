package app

import (
	"context"
	"errors"
	"fmt"

	"wallet_service/internal/entity"

	"github.com/google/uuid"
)

var (
	SystemEscrowWalletID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

type WalletUseCase interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error)
	Deposit(ctx context.Context, userID uuid.UUID, amount int64, description string) (*entity.Transaction, error)
	HoldDeposit(ctx context.Context, driverID uuid.UUID, amount int64, refID string) error
	ReleaseAndPay(ctx context.Context, driverID, shipperID uuid.UUID, escrowAmount, tripFare int64, refID string) error
	TransferMoney(ctx context.Context, fromUser, toUser uuid.UUID, amount int64, kafkaMsgID string) error
}

type wallet struct {
	uow        UnitOfWork
	walletRepo WalletRepository
	txRepo     TransactionRepository
	indexer    WalletIndexer
}

func NewWalletUseCase(
	uow UnitOfWork,
	walletRepo WalletRepository,
	txRepo TransactionRepository,
	indexer WalletIndexer,
) WalletUseCase {
	return &wallet{uow: uow, walletRepo: walletRepo, txRepo: txRepo, indexer: indexer}
}

// notFound phân biệt "không có ví" với mọi lỗi lưu trữ khác. Adapter đã dịch
// sang entity.ErrWalletNotFound nên ở đây không cần biết ent hay driver SQL.
func notFound(err error) bool { return errors.Is(err, entity.ErrWalletNotFound) }

func (uc *wallet) GetBalance(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error) {
	w, err := uc.walletRepo.GetWallet(ctx, userID)
	if err != nil {
		if notFound(err) {
			return nil, fmt.Errorf("%w: user %s", entity.ErrWalletNotFound, userID)
		}
		return nil, fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return w, nil
}

func (uc *wallet) Deposit(ctx context.Context, userID uuid.UUID, amount int64, description string) (*entity.Transaction, error) {
	if amount <= 0 {
		return nil, entity.ErrInvalidAmount
	}

	var txRes *entity.Transaction
	var walletRes *entity.Wallet

	err := uc.uow.Do(ctx, func(ctxTx context.Context) error {
		w, err := uc.walletRepo.GetWalletForUpdate(ctxTx, userID)
		if err != nil {
			if notFound(err) {
				return fmt.Errorf("%w: user %s", entity.ErrWalletNotFound, userID)
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}

		if err := uc.walletRepo.UpdateBalance(ctxTx, w.ID, amount); err != nil {
			return err
		}

		tx, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID:        w.ID,
			Amount:          amount,
			TransactionType: 1,
			ReferenceID:     uuid.New().String(),
			Description:     description,
			Status:          1,
		})
		if err != nil {
			return err
		}

		walletRes, err = uc.walletRepo.GetWallet(ctxTx, userID)
		if err != nil {
			return err
		}
		txRes = tx
		return nil
	})

	if err == nil {
		uc.index(ctx, []*entity.Wallet{walletRes}, []*entity.Transaction{txRes})
	}

	return txRes, err
}

func (uc *wallet) HoldDeposit(ctx context.Context, driverID uuid.UUID, amount int64, refID string) error {
	if amount <= 0 {
		return entity.ErrInvalidAmount
	}

	var wallets []*entity.Wallet
	var txs []*entity.Transaction

	err := uc.uow.Do(ctx, func(ctxTx context.Context) error {
		wallets, txs = nil, nil

		if err := uc.walletRepo.MarkMessageProcessed(ctxTx, refID); err != nil {
			if errors.Is(err, entity.ErrMessageAlreadyProcessed) {
				return nil
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}

		driverWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, driverID)
		if err != nil {
			if notFound(err) {
				return fmt.Errorf("%w: driver %s", entity.ErrWalletNotFound, driverID)
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}

		if driverWallet.Balance < amount {
			return fmt.Errorf("%w: required %d, has %d", entity.ErrInsufficientBalance, amount, driverWallet.Balance)
		}

		escrowWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, SystemEscrowWalletID)
		if err != nil {
			escrowWallet, err = uc.walletRepo.CreateWallet(ctxTx, SystemEscrowWalletID, 0)
			if err != nil {
				return err
			}
		}

		if err := uc.walletRepo.UpdateBalance(ctxTx, driverWallet.ID, -amount); err != nil {
			return err
		}
		if err := uc.walletRepo.UpdateBalance(ctxTx, escrowWallet.ID, amount); err != nil {
			return err
		}

		tx1, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID:        driverWallet.ID,
			Amount:          -amount,
			TransactionType: 5,
			ReferenceID:     refID,
			Description:     "Đóng băng tiền cọc nhận cuốc",
			Status:          1,
		})
		if err != nil {
			return err
		}

		tx2, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID:        escrowWallet.ID,
			Amount:          amount,
			TransactionType: 6,
			ReferenceID:     refID,
			Description:     "Hệ thống nhận tiền cọc của tài xế",
			Status:          1,
		})
		if err != nil {
			return err
		}

		w1, _ := uc.walletRepo.GetWallet(ctxTx, driverWallet.ID)
		w2, _ := uc.walletRepo.GetWallet(ctxTx, escrowWallet.ID)
		wallets = []*entity.Wallet{w1, w2}
		txs = []*entity.Transaction{tx1, tx2}
		return nil
	})

	if err == nil {
		uc.index(ctx, wallets, txs)
	}
	return err
}

func (uc *wallet) ReleaseAndPay(ctx context.Context, driverID, shipperID uuid.UUID, escrowAmount, tripFare int64, refID string) error {
	var wallets []*entity.Wallet
	var txs []*entity.Transaction

	err := uc.uow.Do(ctx, func(ctxTx context.Context) error {
		wallets, txs = nil, nil

		if err := uc.walletRepo.MarkMessageProcessed(ctxTx, refID); err != nil {
			if errors.Is(err, entity.ErrMessageAlreadyProcessed) {
				return nil
			}
			return err
		}

		escrowWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, SystemEscrowWalletID)
		if err != nil {
			if notFound(err) {
				return entity.ErrSystemWalletNotFound
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}
		if escrowWallet.Balance < escrowAmount {
			return fmt.Errorf("%w: system escrow required %d, has %d", entity.ErrInsufficientBalance, escrowAmount, escrowWallet.Balance)
		}

		shipperWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, shipperID)
		if err != nil {
			if notFound(err) {
				return fmt.Errorf("%w: shipper %s", entity.ErrWalletNotFound, shipperID)
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}
		if shipperWallet.Balance < tripFare {
			return fmt.Errorf("%w: shipper required %d, has %d", entity.ErrInsufficientBalance, tripFare, shipperWallet.Balance)
		}

		driverWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, driverID)
		if err != nil {
			if notFound(err) {
				return fmt.Errorf("%w: driver %s", entity.ErrWalletNotFound, driverID)
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}

		if err := uc.walletRepo.UpdateBalance(ctxTx, escrowWallet.ID, -escrowAmount); err != nil {
			return err
		}
		if err := uc.walletRepo.UpdateBalance(ctxTx, driverWallet.ID, escrowAmount); err != nil {
			return err
		}

		if err := uc.walletRepo.UpdateBalance(ctxTx, shipperWallet.ID, -tripFare); err != nil {
			return err
		}
		if err := uc.walletRepo.UpdateBalance(ctxTx, driverWallet.ID, tripFare); err != nil {
			return err
		}

		tx1, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID: escrowWallet.ID, Amount: -escrowAmount, TransactionType: 7, ReferenceID: refID, Description: "Hoàn cọc cho tài xế", Status: 1,
		})
		if err != nil {
			return err
		}

		tx2, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID: driverWallet.ID, Amount: escrowAmount, TransactionType: 8, ReferenceID: refID, Description: "Nhận lại tiền cọc", Status: 1,
		})
		if err != nil {
			return err
		}

		tx3, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID: shipperWallet.ID, Amount: -tripFare, TransactionType: 4, ReferenceID: refID, Description: "Thanh toán phí cuốc xe", Status: 1,
		})
		if err != nil {
			return err
		}

		tx4, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID: driverWallet.ID, Amount: tripFare, TransactionType: 3, ReferenceID: refID, Description: "Nhận cước phí cuốc xe", Status: 1,
		})
		if err != nil {
			return err
		}

		w1, _ := uc.walletRepo.GetWallet(ctxTx, escrowWallet.ID)
		w2, _ := uc.walletRepo.GetWallet(ctxTx, driverWallet.ID)
		w3, _ := uc.walletRepo.GetWallet(ctxTx, shipperWallet.ID)
		wallets = []*entity.Wallet{w1, w2, w3}
		txs = []*entity.Transaction{tx1, tx2, tx3, tx4}
		return nil
	})

	if err == nil {
		uc.index(ctx, wallets, txs)
	}
	return err
}

func (uc *wallet) TransferMoney(ctx context.Context, fromUser, toUser uuid.UUID, amount int64, kafkaMsgID string) error {
	if amount <= 0 {
		return entity.ErrInvalidAmount
	}
	if fromUser == toUser {
		return entity.ErrSelfTransfer
	}

	var wallets []*entity.Wallet
	var txs []*entity.Transaction

	err := uc.uow.Do(ctx, func(ctxTx context.Context) error {
		wallets, txs = nil, nil

		if err := uc.walletRepo.MarkMessageProcessed(ctxTx, kafkaMsgID); err != nil {
			if errors.Is(err, entity.ErrMessageAlreadyProcessed) {
				return nil
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}

		senderWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, fromUser)
		if err != nil {
			if notFound(err) {
				return fmt.Errorf("%w: sender %s", entity.ErrWalletNotFound, fromUser)
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}
		if senderWallet.Balance < amount {
			return fmt.Errorf("%w: sender required %d, has %d", entity.ErrInsufficientBalance, amount, senderWallet.Balance)
		}

		receiverWallet, err := uc.walletRepo.GetWalletForUpdate(ctxTx, toUser)
		if err != nil {
			if notFound(err) {
				return fmt.Errorf("%w: receiver %s", entity.ErrWalletNotFound, toUser)
			}
			return fmt.Errorf("%w: %v", ErrStorage, err)
		}

		if err := uc.walletRepo.UpdateBalance(ctxTx, senderWallet.ID, -amount); err != nil {
			return err
		}
		if err := uc.walletRepo.UpdateBalance(ctxTx, receiverWallet.ID, amount); err != nil {
			return err
		}

		tx1, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID:        senderWallet.ID,
			Amount:          -amount,
			TransactionType: 3,
			ReferenceID:     kafkaMsgID,
			Description:     "Chuyển tiền",
			Status:          1,
		})
		if err != nil {
			return err
		}

		tx2, err := uc.txRepo.CreateTransaction(ctxTx, &CreateTransactionParam{
			WalletID:        receiverWallet.ID,
			Amount:          amount,
			TransactionType: 4,
			ReferenceID:     kafkaMsgID,
			Description:     "Nhận tiền",
			Status:          1,
		})
		if err != nil {
			return err
		}

		w1, _ := uc.walletRepo.GetWallet(ctxTx, senderWallet.ID)
		w2, _ := uc.walletRepo.GetWallet(ctxTx, receiverWallet.ID)
		wallets = []*entity.Wallet{w1, w2}
		txs = []*entity.Transaction{tx1, tx2}
		return nil
	})

	if err == nil {
		uc.index(ctx, wallets, txs)
	}
	return err
}

// index đẩy bản ghi sang chỉ mục tìm kiếm SAU khi transaction đã commit.
//
// CẢNH BÁO — đây vẫn là dual write: Postgres/MySQL đã commit mà lệnh index có
// thể hỏng, và lỗi ở đây đang bị nuốt hoàn toàn nên chỉ mục lệch vĩnh viễn mà
// không ai biết. Chỗ này phải được thay bằng outbox: ghi một dòng vào bảng
// outbox TRONG cùng transaction, rồi một relay đọc bảng đó và đánh chỉ mục.
// Cấu trúc hiện tại đã sẵn sàng cho việc đó — outbox chỉ là một adapter khác
// thoả WalletIndexer, use case không phải sửa dòng nào.
func (uc *wallet) index(ctx context.Context, wallets []*entity.Wallet, txs []*entity.Transaction) {
	if uc.indexer == nil {
		return
	}
	for _, w := range wallets {
		if w == nil {
			continue
		}
		_ = uc.indexer.IndexWallet(ctx, w)
	}
	for _, t := range txs {
		if t == nil {
			continue
		}
		_ = uc.indexer.IndexTransaction(ctx, t)
	}
}
