package app

import (
	"context"

	"vehicle_service/internal/entity"

	"github.com/google/uuid"
)

type VehicleEngine interface {
	RegisterVehicle(ctx context.Context, param *entity.RegisterVehicleParam) (*entity.Vehicle, error)
	GetVehicle(ctx context.Context, id, driverID uuid.UUID) (*entity.Vehicle, error)
	ListVehicles(ctx context.Context, param *entity.ListVehiclesParam) (*entity.ListVehiclesResult, error)
	UpdateVehicle(ctx context.Context, param *entity.UpdateVehicleParam) (*entity.Vehicle, error)
	DeleteVehicle(ctx context.Context, id, driverID uuid.UUID) error
	UpdateVehicleStatus(ctx context.Context, id, driverID uuid.UUID, status string) (*entity.Vehicle, error)

	UploadDocument(ctx context.Context, param *entity.UploadDocumentParam) (*entity.VehicleDocument, error)
	ListDocuments(ctx context.Context, param *entity.ListDocumentsParam) ([]entity.VehicleDocument, error)
	DeleteDocument(ctx context.Context, id, driverID uuid.UUID) error

	ReportLocation(ctx context.Context, param *entity.ReportLocationParam) (*entity.VehicleLocation, error)
	GetLocation(ctx context.Context, vehicleID, driverID uuid.UUID) (*entity.VehicleLocation, error)

	SetAvailability(ctx context.Context, param *entity.SetAvailabilityParam) (*entity.DriverAvailability, error)
	GetAvailability(ctx context.Context, driverID uuid.UUID) (*entity.DriverAvailability, error)

	SearchNearby(ctx context.Context, param *entity.SearchNearbyParam) ([]entity.NearbyVehicle, error)

	AdminListVehicles(ctx context.Context, param *entity.AdminListVehiclesParam) (*entity.ListVehiclesResult, error)
	AdminVerifyVehicle(ctx context.Context, param *entity.VerifyVehicleParam) (*entity.Vehicle, error)
	AdminListPendingDocuments(ctx context.Context, page, pageSize int) (*entity.ListDocumentsResult, error)
	AdminReviewDocument(ctx context.Context, param *entity.ReviewDocumentParam) (*entity.VehicleDocument, error)
	AdminGetStats(ctx context.Context) (*entity.VehicleStats, error)
}

type vehicleEngineImpl struct {
	repo VehicleRepo
}

func NewVehicleEngine(repo VehicleRepo) VehicleEngine {
	return &vehicleEngineImpl{repo: repo}
}

func (e *vehicleEngineImpl) RegisterVehicle(ctx context.Context, param *entity.RegisterVehicleParam) (*entity.Vehicle, error) {
	if param.DriverID == uuid.Nil {
		return nil, entity.ErrInvalidDriverID
	}
	if param.LicensePlate == "" {
		return nil, entity.ErrPlateRequired
	}
	if !entity.IsValidVehicleType(param.VehicleType) {
		return nil, entity.ErrInvalidType.WithDetail("vehicle_type", param.VehicleType)
	}

	if param.CapacityWeightKg <= 0 || param.CapacityVolumeCbm <= 0 {
		return nil, entity.ErrInvalidCapacity
	}
	return e.repo.CreateVehicle(ctx, param)
}

// Mọi thao tác của tài xế phải qua đây. driverID rỗng = luồng quản trị, bỏ kiểm tra.
func (e *vehicleEngineImpl) ownedVehicle(ctx context.Context, vehicleID, driverID uuid.UUID) (*entity.Vehicle, error) {
	if vehicleID == uuid.Nil {
		return nil, entity.ErrInvalidVehicleID
	}
	v, err := e.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if driverID != uuid.Nil && v.DriverID != driverID {
		return nil, entity.ErrVehicleNotOwned
	}
	return v, nil
}

func (e *vehicleEngineImpl) GetVehicle(ctx context.Context, id, driverID uuid.UUID) (*entity.Vehicle, error) {
	return e.ownedVehicle(ctx, id, driverID)
}

