# Kiểm thử

Bốn tầng test, mỗi tầng trả lời một câu hỏi khác nhau.

| Tầng | Trả lời câu hỏi | Vị trí | Lệnh | Cần hạ tầng |
|---|---|---|---|---|
| Unit | Hàm này có đúng không? | cạnh source | `go test ./...` | không |
| Bất biến | Hệ thống có còn giữ luật của nó không? | cạnh source | `go test ./...` | không |
| Tích hợp | Code có nói chuyện đúng với DB/Redis/MQ không? | cạnh source, build tag | `go test -tags=integration ./...` | Postgres, Redis, RabbitMQ |
| API | Toàn hệ thống có phục vụ đúng không? | `logistic.postman_collection.json` | Postman / Newman | cả cụm |

---

## Vì sao file test nằm cạnh source

Khác Java/C#, Go **không** tách `src/test`. Ba lý do:

**File `_test.go` không vào binary.** Trình biên dịch có build constraint ngầm cho
hậu tố này: chúng bị loại khỏi mọi lệnh build thường, chỉ nạp khi `go test`. Kiểm
chứng được: `Dockerfile` chạy `go build ./cmd` và file test không hề được biên dịch.

**`go test` chạy theo package, không theo thư mục.** Muốn test một package thì file
test phải thuộc package đó, tức phải cùng thư mục. Tách ra là mất khả năng test hàm
unexported và mất coverage theo package.

**Đó là cách thư viện chuẩn làm.** `net/http/server.go` nằm cạnh `net/http/server_test.go`.

### Hai kiểu package test

Go cho phép hai package trong cùng một thư mục, nếu package thứ hai tên `foo_test`.
Đây là ngoại lệ duy nhất của luật "một thư mục một package".

| Kiểu | Khai báo | Dùng khi | Ví dụ trong repo |
|---|---|---|---|
| Black-box | `package foo_test` | Chỉ dùng API công khai — ép test đúng bề mặt mà người khác sẽ dùng | `pkg/uuidx`, `pkg/authn` |
| White-box | `package foo` | Cần chạm hàm/biến unexported | `gateway_service/.../http` |

Mặc định nên chọn black-box. Chỉ dùng white-box khi thật sự cần vào bên trong —
như bộ test route của gateway, nơi phải dựng engine bằng thành phần nội bộ.

---

## Tầng 1 — Unit test

### `pkg/authn` — bộ test tấn công

Đây là bộ test đáng chú ý nhất trong repo, vì mỗi test tương ứng với một cách phá
thật sự có tên trong tài liệu bảo mật.

| Test | Mô phỏng đòn tấn công nào |
|---|---|
| `TestRejectsAlgConfusion` | Đổi header token sang HS256 rồi **ký bằng chính public key** — thứ ai cũng lấy được. Không chốt thuật toán thì thư viện chấp nhận, và kẻ tấn công tự ký được token admin |
| `TestRejectsAlgNone` | Đặt `alg: none`, bỏ hẳn chữ ký. Biến thể cổ điển nhất |
| `TestRejectsForeignKey` | Token do một hệ thống khác ký |
| `TestRejectsExpired` | Token quá hạn |
| `TestRejectsWrongIssuer` | Token đúng chữ ký nhưng sai `iss` |
| `TestRefreshTokenRejectedAsAccess` | Dùng refresh token (sống 7 ngày) thay cho access token (15 phút). Lọt là toàn bộ ý nghĩa của việc để access token sống ngắn mất sạch |
| `TestRefreshCarriesNoRole` | Refresh token mang `role` — người bị hạ quyền sẽ tự gia hạn được vai trò cũ |
| `TestSubjectMustBeUUID` | `sub` là số nguyên kiểu cũ thay vì UUID |
| `TestRejectsSmallKey` | Khoá RSA dưới 2048 bit |
| `TestKeyRotation` | Xoay khoá: token ký bằng khoá cũ và khoá mới đều phải verify được cùng lúc |

Kèm `BenchmarkVerifyAccess` — đo chi phí verify (~31µs). Con số này là căn cứ cho
quyết định verify tại gateway thay vì gọi gRPC xuống auth_service.

```bash
go test ./pkg/authn/ -v
go test ./pkg/authn/ -run=XXX -bench=. -benchtime=2000x
```

