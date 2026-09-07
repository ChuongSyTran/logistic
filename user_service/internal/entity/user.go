package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleDriver  = "driver"
	RoleShipper = "shipper"
	RoleAdmin   = "admin"
)

const (
	StatusActive    = "active"
	StatusBanned    = "banned"
	StatusSuspended = "suspended"
)

const (
	KycPending  = "pending"
	KycApproved = "approved"
	KycRejected = "rejected"
)

const (
	AddressTypePickup   = "pickup"
	AddressTypeDelivery = "delivery"
	AddressTypeBoth     = "both"
)

const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWeb     = "web"
)

func IsValidRole(r string) bool {
	return r == RoleDriver || r == RoleShipper || r == RoleAdmin
}

func IsValidStatus(s string) bool {
	return s == StatusActive || s == StatusBanned || s == StatusSuspended
}

func IsValidKycStatus(s string) bool {
	return s == KycPending || s == KycApproved || s == KycRejected
}

func IsValidAddressType(t string) bool {
	return t == AddressTypePickup || t == AddressTypeDelivery || t == AddressTypeBoth
}

func IsValidPlatform(p string) bool {
	return p == PlatformAndroid || p == PlatformIOS || p == PlatformWeb
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	AvatarURL    string    `json:"avatar_url"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	StatusReason string    `json:"status_reason"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DriverProfile struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	LicenseNumber string    `json:"license_number"`
	IDCard        string    `json:"id_card"`
	Rating        float64   `json:"rating"`
	TotalTrips    int       `json:"total_trips"`
	KycStatus     string    `json:"kyc_status"`
	KycNote       string    `json:"kyc_note"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ShipperProfile struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	CompanyName     string    `json:"company_name"`
	TaxCode         string    `json:"tax_code"`
	BusinessAddress string    `json:"business_address"`
	TotalOrders     int       `json:"total_orders"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Address struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Label        string    `json:"label"`
	ContactName  string    `json:"contact_name"`
	ContactPhone string    `json:"contact_phone"`
	Line1        string    `json:"line1"`
	Ward         string    `json:"ward"`
	District     string    `json:"district"`
	City         string    `json:"city"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	AddressType  string    `json:"address_type"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserDevice struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	DeviceToken string    `json:"device_token"`
	Platform    string    `json:"platform"`
	DeviceName  string    `json:"device_name"`
	IsActive    bool      `json:"is_active"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	CreatedAt   time.Time `json:"created_at"`
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
	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

type RegisterUserParam struct {
	// Danh tính auth_service cấp; bỏ trống thì user_service tự sinh.
	ID       uuid.UUID
	Phone    string
	Email    string
	Password string
	Role     string
	FullName string
}

// Khi đúng: auth_service giữ credential, nên không đòi mật khẩu lẫn phone.
func (p *RegisterUserParam) ProvisionedFromAuth() bool {
	return p.ID != uuid.Nil
}

type RegisterUserResult struct {
	ID      uuid.UUID
	User    *User
	Message string
}

type GetUserResult struct {
	User           *User
	DriverProfile  *DriverProfile
	ShipperProfile *ShipperProfile
}

type UpdateUserParam struct {
	ID        uuid.UUID
	FullName  string
	Email     string
	AvatarURL string
}

type UpdateDriverProfileParam struct {
	UserID        uuid.UUID
	LicenseNumber string
	IDCard        string
}

type UpdateShipperProfileParam struct {
	UserID          uuid.UUID
	CompanyName     string
	TaxCode         string
	BusinessAddress string
}

type UpdateDriverKYCParam struct {
	UserID     uuid.UUID
	KycStatus  string
	Note       string
	ReviewerID uuid.UUID
}

type CreateAddressParam struct {
	UserID       uuid.UUID
	Label        string
	ContactName  string
	ContactPhone string
	Line1        string
	Ward         string
	District     string
	City         string
	Latitude     float64
	Longitude    float64
	AddressType  string
	IsDefault    bool
}

type UpdateAddressParam struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Label        string
	ContactName  string
	ContactPhone string
	Line1        string
	Ward         string
	District     string
	City         string
	Latitude     float64
	Longitude    float64
	AddressType  string
	IsDefault    bool
}

type ListAddressesParam struct {
	UserID      uuid.UUID
	AddressType string
	Page        int
	PageSize    int
}

type RegisterDeviceParam struct {
	UserID      uuid.UUID
	DeviceToken string
	Platform    string
	DeviceName  string
}

type ListUsersFilter struct {
	Role     string
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

type ListUsersResult struct {
	Users      []User
	Pagination Pagination
}

type ListAddressesResult struct {
	Addresses  []Address
	Pagination Pagination
}

type ListDriverProfilesResult struct {
	DriverProfiles []DriverProfile
	Pagination     Pagination
}

type UpdateUserStatusParam struct {
	ID     uuid.UUID
	Status string
	Reason string
}

type ReviewKYCParam struct {
	UserID     uuid.UUID
	Approved   bool
	Note       string
	ReviewerID uuid.UUID
}

type UserStats struct {
	TotalUsers    int64
	TotalDrivers  int64
	TotalShippers int64
	ActiveUsers   int64
	BannedUsers   int64
	PendingKyc    int64
}
