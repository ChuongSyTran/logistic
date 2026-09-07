package biz_test

import (
	"testing"

	"matching_service/internal/biz"
	"matching_service/internal/entity"
)

var (
	hcm     = entity.Location{Latitude: 10.8231, Longitude: 106.6297}
	danang  = entity.Location{Latitude: 16.0544, Longitude: 108.2022}
	hue     = entity.Location{Latitude: 16.4637, Longitude: 107.5909}
	hanoi   = entity.Location{Latitude: 21.0278, Longitude: 105.8342}
	cantho  = entity.Location{Latitude: 10.0452, Longitude: 105.7469}
	bienhoa = entity.Location{Latitude: 10.9574, Longitude: 106.8426}
)

func truckSaigonToHanoi() *entity.Ask {
	return &entity.Ask{
		CurrentLocation:   hcm,
		Destination:       hanoi,
		AvailableWeightKg: 10000,
		AvailableVolumeM3: 40,
		MinPrice:          3_000_000,
	}
}

func cargo(origin, dest entity.Location, weightKg, volumeM3, maxPrice float64) entity.Bid {
	return entity.Bid{
		Origin:      origin,
		Destination: dest,
		WeightKg:    weightKg,
		VolumeM3:    volumeM3,
		MaxPrice:    maxPrice,
	}
}

func TestThuanTuyenXepTrenNghichTuyen(t *testing.T) {
	truck := truckSaigonToHanoi()

	thuanTuyen := cargo(danang, hanoi, 3000, 12, 5_000_000)
	nghichTuyen := cargo(danang, hcm, 3000, 12, 5_000_000)

	ranked := biz.RankBidsForAsk(truck, []entity.Bid{nghichTuyen, thuanTuyen}, biz.DefaultScoreWeights())
	if len(ranked) == 0 {
		t.Fatal("không có đơn nào qua ngưỡng")
	}

	best := ranked[0]
	if best.Bid.Destination != hanoi {
		t.Errorf("đơn thuận tuyến phải xếp trên, nhận được đơn đi %v", best.Bid.Destination)
	}
	if len(ranked) > 1 && ranked[0].Breakdown.Alignment <= ranked[1].Breakdown.Alignment {
		t.Errorf("điểm thuận tuyến %.3f phải cao hơn nghịch tuyến %.3f",
			ranked[0].Breakdown.Alignment, ranked[1].Breakdown.Alignment)
	}
}

func TestDonGanXeXepTrenDonXa(t *testing.T) {
	truck := truckSaigonToHanoi()

	gan := cargo(bienhoa, hanoi, 2000, 8, 5_000_000)
	xa := cargo(cantho, hanoi, 2000, 8, 5_000_000)

	ranked := biz.RankBidsForAsk(truck, []entity.Bid{xa, gan}, biz.DefaultScoreWeights())
	if len(ranked) == 0 {
		t.Fatal("không có đơn nào qua ngưỡng")
	}
	if ranked[0].DeadheadKm > 40 {
		t.Errorf("đơn đầu bảng có deadhead %.1f km, quá xa", ranked[0].DeadheadKm)
	}
}

func TestDonVuotTaiTrongBiLoai(t *testing.T) {
	truck := truckSaigonToHanoi()

	quaNang := cargo(danang, hanoi, 15000, 10, 5_000_000)
	quaTo := cargo(danang, hanoi, 1000, 80, 5_000_000)
	vua := cargo(danang, hanoi, 5000, 20, 5_000_000)

	ranked := biz.RankBidsForAsk(truck, []entity.Bid{quaNang, quaTo, vua}, biz.DefaultScoreWeights())
	if len(ranked) != 1 {
		t.Fatalf("chỉ đơn vừa sức mới được giữ, nhận %d đơn", len(ranked))
	}
	if ranked[0].Bid.WeightKg != 5000 {
		t.Errorf("giữ nhầm đơn: %.0f kg", ranked[0].Bid.WeightKg)
	}
}

func TestGiaThapHonMucToiThieuBiLoai(t *testing.T) {
	truck := truckSaigonToHanoi()
	reQua := cargo(danang, hanoi, 3000, 12, 2_000_000)

	if ranked := biz.RankBidsForAsk(truck, []entity.Bid{reQua}, biz.DefaultScoreWeights()); len(ranked) != 0 {
		t.Errorf("đơn trả %.0f dưới mức tối thiểu %.0f nhưng vẫn lọt",
			reQua.MaxPrice, truck.MinPrice)
	}
}

