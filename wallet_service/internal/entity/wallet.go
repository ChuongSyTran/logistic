package entity

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	UserType  uint8
	Balance   int64
	Currency  string
	Status    uint8
	CreatedAt time.Time
	UpdatedAt time.Time
}
