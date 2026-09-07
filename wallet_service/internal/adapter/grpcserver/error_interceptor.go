package grpcserver

import (
	"context"
	"errors"

	"wallet_service/internal/entity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorHandlerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		res, err := handler(ctx, req)
		if err != nil {
			return res, mapErrorToGrpcStatus(err)
		}
		return res, nil
	}
}

// Chỉ ánh xạ lỗi nghiệp vụ sang mã gRPC. Trước đây hàm này còn phải gọi
// ent.IsNotFound — tức là tầng ngoài cùng vẫn biết dự án dùng ORM nào. Giờ
// persistence đã dịch sẵn nên ở đây không còn dấu vết hạ tầng.
func mapErrorToGrpcStatus(err error) error {
	switch {
	case errors.Is(err, entity.ErrInvalidAmount), errors.Is(err, entity.ErrSelfTransfer):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, entity.ErrInsufficientBalance):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, entity.ErrWalletNotFound), errors.Is(err, entity.ErrSystemWalletNotFound):
		return status.Error(codes.NotFound, err.Error())

	default:
		return status.Errorf(codes.Internal, "internal server error: %v", err)
	}
}
