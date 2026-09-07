# Tài liệu

Mục lục toàn bộ tài liệu của repo. Bắt đầu từ
[Tổng quan hệ thống](architecture/system-overview.md).

---

## Ba loại tài liệu về hệ thống này

Ranh giới giữa chúng là câu hỏi mà tài liệu trả lời:

| Loại | Trả lời | Thư mục |
|---|---|---|
| **Kiến trúc** | Hệ thống nhìn tổng thể ra sao? | `architecture/` |
| **Service** | Bên trong MỘT tiến trình có gì? | `services/` |
| **Flow** | Một nghiệp vụ chảy qua NHIỀU tiến trình thế nào? | `flows/` |

Một API mới thuộc về `services/`. Một nghiệp vụ mới cần ba service phối hợp thuộc
về `flows/`. Nếu phân vân: cứ nhìn xem tài liệu có nhắc tới nhiều hơn một service
trong phần thân hay không.

---

## `architecture/` — Nhìn tổng thể

| Tài liệu | Nội dung |
|---|---|
| [system-overview.md](architecture/system-overview.md) | **Điểm vào**: sơ đồ tổng thể, danh sách service, cổng, bảng DB, phân tầng |
| [oauth-google-flow.md](architecture/oauth-google-flow.md) | Đăng nhập Google và cơ chế chống CSRF bằng state + cookie |
| [database-replication.md](architecture/database-replication.md) | Cấu hình master-slave cho Postgres và MySQL |
| [observability.md](architecture/observability.md) | OpenTelemetry, tracing xuyên service |

## `services/` — Bên trong một service

Mỗi tài liệu có cùng bố cục: bảng thông tin nhanh → sơ đồ → dữ liệu → quyết định
thiết kế → cấu hình.

| Service | Cổng | Tài liệu |
|---|---|---|
| `gateway_service` | 8080 | [gateway-service.md](services/gateway-service.md) |
| `auth_service` | 9001 | [auth-service.md](services/auth-service.md) |
| `media_service` | 9002 | [media-service.md](services/media-service.md) |
| `matching_service` | 9003 | [matching-service.md](services/matching-service.md) |
| `user_service` | 9004 | [user-service.md](services/user-service.md) |
| `vehicle_service` | 9005 | [vehicle-service.md](services/vehicle-service.md) |
| `notification_service` | 9006 | [notification-service.md](services/notification-service.md) |
| `wallet_service` | 9007 | [wallet-service.md](services/wallet-service.md) |

## `flows/` — Nghiệp vụ qua nhiều service

Mỗi tài liệu có cùng bố cục: tóm tắt → sơ đồ → từng bước → điều gì có thể sai.

| Tài liệu | Đi qua |
|---|---|
| [driver-onboarding-flow.md](flows/driver-onboarding-flow.md) | user · vehicle · admin |
| [shipper-order-flow.md](flows/shipper-order-flow.md) | matching · wallet · notification |
| [driver-location-flow.md](flows/driver-location-flow.md) | nginx · vehicle · Redis GEO |
| [matching-notification-flow.md](flows/matching-notification-flow.md) | matching · RabbitMQ · notification |
| [matching-scoring.md](flows/matching-scoring.md) | matching · chấm điểm ứng viên |
| [authentication-flow.md](flows/authentication-flow.md) | auth · gateway |
| [error-handling-flow.md](flows/error-handling-flow.md) | mọi service · gateway |

## `operations/` — Build, deploy, vận hành

| Tài liệu | Nội dung |
|---|---|
| [build-process.md](operations/build-process.md) | Docker multi-stage, cache layer, vì sao `GOWORK=off` |
| [devops-workflow.md](operations/devops-workflow.md) | CI/CD GitHub Actions, GHCR, deploy lên EC2 |

## `guides/` — Hướng dẫn thao tác lặp lại

| Tài liệu | Nội dung |
|---|---|
| [ent-codegen.md](guides/ent-codegen.md) | Khai báo schema và sinh code ent |
| [atlas-migration.md](guides/atlas-migration.md) | Migration có version bằng Atlas |
| [entimport-existing-db.md](guides/entimport-existing-db.md) | Sinh ent schema từ database có sẵn |

## `reference/` — Ghi chú kiến thức nền

Không riêng cho repo này. Là tài liệu học, giữ lại để tra cứu.

| Tài liệu | Nội dung |
|---|---|
| [kafka.md](reference/kafka.md) | Apache Kafka |
| [elasticsearch.md](reference/elasticsearch.md) | Elasticsearch |
| [nats-jetstream.md](reference/nats-jetstream.md) | NATS JetStream |
| [networking.md](reference/networking.md) | Mạng, TCP/IP, HTTP |
| [network-to-vpc.md](reference/network-to-vpc.md) | Lộ trình sâu: mạng trần → chia subnet → VPC → xếp EC2 |
| [aws-architecture.md](reference/aws-architecture.md) | Khái niệm kiến trúc AWS |
| [notes/go-tooling.md](reference/notes/go-tooling.md) | Ghi chú lặt vặt về Go tooling |