func TestDonLapDayNhieuHonDuocUuTien(t *testing.T) {
	truck := truckSaigonToHanoi()

	itHang := cargo(danang, hanoi, 500, 2, 5_000_000)
	nhieuHang := cargo(danang, hanoi, 9000, 35, 5_000_000)

	ranked := biz.RankBidsForAsk(truck, []entity.Bid{itHang, nhieuHang}, biz.DefaultScoreWeights())
	if len(ranked) < 2 {
		t.Fatalf("cần cả hai đơn để so sánh, nhận %d", len(ranked))
	}
	if ranked[0].Bid.WeightKg != 9000 {
		t.Errorf("đơn lấp đầy hơn phải xếp trên, đầu bảng là %.0f kg", ranked[0].Bid.WeightKg)
	}
	if ranked[0].Breakdown.Fill <= ranked[1].Breakdown.Fill {
		t.Errorf("điểm lấp đầy %.3f phải cao hơn %.3f",
			ranked[0].Breakdown.Fill, ranked[1].Breakdown.Fill)
	}
}

func TestDiemNamTrongKhoangHopLe(t *testing.T) {
	truck := truckSaigonToHanoi()
	ranked := biz.RankBidsForAsk(truck,
		[]entity.Bid{cargo(danang, hanoi, 3000, 12, 5_000_000)},
		biz.DefaultScoreWeights())

	if len(ranked) == 0 {
		t.Fatal("không có kết quả")
	}
	s := ranked[0]
	if s.Score < 0 || s.Score > 1 {
		t.Errorf("điểm tổng %.4f nằm ngoài [0,1]", s.Score)
	}
	for name, v := range map[string]float64{
		"deadhead":  s.Breakdown.Deadhead,
		"alignment": s.Breakdown.Alignment,
		"detour":    s.Breakdown.Detour,
		"fill":      s.Breakdown.Fill,
		"price":     s.Breakdown.Price,
	} {
		if v < 0 || v > 1 {
			t.Errorf("điểm %s = %.4f nằm ngoài [0,1]", name, v)
		}
	}
}

func TestTrongSoCongLaiBangMot(t *testing.T) {
	w := biz.DefaultScoreWeights()
	total := w.Deadhead + w.Alignment + w.Detour + w.Fill + w.Price
	if total < 0.999 || total > 1.001 {
		t.Errorf("tổng trọng số = %.4f, cần bằng 1 để điểm nằm trong [0,1]", total)
	}
}

func TestKetQuaSapXepGiamDan(t *testing.T) {
	truck := truckSaigonToHanoi()
	bids := []entity.Bid{
		cargo(cantho, hcm, 1000, 5, 4_000_000),
		cargo(danang, hanoi, 8000, 30, 6_000_000),
		cargo(hue, hanoi, 4000, 15, 5_000_000),
		cargo(bienhoa, danang, 2000, 10, 4_500_000),
	}

	ranked := biz.RankBidsForAsk(truck, bids, biz.DefaultScoreWeights())
	for i := 1; i < len(ranked); i++ {
		if ranked[i-1].Score < ranked[i].Score {
			t.Errorf("thứ tự sai tại vị trí %d: %.4f < %.4f", i, ranked[i-1].Score, ranked[i].Score)
		}
	}
	t.Logf("xếp hạng %d đơn cho xe Sài Gòn → Hà Nội:", len(ranked))
	for i, r := range ranked {
		t.Logf("  %d. điểm=%.4f deadhead=%.1fkm lấp đầy=%.0f%% (gần=%.2f thuận tuyến=%.2f lệch=%.2f đầy=%.2f giá=%.2f)",
			i+1, r.Score, r.DeadheadKm, r.FillRatio*100,
			r.Breakdown.Deadhead, r.Breakdown.Alignment, r.Breakdown.Detour,
			r.Breakdown.Fill, r.Breakdown.Price)
	}
}

func TestXeTimHangChoChuyenVe(t *testing.T) {
	xeDangOHanoi := &entity.Ask{
		CurrentLocation:   hanoi,
		Destination:       hcm,
		AvailableWeightKg: 10000,
		AvailableVolumeM3: 40,
		MinPrice:          3_000_000,
	}

	hangVeNam := cargo(hanoi, hcm, 6000, 25, 6_000_000)
	hangDiNguoc := cargo(hanoi, danang, 6000, 25, 6_000_000)

	ranked := biz.RankBidsForAsk(xeDangOHanoi,
		[]entity.Bid{hangDiNguoc, hangVeNam}, biz.DefaultScoreWeights())
	if len(ranked) == 0 {
		t.Fatal("không có đơn nào qua ngưỡng")
	}
	if ranked[0].Bid.Destination != hcm {
		t.Error("hàng chiều về phải xếp trên hàng dừng giữa đường")
	}
}
