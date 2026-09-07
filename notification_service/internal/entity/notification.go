package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	RoleDriver  = "driver"
	RoleShipper = "shipper"
	RoleAdmin   = "admin"
)

const (
	ChannelInApp = "in_app"
	ChannelPush  = "push"
	ChannelEmail = "email"
	ChannelSMS   = "sms"
)

const (
	LocaleVI = "vi"
	LocaleEN = "en"
)

const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
	StatusRead    = "read"
)

const (
	TypeDriverCandidate = "driver_candidate"
	TypeMatchFound      = "match_found"
	TypeOfferReceived   = "offer_received"
	TypeOfferRejected   = "offer_rejected"
	TypeCargoSuggested  = "cargo_suggested"
	TypeSystem          = "system"
)

const (
	TplDriverCandidate  = "DRIVER_CANDIDATE"
	TplMatchFoundShip   = "MATCH_FOUND_SHIPPER"
	TplMatchFoundDriver = "MATCH_FOUND_DRIVER"
	TplOfferReceived    = "OFFER_RECEIVED"
	TplOfferRejected    = "OFFER_REJECTED"
	TplCargoSuggested   = "CARGO_SUGGESTED"
)

const (
	RefTypeBid   = "bid"
	RefTypeAsk   = "ask"
	RefTypeMatch = "match"
)

func IsValidChannel(c string) bool {
	switch c {
	case ChannelInApp, ChannelPush, ChannelEmail, ChannelSMS:
		return true
	}
	return false
}

func IsValidRole(r string) bool {
	return r == RoleDriver || r == RoleShipper || r == RoleAdmin
}

func IsValidStatus(s string) bool {
	switch s {
	case StatusPending, StatusSent, StatusFailed, StatusRead:
		return true
	}
	return false
}

type Notification struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	RecipientRole string    `json:"recipient_role"`
	Type          string    `json:"type"`
	Channel       string    `json:"channel"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	Data          string    `json:"data"`
	RefType       string    `json:"ref_type"`
	RefID         string    `json:"ref_id"`
	IsRead        bool      `json:"is_read"`
	Status        string    `json:"status"`
	ErrorMessage  string    `json:"error_message"`
	ReadAt        time.Time `json:"read_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type NotificationTemplate struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Channel       string    `json:"channel"`
	Locale        string    `json:"locale"`
	TitleTemplate string    `json:"title_template"`
	BodyTemplate  string    `json:"body_template"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (t *NotificationTemplate) Render(vars map[string]string) (string, string) {
	return replacePlaceholders(t.TitleTemplate, vars), replacePlaceholders(t.BodyTemplate, vars)
}

func replacePlaceholders(tpl string, vars map[string]string) string {
	if tpl == "" || len(vars) == 0 {
		return tpl
	}
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

type NotificationPreference struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	InAppEnabled       bool      `json:"in_app_enabled"`
	PushEnabled        bool      `json:"push_enabled"`
	EmailEnabled       bool      `json:"email_enabled"`
	SMSEnabled         bool      `json:"sms_enabled"`
	MatchEventsEnabled bool      `json:"match_events_enabled"`
	PromotionEnabled   bool      `json:"promotion_enabled"`
	QuietHoursStart    string    `json:"quiet_hours_start"`
	QuietHoursEnd      string    `json:"quiet_hours_end"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func DefaultPreference(userID uuid.UUID) NotificationPreference {
	return NotificationPreference{
		UserID:             userID,
		InAppEnabled:       true,
		PushEnabled:        true,
		EmailEnabled:       false,
		SMSEnabled:         false,
		MatchEventsEnabled: true,
		PromotionEnabled:   true,
	}
}

func (p *NotificationPreference) AllowsChannel(channel string) bool {
	switch channel {
	case ChannelInApp:
		return p.InAppEnabled
	case ChannelPush:
		return p.PushEnabled
	case ChannelEmail:
		return p.EmailEnabled
	case ChannelSMS:
		return p.SMSEnabled
	default:
		return false
	}
}

func (p *NotificationPreference) AllowsType(notiType string) bool {
	switch notiType {
	case TypeDriverCandidate, TypeMatchFound, TypeOfferReceived, TypeOfferRejected, TypeCargoSuggested:
		return p.MatchEventsEnabled
	case TypeSystem:
		return true
	default:
		return p.PromotionEnabled
	}
}

func (p *NotificationPreference) IsQuietHour(t time.Time) bool {
	if p.QuietHoursStart == "" || p.QuietHoursEnd == "" {
		return false
	}

	start, ok1 := parseHourMinute(p.QuietHoursStart)
	end, ok2 := parseHourMinute(p.QuietHoursEnd)
	if !ok1 || !ok2 {
		return false
	}

	now := t.Hour()*60 + t.Minute()
	if start <= end {
		return now >= start && now < end
	}
	return now >= start || now < end
}

func parseHourMinute(s string) (int, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, err1 := atoi(parts[0])
	m, err2 := atoi(parts[1])
	if !err1 || !err2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func atoi(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

func NormalizePaging(page, pageSize int) (int, int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

func BuildPagination(page, pageSize int, total int64) Pagination {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return Pagination{Page: page, PageSize: pageSize, TotalItems: total, TotalPages: totalPages}
}

type CreateNotificationParam struct {
	UserID        uuid.UUID
	RecipientRole string
	Type          string
	Channel       string
	Title         string
	Body          string
	Data          string
	RefType       string
	RefID         string
}

type ListNotificationsParam struct {
	UserID     uuid.UUID
	Type       string
	UnreadOnly bool
	Page       int
	PageSize   int
}

type ListNotificationsResult struct {
	Notifications []Notification
	Pagination    Pagination
	UnreadCount   int64
}

type AdminListNotificationsParam struct {
	UserID   uuid.UUID
	Type     string
	Status   string
	Page     int
	PageSize int
}

type UpdatePreferenceParam struct {
	UserID             uuid.UUID
	InAppEnabled       bool
	PushEnabled        bool
	EmailEnabled       bool
	SMSEnabled         bool
	MatchEventsEnabled bool
	PromotionEnabled   bool
	QuietHoursStart    string
	QuietHoursEnd      string
}

type ListTemplatesParam struct {
	Channel string
	Locale  string
}

type CreateTemplateParam struct {
	Code          string
	Name          string
	Channel       string
	Locale        string
	TitleTemplate string
	BodyTemplate  string
	IsActive      bool
}

type UpdateTemplateParam struct {
	ID            uuid.UUID
	Name          string
	TitleTemplate string
	BodyTemplate  string
	IsActive      bool
}

type SendNotificationParam struct {
	UserIDs       []uuid.UUID
	BroadcastRole string
	Type          string
	Channel       string
	Title         string
	Body          string
	Data          string
}

type SendNotificationResult struct {
	SentCount int64
	Message   string
}

type NotificationStats struct {
	TotalNotifications  int64
	UnreadNotifications int64
	SentToday           int64
	FailedNotifications int64
	TotalTemplates      int64
}
