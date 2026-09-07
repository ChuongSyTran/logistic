package controller

import (
	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	pb "github.com/logistic/api/logistic/notification_service/v1"
	"github.com/logistic/pkg/uuidx"
)

type NotificationController struct {
	notifClient pb.NotificationServiceClient
}

func NewNotificationController(notifClient pb.NotificationServiceClient) *NotificationController {
	return &NotificationController{notifClient: notifClient}
}

// ListNotifications godoc
// @Summary      Hộp thư thông báo
// @Tags         Notification
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/notifications [get]
func (c *NotificationController) ListNotifications(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}

	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.notifClient.ListNotifications(ctx.Request.Context(), &pb.ListNotificationsRequest{
		UserId:     userID,
		Type:       ctx.Query("type"),
		UnreadOnly: queryBool(ctx, "unread_only"),
		Page:       queryInt(ctx, "page"),
		PageSize:   queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OK(ctx, gin.H{
		"notifications": toNotificationDTOs(resp.Notifications),
		"pagination":    toNotifPaginationDTO(resp.Pagination),
		"unread_count":  resp.UnreadCount,
	})
}

// GetNotification godoc
// @Summary      Chi tiết một thông báo
// @Tags         Notification
// @Produce      json
// @Param        id path string true "Notification ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/notifications/{id} [get]
func (c *NotificationController) GetNotification(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.notifClient.GetNotification(ctx.Request.Context(), &pb.GetNotificationRequest{
		Id:     id,
		UserId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"notification": toNotificationDTO(resp.Notification)})
}

// MarkAsRead godoc
// @Summary      Đánh dấu đã đọc
// @Tags         Notification
// @Produce      json
// @Param        id path string true "Notification ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/notifications/{id}/read [put]
func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.notifClient.MarkAsRead(ctx.Request.Context(), &pb.MarkAsReadRequest{
		Id:     id,
		UserId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"unread_count": resp.UnreadCount}, resp.Message)
}

// MarkAllAsRead godoc
// @Summary      Đánh dấu tất cả đã đọc
// @Tags         Notification
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/notifications/read-all [put]
func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.notifClient.MarkAllAsRead(ctx.Request.Context(), &pb.MarkAllAsReadRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"marked_count": resp.MarkedCount}, resp.Message)
}

// DeleteNotification godoc
// @Summary      Xoá thông báo
// @Tags         Notification
// @Produce      json
// @Param        id path string true "Notification ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/notifications/{id} [delete]
func (c *NotificationController) DeleteNotification(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.notifClient.DeleteNotification(ctx.Request.Context(), &pb.DeleteNotificationRequest{
		Id:     id,
		UserId: selfID(ctx),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}

// GetUnreadCount godoc
// @Summary      Số thông báo chưa đọc
// @Description  App gọi ở mọi màn hình để vẽ chấm đỏ; con số này được cache trên Redis.
// @Tags         Notification
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/notifications/unread-count [get]
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.notifClient.GetUnreadCount(ctx.Request.Context(), &pb.GetUnreadCountRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"unread_count": resp.UnreadCount})
}

// GetPreferences godoc
// @Summary      Cài đặt nhận thông báo
// @Tags         Notification
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/notification-preferences [get]
func (c *NotificationController) GetPreferences(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	resp, err := c.notifClient.GetPreferences(ctx.Request.Context(), &pb.GetPreferencesRequest{
		UserId: userID,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"preference": toNotificationPreferenceDTO(resp.Preference)})
}

type UpdatePreferencesReq struct {
	InAppEnabled       bool   `json:"in_app_enabled"`
	PushEnabled        bool   `json:"push_enabled"`
	EmailEnabled       bool   `json:"email_enabled"`
	SMSEnabled         bool   `json:"sms_enabled"`
	MatchEventsEnabled bool   `json:"match_events_enabled"`
	PromotionEnabled   bool   `json:"promotion_enabled"`
	QuietHoursStart    string `json:"quiet_hours_start"`
	QuietHoursEnd      string `json:"quiet_hours_end"`
}

