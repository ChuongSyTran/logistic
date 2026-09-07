package biz

import (
	"context"
	"fmt"
	"matching_service/internal/entity"

	"github.com/mmcloughlin/geohash"
)

type geoHashEngineImpl struct {
}

func NewGeoHashEngine() SpatialEngine {
	return &geoHashEngineImpl{}
}

func (g *geoHashEngineImpl) GetZoneId(ctx context.Context, lat, lng float64) (string, error) {
	if lat == 0 || lng == 0 {
		return "", fmt.Errorf("%w: Empty lattitude or longtitude", entity.ErrEmptyLocation)
	}

	result := geohash.EncodeWithPrecision(lat, lng, 10)
	if result == "" {
		return "", fmt.Errorf("%w: Invalid location", entity.ErrInvalidLocation)
	}

	return result, nil
}

func (g *geoHashEngineImpl) GetNeighborZones(ctx context.Context, zoneID string) ([]string, error) {
	if zoneID == "" {
		return nil, fmt.Errorf("%w: Empty zone ID", entity.ErrInvalidLocation)
	}
	neighbors := geohash.Neighbors(zoneID)

	return neighbors, nil
}