### `pkg/uuidx` — bẫy chuyển kiểu

| Test | Bắt lỗi gì |
|---|---|
| `TestFromBytesRejectsStringifiedUUID` | `[]byte("3f2b7c10-…")` cho ra **36 byte** thay vì 16. Go cho phép ép kiểu này, proto nhận `bytes` thì độ dài nào cũng vừa — sai lệch trôi thẳng vào DB nếu không chặn |
| `TestRoundTrip` | bytes → chuỗi → bytes phải cho lại đúng giá trị |
| `TestV7IsMonotonic` | 1000 id sinh liên tiếp phải tăng dần |
| `TestV7Version` | Đúng phiên bản 7, không phải 4 |
| `TestIsZero`, `TestEmptyAndNil` | Phân biệt "chưa set" với "set giá trị rỗng" |

### `user_service/ent/schema` — khoá chính

`TestPrimaryKeysAreUUIDv7` kiểm 5 bảng: hàm sinh id mặc định phải cho ra UUID v7.

Đổi nhầm về `uuid.New` (v4) thì **mọi thứ vẫn chạy bình thường**, chỉ chậm dần theo
kích thước bảng: v4 ngẫu nhiên làm mỗi insert rơi vào một trang B-tree khác nhau,
gây cache miss và page split liên tục. Không có test này thì gần như không ai phát
hiện ra.

`TestGeneratedIDsAreOrdered` kiểm 500 id liên tiếp thật sự tăng dần.

### Các service

| Package | Kiểm gì |
|---|---|
| `matching_service/internal/biz` | Luật ghép Bid/Ask, thứ tự khoá, nối notifier |
| `notification_service/internal/consumer` | Dựng thông báo từ sự kiện RabbitMQ |
| `notification_service/internal/entity` | Validate kênh, vai trò, loại thông báo |
| `vehicle_service/internal/entity` | Ràng buộc nghiệp vụ của phương tiện |
| `pkg/apperr` | Ánh xạ lỗi nghiệp vụ sang mã gRPC |

---

## Tầng 2 — Test bất biến

Sáu test không kiểm một hàm nào cả. Chúng kiểm **luật của cả hệ thống**, và bắt
đúng loại lỗi mà đọc code khó thấy vì mọi thứ trông vẫn hợp lý.

### `TestNoUnintentionallyPublicRoute`

`gateway_service/internal/delivery/http/auth_guard_test.go`

Duyệt **mọi** route dưới `/api/v1`, gửi request không kèm token, và bắt buộc phải
nhận 401 — trừ những route khai tường minh trong `publicByDesign`.

Lỗi nó chặn: khai endpoint mới vào `api` thay vì `secured`. Cây route vẫn trông
hợp lý, không có dấu hiệu gì, cho tới lúc dữ liệu ra ngoài.

Muốn thêm endpoint công khai thật thì phải sửa `publicByDesign` — một dòng cố ý,
không phải một chỗ quên.

### `TestForgedRoleHeaderCannotReachAdmin`

Cùng file. Gửi `X-User-Role: admin` mà không có token, tới 5 endpoint quản trị.

Header danh tính là kênh **nội bộ** giữa gateway và service phía sau. Giá trị từ
bên ngoài phải bị xoá trước khi bất cứ thứ gì đọc chúng; nếu không thì một dòng
curl là vào được khu duyệt KYC, khoá tài khoản, xoá người dùng.

Cùng file còn `TestDriverTokenCannotReachAdmin` (token thật nhưng sai vai trò →
403), `TestSecuredRoutesRejectAnonymous`, `TestGarbageTokenRejected`.

### `TestSwaggerAnnotationsMatchRoutes`

`gateway_service/internal/delivery/http/swagger_sync_test.go`

Đối chiếu annotation `@Router` trong controller với cây route thật.

Annotation là **comment** — trình biên dịch không kiểm tra chúng. Đổi đường dẫn
trong `gateway_route.go` mà quên sửa comment thì Swagger UI vẫn chỉ đường cũ:
người dùng API gọi theo tài liệu, nhận 404, còn phía server không thấy gì bất
thường vì request đó chưa từng tới nơi.

Test so khớp theo dạng chuẩn hoá — `{id}` của swagger và `:id` của gin đều quy về
`{}`, vì tên tham số không ảnh hưởng tới định tuyến.

