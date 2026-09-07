package controller

import (
	"context"
	"encoding/base64"

	"notification_service/internal/biz"
	cerr "notification_service/internal/common/errors"
	"notification_service/internal/entity"
	"notification_service/internal/mapper"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/notification_service/v1"
	"github.com/logistic/pkg/uuidx"
)

type notificationController struct {
	pb.UnimplementedNotificationServiceServer
	engine biz.NotificationEngine
	mapper mapper.AppMapper
}

func NewNotificationController(engine biz.NotificationEngine, appMapper mapper.AppMapper) pb.NotificationServiceServer {
	return &notificationController{engine: engine, mapper: appMapper}
}

func parseID(raw []byte, invalid error) (uuid.UUID, error) {
	id, err := uuidx.FromBytes(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, invalid
	}
	return id, nil
}

func parseOptionalID(raw []byte, invalid error) (uuid.UUID, error) {
	if len(raw) == 0 {
		return uuid.Nil, nil
	}
	return parseID(raw, invalid)
}

func (c *notificationController) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	param, err := c.mapper.PbListNotificationsToParam(req)
	if err != nil {
		return nil, cerr.ErrInvalidUserID.WithCause(err)
	}

	res, err := c.engine.List(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.ListNotificationsResponse{
		Notifications: c.mapper.EntityNotificationListToPbList(res.Notifications),
		Pagination:    c.mapper.EntityPaginationToPb(res.Pagination),
		UnreadCount:   res.UnreadCount,
	}, nil
}

func (c *notificationController) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	id, err := parseID(req.Id, cerr.ErrInvalidNotifID)
	if err != nil {
		return nil, err
	}
	userID, err := parseOptionalID(req.UserId, cerr.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	n, err := c.engine.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetNotificationResponse{Notification: c.mapper.EntityNotificationToPb(*n)}, nil
}

