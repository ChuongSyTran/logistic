package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gateway_service/internal/middleware"

	"github.com/logistic/pkg/authn"
)

func TestForgedRoleHeaderCannotReachAdmin(t *testing.T) {
	engine := newTestEngine()

	for _, path := range []string{
		"/api/v1/admin/users",
		"/api/v1/admin/users/stats",
		"/api/v1/admin/kyc/pending",
		"/api/v1/admin/vehicles",
		"/api/v1/admin/notifications",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(middleware.HeaderUserRole, authn.RoleAdmin)
		req.Header.Set(middleware.HeaderUserID, "3f2b7c10-9a4e-7b21-8f33-5c0d2e6a71b4")

		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("LỖ HỔNG: %s trả %d với header tự khai, mong đợi 401", path, rec.Code)
		}
	}
}

func TestDriverTokenCannotReachAdmin(t *testing.T) {
	engine := newTestEngine()
	token := mintToken(t, authn.RoleDriver)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("token tài xế vào được /admin: nhận %d, mong đợi 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "PERMISSION_DENIED") {
		t.Errorf("thân phản hồi không mang mã PERMISSION_DENIED: %s", rec.Body.String())
	}
}

func TestSecuredRoutesRejectAnonymous(t *testing.T) {
	engine := newTestEngine()

	cases := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/3f2b7c10-9a4e-7b21-8f33-5c0d2e6a71b4"},
		{http.MethodGet, "/api/v1/vehicles"},
		{http.MethodPost, "/api/v1/matching/bids"},
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/users/3f2b7c10-9a4e-7b21-8f33-5c0d2e6a71b4/notifications"},
	}

	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s trả %d khi không có token, mong đợi 401", c.method, c.path, rec.Code)
		}
	}
}

func TestPublicRoutesDoNotRequireToken(t *testing.T) {
	engine := newTestEngine()

	for _, path := range []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/users/register",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code == http.StatusUnauthorized {
			t.Errorf("%s yêu cầu token, nhưng đây là endpoint công khai", path)
		}
	}
}

func TestGarbageTokenRejected(t *testing.T) {
	engine := newTestEngine()

	for _, token := range []string{
		"không-phải-jwt",
		"eyJhbGciOiJub25lIn0.eyJzdWIiOiJhZG1pbiJ9.",
		"",
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("token %q trả %d, mong đợi 401", token, rec.Code)
		}
	}
}

func TestNoUnintentionallyPublicRoute(t *testing.T) {
	engine := newTestEngine()

	publicByDesign := map[string]bool{
		"POST /api/v1/auth/login":          true,
		"POST /api/v1/auth/register":       true,
		"POST /api/v1/auth/refresh":        true,
		"GET /api/v1/auth/google/login":    true,
		"GET /api/v1/auth/google/callback": true,
		"POST /api/v1/users/register":      true,
	}

	for _, r := range engine.Routes() {
		if !strings.HasPrefix(r.Path, "/api/v1") {
			continue
		}
		key := r.Method + " " + r.Path
		if publicByDesign[key] {
			continue
		}

		path := strings.NewReplacer(
			":user_id", "3f2b7c10-9a4e-7b21-8f33-5c0d2e6a71b4",
			":driver_id", "3f2b7c10-9a4e-7b21-8f33-5c0d2e6a71b4",
			":id", "3f2b7c10-9a4e-7b21-8f33-5c0d2e6a71b4",
			":publicID", "abc",
		).Replace(r.Path)

		req := httptest.NewRequest(r.Method, path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s không yêu cầu xác thực (trả %d) — nếu đây là chủ ý, hãy thêm vào publicByDesign",
				key, rec.Code)
		}
	}
}