// UpdatePreferences godoc
// @Summary      Cập nhật cài đặt nhận thông báo
// @Tags         Notification
// @Accept       json
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/users/{user_id}/notification-preferences [put]
func (c *NotificationController) UpdatePreferences(ctx *gin.Context) {
	userID, ok := resolveOwnID(ctx, "user_id")
	if !ok {
		return
	}
	if !requireSelfOrAdmin(ctx, userID) {
		return
	}

	var req UpdatePreferencesReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.notifClient.UpdatePreferences(ctx.Request.Context(), &pb.UpdatePreferencesRequest{
		UserId:             userID,
		InAppEnabled:       req.InAppEnabled,
		PushEnabled:        req.PushEnabled,
		EmailEnabled:       req.EmailEnabled,
		SmsEnabled:         req.SMSEnabled,
		MatchEventsEnabled: req.MatchEventsEnabled,
		PromotionEnabled:   req.PromotionEnabled,
		QuietHoursStart:    req.QuietHoursStart,
		QuietHoursEnd:      req.QuietHoursEnd,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"preference": toNotificationPreferenceDTO(resp.Preference)}, resp.Message)
}

type AdminSendNotificationReq struct {
	UserIDs       []string `json:"user_ids"`
	BroadcastRole string   `json:"broadcast_role" binding:"omitempty,oneof=driver shipper admin"`
	Type          string   `json:"type"`
	Channel       string   `json:"channel" binding:"omitempty,oneof=in_app push email sms"`
	Title         string   `json:"title" binding:"required"`
	Body          string   `json:"body" binding:"required"`
	Data          string   `json:"data"`
}

// AdminSendNotification godoc
// @Summary      [Admin] Gửi thông báo thủ công
// @Tags         Admin-Notification
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/notifications/send [post]
func (c *NotificationController) AdminSendNotification(ctx *gin.Context) {
	var req AdminSendNotificationReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	userIDs := make([][]byte, 0, len(req.UserIDs))
	for _, raw := range req.UserIDs {
		id, err := uuidx.ParseRequired(raw)
		if err != nil {
			response.BadRequest(ctx, "INVALID_USER_ID", "user_ids chứa giá trị không phải UUID: "+raw)
			return
		}
		userIDs = append(userIDs, id)
	}

	resp, err := c.notifClient.AdminSendNotification(ctx.Request.Context(), &pb.AdminSendNotificationRequest{
		UserIds:       userIDs,
		BroadcastRole: req.BroadcastRole,
		Type:          req.Type,
		Channel:       req.Channel,
		Title:         req.Title,
		Body:          req.Body,
		Data:          req.Data,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"sent_count": resp.SentCount}, resp.Message)
}

