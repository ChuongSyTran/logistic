# Đăng nhập Google OAuth 2.0 và cơ chế chống CSRF

Áp dụng cho `auth_service` và hai endpoint ở gateway:

- `GET /api/v1/auth/google/login`
- `GET /api/v1/auth/google/callback`

## Client ID và Client Secret là của ai?

Đây là **căn cước của ứng dụng** (Logistics Server), KHÔNG phải của người dùng.
Server dùng cặp này để chứng minh danh tính với Google khi đi đổi `code` lấy token.

Hệ quả thực tế: `client_secret` không bao giờ được xuất hiện ở phía trình duyệt.
Chỉ `client_id` được kẹp vào link redirect.

## Vì sao cần tham số `state`

Không có `state`, kẻ tấn công có thể lấy link callback hợp lệ của chính mình rồi
lừa nạn nhân bấm vào — kết quả là tài khoản Google của **kẻ tấn công** bị gắn vào
phiên đăng nhập của **nạn nhân**. Đó là CSRF trong luồng OAuth.

## Luồng chống CSRF bằng state + cookie

![Luồng xác thực](../diagrams/svg/authentication-flow.svg)

```
1. User bấm "Đăng nhập với Google"
      │
2. Gateway sinh chuỗi ngẫu nhiên  state = "state123"
      │
3. Gateway ghi "state123" vào HttpOnly cookie trên máy User (sống 5 phút)
      │
4. Gateway redirect 307 sang Google, kẹp theo state123 + client_id
   (KHÔNG kẹp client_secret)
      │
5. Google trả về callback:  /callback?code=...&state=state123
      │
6. Gateway lấy state từ URL, lấy state từ cookie, SO SÁNH
      │
      ├── khớp     -> đổi code lấy token, đăng nhập thành công
      └── lệch/thiếu -> 400 INVALID_OAUTH_STATE
```

## Vì sao cách này chặn được tấn công

Giả sử kẻ tấn công copy nguyên link callback (có cả `code` và `state123`) rồi lừa
nạn nhân bấm vào:

- Trình duyệt **nạn nhân** không hề có cookie chứa `state123` — cookie đó nằm trên
  máy kẻ tấn công.
- Gateway so URL với cookie: một đằng một nẻo → từ chối ngay.

Điểm cốt lõi: **chuỗi `state` lộ trên URL cũng vô hại**, vì nó bắt buộc phải đi
kèm chiếc cookie chỉ tồn tại trên máy của đúng người đã khởi tạo yêu cầu.

Đó cũng là lý do cookie phải là `HttpOnly`: JavaScript trên trang không đọc được
nó, nên XSS cũng không lấy được để ghép cặp.

## Cấu hình liên quan

```
AUTH_SERVICE_GOOGLE_CLIENT_ID
AUTH_SERVICE_GOOGLE_CLIENT_SECRET
AUTH_SERVICE_GOOGLE_REDIRECT_URL
```

Xem `.env` ở gốc repo.
