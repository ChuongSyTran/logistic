package persistence

import (
	"math"
	"sort"

	"vehicle_service/internal/entity"
)

const earthRadiusKm = 6371.0

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func toRadians(deg float64) float64 { return deg * math.Pi / 180 }

func neighborZones(lat, lng, radiusKm float64) []string {
	if !entity.IsValidCoordinate(lat, lng) {
		return nil
	}

	cellsNeeded := int(math.Ceil(radiusKm / (entity.ZoneSize * 111.0)))
	if cellsNeeded < 1 {
		cellsNeeded = 1
	}
	if cellsNeeded > 2 {
		return nil
	}

	zones := make([]string, 0, (2*cellsNeeded+1)*(2*cellsNeeded+1))
	for dLat := -cellsNeeded; dLat <= cellsNeeded; dLat++ {
		for dLng := -cellsNeeded; dLng <= cellsNeeded; dLng++ {
			z := entity.ComputeZoneID(
				lat+float64(dLat)*entity.ZoneSize,
				lng+float64(dLng)*entity.ZoneSize,
			)
			if z != "" {
				zones = append(zones, z)
			}
		}
	}
	return zones
}

func sortByDistance(list []entity.NearbyVehicle) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].DistanceKm < list[j].DistanceKm
	})
}