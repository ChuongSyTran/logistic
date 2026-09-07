package events

import "time"

const ExchangeLogistic = "logistic.events"

const (
	RoutingKeyDriverCandidatesFound = "matching.driver.candidates_found"

	RoutingKeyMatchFound = "matching.match.found"

	RoutingKeyOfferReceived = "matching.offer.received"

	RoutingKeyOfferRejected = "matching.offer.rejected"

	RoutingKeyCargoSuggested = "matching.cargo.suggested"
)

const (
	ChannelInApp = "in_app"
	ChannelPush  = "push"
	ChannelEmail = "email"
	ChannelSMS   = "sms"
)

type Envelope struct {
	EventID    string         `json:"event_id"`
	EventType  string         `json:"event_type"`
	OccurredAt time.Time      `json:"occurred_at"`
	TraceID    string         `json:"trace_id,omitempty"`
	Source     string         `json:"source"`
	Data       map[string]any `json:"data"`
}

type DriverCandidate struct {
	DriverID   string  `json:"driver_id"`
	AskID      string  `json:"ask_id"`
	VehicleID  string  `json:"vehicle_id"`
	Phone      string  `json:"phone,omitempty"`
	Email      string  `json:"email,omitempty"`
	DistanceKm float64 `json:"distance_km"`
	Score      float64 `json:"score"`
}

type DriverCandidatesFound struct {
	BidID          string            `json:"bid_id"`
	ShipperID      string            `json:"shipper_id"`
	OriginZoneID   string            `json:"origin_zone_id"`
	OriginLat      float64           `json:"origin_lat"`
	OriginLng      float64           `json:"origin_lng"`
	DestinationLat float64           `json:"destination_lat"`
	DestinationLng float64           `json:"destination_lng"`
	WeightKg       float64           `json:"weight_kg"`
	VolumeM3       float64           `json:"volume_m3"`
	MaxPrice       float64           `json:"max_price"`
	Candidates     []DriverCandidate `json:"candidates"`
}

type MatchFound struct {
	ContractID       string  `json:"contract_id"`
	BidID            string  `json:"bid_id"`
	AskID            string  `json:"ask_id"`
	ShipperID        string  `json:"shipper_id"`
	DriverID         string  `json:"driver_id"`
	VehicleID        string  `json:"vehicle_id"`
	ConsensusPrice   float64 `json:"consensus_price"`
	ConsensusDeposit float64 `json:"consensus_deposit"`
	ShipperPhone     string  `json:"shipper_phone,omitempty"`
	ShipperEmail     string  `json:"shipper_email,omitempty"`
	DriverPhone      string  `json:"driver_phone,omitempty"`
	DriverEmail      string  `json:"driver_email,omitempty"`
}

type OfferReceived struct {
	BidID     string  `json:"bid_id"`
	AskID     string  `json:"ask_id"`
	ShipperID string  `json:"shipper_id"`
	DriverID  string  `json:"driver_id"`
	Price     float64 `json:"price"`
}

type OfferRejected struct {
	BidID    string `json:"bid_id"`
	AskID    string `json:"ask_id"`
	DriverID string `json:"driver_id"`
	Reason   string `json:"reason,omitempty"`
}

type CargoSuggested struct {
	AskID      string   `json:"ask_id"`
	DriverID   string   `json:"driver_id"`
	VehicleID  string   `json:"vehicle_id"`
	BidIDs     []string `json:"bid_ids"`
	TotalFound int      `json:"total_found"`
}