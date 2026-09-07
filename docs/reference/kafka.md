# Apache Kafka

## 1. Thành Phần Cấu Trúc (Architecture Components)

### 1.1 Cấu tạo của một Kafka Message (Anatomy of a Message)
Mỗi tin nhắn (Message/Record) được Producer tạo ra và gửi đi bao gồm các phần cốt lõi sau:
- **Key (Binary - Optional):** Dùng để định tuyến (Routing). Mọi message có cùng Key chắc chắn sẽ bay vào cùng một Partition. (Ví dụ: `BidID`).
- **Value (Binary - Optional):** Nội dung thực sự của tin nhắn (thường là chuỗi JSON hoặc Protobuf đã bị nén thành mảng bytes). Có thể để `null` (thường dùng trong khái niệm Tombstone để xóa dữ liệu).
- **Compression Type:** Loại nén dữ liệu (None, Gzip, Snappy, LZ4, Zstd). Giúp giảm băng thông mạng và dung lượng lưu trữ trên đĩa.
- **Headers (Optional):** Chứa các cặp Key-Value siêu dữ liệu (metadata) giống như HTTP Header (Ví dụ: Chứa Trace ID để phục vụ Distributed Tracing).
- **Partition + Offset:** Thông tin này do Broker gắn vào sau khi tin nhắn được ghi xuống đĩa thành công.
- **Timestamp (System or User set):** Thời gian tạo ra tin nhắn (do Producer set) hoặc thời gian tin nhắn được Broker ghi nhận (LogAppendTime).

### 1.2 Tuần tự hóa (Serialization) và Giải tuần tự hóa (Deserialization)
Bản thân Kafka Broker là những "gã mù mờ", nó **CHỈ** nhìn thấy và lưu trữ mảng Bytes (Binary raw). Nó không hề hiểu Object (như `Struct` trong Go hay `Class` trong Java) là gì.

- **Serialization (Ở Producer):** Là quá trình biến đổi Object thành chuỗi Bytes. 
  - Ví dụ: `Key Object (123) -> IntegerSerializer -> Bytes`.
  - `Value Object ("hello world") -> StringSerializer -> Bytes`.
- **Deserialization (Ở Consumer):** Là quá trình biến chuỗi Bytes ngược lại thành Object để ứng dụng đọc được. (Ví dụ dùng `IntegerDeserializer` cho Key và `StringDeserializer` cho Value).
- **Common Serializers/Deserializers:** String (bao gồm JSON), Int, Float, Avro, Protobuf. 

**Câu hỏi: Làm sao Serializer và Deserializer "hiểu" được type của Key và Data?**
1. **Thực tế phũ phàng:** Kafka Broker không lưu và không truyền bất kỳ metadata nào về Type của Data (nếu không dùng Header). Nên bản thân Deserializer **KHÔNG THỂ TỰ HIỂU**.
2. **Thỏa thuận ngầm (Contract):** Lập trình viên ở 2 đầu (Producer và Consumer) phải **tự cấu hình cứng (hardcode) khớp với nhau**. Nếu Producer dùng `StringSerializer` nhưng Consumer lại ngây thơ dùng `IntegerDeserializer` để đọc, Consumer sẽ bị văng lỗi (Crash / SerializationException) ngay lập tức.
3. **Quy tắc bất di bất dịch (Topic Lifecycle):** Bạn KHÔNG BAO GIỜ được phép thay đổi kiểu Serialization trong vòng đời của một Topic. Nếu ngày xưa dùng JSON, bây giờ muốn chuyển sang Protobuf, **hãy tạo một Topic hoàn toàn mới**. Cố tình thay đổi sẽ làm sập toàn bộ các Consumer đang chạy.
4. *(Nâng cao: Để giải quyết triệt để bài toán này ở quy mô tập đoàn, người ta dùng thêm hệ thống **Schema Registry** (vd: Confluent Schema Registry). Producer sẽ gửi ID của Schema kèm với mảng Bytes, Consumer đọc Bytes, lấy ID đi hỏi Schema Registry để tải cấu trúc (Schema) về rồi mới Deserialize).*

### 1.3 Kafka Brokers & Cluster (Máy chủ và Cụm)
- **Bản chất vật lý:** Là một tiến trình (process) Java chạy trên một Server/Pod. Một **Kafka Cluster (Cụm Kafka)** được tạo thành từ nhiều Broker gộp lại.
- **Định danh (ID):** Mỗi Broker trong cụm được định danh bằng một số nguyên (Integer ID), ví dụ Broker 101, 102, 103 (con số thường chọn ngẫu nhiên bắt đầu từ 100 cho dễ phân biệt).
- **Quy mô (Scale):** Để chạy Production an toàn (có tính Availability cao), con số khởi điểm lý tưởng là **3 Brokers**. Các cụm Kafka khổng lồ của các tập đoàn có thể lên tới hơn 100 Brokers.

**A. Cơ chế kết nối thông minh (Broker Discovery / Bootstrap Server)**
- Mọi Broker trong cụm Kafka đều đóng vai trò là một **"Bootstrap Server"**.
- Điểm làm nên tên tuổi của Kafka Client (Producer/Consumer) là nó rất **Thông minh (Smart Clients)**:
  - Bạn **CHỈ CẦN** cung cấp IP của 1 Broker duy nhất (ví dụ Broker 101) vào cấu hình kết nối.
  - Khi Client kết nối vào Broker 101 và gửi `Metadata Request`, Broker 101 (do nắm toàn bộ thông tin của Cluster) sẽ trả về danh sách tất cả các Broker còn lại, cùng với sơ đồ vị trí của các Topic/Partition.
  - Sau đó Client sẽ tự động biết cách mở kết nối ngang hàng tới các Broker chứa đúng Partition mà nó cần đọc/ghi (Broker 102, 103...).

**B. Sự phân bố dữ liệu (Brokers and Topics)**
- Các Partition của một Topic sẽ được phân tán (distributed) rải rác trên các Broker.
- **Nguyên tắc cốt lõi:** Một Broker **CHỈ** nắm giữ dữ liệu mà nó được phân công, nó không bắt buộc phải chứa toàn bộ Topic của hệ thống.
  - Ví dụ: `Topic-A` có 3 Partitions -> Có thể chia đều cho Broker 101 (P0), Broker 102 (P1), Broker 103 (P2).
  - Nhưng `Topic-B` chỉ có 2 Partitions -> Nó có thể chỉ nằm ở Broker 101 (P1) và Broker 102 (P0). Lúc này Broker 103 sẽ **không chứa một giọt data nào** của Topic-B cả (chuyện này hết sức bình thường).

**C. Tại sao Kafka lại siêu tốc dù ghi xuống ổ cứng?**
- Khác với Database thông thường ghi ngẫu nhiên (Random Access), Kafka ghi dữ liệu theo kiểu **Sequential I/O** (Ghi tuần tự vào đít file). Tốc độ ghi tuần tự của ổ cứng gần tương đương với tốc độ RAM.
- Sử dụng **Zero-Copy (sendfile system call)**: Data đi thẳng từ Disk Buffer qua Network Socket bay ra ngoài mạng mà không cần copy ngược lên User Space (Application memory), tiết kiệm cực kỳ nhiều CPU.

**D. Zookeeper vs KRaft Mode (KIP-500)**
- *Cũ (Zookeeper):* Phải chạy một cụm Zookeeper riêng biệt để lưu trữ metadata.
  - Nhược điểm chí mạng: Nút thắt cổ chai (Scaling issues) khi cụm Kafka vượt quá 100,000 Partitions.
- *Mới (KRaft - Kafka Raft Architecture):* Khởi động từ 2020, Kafka tự chạy giao thức đồng thuận Raft để bầu ra Quorum Controller, chính thức gỡ bỏ hoàn toàn sự phụ thuộc vào Zookeeper.
  - **Lợi ích khổng lồ:** Khả năng scale lên **hàng triệu Partitions**. Giảm thời gian shutdown và recovery. Quản lý chung một hệ thống bảo mật (Single security model). Khởi chạy chỉ với 1 process duy nhất (nhẹ nhàng hơn).
  - **Lộ trình:** Production-ready từ bản 3.3.1. Kể từ **Kafka 4.0**, KRaft là lựa chọn DUY NHẤT (Zookeeper bị khai tử hoàn toàn).

### 1.4 Topic, Partition & Offset (Luồng dữ liệu cốt lõi)

**A. Topic (Chủ đề dữ liệu - Logic)**
- **Định nghĩa:** Topic đại diện cho một luồng dữ liệu (data stream) cụ thể. Nó khá giống với khái niệm "Table" (Bảng) trong Database, nhưng không hề có ràng buộc (constraints) hay schema khắt khe.
- **Đặc tính:** 
  - Bạn có thể tạo vô số Topic trong một Cluster Kafka (VD: `logs`, `purchases`, `trucks_gps`).
  - Nhận mọi định dạng dữ liệu (JSON, Avro, Text...).
  - **Không thể Query:** Khác với Database có thể dùng lệnh `SELECT * WHERE`, bạn không thể query trực tiếp vào Topic. Cách duy nhất để lấy dữ liệu là dùng Kafka Consumer để đọc (đọc tuần tự như đọc băng cassette).

**B. Partition (Phân mảnh - Vật lý)**
- Mỗi Topic sẽ được chia nhỏ thành nhiều Partition (VD: Topic `trucks_gps` có thể chia làm 100 partitions).
- **Mục đích:** Để phân tán dữ liệu ra nhiều Server/Broker khác nhau, giúp tăng tốc độ ghi và đọc song song (Parallelism) lên mức tối đa. Bạn có thể có bao nhiêu Partition tùy thích.
- **Phân phối:** Nếu tin nhắn không có Key, dữ liệu sẽ được Kafka rải ngẫu nhiên (randomly assigned) vào các Partition.

