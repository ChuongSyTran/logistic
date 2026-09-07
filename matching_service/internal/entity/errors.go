package entity

import "errors"

var (
	ErrNilBid   = errors.New("bid is nil")
	ErrNilAsk   = errors.New("ask is nil")
	ErrNilMatch = errors.New("match is nil")

	ErrEmptyLocation   = errors.New("empty location")
	ErrInvalidLocation = errors.New("invalid location")
	ErrEmptyZoneID     = errors.New("empty zone ID")

	ErrBidNotFound   = errors.New("bid not found")
	ErrAskNotFound   = errors.New("ask not found")
	ErrMatchNotFound = errors.New("match not found")

	ErrAlreadyMatched  = errors.New("entity is already matched")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrNotEnoughVolume = errors.New("not enough volume available")
	ErrNotEnoughWeight = errors.New("not enough weight available")
	ErrPriceMismatch   = errors.New("price conditions not met")

	ErrInternal = errors.New("internal system error")
)