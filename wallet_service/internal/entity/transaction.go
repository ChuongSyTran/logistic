package entity

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID              uuid.UUID
	WalletID        uuid.UUID
	Amount          int64
	TransactionType uint8
	ReferenceID     string
	Description     string
	Status          uint8
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
