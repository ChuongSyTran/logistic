# Logistics OS

Nền tảng ghép chuyến vận tải hàng hoá. Chủ hàng đăng đơn, hệ thống tìm xe đang
chạy phù hợp quanh điểm lấy hàng, hai bên thương lượng giá và chốt hợp đồng.

Kiến trúc microservice viết bằng Go: **Nginx → gateway_service → các service nội
bộ qua gRPC**.

---

## Bắt đầu từ đâu

| Bạn muốn | Đọc |
|---|---|
| Hiểu hệ thống nhìn tổng thể | [Tổng quan hệ thống](docs/architecture/system-overview.md) |
| Hiểu bên trong **một** service | [docs/services/](docs/services/) — 8 tài liệu, mỗi cái một sơ đồ |
| Hiểu một nghiệp vụ chảy qua **nhiều** service | [docs/flows/](docs/flows/) — 6 tài liệu |
| Xem tất cả sơ đồ trong một trang | [docs/rendered/diagrams.html](docs/rendered/diagrams.html) |
| Build / deploy | [Quy trình build](docs/operations/build-process.md) |
| Toàn bộ tài liệu | [Mục lục docs/](docs/README.md) |

---

## Chạy nhanh

```bash
make auth-keys       # BẮT BUỘC chạy trước: sinh cặp khoá RSA ký/kiểm JWT
cp .env .env.local   # xem lại các giá trị bí mật trước khi chạy thật
docker compose up -d
```

Trước khi chạy, đặt `AUTH_SERVICE_BOOTSTRAP_ADMIN_EMAIL` và
`AUTH_SERVICE_BOOTSTRAP_ADMIN_PASSWORD` trong `.env`: API đăng ký chỉ nhận
`driver` và `shipper`, nên đây là lối duy nhất để có admin. Không có admin thì
không ai duyệt được giấy tờ xe và tài xế không bật được nhận đơn.

`make auth-keys` tạo `secrets/jwt_private.pem` và `secrets/jwt_public.pem`.
auth_service ký token bằng private key, gateway kiểm bằng public key; thiếu file
thì cả hai service dừng ngay lúc khởi động. Thư mục `secrets/` nằm trong
`.gitignore` nên mỗi máy tự sinh khoá riêng.

Sau khi các container khoẻ:

| Giao diện | Địa chỉ |
|---|---|
| API Gateway | http://localhost:8080 |
| Swagger UI | http://localhost:8080/swagger/index.html |
| Jaeger (trace) | http://localhost:16686 |
| RabbitMQ Management | http://localhost:15672 |
| Kafka UI | http://localhost:8081 |
| Kibana | http://localhost:5601 |

---

## Cấu trúc thư mục

```
api/                    hợp đồng protobuf + code sinh ra (dto)
pkg/                    thư viện dùng chung giữa các service
  ├── apperr/           bảng mã lỗi, dịch sang gRPC/HTTP
  ├── cache/            bọc Redis (cache + GEO)
  ├── mq/               bọc RabbitMQ (publisher/consumer + DLQ)
  ├── events/           hợp đồng sự kiện giữa producer và consumer
  ├── middleware/       interceptor gRPC dùng chung
  └── tracer/           OpenTelemetry

gateway_service/        HTTP → gRPC, phân quyền, chuẩn hoá lỗi
auth_service/           đăng nhập, JWT, Google OAuth2
user_service/           danh tính, hồ sơ, sổ địa chỉ, thiết bị
vehicle_service/        phương tiện, giấy tờ, GPS, trạng thái nhận đơn
matching_service/       lõi ghép đơn ↔ xe
notification_service/   hộp thư thông báo (consumer RabbitMQ)
media_service/          upload file
wallet_service/         ví và đặt cọc (chưa có trong docker-compose)

config/                 file cấu hình cho Postgres/MySQL
scripts/                script khởi tạo replication
nginx/                  reverse proxy
terraform/              hạ tầng AWS
docs/                   tài liệu — xem docs/README.md
```

---

## Lệnh hay dùng

```bash
make gen-all          # sinh lại proto + ent + goverter
make verify-modules   # kiểm tra mỗi module tự build được khi không có go.work
make tidy-modules     # go mod tidy từng module, không qua workspace
make docker-build     # build thử toàn bộ image ở local
make diagrams         # sinh lại sơ đồ .drawio + .svg + trang HTML
make docs-lint        # kiểm tra liên kết chết và quy ước đặt tên tài liệu
```

Chi tiết từng target: `make help`.

---

## Kiểm thử

```bash
# Unit test
go test ./...

# Test tích hợp (cần RabbitMQ + Redis + Postgres đang chạy)
cd notification_service && go test -tags=integration ./... -v
cd vehicle_service      && go test -tags=integration ./... -v
cd user_service         && go test -tags=integration ./... -v
```

```bash
# Test API bằng Postman: import hai file này
logistic.postman_collection.json
logistic.postman_environment.json

# hoặc chạy bằng dòng lệnh
newman run logistic.postman_collection.json -e logistic.postman_environment.json
```

Chi tiết bốn tầng test, cách import Postman, và bốn test bất biến canh giữ hệ
thống: [docs/guides/testing.md](docs/guides/testing.md).

Chạy bằng Podman trên Windows: [docs/operations/podman-windows.md](docs/operations/podman-windows.md).

Biến môi trường cho test tích hợp: xem
[matching-notification-flow.md](docs/flows/matching-notification-flow.md#8-chạy-và-kiểm-thử).
