package grpcserver

import (
	"context"
	"user_service/internal/entity"

	"user_service/internal/app"
	"user_service/internal/mapper"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/user_service/v1"
	"github.com/logistic/pkg/uuidx"
)

type userController struct {
	pb.UnimplementedUserServiceServer
	engine app.UserEngine
	mapper mapper.AppMapper
}

func NewUserServer(engine app.UserEngine, appMapper mapper.AppMapper) pb.UserServiceServer {
	return &userController{engine: engine, mapper: appMapper}
}

func parseID(raw []byte, invalid error) (uuid.UUID, error) {
	id, err := uuidx.FromBytes(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, invalid
	}
	return id, nil
}

func (c *userController) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	param := c.mapper.PbRegisterUserToParam(req)

	// Mapper bỏ qua ID vì đây là trường tuỳ chọn.
	if len(req.Id) > 0 {
		id, err := parseID(req.Id, entity.ErrInvalidUserID)
		if err != nil {
			return nil, err
		}
		param.ID = id
	}

	res, err := c.engine.RegisterUser(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterUserResponse{
		Id:      uuidx.ToBytes(res.ID),
		Message: res.Message,
		User:    c.mapper.EntityUserToPbUser(*res.User),
	}, nil
}

func (c *userController) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	res, err := c.engine.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetUserResponse{User: c.mapper.EntityUserToPbUser(*res.User)}
	if res.DriverProfile != nil {
		resp.DriverProfile = c.mapper.EntityDriverProfileToPb(*res.DriverProfile)
	}
	if res.ShipperProfile != nil {
		resp.ShipperProfile = c.mapper.EntityShipperProfileToPb(*res.ShipperProfile)
	}
	return resp, nil
}

func (c *userController) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	param, err := c.mapper.PbUpdateUserToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	u, err := c.engine.UpdateUser(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateUserResponse{
		User:    c.mapper.EntityUserToPbUser(*u),
		Message: "Cập nhật thông tin thành công",
	}, nil
}

func (c *userController) GetDriverProfile(ctx context.Context, req *pb.GetDriverProfileRequest) (*pb.GetDriverProfileResponse, error) {
	id, err := parseID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	dp, err := c.engine.GetDriverProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.GetDriverProfileResponse{DriverProfile: c.mapper.EntityDriverProfileToPb(*dp)}, nil
}

