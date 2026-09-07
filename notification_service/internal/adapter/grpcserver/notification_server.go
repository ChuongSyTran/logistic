package grpcserver

import (
	"context"
	"encoding/base64"

	"notification_service/internal/app"
	"notification_service/internal/entity"
	"notification_service/internal/mapper"

	"github.com/google/uuid"
	pb "github.com/logistic/api/logistic/notification_service/v1"
	"github.com/logistic/pkg/uuidx"
)

type notificationServer struct {
	pb.UnimplementedNotificationServiceServer
	engine app.NotificationEngine
	mapper mapper.AppMapper
}

func NewNotificationServer(engine app.NotificationEngine, appMapper mapper.AppMapper) pb.NotificationServiceServer {
	return &notificationServer{engine: engine, mapper: appMapper}
}

// NewNotificationController là alias giữ tương thích nếu cần.
var NewNotificationController = NewNotificationServer

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

func (s *notificationServer) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	param, err := s.mapper.PbListNotificationsToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	res, err := s.engine.List(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.ListNotificationsResponse{
		Notifications: s.mapper.EntityNotificationListToPbList(res.Notifications),
		Pagination:    s.mapper.EntityPaginationToPb(res.Pagination),
		UnreadCount:   res.UnreadCount,
	}, nil
}

func (s *notificationServer) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidNotifID)
	if err != nil {
		return nil, err
	}
	userID, err := parseOptionalID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	n, err := s.engine.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetNotificationResponse{Notification: s.mapper.EntityNotificationToPb(*n)}, nil
}

func (s *notificationServer) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidNotifID)
	if err != nil {
		return nil, err
	}
	userID, err := parseOptionalID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	unread, err := s.engine.MarkAsRead(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAsReadResponse{
		Message:     "Đã đánh dấu là đã đọc",
		UnreadCount: unread,
	}, nil
}

func (s *notificationServer) MarkAllAsRead(ctx context.Context, req *pb.MarkAllAsReadRequest) (*pb.MarkAllAsReadResponse, error) {
	userID, err := parseID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	marked, err := s.engine.MarkAllAsRead(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAllAsReadResponse{
		Message:     "Đã đánh dấu tất cả là đã đọc",
		MarkedCount: marked,
	}, nil
}

func (s *notificationServer) DeleteNotification(ctx context.Context, req *pb.DeleteNotificationRequest) (*pb.DeleteNotificationResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidNotifID)
	if err != nil {
		return nil, err
	}
	userID, err := parseOptionalID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	if err := s.engine.Delete(ctx, id, userID); err != nil {
		return nil, err
	}
	return &pb.DeleteNotificationResponse{Message: "Đã xoá thông báo"}, nil
}

func (s *notificationServer) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	userID, err := parseID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	count, err := s.engine.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetUnreadCountResponse{UnreadCount: count}, nil
}

func (s *notificationServer) GetPreferences(ctx context.Context, req *pb.GetPreferencesRequest) (*pb.GetPreferencesResponse, error) {
	userID, err := parseID(req.UserId, entity.ErrInvalidUserID)
	if err != nil {
		return nil, err
	}

	pref, err := s.engine.GetPreference(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &pb.GetPreferencesResponse{Preference: s.mapper.EntityPreferenceToPb(*pref)}, nil
}

func (s *notificationServer) UpdatePreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.UpdatePreferencesResponse, error) {
	param, err := s.mapper.PbUpdatePreferencesToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	pref, err := s.engine.UpdatePreference(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.UpdatePreferencesResponse{
		Preference: s.mapper.EntityPreferenceToPb(*pref),
		Message:    "Cập nhật cài đặt thông báo thành công",
	}, nil
}

func (s *notificationServer) AdminSendNotification(ctx context.Context, req *pb.AdminSendNotificationRequest) (*pb.AdminSendNotificationResponse, error) {
	userIDs := make([]uuid.UUID, 0, len(req.UserIds))
	for _, raw := range req.UserIds {
		id, err := parseID(raw, entity.ErrInvalidUserID)
		if err != nil {
			return nil, entity.ErrInvalidUserID.WithDetail("user_id", base64.RawURLEncoding.EncodeToString(raw))
		}
		userIDs = append(userIDs, id)
	}

	res, err := s.engine.AdminSend(ctx, &entity.SendNotificationParam{
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

func (s *notificationServer) AdminListNotifications(ctx context.Context, req *pb.AdminListNotificationsRequest) (*pb.AdminListNotificationsResponse, error) {
	param, err := s.mapper.PbAdminListNotificationsToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidUserID.WithCause(err)
	}

	res, err := s.engine.AdminList(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminListNotificationsResponse{
		Notifications: s.mapper.EntityNotificationListToPbList(res.Notifications),
		Pagination:    s.mapper.EntityPaginationToPb(res.Pagination),
	}, nil
}

func (s *notificationServer) AdminListTemplates(ctx context.Context, req *pb.AdminListTemplatesRequest) (*pb.AdminListTemplatesResponse, error) {
	param := s.mapper.PbListTemplatesToParam(req)

	list, err := s.engine.AdminListTemplates(ctx, &param)
	if err != nil {
		return nil, err
	}
	return &pb.AdminListTemplatesResponse{Templates: s.mapper.EntityTemplateListToPbList(list)}, nil
}

func (s *notificationServer) AdminCreateTemplate(ctx context.Context, req *pb.AdminCreateTemplateRequest) (*pb.AdminCreateTemplateResponse, error) {
	param := s.mapper.PbCreateTemplateToParam(req)

	tpl, err := s.engine.AdminCreateTemplate(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminCreateTemplateResponse{
		Template: s.mapper.EntityTemplateToPb(*tpl),
		Message:  "Tạo template thành công",
	}, nil
}

func (s *notificationServer) AdminUpdateTemplate(ctx context.Context, req *pb.AdminUpdateTemplateRequest) (*pb.AdminUpdateTemplateResponse, error) {
	param, err := s.mapper.PbUpdateTemplateToParam(req)
	if err != nil {
		return nil, entity.ErrInvalidTemplateID.WithCause(err)
	}

	tpl, err := s.engine.AdminUpdateTemplate(ctx, &param)
	if err != nil {
		return nil, err
	}

	return &pb.AdminUpdateTemplateResponse{
		Template: s.mapper.EntityTemplateToPb(*tpl),
		Message:  "Cập nhật template thành công",
	}, nil
}

func (s *notificationServer) AdminDeleteTemplate(ctx context.Context, req *pb.AdminDeleteTemplateRequest) (*pb.AdminDeleteTemplateResponse, error) {
	id, err := parseID(req.Id, entity.ErrInvalidTemplateID)
	if err != nil {
		return nil, err
	}
	if err := s.engine.AdminDeleteTemplate(ctx, id); err != nil {
		return nil, err
	}
	return &pb.AdminDeleteTemplateResponse{Message: "Xoá template thành công"}, nil
}

func (s *notificationServer) AdminGetNotificationStats(ctx context.Context, _ *pb.AdminGetNotificationStatsRequest) (*pb.AdminGetNotificationStatsResponse, error) {
	stats, err := s.engine.AdminGetStats(ctx)
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