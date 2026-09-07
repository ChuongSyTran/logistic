package app

import (
	"context"

	"matching_service/internal/entity"
)

type NoopNotifier struct{}

var _ Notifier = (*NoopNotifier)(nil)

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
