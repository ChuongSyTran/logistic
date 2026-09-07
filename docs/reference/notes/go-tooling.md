# Ghi chú lặt vặt về Go tooling

Ghi chú thô, không phải tài liệu kiến trúc. Giữ lại vì thỉnh thoảng vẫn cần tra.

## go.sum dùng để làm gì

`go.sum` lưu mã băm của từng thư viện để xác minh code tải về đúng là bản gốc,
không bị chỉnh sửa trên đường truyền hay ở proxy.

Mỗi thư viện có **hai** dòng băm:

```
github.com/bytedance/gopkg v0.1.4 h1:oZnQwnX82KAIWb7033bEwtxvTqXcYMxDBaQxo5JJHWM=
github.com/bytedance/gopkg v0.1.4/go.mod h1:v1zWfPm21Fb+OsyXN2VAHdL6TBb2L88anLQgdyje6R4=
```

- Dòng `h1:` — băm của toàn bộ mã nguồn module.
- Dòng `/go.mod h1:` — băm của riêng file `go.mod`, để Go kiểm tra được đồ thị
  dependency mà chưa cần tải hết mã nguồn.

Liên quan tới repo này: mono-repo dùng `go.work` ở local nhưng Docker build với
`GOWORK=off`, nên mỗi module phải có `go.sum` ĐỦ của riêng nó. Xem ghi chú dài
trong `go.work` và target `make tidy-modules`.

## Chọn cổng cho service

Cổng 1–3000 phần lớn đã bị hệ điều hành và phần mềm phổ thông chiếm. Đặt service
từ 3000 trở lên cho an toàn.

Bảng cổng đang dùng trong repo: xem [tổng quan hệ thống](../../architecture/system-overview.md).

## Gin

Gin là framework: nó bó dev theo khung nó thiết kế sẵn, đổi lại được routing,
binding và middleware chuẩn hoá. Trong repo này chỉ `gateway_service` dùng Gin;
các service nội bộ nói gRPC thuần.
