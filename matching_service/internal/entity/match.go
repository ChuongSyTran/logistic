package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	MatchStatusProposed int8 = 1
	MatchStatusAccepted int8 = 2
	MatchStatusRejected int8 = 3
)

type MatchContract struct {
	ID     uuid.UUID `json:"id"`
	BidID  uuid.UUID `json:"bid_id"`
	AskID  uuid.UUID `json:"ask_id"`
	Status int8      `json:"status"`

	ConsensusPrice   float64 `json:"consensus_price"`
	ConsensusDeposit float64 `json:"consensus_deposit"`

	ShipperSignature string    `json:"shipper_signature"`
	DriverSignature  string    `json:"driver_signature"`
	SystemSignature  string    `json:"system_signature"`
	AgreedAt         time.Time `json:"agreed_at"`
	CreatedAt        time.Time `json:"created_at"`
}