**C. Offset (Chỉ số định danh)**
- Bên trong mỗi Partition, mỗi khi có một tin nhắn mới được ghi vào, nó sẽ được cấp một số ID tăng dần liên tục, gọi là **Offset** (Bắt đầu từ 0, 1, 2, 3...).
- **Những lưu ý SỐNG CÒN về Offset & Partition:**
  1. **Tính Bất Biến (Immutability):** Một khi dữ liệu đã được ghi vào một Partition, nó **KHÔNG THỂ BỊ THAY ĐỔI** (không có lệnh UPDATE hay DELETE từng dòng như SQL).
  2. **Thứ Tự (Ordering):** Thứ tự của tin nhắn **CHỈ ĐƯỢC ĐẢM BẢO TRONG CÙNG 1 PARTITION**. Hoàn toàn không có khái niệm thứ tự global trên toàn bộ Topic.
  3. **Độc lập Offset:** Offset số `3` ở Partition `0` mang dữ liệu hoàn toàn khác biệt với Offset số `3` ở Partition `1`. Offset chỉ có ý nghĩa khi đi kèm với một Partition cụ thể.
  4. **Không Tái Sử Dụng:** Kể cả khi các tin nhắn cũ bị xóa (do hết hạn Retention 7 ngày), Kafka vẫn tiếp tục tăng mã Offset lên (Ví dụ từ 1000 lên 1001), các số Offset cũ đã xóa sẽ không bao giờ được tái sử dụng.

### 1.5 Replication Factor, Leader & ISR (Nhân bản & Chống sập)

**A. Phân biệt Rạch ròi: Partition vs Replication Factor (Cực kỳ dễ nhầm lẫn)**
Rất nhiều người mới học lầm tưởng "Partition 1 là bản copy của Partition 0". **ĐIỀU NÀY LÀ SAI HOÀN TOÀN.**
- **Partition (Phân mảnh - Scale Horizontal):** Dùng để CHIA NHỎ dữ liệu. Ví dụ Topic có 100GB, chia làm 2 Partition thì P0 chứa 50GB data đầu, P1 chứa 50GB data sau (2 data này hoàn toàn khác nhau). Mục đích là để nhiều Consumer cùng đọc song song cho nhanh.
- **Replication Factor (Nhân bản - Persistence/Durability):** Dùng để COPY nguyên si dữ liệu nhằm chống mất mát (phòng hờ server cháy). Nếu Replication Factor = 2, thì toàn bộ 50GB của P0 sẽ được nhân bản thành một "P0 Replica" nằm ở một Broker khác. **Tức là bản sao của Partition 0 CHÍNH LÀ Partition 0, chứ KHÔNG PHẢI là Partition 1.**

**B. Nguyên lý Nhân bản (Replication) & Độ bền (Durability)**
- Một Topic LUÔN nên thiết lập Replication Factor > 1 (thông thường là 2 hoặc 3, chuẩn công nghiệp là 3).
- **Mục đích (Failover):** Giả sử P0 có bản chính ở Broker 101 và bản sao ở Broker 102. Nếu Broker 102 đột ngột chết (mất điện), Broker 101 và 103 vẫn đứng ra gánh vác việc phục vụ data bình thường, hệ thống không hề có downtime.
- **Công thức tính độ bền (Rule of thumb):** Với Replication Factor là `N`, hệ thống có thể chịu đựng được việc mất đi vĩnh viễn tối đa `N - 1` Broker mà không bị mất dữ liệu. (VD: Replication Factor = 3 thì có thể chết vĩnh viễn 2 Broker cùng lúc vẫn an toàn).

**C. Concept of Leader (Ai làm đại ca?)**
- Tại bất kỳ thời điểm nào, đối với một Partition cụ thể (ví dụ P0), **CHỈ CÓ DUY NHẤT 1 Broker** được nắm chức **Leader**.
- Các Broker chứa bản sao còn lại được gọi là **Follower** (hoặc Replica).
- **Quy tắc Đọc/Ghi (Behavior):**
  - Kafka Producers **CHỈ** được phép bắn dữ liệu vào Broker đang làm Leader của Partition đó.
  - Các Follower đứng ngoài rìa hì hục copy (replicate) data từ Leader về.
  - Kafka Consumers **theo mặc định** sẽ chỉ kéo dữ liệu từ Broker đang làm Leader.
  - *(Ngoại lệ)* **Consumer Replica Fetching (Từ Kafka v2.4+):** Giờ đây bạn có thể cấu hình cho Consumer đọc dữ liệu từ một Follower (ISR) gần nó nhất thay vì Leader. Điều này giúp giảm thiểu độ trễ (latency) và tiết kiệm cực nhiều chi phí mạng (network costs) nếu chạy trên môi trường Cloud đắt đỏ (cross-AZ data transfer).

**D. ISR (In-Sync Replicas - Bản sao đồng bộ)**
- Bất kỳ Follower nào đang sao chép dữ liệu và bắt kịp tốc độ của Leader (không bị trễ quá thời gian `replica.lag.time.max.ms`) thì được liệt kê vào danh sách **ISR**.
- Nếu một Follower bị chậm (do đứt cáp, ổ cứng bad sector...), nó sẽ bị Leader đá văng khỏi danh sách ISR.
- Nếu Leader chết đột ngột, Kafka **CHỈ ĐỀ CỬ** những Follower đang nằm trong danh sách ISR lên làm Leader mới để đảm bảo dữ liệu không bị thất thoát.

### 1.6 Kỷ nguyên KRaft (Kafka Raft) & Vai trò cốt lõi của Controller
Nếu bạn học Kafka qua các tài liệu cũ (trước 2022), bạn sẽ quen với việc Kafka luôn phải đi kèm với một hệ thống quản lý phân tán tên là **Zookeeper**. Zookeeper đóng vai trò là "bộ não", chịu trách nhiệm quản lý toàn bộ Metadata, bầu chọn Leader và theo dõi vòng đời của các Broker. Tuy nhiên, việc phải duy trì hai hệ thống độc lập (Kafka và Zookeeper) tạo ra nút thắt cổ chai về hiệu suất và gia tăng độ phức tạp trong khâu vận hành.

Từ phiên bản Kafka 3.3 trở đi, kiến trúc **KRaft (Kafka Raft)** chính thức ra đời để **khai tử Zookeeper**. Bộ não quản lý giờ đây được nhúng trực tiếp vào bên trong nội tại của các Node Kafka.

**A. Vai trò thực sự của Controller**
- **Khái niệm:** Trong cụm Kafka, Controller là một Node đặc biệt được giao trọng trách quản lý trạng thái toàn cục (Global Metadata) của cả hệ thống.
- **Nhiệm vụ cụ thể:**
  - Quản lý vòng đời Broker: Nhận diện ngay lập tức khi một Broker gia nhập (Join) hoặc rời khỏi (Crash/Leave) cụm.
  - Quản trị Topology: Ghi nhận sự tạo mới, thay đổi cấu hình, hoặc xóa Topic/Partition. Quyết định Broker nào sẽ đảm nhận vai trò Leader cho một Partition cụ thể.
  - Phân phối Metadata: Đồng bộ (Broadcast) trạng thái mới nhất của hệ thống đến tất cả các Broker còn lại, đảm bảo toàn cụm có chung một nhận thức về không gian dữ liệu.

**B. Cơ chế hoạt động kép (Dual-Role: Broker & Controller)**
Khi một Node được cấu hình `KAFKA_PROCESS_ROLES=broker,controller`, nó sẽ thực thi đồng thời hai vai trò hoàn toàn độc lập trên cùng một tiến trình bộ nhớ (JVM Process):
- **Luồng Broker (Data Plane):** Chịu trách nhiệm xử lý trực tiếp lưu lượng đọc/ghi (Produce/Consume) các tin nhắn thực tế từ Client. Dữ liệu này được tuần tự hóa và ghi xuống đĩa thông qua các thư mục log tiêu chuẩn.
- **Luồng Controller (Control Plane):** Vận hành một cỗ máy trạng thái (State Machine) độc lập để quản lý Metadata. Toàn bộ các sự kiện thay đổi cấu trúc (ví dụ: "Broker 2 vừa mất kết nối") đều được Controller tuần tự hóa và ghi vào một Topic nội bộ cực kỳ đặc biệt mang tên `__cluster_metadata` (log này chỉ có **đúng 1 partition** — xem 1.7).

**C. Thuật toán Đồng thuận Raft & Bài toán Bầu cử (Leader Election)**
- **Khái niệm:** Raft là một thuật toán đồng thuận (Consensus Algorithm) được thiết kế để giải quyết bài toán cốt lõi của hệ thống phân tán: Làm thế nào để một nhóm các máy chủ độc lập có thể đạt được sự nhất trí tuyệt đối về một trạng thái duy nhất, ngay cả khi hệ thống mạng có thể bị lỗi.
- **Epoch (Kỷ nguyên):** Raft quản lý thời gian bằng khái niệm Epoch (Kỷ nguyên). Mỗi khi một cuộc bầu cử mới diễn ra, mã số Epoch sẽ tự động tăng lên (Ví dụ: Chuyển từ Epoch 1 sang Epoch 2). Mọi quyết định hoặc mệnh lệnh mang nhãn Epoch cũ sẽ lập tức bị hệ thống từ chối. Đây là chốt chặn hoàn hảo để triệt tiêu hiện tượng "Split-Brain" (Phân liệt não - Trạng thái nguy hiểm khi hai Node cùng huyễn hoặc bản thân mình là Controller).