func (e *vehicleEngineImpl) ListVehicles(ctx context.Context, param *entity.ListVehiclesParam) (*entity.ListVehiclesResult, error) {
	if param.Status != "" && !entity.IsValidVehicleStatus(param.Status) {
		return nil, entity.ErrInvalidStatus.WithDetail("status", param.Status)
	}
	if param.VehicleType != "" && !entity.IsValidVehicleType(param.VehicleType) {
		return nil, entity.ErrInvalidType.WithDetail("vehicle_type", param.VehicleType)
	}

	page, pageSize, _ := entity.NormalizePaging(param.Page, param.PageSize)
	list, total, err := e.repo.ListVehicles(ctx, param)
	if err != nil {
		return nil, err
	}
	return &entity.ListVehiclesResult{
		Vehicles:   list,
		Pagination: entity.BuildPagination(page, pageSize, total),
	}, nil
}

func (e *vehicleEngineImpl) UpdateVehicle(ctx context.Context, param *entity.UpdateVehicleParam) (*entity.Vehicle, error) {
	if param.ID == uuid.Nil {
		return nil, entity.ErrInvalidVehicleID
	}
	if param.VehicleType != "" && !entity.IsValidVehicleType(param.VehicleType) {
		return nil, entity.ErrInvalidType.WithDetail("vehicle_type", param.VehicleType)
	}
	if _, err := e.ownedVehicle(ctx, param.ID, param.DriverID); err != nil {
		return nil, err
	}
	return e.repo.UpdateVehicle(ctx, param)
}

func (e *vehicleEngineImpl) DeleteVehicle(ctx context.Context, id, driverID uuid.UUID) error {
	if _, err := e.ownedVehicle(ctx, id, driverID); err != nil {
		return err
	}
	return e.repo.DeleteVehicle(ctx, id)
}

func (e *vehicleEngineImpl) UpdateVehicleStatus(ctx context.Context, id, driverID uuid.UUID, status string) (*entity.Vehicle, error) {
	if !entity.IsValidVehicleStatus(status) {
		return nil, entity.ErrInvalidStatus.WithDetail("status", status)
	}
	if _, err := e.ownedVehicle(ctx, id, driverID); err != nil {
		return nil, err
	}
	return e.repo.UpdateVehicleStatus(ctx, id, status)
}

func (e *vehicleEngineImpl) UploadDocument(ctx context.Context, param *entity.UploadDocumentParam) (*entity.VehicleDocument, error) {
	if param.VehicleID == uuid.Nil {
		return nil, entity.ErrInvalidVehicleID
	}
	if !entity.IsValidDocumentType(param.DocumentType) {
		return nil, entity.ErrInvalidDocType.WithDetail("document_type", param.DocumentType)
	}
	if param.FileURL == "" {
		return nil, entity.ErrFileURLRequired
	}
	if _, err := e.ownedVehicle(ctx, param.VehicleID, param.DriverID); err != nil {
		return nil, err
	}
	return e.repo.CreateDocument(ctx, param)
}

func (e *vehicleEngineImpl) ListDocuments(ctx context.Context, param *entity.ListDocumentsParam) ([]entity.VehicleDocument, error) {
	if param.ReviewStatus != "" && !entity.IsValidReviewStatus(param.ReviewStatus) {
		return nil, entity.ErrInvalidReviewStat.WithDetail("review_status", param.ReviewStatus)
	}
	if _, err := e.ownedVehicle(ctx, param.VehicleID, param.DriverID); err != nil {
		return nil, err
	}
	return e.repo.ListDocuments(ctx, param)
}

func (e *vehicleEngineImpl) DeleteDocument(ctx context.Context, id, driverID uuid.UUID) error {
	if id == uuid.Nil {
		return entity.ErrInvalidDocumentID
	}
	doc, err := e.repo.GetDocument(ctx, id)
	if err != nil {
		return err
	}
	// Giấy tờ không mang driver_id, phải qua xe mới biết chủ.
	if _, err := e.ownedVehicle(ctx, doc.VehicleID, driverID); err != nil {
		return err
	}
	return e.repo.DeleteDocument(ctx, id)
}

