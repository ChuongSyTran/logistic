# Luồng Matching → Notification (Redis + RabbitMQ)

Tài liệu này mô tả phần mở rộng mới của hệ thống: nghiệp vụ ghép xe chạy, lớp
cache/chỉ mục Redis, và đường đi thông báo qua RabbitMQ.

![Sơ đồ luồng matching → notification](../diagrams/svg/matching-notification-flow.svg)

---

## 1. Kiến trúc tổng thể

```
                 Internet
                    │
                    ▼
            ┌───────────────┐
            │     Nginx     │  TLS, rate-limit (GPS có ngưỡng riêng)
            │   :80/:443    │
            └───────┬───────┘
                    │ HTTP
                    ▼
          ┌─────────────────────┐
          │  gateway_service    │  gin · /api/v1 (client) · /api/v1/admin (admin)
          │       :8080         │  RequestID · Recovery · AccessLog · RequireRole
          └──┬───┬───┬───┬───┬──┘
   gRPC      │   │   │   │   │
    ┌────────┘   │   │   │   └──────────────┐
    ▼            ▼   ▼   ▼                  ▼
 auth:9001  media  matching  user:9004  notification:9006
            :9002   :9003    vehicle:9005
                      │           │              ▲
                      │           │              │
                      │      ┌────┴────┐         │
                      │      │  Redis  │         │
                      │      │  GEO +  │         │
                      │      │  cache  │         │
                      │      └─────────┘         │
                      │                          │
                      └────► RabbitMQ ───────────┘
                          logistic.events
                          (topic exchange)
```

**Nguyên tắc**: client bên ngoài chỉ nói chuyện với Nginx → gateway. Các service
nội bộ chỉ nhận gRPC trong mạng `logistic_net`.

---

## 2. Hai nghiệp vụ thông báo chính

### (1) Chủ hàng tìm xe → báo cho các tài xế tiềm năng

```
POST /api/v1/matching/bids
        │
        ▼
matching_service.SubmitBid
        │
        ├─► repo.CreateBid                    (Postgres: lưu đơn)
        ├─► repo.FindAskForBid                (tìm tài xế phù hợp)
        │
        └─► notifier.NotifyDriverCandidates
                 │
                 ▼  routing key: matching.driver.candidates_found
            RabbitMQ  logistic.events
                 │  binding "matching.#"
                 ▼
            notification.events (queue)
                 │
                 ▼
        notification_service consumer
                 │
                 ├─► lọc theo NotificationPreference của TỪNG tài xế
                 ├─► ghi processed_events + notifications (1 transaction)
                 └─► xoá bộ đếm chưa đọc trên Redis
```

Kết quả: mỗi tài xế trong danh sách nhận **một** thông báo
`"Có đơn hàng phù hợp gần bạn — cách bạn 2.4 km"`, kênh `push`.

### (2) Ghép được xe → báo cả hai phía

```
POST /api/v1/matching/matches/accept
        │
        ▼
matching_service.AcceptOffer
        │
        ├─► walletClient.CheckBalance         (đủ tiền cọc?)
        ├─► kafkaPub → wallet.hold_deposit    (đóng băng cọc, bất đồng bộ)
        ├─► repo.CreateMatchContract          (Postgres)
        │
        └─► notifier.NotifyMatchFound
                 │
                 ▼  routing key: matching.match.found
            RabbitMQ → notification_service
                 │
                 ├─► chủ hàng: "Đã tìm được xe cho đơn hàng của bạn"
                 └─► tài xế:   "Bạn vừa nhận được một đơn hàng"
```

Một sự kiện → **hai** bản ghi notification, vì cờ `is_read` của hai người là
độc lập.

### Các sự kiện còn lại

| Routing key | Người nhận | Ý nghĩa |
|---|---|---|
| `matching.driver.candidates_found` | tài xế | có đơn phù hợp gần bạn |
| `matching.match.found` | chủ hàng + tài xế | đã chốt xe |
| `matching.offer.received` | chủ hàng | tài xế vừa báo giá |
| `matching.offer.rejected` | tài xế | báo giá không được chọn |
| `matching.cargo.suggested` | tài xế | gợi ý đơn cho chuyến rỗng |