**D. Phân tích thực tế: Vòng đời Bầu cử trong Cụm 3 Nodes**

**Bước 1: Trạng thái Vận hành Ổn định (Normal State)**
- Hệ thống gồm 3 Node: `kafka-1`, `kafka-2`, `kafka-3`. Kích thước Quorum là 3. Để thông qua bất kỳ quyết định nào, hệ thống cần đạt được đa số tối thiểu là 2 phiếu (**Q = ⌊N/2⌋ + 1**, tức chia lấy phần nguyên rồi cộng 1 — với N=3 ra 2, với N=5 ra 3).
- Giả định qua quá trình khởi tạo, `kafka-1` đắc cử làm **Active Controller (Kỷ nguyên - Epoch 1)**.
- **Phân vai:** `kafka-1` trực tiếp ghi nhận các thay đổi vào Topic `__cluster_metadata`. Trong khi đó, `kafka-2` và `kafka-3` đóng vai trò là Standby Controllers. Chức năng duy nhất của chúng lúc này là liên tục "kéo" (fetch) dữ liệu từ Topic `__cluster_metadata` của `kafka-1` để sao chép y hệt trạng thái hệ thống, đồng thời định kỳ gửi tín hiệu nhịp tim (Heartbeat) báo cáo sự tồn tại.

**Bước 2: Sự cố Controller (Active Controller Crash)**
- Biến cố xảy ra: `kafka-1` đột ngột mất điện hoặc bị lỗi tiến trình (Kernel Panic), dẫn đến mất kết nối hoàn toàn.
- Lúc này, `kafka-2` và `kafka-3` không còn nhận được Heartbeat từ `kafka-1`. Thời gian đếm ngược (Election Timeout) bắt đầu tích lũy.

**Bước 3: Hội thoại Đồng thuận & Chuyển giao Quyền lực (Consensus & Election)**
- `kafka-2` có bộ đếm Timeout kết thúc trước. Nó lập tức chuyển trạng thái sang **Candidate (Ứng cử viên)**.
- `kafka-2` tự động tăng mã kỷ nguyên lên thành **Epoch 2**. Nó gửi bản tin "Yêu cầu Bầu phiếu" (RequestVote) sang `kafka-3` với thông điệp: "Tôi đang giữ bản ghi Metadata hoàn chỉnh nhất của Epoch 1, hãy bầu cho tôi".
- `kafka-3` nhận được yêu cầu, tiến hành đối chiếu. Nhận thấy thông điệp hợp lệ, số Epoch mới (2) lớn hơn Epoch hiện tại (1), và bản ghi Metadata của `kafka-2` không hề cũ hơn của mình, `kafka-3` phản hồi "Tán thành" (Vote Granted).
- Cộng thêm lá phiếu tự bầu của chính mình, `kafka-2` thu thập đủ 2/3 phiếu (Đạt chuẩn Quorum). Nó chính thức thăng cấp thành **Active Controller của Epoch 2**.
- Ngay lập tức, `kafka-2` phát đi tín hiệu (Broadcast) toàn mạng lưới để xác lập quyền lực. Hệ thống tiếp tục vận hành liền mạch dưới sự điều phối của Controller mới.

**Bước 4: Bóng ma Trở về (Zombie Node & Ngăn chặn Split-Brain)**
- Chuyện gì sẽ xảy ra nếu `kafka-1` được cấp điện lại và kết nối thành công vào mạng lưới?
- Do bộ nhớ tạm (Volatile Memory) đã mất, trí nhớ của `kafka-1` dựa vào dữ liệu trên đĩa, vẫn dừng lại ở quá khứ. Nó hoàn toàn không biết sự tồn tại của Epoch 2. Nó "ngây thơ" cho rằng mình vẫn là Controller của **Epoch 1**, và cố gắng phát ra các mệnh lệnh quản trị đến các Node khác.
- Tuy nhiên, khi mệnh lệnh mang nhãn "Epoch 1" của `kafka-1` chạm đến `kafka-2` hoặc `kafka-3`, chúng lập tức kiểm tra và từ chối thẳng thừng, kèm theo phản hồi: "Kỷ nguyên hiện tại đã là **Epoch 2**, mệnh lệnh của bạn đã lỗi thời và không còn giá trị!".
- Khi `kafka-1` nhận được phản hồi chứa nhãn Epoch 2 (lớn hơn Epoch 1 của nó), nó ngay lập tức "bừng tỉnh" và nhận ra bản thân đã bị phế truất.
- Cơ chế tự sửa sai kích hoạt: `kafka-1` lập tức tự phế truất, giáng cấp xuống làm Standby Controller, xóa bỏ các trạng thái tạm thời đang xử lý dở dang của Epoch 1, và ngoan ngoãn gửi Request xin sao chép dữ liệu (Fetch Metadata) từ `kafka-2` để cập nhật nhận thức theo thời đại mới.
- **Kết luận:** Nhờ cơ chế quản lý **Epoch** cực kỳ khắt khe này, kiến trúc KRaft miễn nhiễm hoàn toàn với các lỗi Split-Brain, đảm bảo tính nhất quán tuyệt đối (Strict Consistency) của dữ liệu quản trị trên toàn cụm.

**Giải thích các thông số cấu hình KRaft (Trên Docker Compose):**
Để thiết lập một cụm KRaft chuẩn chỉ, bạn cần nắm vững các biến môi trường sau:

1. **`KAFKA_PROCESS_ROLES: broker,controller`**
   - *Ý nghĩa:* Cho phép Node này "chơi 2 vai". Vừa làm Broker (chứa data thực tế của Topic), vừa làm Controller (Nằm trong hội đồng bầu cử quản lý Cluster). Nếu cụm rất lớn, người ta có thể tách riêng Node chỉ làm `broker`, Node chỉ làm `controller`.
2. **`KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka-1:9093,2@kafka-2:9093,3@kafka-3:9093`**
   - *Ý nghĩa:* Khai báo danh sách "Thành viên Hội đồng Bầu cử". Các Node dùng Port 9093 để giao tiếp, bỏ phiếu Raft và sao chép Metadata.
   - *Tại sao chỉ có 1 Active mà phải cấu hình cả 3?* Trong cơ chế Raft, một cuộc bầu cử chỉ hợp lệ khi đạt đủ số phiếu quá bán (Quorum = ⌊N/2⌋ + 1). Bắt buộc mọi Node phải biết **ngay từ lúc khởi động** danh sách chính xác của toàn bộ thành viên hội đồng. Nếu không khai báo trước cả 3, khi một Node mất tín hiệu Leader, nó sẽ không biết phải gửi yêu cầu xin phiếu (RequestVote) cho ai, và cũng không biết làm thế nào để đo lường mức độ quá bán. Đây chính là "Bản hiến pháp" quy định cấu trúc quyền lực của cụm.
3. **Mạng lưới Listeners & Bảo mật (Cực kỳ quan trọng):**
   - `KAFKA_LISTENERS`: Xác định các Port mà Kafka sẽ MỞ ra để lắng nghe. (Ví dụ: Mở port 9093 cho Hội đồng Controller nói chuyện, mở port 29092 cho các Broker chép data nội bộ, mở port 9092 cho App bên ngoài gọi vào).
   - `KAFKA_ADVERTISED_LISTENERS`: Là địa chỉ (Hostname/IP) mà Kafka sẽ "hét lên" cho Client biết để kết nối. (Ví dụ: "Ê App ơi, muốn kết nối tao thì gọi địa chỉ `localhost:9092` nhé").
   - `KAFKA_LISTENER_SECURITY_PROTOCOL_MAP`: Định nghĩa giao thức bảo mật cho từng luồng mạng.
     - **PLAINTEXT:** Dữ liệu không mã hóa. CHỈ DÙNG trong Local Dev hoặc mạng nội bộ (VPC) cách ly hoàn toàn với Internet.
     - **SSL / SASL_SSL:** Mã hóa đường truyền (TLS) và yêu cầu xác thực (SASL). **BẮT BUỘC PHẢI DÙNG** trên Production, đặc biệt khi Client kết nối qua mạng Internet/Public. Tuy nhiên, việc setup SSL trong Kafka khá vất vả vì phải tạo và quản lý Truststore/Keystore cho từng Node, nên khi học hoặc test cục bộ ta thường dùng PLAINTEXT để bỏ qua rào cản này.
4. **Hệ số nhân bản hệ thống (System Replication Factors):**
   - `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR`: Hệ số nhân bản cho Topic ẩn `__consumer_offsets` (Nơi lưu điểm nhớ của Consumer). Nếu cấu hình cụm 3 Node, phải set bằng 3 để chống sập.
   - `KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR` & `KAFKA_TRANSACTION_STATE_LOG_MIN_ISR`: Dùng cho tính năng Exactly-Once (Giao dịch). Tương tự, cấu hình cụm 3 Node thì set RF=3, MIN_ISR=2.