func (c *notificationController) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	id, err := parseID(req.Id, cerr.ErrInvalidNotifID)
	if err != nil {
		return nil, err
	}
	userID, err := parseOptionalID(req.UserId, cerr.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	unread, err := c.engine.MarkAsRead(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAsReadResponse{
		Message:     "Đã đánh dấu là đã đọc",
		UnreadCount: unread,
	}, nil
}

func (c *notificationController) MarkAllAsRead(ctx context.Context, req *pb.MarkAllAsReadRequest) (*pb.MarkAllAsReadResponse, error) {
	userID, err := parseID(req.UserId, cerr.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	marked, err := c.engine.MarkAllAsRead(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAllAsReadResponse{
		Message:     "Đã đánh dấu tất cả là đã đọc",
		MarkedCount: marked,
	}, nil
}

func (c *notificationController) DeleteNotification(ctx context.Context, req *pb.DeleteNotificationRequest) (*pb.DeleteNotificationResponse, error) {
	id, err := parseID(req.Id, cerr.ErrInvalidNotifID)
	if err != nil {
		return nil, err
	}
	userID, err := parseOptionalID(req.UserId, cerr.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	if err := c.engine.Delete(ctx, id, userID); err != nil {
		return nil, err
	}
	return &pb.DeleteNotificationResponse{Message: "Đã xoá thông báo"}, nil
}

func (c *notificationController) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	userID, err := parseID(req.UserId, cerr.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	count, err := c.engine.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetUnreadCountResponse{UnreadCount: count}, nil
}

func (c *notificationController) GetPreferences(ctx context.Context, req *pb.GetPreferencesRequest) (*pb.GetPreferencesResponse, error) {
	userID, err := parseID(req.UserId, cerr.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	pref, err := c.engine.GetPreference(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetPreferencesResponse{Preference: c.mapper.EntityPreferenceToPb(*pref)}, nil
}

func (c *notificationController) UpdatePreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.UpdatePreferencesResponse, error) {
	param, err := c.mapper.PbUpdatePreferencesToParam(req)
	if err != nil {
		return nil, cerr.ErrInvalidUserID.WithCause(err)
	}

	pref, err := c.engine.UpdatePreference(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdatePreferencesResponse{
		Preference: c.mapper.EntityPreferenceToPb(*pref),
		Message:    "Cập nhật cài đặt thông báo thành công",
	}, nil
}

func (c *notificationController) AdminSendNotification(ctx context.Context, req *pb.AdminSendNotificationRequest) (*pb.AdminSendNotificationResponse, error) {
	userIDs := make([]uuid.UUID, 0, len(req.UserIds))
	for _, raw := range req.UserIds {
		id, err := parseID(raw, cerr.ErrInvalidUserID)
		if err != nil {
			return nil, cerr.ErrInvalidUserID.WithDetail("user_id", base64.RawURLEncoding.EncodeToString(raw))
		}
		userIDs = append(userIDs, id)
	}

	res, err := c.engine.AdminSend(ctx, &entity.SendNotificationParam{
		UserIDs:       userIDs,
		BroadcastRole: req.BroadcastRole,
		Type:          req.Type,
		Channel:       req.Channel,
		Title:         req.Title,
		Body:          req.Body,
		Data:          req.Data,
	})
	if err != nil {
		return nil, err
	}

	return &pb.AdminSendNotificationResponse{
		Message:   res.Message,
		SentCount: res.SentCount,
	}, nil
}

func (c *notificationController) AdminListNotifications(ctx context.Context, req *pb.AdminListNotificationsRequest) (*pb.AdminListNotificationsResponse, error) {
	param, err := c.mapper.PbAdminListNotificationsToParam(req)
	if err != nil {
		return nil, cerr.ErrInvalidUserID.WithCause(err)
	}

	res, err := c.engine.AdminList(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminListNotificationsResponse{
		Notifications: c.mapper.EntityNotificationListToPbList(res.Notifications),
		Pagination:    c.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (c *notificationController) AdminListTemplates(ctx context.Context, req *pb.AdminListTemplatesRequest) (*pb.AdminListTemplatesResponse, error) {
	param := c.mapper.PbListTemplatesToParam(req)

	list, err := c.engine.AdminListTemplates(ctx, &param)
	if err != nil {
		return nil, err
	}
	return &pb.AdminListTemplatesResponse{Templates: c.mapper.EntityTemplateListToPbList(list)}, nil
}

func (c *notificationController) AdminCreateTemplate(ctx context.Context, req *pb.AdminCreateTemplateRequest) (*pb.AdminCreateTemplateResponse, error) {
	param := c.mapper.PbCreateTemplateToParam(req)

	tpl, err := c.engine.AdminCreateTemplate(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminCreateTemplateResponse{
		Template: c.mapper.EntityTemplateToPb(*tpl),
		Message:  "Tạo template thành công",
	}, nil
}

func (c *notificationController) AdminUpdateTemplate(ctx context.Context, req *pb.AdminUpdateTemplateRequest) (*pb.AdminUpdateTemplateResponse, error) {
	param, err := c.mapper.PbUpdateTemplateToParam(req)
	if err != nil {
		return nil, cerr.ErrInvalidTemplateID.WithCause(err)
	}

	tpl, err := c.engine.AdminUpdateTemplate(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminUpdateTemplateResponse{
		Template: c.mapper.EntityTemplateToPb(*tpl),
		Message:  "Cập nhật template thành công",
	}, nil
}

func (c *notificationController) AdminDeleteTemplate(ctx context.Context, req *pb.AdminDeleteTemplateRequest) (*pb.AdminDeleteTemplateResponse, error) {
	id, err := parseID(req.Id, cerr.ErrInvalidTemplateID)
	if err != nil {
		return nil, err
	}
	if err := c.engine.AdminDeleteTemplate(ctx, id); err != nil {
		return nil, err
	}
	return &pb.AdminDeleteTemplateResponse{Message: "Xoá template thành công"}, nil
}

func (c *notificationController) AdminGetNotificationStats(ctx context.Context, _ *pb.AdminGetNotificationStatsRequest) (*pb.AdminGetNotificationStatsResponse, error) {
	stats, err := c.engine.AdminGetStats(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.AdminGetNotificationStatsResponse{
		TotalNotifications:  stats.TotalNotifications,
		UnreadNotifications: stats.UnreadNotifications,
		SentToday:           stats.SentToday,
		FailedNotifications: stats.FailedNotifications,
		TotalTemplates:      stats.TotalTemplates,
	}, nil
}