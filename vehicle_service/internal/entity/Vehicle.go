package entity

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

const (
	VehicleTypeTruck     = "truck"
	VehicleTypeVan       = "van"
	VehicleTypeBike      = "bike"
	VehicleTypeContainer = "container"
	VehicleTypeTrailer   = "trailer"
)

const (
	VehicleStatusActive      = "active"
	VehicleStatusMaintenance = "maintenance"
	VehicleStatusInactive    = "inactive"
)

const (
	VerificationPending  = "pending"
	VerificationVerified = "verified"
	VerificationRejected = "rejected"
)

const (
	DocTypeRegistration = "registration"
	DocTypeInspection   = "inspection"
	DocTypeInsurance    = "insurance"
	DocTypeLicense      = "license"
)

const (
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

func IsValidVehicleType(t string) bool {
	switch t {
	case VehicleTypeTruck, VehicleTypeVan, VehicleTypeBike, VehicleTypeContainer, VehicleTypeTrailer:
		return true
	}
	return false
}

func IsValidVehicleStatus(s string) bool {
	switch s {
	case VehicleStatusActive, VehicleStatusMaintenance, VehicleStatusInactive:
		return true
	}
	return false
}

func IsValidVerificationStatus(s string) bool {
	switch s {
	case VerificationPending, VerificationVerified, VerificationRejected:
		return true
	}
	return false
}

func IsValidDocumentType(t string) bool {
	switch t {
	case DocTypeRegistration, DocTypeInspection, DocTypeInsurance, DocTypeLicense:
		return true
	}
	return false
}

func IsValidReviewStatus(s string) bool {
	switch s {
	case ReviewPending, ReviewApproved, ReviewRejected:
		return true
	}
	return false
}

func IsValidCoordinate(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return false
	}
	return !(lat == 0 && lng == 0)
}

const ZoneSize = 0.05

func ComputeZoneID(lat, lng float64) string {
	if !IsValidCoordinate(lat, lng) {
		return ""
	}
	latCell := int(math.Floor(lat / ZoneSize))
	lngCell := int(math.Floor(lng / ZoneSize))
	return fmt.Sprintf("Z%d_%d", latCell, lngCell)
}

type Vehicle struct {
	ID                 uuid.UUID `json:"id"`
	DriverID           uuid.UUID `json:"driver_id"`
	LicensePlate       string    `json:"license_plate"`
	Brand              string    `json:"brand"`
	Model              string    `json:"model"`
	ManufactureYear    int       `json:"manufacture_year"`
	VehicleType        string    `json:"vehicle_type"`
	CapacityWeightKg   float64   `json:"capacity_weight_kg"`
	CapacityVolumeCbm  float64   `json:"capacity_volume_cbm"`
	Status             string    `json:"status"`
	VerificationStatus string    `json:"verification_status"`
	VerificationNote   string    `json:"verification_note"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type VehicleDocument struct {
	ID             uuid.UUID `json:"id"`
	VehicleID      uuid.UUID `json:"vehicle_id"`
	DocumentType   string    `json:"document_type"`
	DocumentNumber string    `json:"document_number"`
	FileURL        string    `json:"file_url"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	ReviewStatus   string    `json:"review_status"`
	ReviewNote     string    `json:"review_note"`
	CreatedAt      time.Time `json:"created_at"`
}