5. **`KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS` (Độ trễ Rebalance ban đầu):**
   - *Vấn đề:* Khi bạn khởi động một Group mới (ví dụ App của bạn scale ra 3 Pods Consumer), các Pod sẽ không khởi động xong cùng một mili-giây mà sẽ kết nối vào Kafka rải rác từng thằng một. Nếu Kafka không đợi, ngay khi Pod 1 kết nối, nó chia toàn bộ Partition cho Pod 1. Vài mili-giây sau Pod 2 kết nối, nó lại ngắt Pod 1 kích hoạt Rebalance thu hồi lại để chia đôi. Pod 3 kết nối, lại Rebalance lần 3. Quá trình khởi động sẽ tạo ra một cơn bão Rebalance (Rebalance Storm) vô nghĩa, cực kỳ tốn CPU và gián đoạn hệ thống.
   - *Giải pháp:* Tham số này (trong Production thường cấu hình 3000ms = 3 giây) chỉ thị cho Coordinator: "Khoan đã, hãy đợi 3 giây kể từ khi thằng Consumer đầu tiên xuất hiện để xem còn thằng nào vào Group nữa không, gom đủ mặt rồi chia Partition một lần duy nhất cho tối ưu".
   - *Lưu ý:* Việc cấu hình giá trị này về `0` (như trong snippet mẫu của Apache) mang ý nghĩa "Đừng đợi, có thằng nào chia luôn thằng đó". Cấu hình này **CHỈ NÊN DÙNG** trong môi trường Local Dev để giảm độ trễ khi test App trên máy tính cá nhân. Lên Production mà để bằng `0` là tự hủy!
   - *Nói thêm:* `3000ms` chính là **giá trị mặc định của Kafka**, nên khai lại nó trong compose là vô hại nhưng cũng không thay đổi gì. Và nó **chỉ áp dụng cho lần rebalance đầu tiên khi Group đang trống**, không phải mọi lần rebalance. Cách xử lý rebalance storm ở tầng cao hơn là dùng **incremental cooperative rebalancing** (`CooperativeStickyAssignor`) — xem 3.3.

---

### 1.7 Mô hình tách riêng Controller và Broker (Dedicated Mode)

Mục 1.6 mô tả cơ chế Raft khi Node "chơi 2 vai". Mục này nói về mô hình **tách hẳn**: một nhóm Node chỉ làm Controller, một nhóm khác chỉ làm Broker — đúng cấu hình mà repo này đang chạy trong `docker-compose.yml`.

#### A. Ba chế độ triển khai

| `process.roles` | Tên gọi | Dùng khi nào |
|---|---|---|
| `broker,controller` | **Combined mode** | Local dev, CI, cụm nhỏ. Tài liệu Apache **không khuyến nghị cho production**. |
| `controller` / `broker` (hai nhóm Node riêng) | **Dedicated mode** (isolated) | Production. Đây là cấu hình của repo này. |
| *(không khai)* | Chế độ ZooKeeper cũ | Đã bị gỡ khỏi Kafka 4.x. |

#### B. Ranh giới thật sự: Control Plane và Data Plane

Đây là khái niệm cần nắm trước, vì nó không chỉ đúng với Kafka mà đúng với gần như mọi hệ phân tán (Kubernetes, Envoy, VPC của AWS đều chia y hệt).

- **Data plane** — nơi *dữ liệu người dùng* chảy qua. Với Kafka là **Broker**: nhận `Produce`, phục vụ `Fetch`, ghi segment xuống đĩa, giữ page cache, chạy replication giữa leader và follower.
- **Control plane** — nơi *quyết định về hệ thống* được đưa ra và lưu lại. Với Kafka là **Controller**: ai còn sống, partition nào của topic nào nằm ở broker nào, ai là leader, ISR gồm những ai.

Hai mặt phẳng này có **đặc tính tải hoàn toàn khác nhau**. Data plane là throughput cao, I/O nặng, heap lớn, GC thường xuyên. Control plane là lưu lượng nhỏ nhưng đòi hỏi **độ trễ thấp và tính nhất quán tuyệt đối**. Nhét chung một tiến trình nghĩa là để hai loại tải đối nghịch tranh nhau cùng một CPU, cùng một heap, cùng một đĩa.

#### C. Broker là **observer**, không phải **voter**

Đây là điểm hay bị hiểu sai nhất, và cũng là câu hỏi phỏng vấn phân loại người dùng Kafka với người hiểu Kafka.

Trong dedicated mode:

- **Chỉ Controller mới bỏ phiếu.** Quorum Raft chỉ gồm các Node có `process.roles=controller`. Broker **không** nằm trong cuộc bầu cử, **không** được tính vào công thức quá bán.
- **Broker vẫn phải khai `controller.quorum.voters`** — không phải để bỏ phiếu, mà để **biết đi đâu mà kéo metadata về**. Broker mở kết nối tới active controller và **fetch** log `__cluster_metadata` y hệt cách một consumer fetch một topic thường: có offset, có vị trí đang đọc, đọc tới đâu replay vào bộ nhớ tới đó.
- Vai trò đó trong thuật ngữ Raft gọi là **observer** (hay learner): sao chép log, không có quyền biểu quyết.

**Vì sao mô hình kéo (pull) này quan trọng:** thời ZooKeeper, controller **đẩy** metadata xuống broker bằng RPC. Broker không biết mình đang thiếu bản cập nhật nào, và controller mới lên phải nạp lại toàn bộ trạng thái rồi phát lại cho mọi broker — thời gian tỉ lệ thuận với số partition. Với KRaft, mỗi broker **biết chính xác mình đang ở metadata offset bao nhiêu**, nên chuyển giao controller chỉ là "đổi địa chỉ fetch", không có bước nạp lại. Đây là lý do gốc rễ KRaft nâng được trần scale lên hàng triệu partition.

#### D. Node ID là **một không gian tên duy nhất cho cả cụm**

`node.id` phải **duy nhất trên toàn bộ cụm**, tính gộp cả Controller lẫn Broker. Không có chuyện "controller đánh số riêng, broker đánh số riêng".

Quy ước dễ nhớ khi tách vai: dành dải thấp cho controller, dải cao cho broker.

```
controller-1 = 1, controller-2 = 2, controller-3 = 3
broker-1     = 4, broker-2     = 5, broker-3     = 6
broker-4     = 7, broker-5     = 8      <-- không được lặp lại 6
```

Trùng `node.id` là lỗi copy-paste phổ biến nhất khi nhân bản block broker trong compose. Triệu chứng: Node lên sau bị controller từ chối đăng ký, hoặc tệ hơn là "cướp" phiên đăng ký của Node trùng id khiến metadata dao động. Kiểm tra bằng:

```bash
podman exec broker-1 /opt/kafka/bin/kafka-metadata-quorum.sh --bootstrap-server broker-1:19092 describe --status
```

#### E. Năm lý do để tách

1. **Cách ly tài nguyên.** Một pha GC dài hoặc đĩa đầy trên broker không được phép làm rung quorum metadata. Khi cụm gặp sự cố cũng chính là lúc control plane cần khoẻ nhất — để chung là đảm bảo nó yếu nhất đúng lúc đó.
2. **Vòng đời khác nhau.** Broker restart rất thường xuyên (rolling upgrade, đổi config, thay đĩa). Mỗi lần restart broker mà kéo theo bầu lại leader quorum là chi phí vô nghĩa.
3. **Bề mặt bảo mật hẹp hơn.** Controller listener chỉ mở cho controller nói với nhau và broker fetch. Client ứng dụng không bao giờ chạm tới. Tách vai cho phép đặt chính sách mạng và chứng chỉ khác nhau cho hai mặt phẳng.
4. **Scale độc lập.** Cần thêm dung lượng lưu trữ hay throughput thì thêm **broker**. Thêm **voter** không giúp gì về dung lượng mà còn làm mọi thao tác metadata chậm đi (mục F).
5. **Vận hành rõ ràng.** Nhìn `process.roles` là biết Node đó hỏng sẽ mất gì. Combined mode làm mờ ranh giới trách nhiệm khi đi truy sự cố.

#### F. Sizing quorum: vì sao là 3, và vì sao số chẵn là vô nghĩa

Quorum chịu lỗi theo công thức `⌊(N−1)/2⌋`:

| N voter | Cần bao nhiêu phiếu để commit | Chịu được mất | Nhận xét |
|---|---|---|---|
| 1 | 1 | 0 | Mất controller là cụm đóng băng metadata |
| **3** | 2 | **1** | **Mặc định đúng cho gần như mọi cụm** |
| 4 | 3 | 1 | Chịu lỗi *bằng* N=3 nhưng **commit chậm hơn** → luôn tệ hơn |
| 5 | 3 | 2 | Chỉ đáng khi thật sự cần sống sót qua 2 lỗi đồng thời |
| 7 | 4 | 3 | Hiếm; độ trễ metadata bắt đầu thành vấn đề |

**Vì sao bắt buộc quá bán?** Để chặn **split-brain**. Nếu chỉ cần một nửa (2/4), một sự cố mạng chia cụm thành hai nhóm 2 Node sẽ cho phép **cả hai nhóm** cùng bầu ra controller của riêng mình, và hai controller cùng ghi metadata mâu thuẫn. Yêu cầu **quá bán** khiến hai nhóm không thể cùng đạt chuẩn — về mặt toán học, hai tập con quá bán của cùng một tập luôn có phần giao. Nhóm thiểu số tự động đứng im.

**Vì sao số chẵn tệ hơn?** N=4 vẫn chỉ chịu được 1 lỗi (mất 2 là còn 2, không quá bán), y hệt N=3. Nhưng mỗi lần ghi metadata phải chờ **3** Node ghi xong đĩa thay vì 2 — chậm hơn mà không đổi lấy được gì. Quy tắc này đúng cho mọi hệ Raft/Paxos: ZooKeeper, etcd, Consul, MongoDB replica set đều thế.

**Vì sao không lên 5 cho chắc?** Vì mỗi voter thêm vào làm **mọi thao tác metadata chậm hơn**: commit phải chờ nhiều bản ghi đĩa hơn, và bầu cử phải thu thập nhiều phiếu hơn. 5 controller chỉ đáng khi cụm trải trên nhiều Availability Zone hoặc nhiều Data Center và bạn thật sự cần chịu 2 lỗi đồng thời.

