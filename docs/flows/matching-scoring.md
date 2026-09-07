# Chấm điểm ghép chuyến

Khi một đơn hàng hoặc một chuyến xe được đăng, hệ thống cần chọn ra vài ứng viên
phù hợp nhất thay vì gửi thông báo cho tất cả. Tài liệu này mô tả cách tính điểm
hiện đang dùng, các tham số có thể chỉnh, và những giới hạn đã biết.

---

## Bài toán

Chi phí lớn nhất của vận tải đường dài là quãng đường xe chạy mà không mang hàng.
Tài liệu ngành gọi đó là *deadhead* hoặc *empty miles*. Các báo cáo công khai ước
lượng con số này vào khoảng 15–30% tổng quãng đường của ngành vận tải Mỹ, và một
số nền tảng cho biết thuật toán ghép chuyến giúp giảm khoảng 20–35%.

Hai tình huống sinh ra quãng đường rỗng:

- **Chiều đi chưa đầy.** Xe rời kho với 60% tải, phần còn trống đi cùng suốt tuyến.
- **Chiều về rỗng.** Xe giao hàng xong ở Hà Nội rồi quay về Sài Gòn không hàng.

Cách tiếp cận chung là chấm điểm từng cặp (đơn hàng, chuyến xe) rồi xếp hạng,
thay vì lọc nhị phân đạt/không đạt.

## Năm thành phần điểm

Mỗi thành phần được chuẩn hoá về khoảng [0, 1] rồi nhân trọng số. Tổng trọng số
bằng 1 nên điểm cuối cũng nằm trong [0, 1].

| Thành phần | Trọng số | Trả lời câu hỏi |
|---|---|---|
| `deadhead` | 0.30 | Điểm lấy hàng lệch bao xa khỏi tuyến xe đang đi? |
| `alignment` | 0.25 | Hàng có đi cùng hướng với xe không? |
| `detour` | 0.20 | Nhận đơn này làm tổng quãng đường tăng thêm bao nhiêu? |
| `fill` | 0.15 | Đơn lấp được bao nhiêu phần chỗ trống? |
| `price` | 0.10 | Giá chủ hàng trả cao hơn mức tối thiểu của tài xế bao nhiêu? |

Trọng số nằm ở `DefaultScoreWeights()` trong `matching_service/internal/biz/scoring.go`
và có thể thay đổi mà không đụng tới phần còn lại.

### deadhead — khoảng cách lệch tuyến

Điểm này **không** đo khoảng cách thẳng từ xe tới điểm lấy hàng. Nó đo khoảng
cách từ điểm lấy hàng tới **đường đi của xe**.

Phân biệt này quan trọng với tuyến dài. Xe chạy Sài Gòn → Hà Nội, đơn hàng ở Đà
Nẵng: khoảng cách thẳng từ xe tới Đà Nẵng là hơn 600 km, nhưng xe dù sao cũng đi
qua đó. Nếu tính theo khoảng cách thẳng, đơn này bị loại oan.

```
offRoute = khoảng cách từ điểm lấy hàng tới đoạn thẳng [vị trí xe → đích xe]
progress = vị trí hình chiếu trên tuyến, tính theo tỉ lệ
```

`progress < 0` nghĩa là điểm lấy hàng nằm **phía sau** xe — tài xế phải quay đầu.
Trường hợp đó điểm bị nhân với hệ số phạt `behindTruckPenalty` (0.4).

Ngân sách lệch tuyến co giãn theo độ dài chuyến:

```
budget = clamp(quãng đường chuyến × 0.25, 30 km, 250 km)
```

Chuyến nội thành 40 km chỉ chấp nhận lệch 30 km; chuyến Bắc–Nam chấp nhận tới
250 km. Tỉ lệ 25% cao hơn thông lệ quốc tế vì địa hình Việt Nam kéo dài và đường
bộ men theo bờ biển, nên đường chim bay chênh khá nhiều so với đường thật.

### alignment — thuận tuyến hay nghịch tuyến

Cosine giữa vector hướng xe và vector hướng hàng, ánh xạ từ [-1, 1] về [0, 1]:

