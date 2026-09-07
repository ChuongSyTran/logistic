# Quản lý migration bằng Atlas

Ent tự tạo bảng bằng `client.Schema.Create()` khi service khởi động — tiện cho
dev nhưng KHÔNG dùng được cho production, vì nó chỉ thêm chứ không bao giờ xoá
hay đổi cột, và không để lại lịch sử để rollback.

Atlas sinh ra file SQL migration có version từ chính ent schema, nên production
chạy đúng những câu lệnh đã được review.

## So sánh schema hiện tại với thư mục migration

Sinh file migration mới từ phần chênh lệch giữa `ent/schema` và các migration đã có:

```bash
atlas migrate diff migration_name \
  --dir "file://ent/migrate/migrations" \
  --to "ent://ent/schema" \
  --dev-url "docker://mysql/8/ent"
```

`--dev-url` là database tạm Atlas dựng lên để tính diff — nó bị xoá sau khi chạy,
không đụng tới database thật.

## Tạo lại atlas.sum sau khi sửa tay file SQL

Atlas ký checksum toàn bộ thư mục migration để phát hiện file bị sửa lén. Sửa tay
một file `.sql` thì phải băm lại, nếu không lệnh `apply` sẽ từ chối chạy:

```bash
atlas migrate hash --dir "file://ent/migrate/migrations"
```

## Áp dụng migration

```bash
atlas migrate apply \
  --dir "file://ent/migrate/migrations" \
  --url "mysql://root:12345@localhost:3307/go-back-prod"
```

## Database đã có sẵn bảng (baseline)

Khi đưa Atlas vào một database đang chạy, các bảng đã tồn tại rồi. Chạy `apply`
thẳng sẽ lỗi vì Atlas cố tạo lại chúng.

Cách xử lý: tạo migration `init` mô tả trạng thái hiện tại, rồi **đánh dấu là đã
chạy** mà không thực thi:

```bash
atlas migrate set <version> \
  --dir "file://ent/migrate/migrations" \
  --url "mysql://root:12345@localhost:3307/go-back-prod"
```

Từ đó trở đi, mọi migration mới sẽ chạy bình thường.

## Liên quan

- [Sinh code ent](ent-codegen.md)
- [Cấu hình master-slave](../architecture/database-replication.md)