---

## 3. Vì sao RabbitMQ, khi đã có Kafka và NATS?

Ba broker phục vụ ba nhu cầu khác nhau, không thay thế nhau được:

| Broker | Vai trò | Vì sao không dùng cái khác |
|---|---|---|
| **Kafka** | nhật ký sự kiện lâu dài, phân tích, dựng lại trạng thái | không có retry/DLQ theo từng message |
| **NATS core** | đẩy realtime tới app **đang mở** | không giữ message cho người offline |
| **RabbitMQ** | thông báo **bền** tới người dùng | fan-out theo routing key, retry, dead-letter |

Tài xế đang lái xe, app ở chế độ nền hoặc mất sóng — message NATS bay mất.
RabbitMQ giữ message tới khi notification_service ghi được vào inbox.

### Topology (code tự khai báo lúc khởi động, `pkg/mq.DeclareTopology`)

```
logistic.events (topic)
   └── binding "matching.#" ──► notification.events
                                    │ x-dead-letter-exchange
                                    ▼
                          logistic.events.dlx ──► notification.events.dlq
```

### Chống xử lý trùng

RabbitMQ bảo đảm *ít nhất một lần*. Service chết sau khi ghi DB nhưng trước khi
ACK → message được giao lại → tài xế nhận hai lần cùng một thông báo.

Cách chặn: bảng `processed_events` với unique index trên `event_id`, và việc ghi
dấu nằm **trong cùng transaction** với việc tạo notification
(`repo.CreateWithEventGuard`). Trùng khoá → rollback → không sinh bản ghi thừa.

---

## 4. Redis: hai vai trò rất khác nhau

| Service | DB | Prefix | Dùng để làm gì |
|---|---|---|---|
| user_service | 0 | `user` | cache hồ sơ, địa chỉ, thiết bị |
| vehicle_service | 1 | `vehicle` | cache xe + **chỉ mục GEO xe đang online** |
| notification_service | 2 | `notif` | bộ đếm thông báo chưa đọc |

### Chỉ mục GEO — trái tim của "tìm xe đang chạy"

Bài toán: mỗi đơn hàng cần tìm xe quanh điểm lấy hàng, trong khi tài xế ping GPS
vài giây một lần. Để Postgres gánh nghĩa là quét chỉ mục không gian trên bảng
đang bị ghi liên tục.

Redis GEO là sorted set với score = geohash 52-bit, nên `GEOSEARCH` trả về trong
vài mili-giây ngay trên RAM.

```
SearchNearby:
  1. GEOSEARCH  → vehicle_id + khoảng cách, đã sắp gần→xa (lấy dư ×3)
  2. 2 truy vấn gộp xuống Postgres (tránh N+1): thông tin xe + sức chứa trống
  3. lọc: đã duyệt giấy tờ · đang active · đúng loại xe · đủ tải/đủ khối
```

**Redis chết thì sao?** Rơi về đường dự phòng: lọc thô theo ô lưới `zone_id` trên
Postgres rồi tính haversine trong Go. Chậm hơn, nhưng hệ thống vẫn ghép được đơn.

**Đồng bộ chỉ mục** — xe phải rời chỉ mục ngay khi:
- tài xế tắt nhận đơn (`SetDriverAvailability` với `is_online=false`)
- xe chuyển sang `maintenance` / `inactive`
- admin từ chối duyệt xe
- xe bị xoá

Thiếu bất kỳ điểm nào ở trên là matching sẽ gán đơn cho một chiếc xe nằm garage.

### Chiến lược cache

`cache-aside` + `invalidate-on-write`:

```
ĐỌC : Redis → miss → Postgres → ghi ngược lên Redis kèm TTL
GHI : Postgres trước, XOÁ key liên quan sau (không ghi đè)
```

Xoá chứ không ghi đè: ghi đè mở ra khả năng hai request đồng thời ghi lộn thứ tự
và để lại bản cũ trong cache. Xoá thì tệ nhất chỉ là một lần cache miss.

---

