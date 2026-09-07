# auth_service

Danh tính và JWT: đăng ký, đăng nhập, Google OAuth2, phát hành và thu hồi token.

**Nó PHÁT HÀNH danh tính, không kiểm tra danh tính cho từng request.** Việc kiểm
do gateway làm tại chỗ bằng public key — xem
[Luồng xác thực và phân quyền](../flows/authentication-flow.md).

| | |
|---|---|
| Cổng | 9001 (chỉ trong mạng nội bộ, không publish) |
| Database | MySQL (master + slave) |
| Bảng | `users`, `refresh_tokens` |
| RPC | 8 |

![Sơ đồ auth_service](../diagrams/svg/auth-service.svg)

## API

| RPC | Việc |
|---|---|
| `Register` | Tạo tài khoản bằng email + mật khẩu |
| `Login` | Kiểm tra mật khẩu, trả `TokenPair` + profile |
| `GetGoogleLoginURL` | Sinh URL tới màn hình consent của Google |
| `GoogleCallback` | Đổi `code` lấy token, tạo/khớp tài khoản |
| `VerifyToken` | Giải mã access token, trả `UserProfile` |
| `RefreshToken` | Đổi refresh token lấy cặp mới, đọc lại vai trò từ DB |
| `Logout` | Thu hồi phiên refresh |
| `GetPublicKeys` | Trả JWKS — phục vụ xoay khoá và đối chiếu vận hành |

`VerifyToken` **không** còn nằm trên đường đi của mỗi request. Gateway verify tại
chỗ; RPC này còn lại để lấy profile theo token và cho công cụ nội bộ dùng.

## Khoá ký

auth_service là **thứ duy nhất** trong hệ thống chạm vào private key. Nó ký bằng
RS256; gateway verify bằng public key.

```
AUTH_SERVICE_JWT_PRIVATE_KEY   -> chỉ service này
GATEWAY_JWT_PUBLIC_KEY         -> chỉ gateway
```

Thuật toán bất đối xứng là điều kiện để tách hai vai trò. Với thuật toán đối
xứng, trao cho gateway khả năng verify là trao luôn khả năng ký, nên gateway bị
chiếm đồng nghĩa kẻ tấn công tự phát hành được token admin.

Cùng lý do đó, hai loại token KHÔNG được ký bằng hai khoá dẫn xuất từ một gốc
chung (kiểu `secret + "_refresh"`): biết một là suy ra được cái kia.

Sinh khoá bằng `make auth-keys`. Khoá được nạp và kiểm **lúc khởi động**, không
phải lúc có request đầu tiên.

## Dữ liệu

### `users`

Khoá chính là **UUID v7**, cùng không gian định danh với user_service và mọi
service khác. Ràng buộc này bắt buộc vì id của bảng này đi vào JWT làm `sub`, rồi
gateway chuyển nó xuống các service khác để tra cứu.

Cột `role` (`driver` | `shipper` | `admin`) nằm ở bảng này vì token phải mang được
vai trò. Không có nó, gateway buộc phải hỏi user_service mỗi request chỉ để biết
người này có phải admin — đúng cái round-trip mà cả thiết kế này sinh ra để bỏ.

Ba trường **Sensitive** theo khai báo ent: `password`, `totp_secret`, `google_id`.
Ent đánh dấu Sensitive nghĩa là chúng không xuất hiện khi in struct ra log — nhưng
đó chỉ là lớp cuối. Nguyên tắc trong service này: `entity.UserProfile` (model đọc)
**không hề có** các trường đó, nên chúng không thể lọt ra ngoài kể cả trong lời gọi
nội bộ.

### `refresh_tokens`

Bảng này **không lưu chuỗi token**. Nó lưu `jti` — mã định danh nằm trong token.
Nguyên tắc giống việc không lưu mật khẩu dạng thô: DB bị lộ thì kẻ đọc được cũng
không có thứ gì dùng để đăng nhập.

| Cột | Vai trò |
|---|---|
| `id` | chính là `jti` của refresh token |
| `user_id` | để thu hồi toàn bộ phiên của một người |
| `expires_at` | để dọn định kỳ |
| `revoked_at` | khác nil = đã thu hồi. Không xoá hẳn dòng, để còn dấu vết điều tra |
| `used_at` | đánh dấu đã đem đi đổi — nền tảng của phát hiện dùng lại |

## Quyết định thiết kế đáng chú ý

