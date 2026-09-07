package app

import "errors"

// Lỗi ở tầng app mô tả *cách xử lý*, không mô tả nghiệp vụ. Consumer đọc chúng
// để quyết định retry hay bỏ; nghiệp vụ thì đọc entity.Err* ở tầng dưới.
var (
	ErrStorage = errors.New("storage operation failed")

	ErrNonRetryable = errors.New("non-retryable business error")

	ErrRetryWithDelay = errors.New("retryable error, retry with delay")

	ErrServiceUnavailable503 = errors.New("service unavailable")
)