## 5. Xử lý lỗi

Chuỗi dịch lỗi xuyên suốt ba tầng:

```
ent.NotFoundError
      │ repo/repo_error.go — wrapError
      ▼
apperr.Error{Kind: NotFound, Code: "USER_NOT_FOUND"}
      │ pkg/middleware — ErrorInterceptor (gRPC)
      ▼
gRPC status NOT_FOUND + errdetails.ErrorInfo{Reason: "USER_NOT_FOUND"}
      │ gateway/internal/response — Error()
      ▼
HTTP 404 { "error": { "code": "USER_NOT_FOUND", "message": "..." } }
```

Nhờ vậy:
- Controller **không có** một dòng `status.Errorf` nào.
- Client switch theo `code` (ổn định) chứ không theo câu chữ.
- Chi tiết kỹ thuật (tên bảng, câu SQL) chỉ nằm trong log, không lộ ra ngoài.

**Interceptor gRPC** (`pkg/middleware.ChainForService`), thứ tự có ý nghĩa:

```
Recovery  → bắt panic, kể cả panic của các middleware bên trong
Logging   → method + code + thời gian
Error     → sát handler nhất, là thứ cuối cùng chạm vào error
```

**Middleware gateway**: `RequestID → Recovery → AccessLog → IdentityContext → ErrorGuard`,
và `RequireRole("admin")` gắn ở **cấp group** `/api/v1/admin` — thêm endpoint
admin mới là tự động được bảo vệ.

---

## 6. Chuyển đổi 3 tầng dữ liệu (goverter)

Mọi service đều theo đúng mô hình của `matching_service`:

```
ent.*     (dao — ent generate)      biết cột, enum, con trỏ nullable
   ↕ goverter
entity.*  (viết tay)                ngôn ngữ nghiệp vụ thuần Go
   ↕ goverter
pb.*      (dto — protobuf generate) biết dây truyền, timestamppb
```

Cấu hình dùng chung: `matchIgnoreCase` (khớp `ID`↔`Id`, `UserID`↔`UserId`),
`ignoreUnexported` (bỏ qua field nội bộ của protobuf), và một loạt `extend` để
goverter tự áp dụng helper cho **mọi** cặp kiểu tương ứng.

Sinh lại: `make mapper-all`.

---

## 7. Cơ sở dữ liệu — 22 bảng

| Service | Bảng |
|---|---|
| user_service | `users`, `driver_profiles`, `shipper_profiles`, `addresses`, `user_devices` |
| vehicle_service | `vehicles`, `vehicle_documents`, `vehicle_locations`, `driver_availabilities` |
| notification_service | `notifications`, `notification_templates`, `notification_preferences`, `processed_events` |
| matching_service | `asks`, `bids`, `matches`, `requirements`, `bids_requirements` |
| auth_service | `users` |
| wallet_service | `wallets`, `transactions`, `processed_messages` |

Sinh lại: `make ent-all`.

---

## 8. Chạy và kiểm thử

```bash
# Sinh lại toàn bộ code generate (proto + ent + goverter)
make gen-all

# Kiểm tra mỗi module tự build được khi KHÔNG có go.work (giống hệt Docker/CI)
make verify-modules

# Dựng toàn bộ hệ thống
docker compose up -d
```

### Test tích hợp

Cần RabbitMQ + Redis + Postgres đang chạy:

```bash
# Luồng RabbitMQ đầu-cuối + chống trùng
cd notification_service && go test -tags=integration ./internal/consumer/... -v

# Chỉ mục Redis GEO + các luật lọc xe
cd vehicle_service && go test -tags=integration ./internal/repo/... -v

# Cache-aside và invalidate
cd user_service && go test -tags=integration ./internal/repo/... -v
```

Biến môi trường: `IT_PG_HOST`, `IT_PG_PORT`, `IT_PG_USER`, `IT_PG_PASSWORD`,
`IT_PG_DB`, `IT_REDIS_HOST`, `IT_REDIS_PORT`, `IT_MQ_HOST`, `IT_MQ_PORT`,
`IT_MQ_USER`, `IT_MQ_PASSWORD`.
