//go:build integration

package persistence

import (
	"context"
	"fmt"
	"os"
	"testing"

	"vehicle_service/ent"
	"vehicle_service/internal/entity"
	"vehicle_service/internal/mapper/generated"

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

const (
	benThanhLat = 10.7721
	benThanhLng = 106.6980
)

func setupVehicleRepo(t *testing.T) (*ent.Client, *cache.Client, *vehicleRepoImpl) {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("IT_PG_HOST", "127.0.0.1"),
		env("IT_PG_PORT", "5432"),
		env("IT_PG_USER", "notif"),
		env("IT_PG_PASSWORD", "notif"),
		env("IT_PG_DB", "vehicle_test"),
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
		Prefix: "it-vehicle-" + uuid.NewString()[:8],
	})
	if err != nil {
		t.Fatalf("kết nối Redis thất bại: %v", err)
	}

	r := NewVehicleRepo(client, redisClient, &generated.AppMapperImpl{}).(*vehicleRepoImpl)

	t.Cleanup(func() {
		_ = redisClient.DeleteByPattern(context.Background(), r.keyGeo()+"*")
		_ = redisClient.Close()
		_ = client.Close()
	})

	return client, redisClient, r
}

func seedOnlineVehicle(t *testing.T, r *vehicleRepoImpl, lat, lng, availWeight, availVolume float64, vType string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	driverID := uuid.New()
	v, err := r.CreateVehicle(ctx, &entity.RegisterVehicleParam{
		DriverID:          driverID,
		LicensePlate:      "51C-" + uuid.NewString()[:8],
		VehicleType:       vType,
		CapacityWeightKg:  10000,
		CapacityVolumeCbm: 40,
	})
	if err != nil {
		t.Fatalf("tạo xe thất bại: %v", err)
	}

	if _, err := r.UpdateVerification(ctx, &entity.VerifyVehicleParam{ID: v.ID}, entity.VerificationVerified); err != nil {
		t.Fatalf("duyệt xe thất bại: %v", err)
	}

	if _, err := r.UpsertAvailability(ctx, &entity.SetAvailabilityParam{
		DriverID:           driverID,
		VehicleID:          v.ID,
		IsOnline:           true,
		AvailableWeightKg:  availWeight,
		AvailableVolumeCbm: availVolume,
		CurrentLat:         lat,
		CurrentLng:         lng,
	}, entity.ComputeZoneID(lat, lng)); err != nil {
		t.Fatalf("bật nhận đơn thất bại: %v", err)
	}

	return v.ID
}

func TestSearchNearbyOrdersByDistance(t *testing.T) {
	_, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	near := seedOnlineVehicle(t, r, benThanhLat+0.009, benThanhLng, 5000, 20, entity.VehicleTypeTruck)
	mid := seedOnlineVehicle(t, r, benThanhLat+0.027, benThanhLng, 5000, 20, entity.VehicleTypeTruck)
	far := seedOnlineVehicle(t, r, benThanhLat+0.45, benThanhLng, 5000, 20, entity.VehicleTypeTruck)

	got, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude:  benThanhLat,
		Longitude: benThanhLng,
		RadiusKm:  10,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("mong đợi 2 xe trong bán kính 10km, nhận %d", len(got))
	}
	if got[0].VehicleID != near {
		t.Errorf("xe gần nhất phải là %s, nhận %s", near, got[0].VehicleID)
	}
	if got[1].VehicleID != mid {
		t.Errorf("xe thứ hai phải là %s, nhận %s", mid, got[1].VehicleID)
	}
	if got[0].DistanceKm > got[1].DistanceKm {
		t.Errorf("kết quả không sắp xếp gần->xa: %.2f rồi %.2f", got[0].DistanceKm, got[1].DistanceKm)
	}
	for _, v := range got {
		if v.VehicleID == far {
			t.Error("xe cách 50km lọt vào kết quả bán kính 10km")
		}
	}
}

func TestSearchNearbyFiltersByCapacity(t *testing.T) {
	_, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	small := seedOnlineVehicle(t, r, benThanhLat+0.001, benThanhLng, 500, 2, entity.VehicleTypeVan)

	big := seedOnlineVehicle(t, r, benThanhLat+0.02, benThanhLng, 8000, 30, entity.VehicleTypeTruck)

	got, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude:    benThanhLat,
		Longitude:   benThanhLng,
		RadiusKm:    10,
		MinWeightKg: 3000,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("mong đợi 1 xe đủ tải, nhận %d", len(got))
	}
	if got[0].VehicleID != big {
		t.Errorf("xe được chọn phải là %s (đủ tải), nhận %s", big, got[0].VehicleID)
	}
	if got[0].VehicleID == small {
		t.Error("xe không đủ tải vẫn lọt vào kết quả")
	}
}

func TestSearchNearbyFiltersByVehicleType(t *testing.T) {
	_, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	truck := seedOnlineVehicle(t, r, benThanhLat+0.005, benThanhLng, 5000, 20, entity.VehicleTypeTruck)
	seedOnlineVehicle(t, r, benThanhLat+0.002, benThanhLng, 5000, 20, entity.VehicleTypeBike)

	got, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude:    benThanhLat,
		Longitude:   benThanhLng,
		RadiusKm:    10,
		VehicleType: entity.VehicleTypeTruck,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}

	if len(got) != 1 || got[0].VehicleID != truck {
		t.Fatalf("lọc theo loại xe sai: %+v", got)
	}
}