func (c *userController) UpdateDriverProfile(ctx context.Context, req *pb.UpdateDriverProfileRequest) (*pb.UpdateDriverProfileResponse, error) {
	param, err := c.mapper.PbUpdateDriverProfileToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	dp, err := c.engine.UpdateDriverProfile(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateDriverProfileResponse{
		DriverProfile: c.mapper.EntityDriverProfileToPb(*dp),
		Message:       "Cập nhật hồ sơ tài xế thành công",
	}, nil
}

func (c *userController) GetShipperProfile(ctx context.Context, req *pb.GetShipperProfileRequest) (*pb.GetShipperProfileResponse, error) {
	id, err := parseID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	sp, err := c.engine.GetShipperProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.GetShipperProfileResponse{ShipperProfile: c.mapper.EntityShipperProfileToPb(*sp)}, nil
}

func (c *userController) UpdateShipperProfile(ctx context.Context, req *pb.UpdateShipperProfileRequest) (*pb.UpdateShipperProfileResponse, error) {
	param, err := c.mapper.PbUpdateShipperProfileToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	sp, err := c.engine.UpdateShipperProfile(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateShipperProfileResponse{
		ShipperProfile: c.mapper.EntityShipperProfileToPb(*sp),
		Message:        "Cập nhật hồ sơ chủ hàng thành công",
	}, nil
}

func (c *userController) CreateAddress(ctx context.Context, req *pb.CreateAddressRequest) (*pb.CreateAddressResponse, error) {
	param, err := c.mapper.PbCreateAddressToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	addr, err := c.engine.CreateAddress(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.CreateAddressResponse{
		Address: c.mapper.EntityAddressToPb(*addr),
		Message: "Thêm địa chỉ thành công",
	}, nil
}

func (c *userController) ListAddresses(ctx context.Context, req *pb.ListAddressesRequest) (*pb.ListAddressesResponse, error) {
	param, err := c.mapper.PbListAddressesToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	res, err := c.engine.ListAddresses(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.ListAddressesResponse{
		Addresses:  c.mapper.EntityAddressListToPbList(res.Addresses),
		Pagination: c.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (c *userController) UpdateAddress(ctx context.Context, req *pb.UpdateAddressRequest) (*pb.UpdateAddressResponse, error) {
	param, err := c.mapper.PbUpdateAddressToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidAddressID.WithCause(err)
	}

	addr, err := c.engine.UpdateAddress(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateAddressResponse{
		Address: c.mapper.EntityAddressToPb(*addr),
		Message: "Cập nhật địa chỉ thành công",
	}, nil
}

func (c *userController) DeleteAddress(ctx context.Context, req *pb.DeleteAddressRequest) (*pb.DeleteAddressResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidAddressID)
	if err != nil {
		return nil, err
	}

	var userID uuid.UUID
	if len(req.UserId) > 0 {
		userID, err = parseID(req.UserId, entity.ErrInvalidUserID)
		if err != nil {
			return nil, err
		}
	}

	if err := c.engine.DeleteAddress(ctx, id, userID); err != nil {
		return nil, err
	}
	return &pb.DeleteAddressResponse{Message: "Xoá địa chỉ thành công"}, nil
}

func (c *userController) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.RegisterDeviceResponse, error) {
	param, err := c.mapper.PbRegisterDeviceToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	device, err := c.engine.RegisterDevice(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterDeviceResponse{
		Device:  c.mapper.EntityUserDeviceToPb(*device),
		Message: "Đăng ký thiết bị thành công",
	}, nil
}

func (c *userController) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	id, err := parseID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	devices, err := c.engine.ListDevices(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.ListDevicesResponse{Devices: c.mapper.EntityUserDeviceListToPbList(devices)}, nil
}

func (c *userController) DeleteDevice(ctx context.Context, req *pb.DeleteDeviceRequest) (*pb.DeleteDeviceResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidDeviceID)
	if err != nil {
		return nil, err
	}

	var userID uuid.UUID
	if len(req.UserId) > 0 {
		userID, err = parseID(req.UserId, entity.ErrInvalidUserID)
		if err != nil {
			return nil, err
		}
	}

	if err := c.engine.DeleteDevice(ctx, id, userID); err != nil {
		return nil, err
	}
	return &pb.DeleteDeviceResponse{Message: "Xoá thiết bị thành công"}, nil
}

func (c *userController) AdminListUsers(ctx context.Context, req *pb.AdminListUsersRequest) (*pb.AdminListUsersResponse, error) {
	filter := c.mapper.PbAdminListUsersToFilter(req)

	res, err := c.engine.AdminListUsers(ctx, &filter)
	if err != nil {
		return nil, err
	}

	return &pb.AdminListUsersResponse{
		Users:      c.mapper.EntityUserListToPbUserList(res.Users),
		Pagination: c.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (c *userController) AdminUpdateUserStatus(ctx context.Context, req *pb.AdminUpdateUserStatusRequest) (*pb.AdminUpdateUserStatusResponse, error) {
	param, err := c.mapper.PbAdminUpdateStatusToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	u, err := c.engine.AdminUpdateUserStatus(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminUpdateUserStatusResponse{
		User:    c.mapper.EntityUserToPbUser(*u),
		Message: "Cập nhật trạng thái tài khoản thành công",
	}, nil
}

func (c *userController) AdminGetUserStats(ctx context.Context, _ *pb.AdminGetUserStatsRequest) (*pb.AdminGetUserStatsResponse, error) {
	stats, err := c.engine.AdminGetUserStats(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.AdminGetUserStatsResponse{
		TotalUsers:    stats.TotalUsers,
		TotalDrivers:  stats.TotalDrivers,
		TotalShippers: stats.TotalShippers,
		ActiveUsers:   stats.ActiveUsers,
		BannedUsers:   stats.BannedUsers,
		PendingKyc:    stats.PendingKyc,
	}, nil
}

func (c *userController) AdminDeleteUser(ctx context.Context, req *pb.AdminDeleteUserRequest) (*pb.AdminDeleteUserResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}
	if err := c.engine.AdminDeleteUser(ctx, id); err != nil {
		return nil, err
	}
	return &pb.AdminDeleteUserResponse{Message: "Xoá người dùng thành công"}, nil
}