```
alignment = (cos(hướng_xe, hướng_hàng) + 1) / 2
```

| Tình huống | cos | điểm |
|---|---|---|
| Hàng đi cùng chiều xe | ≈ 1 | ≈ 1.0 |
| Hàng đi vuông góc | ≈ 0 | ≈ 0.5 |
| Hàng đi ngược chiều | ≈ -1 | ≈ 0.0 |

Đây là thành phần xử lý tình huống "xe từ Nam ra Bắc ghé miền Trung lấy thêm
hàng": hàng ở Đà Nẵng đi Hà Nội có cùng hướng với xe, nên được cộng điểm; hàng ở
Đà Nẵng đi ngược vào Sài Gòn thì không.

### detour — quãng đường phát sinh

```
detour = (xe→lấy hàng) + (lấy hàng→giao hàng) + (giao hàng→đích xe) − (xe→đích xe)
```

Chuẩn hoá theo tỉ lệ so với quãng đường gốc, ngưỡng chấp nhận `maxDetourRatio`
là 0.35 — vượt 35% thì thành phần này về 0.

### fill — mức lấp đầy

```
fill = max(khối lượng hàng / tải trọng còn trống, thể tích hàng / thể tích còn trống)
```

Lấy `max` chứ không phải trung bình, vì ràng buộc chặt hơn quyết định. Hàng nhẹ
nhưng cồng kềnh vẫn chiếm hết thùng.

Đơn lấp đầy nhiều được ưu tiên: cùng một chuyến đi, chở 8 tấn có lợi hơn chở 1 tấn.

### price — biên giá

```
price = (giá tối đa chủ hàng trả − giá tối thiểu tài xế nhận) / giá tối đa
```

Trọng số thấp nhất (0.10) vì giá còn được thương lượng ở bước báo giá phía sau.

## Lọc cứng trước khi chấm điểm

Một số điều kiện không đánh đổi được, nên loại thẳng thay vì trừ điểm:

| Điều kiện | Lý do |
|---|---|
| Hàng nặng hơn tải trọng còn trống | Không chở được |
| Hàng to hơn thể tích còn trống | Không chở được |
| Giá tối đa < giá tối thiểu | Không có vùng thương lượng |
| Tổng điểm < `minAcceptableScore` (0.35) | Gửi thông báo chỉ gây nhiễu |

Sau khi xếp hạng, hệ thống lấy tối đa 20 ứng viên đầu (`maxCandidatesPerBid`,
`maxSuggestionsPerAsk`) rồi mới phát sự kiện thông báo.

## Hai tầng lọc

| Tầng | Nơi chạy | Điều kiện |
|---|---|---|
| Thô | SQL, `matching_service/internal/repo` | Trạng thái, sức chứa, giá, bán kính 150 km |
| Tinh | Go, `matching_service/internal/biz` | Năm thành phần điểm ở trên |

Tầng SQL giữ số dòng phải nạp vào bộ nhớ ở mức hợp lý; tầng Go tính hình học mà
SQL thuần khó biểu diễn.

Bán kính lọc thô (150 km) cố tình rộng hơn ngân sách lệch tuyến, để không loại
nhầm ứng viên trước khi kịp chấm điểm.

## Ví dụ

Xe chạy Sài Gòn → Hà Nội, còn trống 10 tấn / 40 m³, giá tối thiểu 3.000.000 đ:

```
1. điểm=0.7438 lệch=24.3km  đầy=25%  (gần=0.90 thuận=0.97 vòng=0.79 đầy=0.25 giá=0.33)
2. điểm=0.6474 lệch=149.0km đầy=40%  (gần=0.40 thuận=0.98 vòng=0.90 đầy=0.40 giá=0.40)
3. điểm=0.6216 lệch=211.1km đầy=80%  (gần=0.16 thuận=0.97 vòng=0.81 đầy=0.80 giá=0.50)
4. điểm=0.3726 lệch=129.6km đầy=12%  (gần=0.19 thuận=0.80 vòng=0.35 đầy=0.12 giá=0.25)
```

