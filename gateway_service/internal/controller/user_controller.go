package controller

import (
	"strconv"

	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	pb "github.com/logistic/api/logistic/user_service/v1"
)

type UserController struct {
	userClient pb.UserServiceClient
}

func NewUserController(userClient pb.UserServiceClient) *UserController {
	return &UserController{userClient: userClient}
}

func queryInt(ctx *gin.Context, key string) int32 {
	raw := ctx.Query(key)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return int32(n)
}

func queryBool(ctx *gin.Context, key string) bool {
	v, _ := strconv.ParseBool(ctx.Query(key))
	return v
}

type RegisterUserReq struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=driver shipper"`
	FullName string `json:"full_name"`
}

// RegisterUser godoc
// @Summary      Đăng ký người dùng
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        request body RegisterUserReq true "Thông tin đăng ký"
// @Success      201 {object} response.Envelope
// @Router       /api/v1/users/register [post]
func (c *UserController) RegisterUser(ctx *gin.Context) {
	var req RegisterUserReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.RegisterUser(ctx.Request.Context(), &pb.RegisterUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
		Role:     req.Role,
		FullName: req.FullName,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.Created(ctx, gin.H{
		"id":   uuidString(resp.Id),
		"user": toUserDTO(resp.User),
	}, resp.Message)
}

// GetUser godoc
// @Summary      Lấy thông tin người dùng
// @Tags         User
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id} [get]
func (c *UserController) GetUser(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id", "id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.userClient.GetUser(ctx.Request.Context(), &pb.GetUserRequest{Id: userID})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OK(ctx, gin.H{
		"user":            toUserDTO(resp.User),
		"driver_profile":  toDriverProfileDTO(resp.DriverProfile),
		"shipper_profile": toShipperProfileDTO(resp.ShipperProfile),
	})
}

type UpdateUserReq struct {
	FullName  string `json:"full_name"`
	Email     string `json:"email" binding:"omitempty,email"`
	AvatarURL string `json:"avatar_url"`
}

// UpdateUser godoc
// @Summary      Cập nhật thông tin người dùng
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id} [put]
func (c *UserController) UpdateUser(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id", "id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req UpdateUserReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.UpdateUser(ctx.Request.Context(), &pb.UpdateUserRequest{
		Id:        userID,
		FullName:  req.FullName,
		Email:     req.Email,
		AvatarUrl: req.AvatarURL,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OKMessage(ctx, gin.H{"user": toUserDTO(resp.User)}, resp.Message)
}

// GetDriverProfile godoc
// @Summary      Lấy hồ sơ tài xế
// @Tags         User
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/driver-profile [get]
func (c *UserController) GetDriverProfile(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.userClient.GetDriverProfile(ctx.Request.Context(), &pb.GetDriverProfileRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"driver_profile": toDriverProfileDTO(resp.DriverProfile)})
}

type UpdateDriverProfileReq struct {
	LicenseNumber string `json:"license_number"`
	IDCard        string `json:"id_card"`
}

// UpdateDriverProfile godoc
// @Summary      Cập nhật hồ sơ tài xế
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/driver-profile [put]
func (c *UserController) UpdateDriverProfile(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req UpdateDriverProfileReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.UpdateDriverProfile(ctx.Request.Context(), &pb.UpdateDriverProfileRequest{
		UserId:        userID,
		LicenseNumber: req.LicenseNumber,
		IdCard:        req.IDCard,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"driver_profile": toDriverProfileDTO(resp.DriverProfile)}, resp.Message)
}

// GetShipperProfile godoc
// @Summary      Lấy hồ sơ chủ hàng
// @Tags         User
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/shipper-profile [get]
func (c *UserController) GetShipperProfile(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.userClient.GetShipperProfile(ctx.Request.Context(), &pb.GetShipperProfileRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"shipper_profile": toShipperProfileDTO(resp.ShipperProfile)})
}

type UpdateShipperProfileReq struct {
	CompanyName     string `json:"company_name"`
	TaxCode         string `json:"tax_code"`
	BusinessAddress string `json:"business_address"`
}

// UpdateShipperProfile godoc
// @Summary      Cập nhật hồ sơ chủ hàng
// @Tags         User
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/shipper-profile [put]
func (c *UserController) UpdateShipperProfile(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req UpdateShipperProfileReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.UpdateShipperProfile(ctx.Request.Context(), &pb.UpdateShipperProfileRequest{
		UserId:          userID,
		CompanyName:     req.CompanyName,
		TaxCode:         req.TaxCode,
		BusinessAddress: req.BusinessAddress,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"shipper_profile": toShipperProfileDTO(resp.ShipperProfile)}, resp.Message)
}

