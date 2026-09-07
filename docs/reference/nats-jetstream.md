# NATS JetStream — ghi chú kiến trúc

Tài liệu này tổng hợp các khái niệm cốt lõi của hệ sinh thái NATS, được giải thích theo ngôn ngữ bình dân và áp dụng trực tiếp vào dự án Logistics của chúng ta.

---

## 1. NATS Server & Tính năng "All-in-one"
Khác với các hệ thống cồng kềnh (Kafka cần KRaft/Zookeeper, RabbitMQ cần Erlang), NATS chỉ là **một file chạy (binary) duy nhất** siêu nhẹ (chưa tới 20MB).
Trong file chạy này, NATS tích hợp sẵn mọi thứ thông qua cấu hình:
- **Clustering:** Chạy cụm nhiều máy chủ.
- **JetStream:** Tính năng lưu trữ dữ liệu bền vững xuống ổ cứng (Persistence) để không mất tin nhắn khi sập nguồn (Giống Kafka).
- **WebSockets:** Cho phép App Frontend (React/Vue) kết nối thẳng vào NATS.
- **MQTT:** Lõi giao thức chuyên dụng cho IoT.
  - *Áp dụng:* Thiết bị định vị GPS trên xe tải có thể dùng MQTT (siêu nhẹ, tiết kiệm pin 3G/4G) để liên tục bắn tọa độ về NATS. NATS sẽ tự động dịch MQTT thành luồng tin nhắn nội bộ để Golang Backend xử lý mượt mà.

---

## 2. Multi-tenancy (Đa hệ thuê) & Accounts
NATS hỗ trợ kiến trúc Multi-tenancy thông qua khái niệm **Accounts**.
- **Ví dụ Dễ hiểu:** NATS giống như một toà chung cư. Điện, nước, thang máy (CPU, RAM) được dùng chung để tiết kiệm chi phí, nhưng mỗi khách hàng (Tenant) được cấp một căn hộ (Account) với chìa khoá riêng.
- **Tính Cách ly (Isolation):** Các Account cách ly với nhau ở tầng mạng. Công ty đối tác A (Dùng Account A) không bao giờ có thể "nghe lén" được tin nhắn nội bộ của Công ty đối tác B (Dùng Account B), dù cả 2 kết nối vào chung một cụm NATS Server của mình. Phù hợp để làm nền tảng B2B SaaS.

---

