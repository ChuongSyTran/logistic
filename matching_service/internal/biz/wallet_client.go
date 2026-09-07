package biz

import (
	"context"
	"log"

	"github.com/google/uuid"
)

type WalletClient interface {
	CheckBalance(ctx context.Context, userID uuid.UUID) (float64, error)
}

type MockWalletClient struct{}

func NewMockWalletClient() WalletClient {
	return &MockWalletClient{}
}

func (m *MockWalletClient) CheckBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	log.Printf("[MockWallet] Check balance for user %s", userID)
	return 10000000.0, nil
}
