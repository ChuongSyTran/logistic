package app

import (
	"context"

	"notification_service/internal/entity"

	"github.com/google/uuid"
)

// NotificationRepository quản lý lưu trữ và truy vấn các bản ghi thông báo và chống trùng lặp sự kiện.
type NotificationRepository interface {
	Create(ctx context.Context, param *entity.CreateNotificationParam) (*entity.Notification, error)
	CreateBatch(ctx context.Context, params []entity.CreateNotificationParam) (int64, error)
	CreateWithEventGuard(ctx context.Context, eventID, routingKey, source string, params []entity.CreateNotificationParam) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error)
	List(ctx context.Context, param *entity.ListNotificationsParam) ([]entity.Notification, int64, error)
	AdminList(ctx context.Context, param *entity.AdminListNotificationsParam) ([]entity.Notification, int64, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) (*entity.Notification, error)
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	CountUnreadAll(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	CountSentToday(ctx context.Context) (int64, error)
}

// TemplateRepository quản lý các mẫu thông báo (template) đa kênh và đa ngôn ngữ.
type TemplateRepository interface {
	CreateTemplate(ctx context.Context, param *entity.CreateTemplateParam) (*entity.NotificationTemplate, error)
	GetTemplateByCode(ctx context.Context, code, channel, locale string) (*entity.NotificationTemplate, error)
	ListTemplates(ctx context.Context, param *entity.ListTemplatesParam) ([]entity.NotificationTemplate, error)
	UpdateTemplate(ctx context.Context, param *entity.UpdateTemplateParam) (*entity.NotificationTemplate, error)
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	CountTemplates(ctx context.Context) (int64, error)
}

// PreferenceRepository quản lý tuỳ chọn nhận thông báo của người dùng (kênh, giờ yên lặng, loại thông báo).
type PreferenceRepository interface {
	GetOrCreatePreference(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreference, error)
	UpdatePreference(ctx context.Context, param *entity.UpdatePreferenceParam) (*entity.NotificationPreference, error)
}

// NotificationRepo là interface hợp thành để adapter persistence cài đặt một lần và DI inject.
type NotificationRepo interface {
	NotificationRepository
	TemplateRepository
	PreferenceRepository
}

// NotificationEngine là inbound (driving) port định nghĩa toàn bộ usecase nghiệp vụ thông báo.
type NotificationEngine interface {
	List(ctx context.Context, param *entity.ListNotificationsParam) (*entity.ListNotificationsResult, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error)
	MarkAsRead(ctx context.Context, id, userID uuid.UUID) (int64, error)
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	GetPreference(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreference, error)
	UpdatePreference(ctx context.Context, param *entity.UpdatePreferenceParam) (*entity.NotificationPreference, error)

	DispatchEvent(ctx context.Context, eventID, routingKey, source string, params []entity.CreateNotificationParam) (int64, error)
	AdminSend(ctx context.Context, param *entity.SendNotificationParam) (*entity.SendNotificationResult, error)

	AdminList(ctx context.Context, param *entity.AdminListNotificationsParam) (*entity.ListNotificationsResult, error)
	AdminListTemplates(ctx context.Context, param *entity.ListTemplatesParam) ([]entity.NotificationTemplate, error)
	AdminCreateTemplate(ctx context.Context, param *entity.CreateTemplateParam) (*entity.NotificationTemplate, error)
	AdminUpdateTemplate(ctx context.Context, param *entity.UpdateTemplateParam) (*entity.NotificationTemplate, error)
	AdminDeleteTemplate(ctx context.Context, id uuid.UUID) error
	AdminGetStats(ctx context.Context) (*entity.NotificationStats, error)

	RenderFromTemplate(ctx context.Context, code, channel, locale string, vars map[string]string) (string, string, bool)
}