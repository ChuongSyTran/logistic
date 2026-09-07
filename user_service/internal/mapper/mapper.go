package mapper

import (
	"time"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/user_service/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"user_service/ent"
	"user_service/ent/address"
	"user_service/ent/driverprofile"
	"user_service/ent/user"
	"user_service/ent/userdevice"
	"user_service/internal/entity"

	"github.com/logistic/pkg/uuidx"
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
// goverter:extend StringPtrToString
// goverter:extend IntToInt32
// goverter:extend Int32ToInt
// goverter:extend EntUserRoleToString
// goverter:extend EntUserStatusToString
// goverter:extend EntKycStatusToString
// goverter:extend EntAddressTypeToString
// goverter:extend EntPlatformToString
//
//go:generate go run github.com/jmattheis/goverter/cmd/goverter@v1.9.4 gen ./
type AppMapper interface {
	EntUserToEntityUser(source *ent.User) entity.User
	EntUserListToEntityUserList(source []*ent.User) []entity.User

	EntDriverProfileToEntityDriverProfile(source *ent.DriverProfile) entity.DriverProfile
	EntDriverProfileListToEntityList(source []*ent.DriverProfile) []entity.DriverProfile

	EntShipperProfileToEntityShipperProfile(source *ent.ShipperProfile) entity.ShipperProfile

	EntAddressToEntityAddress(source *ent.Address) entity.Address
	EntAddressListToEntityAddressList(source []*ent.Address) []entity.Address

	EntUserDeviceToEntityUserDevice(source *ent.UserDevice) entity.UserDevice
	EntUserDeviceListToEntityList(source []*ent.UserDevice) []entity.UserDevice

	EntityUserToPbUser(source entity.User) *pb.User
	EntityUserListToPbUserList(source []entity.User) []*pb.User

	EntityDriverProfileToPb(source entity.DriverProfile) *pb.DriverProfile
	EntityDriverProfileListToPbList(source []entity.DriverProfile) []*pb.DriverProfile

	EntityShipperProfileToPb(source entity.ShipperProfile) *pb.ShipperProfile

	EntityAddressToPb(source entity.Address) *pb.Address
	EntityAddressListToPbList(source []entity.Address) []*pb.Address

	EntityUserDeviceToPb(source entity.UserDevice) *pb.UserDevice
	EntityUserDeviceListToPbList(source []entity.UserDevice) []*pb.UserDevice

	EntityPaginationToPb(source entity.Pagination) *pb.Pagination

	// goverter:ignore ID
	PbRegisterUserToParam(req *pb.RegisterUserRequest) entity.RegisterUserParam

	PbUpdateUserToParam(req *pb.UpdateUserRequest) (entity.UpdateUserParam, error)

	PbUpdateDriverProfileToParam(req *pb.UpdateDriverProfileRequest) (entity.UpdateDriverProfileParam, error)

	PbUpdateShipperProfileToParam(req *pb.UpdateShipperProfileRequest) (entity.UpdateShipperProfileParam, error)

	PbCreateAddressToParam(req *pb.CreateAddressRequest) (entity.CreateAddressParam, error)

	PbUpdateAddressToParam(req *pb.UpdateAddressRequest) (entity.UpdateAddressParam, error)

	PbListAddressesToParam(req *pb.ListAddressesRequest) (entity.ListAddressesParam, error)

	PbRegisterDeviceToParam(req *pb.RegisterDeviceRequest) (entity.RegisterDeviceParam, error)

	PbAdminListUsersToFilter(req *pb.AdminListUsersRequest) entity.ListUsersFilter

	PbAdminUpdateStatusToParam(req *pb.AdminUpdateUserStatusRequest) (entity.UpdateUserStatusParam, error)
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

func StringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func IntToInt32(i int) int32 { return int32(i) }
func Int32ToInt(i int32) int { return int(i) }

func EntUserRoleToString(r user.Role) string                { return string(r) }
func EntUserStatusToString(s user.Status) string            { return string(s) }
func EntKycStatusToString(s driverprofile.KycStatus) string { return string(s) }
func EntAddressTypeToString(t address.AddressType) string   { return string(t) }
func EntPlatformToString(p userdevice.Platform) string      { return string(p) }
