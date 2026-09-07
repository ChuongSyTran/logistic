package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsQuietHourOvernight(t *testing.T) {
	p := &NotificationPreference{QuietHoursStart: "22:00", QuietHoursEnd: "07:00"}

	cases := []struct {
		hour, minute int
		want         bool
	}{
		{23, 30, true},
		{2, 0, true},
		{6, 59, true},
		{7, 0, false},
		{12, 0, false},
		{21, 59, false},
		{22, 0, true},
	}

	for _, tc := range cases {
		at := time.Date(2026, 8, 20, tc.hour, tc.minute, 0, 0, time.UTC)
		if got := p.IsQuietHour(at); got != tc.want {
			t.Errorf("%02d:%02d -> %v, mong đợi %v", tc.hour, tc.minute, got, tc.want)
		}
	}
}

func TestIsQuietHourSameDay(t *testing.T) {
	p := &NotificationPreference{QuietHoursStart: "13:00", QuietHoursEnd: "15:00"}

	cases := []struct {
		hour int
		want bool
	}{
		{12, false},
		{13, true},
		{14, true},
		{15, false},
		{23, false},
		{3, false},
	}

	for _, tc := range cases {
		at := time.Date(2026, 8, 20, tc.hour, 0, 0, 0, time.UTC)
		if got := p.IsQuietHour(at); got != tc.want {
			t.Errorf("%02d:00 -> %v, mong đợi %v", tc.hour, got, tc.want)
		}
	}
}

func TestIsQuietHourInvalidInput(t *testing.T) {
	at := time.Date(2026, 8, 20, 23, 0, 0, 0, time.UTC)

	for _, p := range []*NotificationPreference{
		{},
		{QuietHoursStart: "22:00"},
		{QuietHoursStart: "hai mươi hai", QuietHoursEnd: "7"},
		{QuietHoursStart: "25:00", QuietHoursEnd: "07:00"},
		{QuietHoursStart: "22:70", QuietHoursEnd: "07:00"},
	} {
		if p.IsQuietHour(at) {
			t.Errorf("cấu hình hỏng %+v không được coi là giờ yên lặng", p)
		}
	}
}

func TestAllowsChannel(t *testing.T) {
	p := &NotificationPreference{InAppEnabled: true, PushEnabled: false, EmailEnabled: true}

	if !p.AllowsChannel(ChannelInApp) {
		t.Error("in_app đang bật mà bị chặn")
	}
	if p.AllowsChannel(ChannelPush) {
		t.Error("push đang tắt mà vẫn cho qua")
	}
	if p.AllowsChannel("kênh_lạ") {
		t.Error("kênh không xác định phải bị từ chối")
	}
}

func TestAllowsTypeSystemAlwaysPasses(t *testing.T) {
	p := &NotificationPreference{MatchEventsEnabled: false, PromotionEnabled: false}

	if !p.AllowsType(TypeSystem) {
		t.Error("thông báo hệ thống phải luôn được gửi")
	}
	if p.AllowsType(TypeMatchFound) {
		t.Error("đã tắt match_events mà match_found vẫn qua")
	}
	if p.AllowsType(TypeDriverCandidate) {
		t.Error("đã tắt match_events mà driver_candidate vẫn qua")
	}
}

func TestDefaultPreferenceEnablesMatchEvents(t *testing.T) {
	uid := uuid.New()
	p := DefaultPreference(uid)

	if p.UserID != uid {
		t.Errorf("UserID = %s, mong đợi %s", p.UserID, uid)
	}

	if !p.MatchEventsEnabled || !p.PushEnabled || !p.InAppEnabled {
		t.Errorf("mặc định phải bật thông báo ghép đơn: %+v", p)
	}

	if p.EmailEnabled || p.SMSEnabled {
		t.Errorf("email/sms phải tắt mặc định: %+v", p)
	}
}

func TestTemplateRender(t *testing.T) {
	tpl := &NotificationTemplate{
		TitleTemplate: "Chào {{name}}",
		BodyTemplate:  "Đơn {{bid_id}} cách bạn {{distance}} km. Chúc {{name}} may mắn.",
	}

	title, body := tpl.Render(map[string]string{
		"name":     "Anh Tuấn",
		"bid_id":   "B-123",
		"distance": "2.4",
	})

	if title != "Chào Anh Tuấn" {
		t.Errorf("title = %q", title)
	}
	want := "Đơn B-123 cách bạn 2.4 km. Chúc Anh Tuấn may mắn."
	if body != want {
		t.Errorf("body = %q, mong đợi %q", body, want)
	}
}

func TestTemplateRenderLeavesUnknownPlaceholder(t *testing.T) {
	tpl := &NotificationTemplate{TitleTemplate: "Xin chào {{missing}}"}

	title, _ := tpl.Render(map[string]string{"other": "x"})
	if title != "Xin chào {{missing}}" {
		t.Errorf("title = %q, mong đợi giữ nguyên placeholder", title)
	}
}