#### G. Mất quorum thì cụm chết tới đâu?

Câu hỏi phỏng vấn hay, và câu trả lời phản trực giác: **cụm không chết ngay**.

Giả sử cụm 3 controller mà mất 2 (còn 1, không đạt quá bán):

| Việc | Còn chạy? |
|---|---|
| Produce/Consume trên partition **đang có leader sống** | **Còn** — broker phục vụ bằng metadata đã fetch được trước đó |
| Bầu leader mới khi một broker chết | **Không** |
| Tạo / xoá topic, đổi config động | **Không** |
| Cập nhật ISR (co hoặc giãn) | **Không** |
| Broker mới đăng ký vào cụm | **Không** |

Và đây là hệ quả dây chuyền tinh tế: leader muốn **co ISR** (loại một follower đã tụt lại) thì phải gửi `AlterPartition` lên controller. Controller không phản hồi được → ISR không co được → với `acks=all`, producer phải chờ một replica đã chết → **ghi bị treo rồi timeout, và không tự hồi phục** cho tới khi quorum trở lại.

Nên phát biểu chuẩn là: mất quorum thì cụm **đóng băng khả năng thay đổi**, chứ không mất dữ liệu và không dừng phục vụ ngay. Nhưng nó sẽ **thoái hoá dần** theo từng sự cố nhỏ tiếp theo.

#### H. Controller **có** state — đừng nói nó stateless

Controller ghi Raft log metadata xuống đĩa (`log.dirs` của Node controller, trong repo này là volume `controller-N-data`). Mất sạch đĩa của **quá bán** controller là mất metadata của cả cụm: topic nào tồn tại, partition nằm ở đâu, ai là leader. Broker vẫn còn nguyên file dữ liệu nhưng **không ai biết chúng là gì**.

Vì vậy volume của controller **cũng cần backup**, dù dung lượng chỉ vài chục MB. Nói "controller là stateless nên không cần lo" là sai, và người phỏng vấn sẽ bắt ngay.

#### I. Config nào thuộc vai nào

| Config | Vai đọc nó | Vì sao |
|---|---|---|
| `process.roles` | cả hai | Khai vai của chính Node đó |
| `node.id` | cả hai | Duy nhất trên toàn cụm |
| `controller.quorum.voters` | cả hai | Controller để bỏ phiếu; Broker để tìm chỗ fetch metadata |
| `controller.listener.names` | cả hai | Tên listener dùng cho control plane |
| `advertised.listeners` | **broker** | Controller không quảng bá cho client |
| `auto.create.topics.enable` | **broker** | Broker quyết định có gửi CreateTopics lên controller không |
| `offsets.topic.replication.factor` | **broker** | GroupCoordinator chạy trên broker và tạo `__consumer_offsets` |
| `min.insync.replicas` | **broker** | Leader của partition đếm ISR lúc ghi |
| `num.partitions` | **controller** (KRaft) | Controller materialize topic khi CreateTopics gửi lên với `-1` |
| `default.replication.factor` | **controller** (KRaft) | Như trên |

Hai dòng cuối **khác thời ZooKeeper**, nơi broker nhận request và tự điền default. Trong KRaft, broker **forward** `CreateTopics` lên active controller và chính controller thay `-1` bằng giá trị mặc định.

> **Chưa kiểm chứng trên bản đang dùng.** Đây là loại chi tiết đổi theo phiên bản, nên kiểm bằng thực nghiệm thay vì tin tài liệu: đặt `KAFKA_NUM_PARTITIONS: 6` **chỉ trên controller**, tạo một topic không khai partition, rồi `--describe`. Ra 6 → controller quyết. Ra 1 → broker quyết. Ghi kết quả vào `docs/adr/`.
>
> Cách an toàn nếu không muốn thí nghiệm: đặt `num.partitions` và `default.replication.factor` trên **cả hai vai**. Config không áp dụng cho vai nào thì bị bỏ qua, không gây lỗi.

#### J. Quorum tĩnh và quorum động (KIP-853)

`controller.quorum.voters` là **quorum tĩnh**: danh sách thành viên nằm cứng trong config. Muốn thêm hoặc bớt một controller thì phải sửa config trên **mọi** Node rồi restart cả cụm — thao tác rủi ro và không làm được lúc đang tải cao.

Kafka 4.x có thêm `controller.quorum.bootstrap.servers` cho **quorum động** (KIP-853): thành viên quorum được lưu trong chính metadata log, và thêm/bớt voter bằng `kafka-metadata-quorum add-controller` / `remove-controller` khi cụm đang chạy. Hai chế độ **loại trừ nhau** — chọn một, và quyết định đó nằm ngay ở bước `kafka-storage format` lúc khởi tạo cụm, không đổi lại dễ dàng.

Repo này đang chạy chế độ **tĩnh**. Chưa kiểm chứng chi tiết API trên bản 4.3.1 — tra lại tài liệu Apache trước khi dùng.

#### K. Khi tách ra nhiều máy: `advertised.listeners` là chỗ vỡ đầu tiên

Trong compose một máy, broker quảng bá `broker-1:19092` và mọi thứ chạy vì Podman có DNS nội bộ. Khi tách sang **nhiều EC2**, tên container không còn phân giải được.

Nhớ nguyên tắc gốc: **`advertised.listeners` không phải địa chỉ broker lắng nghe, mà là địa chỉ broker khai với client rằng "hãy quay lại tìm tôi ở đây"**. Client bootstrap tới một broker bất kỳ, nhận về metadata gồm địa chỉ leader từng partition, rồi **mở kết nối mới** tới đúng địa chỉ đó. Quảng bá sai thì triệu chứng đặc trưng là **bootstrap thành công rồi treo** — dấu hiệu kinh điển của cấu hình listener sai.

Khi lên nhiều máy, `advertised.listeners` của mỗi broker phải là địa chỉ **mà client thật sự gọi tới được**: private IP trong VPC cho client nội bộ, và một listener riêng nếu có client ngoài VPC. Cùng với đó, controller listener (9093) chỉ nên mở giữa các Node trong Security Group, không bao giờ mở ra Internet.

#### L. Checklist tự kiểm sau khi dựng cụm tách vai

```bash
# 1. Quorum có đủ voter, ai là leader, follower có tụt lại không
podman exec broker-1 /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server broker-1:19092 describe --status

# 2. Broker nào đang đăng ký sống trong cụm
podman exec broker-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server broker-1:19092 | grep -c "id:"

# 3. RF và ISR thật của từng topic, gồm cả topic nội bộ
podman exec broker-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server broker-1:19092 --describe
```

Và bài kiểm chứng thật, không thay thế được bằng đọc config: **dừng lần lượt từng Node và quan sát**. Dừng 1 controller → quorum vẫn commit được. Dừng 2 controller → mọi thao tác tạo topic đứng im nhưng produce/consume vẫn chạy (mục G). Dừng 1 broker với RF=3, `min.insync.replicas=2` → vẫn ghi được. Dừng thêm 1 broker nữa → producer nhận `NOT_ENOUGH_REPLICAS` chứ không được ack giả.

---

## 2. Đi Sâu Vào Producer (Nhà Sản Xuất)

Producer không hề ngốc nghếch, nó rất thông minh và làm nhiều việc ngầm:

### 2.1 Khả năng tự nhận thức (Cluster Topology Awareness)
- Producer không gửi tin nhắn một cách mù quáng. Nó biết chính xác **Partition nào đang nằm ở Broker nào**. 
- Trong trường hợp một Broker bị chết (Failures), Producer sẽ tự động phục hồi (Auto Recover) và tìm ra Broker mới đang nắm giữ Leader Partition để tiếp tục gửi data.
- Quá trình chia tải (Load Balancing) được Producer chủ động tính toán trước khi đẩy data sang mạng, giúp cụm Broker không bị quá tải cục bộ.

### 2.2 Thuật toán Phân luồng (Producer Partitioner Logic & Key Hashing)
Kafka Partitioner là một khối logic mã nguồn nằm bên trong Producer, có nhiệm vụ tiếp nhận một Record (tin nhắn) và quyết định chính xác xem tin nhắn này sẽ được đẩy vào Partition nào. Quá trình ánh xạ (mapping) từ một Key ra một Partition cụ thể được gọi là **Key Hashing** (Băm khóa).

Trong Partitioner mặc định (Default Partitioner) của Kafka, thuật toán băm được sử dụng là **murmur2**.
- **Murmur2 là gì?** Đây là một hàm băm phi mật mã (non-cryptographic hash function). Nó không được sinh ra để bảo mật như SHA-256 hay MD5, mà được thiết kế tối thượng cho **tốc độ thực thi cực nhanh** và **khả năng phân phối đồng đều (good distribution)**. Nhờ đó, dữ liệu được rải rất mượt mà trên toàn bộ các Partitions, tránh tình trạng "nút cổ chai" cục bộ (hotspots).

**Công thức định tuyến (Routing Formula):**
Mã nguồn thực tế ẩn dưới nắp capo của Producer để tính toán Partition mục tiêu là:
`targetPartition = Math.abs(Utils.murmur2(keyBytes)) % (numPartitions - 1)`

