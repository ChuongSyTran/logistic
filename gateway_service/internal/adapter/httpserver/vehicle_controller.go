package httpserver

import (
	"context"
	"time"

	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	pbuser "github.com/logistic/api/logistic/user_service/v1"
	pb "github.com/logistic/api/logistic/vehicle_service/v1"
	pbkyc "github.com/logistic/api/logistic/kyc_service/v1"
)

type VehicleController struct {
	userClient    pbuser.UserServiceClient
	vehicleClient pb.VehicleServiceClient
	kycClient     pbkyc.KycServiceClient
}

func NewVehicleController(
	vehicleClient pb.VehicleServiceClient,
	userClient pbuser.UserServiceClient,
	kycClient pbkyc.KycServiceClient,
) *VehicleController {
	return &VehicleController{vehicleClient: vehicleClient, userClient: userClient, kycClient: kycClient}
}

const kycApproved = "approved"

// kycApprovedFor chặn tài xế chưa qua KYC lên online. Hồ sơ KYC nằm ở
// kyc_service còn trạng thái online ở vehicle_service, nên gateway là chỗ duy
// nhất nhìn được cả hai. Không chặn thì cửa duyệt KYC không có tác dụng gì.
func (c *VehicleController) kycApprovedFor(ctx *gin.Context, driverID []byte) bool {
	if c.kycClient == nil {
		return true
	}

	resp, err := c.kycClient.GetKYC(ctx.Request.Context(), &pbkyc.GetKYCRequest{
		UserId: driverID,
	})
	if err != nil {
		response.Error(ctx, err)
		return false
	}
	if resp.GetKyc().GetStatus() != kycApproved {
		response.FailedPrecondition(ctx, "KYC_NOT_APPROVED",
			"hồ sơ KYC chưa được duyệt, chưa thể nhận đơn")
		return false
	}
	return true
}

type RegisterVehicleReq struct {
	LicensePlate      string  `json:"license_plate" binding:"required"`
	Brand             string  `json:"brand"`
	Model             string  `json:"model"`
	ManufactureYear   int32   `json:"manufacture_year"`
	VehicleType       string  `json:"vehicle_type" binding:"required,oneof=truck van bike container trailer"`
	CapacityWeightKg  float64 `json:"capacity_weight_kg" binding:"required,gt=0"`
	CapacityVolumeCbm float64 `json:"capacity_volume_cbm" binding:"required,gt=0"`
}

// RegisterVehicle godoc
// @Summary      Đăng ký phương tiện
// @Description  Xe luôn được đăng ký cho CHÍNH tài xế đang đăng nhập.
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Success      201 {object} response.Envelope
// @Router       /api/v1/vehicles [post]
func (c *VehicleController) RegisterVehicle(ctx *gin.Context) {
	var req RegisterVehicleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.vehicleClient.RegisterVehicle(ctx.Request.Context(), &pb.RegisterVehicleRequest{
		DriverId:          selfID(ctx),
		LicensePlate:      req.LicensePlate,
		Brand:             req.Brand,
		Model:             req.Model,
		ManufactureYear:   req.ManufactureYear,
		VehicleType:       req.VehicleType,
		CapacityWeightKg:  req.CapacityWeightKg,
		CapacityVolumeCbm: req.CapacityVolumeCbm,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.Created(ctx, gin.H{
		"id":      uuidString(resp.Id),
		"vehicle": toVehicleDTO(resp.Vehicle),
	}, resp.Message)
}

// GetVehicle godoc
// @Summary      Chi tiết phương tiện
// @Tags         Vehicle
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id} [get]
func (c *VehicleController) GetVehicle(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.vehicleClient.GetVehicle(ctx.Request.Context(), &pb.GetVehicleRequest{
		Id:       id,
		DriverId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"vehicle": toVehicleDTO(resp.Vehicle)})
}

// ListVehicles godoc
// @Summary      Danh sách phương tiện của tài xế
// @Tags         Vehicle
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles [get]
func (c *VehicleController) ListVehicles(ctx *gin.Context) {
	driverID, ok := queryID(ctx, "driver_id")
	if !ok {
		return
	}
	if driverID == nil {
		driverID = selfID(ctx)
	}
	if !requireSelfOrAdmin(ctx, driverID) {
		return
	}

	resp, err := c.vehicleClient.ListVehicles(ctx.Request.Context(), &pb.ListVehiclesRequest{
		DriverId:    driverID,
		Status:      ctx.Query("status"),
		VehicleType: ctx.Query("vehicle_type"),
		Page:        queryInt(ctx, "page"),
		PageSize:    queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"vehicles":   toVehicleDTOs(resp.Vehicles),
		"pagination": toVehiclePaginationDTO(resp.Pagination),
	})
}

type UpdateVehicleReq struct {
	Brand             string  `json:"brand"`
	Model             string  `json:"model"`
	ManufactureYear   int32   `json:"manufacture_year"`
	VehicleType       string  `json:"vehicle_type" binding:"omitempty,oneof=truck van bike container trailer"`
	CapacityWeightKg  float64 `json:"capacity_weight_kg"`
	CapacityVolumeCbm float64 `json:"capacity_volume_cbm"`
}

// UpdateVehicle godoc
// @Summary      Cập nhật phương tiện
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id} [put]
func (c *VehicleController) UpdateVehicle(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req UpdateVehicleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.vehicleClient.UpdateVehicle(ctx.Request.Context(), &pb.UpdateVehicleRequest{
		Id:                id,
		DriverId:          selfID(ctx),
		Brand:             req.Brand,
		Model:             req.Model,
		ManufactureYear:   req.ManufactureYear,
		VehicleType:       req.VehicleType,
		CapacityWeightKg:  req.CapacityWeightKg,
		CapacityVolumeCbm: req.CapacityVolumeCbm,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"vehicle": toVehicleDTO(resp.Vehicle)}, resp.Message)
}

