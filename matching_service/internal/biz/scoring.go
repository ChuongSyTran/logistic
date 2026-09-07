package biz

import (
	"math"
	"sort"

	"matching_service/internal/entity"
)

type ScoreWeights struct {
	Deadhead  float64
	Alignment float64
	Detour    float64
	Fill      float64
	Price     float64
}

func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		Deadhead:  0.30,
		Alignment: 0.25,
		Detour:    0.20,
		Fill:      0.15,
		Price:     0.10,
	}
}

const (
	offRouteShareOfTrip  = 0.25
	minOffRouteBudgetKm  = 30.0
	maxOffRouteBudgetKm  = 250.0
	maxDetourRatio       = 0.35
	minAcceptableScore   = 0.35
	maxCandidatesPerBid  = 20
	maxSuggestionsPerAsk = 20
	behindTruckPenalty   = 0.4
	earthRadiusKm        = 6371.0
)

type ScoreBreakdown struct {
	Deadhead  float64 `json:"deadhead"`
	Alignment float64 `json:"alignment"`
	Detour    float64 `json:"detour"`
	Fill      float64 `json:"fill"`
	Price     float64 `json:"price"`
}

type ScoredBid struct {
	Bid        entity.Bid     `json:"bid"`
	Score      float64        `json:"score"`
	Breakdown  ScoreBreakdown `json:"breakdown"`
	DeadheadKm float64        `json:"deadhead_km"`
	DetourKm   float64        `json:"detour_km"`
	FillRatio  float64        `json:"fill_ratio"`
}

type ScoredAsk struct {
	Ask        entity.Ask     `json:"ask"`
	Score      float64        `json:"score"`
	Breakdown  ScoreBreakdown `json:"breakdown"`
	DeadheadKm float64        `json:"deadhead_km"`
	DetourKm   float64        `json:"detour_km"`
	FillRatio  float64        `json:"fill_ratio"`
}

