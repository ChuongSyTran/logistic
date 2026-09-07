# PostgreSQL master-slave replication (streaming)

Tài liệu này cung cấp bức tranh toàn cảnh về kiến trúc Master-Slave (Replication) trong PostgreSQL, cùng với hướng dẫn "cầm tay chỉ việc" để tự tay thiết lập một cụm Database hoạt động trơn tru từ con số 0 bằng `postgres:alpine`.

---

## I. Tổng Quan Kiến Trúc (The Big Picture)

### 1. Mô hình Master-Slave là gì?
Đây là mô hình phân tán cơ sở dữ liệu phổ biến nhất. Trong đó:
- **Master (Primary):** Là Node duy nhất có quyền **GHI (Write)** dữ liệu (INSERT, UPDATE, DELETE). Nó cũng có thể đọc.
- **Slave (Replica/Standby):** Là Node sao chép dữ liệu từ Master sang. Nó ở chế độ **CHỈ ĐỌC (Read-Only)**. Bất kỳ nỗ lực Ghi nào vào Node này đều bị chặn lại.

Cơ chế sao chép (Streaming Replication) hoạt động bằng cách Master liên tục gửi các bản ghi nhật ký (WAL - Write Ahead Logs) qua mạng cho Slave. Slave nhận được WAL sẽ "phát lại" (replay) các thao tác đó vào ổ cứng của mình để giữ dữ liệu đồng bộ.

### 2. Tại sao phải làm? Khi nào thì áp dụng?
Áp dụng khi hệ thống của bạn bắt đầu đối mặt với một trong các bài toán sau:
- **Tải trọng Đọc (Read) quá lớn:** Thống kê cho thấy 80% truy vấn của ứng dụng là ĐỌC, chỉ 20% là GHI. Thay vì dồn mọi thứ vào 1 con DB gây nghẽn cổ chai, ta tách luồng GHI vào Master, và chia đều luồng ĐỌC (Load Balancing) ra các con Slave.
- **Dự phòng thảm họa (High Availability - HA):** Nếu Master đột ngột chết (cháy ổ cứng, sập nguồn), ta có thể thăng cấp (Promote) ngay lập tức một con Slave lên làm Master mới chỉ trong vài giây, giúp hệ thống không bị "down" lâu.
- **Backup không ảnh hưởng hiệu năng:** Chạy các lệnh Backup nặng nề trên Slave sẽ không làm chậm Master đang phục vụ User.

### 3. Ưu và Nhược Điểm
- **Ưu điểm:**
  - Tăng khả năng chịu tải (Scalability) cho tác vụ Đọc.
  - Tăng độ tin cậy (Reliability) và an toàn dữ liệu.
  - Cấu hình gốc (Native) của PostgreSQL rất ổn định, không cần cài cắm công cụ bên thứ ba.
- **Nhược điểm:**
  - Có độ trễ nhất định (Replication Lag). Dữ liệu vừa ghi vào Master có thể mất vài mili-giây đến vài giây mới xuất hiện ở Slave.
  - Ứng dụng (Code) phải tự biết phân luồng: API Ghi thì gọi Master, API Đọc thì gọi Slave (hoặc dùng thư viện/Proxy hỗ trợ).

---

## II. Tổng Quan Triển Khai (Deployment Overview)

Để dựng được mô hình này, chúng ta cần 4 Component chính tương tác với nhau:
1. **Network (Mạng nội bộ):** Để Master và Slave nói chuyện.
2. **Cấu hình Master (`postgresql.conf`):** Phải bật cờ `wal_level = replica` để báo cho DB biết nó chuẩn bị xuất khẩu Data.
3. **Script Cấp Quyền (`register-replica.sh`):** Đặt tại Master để tạo tài khoản Replication và đục lỗ tường lửa (`pg_hba.conf`) đón Slave.
4. **Script Sao Chép (`start-replica.sh`):** Đặt tại Slave để chạy lệnh "cào" dữ liệu (`pg_basebackup`) lúc mới khởi tạo, sau đó mới bật Engine Postgres lên.

---

## III. Triển Khai Thực Tế (Step-by-Step)

Chúng ta sẽ setup bằng Docker Compose. Vui lòng tạo các file sau tại thư mục `scripts/matching_service/db/`.

### Bước 1: File cấu hình Master (`postgresql.conf`)
File này chứa các cấu hình TỐI QUAN TRỌNG để Master cho phép Slave lấy dữ liệu.

```ini
# Lắng nghe mọi kết nối TCP/IP
listen_addresses = '*'

# --- CẤU HÌNH REPLICATION ---
# Báo cho Master biết phải chuẩn bị WAL format để truyền đi
wal_level = replica
# Số lượng Slave tối đa có thể kết nối đồng thời
max_wal_senders = 10
max_replication_slots = 10
```

### Bước 2: Script chuẩn bị trên Master (`register-replica.sh`)
Script này sẽ được tự động chạy khi Master vừa tạo xong DB trống.

```bash
#!/bin/bash
# Dừng ngay nếu có lỗi
set -e

echo "=== REGISTER REPLICATION USER ==="

# 1. Tạo User Replication (Mật khẩu tự lấy từ biến môi trường)
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER $REPLICATION_USER WITH REPLICATION ENCRYPTED PASSWORD '$REPLICATION_PASSWORD';
EOSQL

# 2. Đục lỗ tường lửa (pg_hba.conf) dùng cơ chế SCRAM-SHA-256 (An toàn bảo mật cao)
FIREWALL_RULE="host replication $REPLICATION_USER 0.0.0.0/0 scram-sha-256"
echo "$FIREWALL_RULE" >> "$PGDATA/pg_hba.conf"
```

