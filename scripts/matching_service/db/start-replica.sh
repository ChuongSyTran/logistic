#!/bin/bash
# Lệnh set -e: Đảm bảo script sẽ DỪNG NGAY LẬP TỨC nếu có bất kỳ lệnh nào thất bại (VD: không thể kết nối tới Master).
set -e

echo "WAITING 10 SECOND FOR MASTER START SUCCESSFULLY"
# Dừng lại 10 giây để đảm bảo Master đã boot xong và ghi xong cấu hình pg_hba.conf.
sleep 10s

# [CƠ CHẾ KIỂM TRA DỮ LIỆU TỒN TẠI]:
# Lệnh '[ ! -s "$PGDATA/PG_VERSION" ]' kiểm tra xem thư mục Data hiện tại có trống hay không.
# -s: File có tồn tại và kích thước lớn hơn 0 byte. Dấu '!': Phủ định.
# Tóm lại: Nếu file PG_VERSION không tồn tại (ổ cứng mới tinh), thì mới chạy pg_basebackup.
# Nếu Container bị restart, file này đã có sẵn -> Script sẽ bỏ qua bước copy để tránh ghi đè data cũ.
if [ ! -s "$PGDATA/PG_VERSION" ]; then
    echo "Starting sync data from Master (pg_basebackup)..."
    
    # [GIẢI PHẪU LỆNH PG_BASEBACKUP]:
    # Lệnh này dùng giao thức Replication (Stream) để "cào" toàn bộ byte vật lý từ ổ cứng Master sang Replica.
    # -h matching-db-master : Địa chỉ Host của con Master (đã được đổi tên trong docker-compose).
    # -D "$PGDATA"          : Đường dẫn lưu data đích (bên trong con Replica).
    # -U "$GLOBAL..."       : Tài khoản Replication.
    # -vP (verbose/progress): Hiển thị log chi tiết và % tiến trình để dễ theo dõi qua 'docker logs'.
    # -w (no-password)      : Ép lệnh chạy ngầm, không được phép dừng lại chờ gõ Password. Nó sẽ tự lấy pass từ biến PGPASSWORD.
    # -R (write-recovery)   : CỜ QUAN TRỌNG NHẤT (Quyết định sinh tử của Replication).
    #                         Sau khi copy xong, nó tự động để lại 2 "tờ giấy note" trong ổ cứng Replica:
    #                         1. File 'standby.signal': Báo cho Postgres biết đây là node Read-Only.
    #                         2. File 'postgresql.auto.conf': Ghi cứng thông tin mạng (IP, User, Pass) của Master.
    #                         => Nhờ 2 file này, ở những lần Restart sau (khi bỏ qua lệnh pg_basebackup), Engine của Postgres sẽ TỰ ĐỘNG đọc cấu hình, tự động kết nối lại vào Master và kéo tiếp các file WAL (Auto-resume) mà không cần ta nhúng tay vào bằng code Bash.
    pg_basebackup -h matching-db-master -D "$PGDATA" -U "$REPLICATION_USER" -vP -w -R
else
    echo "The data already exists, skip the data scraping step."
fi

echo "Starting Replica Server..."
# [GIẢI PHẪU LỆNH EXEC]:
# Lệnh 'exec' trong Linux có nghĩa là "chiếm xác". Tiến trình bash script hiện tại sẽ tự sát và nhường lại Process ID (PID 1) cho tiến trình docker-entrypoint gốc của Postgres. 
# Nhờ vậy, Postgres Daemon mới nhận được các tín hiệu SIGTERM từ Docker khi ta gõ lệnh 'docker stop'.
exec docker-entrypoint.sh postgres