**5 Chiến lược phân luồng (Partitioner Strategies) của Producer:**
1. **Key Hashing (Mặc định khi có Key):** Áp dụng thuật toán `murmur2`. Mọi tin nhắn có cùng Key (VD: `BidID=123`) sẽ luôn bay vào cùng một Partition. Cực kỳ quan trọng để bảo toàn **Strict Ordering** (Thứ tự tuyệt đối).
2. **Sticky Partitioner (Mặc định khi Key = null, từ Kafka 2.4+):** Cực kỳ tối ưu! Producer gom các tin nhắn (dù khác loại) thành một Batch lớn gửi vào 1 Partition ngẫu nhiên cho đến khi đầy Batch (hoặc hết thời gian `linger.ms`), sau đó mới đổi sang Partition khác. Giảm triệt để độ trễ mạng.
3. **RoundRobin Partitioner (Legacy khi Key = null, Kafka cũ):** Phân phối kiểu chia bài (Msg 1 vào P0, Msg 2 vào P1, Msg 3 vào P2). Nghe có vẻ công bằng nhưng **CỰC KỲ TỆ** vì nó tạo ra vô số các Batch nhỏ xíu tốn băng thông. Hiện tại gần như không ai dùng.
4. **Explicit Partitioner (Chỉ định tường minh):** Lập trình viên truyền cứng `Partition ID` vào record, Producer sẽ nhắm mắt ném thẳng vào Partition đó, bỏ qua mọi logic.
5. **Custom Partitioner:** Tự code logic phân luồng (Implement interface `Partitioner`). Ví dụ: Khách Vip ném vào P0 (server xịn), khách thường ném vào P1, P2.

### 2.3 Cấu hình `acks` (Producer Acknowledgements - Đánh đổi Mạng sống và Tốc độ)
Producer có quyền tự quyết định xem nó cần mức độ xác nhận (acknowledgment) nào từ Broker sau khi ghi dữ liệu:
- **`acks=0` (Fire and Forget):** Producer ném data vào mạng là xong, hoàn toàn **KHÔNG ĐỢI** xác nhận từ Broker. Tốc độ bàn thờ nhưng rủi ro **mất dữ liệu (Data Loss)** rất cao nếu Broker chưa kịp nhận mà rớt mạng.
- **`acks=1` (Leader Ack):** Producer đợi xác nhận từ Leader. Chỉ cần Leader ghi xuống đĩa thành công là báo OK. Rủi ro **mất dữ liệu hữu hạn (Limited Data Loss)** nếu Leader báo OK xong tẻo luôn trước khi Follower kịp chép về.
- **`acks=all` (hoặc `-1`):** Producer đợi xác nhận từ CẢ Leader và các Replicas. Leader phải đợi tất cả các thằng trong danh sách ISR chép xong thì mới báo OK. Chậm nhất nhưng **Không bao giờ mất dữ liệu (No Data Loss)**.

### 2.4 Idempotent Producer & Retries (Chống gửi đúp)
- Nếu Producer gửi tin nhắn, Broker nhận được nhưng lúc báo Ack về thì rớt mạng. Producer tưởng gửi xịt nên gửi lại -> **Dữ liệu bị đúp (Duplicate)**.
- **Giải pháp:** Bật `enable.idempotence=true`. Lúc này Producer sẽ dán cho mỗi tin nhắn 1 cái `PID` (Producer ID) và 1 `Sequence Number`. Nếu Broker thấy `Seq No` bị trùng, nó tự động vứt tin nhắn thứ 2 đi -> Đảm bảo **Exactly-Once Semantics (EOS)** ở cấp độ Producer.

---

## 3. Đi Sâu Vào Consumer & Consumer Group

### 3.0 Đặc tính cốt lõi của Consumer
- **Mô hình Kéo (Pull Model):** Khác với RabbitMQ (Broker chủ động đẩy - Push data cho worker), ở Kafka, Consumer tự mình chủ động gọi lệnh (pull) để xin dữ liệu từ Topic. Điều này giúp Consumer không bao giờ bị "ngộp" (overwhelmed) vì nó tự quyết định tốc độ đọc.
- **Nhận thức không gian:** Khi kết nối vào cụm, Consumer tự động "biết" chính xác phải đọc Partition nào và phải kết nối đến Broker vật lý nào (nhờ cơ chế Discovery).
- **Tự phục hồi (Recovery):** Nếu Broker mà Consumer đang đọc bị sập (Failures), Consumer đủ thông minh để tự động chuyển hướng kết nối sang Broker khác đang nắm giữ bản sao Leader mới.
- **Thứ tự (Ordering):** Dữ liệu luôn được đọc theo thứ tự nghiêm ngặt từ Offset thấp (cũ nhất) đến Offset cao (mới nhất) **TRONG PHẠM VI TỪNG PARTITION**.

### 3.1 Cân bằng tải (Load Balancing) & Starvation
- **Rule thép:** 1 Partition CHỈ được đọc bởi TỐI ĐA 1 Consumer trong CÙNG 1 Group.
- Nếu Số Consumer > Số Partition -> Các Consumer dư thừa sẽ ở trạng thái Idle (Ngồi chơi - Starvation).
- Nếu Số Consumer < Số Partition -> 1 Consumer sẽ phải cõng (đọc) nhiều Partition.

### 3.2 Đa Consumer Group (Multiple Consumer Groups trên 1 Topic)
- **Hoàn toàn hợp lệ (Acceptable):** Kafka được thiết kế lõi theo mô hình Pub/Sub (Fan-out), cho phép rất nhiều Consumer Group cùng subscribe và đọc dữ liệu trên cùng 1 Topic mà không tranh giành hay triệt tiêu data của nhau.
- **Cách phân tách:** Chỉ cần cấu hình thuộc tính `group.id` khác nhau cho mỗi cụm ứng dụng (Ví dụ: `group.id=consumer-group-application-1`, `group.id=consumer-group-application-2`).
- **Cơ chế hoạt động:**
  - Giả sử Topic-A có 3 Partition.
  - **Group 1** (có 2 Consumer): Sẽ chia nhau đọc 3 Partition (vd: C1 đọc P0, P1; C2 đọc P2).
  - **Group 2** (có 3 Consumer): Mỗi Consumer đọc đúng 1 Partition (C1->P0, C2->P1, C3->P2).
  - **Group 3** (có 1 Consumer): Mình nó ôm "show" đọc cả 3 Partition.
- **Điểm ăn tiền:** Mỗi Consumer Group duy trì một con trỏ Offset hoàn toàn độc lập. Ứng dụng này đọc xong, commit Offset rồi thì ứng dụng kia vẫn có thể bắt đầu đọc lại từ Offset số 0.

### 3.3 Rebalance (Tái cân bằng)

**A. Bản chất của Rebalance là gì?**
- Việc thuyên chuyển (di dời) quyền đọc các Partition từ Consumer này sang Consumer khác được gọi là **Rebalance** (Tái cân bằng).
- **Khi nào xảy ra?**
  1. Khi có một Consumer rời khỏi nhóm (bị crash, tắt máy) hoặc join mới vào nhóm (scale up).
  2. Khi quản trị viên hệ thống (Administrator) add thêm Partition mới vào Topic.

**B. Sự khác biệt giữa 2 trường phái Rebalance:**
1. **Trường phái Eager Rebalance (Cực kỳ tốn kém):**
   - **Tất cả Consumer phải DỪNG HOẠT ĐỘNG (Stop-The-World).**
   - Mọi người phải tự nguyện từ bỏ (give up) quyền đọc tất cả các Partition đang giữ.
   - Tham gia lại vào Group và nhận lệnh phân công mới từ Coordinator.
   - Nhược điểm: Cả một khoảng thời gian ngắn, toàn bộ Group bị tê liệt không xử lý data. Khi nhận lại, chưa chắc Consumer được chia lại đúng Partition mà nó vừa thả ra.
2. **Trường phái Cooperative Rebalance / Incremental (Khuyên dùng):**
   - Chỉ thu hồi (revoke) một **tập hợp nhỏ** các Partition thực sự cần phải dời đi.
   - **Đột phá:** Những Consumer không bị dính líu đến các Partition bị thu hồi vẫn tiếp tục xử lý data bình thường (Không bị ngắt quãng).
   - Quá trình này diễn ra qua vài vòng lặp (iterations) để đạt được trạng thái ổn định (incremental).
   - Chấm dứt hoàn toàn hiện tượng Stop-the-world.

**C. Cấu hình phân chia (partition.assignment.strategy):**
*(Ghi chú: Mặc định (Default Assignor) của Kafka Client là một danh sách `[RangeAssignor, CooperativeStickyAssignor]`. Hệ thống sẽ ưu tiên dùng RangeAssignor, nhưng thiết kế này cho phép nâng cấp mượt mà lên CooperativeStickyAssignor chỉ với một lần **rolling bounce** (khởi động lại xoay vòng) bằng cách xóa RangeAssignor ra khỏi danh sách).*
*(Lưu ý đối với hệ sinh thái Kafka:)*
*- **Kafka Connect:** Cooperative Rebalance đã được tích hợp và bật mặc định.*
*- **Kafka Streams:** Cooperative Rebalance cũng được bật mặc định thông qua thuật toán `StreamsPartitionAssignor`.*

*(Giả sử hệ thống có 2 Topic: Topic-A (3 Partition A0, A1, A2) và Topic-B (3 Partition B0, B1, B2). Group ban đầu có 2 Consumer: C1, C2)*

*▶ Nhóm Eager Rebalance (Gây Stop-The-World):*

**1. Range Assignor (Mặc định cũ)**
- **Khái niệm:** Phân chia Partition trên cơ sở tính toán độc lập cho từng Topic (per-topic basis).
- **Các bước hoạt động:**
  - Bước 1: Sắp xếp các Partition của Topic đang xét theo thứ tự tăng dần.
  - Bước 2: Sắp xếp các Consumer trong Group theo danh pháp (bảng chữ cái hoặc ID).
  - Bước 3: Lấy tổng số Partition chia cho tổng số Consumer.
  - Bước 4: Cắt lát mảng Partition theo tỷ lệ. Các Consumer đứng đầu danh sách sẽ phải gánh thêm phần dư (nếu có). Lặp lại quy trình với các Topic tiếp theo.
