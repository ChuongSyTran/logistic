package di

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func timeoutInterceptor(defaultTimeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if _, hasDeadline := ctx.Deadline(); hasDeadline {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func concurrencyLimiter(limit int) grpc.UnaryClientInterceptor {
	if limit <= 0 {
		return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	slots := make(chan struct{}, limit)

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			return status.Error(codes.ResourceExhausted,
				"gateway đang quá tải tới dịch vụ này, hãy thử lại sau")
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}