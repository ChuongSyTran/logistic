package entity

import (
	"time"

	"github.com/google/uuid"
)

type Bid struct {
	ID              uuid.UUID `json:"id"`
	ShipperID       uuid.UUID `json:"shipper_id"`
	ShipperPhone    string    `json:"shipper_phone"`
	ShipperMail     string    `json:"shipper_mail"`
	ConsigneeID     uuid.UUID `json:"consignee_id"`
	ConsigneePhone  string    `json:"consignee_phone"`
	ConsigneeMail   string    `json:"consignee_mail"`
	Origin          Location  `json:"origin"`
	Destination     Location  `json:"destination"`
	VolumeM3        float64   `json:"volume_m3"`
	WeightKg        float64   `json:"weight_kg"`
	MaxPrice        float64   `json:"max_price"`
	OfferedPrice    float64   `json:"offered_price"`
	OfferedAskID    uuid.UUID `json:"offered_ask_id"`
	Status          int8      `json:"status"`
	CargoValue      float64   `json:"cargo_value"`
	RequiredDeposit float64   `json:"required_deposit"`
	DesiredDeposit  float64   `json:"desired_deposit"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

const (
	BidStatusPending int8 = iota
	BidStatusNegotiating
	BidStatusMatched
	BidStatusCancelled
)

func IsValidBidStatus(status int8) bool {
	switch status {
	case BidStatusPending, BidStatusNegotiating, BidStatusMatched, BidStatusCancelled:
		return true
	default:
		return false
	}
}

func (b *Bid) StatusString() string {
	switch b.Status {
	case BidStatusPending:
		return "PENDING"
	case BidStatusNegotiating:
		return "NEGOTIATING"
	case BidStatusMatched:
		return "MATCHED"
	case BidStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}
