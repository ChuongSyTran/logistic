package middleware

import (
	"context"
	"log"
	"runtime/debug"
	"time"

	"github.com/logistic/pkg/apperr"
	"github.com/logistic/pkg/authn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		return nil, toStatusError(serviceName, info.FullMethod, err)
	}
}

func toStatusError(serviceName, method string, err error) error {
	if appErr, ok := apperr.From(err); ok {
		st := status.New(appErr.GRPCCode(), appErr.Message)

		info := &errdetails.ErrorInfo{
			Reason:   appErr.Code,
			Domain:   serviceName,
			Metadata: appErr.Details,
		}
		if withDetails, dErr := st.WithDetails(info); dErr == nil {
			st = withDetails
		}

		if appErr.Kind == apperr.KindInternal {
			log.Printf("[%s][ERROR] %s -> %v", serviceName, method, appErr.Error())
		}
		return st.Err()
	}

	if st, ok := status.FromError(err); ok && st.Code() != codes.Unknown {
		return err
	}

	log.Printf("[%s][UNHANDLED] %s -> %v", serviceName, method, err)
	return status.Error(codes.Internal, "internal server error")
}

func RecoveryInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[%s][PANIC] %s -> %v\n%s", serviceName, info.FullMethod, r, debug.Stack())
				err = status.Error(codes.Internal, "internal server error")
				resp = nil
			}
		}()
		return handler(ctx, req)
	}
}

func LoggingInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Printf("[%s] %s code=%s dur=%s", serviceName, info.FullMethod, status.Code(err), time.Since(start))
		return resp, err
	}
}

func ChainForService(serviceName string) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		RecoveryInterceptor(serviceName),
		LoggingInterceptor(serviceName),
		authn.IdentityUnaryInterceptor(),
		ErrorInterceptor(serviceName),
	)
}