Kèm `TestEveryRouteIsDocumented` bắt chiều ngược lại: có route nhưng quên viết
annotation, nên không ai biết nó tồn tại.

### `TestSwaggerJSONMatchesRoutes`

`gateway_service/internal/delivery/http/swagger_freshness_test.go`

Đọc `swagger.json` đã commit và đối chiếu với cây route thật.

Khác với `TestSwaggerAnnotationsMatchRoutes` ở trên — cái đó đối chiếu *annotation
trong controller* với *route*, cả hai đều là mã nguồn nên luôn đồng bộ. Test này
nhìn vào **file đã sinh ra**, thứ chỉ cập nhật khi ai đó chạy `make swagger`.

Khoảng trống này có thật: thêm hai endpoint `/auth/refresh` và `/auth/logout` với
annotation đầy đủ, `TestSwaggerAnnotationsMatchRoutes` vẫn xanh, nhưng Swagger UI
thiếu chúng vì file chưa được sinh lại.

Kèm `TestSwaggerJSONHasNoStaleOperation` bắt chiều ngược lại: swagger mô tả một
endpoint đã bị xoá khỏi code.

### `TestCollectionCoversEveryRoute`

`tools/postman/coverage_test.go`

Đối chiếu collection Postman với cây route: **70/70 route phải được phủ**.

Kèm hai test nữa:
- `TestCollectionHasNoUnknownRoute` — collection trỏ vào route không tồn tại.
- `TestGeneratedFileIsUpToDate` — file JSON đã commit phải khớp generator. Sửa
  `tools/postman/endpoints.go` mà quên chạy `make postman` thì test đỏ.

---

## Tầng 3 — Test tích hợp

Tách khỏi unit test bằng build tag, nên `go test ./...` không chạm tới:

```go
//go:build integration
```

| File | Cần gì | Kiểm gì |
|---|---|---|
| `notification_service/.../pipeline_integration_test.go` | RabbitMQ, Postgres | Sự kiện từ MQ → thông báo trong DB, chống trùng theo `event_id` |
| `user_service/.../cache_integration_test.go` | Redis, Postgres | Cache đọc, xoá cache khi ghi |
| `vehicle_service/.../geo_integration_test.go` | Redis, Postgres | Chỉ mục GEO, truy vấn xe gần |

```bash
cd notification_service && go test -tags=integration ./... -v
cd vehicle_service      && go test -tags=integration ./... -v
cd user_service         && go test -tags=integration ./... -v
```