**Vì sao access token không trạng thái còn refresh token thì có?**
Access token verify chỉ cần chữ ký, không chạm DB, nên chịu được tải lớn — cái giá
là không thu hồi được, và ta trả giá đó bằng cách để nó sống 15 phút. Refresh
token sống 7 ngày nên bắt buộc phải thu hồi được, và vì mỗi phiên chỉ dùng nó 15
phút một lần nên một lượt đọc DB là không đáng kể. Đây chính là đánh đổi mà OAuth
2.0 (RFC 6749) mô tả, và là lý do hai loại token tồn tại song song.

**Vì sao refresh token không mang `role`?**
Vai trò được đọc lại từ DB ở mỗi lần làm mới. Nhờ vậy một người vừa bị hạ quyền
không tự gia hạn được vai trò cũ suốt 7 ngày.

**Vì sao `MarkUsed` đặt điều kiện trong chính câu UPDATE?**

```go
Where(refreshtoken.IDEQ(id), refreshtoken.UsedAtIsNil(), refreshtoken.RevokedAtIsNil())
```

Hai request refresh cùng một token đến đồng thời — chuyện có thật khi app mở nhiều
tab hoặc mạng chập chờn khiến client gửi lại. Kiểm trước rồi ghi sau thì **cả
hai** đều thấy `used_at` rỗng và cả hai đều được cấp token. Để DB phân xử thì đúng
một request thắng.

**Vì sao đăng nhập sai vẫn chạy bcrypt?**
Khi email không tồn tại, service vẫn chạy một lần bcrypt trên hash giả. Không có
bước đó, email chưa đăng ký trả lời trong ~1ms còn email có thật mất ~250ms —
chênh lệch đủ để dò xem địa chỉ nào đã đăng ký. Cùng một rò rỉ mà đoạn code phía
trên định chặn, chỉ là qua đồng hồ thay vì qua câu chữ.

**Vì sao việc ký token nằm ở `pkg/authn` chứ không ở tầng biz?**
Cách ký token là **hợp đồng** giữa auth_service và gateway, không phải luật nghiệp
vụ của riêng auth_service. Đặt hợp đồng ở một chỗ thì bên ký và bên kiểm không thể
hiểu khác nhau về thuật toán, issuer hay tên claim.

**Tách `UserRegister` và `UserLogin` thay vì một struct chung.**
Một struct làm nhiều việc dẫn tới validate rối (trường nào bắt buộc tuỳ luồng),
khó mock trong test, và mỗi lần luật nghiệp vụ của một luồng đổi là struct chung
lại phình thêm.

**Vì sao có `entity.User` riêng thay vì dùng thẳng `ent.Users`?**
`ent` là mối quan tâm hạ tầng — nó map thẳng vào cơ sở dữ liệu. Dùng `ent.Users`
trong biz/controller là để tầng nghiệp vụ phụ thuộc vào chi tiết lưu trữ. Tách ra
thì đổi ORM hay đổi loại database chỉ phải viết lại tầng repo.

## Cấu hình

```
AUTH_SERVICE_GRPC_PORT=9001

# Khoá ký — CHỈ service này có private key
AUTH_SERVICE_JWT_PRIVATE_KEY=/run/secrets/jwt_private.pem
AUTH_SERVICE_JWT_ACCESS_TTL=15m
AUTH_SERVICE_JWT_REFRESH_TTL=168h

AUTH_SERVICE_DB_HOST=auth-db-master
AUTH_SERVICE_GOOGLE_CLIENT_ID=…
AUTH_SERVICE_GOOGLE_CLIENT_SECRET=…
AUTH_SERVICE_GOOGLE_REDIRECT_URL=…

OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4317
```

## Vận hành

**Dọn phiên hết hạn.** `SessionRepo.DeleteExpired` xoá các dòng đã quá
`expires_at`. Chưa có lịch chạy tự động, nên bảng `refresh_tokens` lớn dần theo
số lần đăng nhập.

**Migrate lược đồ.** `client.Schema.Create` của ent thêm được cột và chỉ mục nhưng
không đổi được kiểu khoá chính. Database tạo trước khi khoá chính chuyển sang UUID
cần chạy `scripts/migrations/auth_users_int_to_uuid.sql`.

## Liên quan

- [Luồng xác thực và phân quyền](../flows/authentication-flow.md)
- [Đăng nhập Google và chống CSRF](../architecture/oauth-google-flow.md)
- [gateway_service](gateway-service.md)
