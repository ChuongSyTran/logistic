# Chạy cụm bằng Podman trên Windows

`docker-compose.yml` chạy được với Podman, nhưng Podman trên WSL2 cần một chỉnh
sửa. Tài liệu này ghi lại cấu hình đã kiểm chứng hoạt động.

## Triệu chứng nếu chưa cấu hình

```
Error response from daemon: netavark (exit code 1): nftables error:
"nft" did not return successfully while applying ruleset:
```

Container build xong, tạo xong, nhưng không khởi động được.

## Nguyên nhân

`netavark` (lớp mạng của Podman) áp ruleset nftables mà kernel WSL2 mặc định không
xử lý được trọn vẹn. Kernel **có** `CONFIG_NF_TABLES=y` và tạo được bảng NAT thủ
công, nên đây là vấn đề ở một rule cụ thể của netavark chứ không phải thiếu hỗ trợ
nftables.

Kiểm chứng nhanh: `podman run --rm --network=host alpine echo ok` chạy được, còn
với bridge network thì lỗi.

## Cấu hình đã kiểm chứng

Tắt firewall driver của netavark. Bridge network và DNS nội bộ vẫn hoạt động; NAT
ra Internet do WSL lo.

```bash
printf '[network]\nfirewall_driver = "none"\n' > /tmp/90-firewall.conf
```

```bash
cat /tmp/90-firewall.conf | podman machine ssh 'sudo tee /etc/containers/containers.conf.d/90-firewall.conf'
```

```bash
podman machine stop && podman machine start
```

> **Viết file bằng cách này, không dùng PowerShell.** Chuỗi lồng nhau trong
> PowerShell bị escape sai và tạo file hỏng ở phía Windows
> (`%APPDATA%\containers\containers.conf.d\`), khiến `podman` từ chối chạy với lỗi
> `toml: unexpected EOF`. Nếu lỡ tạo, xoá file đó đi là podman hoạt động lại.

## Phải chạy ở chế độ rootless

| Chế độ | Bridge network | Publish port ra Windows |
|---|---|---|
| rootless (mặc định) | có | **có** |
| rootful | có | không |

Với `firewall_driver = "none"`, chế độ rootful mất khả năng publish port vì không
còn rule DNAT. Rootless chuyển tiếp cổng bằng tiến trình ở không gian người dùng,
không phụ thuộc nftables — nên vẫn gọi được `http://localhost:8080` từ Windows.

Kiểm tra:

```bash
podman info --format "rootless={{.Host.Security.Rootless}}"
```

Nếu ra `false`:

```bash
podman machine stop && podman machine set --rootful=false && podman machine start
```

> **Rootless và rootful dùng kho lưu trữ riêng biệt.** Đổi qua lại là container,
> image và volume của chế độ kia không còn nhìn thấy — dữ liệu không mất, chỉ nằm
> ở namespace khác. Sau khi đổi, dữ liệu trong database phải tạo lại.

## Tài nguyên máy ảo

`podman machine list` hiển thị RAM/CPU ghi lúc tạo máy, **không phải** giá trị
thực. WSL2 tự cấp khoảng 50% RAM của máy thật. Kiểm tra thực tế:

```bash
podman machine ssh "free -h | head -2; nproc"
```

`podman machine set --cpus/--memory` **không dùng được** cho máy WSL — Podman báo
`changing CPUs not supported for WSL machines`. Muốn đổi thì sửa `%USERPROFILE%\.wslconfig`:

```ini
[wsl2]
memory=12GB
processors=8
```

Rồi `wsl --shutdown` và khởi động lại máy ảo.

## Khởi động

```bash
make auth-keys
```

```bash
podman compose up -d
```

Cụm đầy đủ khá nặng (3 Kafka broker, Elasticsearch, Kibana, 6 database). Chạy một
tập con khi chỉ cần thử luồng chính:

```bash
podman compose up -d auth-db-master auth-service gateway-service
```

## Kiểm chứng

Cả 26 container đã được chạy thử cùng lúc trên cấu hình này: 4 cặp
master-slave (1 MySQL + 3 Postgres), 6 Kafka node, Elasticsearch, RabbitMQ,
Redis, NATS và 6 service ứng dụng.

```bash
curl http://localhost:8080/healthz
```

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@logistic.vn","full_name":"Demo","password":"Matkhau@12345","role":"shipper"}'
```

Đăng ký trả về `201` kèm `id` dạng UUID v7 (bắt đầu bằng `01a…` vì 48 bit đầu là
timestamp) nghĩa là chuỗi gateway → gRPC → auth_service → MySQL đã thông.

Kiểm luồng bất đồng bộ — tài xế đăng chuyến, chủ hàng đăng đơn, rồi đọc hộp thư:

```bash
podman logs logistic-notification-service-1 | grep consumer
```

```
[consumer] matching.cargo.suggested         -> đã tạo 1 thông báo
[consumer] matching.driver.candidates_found -> đã tạo 3 thông báo
```

Hai dòng này chứng minh cả chuỗi: matching_service ghi đơn, tìm ứng viên bằng
truy vấn haversine, phát sự kiện qua RabbitMQ, notification_service nhận và dựng
thông báo từ template.

Kiểm replication:

```bash
podman exec user-db-master psql -U user_db_user -d user_db \n  -c "SELECT client_addr, state, sync_state FROM pg_stat_replication;"
```

## Lỗi thường gặp

| Lỗi | Nguyên nhân | Cách xử lý |
|---|---|---|
| `machine ... already exists` | `podman machine init` khi máy đã tồn tại | Không cần init lại; dùng `podman machine set` |
| `cannot be destroyed` | `podman machine rm` khi máy đang chạy | `podman machine stop` trước |
| `unknown driver "mysql"` | Thiếu driver trong service | Đã sửa — auth_service import cả `go-sql-driver/mysql` và `lib/pq` |
| `connection refused` tới database | Service khởi động trước khi DB sẵn sàng | Đã sửa — `auth-db-master` có healthcheck, `auth-service` có `depends_on: service_healthy` |
| Gateway trả `504 DEADLINE_EXCEEDED` | Service đích chưa chạy | Đúng hành vi: hạn chờ 5s thay vì treo vô hạn |
| RabbitMQ `.erlang.cookie: eacces` | Volume thuộc UID khác trong podman rootless | Đã sửa — volume khai `:U` để runtime tự chown |
| Postgres slave `connection refused` dù master healthy | `pg_isready` qua unix socket báo OK khi TCP chưa mở | Đã sửa — healthcheck thêm `-h 127.0.0.1` |
| `matching-service` chết lúc khởi động | Thiếu NATS/Kafka/RabbitMQ | Đã sửa — `depends_on` đủ cả năm dependency |
| Elasticsearch trả `unable to authenticate user [elastic]` | ES chưa khởi tạo xong security | Chờ tới khi healthcheck xanh rồi thử lại |