func (e *vehicleEngineImpl) ReportLocation(ctx context.Context, param *entity.ReportLocationParam) (*entity.VehicleLocation, error) {
	if param.VehicleID == uuid.Nil {
		return nil, entity.ErrInvalidVehicleID
	}
	if !entity.IsValidCoordinate(param.Latitude, param.Longitude) {
		return nil, entity.ErrInvalidCoordinate.
			WithDetail("latitude", formatFloat(param.Latitude)).
			WithDetail("longitude", formatFloat(param.Longitude))
	}

	v, err := e.ownedVehicle(ctx, param.VehicleID, param.DriverID)
	if err != nil {
		return nil, err
	}

	if param.DriverID == uuid.Nil {
		param.DriverID = v.DriverID
	}

	zoneID := entity.ComputeZoneID(param.Latitude, param.Longitude)
	return e.repo.UpsertLocation(ctx, param, zoneID)
}

func (e *vehicleEngineImpl) GetLocation(ctx context.Context, vehicleID, driverID uuid.UUID) (*entity.VehicleLocation, error) {
	if _, err := e.ownedVehicle(ctx, vehicleID, driverID); err != nil {
		return nil, err
	}
	return e.repo.GetLocation(ctx, vehicleID)
}

func (e *vehicleEngineImpl) SetAvailability(ctx context.Context, param *entity.SetAvailabilityParam) (*entity.DriverAvailability, error) {
	if param.DriverID == uuid.Nil {
		return nil, entity.ErrInvalidDriverID
	}
	if param.VehicleID == uuid.Nil {
		return nil, entity.ErrInvalidVehicleID
	}

	v, err := e.ownedVehicle(ctx, param.VehicleID, param.DriverID)
	if err != nil {
		return nil, err
	}

	if param.IsOnline {
		if v.VerificationStatus != entity.VerificationVerified {
			return nil, entity.ErrVehicleNotVerified.WithDetail("verification_status", v.VerificationStatus)
		}
		if v.Status == entity.VehicleStatusMaintenance {
			return nil, entity.ErrVehicleInMaintenance
		}
		if !entity.IsValidCoordinate(param.CurrentLat, param.CurrentLng) {
			return nil, entity.ErrInvalidCoordinate
		}

		if param.AvailableWeightKg <= 0 {
			param.AvailableWeightKg = v.CapacityWeightKg
		}
		if param.AvailableVolumeCbm <= 0 {
			param.AvailableVolumeCbm = v.CapacityVolumeCbm
		}

		if param.AvailableWeightKg > v.CapacityWeightKg {
			param.AvailableWeightKg = v.CapacityWeightKg
		}
		if param.AvailableVolumeCbm > v.CapacityVolumeCbm {
			param.AvailableVolumeCbm = v.CapacityVolumeCbm
		}
	}

	zoneID := entity.ComputeZoneID(param.CurrentLat, param.CurrentLng)
	return e.repo.UpsertAvailability(ctx, param, zoneID)
}

func (e *vehicleEngineImpl) GetAvailability(ctx context.Context, driverID uuid.UUID) (*entity.DriverAvailability, error) {
	if driverID == uuid.Nil {
		return nil, entity.ErrInvalidDriverID
	}
	return e.repo.GetAvailability(ctx, driverID)
}

func (e *vehicleEngineImpl) SearchNearby(ctx context.Context, param *entity.SearchNearbyParam) ([]entity.NearbyVehicle, error) {
	if !entity.IsValidCoordinate(param.Latitude, param.Longitude) {
		return nil, entity.ErrInvalidCoordinate
	}
	if param.VehicleType != "" && !entity.IsValidVehicleType(param.VehicleType) {
		return nil, entity.ErrInvalidType.WithDetail("vehicle_type", param.VehicleType)
	}
	param.Normalize()
	return e.repo.SearchNearby(ctx, param)
}