## 3. Phân cấp Thư viện (Clients Tier)
NATS hỗ trợ hàng chục ngôn ngữ lập trình, nhưng đội ngũ phát triển (Synadia) phân cấp ưu tiên cập nhật thành 3 bậc:
- **Tier 1 (Con cưng):** (Go, JS/TS, Python, Java, C#) Được cập nhật tính năng mới ngay lập tức. Trong đó, **Golang** là ngôn ngữ mẹ đẻ của NATS, được coi là *"Reference Implementation" (Bản tham chiếu chuẩn nhất)*. Dự án của chúng ta đang dùng Golang, tức là đang xài "Thượng phương bảo kiếm" mạnh và ít lỗi nhất.
- **Tier 2 (Con thứ):** (Swift, Ruby, Zig...) Do Synadia viết nhưng ưu tiên thấp, thường chậm cập nhật tính năng mới vài tháng.
- **Community:** Thư viện trôi nổi do cộng đồng Github tự viết, hên xui.

---

## 4. Orbit (Kho "đồ chơi" mở rộng)
Orbit (như `synadia-io/orbit.go`) là một kho chứa các tiện ích nâng cao mang tính **thử nghiệm (experimental)**, nằm ngoài thư viện lõi `nats.go`.
- **Ví dụ dễ hiểu:** `nats.go` là chiếc xe hơi bản tiêu chuẩn cực kỳ ổn định. Quán độ xe "Orbit" bán thêm các đồ chơi như: Hẹn giờ gửi tin nhắn (Scheduled messages), Đóng gói tin nhắn gửi 1 lần (Batch publish).
- Nếu các tính năng này chạy tốt và được cộng đồng tin dùng, chúng sẽ được "Tốt nghiệp" và sáp nhập vào bản chính thức. Khuyên dùng khi thực sự cần giải quyết các bài toán hóc búa.

---

## 5. Command-line Tooling (Đồ nghề dòng lệnh)
Bộ công cụ để dân Devops/Backend quản trị NATS Server:
- **`natscli` (`nats`):** "Cờ lê vạn năng" giống lệnh `kubectl` hay `docker`. Dùng để test bắn tin (publish), đọc tin (subscribe), quản lý nhà kho JetStream.
- **`nats-top`:** Dashboard chạy ngay trên màn hình Terminal. Giống lệnh `top` của Linux, dùng để soi lượng RAM, CPU, số kết nối và tốc độ tin nhắn của NATS theo thời gian thực.
- **`nats-box`:** Một file Docker Image nhỏ xíu chứa sẵn toàn bộ đồ nghề trên. Chuyên dùng quăng vào Kubernetes Cluster để chui vào gõ lệnh debug khi server có biến.

---

## 6. Identity & Authentication (Hệ thống Bảo mật)
NATS **KHÔNG DÙNG** username/password truyền thống lưu trong Database để tránh làm chậm hệ thống. NATS bảo mật bằng cơ chế mã hoá phi đối xứng (Public-key cryptography) thông qua NKeys và JWT. Hệ thống này hoạt động giống như một chiếc "Thẻ từ kỹ thuật số" ở các chung cư cao cấp:

### A. NKey là gì?
- **NKey** là một hệ thống khoá công khai/bí mật (Public/Private Key) dựa trên thuật toán mã hoá cực mạnh `Ed25519`, do chính NATS tự chế tạo lại để an toàn và dễ đọc hơn.
- Nhìn nó khá giống với SSH Key, nhưng thông minh hơn ở chỗ NKey có chứa các **Ký tự tiền tố (Prefix)**. Nhìn vào một chuỗi NKey public, nhận ra ngay nó dành cho ai:
  - Bắt đầu bằng chữ `O` (ví dụ: `O...`): Khoá của **Operator** (Trùm cuối - Người quản trị cả cụm NATS).
  - Bắt đầu bằng chữ `A` (ví dụ: `A...`): Khoá của **Account** (Công ty/Phòng ban thuê nhà).
  - Bắt đầu bằng chữ `U` (ví dụ: `U...`): Khoá của **User** (Tài xế/Nhân viên - Client thực tế kết nối vào NATS).
- **Tuyệt mật:** Khoá bí mật (Private Key / Seed) bắt đầu bằng chữ `S`, phải giữ kín không bao giờ được gửi qua mạng.

### B. JWT (JSON Web Token)
- Thay vì bắt Client gửi thẳng khoá bí mật qua mạng (rất nguy hiểm), NATS bắt Client tạo ra một cái Thẻ JWT.
- Bên trong Thẻ JWT này chứa các **Quyền (Claims)** (ví dụ: User này chỉ được phép Publish vào topic `vehicle.location`, bị giới hạn 50 tin nhắn/giây). Cái thẻ này được ký xác nhận bằng chính NKey của Account cha.
- Khi Client mang thẻ JWT này cắm vào NATS Server, Server chỉ việc đối chiếu chữ ký mã hoá. Nếu khớp -> Cho qua. Tốc độ xác thực cực kỳ nhanh vì Server **không cần truy vấn Database**.

### C. Công cụ và Thư viện hỗ trợ
Để quản lý mớ NKey và JWT này, hệ sinh thái NATS cung cấp sẵn đồ chơi cho người dùng:
- **`nsc` (NATS Security CLI):** Phần mềm độc lập cài trên máy trạm, đóng vai trò như cái "Máy ghi thẻ từ". Dùng nó để gõ lệnh tạo ra NKeys mới, đóng dấu JWT cho các User, sau đó ném thẻ đó cho Frontend hoặc App tải về xài.
- **Thư viện JWT/NKey Libraries:** nếu không muốn gõ lệnh bằng tay mà muốn viết một hệ thống tự động sinh ra thẻ từ cho người dùng (để User bấm nút đăng ký là có ngay thẻ), có thể dùng code (ví dụ `nats-io/jwt` và `nats-io/nkeys` trong Golang).
- **Auth Callout (`synadia-io/callout.go`):** Tính năng nâng cao, dùng khi không muốn dùng JWT của NATS mà muốn bắt NATS Server phải gọi sang cái hệ thống Auth (Google, Facebook, OAuth2) có sẵn của mình để kiểm tra. Thư viện này hỗ trợ viết API trả lời truy vấn đó bằng Golang.

---

## 7. Kubernetes (Triển khai Hạ tầng)
NATS hỗ trợ cực tốt cho môi trường Cloud-native và Kubernetes (K8s):
- **`k8s` (Helm Charts):** Bộ cấu hình chuẩn DevOps do NATS viết sẵn. Chỉ cần chạy 1 dòng lệnh Helm là nó tự động dựng nguyên một cụm NATS Cluster (với hàng chục node) lên K8s cực mượt.
- **`nack` (NATS Controller for K8s):** Thay vì phải tự gõ lệnh `nats` để tạo Stream hay Consumer, chỉ cần viết file cấu hình YAML. `nack` sẽ tự động đọc file YAML đó và gỡ/tạo tài nguyên trên NATS (Chuẩn GitOps/Declarative).

---

## 8. Observability (Giám sát & Đo lường)
Hệ thống chạy trên production thì không thể mù tịt được. Cần biết nó đang ngốn bao nhiêu RAM, chạy bao nhiêu Msg/sec.
- **`prometheus-nats-exporter` & `nats-surveyor`:** Hai công cụ này giống như bác sĩ khám sức khỏe. Chúng liên tục cào các "chỉ số sinh tồn" (Metrics) của NATS Server rồi xuất ra định dạng chuẩn của **Prometheus**. Từ đó, có thể cắm **Grafana** vào để vẽ biểu đồ theo dõi tuyệt đẹp.

---

## 9. Bridges & Integrations (Cầu nối Hệ sinh thái)
NATS không hề chơi một mình mà có sẵn các "Cây cầu" để nối sang các công nghệ dữ liệu lớn (Big Data) khác:
- **`nats-kafka` (Cực kỳ quan trọng với dự án ta):** Đây là chiếc cầu nối 2 chiều giữa NATS và Kafka. nếu muốn bắn tin nhắn vào NATS cho lẹ, nhưng lại muốn lưu trữ lâu dài bên Kafka (để đẩy sang Elasticsearch) thì dùng cái Bridge này. Nó sẽ tự động copy tin nhắn qua lại mà **không cần tự code**.
- **Spark / Flink Connectors:** Dành cho đội ngũ Data Engineer hút dữ liệu Real-time từ NATS ra để làm AI/Machine Learning hoặc xử lý luồng (Stream Processing) cực nặng.
- **`terraform-provider-jetstream`:** Cho phép dân DevOps dùng mã nguồn (Infrastructure as Code) để tự động hóa việc tạo Stream, tạo Account trên NATS thay vì bấm tay.
