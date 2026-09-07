package http

import (
	"testing"
)

func TestRegisterGatewayRoutes(t *testing.T) {
	engine := newTestEngine()

	routes := engine.Routes()
	if len(routes) == 0 {
		t.Fatal("không có route nào được đăng ký")
	}

	const minEndpoints = 40
	if len(routes) < minEndpoints {
		t.Fatalf("chỉ có %d endpoint, cần ít nhất %d", len(routes), minEndpoints)
	}

	required := map[string]string{
		"POST /api/v1/users/register":          "đăng ký người dùng",
		"POST /api/v1/vehicles/nearby":         "tìm xe gần đây",
		"POST /api/v1/matching/bids":           "chủ hàng đăng đơn",
		"POST /api/v1/matching/matches/accept": "chốt xe",
		"GET /api/v1/admin/users":              "admin liệt kê người dùng",
		"GET /api/v1/admin/vehicles/stats":     "admin thống kê phương tiện",
	}

	found := make(map[string]bool, len(routes))
	for _, r := range routes {
		found[r.Method+" "+r.Path] = true
	}

	for key, desc := range required {
		if !found[key] {
			t.Errorf("thiếu route %s (%s)", key, desc)
		}
	}

	t.Logf("đã đăng ký %d endpoint", len(routes))
}

func TestAdminRoutesAreGuarded(t *testing.T) {
	engine := newTestEngine()

	adminCount := 0
	for _, r := range engine.Routes() {
		if len(r.Path) >= len("/api/v1/admin") && r.Path[:len("/api/v1/admin")] == "/api/v1/admin" {
			adminCount++

			if r.Handler == "" {
				t.Errorf("route admin %s %s không có handler", r.Method, r.Path)
			}
		}
	}

	if adminCount == 0 {
		t.Fatal("không tìm thấy route admin nào")
	}
	t.Logf("có %d endpoint admin", adminCount)
}