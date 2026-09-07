package app

import (
	"context"

	"vehicle_service/internal/entity"

	"github.com/google/uuid"
)

// Trước đây là MỘT interface 23 method bao phủ toàn bộ xe, giấy tờ, vị trí GPS và tìm kiếm GEO.
// Tách theo aggregate/trách nhiệm để mỗi use case hay adapter chỉ phụ thuộc vào phần nó thật sự dùng.
//
// VehicleRepo tồn tại như một interface hợp thành: adapter persistence cài đặt một lần,
// còn phía dùng nhận đúng port hẹp mà nó cần.

type VehicleRepository interface {
	CreateVehicle(ctx context.Context, param *entity.RegisterVehicleParam) (*entity.Vehicle, error)
	GetVehicleByID(ctx context.Context, id uuid.UUID) (*entity.Vehicle, error)
	GetVehiclesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Vehicle, error)
	ListVehicles(ctx context.Context, param *entity.ListVehiclesParam) ([]entity.Vehicle, int64, error)
	AdminListVehicles(ctx context.Context, param *entity.AdminListVehiclesParam) ([]entity.Vehicle, int64, error)
	UpdateVehicle(ctx context.Context, param *entity.UpdateVehicleParam) (*entity.Vehicle, error)
	UpdateVehicleStatus(ctx context.Context, id uuid.UUID, status string) (*entity.Vehicle, error)
	UpdateVerification(ctx context.Context, param *entity.VerifyVehicleParam, status string) (*entity.Vehicle, error)
	DeleteVehicle(ctx context.Context, id uuid.UUID) error
	CountVehicles(ctx context.Context, status, verificationStatus string) (int64, error)
}

type DocumentRepository interface {
	CreateDocument(ctx context.Context, param *entity.UploadDocumentParam) (*entity.VehicleDocument, error)
	GetDocument(ctx context.Context, id uuid.UUID) (*entity.VehicleDocument, error)
	ListDocuments(ctx context.Context, param *entity.ListDocumentsParam) ([]entity.VehicleDocument, error)
	ListPendingDocuments(ctx context.Context, page, pageSize int) ([]entity.VehicleDocument, int64, error)
	ReviewDocument(ctx context.Context, param *entity.ReviewDocumentParam, status string) (*entity.VehicleDocument, error)
	DeleteDocument(ctx context.Context, id uuid.UUID) error
	CountPendingDocuments(ctx context.Context) (int64, error)
}

type LocationRepository interface {
	UpsertLocation(ctx context.Context, param *entity.ReportLocationParam, zoneID string) (*entity.VehicleLocation, error)
	GetLocation(ctx context.Context, vehicleID uuid.UUID) (*entity.VehicleLocation, error)
}

type AvailabilityRepository interface {
	UpsertAvailability(ctx context.Context, param *entity.SetAvailabilityParam, zoneID string) (*entity.DriverAvailability, error)
	GetAvailability(ctx context.Context, driverID uuid.UUID) (*entity.DriverAvailability, error)
	GetAvailabilitiesByVehicleIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.DriverAvailability, error)
	CountOnlineDrivers(ctx context.Context) (int64, error)
}

type GeoSearchRepository interface {
	SearchNearby(ctx context.Context, param *entity.SearchNearbyParam) ([]entity.NearbyVehicle, error)
}

// VehicleRepo gom lại để di và persistence adapter cài đặt một lần.
type VehicleRepo interface {
	VehicleRepository
	DocumentRepository
	LocationRepository
	AvailabilityRepository
	GeoSearchRepository
}