func RankBidsForAsk(ask *entity.Ask, bids []entity.Bid, w ScoreWeights) []ScoredBid {
	if ask == nil {
		return nil
	}

	ranked := make([]ScoredBid, 0, len(bids))
	for _, bid := range bids {
		if !fitsCapacity(ask, &bid) {
			continue
		}

		m := measure(ask, &bid)
		breakdown := ScoreBreakdown{
			Deadhead:  deadheadScore(m.deadheadKm, m.truckDirectKm, m.aheadOfTruck),
			Alignment: alignmentScore(ask, &bid),
			Detour:    detourScore(m.detourKm, m.truckDirectKm),
			Fill:      fillScore(ask, &bid),
			Price:     priceScore(ask, &bid),
		}

		total := w.Deadhead*breakdown.Deadhead +
			w.Alignment*breakdown.Alignment +
			w.Detour*breakdown.Detour +
			w.Fill*breakdown.Fill +
			w.Price*breakdown.Price

		if total < minAcceptableScore {
			continue
		}

		ranked = append(ranked, ScoredBid{
			Bid:        bid,
			Score:      round4(total),
			Breakdown:  roundBreakdown(breakdown),
			DeadheadKm: round2(m.deadheadKm),
			DetourKm:   round2(m.detourKm),
			FillRatio:  round4(fillRatio(ask, &bid)),
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}

func RankAsksForBid(bid *entity.Bid, asks []entity.Ask, w ScoreWeights) []ScoredAsk {
	if bid == nil {
		return nil
	}

	ranked := make([]ScoredAsk, 0, len(asks))
	for _, ask := range asks {
		if !fitsCapacity(&ask, bid) {
			continue
		}

		m := measure(&ask, bid)
		breakdown := ScoreBreakdown{
			Deadhead:  deadheadScore(m.deadheadKm, m.truckDirectKm, m.aheadOfTruck),
			Alignment: alignmentScore(&ask, bid),
			Detour:    detourScore(m.detourKm, m.truckDirectKm),
			Fill:      fillScore(&ask, bid),
			Price:     priceScore(&ask, bid),
		}

		total := w.Deadhead*breakdown.Deadhead +
			w.Alignment*breakdown.Alignment +
			w.Detour*breakdown.Detour +
			w.Fill*breakdown.Fill +
			w.Price*breakdown.Price

		if total < minAcceptableScore {
			continue
		}

		ranked = append(ranked, ScoredAsk{
			Ask:        ask,
			Score:      round4(total),
			Breakdown:  roundBreakdown(breakdown),
			DeadheadKm: round2(m.deadheadKm),
			DetourKm:   round2(m.detourKm),
			FillRatio:  round4(fillRatio(&ask, bid)),
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}

type measurement struct {
	deadheadKm    float64
	truckDirectKm float64
	detourKm      float64
	aheadOfTruck  bool
}

func measure(ask *entity.Ask, bid *entity.Bid) measurement {
	offRoute, progress := offRouteKm(ask.CurrentLocation, ask.Destination, bid.Origin)

	cargoLeg := haversineKm(bid.Origin, bid.Destination)
	returnLeg := haversineKm(bid.Destination, ask.Destination)
	fromTruck := haversineKm(ask.CurrentLocation, bid.Origin)
	direct := haversineKm(ask.CurrentLocation, ask.Destination)

	return measurement{
		deadheadKm:    offRoute,
		truckDirectKm: direct,
		detourKm:      math.Max(0, fromTruck+cargoLeg+returnLeg-direct),
		aheadOfTruck:  progress >= 0,
	}
}

func offRouteKm(from, to, point entity.Location) (distanceKm, progress float64) {
	route := vectorBetween(from, to)
	toPoint := vectorBetween(from, point)

	routeLenSq := route.x*route.x + route.y*route.y
	if routeLenSq == 0 {
		return haversineKm(from, point), 0
	}

	progress = (toPoint.x*route.x + toPoint.y*route.y) / routeLenSq
	nearest := entity.Location{
		Latitude:  from.Latitude + clamp01(progress)*(to.Latitude-from.Latitude),
		Longitude: from.Longitude + clamp01(progress)*(to.Longitude-from.Longitude),
	}
	return haversineKm(point, nearest), progress
}

func deadheadScore(offRouteKm, tripKm float64, aheadOfTruck bool) float64 {
	budget := offRouteBudgetKm(tripKm)
	if offRouteKm >= budget {
		return 0
	}
	score := clamp01(1 - offRouteKm/budget)
	if !aheadOfTruck {
		score *= behindTruckPenalty
	}
	return score
}

func alignmentScore(ask *entity.Ask, bid *entity.Bid) float64 {
	truck := vectorBetween(ask.CurrentLocation, ask.Destination)
	cargo := vectorBetween(bid.Origin, bid.Destination)

	truckLen := math.Hypot(truck.x, truck.y)
	cargoLen := math.Hypot(cargo.x, cargo.y)
	if truckLen == 0 || cargoLen == 0 {
		return 0.5
	}

	cosine := (truck.x*cargo.x + truck.y*cargo.y) / (truckLen * cargoLen)
	return clamp01((cosine + 1) / 2)
}

func detourScore(detourKm, directKm float64) float64 {
	if directKm <= 0 {
		return 0.5
	}
	ratio := detourKm / directKm
	if ratio >= maxDetourRatio {
		return 0
	}
	return clamp01(1 - ratio/maxDetourRatio)
}

func fillScore(ask *entity.Ask, bid *entity.Bid) float64 {
	return clamp01(fillRatio(ask, bid))
}

func fillRatio(ask *entity.Ask, bid *entity.Bid) float64 {
	byWeight, byVolume := 0.0, 0.0
	if ask.AvailableWeightKg > 0 {
		byWeight = bid.WeightKg / ask.AvailableWeightKg
	}
	if ask.AvailableVolumeM3 > 0 {
		byVolume = bid.VolumeM3 / ask.AvailableVolumeM3
	}
	return math.Max(byWeight, byVolume)
}

func priceScore(ask *entity.Ask, bid *entity.Bid) float64 {
	if bid.MaxPrice <= 0 || bid.MaxPrice < ask.MinPrice {
		return 0
	}
	return clamp01((bid.MaxPrice - ask.MinPrice) / bid.MaxPrice)
}

func fitsCapacity(ask *entity.Ask, bid *entity.Bid) bool {
	if bid.WeightKg > ask.AvailableWeightKg || bid.VolumeM3 > ask.AvailableVolumeM3 {
		return false
	}
	return bid.MaxPrice >= ask.MinPrice
}

type vector struct{ x, y float64 }

func vectorBetween(from, to entity.Location) vector {
	midLatRad := (from.Latitude + to.Latitude) / 2 * math.Pi / 180
	return vector{
		x: (to.Longitude - from.Longitude) * math.Cos(midLatRad),
		y: to.Latitude - from.Latitude,
	}
}

func haversineKm(from, to entity.Location) float64 {
	lat1 := from.Latitude * math.Pi / 180
	lat2 := to.Latitude * math.Pi / 180
	dLat := lat2 - lat1
	dLng := (to.Longitude - from.Longitude) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func roundBreakdown(b ScoreBreakdown) ScoreBreakdown {
	return ScoreBreakdown{
		Deadhead:  round4(b.Deadhead),
		Alignment: round4(b.Alignment),
		Detour:    round4(b.Detour),
		Fill:      round4(b.Fill),
		Price:     round4(b.Price),
	}
}

func offRouteBudgetKm(tripKm float64) float64 {
	budget := tripKm * offRouteShareOfTrip
	return math.Max(minOffRouteBudgetKm, math.Min(maxOffRouteBudgetKm, budget))
}