type AddressReq struct {
	Label        string  `json:"label"`
	ContactName  string  `json:"contact_name"`
	ContactPhone string  `json:"contact_phone"`
	Line1        string  `json:"line1" binding:"required"`
	Ward         string  `json:"ward"`
	District     string  `json:"district"`
	City         string  `json:"city"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	AddressType  string  `json:"address_type" binding:"omitempty,oneof=pickup delivery both"`
	IsDefault    bool    `json:"is_default"`
}

// CreateAddress godoc
// @Summary      Thêm địa chỉ vào sổ địa chỉ
// @Tags         Address
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      201 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/addresses [post]
func (c *UserController) CreateAddress(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req AddressReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.CreateAddress(ctx.Request.Context(), &pb.CreateAddressRequest{
		UserId:       userID,
		Label:        req.Label,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Line1:        req.Line1,
		Ward:         req.Ward,
		District:     req.District,
		City:         req.City,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		AddressType:  req.AddressType,
		IsDefault:    req.IsDefault,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.Created(ctx, gin.H{"address": toAddressDTO(resp.Address)}, resp.Message)
}

// ListAddresses godoc
// @Summary      Danh sách địa chỉ
// @Tags         Address
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/addresses [get]
func (c *UserController) ListAddresses(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.userClient.ListAddresses(ctx.Request.Context(), &pb.ListAddressesRequest{
		UserId:      userID,
		AddressType: ctx.Query("address_type"),
		Page:        queryInt(ctx, "page"),
		PageSize:    queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"addresses":  toAddressDTOs(resp.Addresses),
		"pagination": toUserPaginationDTO(resp.Pagination),
	})
}

// UpdateAddress godoc
// @Summary      Cập nhật địa chỉ
// @Tags         Address
// @Accept       json
// @Produce      json
// @Param        id path string true "Address ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/addresses/{id} [put]
func (c *UserController) UpdateAddress(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req AddressReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.UpdateAddress(ctx.Request.Context(), &pb.UpdateAddressRequest{
		Id:           id,
		UserId:       selfID(ctx),
		Label:        req.Label,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Line1:        req.Line1,
		Ward:         req.Ward,
		District:     req.District,
		City:         req.City,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		AddressType:  req.AddressType,
		IsDefault:    req.IsDefault,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"address": toAddressDTO(resp.Address)}, resp.Message)
}

// DeleteAddress godoc
// @Summary      Xoá địa chỉ
// @Tags         Address
// @Produce      json
// @Param        id path string true "Address ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/addresses/{id} [delete]
func (c *UserController) DeleteAddress(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.userClient.DeleteAddress(ctx.Request.Context(), &pb.DeleteAddressRequest{
		Id:     id,
		UserId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}

type RegisterDeviceReq struct {
	DeviceToken string `json:"device_token" binding:"required"`
	Platform    string `json:"platform" binding:"omitempty,oneof=android ios web"`
	DeviceName  string `json:"device_name"`
}

// RegisterDevice godoc
// @Summary      Đăng ký thiết bị nhận push
// @Tags         Device
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      201 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/devices [post]
func (c *UserController) RegisterDevice(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}

	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req RegisterDeviceReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.RegisterDevice(ctx.Request.Context(), &pb.RegisterDeviceRequest{
		UserId:      userID,
		DeviceToken: req.DeviceToken,
		Platform:    req.Platform,
		DeviceName:  req.DeviceName,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.Created(ctx, gin.H{"device": toUserDeviceDTO(resp.Device)}, resp.Message)
}

// ListDevices godoc
// @Summary      Danh sách thiết bị
// @Tags         Device
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/devices [get]
func (c *UserController) ListDevices(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.userClient.ListDevices(ctx.Request.Context(), &pb.ListDevicesRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"devices": toUserDeviceDTOs(resp.Devices)})
}

// DeleteDevice godoc
// @Summary      Xoá thiết bị
// @Tags         Device
// @Produce      json
// @Param        id path string true "Device ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/devices/{id} [delete]
func (c *UserController) DeleteDevice(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.userClient.DeleteDevice(ctx.Request.Context(), &pb.DeleteDeviceRequest{
		Id:     id,
		UserId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}

// AdminListUsers godoc
// @Summary      [Admin] Danh sách người dùng
// @Tags         Admin-User
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/users [get]
func (c *UserController) AdminListUsers(ctx *gin.Context) {
	resp, err := c.userClient.AdminListUsers(ctx.Request.Context(), &pb.AdminListUsersRequest{
		Role:     ctx.Query("role"),
		Status:   ctx.Query("status"),
		Keyword:  ctx.Query("keyword"),
		Page:     queryInt(ctx, "page"),
		PageSize: queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"users":      toUserDTOs(resp.Users),
		"pagination": toUserPaginationDTO(resp.Pagination),
	})
}

type AdminUpdateUserStatusReq struct {
	Status string `json:"status" binding:"required,oneof=active banned suspended"`
	Reason string `json:"reason"`
}

// AdminUpdateUserStatus godoc
// @Summary      [Admin] Khoá/mở tài khoản
// @Tags         Admin-User
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/users/{id}/status [put]
func (c *UserController) AdminUpdateUserStatus(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req AdminUpdateUserStatusReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.userClient.AdminUpdateUserStatus(ctx.Request.Context(), &pb.AdminUpdateUserStatusRequest{
		Id:     id,
		Status: req.Status,
		Reason: req.Reason,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"user": toUserDTO(resp.User)}, resp.Message)
}

// AdminGetUserStats godoc
// @Summary      [Admin] Thống kê người dùng
// @Tags         Admin-User
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/users/stats [get]
func (c *UserController) AdminGetUserStats(ctx *gin.Context) {
	resp, err := c.userClient.AdminGetUserStats(ctx.Request.Context(), &pb.AdminGetUserStatsRequest{})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"total_users":    resp.TotalUsers,
		"total_drivers":  resp.TotalDrivers,
		"total_shippers": resp.TotalShippers,
		"active_users":   resp.ActiveUsers,
		"banned_users":   resp.BannedUsers,
		"pending_kyc":    resp.PendingKyc,
	})
}

// AdminDeleteUser godoc
// @Summary      [Admin] Xoá người dùng
// @Tags         Admin-User
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/users/{id} [delete]
func (c *UserController) AdminDeleteUser(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.userClient.AdminDeleteUser(ctx.Request.Context(), &pb.AdminDeleteUserRequest{Id: id})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}