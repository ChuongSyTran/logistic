package mapper

import (
	"time"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/vehicle_service/v1"
	"github.com/logistic/pkg/uuidx"
	"google.golang.org/protobuf/types/known/timestamppb"

	"vehicle_service/ent"
	"vehicle_service/ent/vehicle"
	"vehicle_service/ent/vehicledocument"
	"vehicle_service/internal/entity"
)

// goverter:converter
// goverter:matchIgnoreCase
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreUnexported
// goverter:extend IdentityTime
// goverter:extend UUIDToBytes
// goverter:extend BytesToUUID
// goverter:extend TimeToTimestamp
// goverter:extend TimestampToTime
// goverter:extend TimePtrToTime
// goverter:extend IntToInt32
// goverter:extend Int32ToInt
// goverter:extend EntVehicleTypeToString
// goverter:extend EntVehicleStatusToString
// goverter:extend EntVerificationStatusToString
// goverter:extend EntDocumentTypeToString
// goverter:extend EntReviewStatusToString
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.4 gen ./
type AppMapper interface {
	EntVehicleToEntityVehicle(source *ent.Vehicle) entity.Vehicle
	EntVehicleListToEntityVehicleList(source []*ent.Vehicle) []entity.Vehicle

	EntDocumentToEntityDocument(source *ent.VehicleDocument) entity.VehicleDocument
	EntDocumentListToEntityList(source []*ent.VehicleDocument) []entity.VehicleDocument

	EntLocationToEntityLocation(source *ent.VehicleLocation) entity.VehicleLocation

	EntAvailabilityToEntity(source *ent.DriverAvailability) entity.DriverAvailability
	EntAvailabilityListToEntityList(source []*ent.DriverAvailability) []entity.DriverAvailability

	EntityVehicleToPbVehicle(source entity.Vehicle) *pb.Vehicle
	EntityVehicleListToPbVehicleList(source []entity.Vehicle) []*pb.Vehicle

	EntityDocumentToPb(source entity.VehicleDocument) *pb.VehicleDocument
	EntityDocumentListToPbList(source []entity.VehicleDocument) []*pb.VehicleDocument

	EntityLocationToPb(source entity.VehicleLocation) *pb.VehicleLocation

	EntityAvailabilityToPb(source entity.DriverAvailability) *pb.DriverAvailability

	EntityNearbyToPb(source entity.NearbyVehicle) *pb.NearbyVehicle
	EntityNearbyListToPbList(source []entity.NearbyVehicle) []*pb.NearbyVehicle

	EntityPaginationToPb(source entity.Pagination) *pb.Pagination

	PbRegisterVehicleToParam(req *pb.RegisterVehicleRequest) (entity.RegisterVehicleParam, error)

	PbUpdateVehicleToParam(req *pb.UpdateVehicleRequest) (entity.UpdateVehicleParam, error)

	PbListVehiclesToParam(req *pb.ListVehiclesRequest) (entity.ListVehiclesParam, error)

	PbAdminListVehiclesToParam(req *pb.AdminListVehiclesRequest) entity.AdminListVehiclesParam

	PbUploadDocumentToParam(req *pb.UploadVehicleDocumentRequest) (entity.UploadDocumentParam, error)

	PbListDocumentsToParam(req *pb.ListVehicleDocumentsRequest) (entity.ListDocumentsParam, error)

	// goverter:ignore ReviewerID
	PbReviewDocumentToParam(req *pb.AdminReviewDocumentRequest) (entity.ReviewDocumentParam, error)

	PbReportLocationToParam(req *pb.ReportLocationRequest) (entity.ReportLocationParam, error)

	PbSetAvailabilityToParam(req *pb.SetDriverAvailabilityRequest) (entity.SetAvailabilityParam, error)

	PbSearchNearbyToParam(req *pb.SearchNearbyVehiclesRequest) entity.SearchNearbyParam

	// goverter:ignore ReviewerID
	PbVerifyVehicleToParam(req *pb.AdminVerifyVehicleRequest) (entity.VerifyVehicleParam, error)
}

func IdentityTime(t time.Time) time.Time { return t }

func UUIDToBytes(u uuid.UUID) []byte {
	if u == uuid.Nil {
		return nil
	}
	return uuidx.ToBytes(u)
}

func BytesToUUID(b []byte) (uuid.UUID, error) {
	return uuidx.FromBytes(b)
}

func TimeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func TimestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts != nil && ts.IsValid() {
		return ts.AsTime()
	}
	return time.Time{}
}

func TimePtrToTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func IntToInt32(i int) int32 { return int32(i) }
func Int32ToInt(i int32) int { return int(i) }

func EntVehicleTypeToString(t vehicle.VehicleType) string { return string(t) }
func EntVehicleStatusToString(s vehicle.Status) string    { return string(s) }
func EntVerificationStatusToString(s vehicle.VerificationStatus) string {
	return string(s)
}
func EntDocumentTypeToString(t vehicledocument.DocumentType) string { return string(t) }
func EntReviewStatusToString(s vehicledocument.ReviewStatus) string { return string(s) }