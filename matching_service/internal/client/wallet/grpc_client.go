package wallet

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/wallet_service/v1"

	"matching_service/internal/biz"
)

type grpcClient struct {
	client pb.WalletServiceClient
}

func NewGrpcClient(client pb.WalletServiceClient) biz.WalletClient {
	return &grpcClient{client: client}
}

// AcceptOffer giữ e.mu suốt lời gọi này, nên ví treo là mọi lượt chốt xếp hàng theo.
const balanceTimeout = 3 * time.Second

func (w *grpcClient) CheckBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, balanceTimeout)
	defer cancel()

	res, err := w.client.GetBalance(ctx, &pb.GetBalanceReq{UserId: userID[:]})
	if err != nil {
		return 0, fmt.Errorf("failed to get balance from wallet_service: %w", err)
	}
	return float64(res.Balance) / 100, nil
}
