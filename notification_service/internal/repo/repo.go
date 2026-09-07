package repo

import (
	"context"
	"log"
	"time"

	"notification_service/ent"
	"notification_service/ent/notification"
	"notification_service/ent/notificationpreference"
	"notification_service/ent/notificationtemplate"
	"notification_service/internal/biz"
	cerr "notification_service/internal/common/errors"
	"notification_service/internal/entity"
	"notification_service/internal/mapper"

	"github.com/google/uuid"
	"github.com/logistic/pkg/cache"
)

const ttlUnreadCount = 5 * time.Minute

type notificationRepoImpl struct {
	client *ent.Client
	cache  *cache.Client
	mapper mapper.AppMapper
}

var _ biz.NotificationRepo = (*notificationRepoImpl)(nil)

func NewNotificationRepo(client *ent.Client, redis *cache.Client, appMapper mapper.AppMapper) biz.NotificationRepo {
	return &notificationRepoImpl{client: client, cache: redis, mapper: appMapper}
}

func (r *notificationRepoImpl) keyUnread(userID uuid.UUID) string {
	return r.cache.Key("unread", userID.String())
}

func (r *notificationRepoImpl) invalidateUnread(ctx context.Context, userIDs ...uuid.UUID) {
	if r.cache == nil || len(userIDs) == 0 {
		return
	}
	keys := make([]string, 0, len(userIDs))
	seen := make(map[uuid.UUID]struct{}, len(userIDs))
	for _, id := range userIDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		keys = append(keys, r.keyUnread(id))
	}
	if err := r.cache.Delete(ctx, keys...); err != nil {
		log.Printf("[repo] invalidate unread counters failed: %v", err)
	}
}

func (r *notificationRepoImpl) Create(ctx context.Context, param *entity.CreateNotificationParam) (*entity.Notification, error) {
	dao, err := r.buildCreate(r.client.Notification.Create(), param).Save(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrNotificationNotFound)
	}

	r.invalidateUnread(ctx, param.UserID)
	e := r.mapper.EntNotificationToEntity(dao)
	return &e, nil
}

func (r *notificationRepoImpl) buildCreate(b *ent.NotificationCreate, param *entity.CreateNotificationParam) *ent.NotificationCreate {
	return b.
		SetUserID(param.UserID).
		SetRecipientRole(notification.RecipientRole(param.RecipientRole)).
		SetType(param.Type).
		SetChannel(notification.Channel(param.Channel)).
		SetTitle(param.Title).
		SetBody(param.Body).
		SetData(param.Data).
		SetRefType(param.RefType).
		SetRefID(param.RefID).
		SetStatus(notification.StatusSent)
}

