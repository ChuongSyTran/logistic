package di

import (
	"context"
	"errors"
	"log"

	"auth_service/internal/biz"
	"auth_service/internal/conf"
	"auth_service/internal/entity"

	"github.com/logistic/pkg/authn"
)

const minAdminPasswordLen = 12

// bootstrapAdmin tạo admin từ biến môi trường nếu chưa có. Idempotent, chạy mỗi
// lần khởi động. API đăng ký chỉ nhận driver/shipper nên đây là lối duy nhất.
func bootstrapAdmin(ctx context.Context, svc biz.AuthService, cfg conf.BootstrapConfig) {
	if !cfg.Enabled() {
		log.Printf("[auth_service] chưa khai AUTH_SERVICE_BOOTSTRAP_ADMIN_EMAIL/PASSWORD — " +
			"hệ thống sẽ không có admin, các API /api/v1/admin/* và việc duyệt xe sẽ không dùng được")
		return
	}

	if len(cfg.AdminPassword) < minAdminPasswordLen {
		log.Printf("[auth_service] BỎ QUA tạo admin: mật khẩu ngắn hơn %d ký tự", minAdminPasswordLen)
		return
	}

	_, err := svc.Register(ctx, entity.UserRegister{
		Email:    cfg.AdminEmail,
		FullName: cfg.AdminFullName,
		Password: cfg.AdminPassword,
		Role:     authn.RoleAdmin,
	})

	switch {
	case err == nil:
		log.Printf("[auth_service] đã tạo tài khoản admin %s", cfg.AdminEmail)
	case errors.Is(err, biz.ErrEmailAlreadyExists):
		log.Printf("[auth_service] tài khoản admin %s đã có, bỏ qua", cfg.AdminEmail)
	default:
		log.Printf("[auth_service] CẢNH BÁO: không tạo được admin %s: %v", cfg.AdminEmail, err)
	}
}
