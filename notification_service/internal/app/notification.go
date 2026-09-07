package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"notification_service/internal/entity"

	"github.com/google/uuid"
)

type notificationEngineImpl struct {
	repo NotificationRepo
}

var _ NotificationEngine = (*notificationEngineImpl)(nil)

func NewNotificationEngine(repo NotificationRepo) NotificationEngine {
	return &notificationEngineImpl{repo: repo}
}

func (e *notificationEngineImpl) List(ctx context.Context, param *entity.ListNotificationsParam) (*entity.ListNotificationsResult, error) {
	if param.UserID == uuid.Nil {
		return nil, entity.ErrInvalidUserID
	}

	page, pageSize, _ := entity.NormalizePaging(param.Page, param.PageSize)
	list, total, err := e.repo.List(ctx, param)
	if err != nil {
		return nil, err
	}

	unread, err := e.repo.CountUnread(ctx, param.UserID)
	if err != nil {
		return nil, err
	}

	return &entity.ListNotificationsResult{
		Notifications: list,
		Pagination:    entity.BuildPagination(page, pageSize, total),
		UnreadCount:   unread,
	}, nil
}

func (e *notificationEngineImpl) Get(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidNotifID
	}

	n, err := e.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if userID != uuid.Nil && n.UserID != userID {
		return nil, entity.ErrNotificationNotOwned
	}
	return n, nil
}

func (e *notificationEngineImpl) MarkAsRead(ctx context.Context, id, userID uuid.UUID) (int64, error) {
	n, err := e.Get(ctx, id, userID)
	if err != nil {
		return 0, err
	}

	if !n.IsRead {
		if _, err := e.repo.MarkAsRead(ctx, id); err != nil {
			return 0, err
		}
	}
	return e.repo.CountUnread(ctx, n.UserID)
}

func (e *notificationEngineImpl) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	if userID == uuid.Nil {
		return 0, entity.ErrInvalidUserID
	}
	return e.repo.MarkAllAsRead(ctx, userID)
}

func (e *notificationEngineImpl) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if _, err := e.Get(ctx, id, userID); err != nil {
		return err
	}
	return e.repo.Delete(ctx, id)
}

func (e *notificationEngineImpl) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	if userID == uuid.Nil {
		return 0, entity.ErrInvalidUserID
	}
	return e.repo.CountUnread(ctx, userID)
}

func (e *notificationEngineImpl) GetPreference(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreference, error) {
	if userID == uuid.Nil {
		return nil, entity.ErrInvalidUserID
	}
	return e.repo.GetOrCreatePreference(ctx, userID)
}

func (e *notificationEngineImpl) UpdatePreference(ctx context.Context, param *entity.UpdatePreferenceParam) (*entity.NotificationPreference, error) {
	if param.UserID == uuid.Nil {
		return nil, entity.ErrInvalidUserID
	}
	return e.repo.UpdatePreference(ctx, param)
}

func (e *notificationEngineImpl) DispatchEvent(
	ctx context.Context,
	eventID, routingKey, source string,
	params []entity.CreateNotificationParam,
) (int64, error) {
	if eventID == "" {
		return 0, entity.ErrInvalidNotifID.WithMessage("event_id là bắt buộc để chống xử lý trùng")
	}
	if len(params) == 0 {
		return 0, nil
	}

	allowed := make([]entity.CreateNotificationParam, 0, len(params))
	now := time.Now()

	for i := range params {
		p := params[i]
		if p.UserID == uuid.Nil {
			continue
		}
		if p.Channel == "" {
			p.Channel = entity.ChannelInApp
		}

		pref, err := e.repo.GetOrCreatePreference(ctx, p.UserID)
		if err != nil {
			log.Printf("[app] không đọc được cài đặt của %s (%v) — vẫn gửi theo mặc định", p.UserID, err)
			allowed = append(allowed, p)
			continue
		}

		if !pref.AllowsType(p.Type) || !pref.AllowsChannel(p.Channel) {
			continue
		}

		if p.Channel == entity.ChannelPush && pref.IsQuietHour(now) {
			p.Channel = entity.ChannelInApp
		}

		allowed = append(allowed, p)
	}

	if len(allowed) == 0 {
		return e.repo.CreateWithEventGuard(ctx, eventID, routingKey, source, nil)
	}

	return e.repo.CreateWithEventGuard(ctx, eventID, routingKey, source, allowed)
}