func (r *notificationRepoImpl) CreateBatch(ctx context.Context, params []entity.CreateNotificationParam) (int64, error) {
	if len(params) == 0 {
		return 0, nil
	}

	builders := make([]*ent.NotificationCreate, 0, len(params))
	userIDs := make([]uuid.UUID, 0, len(params))
	for i := range params {
		builders = append(builders, r.buildCreate(r.client.Notification.Create(), &params[i]))
		userIDs = append(userIDs, params[i].UserID)
	}

	created, err := r.client.Notification.CreateBulk(builders...).Save(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	r.invalidateUnread(ctx, userIDs...)
	return int64(len(created)), nil
}

func (r *notificationRepoImpl) CreateWithEventGuard(
	ctx context.Context,
	eventID, routingKey, source string,
	params []entity.CreateNotificationParam,
) (int64, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return 0, wrapError(err, nil)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ProcessedEvent.Create().
		SetEventID(eventID).
		SetRoutingKey(routingKey).
		SetSource(source).
		Save(ctx); err != nil {
		return 0, wrapError(err, nil)
	}

	var createdCount int
	userIDs := make([]uuid.UUID, 0, len(params))

	if len(params) > 0 {
		builders := make([]*ent.NotificationCreate, 0, len(params))
		for i := range params {
			builders = append(builders, r.buildCreate(tx.Notification.Create(), &params[i]))
			userIDs = append(userIDs, params[i].UserID)
		}

		created, err := tx.Notification.CreateBulk(builders...).Save(ctx)
		if err != nil {
			return 0, wrapError(err, cerr.ErrNotificationNotFound)
		}
		createdCount = len(created)
	}

	if err := tx.Commit(); err != nil {
		return 0, wrapError(err, nil)
	}

	r.invalidateUnread(ctx, userIDs...)
	return int64(createdCount), nil
}

func (r *notificationRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	dao, err := r.client.Notification.Get(ctx, id)
	if err != nil {
		return nil, wrapError(err, cerr.ErrNotificationNotFound)
	}
	e := r.mapper.EntNotificationToEntity(dao)
	return &e, nil
}

func (r *notificationRepoImpl) List(ctx context.Context, param *entity.ListNotificationsParam) ([]entity.Notification, int64, error) {
	_, pageSize, offset := entity.NormalizePaging(param.Page, param.PageSize)

	q := r.client.Notification.Query().Where(notification.UserIDEQ(param.UserID))
	if param.Type != "" {
		q = q.Where(notification.TypeEQ(param.Type))
	}
	if param.UnreadOnly {
		q = q.Where(notification.IsReadEQ(false))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	daos, err := q.
		Order(ent.Desc(notification.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	return r.mapper.EntNotificationListToEntityList(daos), int64(total), nil
}

func (r *notificationRepoImpl) AdminList(ctx context.Context, param *entity.AdminListNotificationsParam) ([]entity.Notification, int64, error) {
	_, pageSize, offset := entity.NormalizePaging(param.Page, param.PageSize)

	q := r.client.Notification.Query()
	if param.UserID != uuid.Nil {
		q = q.Where(notification.UserIDEQ(param.UserID))
	}
	if param.Type != "" {
		q = q.Where(notification.TypeEQ(param.Type))
	}
	if param.Status != "" {
		q = q.Where(notification.StatusEQ(notification.Status(param.Status)))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	daos, err := q.
		Order(ent.Desc(notification.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	return r.mapper.EntNotificationListToEntityList(daos), int64(total), nil
}

func (r *notificationRepoImpl) MarkAsRead(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	dao, err := r.client.Notification.UpdateOneID(id).
		SetIsRead(true).
		SetStatus(notification.StatusRead).
		SetReadAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrNotificationNotFound)
	}

	r.invalidateUnread(ctx, dao.UserID)
	e := r.mapper.EntNotificationToEntity(dao)
	return &e, nil
}

func (r *notificationRepoImpl) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	n, err := r.client.Notification.Update().
		Where(notification.UserIDEQ(userID), notification.IsReadEQ(false)).
		SetIsRead(true).
		SetStatus(notification.StatusRead).
		SetReadAt(time.Now()).
		Save(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	r.invalidateUnread(ctx, userID)
	return int64(n), nil
}

func (r *notificationRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	dao, err := r.client.Notification.Get(ctx, id)
	if err != nil {
		return wrapError(err, cerr.ErrNotificationNotFound)
	}
	if err := r.client.Notification.DeleteOneID(id).Exec(ctx); err != nil {
		return wrapError(err, cerr.ErrNotificationNotFound)
	}
	r.invalidateUnread(ctx, dao.UserID)
	return nil
}

func (r *notificationRepoImpl) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := r.keyUnread(userID)

	var cached int64
	if r.cache != nil {
		if err := r.cache.GetJSON(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}

	n, err := r.client.Notification.Query().
		Where(notification.UserIDEQ(userID), notification.IsReadEQ(false)).
		Count(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}

	if r.cache != nil {
		_ = r.cache.SetJSON(ctx, key, int64(n), ttlUnreadCount)
	}
	return int64(n), nil
}

func (r *notificationRepoImpl) CountAll(ctx context.Context) (int64, error) {
	n, err := r.client.Notification.Query().Count(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}
	return int64(n), nil
}

func (r *notificationRepoImpl) CountByStatus(ctx context.Context, status string) (int64, error) {
	n, err := r.client.Notification.Query().
		Where(notification.StatusEQ(notification.Status(status))).
		Count(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}
	return int64(n), nil
}

func (r *notificationRepoImpl) CountSentToday(ctx context.Context) (int64, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	n, err := r.client.Notification.Query().
		Where(notification.CreatedAtGTE(startOfDay)).
		Count(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}
	return int64(n), nil
}

func (r *notificationRepoImpl) CreateTemplate(ctx context.Context, param *entity.CreateTemplateParam) (*entity.NotificationTemplate, error) {
	locale := param.Locale
	if locale == "" {
		locale = "vi"
	}
	channel := param.Channel
	if channel == "" {
		channel = entity.ChannelInApp
	}

	dao, err := r.client.NotificationTemplate.Create().
		SetCode(param.Code).
		SetName(param.Name).
		SetChannel(notificationtemplate.Channel(channel)).
		SetLocale(locale).
		SetTitleTemplate(param.TitleTemplate).
		SetBodyTemplate(param.BodyTemplate).
		SetIsActive(param.IsActive).
		Save(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrTemplateNotFound)
	}

	e := r.mapper.EntTemplateToEntity(dao)
	return &e, nil
}

func (r *notificationRepoImpl) GetTemplateByCode(ctx context.Context, code, channel, locale string) (*entity.NotificationTemplate, error) {
	dao, err := r.client.NotificationTemplate.Query().
		Where(
			notificationtemplate.CodeEQ(code),
			notificationtemplate.ChannelEQ(notificationtemplate.Channel(channel)),
			notificationtemplate.LocaleEQ(locale),
			notificationtemplate.IsActiveEQ(true),
		).
		Only(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrTemplateNotFound)
	}

	e := r.mapper.EntTemplateToEntity(dao)
	return &e, nil
}

func (r *notificationRepoImpl) ListTemplates(ctx context.Context, param *entity.ListTemplatesParam) ([]entity.NotificationTemplate, error) {
	q := r.client.NotificationTemplate.Query()
	if param.Channel != "" {
		q = q.Where(notificationtemplate.ChannelEQ(notificationtemplate.Channel(param.Channel)))
	}
	if param.Locale != "" {
		q = q.Where(notificationtemplate.LocaleEQ(param.Locale))
	}

	daos, err := q.Order(ent.Asc(notificationtemplate.FieldCode)).All(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrTemplateNotFound)
	}
	return r.mapper.EntTemplateListToEntityList(daos), nil
}

func (r *notificationRepoImpl) UpdateTemplate(ctx context.Context, param *entity.UpdateTemplateParam) (*entity.NotificationTemplate, error) {
	builder := r.client.NotificationTemplate.UpdateOneID(param.ID).SetIsActive(param.IsActive)

	if param.Name != "" {
		builder = builder.SetName(param.Name)
	}
	if param.TitleTemplate != "" {
		builder = builder.SetTitleTemplate(param.TitleTemplate)
	}
	if param.BodyTemplate != "" {
		builder = builder.SetBodyTemplate(param.BodyTemplate)
	}

	dao, err := builder.Save(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrTemplateNotFound)
	}

	e := r.mapper.EntTemplateToEntity(dao)
	return &e, nil
}

func (r *notificationRepoImpl) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if err := r.client.NotificationTemplate.DeleteOneID(id).Exec(ctx); err != nil {
		return wrapError(err, cerr.ErrTemplateNotFound)
	}
	return nil
}

