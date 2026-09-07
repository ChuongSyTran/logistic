package mapper

import (
	"time"

	"wallet_service/ent"
	"wallet_service/internal/adapter/search"
	"wallet_service/internal/entity"

	"github.com/google/uuid"

	pb "github.com/logistic/api/logistic/wallet_service/v1"
)

// LƯU Ý VỀ RANH GIỚI: đây là package duy nhất còn đứng giữa nhiều tầng — nó
// biết cả ent (persistence), search (adapter tìm kiếm) lẫn pb (API). Lý do là
// goverter sinh ra một implementation duy nhất cho một interface duy nhất.
// Bước dọn tiếp theo là tách thành ba interface, mỗi cái nằm trong adapter sở
// hữu nó, rồi chạy lại goverter.
//
// goverter:converter
// goverter:ignoreUnexported
// goverter:extend TimeToString
// goverter:extend UUIDToBytes
// goverter:extend StringToBytes
// goverter:extend UUIDToString
// goverter:extend Uint8ToUint32
// goverter:extend IdentityTime
//
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@latest gen ./
type WalletMapper interface {
	EntToWalletEntity(source *ent.Wallet) *entity.Wallet
	EntToTransactionEntity(source *ent.Transaction) *entity.Transaction

	// goverter:map ID Id
	// goverter:map UserID UserId
	ESWalletToProto(source *search.WalletDocument) *pb.WalletInfo

	// goverter:map ID Id
	// goverter:map WalletID WalletId
	// goverter:map ReferenceID ReferenceId
	ESTransactionToProto(source *search.TransactionDocument) *pb.TransactionInfo

	// goverter:map ID Id
	// goverter:map UserID UserId
	EntityToProtoWallet(source *entity.Wallet) *pb.WalletInfo
}

func TimeToString(t time.Time) string {
	return t.Format(time.RFC3339)
}

func IdentityTime(t time.Time) time.Time {
	return t
}

func UUIDToBytes(u uuid.UUID) []byte {
	return u[:]
}

func StringToBytes(s string) []byte {
	return []byte(s)
}

func UUIDToString(u uuid.UUID) string {
	return u.String()
}

func Uint8ToUint32(u uint8) uint32 {
	return uint32(u)
}