- **Ví dụ chạy thực tế:** Xử lý Topic A: C1 nhận (A0, A1), C2 nhận (A2). Xử lý Topic B: C1 nhận (B0, B1), C2 nhận (B2). Kết quả: C1 quản lý 4 Partition, C2 quản lý 2 Partition.
- **Ưu điểm:** Logic đơn giản, dễ triển khai. Các Partition của cùng 1 Topic có xu hướng hội tụ (co-locate) vào cùng một Consumer.
- **Nhược điểm:** Gây mất cân bằng tải trầm trọng (Imbalance) khi số lượng Topic lớn. Consumer đầu tiên (C1) sẽ liên tục phải chịu tải cao hơn, dẫn đến nguy cơ quá tải cục bộ.

**2. RoundRobin Assignor**
- **Khái niệm:** Loại bỏ ranh giới giữa các Topic. Hợp nhất tất cả Partition và phân bổ tuần tự (round-robin fashion) cho toàn bộ Consumer.
- **Các bước hoạt động:**
  - Bước 1: Tổng hợp tất cả Partition của các Topic mà Group đang theo dõi thành một danh sách duy nhất.
  - Bước 2: Sắp xếp danh sách Partition và danh sách Consumer.
  - Bước 3: Phân bổ tuần tự từng Partition cho từng Consumer cho đến khi duyệt hết danh sách.
- **Ví dụ chạy thực tế:** Tập hợp Partition: (A0, A1, A2, B0, B1, B2). Vòng 1: C1 nhận A0, C2 nhận A1. Vòng 2: C1 nhận A2, C2 nhận B0. Vòng 3: C1 nhận B1, C2 nhận B2.
- **Ưu điểm:** Đạt được độ cân bằng tải tối ưu (Optimal balance). Mỗi Consumer quản lý chính xác 3 Partition.
- **Nhược điểm:** Khi cấu trúc Group thay đổi (ví dụ C3 tham gia), trình tự phân bổ bị xáo trộn hoàn toàn. C1 và C2 có nguy cơ bị thu hồi toàn bộ Partition hiện tại để nhận các Partition mới, gây lãng phí tài nguyên và thời gian tái thiết lập bộ nhớ đệm (State/Cache).

**3. Sticky Assignor**
- **Khái niệm:** Một chiến lược cân bằng tải dựa trên nguyên tắc của RoundRobin, nhưng bổ sung cơ chế "bám dính" (sticky) nhằm tối thiểu hóa số lượng Partition phải di dời (partition movements) mỗi khi có sự kiện thay đổi thành viên trong Group.
- **Các bước hoạt động:**
  - Bước 1: Ở trạng thái ban đầu, hệ thống phân bổ cân bằng tương tự RoundRobin.
  - Bước 2: Khi có Consumer mới (C3) gia nhập, Coordinator kích hoạt Rebalance. Giai đoạn này thuộc nhóm Eager Rebalance, do đó **TẤT CẢ Consumer phải tạm dừng xử lý dữ liệu (Stop-The-World)** và thu hồi quyền kiểm soát Partition.
  - Bước 3: Coordinator tính toán phương án phân bổ mới, với ưu tiên hàng đầu là gán lại các Partition cũ cho chính Consumer đang xử lý nó (C1 và C2).
  - Bước 4: Sau khi bảo toàn tối đa cấu trúc cũ, Coordinator sẽ bóc tách những Partition dư thừa (do mất cân bằng) để gán cho Consumer mới (C3).
- **Ví dụ chạy thực tế (Chi tiết từng bước):**
  - *Trạng thái ban đầu:* Hệ thống có Topic-A (A0, A1, A2) và Topic-B (B0, B1, B2). Group hiện có 2 Consumer. Phân bổ đang tối ưu: C1 xử lý (A0, A2, B1), C2 xử lý (A1, B0, B2).
  - *Sự kiện phát sinh:* C3 khởi động và gửi yêu cầu gia nhập Group.
  - *Bước 1 (Stop-The-World):* Coordinator phát lệnh dừng toàn hệ thống. C1 ngừng đọc A0, A2, B1. C2 ngừng đọc A1, B0, B2. Mọi Partition bị thu hồi tạm thời về trạng thái vô chủ.
  - *Bước 2 (Bảo toàn State):* Coordinator tính toán phân chia lại 6 Partition cho 3 Consumer (mỗi Consumer sẽ nhận 2 Partition). Nó ưu tiên giữ lại các kết nối cũ: Trả lại A0, A2 cho C1; trả lại A1, B0 cho C2.
  - *Bước 3 (Gán phần dư):* Còn lại B1 (trước đây của C1) và B2 (trước đây của C2) được gom lại thành cặp (B1, B2) để gán cho C3.
  - *Kết quả cuối cùng:* C1(A0, A2), C2(A1, B0), C3(B1, B2). Dù được tối ưu không làm xáo trộn các Partition cũ, nhưng ở Bước 1, toàn bộ hệ thống vẫn phải trải qua một nhịp "đóng băng" (downtime).
- **Ưu điểm:** Đảm bảo tải phân bổ đồng đều trong khi vẫn giữ lại phần lớn bộ nhớ đệm (State/Cache) của các Consumer, giúp quá trình phục hồi sau Rebalance diễn ra nhanh hơn.
- **Nhược điểm:** Vẫn tồn tại giai đoạn **Stop-The-World** (Bước 2). Dù thực tế số lượng Partition bị thay đổi rất ít, nhưng toàn bộ Group vẫn bị gián đoạn hoạt động hoàn toàn cho đến khi quá trình tái phân bổ hoàn tất.

*▶ Nhóm Cooperative Rebalance (KHÔNG Stop-The-World):*

**4. Cooperative Sticky Assignor (Khuyên dùng từ Kafka 2.4+)**
- **Khái niệm:** Kế thừa thuật toán tối ưu của Sticky Assignor, nhưng nâng cấp toàn diện cơ chế thực thi sang mô hình Cooperative (Hiệp đồng). Nó cho phép các Consumer giữ lại quyền kiểm soát Partition và **tiếp tục xử lý dữ liệu (keep on consuming)** trong khi quá trình Rebalance diễn ra.
- **Các bước hoạt động:**
  - Bước 1: Khi C3 gia nhập, Coordinator kích hoạt Rebalance lần 1.
  - Bước 2: Đột phá lớn nhất: Các Consumer (C1, C2) **KHÔNG DỪNG HOẠT ĐỘNG**. Chúng tiếp tục xử lý các Partition hiện tại. Dựa trên lệnh từ Coordinator, C1 và C2 chỉ chủ động thu hồi (revoke) các Partition bị dư ra (B1, B2) và trả về cho hệ thống.
  - Bước 3: Coordinator nhận được các Partition vừa thu hồi, lập tức kích hoạt Rebalance lần 2 để gán các Partition đang chờ (B1, B2) cho Consumer mới (C3).
- **Ví dụ chạy thực tế (Chi tiết 2 vòng lặp):**
  - *Trạng thái ban đầu:* C1 đang miệt mài xử lý (A0, A2, B1) và C2 đang xử lý (A1, B0, B2).
  - *Sự kiện phát sinh:* C3 xin gia nhập Group. Coordinator nhận tín hiệu và kích hoạt **Rebalance Vòng 1**.
  - *Vòng 1 (Thu hồi hiệp đồng):* Coordinator gửi tín hiệu "Có thành viên mới (C3), yêu cầu san sẻ 2 Partition". Khác biệt cực lớn: C1 và C2 **KHÔNG ĐÓNG BĂNG TOÀN BỘ**. C1 chỉ chủ động ngắt kết nối với B1, C2 ngắt kết nối với B2. Trong khoảnh khắc đó, C1 VẪN liên tục kéo và xử lý dữ liệu từ A0, A2; C2 VẪN miệt mài xử lý A1, B0 bình thường.
  - *Vòng 2 (Gán Partition trống):* Coordinator thu thập B1 và B2 (vừa bị nhả ra thành vô chủ). Kích hoạt **Rebalance Vòng 2** để chính thức gán B1 và B2 cho C3. C3 bắt đầu vào guồng xử lý.
  - *Kết quả cuối cùng:* Hệ thống hoàn tất việc scale từ 2 lên 3 Consumer mà dòng chảy dữ liệu tại A0, A1, A2, B0 chưa hề bị khựng lại một giây nào (Zero Downtime).
- **Ưu điểm:** Loại bỏ triệt để hiện tượng Stop-The-World. Hệ thống duy trì tính sẵn sàng (Zero Downtime) và hiệu suất cao nhất ngay cả khi mở rộng quy mô.
- **Nhược điểm:** Giao thức xử lý ngầm (internal protocol) trải qua nhiều vòng lặp (iterations) phức tạp hơn. Tuy nhiên, rào cản này đã được các thư viện Kafka Client xử lý hoàn toàn trong suốt với lập trình viên.

