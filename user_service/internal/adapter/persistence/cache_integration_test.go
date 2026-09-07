//go:build integration

package persistence

import (
	"context"
	"fmt"
	"os"
	"testing"

	"user_service/ent"
	"user_service/internal/entity"
	"user_service/internal/mapper/generated"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/logistic/pkg/cache"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func setupUserRepo(t *testing.T) (*ent.Client, *cache.Client, *userRepoImpl) {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("IT_PG_HOST", "127.0.0.1"),
		env("IT_PG_PORT", "5432"),
		env("IT_PG_USER", "notif"),
		env("IT_PG_PASSWORD", "notif"),
		env("IT_PG_DB", "user_test"),
	)

	client, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("mở Postgres thất bại: %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("tạo schema thất bại: %v", err)
	}

	redisClient, err := cache.New(cache.Config{
		Host:   env("IT_REDIS_HOST", "127.0.0.1"),
		Port:   env("IT_REDIS_PORT", "6379"),
		DB:     9,
		Prefix: "it-user-" + uuid.NewString()[:8],
	})
	if err != nil {
		t.Fatalf("kết nối Redis thất bại: %v", err)
	}

	r := NewUserRepo(client, redisClient, &generated.AppMapperImpl{}).(*userRepoImpl)

	t.Cleanup(func() {
		_ = redisClient.Close()
		_ = client.Close()
	})

	return client, redisClient, r
}

func createTestUser(t *testing.T, r *userRepoImpl, role string) *entity.User {
	t.Helper()

	u, err := r.CreateUser(context.Background(), &entity.User{
		Phone:        "09" + uuid.NewString()[:8],
		Email:        uuid.NewString()[:8] + "@example.com",
		FullName:     "Người Dùng Test",
		PasswordHash: "hash",
		Role:         role,
	})
	if err != nil {
		t.Fatalf("tạo user thất bại: %v", err)
	}
	return u
}

func TestCacheIsPopulatedOnRead(t *testing.T) {
	client, _, r := setupUserRepo(t)
	ctx := context.Background()

	u := createTestUser(t, r, entity.RoleDriver)

	if _, err := r.GetUserByID(ctx, u.ID); err != nil {
		t.Fatalf("đọc lần 1 thất bại: %v", err)
	}

	if err := client.User.UpdateOneID(u.ID).SetFullName("Tên Đã Đổi Ngầm").Exec(ctx); err != nil {
		t.Fatalf("sửa DB thất bại: %v", err)
	}

	got, err := r.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("đọc lần 2 thất bại: %v", err)
	}
	if got.FullName != "Người Dùng Test" {
		t.Errorf("lần đọc thứ hai trả %q — đáng lẽ phải lấy từ cache", got.FullName)
	}
}

func TestWriteInvalidatesCache(t *testing.T) {
	_, _, r := setupUserRepo(t)
	ctx := context.Background()

	u := createTestUser(t, r, entity.RoleShipper)

	if _, err := r.GetUserByID(ctx, u.ID); err != nil {
		t.Fatalf("đọc để nạp cache thất bại: %v", err)
	}

	if _, err := r.UpdateUser(ctx, &entity.UpdateUserParam{
		ID:       u.ID,
		FullName: "Tên Mới Chính Thức",
	}); err != nil {
		t.Fatalf("cập nhật thất bại: %v", err)
	}

	got, err := r.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("đọc sau khi ghi thất bại: %v", err)
	}
	if got.FullName != "Tên Mới Chính Thức" {
		t.Fatalf("đọc sau khi ghi vẫn ra %q — cache không được invalidate", got.FullName)
	}
}

