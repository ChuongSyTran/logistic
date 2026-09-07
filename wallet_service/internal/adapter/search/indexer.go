package search

import (
	"context"

	"wallet_service/internal/entity"
)

// Indexer là adapter thoả app.WalletIndexer. Nó tách khỏi elasticSearchEngine
// một cách có chủ ý: engine là client ES thuần (biết index, query, mapping),
// còn Indexer là chỗ dịch entity nghiệp vụ sang document. Nhờ vậy tầng app
// không bao giờ nhìn thấy WalletDocument.
type Indexer struct {
	engine WalletSearchEngine
}

func NewIndexer(engine WalletSearchEngine) *Indexer {
	return &Indexer{engine: engine}
}

func (i *Indexer) IndexWallet(ctx context.Context, w *entity.Wallet) error {
	if i == nil || i.engine == nil || w == nil {
		return nil
	}
	return i.engine.IndexWallet(ctx, toWalletDoc(w))
}

func (i *Indexer) IndexTransaction(ctx context.Context, t *entity.Transaction) error {
	if i == nil || i.engine == nil || t == nil {
		return nil
	}
	return i.engine.IndexTransaction(ctx, toTransactionDoc(t))
}

// Hai hàm dưới viết tay chứ không dùng goverter: để goverter sinh chúng thì
// package mapper phải import search, mà Indexer lại cần mapper — thành import
// cycle. Chuyển đổi chỉ là chép trường nên viết tay rẻ hơn gỡ vòng lặp.
func toWalletDoc(w *entity.Wallet) *WalletDocument {
	return &WalletDocument{
		ID:        w.ID.String(),
		UserID:    w.UserID.String(),
		UserType:  w.UserType,
		Balance:   w.Balance,
		Currency:  w.Currency,
		Status:    w.Status,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func toTransactionDoc(t *entity.Transaction) *TransactionDocument {
	return &TransactionDocument{
		ID:              t.ID.String(),
		WalletID:        t.WalletID.String(),
		Amount:          t.Amount,
		TransactionType: t.TransactionType,
		ReferenceID:     t.ReferenceID,
		Description:     t.Description,
		Status:          t.Status,
		CreatedAt:       t.CreatedAt,
	}
}
