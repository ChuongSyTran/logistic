package consumer

import (
	"context"
	"strconv"

	"notification_service/internal/biz"
	"notification_service/internal/entity"
)

type templateCode string

const (
	codeDriverCandidate   templateCode = "DRIVER_CANDIDATE"
	codeMatchFoundShipper templateCode = "MATCH_FOUND_SHIPPER"
	codeMatchFoundDriver  templateCode = "MATCH_FOUND_DRIVER"
	codeOfferReceived     templateCode = "OFFER_RECEIVED"
	codeOfferRejected     templateCode = "OFFER_REJECTED"
	codeCargoSuggested    templateCode = "CARGO_SUGGESTED"
)

type notificationText struct {
	title string
	body  string
}

func renderText(
	ctx context.Context,
	engine biz.NotificationEngine,
	code templateCode,
	channel string,
	vars map[string]string,
	fallback notificationText,
) notificationText {
	title, body, ok := engine.RenderFromTemplate(ctx, string(code), channel, entity.LocaleVI, vars)
	if !ok || title == "" || body == "" {
		return fallback
	}
	return notificationText{title: title, body: body}
}

func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', 0, 64)
}

func formatDecimal(v float64) string {
	return strconv.FormatFloat(v, 'f', 1, 64)
}