type VehicleLocation struct {
	VehicleID  uuid.UUID `json:"vehicle_id"`
	DriverID   uuid.UUID `json:"driver_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Heading    float64   `json:"heading"`
	SpeedKph   float64   `json:"speed_kph"`
	ZoneID     string    `json:"zone_id"`
	RecordedAt time.Time `json:"recorded_at"`
}

type DriverAvailability struct {
	ID                 uuid.UUID `json:"id"`
	DriverID           uuid.UUID `json:"driver_id"`
	VehicleID          uuid.UUID `json:"vehicle_id"`
	IsOnline           bool      `json:"is_online"`
	AvailableWeightKg  float64   `json:"available_weight_kg"`
	AvailableVolumeCbm float64   `json:"available_volume_cbm"`
	CurrentLat         float64   `json:"current_lat"`
	CurrentLng         float64   `json:"current_lng"`
	ZoneID             string    `json:"zone_id"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type NearbyVehicle struct {
	VehicleID          uuid.UUID `json:"vehicle_id"`
	DriverID           uuid.UUID `json:"driver_id"`
	LicensePlate       string    `json:"license_plate"`
	VehicleType        string    `json:"vehicle_type"`
	DistanceKm         float64   `json:"distance_km"`
	AvailableWeightKg  float64   `json:"available_weight_kg"`
	AvailableVolumeCbm float64   `json:"available_volume_cbm"`
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

func NormalizePaging(page, pageSize int) (int, int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

func BuildPagination(page, pageSize int, total int64) Pagination {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return Pagination{Page: page, PageSize: pageSize, TotalItems: total, TotalPages: totalPages}
}

type RegisterVehicleParam struct {
	DriverID          uuid.UUID
	LicensePlate      string
	Brand             string
	Model             string
	ManufactureYear   int
	VehicleType       string
	CapacityWeightKg  float64
	CapacityVolumeCbm float64
}

type UpdateVehicleParam struct {
	ID                uuid.UUID
	DriverID          uuid.UUID
	Brand             string
	Model             string
	ManufactureYear   int
	VehicleType       string
	CapacityWeightKg  float64
	CapacityVolumeCbm float64
}

type ListVehiclesParam struct {
	DriverID    uuid.UUID
	Status      string
	VehicleType string
	Page        int
	PageSize    int
}

type ListVehiclesResult struct {
	Vehicles   []Vehicle
	Pagination Pagination
}

type AdminListVehiclesParam struct {
	Status             string
	VerificationStatus string
	VehicleType        string
	Keyword            string
	Page               int
	PageSize           int
}

type UploadDocumentParam struct {
	VehicleID      uuid.UUID
	DriverID       uuid.UUID
	DocumentType   string
	DocumentNumber string
	FileURL        string
	IssuedAt       time.Time
	ExpiresAt      time.Time
}

type ListDocumentsParam struct {
	VehicleID    uuid.UUID
	DriverID     uuid.UUID
	ReviewStatus string
}

type ReviewDocumentParam struct {
	ID         uuid.UUID
	Approved   bool
	Note       string
	ReviewerID uuid.UUID
}

type ListDocumentsResult struct {
	Documents  []VehicleDocument
	Pagination Pagination
}

type ReportLocationParam struct {
	VehicleID uuid.UUID
	DriverID  uuid.UUID
	Latitude  float64
	Longitude float64
	Heading   float64
	SpeedKph  float64
}

type SetAvailabilityParam struct {
	DriverID           uuid.UUID
	VehicleID          uuid.UUID
	IsOnline           bool
	AvailableWeightKg  float64
	AvailableVolumeCbm float64
	CurrentLat         float64
	CurrentLng         float64
}

type SearchNearbyParam struct {
	Latitude     float64
	Longitude    float64
	RadiusKm     float64
	MinWeightKg  float64
	MinVolumeCbm float64
	VehicleType  string
	Limit        int
}

const (
	DefaultSearchRadiusKm = 5.0
	MaxSearchRadiusKm     = 100.0
	DefaultSearchLimit    = 50
	MaxSearchLimit        = 200
)

func (p *SearchNearbyParam) Normalize() {
	if p.RadiusKm <= 0 {
		p.RadiusKm = DefaultSearchRadiusKm
	}
	if p.RadiusKm > MaxSearchRadiusKm {
		p.RadiusKm = MaxSearchRadiusKm
	}
	if p.Limit <= 0 {
		p.Limit = DefaultSearchLimit
	}
	if p.Limit > MaxSearchLimit {
		p.Limit = MaxSearchLimit
	}
}

type VerifyVehicleParam struct {
	ID         uuid.UUID
	Approved   bool
	Note       string
	ReviewerID uuid.UUID
}

type VehicleStats struct {
	TotalVehicles       int64
	ActiveVehicles      int64
	MaintenanceVehicles int64
	PendingVerification int64
	OnlineDrivers       int64
	PendingDocuments    int64
}
