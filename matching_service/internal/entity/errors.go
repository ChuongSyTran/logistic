package entity

import (
	"errors"

	"github.com/logistic/pkg/apperr"
)

var (
	// Legacy / Core domain validation errors
	ErrNilBid   = errors.New("bid is nil")
	ErrNilAsk   = errors.New("ask is nil")
	ErrNilMatch = errors.New("match is nil")

	ErrEmptyLocation   = errors.New("empty location")
	ErrInvalidLocation = errors.New("invalid location")
	ErrEmptyZoneID     = errors.New("empty zone ID")

	ErrBidNotFound   = errors.New("bid not found")
	ErrAskNotFound   = errors.New("ask not found")
	ErrMatchNotFound = errors.New("match not found")

	ErrAlreadyMatched  = errors.New("entity is already matched")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrNotEnoughVolume = errors.New("not enough volume available")
	ErrNotEnoughWeight = errors.New("not enough weight available")

	ErrInternal = errors.New("internal system error")

	// Broker retry errors
	ErrNonRetryable          = errors.New("fatal_error_do_not_retry")
	ErrRetryWithDelay        = errors.New("service_unavailable_retry_later")
	ErrServiceUnavailable503 = errors.New("service_unavailable_503")

	// Apperr structured sentinel errors (API / RPC level)
	ErrInvalidID       = apperr.InvalidArgument("INVALID_ID", "định dạng id không hợp lệ")
	ErrRecordNotFound  = apperr.NotFound("RECORD_NOT_FOUND", "không tìm thấy bản ghi")
	ErrInlavidInput    = apperr.InvalidArgument("INVALID_INPUT", "dữ liệu đầu vào không hợp lệ")
	ErrDuplicateRecord = apperr.AlreadyExists("DUPLICATE_RECORD", "bản ghi đã tồn tại")

	ErrValidationFailed    = apperr.InvalidArgument("VALIDATION_FAILED", "dữ liệu không hợp lệ")
	ErrUnauthorized        = apperr.PermissionDenied("UNAUTHORIZED", "không có quyền thực hiện hành động này")
	ErrInsufficientBalance = apperr.FailedPrecondition("INSUFFICIENT_BALANCE", "số dư ví không đủ để đặt cọc")

	ErrBidNotPending     = apperr.FailedPrecondition("BID_NOT_PENDING", "đơn hàng không còn ở trạng thái chờ")
	ErrBidNotNegotiating = apperr.FailedPrecondition("BID_NOT_NEGOTIATING", "đơn hàng không ở trạng thái đang thương lượng")
	ErrOfferAskMismatch  = apperr.FailedPrecondition("OFFER_ASK_MISMATCH", "chuyến này không phải chuyến đang thương lượng với đơn hàng")
	ErrPriceMismatch     = apperr.Conflict("PRICE_MISMATCH", "giá xác nhận lệch với giá tài xế đã báo")

	ErrOfferQueueUnavailable = apperr.Unavailable("OFFER_QUEUE_UNAVAILABLE", "hàng đợi báo giá tạm thời không nhận, thử lại sau")
	ErrWalletUnavailable     = apperr.Unavailable("WALLET_UNAVAILABLE", "chưa kiểm tra được số dư ví, thử lại sau")

	ErrInternalServer = apperr.Internal("INTERNAL_SERVER_ERROR", "lỗi hệ thống")
)