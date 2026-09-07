package biz

import "errors"

var (
	ErrNonRetryable = errors.New("fatal_error_do_not_retry")

	ErrRetryWithDelay = errors.New("service_unavailable_retry_later")

	ErrServiceUnavailable503 = errors.New("service_unavailable_503")
)