Đơn ở Biên Hòa xếp trên dù chỉ lấp 25%, vì gần như không lệch tuyến. Đơn Đà Nẵng
lấp tới 80% nhưng lệch 211 km nên xuống hạng ba — đánh đổi này do bộ trọng số
quyết định và có thể chỉnh lại nếu số liệu vận hành cho thấy nên ưu tiên khác.

## Giới hạn đã biết

**Khoảng cách là đường chim bay.** Công thức haversine không biết đường thật đi
đâu. Với địa hình Việt Nam, các thành phố ven biển miền Trung "lệch" khá nhiều so
với đường thẳng Sài Gòn – Hà Nội, dù quốc lộ 1A đi ngay qua đó. Một routing engine
(OSRM, Google Directions, Mapbox) sẽ cho khoảng cách và thời gian sát thực tế hơn.

**Chưa xét thời gian.** Khung giờ lấy/giao hàng, thời gian lái xe theo quy định,
và thời điểm xe rảnh đều chưa vào công thức. Đây là yếu tố các nền tảng lớn đều có.

**Chưa xét yêu cầu đặc thù.** Hàng lạnh, hàng nguy hiểm, hàng quá khổ cần loại xe
và giấy phép tương ứng. Hiện `vehicle_type` chưa được dùng trong lọc cứng.

**Chưa xét lịch sử tài xế.** Tỉ lệ giao đúng hẹn, tỉ lệ huỷ chuyến, đánh giá của
chủ hàng — các nền tảng thường đưa vào như một thành phần điểm riêng.

**Trọng số đang là phỏng đoán có cơ sở.** Chúng dựa trên thứ tự ưu tiên mà tài
liệu ngành nêu (quãng đường rỗng là chi phí lớn nhất), chưa được hiệu chỉnh bằng
dữ liệu vận hành thật. Khi có đủ lịch sử ghép chuyến, nên kiểm định lại bằng cách
đo tỉ lệ tài xế chấp nhận đơn được gợi ý.

## Kiểm thử

`matching_service/internal/biz/scoring_test.go` phủ các tình huống:

| Test | Kiểm điều gì |
|---|---|
| `TestThuanTuyenXepTrenNghichTuyen` | Hàng cùng hướng xếp trên hàng ngược hướng |
| `TestDonGanXeXepTrenDonXa` | Đơn ít lệch tuyến được ưu tiên |
| `TestDonVuotTaiTrongBiLoai` | Hàng quá nặng hoặc quá cồng kềnh bị loại |
| `TestGiaThapHonMucToiThieuBiLoai` | Đơn dưới giá sàn của tài xế bị loại |
| `TestDonLapDayNhieuHonDuocUuTien` | Cùng tuyến thì đơn lấp đầy hơn xếp trên |
| `TestXeTimHangChoChuyenVe` | Hàng chiều về xếp trên hàng dừng giữa đường |
| `TestDiemNamTrongKhoangHopLe` | Mọi điểm thành phần nằm trong [0, 1] |
| `TestTrongSoCongLaiBangMot` | Tổng trọng số bằng 1 |

## Tham khảo

- [How efficient load matching can reduce deadhead miles in trucking — Uber Freight](https://medium.com/uber-under-the-hood/how-efficient-load-matching-can-reduce-deadhead-miles-in-trucking-75e77ccb7d4d)
- [Truck Backhaul Optimization: Reduce Deadhead Miles and Protect Margins — PCS Software](https://pcssoft.com/blog/truck-backhaul-optimization/)
- [Cut Deadhead Miles with AI: A Practical Framework — PCS Software](https://pcssoft.com/blog/cut-deadhead-miles-ai/)
- [Optimization of Truck–Cargo Online Matching for the Less-Than-Truck-Load Logistics Hub under Real-Time Demand](https://doi.org/10.3390/math12050755)
- [Solving truck-cargo matching for drop-and-pull transport with genetic algorithm based on demand-capacity fitness](https://www.sciencedirect.com/science/article/pii/S1110016820302295)