## `diagrams/` — Sơ đồ

**Mỗi tài liệu trong `architecture/`, `services/`, `flows/` có đúng một sơ đồ
cùng tên gốc.** Sơ đồ được nhúng thẳng vào file `.md` dưới dạng SVG, nên đọc tài
liệu là thấy hình ngay trên GitHub lẫn trong IDE.

```
docs/diagrams/<tên>.drawio       ← nguồn, mở bằng diagrams.net để chỉnh tay
docs/diagrams/svg/<tên>.svg      ← bản nhúng vào .md
docs/rendered/diagrams.html      ← xem tất cả trong một trang
```

### Hai loại sơ đồ cho hai loại tài liệu

Loại sơ đồ phải khớp với thứ mà tài liệu đang mô tả:

| Tài liệu | Loại sơ đồ | Vì sao |
|---|---|---|
| `flows/` | **Sequence diagram** (UML) | Nghiệp vụ có TRỤC THỜI GIAN: thứ tự bước, ai gọi ai, nhánh rẽ |
| `services/`, `architecture/` | **Sơ đồ thành phần** | Mô tả cấu trúc tĩnh, không có trục thời gian |

Sơ đồ hộp - mũi tên KHÔNG dùng cho flow được, vì nó làm mất ba thứ quan trọng
nhất của một luồng: thứ tự các bước, phân biệt gọi đồng bộ / trả về / bất đồng
bộ, và nhánh alt/else. Test `TestFlowsAreNotComponentDiagrams` khoá lại quyết
định này — khai báo một flow ở `diagrams()` thay vì `sequences()` là test đỏ.

### Ký hiệu trong sequence diagram

| Ký hiệu | Nghĩa |
|---|---|
| Nét liền, đầu mũi tên **đặc** | Gọi đồng bộ — bên gửi chờ kết quả |
| Nét **đứt**, đầu mũi tên mảnh | Giá trị trả về |
| Nét liền, đầu mũi tên **mảnh** | Bất đồng bộ — phát rồi đi tiếp, không chờ |
| Vòng cung về chính mình | Tính toán / kiểm tra nội bộ |
| Thanh xám dọc trên lifeline | Activation — khoảng thời gian đối tượng đang xử lý |
| Khung có tab góc trên trái | `alt` / `opt`; đường đứt bên trong ngăn nhánh else |

Message được đánh số theo đúng thứ tự thực thi. Chữ nghiêng dưới mũi tên là ghi
chú kỹ thuật, không phải một bước riêng.

### Bảng màu (sơ đồ thành phần)

client (tím nhạt) · Nginx (đỏ nhạt) · gateway (xanh dương) · service (xanh lá) ·
kho dữ liệu (cam) · broker (tím) · dịch vụ ngoài (xám) · ghi chú (vàng).

### Sinh lại

16 sơ đồ do `make diagrams` sinh từ khai báo trong `tools/diagrams/`:
`sequence.go` cho flow, `spec.go` + `services.go` cho sơ đồ thành phần.
**Đừng sửa tay file sinh ra** — sửa spec rồi chạy lại, vì cả `.drawio` lẫn `.svg`
đều đến từ một nguồn nên không thể lệch nhau.

| Nhóm | Sơ đồ |
|---|---|
| Kiến trúc | `system-overview` · `service-layering` |
| Flow (sequence) | `matching-notification-flow` · `driver-onboarding-flow` · `shipper-order-flow` · `driver-location-flow` · `authentication-flow` · `error-handling-flow` |
| Service (thành phần) | `gateway-service` · `auth-service` · `user-service` · `vehicle-service` · `matching-service` · `notification-service` · `media-service` · `wallet-service` |

### Sơ đồ vẽ tay

Không do tool sinh, chỉnh trực tiếp bằng diagrams.net:

| File | Nội dung |
|---|---|
| `system-blueprint.drawio` | Bản vẽ tổng thể toàn hệ thống |
| `system-blueprint-tabs.drawio` | Bản vẽ tổng thể, chia theo tab |
| `build-process.drawio` | Sơ đồ đi kèm [build-process.md](operations/build-process.md) |
| `backend-overview.drawio` | Tổng quan backend |
| `logistic-domain.drawio` | Mô hình miền nghiệp vụ logistics |
| `aws-architecture.drawio` | Sơ đồ đi kèm [aws-architecture.md](reference/aws-architecture.md) |

## `rendered/` — Bản HTML đọc offline

Mở thẳng bằng trình duyệt, không cần công cụ gì.

