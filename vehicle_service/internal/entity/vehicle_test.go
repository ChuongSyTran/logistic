package entity

import (
	"math"
	"testing"
)

func TestIsValidCoordinate(t *testing.T) {
	cases := []struct {
		name     string
		lat, lng float64
		want     bool
	}{
		{"TP.HCM", 10.7769, 106.7009, true},
		{"Hà Nội", 21.0278, 105.8342, true},
		{"cực nam", -90, 0, true},
		{"cực bắc", 90, 180, true},
		{"đảo Null (0,0)", 0, 0, false},
		{"vĩ độ vượt ngưỡng", 91, 100, false},
		{"vĩ độ âm vượt ngưỡng", -91, 100, false},
		{"kinh độ vượt ngưỡng", 10, 181, false},
		{"NaN", math.NaN(), 100, false},
		{"vô cực", math.Inf(1), 100, false},
	}

	for _, tc := range cases {
		if got := IsValidCoordinate(tc.lat, tc.lng); got != tc.want {
			t.Errorf("%s (%v,%v) -> %v, mong đợi %v", tc.name, tc.lat, tc.lng, got, tc.want)
		}
	}
}

func TestComputeZoneIDIsStable(t *testing.T) {
	lat, lng := 10.7769, 106.7009

	first := ComputeZoneID(lat, lng)
	if first == "" {
		t.Fatal("toạ độ hợp lệ mà không tính được zone")
	}
	if second := ComputeZoneID(lat, lng); second != first {
		t.Errorf("không ổn định: %q rồi %q", first, second)
	}

	if near := ComputeZoneID(lat+ZoneSize/10, lng+ZoneSize/10); near != first {
		t.Errorf("điểm rất gần lại ra zone khác: %q vs %q", near, first)
	}

	if far := ComputeZoneID(lat+ZoneSize*2, lng); far == first {
		t.Errorf("điểm cách hơn 2 ô vẫn cùng zone %q", first)
	}
}

func TestComputeZoneIDRejectsInvalid(t *testing.T) {
	if z := ComputeZoneID(0, 0); z != "" {
		t.Errorf("toạ độ (0,0) phải trả zone rỗng, nhận %q", z)
	}
	if z := ComputeZoneID(999, 999); z != "" {
		t.Errorf("toạ độ ngoài ngưỡng phải trả zone rỗng, nhận %q", z)
	}
}

func TestSearchNearbyParamNormalize(t *testing.T) {
	cases := []struct {
		name       string
		in         SearchNearbyParam
		wantRadius float64
		wantLimit  int
	}{
		{"để trống -> mặc định", SearchNearbyParam{}, DefaultSearchRadiusKm, DefaultSearchLimit},
		{"âm -> mặc định", SearchNearbyParam{RadiusKm: -5, Limit: -1}, DefaultSearchRadiusKm, DefaultSearchLimit},
		{"vượt trần -> bị kẹp", SearchNearbyParam{RadiusKm: 100000, Limit: 99999}, MaxSearchRadiusKm, MaxSearchLimit},
		{"hợp lệ -> giữ nguyên", SearchNearbyParam{RadiusKm: 12.5, Limit: 30}, 12.5, 30},
	}

	for _, tc := range cases {
		p := tc.in
		p.Normalize()
		if p.RadiusKm != tc.wantRadius {
			t.Errorf("%s: radius = %v, mong đợi %v", tc.name, p.RadiusKm, tc.wantRadius)
		}
		if p.Limit != tc.wantLimit {
			t.Errorf("%s: limit = %d, mong đợi %d", tc.name, p.Limit, tc.wantLimit)
		}
	}
}

func TestNormalizePagingClampsPageSize(t *testing.T) {
	cases := []struct {
		page, size                     int
		wantPage, wantSize, wantOffset int
	}{
		{0, 0, 1, 20, 0},
		{-3, -1, 1, 20, 0},
		{2, 50, 2, 50, 50},
		{3, 100000, 3, 100, 200},
	}

	for _, tc := range cases {
		page, size, offset := NormalizePaging(tc.page, tc.size)
		if page != tc.wantPage || size != tc.wantSize || offset != tc.wantOffset {
			t.Errorf("NormalizePaging(%d,%d) = (%d,%d,%d), mong đợi (%d,%d,%d)",
				tc.page, tc.size, page, size, offset, tc.wantPage, tc.wantSize, tc.wantOffset)
		}
	}
}

func TestBuildPaginationRoundsUp(t *testing.T) {
	p := BuildPagination(1, 20, 41)
	if p.TotalPages != 3 {
		t.Errorf("41 bản ghi / 20 mỗi trang = %d trang, mong đợi 3", p.TotalPages)
	}

	if empty := BuildPagination(1, 20, 0); empty.TotalPages != 0 {
		t.Errorf("0 bản ghi -> %d trang, mong đợi 0", empty.TotalPages)
	}
}

func TestValidators(t *testing.T) {
	if !IsValidVehicleType(VehicleTypeContainer) {
		t.Error("container phải là loại xe hợp lệ")
	}
	if IsValidVehicleType("xe_đạp_điện") {
		t.Error("loại xe lạ phải bị từ chối")
	}
	if !IsValidVehicleStatus(VehicleStatusMaintenance) {
		t.Error("maintenance phải là trạng thái hợp lệ")
	}
	if IsValidDocumentType("giấy_khai_sinh") {
		t.Error("loại giấy tờ lạ phải bị từ chối")
	}
}