func TestStatusChangeInvalidatesCache(t *testing.T) {
	_, _, r := setupUserRepo(t)
	ctx := context.Background()

	u := createTestUser(t, r, entity.RoleDriver)

	if _, err := r.GetUserByID(ctx, u.ID); err != nil {
		t.Fatalf("nạp cache thất bại: %v", err)
	}

	if _, err := r.UpdateUserStatus(ctx, u.ID, entity.StatusBanned, "vi phạm điều khoản"); err != nil {
		t.Fatalf("khoá tài khoản thất bại: %v", err)
	}

	got, err := r.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("đọc sau khi khoá thất bại: %v", err)
	}
	if got.Status != entity.StatusBanned {
		t.Fatalf("sau khi khoá vẫn đọc ra status %q — cache giữ dữ liệu cũ nguy hiểm", got.Status)
	}
}

func TestPhoneCacheIsInvalidatedToo(t *testing.T) {
	_, _, r := setupUserRepo(t)
	ctx := context.Background()

	u := createTestUser(t, r, entity.RoleDriver)

	if _, err := r.GetUserByPhone(ctx, u.Phone); err != nil {
		t.Fatalf("nạp cache theo phone thất bại: %v", err)
	}

	if _, err := r.UpdateUser(ctx, &entity.UpdateUserParam{ID: u.ID, FullName: "Tên Sau Sửa"}); err != nil {
		t.Fatalf("cập nhật thất bại: %v", err)
	}

	got, err := r.GetUserByPhone(ctx, u.Phone)
	if err != nil {
		t.Fatalf("đọc theo phone thất bại: %v", err)
	}
	if got.FullName != "Tên Sau Sửa" {
		t.Errorf("cache theo phone không được invalidate: %q", got.FullName)
	}
}

func TestRepoWorksWithoutRedis(t *testing.T) {
	client, _, _ := setupUserRepo(t)
	ctx := context.Background()

	noCacheRepo := NewUserRepo(client, nil, &generated.AppMapperImpl{}).(*userRepoImpl)

	u, err := noCacheRepo.CreateUser(ctx, &entity.User{
		Phone:        "09" + uuid.NewString()[:8],
		Email:        uuid.NewString()[:8] + "@example.com",
		FullName:     "Không Cache",
		PasswordHash: "hash",
		Role:         entity.RoleDriver,
	})
	if err != nil {
		t.Fatalf("tạo user khi không có Redis thất bại: %v", err)
	}

	got, err := noCacheRepo.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("đọc khi không có Redis thất bại: %v", err)
	}
	if got.FullName != "Không Cache" {
		t.Errorf("dữ liệu sai khi không có Redis: %+v", got)
	}

	if _, err := noCacheRepo.UpdateUser(ctx, &entity.UpdateUserParam{ID: u.ID, FullName: "Vẫn Chạy"}); err != nil {
		t.Fatalf("ghi khi không có Redis thất bại: %v", err)
	}
}

func TestDriverProfileNullableUniqueColumns(t *testing.T) {
	_, _, r := setupUserRepo(t)
	ctx := context.Background()

	first := createTestUser(t, r, entity.RoleDriver)
	second := createTestUser(t, r, entity.RoleDriver)

	if _, err := r.CreateDriverProfile(ctx, first.ID, &entity.DriverProfile{}); err != nil {
		t.Fatalf("tạo hồ sơ tài xế thứ nhất thất bại: %v", err)
	}
	if _, err := r.CreateDriverProfile(ctx, second.ID, &entity.DriverProfile{}); err != nil {
		t.Fatalf("tạo hồ sơ tài xế THỨ HAI thất bại (unique index nổ vì chuỗi rỗng): %v", err)
	}

	if _, err := r.UpdateDriverProfile(ctx, &entity.UpdateDriverProfileParam{
		UserID: first.ID, LicenseNumber: "B2-999999",
	}); err != nil {
		t.Fatalf("gán bằng lái cho tài xế thứ nhất thất bại: %v", err)
	}
	if _, err := r.UpdateDriverProfile(ctx, &entity.UpdateDriverProfileParam{
		UserID: second.ID, LicenseNumber: "B2-999999",
	}); err == nil {
		t.Error("hai tài xế dùng chung một số bằng lái mà không bị chặn")
	}
}