// AdminListNotifications godoc
// @Summary      [Admin] Danh sách toàn bộ thông báo
// @Tags         Admin-Notification
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/notifications [get]
func (c *NotificationController) AdminListNotifications(ctx *gin.Context) {
	userID, ok := queryID(ctx, "user_id")
	if !ok {
		return
	}

	resp, err := c.notifClient.AdminListNotifications(ctx.Request.Context(), &pb.AdminListNotificationsRequest{
		UserId:   userID,
		Type:     ctx.Query("type"),
		Status:   ctx.Query("status"),
		Page:     queryInt(ctx, "page"),
		PageSize: queryInt(ctx, "page_size"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{
		"notifications": toNotificationDTOs(resp.Notifications),
		"pagination":    toNotifPaginationDTO(resp.Pagination),
	})
}

// AdminListTemplates godoc
// @Summary      [Admin] Danh sách template
// @Tags         Admin-Notification
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/notification-templates [get]
func (c *NotificationController) AdminListTemplates(ctx *gin.Context) {
	resp, err := c.notifClient.AdminListTemplates(ctx.Request.Context(), &pb.AdminListTemplatesRequest{
		Channel: ctx.Query("channel"),
		Locale:  ctx.Query("locale"),
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OK(ctx, gin.H{"templates": toNotificationTemplateDTOs(resp.Templates)})
}

type CreateTemplateReq struct {
	Code          string `json:"code" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Channel       string `json:"channel" binding:"omitempty,oneof=in_app push email sms"`
	Locale        string `json:"locale"`
	TitleTemplate string `json:"title_template" binding:"required"`
	BodyTemplate  string `json:"body_template" binding:"required"`
	IsActive      bool   `json:"is_active"`
}

// AdminCreateTemplate godoc
// @Summary      [Admin] Tạo template thông báo
// @Tags         Admin-Notification
// @Accept       json
// @Produce      json
// @Success      201 {object} response.Envelope
// @Router       /api/v1/admin/notification-templates [post]
func (c *NotificationController) AdminCreateTemplate(ctx *gin.Context) {
	var req CreateTemplateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.notifClient.AdminCreateTemplate(ctx.Request.Context(), &pb.AdminCreateTemplateRequest{
		Code:          req.Code,
		Name:          req.Name,
		Channel:       req.Channel,
		Locale:        req.Locale,
		TitleTemplate: req.TitleTemplate,
		BodyTemplate:  req.BodyTemplate,
		IsActive:      req.IsActive,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.Created(ctx, gin.H{"template": toNotificationTemplateDTO(resp.Template)}, resp.Message)
}

type UpdateTemplateReq struct {
	Name          string `json:"name"`
	TitleTemplate string `json:"title_template"`
	BodyTemplate  string `json:"body_template"`
	IsActive      bool   `json:"is_active"`
}

// AdminUpdateTemplate godoc
// @Summary      [Admin] Sửa template thông báo
// @Tags         Admin-Notification
// @Accept       json
// @Produce      json
// @Param        id path string true "Template ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/notification-templates/{id} [put]
func (c *NotificationController) AdminUpdateTemplate(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	var req UpdateTemplateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "VALIDATION_FAILED", err.Error())
		return
	}

	resp, err := c.notifClient.AdminUpdateTemplate(ctx.Request.Context(), &pb.AdminUpdateTemplateRequest{
		Id:            id,
		Name:          req.Name,
		TitleTemplate: req.TitleTemplate,
		BodyTemplate:  req.BodyTemplate,
		IsActive:      req.IsActive,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, gin.H{"template": toNotificationTemplateDTO(resp.Template)}, resp.Message)
}

// AdminDeleteTemplate godoc
// @Summary      [Admin] Xoá template thông báo
// @Tags         Admin-Notification
// @Produce      json
// @Param        id path string true "Template ID"
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/notification-templates/{id} [delete]
func (c *NotificationController) AdminDeleteTemplate(ctx *gin.Context) {
	id, ok := pathID(ctx, "id")
	if !ok {
		return
	}

	resp, err := c.notifClient.AdminDeleteTemplate(ctx.Request.Context(), &pb.AdminDeleteTemplateRequest{
		Id: id,
	})
	if err != nil {
		response.Error(ctx, err)
		return
	}
	response.OKMessage(ctx, nil, resp.Message)
}

// AdminGetNotificationStats godoc
// @Summary      [Admin] Thống kê thông báo
// @Tags         Admin-Notification
// @Produce      json
// @Success      200 {object} response.Envelope
// @Router       /api/v1/admin/notifications/stats [get]
func (c *NotificationController) AdminGetNotificationStats(ctx *gin.Context) {
	resp, err := c.notifClient.AdminGetNotificationStats(ctx.Request.Context(), &pb.AdminGetNotificationStatsRequest{})
	if err != nil {
		response.Error(ctx, err)
		return
	}

	response.OK(ctx, gin.H{
		"total_notifications":  resp.TotalNotifications,
		"unread_notifications": resp.UnreadNotifications,
		"sent_today":           resp.SentToday,
		"failed_notifications": resp.FailedNotifications,
		"total_templates":      resp.TotalTemplates,
	})
}