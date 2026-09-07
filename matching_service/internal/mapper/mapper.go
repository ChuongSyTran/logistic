package mapper

import (
	"time"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/matching_service/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"matching_service/ent"
	"matching_service/internal/entity"
)

// goverter:converter
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreUnexported
// goverter:extend IdentityTime
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.4 gen ./
type MatchingMapper interface {
	// goverter:map . CurrentLocation | MapAskCurrentLocation
	// goverter:map . Destination | MapAskDestination
	// goverter:map MinPrice | Float64PtrToFloat64
	// goverter:ignore ExpiresAt
	EntAskToEntityAsk(source *ent.Asks) entity.Ask
	EntAskListToEntityAskList(source []*ent.Asks) []entity.Ask

	// goverter:map . Origin | MapBidOrigin
	// goverter:map . Destination | MapBidDestination
	// goverter:map MaxPrice | Float64PtrToFloat64
	// goverter:ignore ExpiresAt
	EntBidToEntityBid(source *ent.Bids) entity.Bid
	EntBidListToEntityBidList(source []*ent.Bids) []entity.Bid

	// goverter:ignore ID
	// goverter:ignore CreatedAt
	// goverter:ignore Status
	// Trạng thái thương lượng do server ghi, không nhận từ client.
	// goverter:ignore OfferedPrice
	// goverter:ignore OfferedAskID
	// goverter:map ShipperId ShipperID | BytesToUUID
	// goverter:map ConsigneeId ConsigneeID | BytesToUUID
	// goverter:map ExpiresAt ExpiresAt | TimestampToTime
	PbBidToEntity(req *pb.Bid) (entity.Bid, error)

	// goverter:ignore ID
	// goverter:ignore CreatedAt
	// goverter:ignore Status
	// goverter:ignore VehicleType
	// goverter:ignore CapacityWeightKg
	// goverter:ignore CapacityVolumeCbm
	// goverter:ignore RouteID
	// goverter:map VehicleId VehicleID | BytesToUUID
	// goverter:map DriverId DriverID | BytesToUUID
	// goverter:map ExpiresAt ExpiresAt | TimestampToTime
	PbAskToEntity(req *pb.Ask) (entity.Ask, error)

	// goverter:map ZoneId ZoneID
	MapLocation(loc *pb.Location) entity.Location

	// goverter:map ZoneID ZoneId
	MapEntityLocationToPb(loc entity.Location) *pb.Location

	// goverter:map ID Id | UUIDToBytes
	// goverter:map ShipperID ShipperId | UUIDToBytes
	// goverter:map ConsigneeID ConsigneeId | UUIDToBytes
	// goverter:map Status Status | Int8ToInt32
	// goverter:map ExpiresAt ExpiresAt | TimeToTimestamp
	// goverter:map CreatedAt CreatedAt | TimeToTimestamp
	EntityBidToPbBid(source entity.Bid) *pb.Bid
	EntityBidListToPbBidList(source []entity.Bid) []*pb.Bid

	// goverter:map ID Id | UUIDToBytes
	// goverter:map VehicleID VehicleId | UUIDToBytes
	// goverter:map DriverID DriverId | UUIDToBytes
	// goverter:map Status Status | Int8ToInt32
	// goverter:map ExpiresAt ExpiresAt | TimeToTimestamp
	// goverter:map CreatedAt CreatedAt | TimeToTimestamp
	EntityAskToPbAsk(source entity.Ask) *pb.Ask
	EntityAskListToPbAskList(source []entity.Ask) []*pb.Ask
}

func Float64PtrToFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func IdentityTime(t time.Time) time.Time {
	return t
}

func MapAskCurrentLocation(source *ent.Asks) entity.Location {
	if source == nil {
		return entity.Location{}
	}
	return entity.Location{
		Latitude:  source.OriginLat,
		Longitude: source.OriginLng,
		ZoneID:    source.ZoneID,
	}
}

func MapAskDestination(source *ent.Asks) entity.Location {
	if source == nil {
		return entity.Location{}
	}
	return entity.Location{
		Latitude:  source.DestinationLat,
		Longitude: source.DestinationLng,
	}
}

func MapBidOrigin(source *ent.Bids) entity.Location {
	if source == nil {
		return entity.Location{}
	}
	return entity.Location{
		Latitude:  source.OriginLat,
		Longitude: source.OriginLng,
		ZoneID:    source.ZoneID,
	}
}

func MapBidDestination(source *ent.Bids) entity.Location {
	if source == nil {
		return entity.Location{}
	}
	return entity.Location{
		Latitude:  source.DestinationLat,
		Longitude: source.DestinationLng,
	}
}

func BytesToUUID(b []byte) (uuid.UUID, error) {
	if len(b) == 0 {
		return uuid.Nil, nil
	}
	return uuid.FromBytes(b)
}

func UUIDToBytes(u uuid.UUID) []byte {
	return u[:]
}

func Int8ToInt32(i int8) int32 {
	return int32(i)
}

func TimestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts != nil && ts.IsValid() {
		return ts.AsTime()
	}
	return time.Time{}
}

func TimeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}