### Bước 3: Script khởi động trên Slave (`start-replica.sh`)
Script này sẽ cướp quyền khởi động của Slave. Nó phải kéo Data xong thì mới cho Slave chạy.

```bash
#!/bin/bash
set -e

# Đợi Master boot xong (10s)
sleep 10s

# Nếu ổ cứng trống thì mới chạy lệnh Clone (pg_basebackup)
if [ ! -s "$PGDATA/PG_VERSION" ]; then
    echo "Starting sync data from Master (pg_basebackup)..."
    # -h: Trỏ tới Host Master
    # -vP: In tiến trình %
    # -w: Không hỏi mật khẩu (Tự lấy từ $PGPASSWORD)
    # -R: ĐỂ LẠI FILE STANDBY.SIGNAL (Biến node này thành Read-only và tự động cấu hình Auto-resume WAL)
    pg_basebackup -h matching-db-master -D "$PGDATA" -U "$MATCHING_DB_REPLICATION_USER" -vP -w -R
else
    echo "The data already exists, skip the data scraping step."
fi

# Nhường quyền PID 1 lại cho tiến trình postgres gốc
exec docker-entrypoint.sh postgres
```

> [!WARNING] Cấp quyền thực thi (Mac/Linux)
> Nếu chạy trên Mac/Linux, hãy chạy lệnh `chmod +x scripts/matching_service/db/*.sh`.

### Bước 4: Tích hợp vào `docker-compose.yml`
Cuối cùng, dán khối cấu hình này vào file `docker-compose.yml` của bạn:

```yaml
  matching-db-master:
    image: postgres:16-alpine
    container_name: matching-db-master
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: ${MATCHING_DB_USER}
      POSTGRES_PASSWORD: ${MATCHING_DB_PASSWORD}
      POSTGRES_DB: ${MATCHING_DB_NAME}
      REPLICATION_USER: ${MATCHING_DB_REPLICATION_USER}
      REPLICATION_PASSWORD: ${MATCHING_DB_REPLICATION_PASSWORD}
    command: 
      - "postgres"
      - "-c"
      - "config_file=/etc/postgresql/postgresql.conf"
    volumes:
      - matching-db-master-data:/var/lib/postgresql/data
      # Hook khởi tạo chạy script cấp quyền
      - ./scripts/matching_service/db/register-replica.sh:/docker-entrypoint-initdb.d/register-replica.sh
      - ./scripts/matching_service/db/postgresql.conf:/etc/postgresql/postgresql.conf
    networks:
      - logistic_net

  matching-db-slave:
    image: postgres:16-alpine
    container_name: matching-db-slave
    ports:
      - "5433:5432"
    environment:
      POSTGRES_USER: ${MATCHING_DB_USER}
      POSTGRES_PASSWORD: ${MATCHING_DB_PASSWORD}
      PGPASSWORD: ${MATCHING_DB_REPLICATION_PASSWORD}
    depends_on:
      - matching-db-master
    volumes:
      - postgres-replica-data:/var/lib/postgresql/data
      # Mount file script vào container
      - ./scripts/matching_service/db/start-replica.sh:/scripts/matching_service/db/start-replica.sh
    command: 
      # Dùng script này làm mồi khởi động
      - "bash"
      - "/scripts/matching_service/db/start-replica.sh"
    networks:
      - logistic_net
```

Chạy `podman-compose up -d --build` là bạn đã có một cụm Enterprise Database Replication.

---

## IV. Cấu Hình Nâng Cao (Advanced Features)

Những tính năng dưới đây dành cho Production có tải trọng cực lớn. Có thể tìm hiểu thêm tại trang chủ PostgreSQL:

1. **Synchronous Replication (Đồng bộ đồng thời):**
   - *Vấn đề:* Mặc định Replication là Asynchronous (Bất đồng bộ). Ghi ở Master xong là Master trả về `Success` luôn, bất chấp Slave đã nhận được WAL chưa. Nếu Master chết ngay lúc đó, ta mất Data.
   - *Giải pháp:* Cấu hình `synchronous_standby_names`. Master sẽ đợi đến khi Slave xác nhận đã nhận được WAL thì mới báo `Success`. Đánh đổi là tốc độ Ghi sẽ chậm hơn.
   - [Đọc thêm tại đây](https://www.postgresql.org/docs/16/warm-standby.html#SYNCHRONOUS-REPLICATION)

2. **Replication Slots (Khe đồng bộ):**
   - *Vấn đề:* Nếu Slave bị đứt mạng vài tiếng, Master sẽ xóa các WAL cũ đi (để tiết kiệm ổ cứng). Khi Slave online lại, nó không thể đồng bộ tiếp được nữa (Mất dấu).
   - *Giải pháp:* Dùng `Replication Slots`. Master sẽ "găm" lại các file WAL chưa được Slave xác nhận, dù ổ cứng có đầy nó cũng không xóa.
   - [Đọc thêm tại đây](https://www.postgresql.org/docs/16/warm-standby.html#STREAMING-REPLICATION-SLOTS)

3. **Connection Pooling (PgBouncer/Pgpool-II):**
   - Khi có nhiều Slave, ứng dụng rất khó để biết phải chĩa luồng Đọc vào IP của con Slave nào. Sử dụng `Pgpool-II` như một lớp Proxy đứng giữa sẽ giúp tự động phân tải (Load Balancing) và tự động Failover (Thăng cấp Slave lên Master) khi sự cố xảy ra.
   - [Đọc thêm tại đây](https://www.pgpool.net/docs/latest/en/html/index.html)
