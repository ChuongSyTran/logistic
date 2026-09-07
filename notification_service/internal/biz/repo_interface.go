package biz

import (
	"context"

	"notification_service/internal/entity"

	"github.com/google/uuid"
)

type NotificationRepo interface {
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

	CreateTemplate(ctx context.Context, param *entity.CreateTemplateParam) (*entity.NotificationTemplate, error)
	GetTemplateByCode(ctx context.Context, code, channel, locale string) (*entity.NotificationTemplate, error)
	ListTemplates(ctx context.Context, param *entity.ListTemplatesParam) ([]entity.NotificationTemplate, error)
	UpdateTemplate(ctx context.Context, param *entity.UpdateTemplateParam) (*entity.NotificationTemplate, error)
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	CountTemplates(ctx context.Context) (int64, error)

	GetOrCreatePreference(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreference, error)
	UpdatePreference(ctx context.Context, param *entity.UpdatePreferenceParam) (*entity.NotificationPreference, error)
}