**5. Cấu hình Static Group Membership (Ngăn chặn Rebalance không cần thiết)**
- **Khái niệm:** Đây không phải là một thuật toán Assignor, mà là một cấu hình cơ sở hạ tầng (`group.instance.id`) giúp Consumer gắn kết tĩnh (Static) với Broker, ngăn chặn việc kích hoạt Rebalance ngay lập tức khi mạng bị gián đoạn tạm thời.
- **Các bước hoạt động:**
  - Bước 1: Cấu hình định danh tĩnh (VD: `group.instance.id = worker-1`) cho C1. C1 đang quản lý A0, A1.
  - Bước 2: C1 bị khởi động lại để triển khai mã nguồn mới (thời gian mất kết nối khoảng 10 giây).
  - Bước 3: Broker phát hiện C1 mất kết nối, nhưng nhận diện được ID tĩnh nên quyết định đưa C1 vào trạng thái chờ (trong khoảng `session.timeout.ms`) thay vì lập tức kích hoạt Rebalance.
  - Bước 4: C1 khởi động thành công và kết nối lại. Broker trả lại đúng A0, A1 cho C1 tiếp tục xử lý.
- **Ví dụ chạy thực tế (Chi tiết đợt Rolling Update):**
  - *Trạng thái ban đầu:* C1 mang ID tĩnh `pod-worker-1` phụ trách (A0, A1). C2 mang ID tĩnh `pod-worker-2` phụ trách (B0, B1).
  - *Sự kiện phát sinh:* Hệ thống triển khai bản Release mới trên Kubernetes (Rolling Update). Pod C1 nhận lệnh `SIGTERM` và khởi động lại. Quá trình tắt và bật lại mất khoảng 15 giây.
  - *Bước 1 (Mất mạng tạm thời):* Broker phát hiện C1 mất tín hiệu heartbeat. Thay vì hoảng loạn và la lên "C1 chết rồi, Rebalance toàn hệ thống ngay!" (hành vi mặc định Eager), Broker dò thấy cấu hình ID tĩnh `pod-worker-1`. Nó ra quyết định: "Tạm thời đóng băng A0, A1 và cho nó cơ hội quay lại trong vòng `session.timeout.ms` (ví dụ 60 giây)".
  - *Bước 2 (Độ ổn định cực cao):* Trong 15 giây C1 vắng mặt đó, C2 VẪN ung dung xử lý B0, B1 bình thường. Hệ thống hoàn toàn không kích hoạt một đợt Rebalance nào.
  - *Bước 3 (Đoàn tụ):* C1 boot xong, mở lại kết nối tới Broker và tự xưng `group.instance.id = pod-worker-1`. Broker nhận ra người quen, lập tức chuyển giao lại A0, A1 cho C1 tiếp tục công việc.
  - *Kết quả cuối cùng:* Một đợt Deploy rủi ro được hoàn thành mượt mà, sự kiện Rebalance vốn cực kỳ đắt đỏ đã bị "vô hiệu hóa" triệt để.
- **Ưu điểm:** Triệt tiêu hoàn toàn sự gián đoạn do Rebalance trong quá trình triển khai hệ thống (Deployment) hoặc khi hạ tầng mạng thiếu ổn định.
- **Nhược điểm:** Đòi hỏi cấu hình vận hành (DevOps) khắt khe. Phải thiết kế hệ thống cấp phát ID động (ví dụ: gán theo Hostname hoặc StatefulSet Pod Name) để đảm bảo không có hai Consumer nào trùng định danh tĩnh.

### 3.4 Consumer Offsets & Delivery Semantics (Lưu điểm nhớ & Độ tin cậy)
Khi Consumer trong một Group đọc và xử lý xong dữ liệu, nó phải định kỳ báo cáo lại cho Kafka biết "Tôi đã xử lý đến Offset số mấy rồi" bằng hành động **Commit Offset**.
- **Topic ẩn `__consumer_offsets`:** Ít ai biết rằng, Kafka lưu trữ các Offset này vào một Topic nội bộ (internal topic) có tên là `__consumer_offsets`. Việc ghi log này diễn ra ở phía Broker chứ không phải lưu ở Client.
- **Phục hồi sau thảm họa (Fault Tolerance):** Nhờ cơ chế này, nếu một Consumer bị chết (die/crash), khi nó sống lại hoặc một Consumer khác lên thay, nó chỉ việc đọc vào Topic `__consumer_offsets` để biết điểm dừng cuối cùng và tiếp tục đọc (read back from where it left off).

**Các chiến lược Commit (Delivery Semantics):**
Mặc định Java Consumer tự động commit (Auto Commit), dẫn đến ngữ nghĩa "At-least-once". Nhưng nếu bạn dùng Manual Commit, bạn sẽ đối mặt với 3 kịch bản:

1. **At-most-once (Tối đa một lần - Rất dễ mất Data):**
   - Offset được commit *ngay lập tức* ngay khi vừa nhận được message từ Kafka (chưa thèm xử lý logic).
   - *Hậu quả:* Nếu trong lúc ứng dụng đang tính toán hoặc lưu DB mà bị crash, message đó bị **mất vĩnh viễn** (vì Kafka đã ghi nhận là đọc rồi và không gửi lại nữa).

2. **At-least-once (Ít nhất một lần - Khuyên dùng/Thường thấy nhất):**
   - Offset chỉ được commit *sau khi* message đã được xử lý xong hoàn toàn (ví dụ: Insert DB thành công).
   - *Hậu quả:* Nếu xử lý DB thành công nhưng chưa kịp Commit mà App bị crash -> Khi khởi động lại, App sẽ đọc và xử lý lại message đó -> Gây ra hiện tượng **Xử lý đúp (Duplicate Processing)**.
   - *Giải pháp:* Ứng dụng của bạn bắt buộc phải có tính **Idempotent** (có chạy lại message đó bao nhiêu lần thì kết quả hệ thống vẫn không thay đổi).

3. **Exactly-once (Chính xác một lần - Chén thánh):**
   - Nếu luồng dữ liệu chỉ nằm gọn trong Kafka (Kafka to Kafka workflows): Sử dụng **Transactional API** (rất dễ làm nếu dùng Kafka Streams API).
   - Nếu luồng dữ liệu đẩy ra hệ thống ngoài (External System workflows như ghi vào Postgres, MySQL): Cực khó. Bắt buộc phải kết hợp *At-least-once* với một **Idempotent Consumer** (thường dùng khóa ngoại hoặc bảng lưu Transaction ID để chặn trùng lặp tuyệt đối).

### 3.5 Hiện Tượng Zombie Consumer & Cơ Chế Generation ID
Trong một Consumer Group, nếu cấu hình thời gian xử lý không hợp lý có thể dẫn đến hiện tượng **Zombie Consumer** và **Split-Brain** ở cấp độ Consumer.

**A. Kịch bản phát sinh (Vấn đề):**
- Consumer A nhận được dữ liệu từ Partition 1, nhưng luồng code xử lý quá lâu (ví dụ gọi API đối tác bị treo), vượt quá ngưỡng thời gian cấu hình `max.poll.interval.ms`.
- Do không nhận được tín hiệu `poll()` mới, Kafka Group Coordinator phán quyết: *"Consumer A đã bị treo"*, và kích hoạt quá trình Rebalance.
- Partition 1 được thu hồi và giao cho Consumer B. B bắt đầu đọc và xử lý lại tin nhắn đó.
- Lát sau, API trả về kết quả, Consumer A "thức tỉnh" (trở thành Zombie). Nó hoàn tất logic và hồn nhiên gửi lệnh `commitSync()` lên Kafka để xác nhận xử lý xong Partition 1. 
- Nếu Kafka cho phép A commit, hệ thống sẽ bị **Duplicate Data** trầm trọng vì cả A và B cùng xử lý một tin nhắn.

**B. Giải pháp bảo vệ của Kafka (Generation ID / Epoch):**
- Để chống lại thảm họa này, mỗi khi Group Rebalance, Coordinator sẽ tăng một bộ đếm lên 1 đơn vị, gọi là **Generation ID** (hay Epoch của Consumer Group).
- Ví dụ: Ban đầu Group ở Generation `1`. Sau khi Rebalance (chuyển P1 cho B), Group chuyển sang Generation `2`.
- Khi Zombie A thức dậy và gửi lệnh commit, request của nó đính kèm nhãn mác cũ là `Generation 1`.
- Kafka Coordinator kiểm tra và phát hiện nhãn này đã lạc hậu so với hiện tại (`Generation 2`). Nó lập tức **từ chối** lệnh commit của A và ném ra lỗi `CommitFailedException`. 
- Nhờ cơ chế định danh thế hệ (Generation ID) này, Kafka ngăn chặn triệt để hiện tượng Split-Brain ở Consumer, đảm bảo tính toàn vẹn cho dữ liệu.

---

## 4. Retention (Dọn Dẹp Rác)
Kafka không xóa tin nhắn ngay khi Consumer đọc xong (khác với RabbitMQ). Nó giữ lại theo cấu hình:
- **Time-based (`log.retention.hours`):** Mặc định 168 giờ (7 ngày). Quá 7 ngày tự xóa.
- **Size-based (`log.retention.bytes`):** Giữ tối đa X Gigabytes cho mỗi Partition. Đầy thì xóa file cũ nhất.
- **Log Compaction:** Xóa dữ liệu cũ nhưng **Dựa trên Key**. Nếu Topic có nhiều message cùng Key (VD: Cập nhật tọa độ tài xế ID=5 liên tục), Kafka chỉ giữ lại message *mới nhất* của tài xế ID=5. (Cực kỳ hữu ích để lưu trạng thái System State).

---

## 5. Kết Luận cho Dự Án Matching Service
- **Topic:** `matching.events`
- **Partition:** Mặc định 3 (để sau này có thể scale lên 3 Consumer Pods).
- **Producer Config:** `acks=all`, `enable.idempotence=true` (Đảm bảo không rớt và không đúp lệnh Match).
- **Consumer Config:** Dùng Manual Commit. Lấy message -> Gửi Email/Push Notification -> Gọi `Commit()`.
