package app

import (
	"context"
	"log"

	"github.com/google/uuid"
)

type MockWalletClient struct{}

var _ WalletClient = (*MockWalletClient)(nil)

func NewMockWalletClient() WalletClient {
	return &MockWalletClient{}
}

func (m *MockWalletClient) CheckBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	log.Printf("[MockWallet] Check balance for user %s", userID)
	return 10000000.0, nil
}
