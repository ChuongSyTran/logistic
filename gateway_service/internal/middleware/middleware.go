package middleware

import (
	"log"
	"runtime/debug"
	"time"

	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.GetHeader(response.HeaderRequestID)

		if id == "" {
			if sc := trace.SpanContextFromContext(ctx.Request.Context()); sc.IsValid() {
				id = sc.TraceID().String()
			} else {
				id = uuid.Must(uuid.NewV7()).String()
			}
		}

		ctx.Set(response.HeaderRequestID, id)
		ctx.Writer.Header().Set(response.HeaderRequestID, id)
		ctx.Next()
	}
}

func Recovery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[gateway][PANIC] %s %s -> %v\n%s",
					ctx.Request.Method, ctx.Request.URL.Path, r, debug.Stack())

				if !ctx.Writer.Written() {
					ctx.AbortWithStatusJSON(500, response.ErrorBody{
						Error: response.ErrorDetail{
							Code:    "INTERNAL",
							Message: "internal gateway error",
						},
						RequestID: ctx.GetString(response.HeaderRequestID),
					})
				}
			}
		}()
		ctx.Next()
	}
}

func ErrorGuard() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 || ctx.Writer.Written() {
			return
		}
		response.Error(ctx, ctx.Errors.Last().Err)
	}
}

func AccessLog() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		userID := ctx.GetString(CtxUserID)
		if userID == "" {
			userID = "-"
		}

		log.Printf("[gateway] %s %s -> %d (%s) trace=%s user=%s",
			ctx.Request.Method,
			ctx.Request.URL.Path,
			ctx.Writer.Status(),
			time.Since(start),
			ctx.GetString(response.HeaderRequestID),
			userID,
		)
	}
}