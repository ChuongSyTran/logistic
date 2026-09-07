package apperr

import (
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
)

type Kind string

const (
	KindInvalidArgument  Kind = "INVALID_ARGUMENT"
	KindUnauthenticated  Kind = "UNAUTHENTICATED"
	KindPermissionDenied Kind = "PERMISSION_DENIED"
	KindNotFound         Kind = "NOT_FOUND"
	KindAlreadyExists    Kind = "ALREADY_EXISTS"
	KindConflict         Kind = "CONFLICT"
	KindFailedPrecond    Kind = "FAILED_PRECONDITION"
	KindResourceExceeded Kind = "RESOURCE_EXHAUSTED"
	KindUnavailable      Kind = "UNAVAILABLE"
	KindTimeout          Kind = "DEADLINE_EXCEEDED"
	KindInternal         Kind = "INTERNAL"
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Details map[string]string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s(%s): %s: %v", e.Kind, e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s(%s): %s", e.Kind, e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code != "" && e.Code == t.Code
}

func (e *Error) WithCause(err error) *Error {
	clone := *e
	clone.cause = err
	return &clone
}

func (e *Error) WithDetail(key, value string) *Error {
	clone := *e
	clone.Details = make(map[string]string, len(e.Details)+1)
	for k, v := range e.Details {
		clone.Details[k] = v
	}
	clone.Details[key] = value
	return &clone
}

func (e *Error) WithMessage(format string, args ...any) *Error {
	clone := *e
	clone.Message = fmt.Sprintf(format, args...)
	return &clone
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func InvalidArgument(code, msg string) *Error    { return New(KindInvalidArgument, code, msg) }
func NotFound(code, msg string) *Error           { return New(KindNotFound, code, msg) }
func AlreadyExists(code, msg string) *Error      { return New(KindAlreadyExists, code, msg) }
func Conflict(code, msg string) *Error           { return New(KindConflict, code, msg) }
func FailedPrecondition(code, msg string) *Error { return New(KindFailedPrecond, code, msg) }
func PermissionDenied(code, msg string) *Error   { return New(KindPermissionDenied, code, msg) }
func Unauthenticated(code, msg string) *Error    { return New(KindUnauthenticated, code, msg) }
func Unavailable(code, msg string) *Error        { return New(KindUnavailable, code, msg) }
func Internal(code, msg string) *Error           { return New(KindInternal, code, msg) }

func From(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func (e *Error) GRPCCode() codes.Code {
	switch e.Kind {
	case KindInvalidArgument:
		return codes.InvalidArgument
	case KindUnauthenticated:
		return codes.Unauthenticated
	case KindPermissionDenied:
		return codes.PermissionDenied
	case KindNotFound:
		return codes.NotFound
	case KindAlreadyExists:
		return codes.AlreadyExists
	case KindConflict:
		return codes.Aborted
	case KindFailedPrecond:
		return codes.FailedPrecondition
	case KindResourceExceeded:
		return codes.ResourceExhausted
	case KindUnavailable:
		return codes.Unavailable
	case KindTimeout:
		return codes.DeadlineExceeded
	default:
		return codes.Internal
	}
}

func (e *Error) HTTPStatus() int {
	switch e.Kind {
	case KindInvalidArgument:
		return http.StatusBadRequest
	case KindUnauthenticated:
		return http.StatusUnauthorized
	case KindPermissionDenied:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindAlreadyExists, KindConflict:
		return http.StatusConflict
	case KindFailedPrecond:
		return http.StatusUnprocessableEntity
	case KindResourceExceeded:
		return http.StatusTooManyRequests
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func HTTPStatusFromGRPC(c codes.Code) int {
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.FailedPrecondition:
		return http.StatusUnprocessableEntity
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unimplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}