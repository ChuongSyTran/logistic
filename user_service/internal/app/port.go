package app

import (
	"context"

	"user_service/internal/entity"

	"github.com/google/uuid"
)

// Trước đây đây là MỘT interface 27 method phủ 5 aggregate. Tách theo aggregate
// để mỗi use case chỉ phụ thuộc vào phần nó thật sự dùng — quan trọng nhất là
// ComplianceRepository, cái đường nối để sau này tách KYC ra service riêng.
//
// UserRepo vẫn tồn tại như một interface hợp thành: adapter persistence cài đặt
// một lần, còn phía dùng thì nhận đúng cái nhỏ mà nó cần.

type UserRepository interface {
	CreateUser(ctx context.Context, u *entity.User) (*entity.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*entity.User, error)
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateUser(ctx context.Context, param *entity.UpdateUserParam) (*entity.User, error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, status, reason string) (*entity.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter *entity.ListUsersFilter) ([]entity.User, int64, error)
	CountUsers(ctx context.Context, role, status string) (int64, error)
}

type DriverProfileRepository interface {
	CreateDriverProfile(ctx context.Context, userID uuid.UUID, dp *entity.DriverProfile) (*entity.DriverProfile, error)
	GetDriverProfile(ctx context.Context, userID uuid.UUID) (*entity.DriverProfile, error)
	UpdateDriverProfile(ctx context.Context, param *entity.UpdateDriverProfileParam) (*entity.DriverProfile, error)
}

type ShipperProfileRepository interface {
	CreateShipperProfile(ctx context.Context, userID uuid.UUID, sp *entity.ShipperProfile) (*entity.ShipperProfile, error)
	GetShipperProfile(ctx context.Context, userID uuid.UUID) (*entity.ShipperProfile, error)
	UpdateShipperProfile(ctx context.Context, param *entity.UpdateShipperProfileParam) (*entity.ShipperProfile, error)
}

type AddressRepository interface {
	CreateAddress(ctx context.Context, param *entity.CreateAddressParam) (*entity.Address, error)
	GetAddress(ctx context.Context, id uuid.UUID) (*entity.Address, error)
	ListAddresses(ctx context.Context, param *entity.ListAddressesParam) ([]entity.Address, int64, error)
	UpdateAddress(ctx context.Context, param *entity.UpdateAddressParam) (*entity.Address, error)
	DeleteAddress(ctx context.Context, id uuid.UUID) error
	ClearDefaultAddress(ctx context.Context, userID uuid.UUID) error
}

type DeviceRepository interface {
	UpsertDevice(ctx context.Context, param *entity.RegisterDeviceParam) (*entity.UserDevice, error)
	GetDevice(ctx context.Context, id uuid.UUID) (*entity.UserDevice, error)
	ListDevices(ctx context.Context, userID uuid.UUID) ([]entity.UserDevice, error)
	DeleteDevice(ctx context.Context, id uuid.UUID) error
}

// UserRepo gom lại để di và adapter khai một lần. Đừng nhận kiểu này ở use case
// mới — nhận đúng port hẹp mà nó cần.
type UserRepo interface {
	UserRepository
	DriverProfileRepository
	ShipperProfileRepository
	AddressRepository
	DeviceRepository
}
