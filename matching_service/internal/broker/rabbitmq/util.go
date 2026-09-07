package rabbitmq

import (
	"encoding/json"
	"log"
	"math"
)

func toMap(v any) map[string]any {
	blob, err := json.Marshal(v)
	if err != nil {
		log.Printf("[notifier] marshal payload thất bại: %v", err)
		return nil
	}

	var out map[string]any
	if err := json.Unmarshal(blob, &out); err != nil {
		log.Printf("[notifier] unmarshal payload về map thất bại: %v", err)
		return nil
	}
	return out
}

const earthRadiusKm = 6371.0

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	if lat1 == 0 && lng1 == 0 {
		return 0
	}
	if lat2 == 0 && lng2 == 0 {
		return 0
	}

	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}