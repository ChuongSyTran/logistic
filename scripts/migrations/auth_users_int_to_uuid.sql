-- ===========================================================================
-- auth_service: đổi khoá chính bảng `users` từ INT tự tăng sang UUID v7
-- ===========================================================================
--
-- VÌ SAO CẦN SCRIPT NÀY
--
-- `client.Schema.Create` của ent tự thêm được cột và chỉ mục, nhưng KHÔNG đổi
-- được kiểu của khoá chính. Chạy service với schema mới trên một database cũ thì
-- ent hoặc báo lỗi, hoặc tệ hơn là bỏ qua trong im lặng và để bảng ở trạng thái
-- lệch với code.
--
-- VÌ SAO PHẢI ĐỔI
--
-- id của bảng này đi vào JWT làm `sub`, rồi gateway chuyển nó xuống user_service
-- — nơi mọi thứ định danh bằng UUID. "Người dùng số 42" không tra được ở đâu cả.
-- Sau khi đổi, toàn hệ thống chỉ còn một không gian định danh duy nhất.
--
-- ===========================================================================
-- TRƯỚC KHI CHẠY
-- ===========================================================================
--
--   1. SAO LƯU. Script này đổi khoá chính; sai một bước là mất liên kết dữ liệu.
--   2. DỪNG auth_service. Không được có kết nối nào ghi vào bảng trong lúc chạy.
--   3. Sau khi chạy: MỌI token đang lưu hành đều vô hiệu (sub cũ không còn tồn
--      tại). Người dùng phải đăng nhập lại. Hãy làm vào giờ thấp điểm.
--
-- Nếu đây là môi trường phát triển và dữ liệu bỏ được, cách nhanh hơn nhiều là
-- DROP DATABASE rồi để ent tạo lại từ đầu.
--
-- Dùng cho MySQL 8.0+ (auth_service chạy MySQL, khác với các service dùng
-- Postgres).
-- ===========================================================================

START TRANSACTION;

-- Bước 1: thêm cột UUID mới, chưa động vào khoá chính.
ALTER TABLE users ADD COLUMN id_uuid BINARY(16) NULL AFTER id;

-- Bước 2: sinh UUID v7 cho từng dòng hiện có.
--
-- MySQL không có hàm sinh UUID v7 sẵn, nên ta dựng thủ công theo RFC 9562:
--   48 bit đầu  = timestamp Unix milli
--   4 bit       = số phiên bản (0111 = 7)
--   12 bit      = phần ngẫu nhiên A
--   2 bit       = variant (10)
--   62 bit      = phần ngẫu nhiên B
--
-- Dùng created_at làm mốc thời gian thay vì thời điểm chạy migration: giữ được
-- tính chất "id tăng dần theo thứ tự tạo" — vốn là toàn bộ lý do chọn v7.
UPDATE users
SET id_uuid = UNHEX(
    CONCAT(
        LPAD(HEX(UNIX_TIMESTAMP(created_at) * 1000), 12, '0'),
        '7',
        LPAD(HEX(FLOOR(RAND() * 4096)), 3, '0'),
        HEX(FLOOR(128 + RAND() * 64)),
        LPAD(HEX(FLOOR(RAND() * POW(2, 56))), 14, '0')
    )
)
WHERE id_uuid IS NULL;

-- Bước 3: chắc chắn không có dòng nào bị bỏ sót hay trùng nhau.
--
-- Kiểm TRƯỚC khi bỏ khoá chính cũ. Sau đó thì không còn đường lùi.
SELECT
    COUNT(*)                                   AS tong_so_dong,
    SUM(id_uuid IS NULL)                       AS thieu_uuid,
    COUNT(*) - COUNT(DISTINCT id_uuid)         AS uuid_trung_nhau
FROM users;
-- thieu_uuid và uuid_trung_nhau đều phải bằng 0. Nếu không, ROLLBACK.

-- Bước 4: đổi khoá chính.
ALTER TABLE users DROP PRIMARY KEY;
ALTER TABLE users DROP COLUMN id;
ALTER TABLE users CHANGE COLUMN id_uuid id BINARY(16) NOT NULL;
ALTER TABLE users ADD PRIMARY KEY (id);

-- Bước 5: cột role mà schema mới yêu cầu.
--
-- Mặc định 'shipper' cho dữ liệu cũ vì đó là vai trò rộng nhất và ít đặc quyền
-- nhất. KHÔNG mặc định 'admin' — nâng quyền nhầm hàng loạt là chuyện không sửa
-- lại được sau khi ai đó đã dùng.
ALTER TABLE users
    ADD COLUMN role ENUM('driver', 'shipper', 'admin') NOT NULL DEFAULT 'shipper';

-- Bước 6: chỉ mục cho tra cứu theo google_id (luồng OAuth).
CREATE INDEX users_google_id ON users (google_id);

COMMIT;

-- ===========================================================================
-- SAU KHI CHẠY
-- ===========================================================================
--
--   1. Gán vai trò admin thủ công cho đúng những người cần:
--
--        UPDATE users SET role = 'admin' WHERE email IN ('...');
--
--   2. Khởi động auth_service. Ent sẽ tự tạo bảng `refresh_tokens` — bảng mới
--      hoàn toàn nên không cần migrate gì.
--
--   3. Kiểm chứng: đăng nhập rồi giải mã access token, `sub` phải là một chuỗi
--      UUID chứ không phải một con số.
-- ===========================================================================
