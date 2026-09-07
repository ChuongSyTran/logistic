package entity

import "errors"

// Từ vựng lỗi của nghiệp vụ ví. Adapter phải dịch lỗi hạ tầng (ent, sarama, ES)
// sang những giá trị này trước khi trả ra ngoài, để tầng app không phải biết
// dữ liệu được lưu bằng công nghệ gì.
var (
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrSelfTransfer        = errors.New("cannot transfer to self")

	ErrWalletNotFound       = errors.New("wallet not found")
	ErrSystemWalletNotFound = errors.New("system escrow wallet not found")

	// Bản tin đã xử lý trước đó. Đây là kết quả *bình thường* của inbox
	// idempotency, không phải sự cố — app bắt nó và bỏ qua bản tin trùng.
	ErrMessageAlreadyProcessed = errors.New("message already processed")
)
