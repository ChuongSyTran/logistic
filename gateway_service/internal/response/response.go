package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/logistic/pkg/apperr"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const HeaderRequestID = "X-Request-ID"

type Envelope struct {
	Data      any    `json:"data,omitempty"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorBody struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id,omitempty"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func requestID(ctx *gin.Context) string {
	if v, ok := ctx.Get(HeaderRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ctx.GetHeader(HeaderRequestID)
}

func OK(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, Envelope{Data: data, RequestID: requestID(ctx)})
}

func OKMessage(ctx *gin.Context, data any, message string) {
	ctx.JSON(http.StatusOK, Envelope{Data: data, Message: message, RequestID: requestID(ctx)})
}

func Created(ctx *gin.Context, data any, message string) {
	ctx.JSON(http.StatusCreated, Envelope{Data: data, Message: message, RequestID: requestID(ctx)})
}

func BadRequest(ctx *gin.Context, code, message string) {
	ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorBody{
		Error:     ErrorDetail{Code: code, Message: message},
		RequestID: requestID(ctx),
	})
}

func Unauthorized(ctx *gin.Context, message string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorBody{
		Error:     ErrorDetail{Code: "UNAUTHENTICATED", Message: message},
		RequestID: requestID(ctx),
	})
}

func Forbidden(ctx *gin.Context, message string) {
	ctx.AbortWithStatusJSON(http.StatusForbidden, ErrorBody{
		Error:     ErrorDetail{Code: "PERMISSION_DENIED", Message: message},
		RequestID: requestID(ctx),
	})
}

// FailedPrecondition cho các điều kiện nghiệp vụ gateway tự kiểm được trước khi
// gọi xuống service. Trả 422 cùng mã với service nội bộ để client xử lý một kiểu.
func FailedPrecondition(ctx *gin.Context, code, message string) {
	ctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, ErrorBody{
		Error:     ErrorDetail{Code: code, Message: message},
		RequestID: requestID(ctx),
	})
}

func Error(ctx *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, ErrorBody{
			Error:     ErrorDetail{Code: "INTERNAL", Message: "internal gateway error"},
			RequestID: requestID(ctx),
		})
		return
	}

	detail := ErrorDetail{
		Code:    defaultCodeFor(st.Code()),
		Message: st.Message(),
	}

	for _, d := range st.Details() {
		info, isInfo := d.(*errdetails.ErrorInfo)
		if !isInfo {
			continue
		}
		if info.GetReason() != "" {
			detail.Code = info.GetReason()
		}
		if len(info.GetMetadata()) > 0 {
			detail.Details = info.GetMetadata()
		}
		break
	}

	ctx.AbortWithStatusJSON(apperr.HTTPStatusFromGRPC(st.Code()), ErrorBody{
		Error:     detail,
		RequestID: requestID(ctx),
	})
}

func defaultCodeFor(c codes.Code) string {
	switch c {
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case codes.Unavailable:
		return "SERVICE_UNAVAILABLE"
	case codes.DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case codes.Aborted:
		return "CONFLICT"
	case codes.Unimplemented:
		return "NOT_IMPLEMENTED"
	default:
		return "INTERNAL"
	}
}