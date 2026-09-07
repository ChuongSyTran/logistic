# media_service

Nhận file và đẩy lên kho lưu trữ ngoài. Service **không trạng thái**, không database.

| | |
|---|---|
| Cổng | 9002 |
| Database | không |
| Kho lưu trữ | Cloudinary |
| RPC | 2 |

![Sơ đồ media_service](../diagrams/svg/media-service.svg)

## API

| RPC | Việc |
|---|---|
| `UploadFile` | Nhận nội dung file + tên + thư mục, trả `public_id` và `url` |
| `DeleteFile` | Xoá theo `public_id` |

## Quyết định thiết kế đáng chú ý

**Interface `storage.FileStorage` tách khỏi cài đặt Cloudinary.**
Đổi sang S3 hay MinIO chỉ cần thêm một cài đặt mới trong `internal/storage/`,
controller không đổi một dòng.

**Vì sao service này không lưu gì?**
`public_id` và `url` được trả về cho nơi gọi, và nơi đó mới là chủ sở hữu dữ liệu
— ví dụ `vehicle_documents.file_url` bên `vehicle_service`. Nếu media_service tự
giữ một bảng "files" thì sẽ có hai nguồn sự thật về việc file nào còn được dùng,
và không ai dám xoá file nào cả.

## Cấu hình

```
MEDIA_SERVICE_GRPC_PORT=9002
MEDIA_SERVICE_CLOUDINARY_URL=cloudinary://API_KEY:API_SECRET@CLOUD_NAME
```

## Liên quan

- [vehicle_service](vehicle-service.md) — nơi tiêu thụ `file_url` cho giấy tờ xe
