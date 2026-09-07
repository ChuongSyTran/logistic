package httpserver

import (
	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	pbkyc "github.com/logistic/api/logistic/kyc_service/v1"
	"github.com/logistic/pkg/uuidx"
)

type KycController struct {
	kycClient pbkyc.KycServiceClient
}

func NewKycController(kycClient pbkyc.KycServiceClient) *KycController {
	return &KycController{kycClient: kycClient}
}

type SubmitKYCReq struct {
	IdCardNumber    string `json:"id_card_number"`
	LicenseNumber   string `json:"license_number"`
	IdCardFrontURL  string `json:"id_card_front_url"`
	IdCardBackURL   string `json:"id_card_back_url"`
	LicenseFrontURL string `json:"license_front_url"`
	LicenseBackURL  string `json:"license_back_url"`
	Status          string `json:"status" binding:"omitempty,oneof=pending approved rejected"`
	Note            string `json:"note"`
}

type ReviewKYCReq struct {
	Approved bool   `json:"approved"`
	Note     string `json:"note"`
}

// SubmitKYC godoc
// @Summary      Nộp/cập nhật hồ sơ KYC
// @Description  Tài xế nộp/cập nhật hồ sơ KYC của chính mình.
// @Tags         KYC
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        request body SubmitKYCReq true "Thông tin hồ sơ KYC"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/kyc [put]
func (c *KycController) SubmitKYC(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req SubmitKYCReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.kycClient.SubmitKYC(ctx.Request.Context(), &pbkyc.SubmitKYCRequest{
		UserId:          userID,
		IdCardNumber:    req.IdCardNumber,
		LicenseNumber:   req.LicenseNumber,
		IdCardFrontUrl:  req.IdCardFrontURL,
		IdCardBackUrl:   req.IdCardBackURL,
		LicenseFrontUrl: req.LicenseFrontURL,
		LicenseBackUrl:  req.LicenseBackURL,
		Status:          req.Status,
		Note:            req.Note,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OKMessage(ctx, gin.H{"kyc": toKycDTO(resp.Kyc)}, "Nộp hồ sơ KYC thành công")
}

// GetKYC godoc
// @Summary      Xem hồ sơ KYC
// @Description  Tài xế xem hồ sơ KYC của chính mình hoặc admin xem hồ sơ KYC của user.
// @Tags         KYC
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/kyc [get]
func (c *KycController) GetKYC(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.kycClient.GetKYC(ctx.Request.Context(), &pbkyc.GetKYCRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OK(ctx, gin.H{"kyc": toKycDTO(resp.Kyc)})
}

// ListPendingKYC godoc
// @Summary      [Admin] Hàng đợi duyệt KYC
// @Description  Danh sách các hồ sơ KYC đang chờ duyệt
// @Tags         Admin-KYC
// @Produce      json
// @Param        page query int false "Số trang"
// @Param        page_size query int false "Kích thước trang"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/kyc/pending [get]
func (c *KycController) ListPendingKYC(ctx *gin.Context) {
	resp, err := c.kycClient.ListPendingKYC(ctx.Request.Context(), &pbkyc.ListPendingKYCRequest{
		Page:     int32(queryInt(ctx, "page")),
		PageSize: int32(queryInt(ctx, "page_size")),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	items := make([]gin.H, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = toKycDTO(item)
	}

	response.OK(ctx, gin.H{
		"items": items,
		"pagination": gin.H{
			"page":        resp.Pagination.GetPage(),
			"page_size":   resp.Pagination.GetPageSize(),
			"total_items": resp.Pagination.GetTotalItems(),
			"total_pages": resp.Pagination.GetTotalPages(),
		},
	})
}

// ReviewKYC godoc
// @Summary      [Admin] Duyệt/từ chối KYC
// @Description  Admin phê duyệt hoặc từ chối hồ sơ KYC
// @Tags         Admin-KYC
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Param        request body ReviewKYCReq true "Thông tin duyệt KYC"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/kyc/{user_id}/review [put]
func (c *KycController) ReviewKYC(ctx *gin.Context) {
	userID, ok := pathID(ctx, "user_id")
	if !ok {
		return
	}

	var req ReviewKYCReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.kycClient.ReviewKYC(ctx.Request.Context(), &pbkyc.ReviewKYCRequest{
		UserId:     userID,
		Approved:   req.Approved,
		Note:       req.Note,
		ReviewerId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	msg := "Đã từ chối hồ sơ KYC"
	if req.Approved {
		msg = "Đã duyệt hồ sơ KYC"
	}
	response.OKMessage(ctx, gin.H{"kyc": toKycDTO(resp.Kyc)}, msg)
}

// CountPendingKYC godoc
// @Summary      [Admin] Đếm số lượng hồ sơ KYC đang chờ duyệt
// @Tags         Admin-KYC
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/kyc/count-pending [get]
func (c *KycController) CountPendingKYC(ctx *gin.Context) {
	resp, err := c.kycClient.CountPendingKYC(ctx.Request.Context(), &pbkyc.CountPendingKYCRequest{})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OK(ctx, gin.H{"count": resp.Count})
}

func toKycDTO(k *pbkyc.KycDocument) gin.H {
	if k == nil {
		return nil
	}
	uID, _ := uuidx.FromBytes(k.UserId)
	reviewerID, _ := uuidx.FromBytes(k.ReviewerId)

	dto := gin.H{
		"id":                k.Id,
		"user_id":           uID.String(),
		"id_card_number":    k.IdCardNumber,
		"license_number":    k.LicenseNumber,
		"id_card_front_url": k.IdCardFrontUrl,
		"id_card_back_url":  k.IdCardBackUrl,
		"license_front_url": k.LicenseFrontUrl,
		"license_back_url":  k.LicenseBackUrl,
		"status":            k.Status,
		"note":              k.Note,
		"created_at":        k.CreatedAt,
		"updated_at":        k.UpdatedAt,
	}
	if len(k.ReviewerId) > 0 {
		dto["reviewer_id"] = reviewerID.String()
	}
	if k.ReviewedAt != "" {
		dto["reviewed_at"] = k.ReviewedAt
	}
	return dto
}