func (e *vehicleEngineImpl) AdminListVehicles(ctx context.Context, param *entity.AdminListVehiclesParam) (*entity.ListVehiclesResult, error) {
	if param.Status != "" && !entity.IsValidVehicleStatus(param.Status) {
		return nil, entity.ErrInvalidStatus.WithDetail("status", param.Status)
	}
	if param.VerificationStatus != "" && !entity.IsValidVerificationStatus(param.VerificationStatus) {
		return nil, entity.ErrInvalidReviewStat.WithDetail("verification_status", param.VerificationStatus)
	}
	if param.VehicleType != "" && !entity.IsValidVehicleType(param.VehicleType) {
		return nil, entity.ErrInvalidType.WithDetail("vehicle_type", param.VehicleType)
	}

	page, pageSize, _ := entity.NormalizePaging(param.Page, param.PageSize)
	list, total, err := e.repo.AdminListVehicles(ctx, param)
	if err != nil {
		return nil, err
	}
	return &entity.ListVehiclesResult{
		Vehicles:   list,
		Pagination: entity.BuildPagination(page, pageSize, total),
	}, nil
}

func (e *vehicleEngineImpl) AdminVerifyVehicle(ctx context.Context, param *entity.VerifyVehicleParam) (*entity.Vehicle, error) {
	if param.ID == uuid.Nil {
		return nil, entity.ErrInvalidVehicleID
	}
	if _, err := e.repo.GetVehicleByID(ctx, param.ID); err != nil {
		return nil, err
	}

	status := entity.VerificationRejected
	if param.Approved {
		status = entity.VerificationVerified
	}
	return e.repo.UpdateVerification(ctx, param, status)
}

func (e *vehicleEngineImpl) AdminListPendingDocuments(ctx context.Context, page, pageSize int) (*entity.ListDocumentsResult, error) {
	page, pageSize, _ = entity.NormalizePaging(page, pageSize)
	list, total, err := e.repo.ListPendingDocuments(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &entity.ListDocumentsResult{
		Documents:  list,
		Pagination: entity.BuildPagination(page, pageSize, total),
	}, nil
}

func (e *vehicleEngineImpl) AdminReviewDocument(ctx context.Context, param *entity.ReviewDocumentParam) (*entity.VehicleDocument, error) {
	if param.ID == uuid.Nil {
		return nil, entity.ErrInvalidDocumentID
	}

	current, err := e.repo.GetDocument(ctx, param.ID)
	if err != nil {
		return nil, err
	}
	if current.ReviewStatus != entity.ReviewPending {
		return nil, entity.ErrDocAlreadyReviewed.WithDetail("current_status", current.ReviewStatus)
	}

	status := entity.ReviewRejected
	if param.Approved {
		status = entity.ReviewApproved
	}
	return e.repo.ReviewDocument(ctx, param, status)
}

func (e *vehicleEngineImpl) AdminGetStats(ctx context.Context) (*entity.VehicleStats, error) {
	total, err := e.repo.CountVehicles(ctx, "", "")
	if err != nil {
		return nil, err
	}
	active, err := e.repo.CountVehicles(ctx, entity.VehicleStatusActive, "")
	if err != nil {
		return nil, err
	}
	maintenance, err := e.repo.CountVehicles(ctx, entity.VehicleStatusMaintenance, "")
	if err != nil {
		return nil, err
	}
	pendingVerify, err := e.repo.CountVehicles(ctx, "", entity.VerificationPending)
	if err != nil {
		return nil, err
	}
	online, err := e.repo.CountOnlineDrivers(ctx)
	if err != nil {
		return nil, err
	}
	pendingDocs, err := e.repo.CountPendingDocuments(ctx)
	if err != nil {
		return nil, err
	}

	return &entity.VehicleStats{
		TotalVehicles:       total,
		ActiveVehicles:      active,
		MaintenanceVehicles: maintenance,
		PendingVerification: pendingVerify,
		OnlineDrivers:       online,
		PendingDocuments:    pendingDocs,
	}, nil
}