func (r *notificationRepoImpl) CountTemplates(ctx context.Context) (int64, error) {
	n, err := r.client.NotificationTemplate.Query().Count(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrTemplateNotFound)
	}
	return int64(n), nil
}

func (r *notificationRepoImpl) GetOrCreatePreference(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreference, error) {
	dao, err := r.client.NotificationPreference.Query().
		Where(notificationpreference.UserIDEQ(userID)).
		Only(ctx)
	if err == nil {
		e := r.mapper.EntPreferenceToEntity(dao)
		return &e, nil
	}
	if !ent.IsNotFound(err) {
		return nil, wrapError(err, cerr.ErrPreferenceNotFound)
	}

	def := entity.DefaultPreference(userID)
	created, cErr := r.client.NotificationPreference.Create().
		SetUserID(userID).
		SetInAppEnabled(def.InAppEnabled).
		SetPushEnabled(def.PushEnabled).
		SetEmailEnabled(def.EmailEnabled).
		SetSmsEnabled(def.SMSEnabled).
		SetMatchEventsEnabled(def.MatchEventsEnabled).
		SetPromotionEnabled(def.PromotionEnabled).
		Save(ctx)
	if cErr != nil {
		if ent.IsConstraintError(cErr) {
			existing, rErr := r.client.NotificationPreference.Query().
				Where(notificationpreference.UserIDEQ(userID)).
				Only(ctx)
			if rErr == nil {
				e := r.mapper.EntPreferenceToEntity(existing)
				return &e, nil
			}
		}
		return nil, wrapError(cErr, cerr.ErrPreferenceNotFound)
	}

	e := r.mapper.EntPreferenceToEntity(created)
	return &e, nil
}

func (r *notificationRepoImpl) UpdatePreference(ctx context.Context, param *entity.UpdatePreferenceParam) (*entity.NotificationPreference, error) {
	if _, err := r.GetOrCreatePreference(ctx, param.UserID); err != nil {
		return nil, err
	}

	dao, err := r.client.NotificationPreference.Update().
		Where(notificationpreference.UserIDEQ(param.UserID)).
		SetInAppEnabled(param.InAppEnabled).
		SetPushEnabled(param.PushEnabled).
		SetEmailEnabled(param.EmailEnabled).
		SetSmsEnabled(param.SMSEnabled).
		SetMatchEventsEnabled(param.MatchEventsEnabled).
		SetPromotionEnabled(param.PromotionEnabled).
		SetQuietHoursStart(param.QuietHoursStart).
		SetQuietHoursEnd(param.QuietHoursEnd).
		Save(ctx)
	if err != nil {
		return nil, wrapError(err, cerr.ErrPreferenceNotFound)
	}
	_ = dao

	return r.GetOrCreatePreference(ctx, param.UserID)
}

func (r *notificationRepoImpl) CountUnreadAll(ctx context.Context) (int64, error) {
	n, err := r.client.Notification.Query().
		Where(notification.IsReadEQ(false)).
		Count(ctx)
	if err != nil {
		return 0, wrapError(err, cerr.ErrNotificationNotFound)
	}
	return int64(n), nil
}