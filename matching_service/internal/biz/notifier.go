package biz

import (
	"context"

	"matching_service/internal/entity"
)

type Notifier interface {
	NotifyDriverCandidates(ctx context.Context, bid *entity.Bid, asks []entity.Ask) error

	NotifyMatchFound(ctx context.Context, contract *entity.MatchContract, bid *entity.Bid, ask *entity.Ask) error

	NotifyOfferReceived(ctx context.Context, bid *entity.Bid, ask *entity.Ask, price float64) error

	NotifyOfferRejected(ctx context.Context, bid *entity.Bid, ask *entity.Ask, reason string) error

	NotifyCargoSuggested(ctx context.Context, ask *entity.Ask, bids []entity.Bid) error
}

type NoopNotifier struct{}

func (NoopNotifier) NotifyDriverCandidates(context.Context, *entity.Bid, []entity.Ask) error {
	return nil
}

func (NoopNotifier) NotifyMatchFound(context.Context, *entity.MatchContract, *entity.Bid, *entity.Ask) error {
	return nil
}

func (NoopNotifier) NotifyOfferReceived(context.Context, *entity.Bid, *entity.Ask, float64) error {
	return nil
}

func (NoopNotifier) NotifyOfferRejected(context.Context, *entity.Bid, *entity.Ask, string) error {
	return nil
}

func (NoopNotifier) NotifyCargoSuggested(context.Context, *entity.Ask, []entity.Bid) error {
	return nil
}
