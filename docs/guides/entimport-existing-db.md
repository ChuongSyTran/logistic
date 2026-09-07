# Sinh ent schema từ database có sẵn (database-first)

Mặc định repo này đi theo hướng **code-first**: viết `ent/schema/*.go` rồi sinh ra
bảng. Khi phải tiếp quản một database đã tồn tại, `entimport` làm chiều ngược lại
— đọc bảng thật và sinh ra file schema.

## Cài đặt và chạy

```bash
go get ariga.io/entimport/cmd/entimport

go run ariga.io/entimport/cmd/entimport \
  -dsn "mysql://root:<password>@tcp(localhost:3306)/<database>"
```

## Lỗi thường gặp

Với `github.com/golang/protobuf` bản cũ:

```
# github.com/golang/protobuf/protoc-gen-go/descriptor
descriptor.pb.go:106:61: undefined: descriptorpb.Default_FileOptions_PhpGenericServices
```

Nguyên nhân: `protoc-gen-go/descriptor` của bản v1.5.2 tham chiếu một hằng số đã
bị đổi tên ở `protobuf` mới hơn. Nâng version là xong:

```bash
go get github.com/golang/protobuf
# go: upgraded github.com/golang/protobuf v1.5.2 => v1.5.4
```

## Quy ước khai báo khoá ngoại trong ent

entimport sinh ra edge, nhưng thường phải sửa tay cho đúng ý. Quy ước dùng trong
repo này:

**Phía "một" (bảng cha)** khai báo `edge.To`:

```go
// user.go
edge.To("articles", Article.Type),
```

**Phía "nhiều" (bảng con)** khai báo `edge.From` kèm `.Field()` để cột khoá ngoại
xuất hiện thành field thật trong struct:

```go
// article.go
edge.From("user", User.Type).
    Ref("articles").
    Field("user_id").      // <- nhờ dòng này, dao có sẵn UserID
    Unique().
    Required(),
```

`.Field("user_id")` là chi tiết quan trọng: không có nó, repo phải JOIN sang bảng
cha mới lọc được theo khoá ngoại. Có nó thì query thẳng `WHERE user_id = ?`, và
tầng entity nhận luôn `UserID` mà mapper không cần xử lý gì thêm.

## Liên quan

- [Sinh code ent](ent-codegen.md)
- [Quản lý migration bằng Atlas](atlas-migration.md)