Vì sao tách bằng build tag thay vì tên file: build tag làm chúng **không được biên
dịch** khi chạy unit test, nên `go test ./...` vẫn nhanh và không cần hạ tầng. Đây
là khuyến nghị trong *100 Go Mistakes* (Harsanyi, #82).

---

## Tầng 4 — Test API bằng Postman

### Chuẩn bị

```bash
make auth-keys          # nếu chưa sinh khoá
docker compose up -d    # hoặc podman compose up -d
make postman            # sinh lại collection (chỉ khi sửa endpoint)
```

### Import vào Postman

1. Mở Postman → nút **Import** (góc trên bên trái).
2. Kéo thả **cả hai** file vào cửa sổ import:
   - `logistic.postman_collection.json`
   - `logistic.postman_environment.json`
3. Bấm **Import**.
4. Góc trên bên phải, đổi environment từ *No Environment* sang
   **Logistics OS — Local**. Bước này bắt buộc — collection có script chặn ở
   `prerequest` và sẽ báo lỗi rõ ràng nếu quên.

### Chạy tay

Mở `01 — Auth > Đăng nhập` rồi bấm **Send**. Script trong tab *Tests* tự lưu
`access_token`, `refresh_token`, `user_id` vào environment; từ đó mọi request khác
tự đính kèm token, không phải dán tay.

Sau đó bấm Send bất kỳ request nào. Các nhóm được đánh số theo trình tự nghiệp vụ,
và request sau dùng biến do request trước lưu lại.

### Chạy cả loạt

Bấm chuột phải lên collection → **Run collection** → **Run Logistics OS**. Chạy
tuần tự từ nhóm 00 tới 99 là đi hết một vòng nghiệp vụ: đăng ký → đăng nhập → tạo
hồ sơ → thêm địa chỉ → đăng ký xe → báo GPS → đăng đơn → ghép đơn → đọc thông báo.

Hoặc bằng dòng lệnh:

```bash
newman run logistic.postman_collection.json -e logistic.postman_environment.json
```

### Bố cục

| Nhóm | Số request | Ghi chú |
|---|---|---|
| 00 — Sức khoẻ hệ thống | 2 | health check, Swagger UI |
| 01 — Auth | 7 | công khai, chạy "Đăng nhập" trước tiên |
| 02 — User | 8 | hồ sơ người dùng, tài xế, chủ hàng, KYC |
| 03 — Sổ địa chỉ | 4 | |
| 04 — Thiết bị nhận push | 3 | |
| 05 — Phương tiện | 7 | gồm tìm xe gần theo Redis GEO |
| 06 — Giấy tờ xe | 3 | |
| 07 — Vị trí & nhận đơn | 4 | |
| 08 — Matching | 5 | Bid → Ask → Offer → chốt/từ chối |
| 09 — Thông báo | 8 | |
| 10 — Media | 2 | nhớ chọn file thật ở tab Body |
| 11 — Admin | 17 | cần `admin_access_token` |
| 99 — Kiểm thử bảo mật | 5 | **mọi request phải thất bại** |

### Mỗi request kiểm gì

Ba test chạy tự động trên **mọi** request:

1. Mã trạng thái đúng như thiết kế — kể cả các mã lỗi có chủ đích ở nhóm 99.
2. Khuôn phản hồi: thành công có `data`, lỗi có `error.code`.
3. Có header `X-Request-ID` — mã này chính là `trace_id` trên Jaeger, nên cầm nó
   là mở được cả log lẫn trace.

Một số request có test riêng, ví dụ "Đăng nhập" kiểm token là JWT ba phần,
`expires_at` nằm trong tương lai, và mật khẩu không lọt vào phản hồi.

### Nhóm 99 — các request cố tình sai

| Request | Mã mong đợi | Chứng minh điều gì |
|---|---|---|
| Header `X-User-Role: admin` giả | 401 `UNAUTHENTICATED` | Gateway xoá header danh tính client tự khai |
| Token rác | 401 | Token hỏng bị từ chối, không bị coi là khách vãng lai |
| Không token → endpoint người dùng | 401 | Mọi endpoint đều được bảo vệ |
| Token người dùng thường → `/admin` | 403 `PERMISSION_DENIED` | Phân quyền theo vai trò hoạt động |
| ID sai định dạng | 400 `INVALID_ID` | Gateway parse UUID trước khi gọi xuống gRPC |

Nếu có request nào ở nhóm này trả 200 thì đó là lỗ hổng thật.

Chạy nhóm này **sau khi** đã đăng nhập bằng tài khoản thường (không phải admin) —
hai request cuối cần một token hợp lệ nhưng thiếu quyền.

### Tài khoản admin

API công khai không tạo được tài khoản admin (`role` chỉ nhận `driver` hoặc
`shipper`). Cách tạo:

```sql
UPDATE users SET role = 'admin' WHERE email = 'admin@logistic.vn';
```

Rồi đăng nhập bằng tài khoản đó và gán token vào biến `admin_access_token`.

### Sửa collection

Collection được **sinh ra** từ `tools/postman`, không gõ tay:

```bash
make postman
```

Thêm endpoint thì sửa `tools/postman/endpoints.go` rồi chạy lệnh trên. Sửa trực
tiếp file JSON sẽ bị `TestGeneratedFileIsUpToDate` bắt.

---

## Chạy toàn bộ

```bash
# Unit + bất biến, tất cả module
go test ./pkg/... ./gateway_service/... ./auth_service/... ./user_service/... \
        ./vehicle_service/... ./notification_service/... ./matching_service/... \
        ./media_service/... ./wallet_service/... ./tools/...
```

```bash
# Kiểm mỗi module tự build được khi KHÔNG có go.work — đúng như Docker build
make verify-modules
```

`make verify-modules` bắt lỗi loại "máy tôi chạy được mà CI đỏ": `go.work` gộp
chung module graph nên ở local cái gì cũng chạy, còn Docker build với `GOWORK=off`
chỉ đọc `go.mod` + `go.sum` riêng của từng module.