func (e *notificationEngineImpl) AdminSend(ctx context.Context, param *entity.SendNotificationParam) (*entity.SendNotificationResult, error) {
	if param.Title == "" {
		return nil, entity.ErrTitleRequired
	}
	if param.Body == "" {
		return nil, entity.ErrBodyRequired
	}
	if len(param.UserIDs) == 0 && param.BroadcastRole == "" {
		return nil, entity.ErrNoRecipient
	}
	if param.BroadcastRole != "" && !entity.IsValidRole(param.BroadcastRole) {
		return nil, entity.ErrInvalidRole.WithDetail("broadcast_role", param.BroadcastRole)
	}

	channel := param.Channel
	if channel == "" {
		channel = entity.ChannelInApp
	}
	if !entity.IsValidChannel(channel) {
		return nil, entity.ErrInvalidChannel.WithDetail("channel", channel)
	}

	notiType := param.Type
	if notiType == "" {
		notiType = entity.TypeSystem
	}

	role := param.BroadcastRole
	if role == "" {
		role = entity.RoleDriver
	}

	params := make([]entity.CreateNotificationParam, 0, len(param.UserIDs))
	for _, uid := range param.UserIDs {
		if uid == uuid.Nil {
			continue
		}
		params = append(params, entity.CreateNotificationParam{
			UserID:        uid,
			RecipientRole: role,
			Type:          notiType,
			Channel:       channel,
			Title:         param.Title,
			Body:          param.Body,
			Data:          param.Data,
		})
	}

	if len(params) == 0 {
		return nil, entity.ErrNoRecipient.WithMessage(
			"broadcast theo vai trò chưa được hỗ trợ — hãy truyền danh sách user_ids cụ thể")
	}

	sent, err := e.repo.CreateBatch(ctx, params)
	if err != nil {
		return nil, err
	}

	return &entity.SendNotificationResult{
		SentCount: sent,
		Message:   fmt.Sprintf("Đã gửi %d thông báo", sent),
	}, nil
}

func (e *notificationEngineImpl) AdminList(ctx context.Context, param *entity.AdminListNotificationsParam) (*entity.ListNotificationsResult, error) {
	if param.Status != "" && !entity.IsValidStatus(param.Status) {
		return nil, entity.ErrInvalidStatus.WithDetail("status", param.Status)
	}

	page, pageSize, _ := entity.NormalizePaging(param.Page, param.PageSize)
	list, total, err := e.repo.AdminList(ctx, param)
	if err != nil {
		return nil, err
	}

	return &entity.ListNotificationsResult{
		Notifications: list,
		Pagination:    entity.BuildPagination(page, pageSize, total),
	}, nil
}

func (e *notificationEngineImpl) AdminListTemplates(ctx context.Context, param *entity.ListTemplatesParam) ([]entity.NotificationTemplate, error) {
	if param.Channel != "" && !entity.IsValidChannel(param.Channel) {
		return nil, entity.ErrInvalidChannel.WithDetail("channel", param.Channel)
	}
	return e.repo.ListTemplates(ctx, param)
}

func (e *notificationEngineImpl) AdminCreateTemplate(ctx context.Context, param *entity.CreateTemplateParam) (*entity.NotificationTemplate, error) {
	if param.Code == "" {
		return nil, entity.ErrCodeRequired
	}
	if param.TitleTemplate == "" {
		return nil, entity.ErrTitleRequired
	}
	if param.BodyTemplate == "" {
		return nil, entity.ErrBodyRequired
	}
	if param.Channel != "" && !entity.IsValidChannel(param.Channel) {
		return nil, entity.ErrInvalidChannel.WithDetail("channel", param.Channel)
	}
	return e.repo.CreateTemplate(ctx, param)
}

func (e *notificationEngineImpl) AdminUpdateTemplate(ctx context.Context, param *entity.UpdateTemplateParam) (*entity.NotificationTemplate, error) {
	if param.ID == uuid.Nil {
		return nil, entity.ErrInvalidTemplateID
	}
	return e.repo.UpdateTemplate(ctx, param)
}

func (e *notificationEngineImpl) AdminDeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return entity.ErrInvalidTemplateID
	}
	return e.repo.DeleteTemplate(ctx, id)
}

func (e *notificationEngineImpl) AdminGetStats(ctx context.Context) (*entity.NotificationStats, error) {
	total, err := e.repo.CountAll(ctx)
	if err != nil {
		return nil, err
	}
	unread, err := e.repo.CountUnreadAll(ctx)
	if err != nil {
		return nil, err
	}
	sentToday, err := e.repo.CountSentToday(ctx)
	if err != nil {
		return nil, err
	}
	failed, err := e.repo.CountByStatus(ctx, entity.StatusFailed)
	if err != nil {
		return nil, err
	}
	templates, err := e.repo.CountTemplates(ctx)
	if err != nil {
		return nil, err
	}

	return &entity.NotificationStats{
		TotalNotifications:  total,
		UnreadNotifications: unread,
		SentToday:           sentToday,
		FailedNotifications: failed,
		TotalTemplates:      templates,
	}, nil
}

func (e *notificationEngineImpl) RenderFromTemplate(ctx context.Context, code, channel, locale string, vars map[string]string) (string, string, bool) {
	if locale == "" {
		locale = "vi"
	}
	if channel == "" {
		channel = entity.ChannelInApp
	}

	tpl, err := e.repo.GetTemplateByCode(ctx, code, channel, locale)
	if err != nil {
		return "", "", false
	}

	title, body := tpl.Render(vars)
	return title, body, true
}

func MarshalData(v any) string {
	blob, err := json.Marshal(v)
	if err != nil {
		log.Printf("[app] marshal notification data thất bại: %v", err)
		return ""
	}
	return string(blob)
}