| File | Nội dung |
|---|---|
| `diagrams.html` | **Toàn bộ 16 sơ đồ trong một trang**, có mục lục — do `make diagrams` sinh |
| `elasticsearch.html` | Bản render của [reference/elasticsearch.md](reference/elasticsearch.md) |
| `aws-architecture.html` | Bản render của [reference/aws-architecture.md](reference/aws-architecture.md) |

---

## Quy ước đặt tên

Áp dụng cho mọi file trong `docs/`.

### 1. Tên file dùng `kebab-case`

```
✅  matching-notification-flow.md
✅  build-process.drawio
❌  MATCHING_NOTIFICATION_FLOW.md
❌  otel_implementation_guide.md
❌  AWS-learn.drawio
```

Lý do: Git trên Windows không phân biệt hoa thường nhưng CI trên Linux thì có —
đổi tên kiểu `Foo.md` → `foo.md` gây ra những commit "không thấy gì thay đổi" rất
khó lần. Chữ thường + gạch nối cũng chính là dạng GitHub dùng cho URL và anchor.

Ngoại lệ duy nhất: `README.md` viết hoa, vì GitHub tự nhận diện và render nó.

### 2. Các file cùng một chủ đề dùng chung tên gốc

```
docs/operations/build-process.md      ─┐
docs/diagrams/build-process.drawio    ─┴─ cùng gốc "build-process"

docs/reference/elasticsearch.md       ─┐
docs/rendered/elasticsearch.html      ─┴─ cùng gốc "elasticsearch"
```

Nhìn tên là biết file nào thuộc về nhau, không cần mở ra xem.

### 3. Đặt file theo mục đích, không theo công nghệ

| Thư mục | Trả lời câu hỏi |
|---|---|
| `architecture/` | Hệ thống **này** nhìn tổng thể ra sao? |
| `services/` | Bên trong **một** service có gì? |
| `flows/` | Một nghiệp vụ chảy qua **nhiều** service thế nào? |
| `operations/` | Làm sao build, deploy, vận hành nó? |
| `guides/` | Làm sao thực hiện thao tác X? |
| `reference/` | Công nghệ Y hoạt động thế nào? (kiến thức nền, không riêng repo) |
| `diagrams/` | Sơ đồ: `.drawio` (nguồn) và `svg/` (bản nhúng) |
| `rendered/` | Bản HTML đọc offline |

Hai ranh giới hay bị lẫn:

- **`architecture/` với `reference/`** — `reference/kafka.md` nói về Kafka nói
  chung; còn *repo này dùng Kafka làm gì* thuộc về `architecture/`.
- **`services/` với `flows/`** — nếu phần thân tài liệu nhắc tới nhiều hơn một
  service thì nó là `flows/`.

### 3b. Mỗi tài liệu kiến trúc / service / flow có một sơ đồ cùng tên

```
docs/services/vehicle-service.md        ─┐
docs/diagrams/vehicle-service.drawio    ─┼─ cùng gốc "vehicle-service"
docs/diagrams/svg/vehicle-service.svg   ─┘
```

Sơ đồ được nhúng ngay sau bảng thông tin nhanh:

```markdown
![Sơ đồ vehicle_service](../diagrams/svg/vehicle-service.svg)
```

Thêm tài liệu mới trong ba nhóm đó thì **phải** thêm một mục vào
`tools/diagrams/` — test `TestEveryFlowAndServiceHasDiagram` sẽ báo đỏ nếu thiếu.

### 4. Tiêu đề `#` đầu file mô tả nội dung, không lặp lại tên file

```
✅  # Luồng Matching → Notification (Redis + RabbitMQ)
❌  # matching-notification-flow
```

### 5. Đường dẫn tương đối khi liên kết chéo

```markdown
[Tổng quan hệ thống](../architecture/system-overview.md)   <!-- từ một file trong docs/operations/ -->
```

Nhờ vậy link chạy được cả trên GitHub lẫn khi xem file trong IDE.

### Kiểm tra tự động

```bash
make docs-lint
```

Công cụ ở `tools/doclint` kiểm tra ba thứ và trả về mã lỗi khác 0 nếu vi phạm:

1. Mọi liên kết tương đối trỏ tới file có thật (ví dụ trong code fence được bỏ qua).
2. Tên file theo kebab-case.
3. Mỗi file mở đầu bằng một tiêu đề `# `.

Chạy nó sau khi đổi tên hoặc di chuyển file tài liệu — đó chính là lúc link dễ
gãy nhất mà không ai nhận ra.

---

## Không nằm trong tài liệu repo

- `docs/programmingBook/` — thư viện sách và code luyện tập cá nhân. Đã được
  `.gitignore` bỏ qua hoàn toàn (0 file tracked), chỉ tồn tại trên máy local.