// DeleteVehicle godoc
// @Summary      Xoá phương tiện
// @Tags         Vehicle
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id} [delete]
func (c *VehicleController) DeleteVehicle(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.vehicleClient.DeleteVehicle(ctx.Request.Context(), &pb.DeleteVehicleRequest{
		Id:       id,
		DriverId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}

type UpdateVehicleStatusReq struct {
	Status string `json:"status" binding:"required,oneof=active maintenance inactive"`
}

// UpdateVehicleStatus godoc
// @Summary      Đổi trạng thái phương tiện
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id}/status [put]
func (c *VehicleController) UpdateVehicleStatus(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req UpdateVehicleStatusReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.vehicleClient.UpdateVehicleStatus(ctx.Request.Context(), &pb.UpdateVehicleStatusRequest{
		Id:       id,
		DriverId: selfID(ctx),
		Status:   req.Status,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"vehicle": toVehicleDTO(resp.Vehicle)}, resp.Message)
}

type UploadDocumentReq struct {
	DocumentType   string `json:"document_type" binding:"required,oneof=registration inspection insurance license"`
	DocumentNumber string `json:"document_number"`
	FileURL        string `json:"file_url" binding:"required,url"`
}

// UploadVehicleDocument godoc
// @Summary      Tải lên giấy tờ xe
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      201 {object} response.Envelope
// @Router       /api/v1/vehicles/{id}/documents [post]
func (c *VehicleController) UploadVehicleDocument(ctx *gin.Context) {
	vehicleID, ok := pathID(ctx, "vehicle_id", "id")
	if !ok {
		return
	}

	var req UploadDocumentReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.vehicleClient.UploadVehicleDocument(ctx.Request.Context(), &pb.UploadVehicleDocumentRequest{
		VehicleId:      vehicleID,
		DriverId:       selfID(ctx),
		DocumentType:   req.DocumentType,
		DocumentNumber: req.DocumentNumber,
		FileUrl:        req.FileURL,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.Created(ctx, gin.H{"document": toVehicleDocumentDTO(resp.Document)}, resp.Message)
}

// ListVehicleDocuments godoc
// @Summary      Danh sách giấy tờ xe
// @Tags         Vehicle
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id}/documents [get]
func (c *VehicleController) ListVehicleDocuments(ctx *gin.Context) {
	vehicleID, ok := pathID(ctx, "vehicle_id", "id")
	if !ok {
		return
	}

	resp, err := c.vehicleClient.ListVehicleDocuments(ctx.Request.Context(), &pb.ListVehicleDocumentsRequest{
		VehicleId:    vehicleID,
		DriverId:     selfID(ctx),
		ReviewStatus: ctx.Query("review_status"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"documents": toVehicleDocumentDTOs(resp.Documents)})
}

// DeleteVehicleDocument godoc
// @Summary      Xoá giấy tờ xe
// @Tags         Vehicle
// @Produce      json
// @Param        id path string true "Document ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicle-documents/{id} [delete]
func (c *VehicleController) DeleteVehicleDocument(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.vehicleClient.DeleteVehicleDocument(ctx.Request.Context(), &pb.DeleteVehicleDocumentRequest{
		Id:       id,
		DriverId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}

type ReportLocationReq struct {
	Latitude  float64 `json:"latitude" binding:"required,latitude"`
	Longitude float64 `json:"longitude" binding:"required,longitude"`
	Heading   float64 `json:"heading"`
	SpeedKph  float64 `json:"speed_kph"`
}

const locationTimeout = 2 * time.Second

// ReportLocation godoc
// @Summary      Tài xế báo vị trí GPS
// @Description  App tài xế gọi định kỳ. Vị trí được ghi xuống DB và cập nhật vào chỉ mục Redis GEO.
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id}/location [post]
func (c *VehicleController) ReportLocation(ctx *gin.Context) {
	vehicleID, ok := pathID(ctx, "vehicle_id", "id")
	if !ok {
		return
	}

	var req ReportLocationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	callCtx, cancel := context.WithTimeout(ctx.Request.Context(), locationTimeout)
	defer cancel()

	resp, err := c.vehicleClient.ReportLocation(callCtx, &pb.ReportLocationRequest{
		VehicleId: vehicleID,
		DriverId:  selfID(ctx),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Heading:   req.Heading,
		SpeedKph:  req.SpeedKph,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"zone_id": resp.ZoneId}, resp.Message)
}

// GetVehicleLocation godoc
// @Summary      Vị trí hiện tại của xe
// @Tags         Vehicle
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/{id}/location [get]
func (c *VehicleController) GetVehicleLocation(ctx *gin.Context) {
	vehicleID, ok := pathID(ctx, "vehicle_id", "id")
	if !ok {
		return
	}

	callCtx, cancel := context.WithTimeout(ctx.Request.Context(), locationTimeout)
	defer cancel()

	resp, err := c.vehicleClient.GetVehicleLocation(callCtx, &pb.GetVehicleLocationRequest{
		VehicleId: vehicleID,
		DriverId:  selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"location": toVehicleLocationDTO(resp.Location)})
}

type SetAvailabilityReq struct {
	VehicleID          string  `json:"vehicle_id" binding:"required"`
	IsOnline           bool    `json:"is_online"`
	AvailableWeightKg  float64 `json:"available_weight_kg"`
	AvailableVolumeCbm float64 `json:"available_volume_cbm"`
	CurrentLat         float64 `json:"current_lat"`
	CurrentLng         float64 `json:"current_lng"`
}

// SetDriverAvailability godoc
// @Summary      Bật/tắt nhận đơn
// @Description  Bật thì xe được đưa vào chỉ mục tìm kiếm của matching; tắt thì gỡ ra.
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/drivers/{driver_id}/availability [post]
func (c *VehicleController) SetDriverAvailability(ctx *gin.Context) {
	driverID, ok := resolveOwnID(ctx, "driver_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, driverID) {
		return
	}

	var req SetAvailabilityReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	vehicleID, ok := bodyID(ctx, "vehicle_id", req.VehicleID, true)
	if !ok {
		return
	}

	if req.IsOnline && !c.kycApprovedFor(ctx, driverID) {
		return
	}

	resp, err := c.vehicleClient.SetDriverAvailability(ctx.Request.Context(), &pb.SetDriverAvailabilityRequest{
		DriverId:           driverID,
		VehicleId:          vehicleID,
		IsOnline:           req.IsOnline,
		AvailableWeightKg:  req.AvailableWeightKg,
		AvailableVolumeCbm: req.AvailableVolumeCbm,
		CurrentLat:         req.CurrentLat,
		CurrentLng:         req.CurrentLng,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"availability": toDriverAvailabilityDTO(resp.Availability)}, resp.Message)
}

// GetDriverAvailability godoc
// @Summary      Trạng thái nhận đơn của tài xế
// @Tags         Vehicle
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/drivers/{driver_id}/availability [get]
func (c *VehicleController) GetDriverAvailability(ctx *gin.Context) {
	driverID, ok := resolveOwnID(ctx, "driver_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, driverID) {
		return
	}

	resp, err := c.vehicleClient.GetDriverAvailability(ctx.Request.Context(), &pb.GetDriverAvailabilityRequest{
		DriverId: driverID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"availability": toDriverAvailabilityDTO(resp.Availability)})
}

type SearchNearbyReq struct {
	Latitude     float64 `json:"latitude" binding:"required,latitude"`
	Longitude    float64 `json:"longitude" binding:"required,longitude"`
	RadiusKm     float64 `json:"radius_km"`
	MinWeightKg  float64 `json:"min_weight_kg"`
	MinVolumeCbm float64 `json:"min_volume_cbm"`
	VehicleType  string  `json:"vehicle_type" binding:"omitempty,oneof=truck van bike container trailer"`
	Limit        int32   `json:"limit"`
}

// SearchNearbyVehicles godoc
// @Summary      Tìm xe đang chạy quanh một điểm
// @Description  Chạy trên chỉ mục Redis GEO nên trả về trong vài mili-giây. Đây cũng là API matching_service dùng nội bộ.
// @Tags         Vehicle
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/vehicles/nearby [post]
func (c *VehicleController) SearchNearbyVehicles(ctx *gin.Context) {
	var req SearchNearbyReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	callCtx, cancel := context.WithTimeout(ctx.Request.Context(), locationTimeout)
	defer cancel()

	resp, err := c.vehicleClient.SearchNearbyVehicles(callCtx, &pb.SearchNearbyVehiclesRequest{
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		RadiusKm:     req.RadiusKm,
		MinWeightKg:  req.MinWeightKg,
		MinVolumeCbm: req.MinVolumeCbm,
		VehicleType:  req.VehicleType,
		Limit:        req.Limit,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"vehicles":    toNearbyVehicleDTOs(resp.Vehicles),
		"total_found": resp.TotalFound,
	})
}

// AdminListVehicles godoc
// @Summary      [Admin] Danh sách toàn bộ phương tiện
// @Tags         Admin-Vehicle
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/vehicles [get]
func (c *VehicleController) AdminListVehicles(ctx *gin.Context) {
	resp, err := c.vehicleClient.AdminListVehicles(ctx.Request.Context(), &pb.AdminListVehiclesRequest{
		Status:             ctx.Query("status"),
		VerificationStatus: ctx.Query("verification_status"),
		VehicleType:        ctx.Query("vehicle_type"),
		Keyword:            ctx.Query("keyword"),
		Page:               queryInt(ctx, "page"),
		PageSize:           queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"vehicles":   toVehicleDTOs(resp.Vehicles),
		"pagination": toVehiclePaginationDTO(resp.Pagination),
	})
}

type AdminReviewReq struct {
	Approved bool   `json:"approved"`
	Note     string `json:"note"`
}

// AdminVerifyVehicle godoc
// @Summary      [Admin] Duyệt phương tiện
// @Tags         Admin-Vehicle
// @Accept       json
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/vehicles/{id}/verify [put]
func (c *VehicleController) AdminVerifyVehicle(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req AdminReviewReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.vehicleClient.AdminVerifyVehicle(ctx.Request.Context(), &pb.AdminVerifyVehicleRequest{
		Id:         id,
		Approved:   req.Approved,
		Note:       req.Note,
		ReviewerId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"vehicle": toVehicleDTO(resp.Vehicle)}, resp.Message)
}

// AdminListPendingDocuments godoc
// @Summary      [Admin] Hàng đợi duyệt giấy tờ
// @Tags         Admin-Vehicle
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/vehicle-documents/pending [get]
func (c *VehicleController) AdminListPendingDocuments(ctx *gin.Context) {
	resp, err := c.vehicleClient.AdminListPendingDocuments(ctx.Request.Context(), &pb.AdminListPendingDocumentsRequest{
		Page:     queryInt(ctx, "page"),
		PageSize: queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"documents":  toVehicleDocumentDTOs(resp.Documents),
		"pagination": toVehiclePaginationDTO(resp.Pagination),
	})
}

// AdminReviewDocument godoc
// @Summary      [Admin] Duyệt giấy tờ xe
// @Tags         Admin-Vehicle
// @Accept       json
// @Produce      json
// @Param        id path string true "Document ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/vehicle-documents/{id}/review [put]
func (c *VehicleController) AdminReviewDocument(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req AdminReviewReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.vehicleClient.AdminReviewDocument(ctx.Request.Context(), &pb.AdminReviewDocumentRequest{
		Id:         id,
		Approved:   req.Approved,
		Note:       req.Note,
		ReviewerId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"document": toVehicleDocumentDTO(resp.Document)}, resp.Message)
}

// AdminGetVehicleStats godoc
// @Summary      [Admin] Thống kê phương tiện
// @Tags         Admin-Vehicle
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/vehicles/stats [get]
func (c *VehicleController) AdminGetVehicleStats(ctx *gin.Context) {
	resp, err := c.vehicleClient.AdminGetVehicleStats(ctx.Request.Context(), &pb.AdminGetVehicleStatsRequest{})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"total_vehicles":       resp.TotalVehicles,
		"active_vehicles":      resp.ActiveVehicles,
		"maintenance_vehicles": resp.MaintenanceVehicles,
		"pending_verification": resp.PendingVerification,
		"online_drivers":       resp.OnlineDrivers,
		"pending_documents":    resp.PendingDocuments,
	})
}