func TestGoingOfflineRemovesFromIndex(t *testing.T) {
	client, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	vehicleID := seedOnlineVehicle(t, r, benThanhLat+0.005, benThanhLng, 5000, 20, entity.VehicleTypeTruck)

	before, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude: benThanhLat, Longitude: benThanhLng, RadiusKm: 10,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}
	if len(before) != 1 {
		t.Fatalf("trước khi offline phải tìm thấy 1 xe, nhận %d", len(before))
	}

	v, err := client.Vehicle.Get(ctx, vehicleID)
	if err != nil {
		t.Fatalf("đọc xe thất bại: %v", err)
	}

	if _, err := r.UpsertAvailability(ctx, &entity.SetAvailabilityParam{
		DriverID:   v.DriverID,
		VehicleID:  vehicleID,
		IsOnline:   false,
		CurrentLat: benThanhLat + 0.005,
		CurrentLng: benThanhLng,
	}, ""); err != nil {
		t.Fatalf("tắt nhận đơn thất bại: %v", err)
	}

	after, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude: benThanhLat, Longitude: benThanhLng, RadiusKm: 10,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("tài xế đã offline mà vẫn tìm thấy %d xe", len(after))
	}
}

func TestMaintenanceRemovesFromIndex(t *testing.T) {
	_, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	vehicleID := seedOnlineVehicle(t, r, benThanhLat+0.005, benThanhLng, 5000, 20, entity.VehicleTypeTruck)

	if _, err := r.UpdateVehicleStatus(ctx, vehicleID, entity.VehicleStatusMaintenance); err != nil {
		t.Fatalf("đổi trạng thái bảo dưỡng thất bại: %v", err)
	}

	got, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude: benThanhLat, Longitude: benThanhLng, RadiusKm: 10,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("xe đang bảo dưỡng vẫn xuất hiện trong kết quả (%d xe)", len(got))
	}
}

func TestReportLocationMovesVehicleInIndex(t *testing.T) {
	client, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	vehicleID := seedOnlineVehicle(t, r, benThanhLat+0.4, benThanhLng, 5000, 20, entity.VehicleTypeTruck)

	got, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude: benThanhLat, Longitude: benThanhLng, RadiusKm: 5,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("xe ở xa mà đã tìm thấy: %d", len(got))
	}

	v, err := client.Vehicle.Get(ctx, vehicleID)
	if err != nil {
		t.Fatalf("đọc xe thất bại: %v", err)
	}

	newLat, newLng := benThanhLat+0.005, benThanhLng
	if _, err := r.UpsertLocation(ctx, &entity.ReportLocationParam{
		VehicleID: vehicleID,
		DriverID:  v.DriverID,
		Latitude:  newLat,
		Longitude: newLng,
		SpeedKph:  35,
	}, entity.ComputeZoneID(newLat, newLng)); err != nil {
		t.Fatalf("báo vị trí thất bại: %v", err)
	}

	got, err = r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude: benThanhLat, Longitude: benThanhLng, RadiusKm: 5,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}
	if len(got) != 1 || got[0].VehicleID != vehicleID {
		t.Fatalf("sau khi ping GPS phải tìm thấy xe %s, nhận %+v", vehicleID, got)
	}
	if got[0].DistanceKm > 1.5 {
		t.Errorf("khoảng cách %.2f km, mong đợi dưới 1.5 km", got[0].DistanceKm)
	}
}

func TestUnverifiedVehicleIsExcluded(t *testing.T) {
	_, _, r := setupVehicleRepo(t)
	ctx := context.Background()

	driverID := uuid.New()
	v, err := r.CreateVehicle(ctx, &entity.RegisterVehicleParam{
		DriverID:          driverID,
		LicensePlate:      "51C-" + uuid.NewString()[:8],
		VehicleType:       entity.VehicleTypeTruck,
		CapacityWeightKg:  10000,
		CapacityVolumeCbm: 40,
	})
	if err != nil {
		t.Fatalf("tạo xe thất bại: %v", err)
	}

	if _, err := r.UpsertAvailability(ctx, &entity.SetAvailabilityParam{
		DriverID:           driverID,
		VehicleID:          v.ID,
		IsOnline:           true,
		AvailableWeightKg:  5000,
		AvailableVolumeCbm: 20,
		CurrentLat:         benThanhLat + 0.002,
		CurrentLng:         benThanhLng,
	}, entity.ComputeZoneID(benThanhLat, benThanhLng)); err != nil {
		t.Fatalf("bật nhận đơn thất bại: %v", err)
	}

	got, err := r.SearchNearby(ctx, &entity.SearchNearbyParam{
		Latitude: benThanhLat, Longitude: benThanhLng, RadiusKm: 10,
	})
	if err != nil {
		t.Fatalf("SearchNearby lỗi: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("xe chưa duyệt giấy tờ vẫn được đưa vào kết quả: %+v", got)
	}
}