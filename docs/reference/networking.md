# Internet và kiến trúc mạng — ghi chú

Ghi chú về mạng máy tính và bảo mật hạ tầng, gom lại để tra cứu khi làm việc với
DevOps, backend hoặc data. Nội dung bám theo cách trình bày của tài liệu Network
Engineering, kèm ví dụ lấy từ chính dự án này.

---

## 1. Mô hình Tham chiếu & Quá trình Đóng gói Dữ liệu (Encapsulation)

### 1.1. Mô hình OSI (7 Lớp) và TCP/IP (4 Lớp)
- **Layer 7 - Application (Ứng dụng):** Nơi phần mềm (Chrome, Nginx) hoạt động. Giao thức: HTTP, HTTPS, SSH, DNS.
- **Layer 6 - Presentation (Trình diễn):** Mã hóa, nén dữ liệu (SSL/TLS mã hóa Payload ở đây).
- **Layer 5 - Session (Phiên):** Quản lý phiên làm việc giữa các ứng dụng.
- **Layer 4 - Transport (Giao vận):** Đảm bảo truyền tải. Quản lý các Cổng (Port). Giao thức: TCP, UDP.
- **Layer 3 - Network (Mạng):** Định tuyến đường đi toàn cầu. Giao thức: IP, ICMP (Ping).
- **Layer 2 - Data Link (Liên kết dữ liệu):** Truyền dữ liệu trong mạng LAN vật lý bằng MAC Address. Giao thức: Ethernet, Wi-Fi.
- **Layer 1 - Physical (Vật lý):** Truyền tải tín hiệu điện/quang/sóng vô tuyến qua dây dẫn.

### 1.2. Cấu tạo Gói tin & Quá trình Đóng gói (Encapsulation)
1. **Payload (Lớp 7):** Khối dữ liệu gốc thực sự (Ví dụ: chuỗi JSON chứa thông tin user).
2. **Segment / Datagram (Lớp 4):** Payload được bọc thêm **TCP Header** (chứa Source/Dest Port) thành **Segment**.
3. **Packet (Lớp 3):** Segment được bọc thêm **IP Header** (chứa Source/Dest IP) thành **Packet**.
4. **Frame (Lớp 2):** Packet đi xuống Card mạng, được bọc thêm **MAC Header** và đuôi kiểm lỗi FCS thành **Frame**.
5. **Bitstream (Lớp 1):** Băm nhỏ thành các xung điện (0 và 1) phóng ra cáp mạng.

### 1.3. Triết lý Thiết kế: Tại sao lại sinh ra MAC, IP và Port?
Nhiều người thắc mắc: *"Tại sao phải đẻ ra tận 3 loại địa chỉ lằng nhằng này? Sao không dùng chung 1 cái?"*. Đây là câu trả lời triệt để:

- **MAC Address (Định danh Phần cứng):** Nó giống như **Căn cước Công dân (CCCD)** hoặc Số khung xe máy. Nó là định danh bất di bất dịch, gán chết vào con chip phần cứng của Card mạng từ nhà máy. Nó giúp nhận diện đích danh thiết bị trong phạm vi hẹp (LAN). Nhưng CCCD không cho biết bạn đang sống ở quốc gia nào! Nếu chỉ dùng MAC, Internet sẽ không thể nào tìm đường (Route) để đẩy gói tin.
- **IP Address (Định danh Vị trí / Địa lý):** Nó giống như **Địa chỉ Thường trú (Số nhà, Tên đường, Thành phố)**. Khi bạn xách laptop từ Việt Nam sang Mỹ, số MAC (CCCD) của bạn không đổi, nhưng số IP (Địa chỉ nhà) của bạn bắt buộc phải đổi để thế giới biết bạn đang ở Mỹ mà gửi thư đến.
  - *Ai là người cấp phát IP?* IP không tự sinh ra. Khi máy tính cắm dây vào mạng (hoặc bắt Wi-Fi), cục Modem/Router của nhà mạng sẽ chạy một giao thức gọi là **DHCP (Dynamic Host Configuration Protocol)** để động cấp phát (cho thuê) một IP Private và gắn nó với cái MAC Address của bạn. Khi bạn ngắt mạng rời đi, Router sẽ thu hồi IP đó lại và cấp cho người khác.
  - *Tại sao 1 máy (1 MAC) lại có nhiều IP?* Một máy tính có thể cắm cáp LAN (nhận IP A), đồng thời bật Wi-Fi (nhận IP B), đồng thời chạy phần mềm VPN (nhận IP ảo C). IP là thứ vô hình, đại diện cho một kết nối logic (Logical Interface) chứ không bị trói buộc 1-1 với phần cứng.
- **Port (Định danh Tiến trình Phần mềm):** Nó giống như **Số Phòng trong 1 Tòa nhà**. Khi bưu tá (Hệ điều hành) dựa vào IP để mang bưu phẩm đến đúng tòa nhà (Máy tính), họ mở cửa bước vào và thấy trong nhà có... 100 phần mềm đang chạy cùng lúc (Chrome, Zalo, Nginx, Database...). Bưu phẩm này dành cho phần mềm nào? Port chính là Số phòng. Gói tin gắn Port 80 sẽ được HĐH ném cho Nginx, gắn Port 5432 sẽ được ném cho Postgres. Một IP có 65535 Port, giúp máy tính có thể làm hàng vạn việc song song!

---

## 2. Cấu trúc Hạ tầng Vật lý & Logic Truyền tải

### 2.1. Card mạng (NIC) & Chuyển đổi Tín hiệu (Layer 1 & 2)
- **Modulation (Điều chế tín hiệu):** Máy tính chỉ hiểu 0 và 1. NIC dịch các bit này thành vật lý.
  - *Cáp đồng:* Dịch bit `1` thành điện áp +5V, bit `0` thành 0V.
  - *Cáp quang:* Kích hoạt đèn tia laser chớp tắt cực nhanh.
  - *Wifi:* Biến thiên tần số hoặc biên độ sóng vô tuyến (Radio Waves).

### 2.2. Switch & Cơ chế Broadcast / ARP (Layer 2)
Switch là "Trạm thu phí" nội bộ của mạng LAN.
- **MAC Address Table (Lưu trên Switch):** Bảng nhớ lưu ánh xạ giữa Cổng (Port vật lý cắm dây) và MAC Address. *(Lưu ý cực kỳ quan trọng: Bảng này KHÔNG chứa IP. Switch hoàn toàn "mù" về IP. Nó chỉ biết "Lỗ cắm cáp số 1 đang nối với thiết bị có MAC là A")*.
- **ARP Table (Lưu trên Máy tính / Router):** Đây mới là bảng lưu ánh xạ (Mapping) giữa `IP Address` và `MAC Address` (Ví dụ: `192.168.1.1` đi liền với MAC `00:1A:2B...`).
- **Cơ chế Broadcast & Flooding (ARP):** Khi Máy A biết IP của Máy B, nhưng chưa biết MAC, Máy A gửi một Frame **Broadcast** tới MAC đích là `FF:FF:FF:FF:FF:FF` (nghĩa là "Gửi tất cả"). Switch nhận được liền "hét lên" (Flooding) đẩy dòng điện ra tất cả các cổng. Máy B nghe thấy, trả lời lại số MAC thật. Từ đó Switch cập nhật bảng nhớ và chỉ truyền trực tiếp (Unicast).

### 2.3. Router & Bản đồ Định tuyến (Layer 3)
Router kết nối các mạng LAN khác nhau. Router "xé" Lớp 2 (Frame), đọc Lớp 3 (Packet) lấy **Destination IP**. Router dò **Bảng định tuyến (Routing Table)** để tìm đường (Cổng mạng) gần nhất, đóng bọc Lớp 2 mới và đẩy tín hiệu đi (Hop-by-Hop).

---

## 3. Bản chất IP, Subnet và Giao thức BGP Toàn cầu

### 3.1. Phân lớp Địa chỉ IPv4 (Classful Addressing)
Vào thời kỳ sơ khai của Internet, tổ chức IANA chưa có khái niệm chia nhỏ Subnet (CIDR) như bây giờ. Họ chặt không gian 4.3 tỷ địa chỉ IPv4 thành 5 Lớp (Classes) cố định dựa vào những bit đầu tiên của địa chỉ. Việc hiểu rõ 5 lớp này là nền tảng để phân biệt đâu là mạng khổng lồ, đâu là mạng nhỏ, và đâu là mạng chuyên dụng.

- **Lớp A (Class A):** Dành cho các Mạng lưới Khổng lồ (Chính phủ, Nhà mạng)
  - **Dải IP:** Từ `1.0.0.0` đến `126.255.255.255`.
  - **Đặc điểm:** Mặc định sử dụng Subnet Mask `/8`. Nghĩa là chỉ dùng 1 Octet đầu để định danh Mạng, dành tận 3 Octet sau cho Host. Một mạng Lớp A có thể chứa tới **16.7 triệu máy tính**.
  - *(Lưu ý: Dải `127.x.x.x` bị bỏ qua vì nó được dùng làm Loopback Address (Localhost) để máy tính tự gọi chính mình).*

- **Lớp B (Class B):** Dành cho các Tổ chức/Doanh nghiệp quy mô Vừa và Lớn
  - **Dải IP:** Từ `128.0.0.0` đến `191.255.255.255`.
  - **Đặc điểm:** Mặc định sử dụng Subnet Mask `/16`. Dùng 2 Octet cho Mạng, 2 Octet cho Host. Mỗi mạng Lớp B chứa được khoảng **65,536 máy tính**.

- **Lớp C (Class C):** Dành cho các Mạng Nhỏ (Quán Cafe, Gia đình, Văn phòng chi nhánh)
  - **Dải IP:** Từ `192.0.0.0` đến `223.255.255.255`.
  - **Đặc điểm:** Mặc định sử dụng Subnet Mask `/24`. Dùng 3 Octet cho Mạng, chỉ chừa 1 Octet cuối cho Host. Một mạng Lớp C chỉ chứa tối đa **254 máy tính**.

- **Lớp D (Class D):** Chuyên dụng cho Multicast (Truyền phát đa hướng)
  - **Dải IP:** Từ `224.0.0.0` đến `239.255.255.255`.
  - **Đặc điểm:** Lớp này KHÔNG dùng để cấp cho máy tính cá nhân (Không có Network ID hay Host ID). Nó được dùng để truyền tin **Multicast** (Gửi 1 gói tin nhưng nhiều máy nhận cùng lúc).
  - *Ví dụ thực tiễn:* Truyền hình cáp IPTV (Khi VTV phát 1 trận bóng đá trực tiếp, họ không gửi 1 triệu gói tin cho 1 triệu tivi, họ chỉ gửi 1 gói tin Multicast vào dải Lớp D, tivi nào muốn xem thì "Subscribe" vào nhóm Multicast đó để nhận). Hoặc dùng cho các con Router nói chuyện với nhau (Giao thức OSPF).

- **Lớp E (Class E):** Dành cho Nghiên cứu & Quân sự (Reserved)
  - **Dải IP:** Từ `240.0.0.0` đến `255.255.255.255`.
  - **Đặc điểm:** Lớp này bị niêm phong, không bao giờ được cấp phát ra Internet công cộng. Cục IETF giữ lại để nghiên cứu và dự phòng. Riêng địa chỉ cuối cùng `255.255.255.255` được quy ước làm địa chỉ **Broadcast** toàn cầu.

> 💡 **Bí ẩn Toán học: Tại sao lại sinh ra các con số 126, 128, 192?**
> Vào những năm 1980, bộ định tuyến (Router) cực kỳ yếu. Để giúp Router nhận diện một IP thuộc Lớp nào nhanh như chớp, các nhà khoa học máy tính đã dùng cơ chế **"Bit dẫn đầu" (High-Order Bits)** của Octet đầu tiên làm lá cờ hiệu (Flag):
> - **Lớp A:** Bit dẫn đầu luôn khóa cứng là `0`. (Dải nhị phân từ `00000000` đến `01111111` ➔ Đổi ra thập phân chính là **0 đến 127**).
> - **Lớp B:** Bit dẫn đầu luôn khóa cứng là `10`. (Dải nhị phân từ `10000000` đến `10111111` ➔ Đổi ra thập phân là **128 đến 191**).
> - **Lớp C:** Bit dẫn đầu luôn khóa cứng là `110`. (Dải nhị phân từ `11000000` đến `11011111` ➔ Đổi ra thập phân là **192 đến 223**).
> - **Lớp D:** Khóa cứng `1110` ➔ Thập phân **224 đến 239**.
> - **Lớp E:** Khóa cứng `1111` ➔ Thập phân **240 đến 255**.
> Chính luật khóa bit thiên tài này đã tự động sinh ra những ranh giới toán học bất di bất dịch của Internet sơ khai mà không cần đến Subnet Mask!

> ⚠️ **Sự nhầm lẫn kinh điển (Public vs Private):**
> Đừng nhầm lẫn giữa Lớp IP (Classful) và IP Nội bộ (Private IP). Ví dụ: Lớp A kéo dài từ `1` đến `126`, nhưng IANA chỉ trích ra ĐÚNG MỘT DẢI DUY NHẤT trong đó là `10.0.0.0/8` để làm IP Private. Toàn bộ các IP Lớp A còn lại (như `8.8.8.8` của Google, hay `1.1.1.1` của Cloudflare) đều là IP Public được mua bằng tiền tỷ! Tương tự với Lớp B (trích dải `172.16`) và Lớp C (trích dải `192.168`).

### 3.2. Subnet Mask (Mặt nạ mạng), CIDR & Phép toán Bitwise AND
Dải IP luôn đi kèm Subnet Mask (vd `/24` hoặc `255.255.255.0`). Nhiệm vụ duy nhất của mặt nạ mạng là làm cái thớt để "chặt" địa chỉ IP ra làm 2 phần: **Network ID** (Khu phố) và **Host ID** (Số nhà). Việc tự do gán Subnet Mask không màng tới Lớp A,B,C được gọi là chuẩn CIDR (Classless Inter-Domain Routing).

**Cách xác định Khúc nào là Network, Khúc nào là Host:**
- Bản chất IP IPv4 gồm 4 cụm số (4 Octet), tổng cộng 32-bit.
- Con số `/24` mang ý nghĩa: **24 bit đầu tiên** (tương đương 3 cụm số đầu) bị "khóa cứng" làm **Network ID**. 8 bit còn lại (cụm số cuối) được tự do cấp phát làm **Host ID**.
- Nhìn dưới góc độ số thập phân, `/24` tương đương với `255.255.255.0`. Số `255` mang ý nghĩa "Khóa chặt phần này làm Network", số `0` mang ý nghĩa "Phần này là Host".
- *Ví dụ với IP `192.168.1.5` đi kèm mask `255.255.255.0`:*
  - **Network ID:** `192.168.1.0` (3 cụm đầu bị khóa cứng. Đây chính là "Tên khu phố").
  - **Host ID:** `.5` (Cụm cuối cùng tự do. Đây là "Số nhà" của bạn trong khu phố đó).

**Sự đánh đổi (Trade-off) giữa Network và Host:**
- Tổng số bit luôn là 32. Nếu bạn lấy nhiều bit làm Network, bạn sẽ có ít bit làm Host (số IP cấp phát được ít đi), và ngược lại.
- Với Subnet `/24` (24 bit Network, 8 bit Host), Router/DHCP của bạn chỉ có thể cấp tối đa $2^8 - 2 = 254$ số nhà (IP) cho các thiết bị truy cập. (Lý do trừ 2: IP `.0` dành riêng cho tên Network, IP `.255` dành riêng cho lệnh Broadcast hét lên toàn mạng).
- Nếu hệ thống của bạn có 1000 user truy cập, dải `/24` sẽ không đủ IP để cấp! Lúc này bạn phải mượn thêm bit, ví dụ xài dải `/22` (22 bit Network, 10 bit Host). Lúc này Router sẽ có $2^{10} - 2 = 1022$ cái IP để tha hồ cấp phát.
- *Câu hỏi đặt ra:* **Tại sao không xài luôn Subnet `/8` để có tận 16 triệu IP ($2^{24}$) cấp cho sướng, khỏi lo hết IP?**
  1. **Thảm họa Bão Broadcast (Broadcast Storm):** Nhớ lại giao thức ARP ở Lớp 2, khi 1 máy muốn tìm MAC, nó phải hét lên (Broadcast) cho TẤT CẢ các máy trong cùng mạng LAN. Nếu bạn nhét 16 triệu thiết bị vào chung 1 mạng LAN `/8`, chỉ cần 1% số máy đó hét lên tìm MAC thôi là phần cứng Switch sẽ bốc cháy, toàn bộ 16 triệu máy tính phải dừng công việc lại để lắng nghe rác. Mạng lưới sẽ sập ngay lập tức!
  2. **Vấn đề Bảo mật (Security):** Trong cùng 1 mạng LAN, các máy tính nói chuyện trực tiếp với nhau qua Switch mà không qua tường lửa của Router. Nếu nhét cả Server DB, máy Kế Toán, và Guest Wi-Fi vào chung 1 dải `/8`, một con virus từ máy khách có thể scan và lây lan sang máy Server ngay lập tức. Bằng cách băm nhỏ ra thành nhiều Subnet `/24` khác nhau, muốn đi từ Subnet Guest sang Subnet DB bắt buộc phải đi vòng qua Router, lúc này Tường lửa trên Router sẽ chặn đứng con virus lại!
  3. **Giới hạn phần cứng Switch:** Switch lưu bảng MAC Address Table trên bộ nhớ RAM đắt tiền (CAM Table). Không có con Switch vật lý nào đủ RAM để chứa được 16 triệu cái MAC Address cả. Khi đầy RAM, nó sẽ biến thành một cục Hub ngu ngốc, xả rác ra mọi cổng.

**Cơ chế định tuyến nội bộ thông minh của HĐH:**
Làm sao máy tính của bạn biết lúc nào thì ném gói tin cho Switch (trong nhà), lúc nào thì ném cho Router (ra ngoài Internet)? Nó dùng **Phép toán Bitwise AND**.
*Ví dụ: Máy tính của bạn có IP `192.168.1.5`, Subnet Mask là `255.255.255.0`.*
- *Lưu ý toán học:* Tại sao `192.168.1.5 [AND] 255.255.255.0` lại ra `192.168.1.0`? Vì phép `AND` nhị phân quy định `1 AND 1 = 1`, còn lại bằng `0`. Số `255` là `11111111`, khi AND với số nào thì giữ nguyên số đó. Còn số `0` là `00000000`, khi AND với số `5` (tức `00000101`) sẽ triệt tiêu về `0`. Do đó kết quả bóc tách ra được phần Network ID là `192.168.1.0`!
- **Trường hợp 1 (Gửi cho máy tính kế bên `192.168.1.10`):** Máy bạn lấy IP đích `192.168.1.10 [AND] 255.255.255.0 = 192.168.1.0`. Máy bạn so sánh thấy Network ID này **TRÙNG** với Network ID của nó. Thế là nó tra bảng ARP lấy MAC rồi quăng thẳng Frame vào **Switch**.
- **Trường hợp 2 (Gửi cho Google `8.8.8.8`):** Máy bạn lấy IP đích `8.8.8.8 [AND] 255.255.255.0 = 8.8.8.0`. Máy bạn so sánh thấy **KHÁC** với Network ID của mình (`192.168.1.0`). Nó bọc gói tin lại và ném thẳng cho **Router (Default Gateway)** để Router tự lo liệu tìm đường.

### 3.2. BGP (Border Gateway Protocol) - Bản Đồ Định Tuyến Của Internet
Làm sao máy tính ở Việt Nam biết đường bắn gói tin sang tận máy chủ AWS ở Mỹ mà không bị lạc giữa hàng triệu rễ cây cáp quang dưới đáy biển? Câu trả lời là nhờ giao thức **BGP**.

- **Hệ thống tự trị (Autonomous System - AS):** Internet không có một máy chủ trung tâm hay "Giám đốc" nào điều phối cả. Nó là một mạng lưới chắp vá khổng lồ từ hàng vạn hệ thống mạng độc lập của các tập đoàn (Ví dụ: Mạng của Viettel là một AS, mạng của FPT là một AS, mạng của Google/AWS là một AS). Mỗi tổ chức này được tổ chức quản lý mạng thế giới cấp một mã số định danh duy nhất gọi là **ASN**.
- **Cơ chế "Ngoại giao" (Peering & Transit):** BGP là ngôn ngữ ngoại giao giữa các quốc gia AS. Các Router khổng lồ ở biên giới sẽ liên tục phát các bản tin **BGP Announcements (Loan báo định tuyến)**. Trong thế giới thực, hành động "Ngoại giao" này được vận hành dựa trên 2 hình thức thương mại (Tiền bạc) cốt lõi:
  - **Peering (Kết nối ngang hàng - Miễn phí):** Hai AS (Ví dụ: VNPT và Google) kéo cáp nối trực tiếp Router với nhau (thường tại các trạm trung chuyển Internet - IXP). Họ thỏa thuận trao đổi dữ liệu miễn phí (Settlement-free). Nhờ đó, user VNPT xem Youtube cực nhanh vì data chạy thẳng từ máy chủ Google sang thẳng VNPT mà không qua trung gian. BGP lúc này chỉ loan báo dải IP của riêng 2 bên cho nhau.
  - **Transit (Mua đường truyền - Trả phí):** VNPT không thể tự kéo cáp nối trực tiếp với 100,000 cái mạng AS trên toàn thế giới được. VNPT bắt buộc phải vác tiền đi thuê một "Ông trùm" mạng viễn thông cấp 1 (Tier-1 ISP như AT&T, Tata). BGP Router của Tier-1 sẽ loan báo cho VNPT: *"Tao có bản đồ đường đi đến TOÀN BỘ thế giới, cứ quăng hết data cho tao, tao tính tiền cước theo dung lượng (GB)"*.
- **Thuật toán tìm đường:** BGP không hề chọn đường đi dựa vào cáp quang nào nhanh nhất, mà nó chọn đường theo **AS Path** (Chặng đường đi qua ít quốc gia AS nhất) và đặc biệt là theo **Chính sách Kinh doanh**.
  - *Ví dụ thực tiễn:* Kỹ sư mạng VNPT sẽ viết code cấu hình BGP ưu tiên đẩy Traffic qua đường **Peering** (vì nó miễn phí). Chỉ khi cáp Peering đứt, Router mới ngậm ngùi bẻ lái (Failover) đẩy traffic qua đường **Transit** đắt tiền. Sự điều hướng dòng chảy Data trên toàn cầu thực chất được quyết định bởi... Bài toán Kinh tế!
- *Thảm họa thực tế:* Nhờ BGP mà các Packet luôn tìm được đường tới đích. Nhưng nếu một kỹ sư mạng ở Facebook gõ sai cấu hình, vô tình xóa mất lời "loan báo" BGP của hệ thống Facebook, thì toàn bộ dải IP của Facebook sẽ lập tức "tàng hình" khỏi bản đồ Internet thế giới, khiến mạng lưới sập toàn cầu (Điều này đã từng xảy ra trên thực tế khiến toàn bộ Facebook, Messenger, Instagram sập sạch vào tháng 10/2021).

### 3.3. Ứng dụng Thực tiễn: Quy hoạch VPC trên Cloud (AWS)
Từ mớ lý thuyết khô khan về Subnet Mask và Private IP ở trên, khi áp dụng vào thực tế triển khai Cloud (AWS/GCP), các Kỹ sư Hệ thống sẽ tư duy quy hoạch hạ tầng (Infrastructure as Code) như sau:

**1. Khai báo Mạng ảo (Virtual Private Cloud - VPC):**
AWS yêu cầu bạn phải tạo một "Mảnh đất tổng" (VPC) để chứa toàn bộ Server. Tiêu chuẩn vàng (Golden Standard) luôn là chọn dải IP **Lớp A (`10.0.0.0/16`)**. Tại sao?
- **Tránh đụng độ Lớp C:** Tuyệt đối không dùng `192.168.0.0/16` vì 99% cục Router Wi-Fi gia đình đang xài dải này. Nếu bạn bật VPN từ nhà kết nối lên Cloud, HĐH máy tính của bạn sẽ bị "lú" vì Routing Table bị trùng lặp, không phân biệt được IP Server trên Cloud và IP cái Tivi thông minh ở nhà.
- **Tránh đụng độ Lớp B:** Không dùng dải `172.x.x.x` vì mạng mặc định của AWS (Default VPC) luôn lấy dải `172.31.0.0/16`, và mạng ảo của Docker Containers cũng tự động xí phần dải `172.17.0.0/16`.
- Dải **Lớp A (`10.0.0.0/16`)** cực kỳ rộng lớn (65,536 IP) và là dải sạch sẽ nhất, an toàn tuyệt đối khi cần đấu nối mạng nối mạng ghép (VPC Peering/Transit Gateway) với công ty đối tác.

**2. Băm nhỏ Mảnh đất (Subnetting):**
Từ mảnh đất khổng lồ `/16`, hệ thống bắt buộc phải cắt nhỏ thành các khu phố (Subnet) hẹp hơn, thường dùng Subnet `/24` (256 IP mỗi khu):
- `10.0.1.0/24`: **Public Subnet** (Khu mặt tiền). Cắm Internet Gateway, cấp Public IP. Dành cho Web Server, Load Balancer.
- `10.0.2.0/24`: **Private Subnet** (Khu hầm ngầm). Không có đường ra Internet. Dành cho Database.

Cách phân lô bằng số nhị phân này giúp giới hạn phạm vi ảnh hưởng khi có sự cố
(Blast Radius). Trong `network.tf` của Terraform, dòng `cidr_block = "10.0.1.0/24"`
đã đủ để suy ra quy mô dải mạng được cấp.

### 3.4. Ánh xạ Kiến trúc Mạng Cơ bản sang Điện toán Đám mây (AWS)
Khi đưa các kiến thức mạng truyền thống lên môi trường Đám mây (Cloud), AWS đã đóng gói chúng thành các khái niệm trực quan. Việc nắm rõ mối liên kết này giúp bạn thiết kế hạ tầng chuẩn mực (Best Practice):

**1. Region và Availability Zone (A-Z) - Nền tảng của sự Bền bỉ (High Availability)**
- **Region (Khu vực):** Là một cụm Data Center nằm ở một vị trí địa lý cụ thể (Ví dụ: Singapore - `ap-southeast-1`). Một mạng ảo **VPC luôn bị "khóa chặt" trong 1 Region**. Bạn không thể tạo 1 VPC vắt ngang qua Singapore và Mỹ.
- **Availability Zone (AZ - Vùng khả thi):** Bên trong 1 Region sẽ có nhiều AZ (Ví dụ: `ap-southeast-1a`, `ap-southeast-1b`). Mỗi AZ là một Data Center biệt lập vật lý hoàn toàn, có hệ thống điện lưới và cáp quang độc lập để phòng chống thiên tai (động đất, cháy nổ).
- **High Availability (H.A - Khả năng sẵn sàng cao):** Để đảm bảo hệ thống không bao giờ sập, các Kỹ sư (DevOps) luôn thiết kế mô hình H.A: Chạy 2 con EC2 Backend giống hệt nhau, nhưng đặt ở 2 AZ khác nhau. Nếu AZ A sập nguồn điện, hệ thống Load Balancer sẽ tự động dồn toàn bộ khách hàng sang AZ B.

**2. Ranh giới mạng: Subnet**
- Khác với VPC bao trùm cả Region, **Một Subnet bị khóa chặt trong đúng 1 AZ**.
- Các Subnet trong cùng 1 VPC (Dù ở khác AZ) mặc định luôn "nhìn thấy nhau" và giao tiếp nội bộ miễn phí thông qua luật `local` mặc định của mạng.

**3. Công tắc hòa mạng Internet: Route Table và Internet Gateway (IGW)**
- Khi tạo VPC, nó hoàn toàn bị cô lập (Isolated). Để kết nối Internet, bạn phải gắn một cái cửa ngõ gọi là **Internet Gateway (IGW)** vào VPC.
- Gói tin từ bên ngoài vào: IGW hoạt động như một cỗ máy NAT khổng lồ, nó biên dịch IP Public sang IP Private của EC2 và ném gói tin thẳng về Subnet chứa EC2.
- Gói tin từ trong đi ra (Quyết định bởi **Route Table**): Bảng định tuyến (Route Table) là bộ não chỉ đường. Nó định nghĩa ranh giới giữa Public và Private:
  - **Public Subnet (Vùng mặt tiền):** Là Subnet có Route Table chứa luật `0.0.0.0/0 -> IGW`. Luật này nói rằng: *"Nếu gói tin muốn đi tới một IP lạ không nằm trong mạng nội bộ, hãy ném nó ra cổng IGW ra Internet"*. Nơi đây chứa Load Balancer, Web Server.
  - **Private Subnet (Vùng hầm ngầm):** Là Subnet mà Route Table của nó KHÔNG CÓ luật trỏ ra IGW. Các Server (như Database) đặt ở đây an toàn tuyệt đối, vì Hacker từ Internet không có cách nào chọc vào được (kể cả khi Server có lỗ hổng bảo mật).

**4. Tường lửa 2 lớp: Security Group (SG) vs Network ACL (NACL)**
Mạng truyền thống thường chỉ có 1 Firewall tổng ở cổng công ty. Cloud AWS thông minh hơn, phân chia Firewall thành 2 lớp:
- **Security Group (Tường lửa Cấp EC2):** Nó bọc quanh từng cái Card mạng (ENI) của một con máy ảo (EC2) cụ thể. Gắn SG cho con Gateway nào thì nó chỉ bảo vệ đúng con Gateway đó. 
  - *Tính chất:* Luôn "Mở" (Mặc định khóa chặt, bạn chỉ được phép viết luật CHO PHÉP). Ví dụ: Cho phép Port 80 và 443 từ IP của Cloudflare đi vào.
- **Network ACL - NACL (Tường lửa Cấp Subnet):** Nó bọc ở cổng ra/vào của toàn bộ 1 Subnet. 
  - *Tính chất:* Luôn "Chặn" (Mặc định cho phép tất cả, bạn dùng NACL để viết luật CHẶN). Ví dụ: Phát hiện 1 dải IP từ nước ngoài liên tục rà quét mạng, bạn dùng NACL để DROP ngay lập tức từ vòng gửi xe của Subnet, không cho chúng chạm tới Security Group của EC2.

### 3.4. Ánh xạ Kiến trúc Mạng Cơ bản sang Điện toán Đám mây (AWS)
Khi đưa các kiến thức mạng truyền thống lên môi trường Đám mây (Cloud), AWS đã đóng gói chúng thành các khái niệm trực quan. Việc nắm rõ mối liên kết này giúp bạn thiết kế hạ tầng chuẩn mực (Best Practice):

**1. Region và Availability Zone (A-Z) - Nền tảng của sự Bền bỉ (High Availability)**
- **Region (Khu vực):** Là một cụm Data Center nằm ở một vị trí địa lý cụ thể (Ví dụ: Singapore - `ap-southeast-1`). Một mạng ảo **VPC luôn bị "khóa chặt" trong 1 Region**. Bạn không thể tạo 1 VPC vắt ngang qua Singapore và Mỹ.
- **Availability Zone (AZ - Vùng khả thi):** Bên trong 1 Region sẽ có nhiều AZ (Ví dụ: `ap-southeast-1a`, `ap-southeast-1b`). Mỗi AZ là một Data Center biệt lập vật lý hoàn toàn, có hệ thống điện lưới và cáp quang độc lập để phòng chống thiên tai (động đất, cháy nổ).
- **High Availability (H.A - Khả năng sẵn sàng cao):** Để đảm bảo hệ thống không bao giờ sập, các Kỹ sư (DevOps) luôn thiết kế mô hình H.A: Chạy 2 con EC2 Backend giống hệt nhau, nhưng đặt ở 2 AZ khác nhau. Nếu AZ A sập nguồn điện, hệ thống Load Balancer sẽ tự động dồn toàn bộ khách hàng sang AZ B.

**2. Ranh giới mạng: Subnet**
- Khác với VPC bao trùm cả Region, **Một Subnet bị khóa chặt trong đúng 1 AZ**.
- Các Subnet trong cùng 1 VPC (Dù ở khác AZ) mặc định luôn "nhìn thấy nhau" và giao tiếp nội bộ miễn phí thông qua luật `local` mặc định của mạng.

**3. Công tắc hòa mạng Internet: Route Table và Internet Gateway (IGW)**
- Khi tạo VPC, nó hoàn toàn bị cô lập (Isolated). Để kết nối Internet, bạn phải gắn một cái cửa ngõ gọi là **Internet Gateway (IGW)** vào VPC.
- Gói tin từ bên ngoài vào: IGW hoạt động như một cỗ máy NAT khổng lồ, nó biên dịch IP Public sang IP Private của EC2 và ném gói tin thẳng về Subnet chứa EC2.
- Gói tin từ trong đi ra (Quyết định bởi **Route Table**): Bảng định tuyến (Route Table) là bộ não chỉ đường. Nó định nghĩa ranh giới giữa Public và Private:
  - **Public Subnet (Vùng mặt tiền):** Là Subnet có Route Table chứa luật `0.0.0.0/0 -> IGW`. Luật này nói rằng: *"Nếu gói tin muốn đi tới một IP lạ không nằm trong mạng nội bộ, hãy ném nó ra cổng IGW ra Internet"*. Nơi đây chứa Load Balancer, Web Server.
  - **Private Subnet (Vùng hầm ngầm):** Là Subnet mà Route Table của nó KHÔNG CÓ luật trỏ ra IGW. Các Server (như Database) đặt ở đây an toàn tuyệt đối, vì Hacker từ Internet không có cách nào chọc vào được (kể cả khi Server có lỗ hổng bảo mật).

**4. Tường lửa 2 lớp: Security Group (SG) vs Network ACL (NACL)**
Mạng truyền thống thường chỉ có 1 Firewall tổng ở cổng công ty. Cloud AWS thông minh hơn, phân chia Firewall thành 2 lớp:
- **Security Group (Tường lửa Cấp EC2):** Nó bọc quanh từng cái Card mạng (ENI) của một con máy ảo (EC2) cụ thể. Gắn SG cho con Gateway nào thì nó chỉ bảo vệ đúng con Gateway đó. 
  - *Tính chất:* Luôn "Mở" (Mặc định khóa chặt, bạn chỉ được phép viết luật CHO PHÉP). Ví dụ: Cho phép Port 80 và 443 từ IP của Cloudflare đi vào.
- **Network ACL - NACL (Tường lửa Cấp Subnet):** Nó bọc ở cổng ra/vào của toàn bộ 1 Subnet. 
  - *Tính chất:* Luôn "Chặn" (Mặc định cho phép tất cả, bạn dùng NACL để viết luật CHẶN). Ví dụ: Phát hiện 1 dải IP từ nước ngoài liên tục rà quét mạng, bạn dùng NACL để DROP ngay lập tức từ vòng gửi xe của Subnet, không cho chúng chạm tới Security Group của EC2.

---

## 4. NAT (Network Address Translation)

Vào những năm 1990, thế giới nhận ra thảm họa: IPv4 chỉ có 4.3 tỷ địa chỉ Public, sớm muộn gì cũng cạn kiệt. Nếu mỗi cái điện thoại, máy tính, tủ lạnh đều chiếm 1 IP Public thì Internet đã "chết" từ năm 2000. Giải pháp vĩ đại được đẻ ra chính là NAT.

Tổ chức IANA quy hoạch ra các dải **IP Private** (`10.x.x.x`, `172.16.x.x`, `192.168.x.x`). IP Private hoàn toàn "vô hình" trên Internet (Các Router công cộng giữa đại dương sẽ DROP ngay lập tức nếu thấy IP Private). NAT cho phép hàng nghìn máy tính trong mạng LAN dùng IP Private, nhưng khi ra Internet thì "núp bóng" chung **Duy nhất 1 IP Public** của Router.

**Cách NAT hoạt động (PAT - Port Address Translation):**
Làm sao 1 con Router với 1 IP Public có thể gánh cho 10 người cùng lướt web mà không bị gửi nhầm dữ liệu của người này sang màn hình người kia? Bí mật nằm ở việc Router không chỉ tráo IP, mà nó tráo luôn cả Cổng (Port)!

- **1. NAT Outbound (Chiến dịch tráo nhãn đi ra):**
  - Bạn mở Chrome, máy bạn (`192.168.1.5`) dùng Port ngẫu nhiên `51000` đẩy Packet ra Router.
  - Router chặn Packet lại, xé bọc Lớp 3 (Network) và Lớp 4 (Transport).
  - Nó xóa `IP Nguồn: 192.168.1.5` và xóa luôn `Port Nguồn: 51000`.
  - Nó đè **IP Public** của nó vào (vd `14.22.33.44`), và chọn ngẫu nhiên một cái Port trống của nó (vd `33333`) gắn vào Packet. Rồi ném ra Internet.
  - Đồng thời, Router lấy sổ tay ra ghi chú vào **NAT Session Table**: *"Mình vừa lấy Port `33333` của mình để đi chợ dùm thằng cu `192.168.1.5:51000` trong nhà"*.

- **2. NAT Inbound (Chiến dịch tráo nhãn trả về):**
  - Web Server (như Facebook) xử lý xong, gửi Response trả về đúng IP Public và Port của Router (`14.22.33.44:33333`).
  - Router nhận được gói hàng. Nó mở sổ tay **NAT Session Table** ra dò: *"À, gói hàng gửi vào Port 33333 là đồ của thằng cu 192.168.1.5:51000"*.
  - Router bóc lớp vỏ `14.22.33.44:33333` vứt đi, dán đè nhãn `192.168.1.5:51000` vào và bơm ngược gói tin vào mạng LAN thẳng tới máy tính của bạn. Gói tin đi xuyên suốt mà Trình duyệt không hề hay biết đã bị tráo nhãn!

- **3. NAT Gateway trên Cloud (AWS/GCP):**
  - Trên AWS, các máy chủ Backend/Database luôn được đặt ở **Private Subnet** (chỉ có IP Private để bảo mật tuyệt đối, chống Hacker từ Internet rà quét).
  - Nhưng nếu không có IP Public, làm sao Server ra ngoài tải được bản cập nhật Linux (apt update)?
  - Lúc này DevOps phải tạo một con **NAT Gateway** (hoặc NAT Instance) đặt ở Public Subnet. Con NAT Gateway này hoạt động y chang cục Router ở nhà bạn: Nó gom tất cả request mồ côi từ các Server nội bộ, lấy danh nghĩa IP Public của nó đi tải bản cập nhật về, rồi bóc nhãn chuyển trả vào lại cho từng Server nội bộ tương ứng một cách an toàn.

---

## 5. Mật mã học Cơ bản & Nguyên lý SSL/TLS

### 5.1. Bài toán: Tại sao cần SSL/TLS?
Internet nguyên thủy truyền dữ liệu dạng văn bản rõ (Clear Text / HTTP). Một gói tin đi từ VN sang Mỹ phải nhảy qua hàng tá Router (Viettel, Singtel...). Bất kỳ kỹ sư mạng nào ở các trạm trung gian (hoặc Hacker nghe lén Wi-Fi quán cafe) đều có thể bắt gói tin (Sniffing) và đọc được Mật khẩu, Thẻ tín dụng của bạn. 
Giải pháp: Giao thức **SSL/TLS (HTTPS)** sinh ra để "băm nát" và mã hóa cái Payload trước khi ném xuống tầng TCP.

### 5.2. Sự kết hợp hoàn hảo giữa 2 loại Mã hóa (Cơ chế Toán học)
Nếu chỉ dùng 1 loại mã hóa thì mạng Internet sẽ hoặc là "Quá chậm", hoặc là "Quá kém bảo mật". Do đó SSL/TLS đã kết hợp cả 2 loại với nhau:

1. **Mã hóa Bất đối xứng (Asymmetric Cryptography - Tiêu biểu: RSA, ECC):**
   - **Bản chất Toán học:** Dựa trên các bài toán hàm một chiều (Trapdoor function) mà máy tính dễ dàng tính xuôi nhưng gần như không thể giải ngược (nếu không có tham số bí mật). Ví dụ, thuật toán RSA dựa trên bài toán **Phân tích thừa số nguyên tố**: Rất dễ để lấy 2 số nguyên tố siêu lớn nhân với nhau ra một tích khổng lồ, nhưng các siêu máy tính cũng cần hàng triệu năm để từ tích khổng lồ đó mò ngược ra 2 số nguyên tố gốc.
   - **Cơ chế hoạt động (Dòng chảy Kỹ thuật Thực tế):** Toán học sinh ra một cặp khóa có liên hệ mật thiết: Khóa công khai (**Public Key**) và Khóa bí mật (**Private Key**). 
     - **Bước 1 (Server phân phối khóa):** Server tạo ra cặp khóa, lưu trữ Private Key vào phân vùng tuyệt mật của hệ điều hành. Sau đó, Server phát Public Key công khai ra Internet cho bất kỳ Client (User) nào muốn kết nối.
     - **Bước 2 (Client mã hóa):** Trình duyệt của User sử dụng Public Key vừa nhận được để mã hóa dữ liệu nhạy cảm (ví dụ: mã hóa cái Khóa đối xứng). Theo quy luật hàm một chiều, dữ liệu sau khi mã hóa bởi Public Key sẽ biến thành một khối byte hỗn độn mà **chỉ có Private Key tương ứng mới có khả năng tính toán dịch ngược lại được**.
     - **Bước 3 (Server giải mã):** Server nhận được khối byte mã hóa, sử dụng Private Key của mình làm tham số bí mật để giải phương trình toán học ngược, từ đó thu lại được dữ liệu gốc của User.
     - **Góc nhìn của Hacker:** Hacker đứng lén giữa mạng lưới bắt được trọn vẹn cả "Public Key" (ở Bước 1) và "Dữ liệu đã mã hóa" (ở Bước 2). Tuy nhiên, vì không có Private Key, Hacker phải sử dụng siêu máy tính để cố gắng Brute-force (mò) ngược từ Public Key ra Private Key. Dựa trên rào cản của bài toán Phân tích thừa số nguyên tố (phải tìm 2 số nguyên tố cấu thành một tích dài hàng nghìn bit), mạng lưới siêu máy tính của Hacker sẽ phải chạy liên tục trong **vài triệu năm** mới giải ra được. Điều này khiến nỗ lực bẻ khóa trở nên hoàn toàn vô vọng!

   - **Đặc điểm:** Giải quyết triệt để bài toán phân phối khóa trên môi trường Public không an toàn. Nhược điểm chí mạng là phép toán lũy thừa số lớn (Modular Exponentiation) cực kỳ cồng kềnh, **Tốc độ cực kỳ chậm**, ngốn rất nhiều tài nguyên CPU.
   
2. **Mã hóa Đối xứng (Symmetric Cryptography - Tiêu biểu: AES, ChaCha20):**
   - **Bản chất Toán học:** Hoạt động dựa trên các phép toán biến đổi ma trận (Substitution-Permutation Network) và các toán hạng Bitwise tuyến tính (như phép XOR). Dữ liệu gốc (Plaintext) được băm thành các khối (Block Cipher) hoặc dòng (Stream Cipher), sau đó trải qua nhiều vòng (rounds) trộn lẫn, dịch bit và hoán vị với Khóa.
   - **Cơ chế hoạt động:** Chỉ tồn tại **một Khóa duy nhất (Symmetric Key)** cho cả hai quá trình Mã hóa và Giải mã. Nếu Plaintext ⊕ Key = Ciphertext, thì Ciphertext ⊕ Key = Plaintext (Bản chất của phép XOR).
   - **Đặc điểm:** Các phép toán dịch bit và XOR được các dòng CPU hiện đại hỗ trợ thẳng ở cấp độ phần cứng (như tập lệnh AES-NI). Do đó, **Tốc độ xử lý cực kỳ nhanh**, đáp ứng được băng thông khổng lồ của Streaming Video hay truyền File lớn. Nhược điểm chí mạng là: Làm sao để Client gửi được chiếc Khóa duy nhất này sang cho Server an toàn?

-> **Giải pháp thiên tài của TLS (Hybrid Encryption - Mã hóa Lai):**
Thay vì đắn đo chọn 1 trong 2, TLS đã kết hợp cả hai để tạo ra một kịch bản hoàn hảo:
- **Bước 1 (Dùng Bất đối xứng để bọc khóa):** Trong vài mili-giây đầu tiên, máy tính của bạn (Client) tự ngẫu nhiên sinh ra một cái "Khóa đối xứng" (Session Key). Sau đó, Client lấy Public Key của Server để mã hóa chính cái Session Key này rồi gửi qua cáp quang biển. 
- **Bước 2 (Giải mã an toàn):** Nhờ cơ chế hàm một chiều RSA, Hacker đứng giữa mạng bắt được gói tin này cũng phải khóc ròng. Chỉ duy nhất Server cầm Private Key mới giải mã ra và thu thập được cái Session Key nguyên vẹn. Vậy là bài toán "Phân phối Khóa đối xứng an toàn qua Internet" đã được giải quyết triệt để!
- **Bước 3 (Chuyển sang Đối xứng tốc độ cao):** Lúc này, cả Client và Server đã cùng nắm giữ một Session Key bí mật chung. Chúng lập tức vứt bỏ bộ khóa Bất đối xứng nặng nề. Kể từ giây phút này, toàn bộ hình ảnh, video, dữ liệu Web sẽ được mã hóa bằng Session Key (AES) với tốc độ ánh sáng!

> 💡 **Giải phẫu Toán học: Sự thật đằng sau Public Key và Dữ liệu là gì?**
> Sự thật: Trong máy tính không hề có khái niệm "Ổ khóa" hay "Chìa khóa" vật lý, tất cả chỉ là những **con số nguyên khổng lồ**.
> 
> **1. Dữ liệu (Payload):** 
> Giả sử bạn muốn gửi con số **M = 4** (Đại diện cho một mẩu dữ liệu nhỏ).
> 
> **2. Bức tranh Toàn cảnh về Định lý Euler (The Big Picture):**
> Trước khi đi vào tính toán, hãy cùng hiểu gốc rễ: **Tại sao thuật toán RSA lại chọn định lý Euler làm nền tảng?**
> 
> - **Mục đích:** 
>   Mục tiêu tối thượng của mã hóa Bất đối xứng là tạo ra một "Hàm một chiều" (Trapdoor function) - một hàm toán học mà: Đi xuôi (mã hóa) thì cực kỳ dễ, nhưng Đi ngược (giải mã) thì vô phương cứu chữa... trừ khi bạn có một "cửa sập bí mật" (Private Key). Định lý Euler chính là công cụ toán học hoàn hảo để xây dựng "cửa sập" này.
> 
> - **Nguyên lý cốt lõi (Sự bất đối xứng của bài toán phân tích thừa số nguyên tố):**
>   - Rất DỄ để lấy 2 số nguyên tố khổng lồ $p$ và $q$ nhân lại với nhau để ra một tích số $n$ (Máy tính chỉ mất vài mili-giây).
>   - Nhưng cực kỳ KHÓ để làm ngược lại: Đưa cho siêu máy tính tích số $n$, yêu cầu nó phân tích ngược ra $p$ và $q$. 
>   *Góc nhìn Toán học:* Với một hợp số bình thường (như 24), nó có vô số cách phân tích ($2 \times 12, 3 \times 8, 4 \times 6$). Máy tính có thể tìm ra quy luật rất nhanh. Tuy nhiên, tích $n$ của 2 số nguyên tố là một "vùng đất chết" (Semi-prime). Nó CHỈ chia hết cho đúng $p$ và $q$. Do các số nguyên tố không tuân theo quy luật dự đoán nào, siêu máy tính buộc phải chia thử (brute-force) từng số lẻ một. Khi $n$ đủ lớn (vd: 600 chữ số), tập hợp các phép thử này lớn hơn cả số nguyên tử trong vũ trụ, ngốn hàng triệu năm để giải.
> 
> - **Cầu nối Toán học - Hàm Phi Euler $\phi(n)$:**
>   Hàm $\phi(n)$ đếm "tổng số lượng các con số nhỏ hơn $n$ và nguyên tố cùng nhau với $n$". 
>   *(Ghi chú cực kỳ quan trọng: Đừng nhầm lẫn giữa "Số nguyên tố" và "Hai số nguyên tố cùng nhau". Hai số là "nguyên tố cùng nhau" (Coprime) nếu chúng KHÔNG có ước số chung nào ngoài số 1. Bản thân từng số đó KHÔNG bắt buộc phải là số nguyên tố! Ví dụ: Xét $\phi(10)$, các số 1, 3, 7, 9 không có ước chung nào với 10 nên được nhận. Nhận thấy số 9 vốn không phải là số nguyên tố (vì chia hết cho 3), nhưng 9 và 10 lại là "nguyên tố cùng nhau" vì chúng không có chung bảng cửu chương. Tổng cộng ta đếm được 4 số $\rightarrow \phi(10) = 4$).*
>   
>   **Cách tính $\phi(n)$ cực kỳ thú vị:**
>   - Theo định nghĩa, nếu $p$ là số nguyên tố, mọi số nhỏ hơn $p$ đều nguyên tố cùng nhau với nó $\rightarrow \phi(p) = p - 1$.
>   - Vì hàm Phi có tính chất nhân (multiplicative) $\rightarrow \phi(n) = \phi(p \times q) = \phi(p) \times \phi(q) = (p - 1) \times (q - 1)$.
>   - Nhờ vậy, nếu bạn là **Server** (biết trước $p$ và $q$), bạn tính ra $\phi(n)$ ngay lập tức.
>   - Nếu bạn là **Hacker** (chỉ biết số $n$ trôi nổi trên mạng), bạn vĩnh viễn không thể tính ra $\phi(n)$ trừ khi bạn đập vỡ được $n$ ra thành $p$ và $q$.
> 
> - **Định lý Euler - Trái tim của RSA:**
>   Toán học gia Euler chứng minh rằng: Mọi con số $M$ (nếu nguyên tố cùng nhau với $n$) khi đem lũy thừa cho $\phi(n)$ rồi chia lấy dư cho $n$, phần dư LUÔN LUÔN BẰNG 1. 
>   Công thức: **$M^{\phi(n)}$ % $n = 1$**.
>   *Ví dụ:* $\phi(10) = 4$. Chọn $M = 3$. Ta có $3^4 = 81$. Đem $81$ chia $10$ dư đúng $1$.
> 
>   **Tại sao lại sinh ra công thức $e \times d = k \times \phi(n) + 1$?**
>   Ba nhà khoa học RSA cần tìm một cách để "triệt tiêu số mũ", tức là họ cần giải bài toán: **$M^{e \times d}$ % $n = M$** (Mã hóa xong giải mã phải ra đúng chữ gốc). 
>   Họ nhìn vào chân lý của Euler và suy luận ngược:
>   - Nếu $M^{\phi(n)}$ % $n$ luôn sinh ra $1$, thì đem lũy thừa lên $k$ lần cũng vẫn bằng $1$: $(M^{\phi(n)})^k = 1^k = 1 \rightarrow M^{k \times \phi(n)}$ % $n = 1$.
>   - Nhân thêm một số $M$ vào cả hai vế: $M^{k \times \phi(n)} \times M$ % $n = 1 \times M \rightarrow$ **$M^{k \times \phi(n) + 1}$ % $n = M$**.
>   
>   Đối chiếu bài toán RSA cần giải (**$M^{e \times d}$ % $n = M$**) với phương trình vừa suy ra (**$M^{k \times \phi(n) + 1}$ % $n = M$**), ta thấy hai số mũ bắt buộc phải bằng nhau!
>   Đó là lý do RSA quy định điều kiện bắt buộc để chọn cặp khóa $(e, d)$ là: **$e \times d = k \times \phi(n) + 1$**.
>
>   *(💡 **Giải phẫu biến k**: Tại sao phải nhét thêm $k$ vào công thức? Nếu rút gọn phương trình trên ta có $d = \frac{\phi(n) + 1}{e}$ (bỏ qua $k$). Về mặt Toán lý thuyết thì đúng, nhưng đưa vào máy tính sẽ **LỖI NGAY LẬP TỨC** nếu phép chia ra số thập phân (vì Mật mã học chỉ chạy bằng số nguyên). Bằng cách nhân $\phi(n)$ lên $k$ lần, biến đệm $k$ linh hoạt này ($k = 1, 2, 3...$) đóng vai trò làm công cụ "trượt", đảm bảo luôn có thể đẩy cái tử số lên đủ lớn để chia chẵn cho $e$. Nó cam kết 100% hệ thống luôn sinh ra được một Private Key $d$ nguyên vẹn, bất chấp bạn chọn Public Key $e$ xấu cỡ nào!)*
>
>   Nhờ điều kiện này, hệ quả sinh ra một phép màu Toán học:
>   $M^{(e \times d)} = M^{k \times \phi(n) + 1} = (M^{\phi(n)})^k \times M$
>   Mà cụm $M^{\phi(n)}$ luôn chia dư ra $1$. Nên phương trình rút gọn thành: $1^k \times M = M$.
>   **Kết luận:** **$M^{(e \times d)}$ % $n = M$**.
>   Nghĩa là: Lấy dữ liệu $M$, đem mũ $e$ (Mã hóa), rồi lại đem mũ $d$ (Giải mã), số mũ tự động triệt tiêu và trả về đúng dữ liệu $M$ ban đầu!
>
>   **Chạy thử Ví dụ (Thay số vào công thức tự triệt tiêu):**
>   Giả sử ta dùng bộ số: $n = 33$, $\phi(n) = 20$. Cặp khóa $e = 3$, $d = 7$. Dữ liệu cần mã hóa $M = 4$.
>   Kiểm tra điều kiện số mũ: $e \times d = 3 \times 7 = 21$. Phân tích theo công thức $k \times \phi(n) + 1$, ta có: $21 = 1 \times 20 + 1$ (tương ứng với $k = 1$).
>   
>   Bây giờ ta đem thế số vào chuỗi công thức hệ quả Toán học:
>   - Bắt đầu với bài toán gốc (Mã hóa rồi giải mã): $M^{(e \times d)}$ % $n$
>   - Thay số $M, e, d$: $\rightarrow 4^{21}$ % $33$
>   - Tách số $21$ thành $(1 \times 20 + 1)$: $\rightarrow 4^{(1 \times 20 + 1)}$ % $33$
>   - Theo luật lũy thừa, bung phép cộng thành phép nhân: $\rightarrow (4^{20})^1 \times 4^1$ % $33$
>   - *Lúc này Định lý Euler lên tiếng:* Theo Euler, $4^{\phi(33)}$ chính là $4^{20}$. Định lý khẳng định cụm này đem `% 33` thì LUÔN LUÔN BẰNG **$1$**.
>   - Gạch bỏ $4^{20}$ và thay bằng số $1$: $\rightarrow (1)^1 \times 4$ % $33$
>   - Phương trình triệt tiêu sạch sẽ: $\rightarrow 1 \times 4$ % $33$ = **$4$**.
>   Thật kỳ diệu! Cái lũy thừa $20$ khổng lồ đã bốc hơi hoàn toàn nhờ phép màu của định lý Euler, trả lại đúng con số $M = 4$ nguyên vẹn.
> 
> **3. Ứng dụng thực tế: Toàn cảnh 9 Bước hoạt động của RSA:** 
> 
> **[GIAI ĐOẠN 1: SERVER TẠO KHÓA]**
> - **Step 1:** Server random sinh ra 2 số nguyên tố cực lớn ($p$ và $q$). *(Ví dụ thu nhỏ: $p=11$, $q=3$)*.
> - **Step 2 (Tạo Modulus):** Server nhân lại để tạo ra tích số nguyên tố $n = p \times q$. *(Ví dụ thu nhỏ: $n = 33$)*.
> - **Step 3 (Hàm Euler):** Server tính nhanh hằng số đếm $\phi(n) = (p-1) \times (q-1)$. *(Ví dụ: $\phi(n) = 10 \times 2 = 20$)*.
> - **Step 4 (Public Key):** Server chọn một số $e$ sao cho $e < \phi(n)$ và là nguyên tố cùng nhau với $\phi(n)$. *(Ví dụ: Chọn $e = 3$)*.
> - **Step 5 (Private Key):** Server tìm số $d$ bằng công thức: $d = (k \times \phi(n) + 1) / e$ (chọn $k$ sao cho phép chia ra số nguyên). *(Ví dụ: Chọn $k=1 \rightarrow d = (1 \times 20 + 1) / 3 = 7$)*.
> - **Step 6 (Phân phối):** Server **XÓA VĨNH VIỄN** $p$, $q$ và $\phi(n)$ khỏi bộ nhớ (để Hacker không thể tìm được). Nó giấu kín **$d=7$** (Private Key) vào két sắt, và gộp cặp số **$(e=3, n=33)$** thành Public Key để phát ném qua mạng cho User.
>
> **[GIAI ĐOẠN 2: MẠNG LƯỚI HOẠT ĐỘNG (CLIENT & SERVER)]**
> - **Step 7 (User Mã hóa):** Trình duyệt của User lấy data (ví dụ mã ASCII $M = 4$). Máy tính User áp dụng công thức mã hóa: Lấy dữ liệu lũy thừa $e$ rồi chia dư cho $n$.
>   $\rightarrow$ Công thức: $C = M^e$ % $n$.
>   *(Ví dụ: $C = 4^3$ % $33 = 64$ % $33 = 31$. Khối data được mã hóa $C=31$ này được truyền qua đại dương).*
> - **Step 8 (Server Giải mã):** Server nhận được khối được mã hóa $C=31$. Server không cần phân tích đa thức gì cả, CPU của Server cứ vô tri lôi $d$ ra và chạy công thức giải mã: 
>   $\rightarrow$ Công thức: $M' = C^d$ % $n$.
>   *(Ví dụ: CPU lấy $31^7$ % $33 = 27,512,614,111$ % $33$. Phép chia dư ra đúng số... **4**).*
> - **Step 9 (Phép màu Toán học):** Nhờ có định lý Euler "bảo kê" từ trước (đã chứng minh trên giấy ở mục 2), phép tính vô tri của CPU ở Step 8 tự động ra kết quả $M'$ trùng khớp hoàn hảo 100% với $M=4$ ban đầu. Dữ liệu được phục hồi mà chiếc chìa khóa $d$ chưa từng phải rời khỏi Server.
> 
> **[GIAI ĐOẠN 3: ĐẢO NGƯỢC RSA - CHỮ KÝ SỐ (DIGITAL SIGNATURE)]**
> Sự vĩ đại của RSA nằm ở chỗ: $e$ và $d$ có tính chất đối xứng hoàn hảo. Khóa bằng chìa nào thì Mở bằng chìa còn lại!
> Do đó, thay vì dùng Public Key để mã hóa như Giai đoạn 2, ta có thể đảo ngược lại để làm công cụ **Chứng minh Danh tính (Xác thực SSH / JWT / TLS)**:
> - **Step 10 (Server thách thức):** Server muốn kiểm tra xem BẠN có đúng là chủ nhân của Private Key không. Server ném cho BẠN một chuỗi số ngẫu nhiên (Ví dụ $M = 4$). 
> - **Step 11 (BẠN Ký tên):** Bạn dùng **Private Key ($d=7$)** của bạn để MÃ HÓA chuỗi đó (Hành động này gọi là Tạo chữ ký). 
>   $\rightarrow$ Công thức y hệt: $S = M^d$ % $n$.
>   *(Ví dụ: $S = 4^7$ % $33 = 16384$ % $33 = 16$. Bạn ném Chữ ký số $S=16$ này lên Server).*
> - **Step 12 (Server Xác minh):** Server nhận được $S=16$. Nó liền lôi **Public Key ($e=3$)** của bạn ra để GIẢI MÃ chữ ký đó.
>   $\rightarrow$ Công thức: $M' = S^e$ % $n$.
>   *(Ví dụ: $M' = 16^3$ % $33 = 4096$ % $33 = 4$. Kết quả giải mã ra đúng số 4 ban đầu!)*
>   $\rightarrow$ Server chốt hạ: *"Giải mã bằng Public Key ra đúng dữ liệu gốc. Vậy tao cá chắc 100% cái chữ ký này phải được tạo ra từ Private Key thực sự!"*.
> - **Chống Replay Attack:** Hacker không thể lấy trộm gói tin $S=16$ gửi lại cho Server vào ngày mai, vì ngày mai Server sẽ ném một Challenge mới tinh (Ví dụ $M = 5$). Chữ ký cũ $S=16$ đem giải mã sẽ bung bét và không bao giờ khớp với 5!
> 
> *Góc nhìn Hacker (Hệ phương trình Vô vọng):* 
> Để tìm được khóa bí mật `d`, Hacker bắt buộc phải tính được giá trị $\phi(n)$. 
> Hãy thử tư duy ngược: Nếu Hacker BẰNG MỘT PHÉP MÀU NÀO ĐÓ biết trước được con số đếm $\phi(n) = 20$ (cùng với $n = 33$ công khai trên mạng). Hắn sẽ lập tức giải ra $p$ và $q$ thông qua hệ phương trình Viète lớp 9:
> 1. Tích hai số: $p \times q = 33$
> 2. Dựa vào $\phi(n)$: $(p-1) \times (q-1) = 20 \implies (p \times q) - p - q + 1 = 20 \implies 33 - (p+q) + 1 = 20 \implies p + q = 14$
> -> Hacker có Tổng = 14, Tích = 33. Lập tức nhẩm ra ngay 2 nghiệm là $11$ và $3$. Hacker lấy được $p=11, q=3$ và bẻ khóa toàn bộ hệ thống!
> 
> **NHƯNG... LÀM SAO ĐỂ BIẾT ĐƯỢC $\phi(n)$?** 
> Hacker trên mạng chỉ bắt được duy nhất con số $n$. Hắn KHÔNG THỂ lập hệ phương trình vì thiếu mảnh ghép $\phi(n)$. Để đếm được $\phi(n)$ của một số $n$ dài 600 chữ số, siêu máy tính của Hacker buộc phải đếm thủ công nghiệm Coprime nghiệm từ 1 lên tới $n$. Quá trình đếm mò mẫm này ngốn thời gian... lớn hơn hàng tỷ lần tuổi thọ của Vũ trụ. Đó chính là sự bất lực tuyệt đối của Hacker trước bức tường Toán học!

### 5.3. Kịch bản Truyền tải Thực tế (TCP & TLS Handshake)
Để đạt được sự tối ưu giữa "Tốc độ của mã hóa đối xứng" và "Tính bảo mật của mã hóa bất đối xứng", máy tính (Client) và máy chủ (Server) bắt buộc phải thực hiện một chuỗi giao thức bắt tay nghiêm ngặt theo đúng trình tự thời gian. Hãy nhớ rằng: **Bắt tay TLS (Layer 7) không thể diễn ra nếu chưa có ống nước kết nối (TCP Socket - Layer 4)**. 

Dưới đây là Dòng chảy Kỹ thuật thuần túy (Packet Flow) cho một truy cập HTTPS (Port 443):

**GIAI ĐOẠN 1: MỞ KẾT NỐI (TCP 3-Way Handshake - Lớp 4)**
Trước khi bàn chuyện bảo mật, Client phải thiết lập một Socket tới Port 443 của Server.
1. `[Client -> Server]:` **SYN** *(Đề nghị mở kết nối TCP).*
2. `[Server -> Client]:` **SYN-ACK** *(Đồng ý mở kết nối, anh xác nhận đi).*
3. `[Client -> Server]:` **ACK** *(Xác nhận. Ống TCP đã nối thành công).*
-> *Kết quả: Kênh truyền tải đã sẵn sàng, nhưng CHƯA có byte dữ liệu Web nào được gửi.*

**GIAI ĐOẠN 2: THỎA THUẬN BẢO MẬT (TLS Handshake - Lớp 7)**
Sử dụng ống TCP vừa mở, hai bên bắt đầu một cuộc đàm phán cực kỳ phức tạp để thiết lập mã hóa. (Lưu ý: Trước khi quá trình này diễn ra, Server đã phải tự sinh ra một cặp khóa Bất đối xứng **RSA Keypair (Public Key & Private Key)** và nộp Public Key cho một tổ chức ủy quyền để xin cấp Chứng thư số).

4. **ClientHello** `[Client -> Server]:` Client khởi tạo luồng kết nối bảo mật, gửi bản tin bao gồm: Phiên bản giao thức (TLS 1.2/1.3), danh sách các bộ thuật toán mã hóa (Cipher Suites) được hỗ trợ, và một chuỗi ngẫu nhiên gọi là `Client Random`.
5. **ServerHello** `[Server -> Client]:` Server phản hồi, chọn một bộ Cipher Suite chung, kèm theo chuỗi ngẫu nhiên `Server Random` của riêng mình.
6. **Certificate Exchange (Truyền tải Chứng thư số)** `[Server -> Client]:` Server gửi cho Client một **Chứng thư số (Digital Certificate chuẩn X.509)**. 
   - *Chứng thư số là cái gì?* Nó đóng vai trò như một "Căn cước công dân Điện tử". Bên trong nó chứa: Tên miền của Server (vd: `facebook.com`), thông tin tổ chức sở hữu, và quan trọng nhất là **Public Key** của Server. Toàn bộ thông tin này được niêm phong bằng Chữ ký số (Digital Signature) của một Tổ chức xác thực uy tín toàn cầu (gọi là **CA - Certificate Authority**, ví dụ: DigiCert, Let's Encrypt).
7. **CA Validation (Xác thực Căn cước):** 
   - Trình duyệt Client không hề tin tưởng Server một cách mù quáng. Nó lôi danh sách các khóa gốc của các Tổ chức CA (Root CA) đã được cài cắm sẵn (hardcode) trong Hệ điều hành Windows/macOS ra để đối chiếu chữ ký số trên Chứng thư.
   - Nếu chữ ký hợp lệ và tên miền khớp, Client chính thức xác nhận: *"À, cái Public Key này đích thực là của facebook.com, không phải của Hacker lừa đảo chặn giữa mạng"*. (Đây là cơ chế chống tấn công giả mạo **Man-in-the-Middle**).
8. **Key Exchange (Trao đổi Khóa bí mật - Sự ra đời của Client Secret):** 
   - *Tại sao lại đẻ ra cái này?* Vì mã hóa Bất đối xứng (RSA) tính toán quá cồng kềnh, tốc độ như rùa bò, không thể dùng để mã hóa cả bộ phim Netflix hay luồng Livestream. Ta bắt buộc phải chuyển sang xài Mã hóa Đối xứng (AES) có tốc độ ánh sáng. Nhưng Mã hóa Đối xứng lại yêu cầu cả hai bên phải có chung 1 cái "Chìa khóa Đối xứng" (Session Key). Làm sao gửi cái chìa khóa này qua mạng Internet đầy rẫy Hacker?
   - *Cách giải quyết:* Trình duyệt tự random sinh ra một chuỗi bit cực kỳ bảo mật gọi là **Pre-Master Secret (hay Client Secret)**. Sau đó, nó bọc cái `Client Secret` này lại, khóa chặt bằng cái **Public Key** của Server (vừa nhận ở bước 6) rồi ném qua mạng. 
   - *Kết quả:* Nhờ tính chất Hàm một chiều của RSA, gói tin chứa `Client Secret` bay qua Internet an toàn tuyệt đối. Hacker dù có chộp được cũng chỉ nhìn thấy một đống rác nhiễu loạn.
9. **Session Key Generation (Sinh Khóa Phiên Đối xứng):** 
   - Server nhận gói tin, lập tức móc **Private Key** (đang giấu sâu trong két sắt ổ cứng) ra để giải mã, và bóc tách thành công `Client Secret` nguyên vẹn. 
   - Lúc này, phép màu hoàn tất: Cả Client và Server đều độc lập nắm trong tay 3 nguyên liệu: (`Client Random` + `Server Random` + `Client Secret`). Cả hai nhét 3 nguyên liệu này vào một hàm băm (Hash Function) để nhào nặn ra chung một cái chìa khóa duy nhất: **Master Secret (Khóa Phiên/Session Key)**.
   - *(Giải ngố kỹ thuật: Tại sao lại rườm rà trộn thêm `Client Random` và `Server Random`? Mục đích là để chống **Tấn công phát lại (Replay Attack)**. Nếu Hacker nghe lén hôm nay, copy y chang gói tin mã hóa rồi ngày mai phát lại gửi cho Server, thì do chuỗi `Server Random` của ngày mai đã bị đổi, cái Session Key sinh ra sẽ hoàn toàn khác. Gói tin cũ hôm qua vĩnh viễn không thể xài lại ở phiên hôm nay!)*
10. **Secure Payload (Thiết lập kênh truyền):** Cả hai bên gửi bản tin `ChangeCipherSpec` và `Finished`, cam kết từ giây phút này sẽ vứt bỏ bộ khóa Bất đối xứng nặng nề (RSA) vào sọt rác. Toàn bộ hình ảnh, tin nhắn, dữ liệu từ đây về sau sẽ được băm nát và mã hóa bằng cái **Session Key** (thuật toán AES) với tốc độ ánh sáng.

**GIAI ĐOẠN 3: BƠM DỮ LIỆU THỰC TẾ (HTTPS Traffic)**

11. `[Client -> Server]:` **HTTP GET /** *(Toàn bộ Payload HTTP lúc này đã bị băm nát và mã hóa bằng Khóa phiên Đối xứng, đóng vào TCP Segment ném qua mạng).*
12. `[Server -> Client]:` **HTTP 200 OK** *(Server giải mã bằng Khóa phiên, xử lý logic Web, và trả về HTML/JSON cũng được mã hóa bằng Khóa phiên).*
-> *Kết quả: Khóa phiên đối xứng chỉ tồn tại tạm thời trong RAM và sẽ bị HĐH xóa sổ vĩnh viễn khi tắt tab trình duyệt (Cơ chế Perfect Forward Secrecy).*

### 5.4. Quy trình Server "Xin" Chứng thư số (CSR - Certificate Signing Request)
Trước khi Giai đoạn 2 (TLS Handshake) ở trên có thể diễn ra, Server (ví dụ: `facebook.com`) phải có Chứng thư số. Quy trình lấy Chứng thư diễn ra như sau:
1. **Tạo Keypair cục bộ:** Máy chủ `facebook.com` tự chạy thuật toán RSA để sinh ra một cặp (Public Key & Private Key). Private Key được cất kỹ vào thư mục `/etc/ssl/private` (không ai được đụng vào).
2. **Làm đơn xin cấp phép (CSR):** Máy chủ gom Public Key vừa sinh ra, kẹp chung với thông tin tên miền `facebook.com`, rồi gói lại thành một tờ đơn gọi là CSR (Certificate Signing Request).
3. **Nộp đơn cho Tổ chức CA:** Máy chủ nộp tờ đơn CSR này cho các tổ chức CA (Certificate Authority - Tổ chức cấp phát chứng chỉ uy tín toàn cầu như Let's Encrypt, DigiCert). CA sẽ bắt Server làm vài bài test (ví dụ: yêu cầu tạo một bản ghi TXT trong cấu hình DNS để chứng minh anh thực sự là chủ sở hữu của domain `facebook.com`).
4. **Đóng dấu (Digital Signature):** Sau khi xác minh thành công, tổ chức CA lấy **Private Key của chính CA (Root Private Key)** đập một "Chữ ký số" lên tờ đơn CSR. Tờ đơn lúc này lột xác trở thành "Chứng thư số X.509" chính thức. Server mang Chứng thư này về cắm vào Nginx/Apache để xài.

### 5.5. Giải ngố: Hệ điều hành cập nhật danh sách CA như thế nào?
Có một sự hiểu lầm cực kỳ tai hại: *"Nếu Trình duyệt lôi danh sách trong Hệ điều hành ra để đối chiếu, vậy không lẽ mỗi khi có một trang web mới ra đời (hoặc xóa đi) thì Microsoft/Apple phải tung ra bản cập nhật OS?"*.

**Câu trả lời là: KHÔNG HỀ!**
- **Sự thật về Trust Store:** Hệ điều hành **KHÔNG LƯU** danh sách hàng tỷ tên miền trên thế giới. Nó CHỈ LƯU duy nhất một danh sách nhỏ gọi là **Trust Store (Kho lưu trữ niềm tin)**. Trong Trust Store này chứa khoảng 100 - 200 cái **Public Key của các tổ chức Root CA** (như Let's Encrypt, GlobalSign, DigiCert,...).
- **Cơ chế Chuỗi niềm tin (Chain of Trust):** Máy tính của bạn không hề quen biết cái tên miền lạ hoắc `facebook.com`. Nhưng khi Facebook chìa cái Chứng thư ra, máy tính thấy trên đó có "Con dấu đỏ" (Chữ ký số) của ông lớn DigiCert. Vì Public Key của DigiCert đã nằm sẵn (hardcode) trong Hệ điều hành từ lúc bạn cài Win rồi, nên máy tính dùng khóa gốc của DigiCert soi vào con dấu. Thấy con dấu chuẩn 100%, máy tính suy luận: *"Mình không biết thằng Facebook là ai, nhưng ông DigiCert đã đóng mộc bảo kê cho cái Public Key của nó, mà mình thì tin ông DigiCert tuyệt đối, nên mình sẽ tin cái Public Key này đích thực là của thằng Facebook"*.
- **Khi nào thì thực sự phải update OS?** Bạn đã đoán đúng, ta CHỈ CẦN update OS khi có **một tổ chức CA mới ra đời** (hoặc một CA cũ bị tước giấy phép do làm lộ Private Key). Bạn có nhớ hiện tượng máy tính xài Windows XP hoặc Windows 7 dạo gần đây không vào được các trang web HTTPS hiện đại không? Lý do chính là vì Win 7 quá cũ, Microsoft ngừng hỗ trợ nên cái Trust Store của nó không được bơm thêm các Public Key của các tổ chức CA mới (như Let's Encrypt). Trình duyệt nhìn thấy con dấu Let's Encrypt nhưng tìm mỏi mắt không thấy khóa Let's Encrypt trong máy để xác minh, thế là nó văng màn hình đỏ *"Kết nối không bảo mật"* dù bản thân trang web không hề có lỗi!

### 5.6. Góc đính chính: Hacker chôm Public Key và Lỗi CORS
Có hai thắc mắc kinh điển mà các Lập trình viên hay nhầm lẫn khi cấu hình Web Server (như Nginx kết nối FE với BE):

**Câu hỏi 1: Tại sao Server ném Public Key công khai qua mạng mà không sợ Hacker bắt được và giả mạo (Man-in-the-Middle Attack)?**
- *Trả lời:* Hacker CÓ THỂ bắt được Public Key trên đường bay, nhưng **Vô dụng**. Hacker không thể dùng Public Key đó để bẻ khóa (vì luật Toán học quy định chỉ có Private Key giấu ở Server mới mở được).
- *Vậy nếu Hacker tráo Public Key của Hacker vào thì sao?* Hacker chặn Chứng chỉ thật, tự tạo một Chứng chỉ giả chứa Public Key của Hacker rồi gửi cho Client. Client tưởng thật, dùng khóa của Hacker bọc Session Key, Hacker sẽ mở được! ĐỂ CHỐNG LẠI TRÒ NÀY, ta có **Chữ ký số (Digital Signature) của Tổ chức CA**. Chứng chỉ gốc đã được ký niêm phong (hashed) bằng khóa bí mật của Tổ chức Root CA (như Let's Encrypt). Bất kỳ ai sửa đổi dù chỉ 1 ký tự trong Chứng chỉ (tráo Public Key), chữ ký đó sẽ bị vỡ. Trình duyệt Chrome check thấy chữ ký vỡ lập tức văng màn hình đỏ chót *"Your connection is not private"* và chặt đứt kết nối.

**Câu hỏi 2: Có phải chứng chỉ SSL/CA bị sai là nguyên nhân gây ra lỗi CORS khi FE gọi API sang BE không?**
- *Trả lời:* Về mặt lý thuyết mạng lưới, **SSL/TLS (Mã hóa đường truyền) KHÔNG HỀ LIÊN QUAN GÌ ĐẾN CORS (Chính sách Trình duyệt)**.
  - **SSL/TLS:** Đảm bảo dữ liệu không bị nghe lén.
  - **CORS (Cross-Origin Resource Sharing):** Là chính sách bảo vệ do Trình duyệt (Chrome/Safari) áp đặt. Nó ngăn cấm web `hacker.com` lén dùng Javascript gọi API sang `api.bank.com` để ăn cắp tiền. Lỗi này là do Backend của Bank quên chưa cấu hình trả về HTTP Header `Access-Control-Allow-Origin`. Nó xảy ra kể cả khi cả 2 bên xài HTTP không mã hóa.
- *Tại sao Dev cực kỳ hay nhầm lẫn 2 cái này làm một?* Khi Dev làm việc ở Local (FE chạy `http://localhost:3000`, BE chạy `https://api.dev.com`). Nếu BE xài chứng chỉ SSL **tự ký (Self-signed - không được Root CA bảo chứng)**, Trình duyệt Chrome sẽ ngầm đánh giá kết nối SSL này là nguy hiểm và chặn toàn bộ các lệnh `fetch/axios` gửi đi. Thay vì báo lỗi "SSL Invalid", cái cửa sổ Console của Chrome đôi lúc lại văng ra dòng chữ ngu ngốc: `CORS error` hoặc `Network Error`. Thế là Dev cuống cuồng đi sửa file config CORS trong Backend suốt 3 ngày mà không fix được, trong khi thực tế chỉ cần click chuột vào nút "Bỏ qua cảnh báo an toàn SSL" hoặc cài chứng chỉ thật là xong!

---

## 6. Tường lửa (Firewall):

Một khi mạng đã thông, dữ liệu đã được mã hóa TLS, bài toán sống còn tiếp theo là: *"Làm sao để cấm một gã ất ơ nào đó ở Nga không được quyền chọc vào Database của tôi?"*. Đó là lúc Firewall ra đời. Firewall giống như ông bảo vệ tòa nhà, đứng xét hỏi từng gói tin (Packet) đi qua.

### 6.1. Tường lửa Lớp 3/4 (Network Firewall)
Đây là loại tường lửa cổ điển và phổ biến nhất, hoạt động ở tầng **Network (IP)** và **Transport (Port)**. Trên Cloud AWS, nó chính là **Security Group** hoặc **Network ACL**.

- **Cách hoạt động:** Ông bảo vệ Lớp 4 khá "nguyên tắc nhưng mù quáng". Ông ta chỉ nhìn vào cái phong bì (Header), kiểm tra `IP Nguồn`, `IP Đích` và `Port`. Ông không thèm bóc phong bì ra xem bên trong (Payload) chứa mã độc hay không.
- **Ví dụ thực tiễn:** Cấu hình Security Group cho EC2 chạy Web Server:
  - Cho phép `Inbound Port 443` từ `0.0.0.0/0` (Mở cửa cho toàn thế giới truy cập Web HTTPS).
  - Cấm `Inbound Port 22` từ mọi nơi, Cụ thể chỉ cho phép IP tĩnh của Văn phòng Cty `14.55.x.x` được phép truy cập để SSH vào bảo trì hệ thống.
- **Stateful vs Stateless:** 
  - **Stateful (Lưu trữ trạng thái):** Điển hình là AWS Security Group. Nếu bạn cho phép một IP đi **VÀO** cổng 443, Firewall sẽ lưu vào trí nhớ tạm thời và tự động mở cửa cho phép luồng dữ liệu trả lời đi **RA** mà không cần bạn phải cấu hình thêm luật chiều ra.
  - **Stateless (Không lưu trữ trạng thái):** Điển hình là AWS Network ACL. Trí nhớ bằng 0. Cấm hay cho phép chiều VÀO thì DevOps cũng phải tự tay viết thêm một luật tương ứng cho phép chiều RA, nếu không gói tin đi vào được nhưng lúc gửi trả Response lại bị kẹt mất hút.

### 6.2. Tường lửa Lớp 7 (WAF - Web Application Firewall)
Bảo vệ vòng ngoài bằng Lớp 4 là chưa đủ. Giả sử Hacker gọi đúng vào Port 443 được phép, nhưng bên trong gói hàng hắn nhét một dòng code SQL Injection (`' OR 1=1 --`) hòng xóa sạch Database thì sao? Ông bảo vệ Lớp 4 mù quáng sẽ cho đi qua tuốt vì vỏ thư ghi đúng Port 443! Lúc này ta cần WAF (Như Cloudflare WAF, AWS WAF).

- **Cách hoạt động:** Ông bảo vệ Lớp 7 (WAF) là một chuyên gia soi mói. Ông ta bắt khách hàng mở khóa phong bì ra. Ông ta soi xét từng chữ trong cái Payload HTTP (Body JSON, URL query, Headers, Cookies). 
- **Quyền năng của WAF:**
  - Nếu thấy URL có chứa mã độc SQL/XSS: *Chặn đứng tắp lự (DROP).*
  - Nếu thấy 1 IP từ một quốc gia lạ gọi Port 443 đến 500 lần/giây (Tấn công DDOS lụt mạng): *Ban IP.*
  - Nếu Header `User-Agent` rỗng hoặc giống một con Bot cào dữ liệu (Scraping): *Bật ngay màn hình CAPTCHA bắt giải đố.*

### 6.3. Kiến trúc Defense-in-Depth (Phòng thủ nhiều lớp)
Kỹ sư hạ tầng (Platform Engineer) không bao giờ giao sinh mạng của hệ thống cho 1 loại Firewall. Họ quy hoạch mạng lưới theo nguyên tắc Phòng thủ nhiều lớp (Defense-in-Depth):
1. **Ngoài cùng (Cloudflare / Edge WAF):** Hứng bão, lọc mọi cuộc tấn công DDOS Lớp 7, soi kỹ Payload để chặn SQL Injection/XSS trước cả khi gói tin chạm tới Cloud của bạn.
2. **Lớp giữa (Public Subnet / DMZ):** Nơi cắm Load Balancer (ELB) hoặc Nginx. Dùng Security Group (Lớp 4) bóp chặt chỉ mở đúng Port 80 và 443. Mọi truy cập vào port khác lập tức bị từ chối.
3. **Lõi (Private Subnet):** Trái tim của hệ thống chứa Database và Backend. Lớp Firewall ở đây thiết lập gắt gao nhất: Cắt hoàn toàn kết nối từ Internet, chỉ cho phép Inbound Port 3306/5432 **ĐỘC NHẤT** từ địa chỉ IP Private của chính con Backend/Nginx nội bộ. Bất kỳ Server nào khác trong mạng dù biết IP cũng không thể đụng vào DB.

### 6.4. Cơ chế Mở cổng động (Dynamic IP Whitelisting) trong CI/CD

#### 6.4.1. Bản chất của Public IP, Private IP và Tường lửa
Khi triển khai hệ thống lên Cloud (như AWS EC2), máy chủ thường được cấp 2 loại địa chỉ IP:
- **Private IP (IP Nội bộ):** Chỉ dùng để giao tiếp trong mạng nội bộ (VPC). Ví dụ: Web Server gọi cho Database Server thông qua Private IP để đảm bảo Hacker ngoài Internet không thể "chạm" được vào Database. Nếu một con Service nội bộ bị nhiễm virus, nó bắt buộc phải chạy ngang qua router mạng nội bộ, đối mặt với các Tường lửa (Firewall) chặn đường trước khi đến được các Service khác.
- **Public IP (IP Công cộng):** Là "mặt tiền" để thế giới bên ngoài (Internet) gọi vào hệ thống của bạn (Ví dụ User truy cập vào Web App hoặc Cổng thanh toán).

**Tường lửa (AWS Security Group):**
Security Group đóng vai trò như "Bảo vệ gác cổng". Nếu bạn mở toang cổng điều khiển hệ thống (Port 22 - SSH) cho dải IP `0.0.0.0/0` (tức là ai cũng vào được), các con Bot rà quét của Hacker bên Nga, Trung Quốc... sẽ liên tục dội bom dò mật khẩu máy chủ của bạn 24/7.
Do đó, Best Practice là: **Chỉ cho phép duy nhất IP tĩnh mạng nhà bạn (ví dụ `42.116.178.24`) được phép kết nối vào cổng 22 (SSH).** Nghĩa là chỉ ai ngồi ở nhà bạn thì mới có quyền gõ lệnh điều khiển Server.

#### 6.4.2. Nghịch lý với CI/CD (GitHub Actions)
Khi bạn dùng GitHub Actions để tự động deploy code lên EC2, con Bot (Runner) của GitHub bản chất là các máy chủ ảo nằm trên hệ thống đám mây của Microsoft Azure.
- Con Bot này không nằm ở nhà bạn, nên IP của nó KHÔNG PHẢI là `42.116.178.24`.
- Mỗi lần bạn bấm Deploy, GitHub lại thuê và cấp cho con Bot một địa chỉ IP ngẫu nhiên khác nhau trên thế giới.
- Kết quả: Khi Bot GitHub gửi gói tin cố gắng SSH vào máy EC2 của bạn, **Tường lửa (Security Group)** thấy một IP lạ hoắc nằm ngoài danh sách ưu tiên liền lập tức "đánh chặn" và đá văng gói tin ra ngoài (Sinh ra lỗi `dial tcp: i/o timeout` mà bạn vừa gặp).

#### 6.4.3. Giải pháp Chuẩn DevOps: Mở cổng động (Dynamic IP Whitelisting)
Thay vì đập bỏ tường lửa và mở toang cổng cho cả thế giới vào (cực kỳ nguy hiểm), quy trình CI/CD sẽ được lập trình để "mở hé cửa" một cách thông minh, tự động và chớp nhoáng:

1. **Lấy tọa độ:** Khi Bot GitHub bắt đầu chạy, nó tự kiểm tra xem IP Public hiện tại của chính nó đang là gì (ví dụ `20.55.10.12`).
2. **Xin phép mở cửa:** Bot dùng một chiếc thẻ đặc quyền (AWS Access Key) gọi ra lệnh cho dịch vụ tường lửa AWS: *"Tôi là người quen, IP của tôi tạm thời hôm nay là `20.55.10.12`. Hãy mở đúng cái Port 22 cho duy nhất IP này vào đi."* (Đây chính là đoạn code `aws ec2 authorize-security-group-ingress`).
3. **Chui vào làm việc:** Lỗ hổng bằng cái mũi kim được tường lửa mở ra vừa đúng cho Bot chui vào. Bot dùng SSH copy source code và chạy lệnh khởi động Podman một cách êm ái.
4. **Hủy thẻ, đóng cổng:** Ngay khi Bot làm xong việc (hoặc lỡ bị lỗi đứt gánh giữa chừng), lệnh thu hồi (`aws ec2 revoke-security-group-ingress`) được cam kết luôn luôn chạy ở phút cuối cùng để "bịt lỗ" tường lửa lại. Máy EC2 của bạn lại trở về trạng thái "bế quan tỏa cảng" an toàn tuyệt đối!

**Kết luận:** Hạ tầng nội bộ vẫn được cách ly an toàn với Virus và Hacker ngoài Internet, nhưng luồng Deploy CI/CD vẫn chạy tự động trơn tru mà không cần con người nhúng tay. Đó là nghệ thuật kết hợp giữa Bảo mật (Security) và Tự động hóa (DevOps Automation).

---

## 7. CASE STUDY TOÀN DIỆN: Hành trình End-to-End của 1 Request

Hãy theo dõi một kịch bản siêu chi tiết: User (IP LAN `192.168.1.5`) gõ URL mở App HTTPS được host trên Cloud AWS (IP Private nội bộ AWS `10.0.0.5`, IP Public `13.212.0.5`).

### CHIỀU ĐI (REQUEST JOURNEY)
**1. Khởi tạo (Layer 7):** HĐH chạy bắt tay SSL/TLS (tạo Session Key). Trình duyệt gen ra khối JSON HTTP Request, rồi dùng Session Key mã hóa nó thành đống ký tự vô nghĩa. Khối này gọi là **Payload**.
**2. Đóng gói TCP (Layer 4):** HĐH chia nhỏ Payload, bọc **TCP Header** (Ghi rõ: `Source Port: 51234`, `Dest Port: 443`). Tạo thành **TCP Segment**.
**3. Đóng gói IP (Layer 3):** Bọc **IP Header** (`Source IP: 192.168.1.5`, `Dest IP: 13.212.0.5`). Tạo thành **IP Packet**.
**4. Đóng gói Frame (Layer 2):** Máy tính tính Subnet thấy IP Đích khác LAN. Tra bảng ARP lấy MAC của Router Wifi. Bọc **MAC Header**. Tạo thành **Ethernet Frame**.
**5. Truyền tải (Layer 1):** Card mạng dịch Frame thành xung điện, đẩy qua cáp đến Switch gia đình.
**6. Switch Nội bộ xử lý:** Đọc MAC Header, Flooding/Unicast đẩy xung điện trúng cổng Router Wifi.
**7. NAT Outbound tại Router:** Router xé Lớp 2. Đọc IP Packet. Thực hiện **NAT**, sửa `Source IP` từ `192.168.1.5` thành IP Public (`14.22.33.44`). Tạo Frame mới ném ra cáp quang Internet.
**8. Định tuyến Internet (Hop-by-Hop):** Qua hàng tá Router của Viettel, APG, Singtel... Tại mỗi trạm, Router chỉ xé Lớp 2 (Frame), tra Routing Table dựa vào Dest IP (Layer 3), rồi đóng một Frame mới ném đi. **Lưu ý: Suốt chặng giữa này không có NAT, IP Packet bên trong được giữ nguyên vẹn.**
**9. NAT Inbound tại Đích (AWS IGW):** Packet tới Internet Gateway (IGW) của AWS. IGW có luật Load Balancer, bóc IP Public đích (`13.212.0.5`), đổi thành IP Private máy ảo EC2 (`10.0.0.5`).
**10. Switch VPC AWS:** VPC đóng gói Frame bọc MAC Address của EC2 và truyền tới EC2.
**11. Giải nén (Decapsulation) tại Server:** 
- Card mạng EC2 nhận xung điện -> Frame. Đúng MAC mình -> Xé bọc Lớp 2.
- Linux đọc Packet. Đúng IP mình `10.0.0.5` -> Xé bọc Lớp 3.
- Linux đọc TCP Segment. Thấy `Dest Port 443` -> Ném Payload cho Nginx.
- Nginx dùng Session Key (đã thỏa thuận từ bước bắt tay TLS) giải mã cái Payload vô nghĩa kia ra thành chuỗi HTTP JSON gốc.
- Nginx xử lý điều hướng `proxy_pass` ném sang Process Backend (Golang ở port 8080).

### CHIỀU VỀ (RESPONSE JOURNEY)
Mọi thứ diễn ra ngược lại hoàn toàn. Backend trả HTTP Response -> Bọc TCP -> Bọc IP (Source: 10.0.0.5, Dest: 14.22.33.44) -> AWS IGW làm NAT đổi Source IP thành Public -> Bay qua đại dương về Router Wifi nhà bạn -> Router dò NAT Table bóc Dest IP Public trả lại thành 192.168.1.5 -> Switch ném về máy tính -> Trình duyệt Giải mã SSL -> Render UI!

---

## 7. Tường lửa (Security Group Layer 4)

- **Stateful Firewall:** Tường lửa hoạt động tại ranh giới mạng ảo (Layer 4). Nó can thiệp bằng cách đọc TCP/UDP Port. Chữ "Stateful" nghĩa là nó Ghi nhớ trạng thái phiên TCP. Nếu gói tin Ingress (Request) vượt qua tường lửa thành công, gói Egress (Response) tương ứng sẽ tự động được xả trạm mà không cần tạo Rule chiều ra.
- **Zero Trust:** Đóng mọi Ingress từ Internet (`0.0.0.0/0`), chỉ mở tối thiểu (80/443). Tuyệt đối không mở các Port Database/Cache ra ngoài Public. Khóa ICMP (Ping) để chống dò quét.

---

## 8. Phân giải Tên miền (DNS) & Đảo ngược Proxy tại Biên

- **Name Server (NS) là Quyền Lực Tối Cao:** Khi đổi NS về Cloudflare, bạn giao quyền sinh sát phân luồng traffic cho Edge Network của họ.
- **Proxying (TLS Termination):** Thay vì trỏ DNS A-Record vào thẳng AWS (dễ bị DDoS). Request đập vào Đám mây Cam Cloudflare. Tại đây Cloudflare đứng ra **Bắt tay SSL (TLS Termination)** với User, giải mã Lớp 7 ra quét WAF (chống SQL Injection), sau đó Cloudflare tự dùng Session Key nội bộ mã hóa lại và mở phiên TCP/IP gửi ngầm về Origin IP của AWS. IP thật của Server hoàn toàn tàng hình.

---

## 9. Sự Tiến Hóa Cơ Sở Hạ Tầng (Infrastructure Evolution)

Tất cả kiến trúc mạng đồ sộ trên đã tiến hóa qua 3 thời kỳ:
- **Bare Metal (Máy chủ vật lý):** Mua thiết bị, tự cắm cáp đồng, setup Switch vật lý tại Data Center. Tiền đầu tư khổng lồ, Scale kém.
- **VPS (Virtual Private Server):** Ảo hóa chia nhỏ máy tính. Giảm chi phí cứng nhưng xài hay không xài vẫn trả đủ tiền.
- **Cloud Computing (AWS/GCP):** Mọi thứ (Kể cả Card mạng, Switch, Router) đều được ảo hóa (Software-Defined Networking - SDN). Hạ tầng quản lý bằng Code (Terraform), tính tiền theo giây (Pay-as-you-go). Kỹ sư thoát khỏi phần cứng để tập trung 100% vào Kiến trúc & Bảo mật.

---

## 10. Xương sống của Internet (The Internet Backbone & Bức màn Bí ẩn)

Rất nhiều kỹ sư nhầm tưởng rằng Internet thuộc về Google, AWS hay Quân đội Mỹ. Thực chất, Internet hiện đại không có Tổng giám đốc, không có máy chủ trung tâm. Nó được thống trị bởi một liên minh các "Ông trùm" thầm lặng nằm dưới đáy đại dương.

### 10.1. Kim tự tháp Quyền lực Internet
- **Đỉnh Kim tự tháp (Tier-1 Networks):** Đây là các "Ông trùm" đích thực (Ví dụ: AT&T, Lumen, Arelion, NTT). Họ sở hữu hệ thống cáp quang biển liên lục địa vĩ đại nhất. **Đặc điểm sống còn:** Họ kết nối ngang hàng (Peering) với nhau và KHÔNG BAO GIỜ phải trả tiền cước cho bất kỳ ai trên thế giới. Nếu 3 trong số họ rút điện, một nửa thế giới sẽ quay về thời đồ đá.
- **Tầng giữa (Tier-2 ISPs):** Điển hình là các nhà mạng quốc gia (VNPT, Viettel, FPT). Họ phủ sóng cáp đến từng hộ gia đình, nhưng họ không có cáp vươn tới Mỹ. Do đó, VNPT bắt buộc phải đem hàng triệu USD đi **Mua đường truyền (Transit)** của các Tier-1 để người dùng quốc gia có thể kết nối ra quốc tế.

### 10.2. Giải mã hiểu lầm: "Mua IP" vs "Mua Đường truyền"
Có một sự nhầm lẫn kinh điển: *"Google mua IP của Tier-1 để làm dịch vụ"*. Đây là nhận định hoàn toàn sai lầm. Bạn cần phân biệt rõ 2 loại tài nguyên:
1. **Tài nguyên Logic (IP Address / ASN):** Được cấp phát bởi Tổ chức quản lý phi lợi nhuận toàn cầu (IANA). Cả Tier-1, Viettel hay Google đều phải làm đơn xin IANA cấp IP. Xin IP giống như được cấp cái "Số nhà".
2. **Tài nguyên Vật lý (Băng thông cáp quang):** Đây là thứ mà Tier-1 sở hữu. Có số nhà (IP) nhưng không có đường nhựa (Cáp quang) thì dữ liệu không thể đi đâu được. 

**Tại sao dạo này Google/Meta lại tự rải cáp quang biển?**
Google và Meta có IP riêng, nhưng ngày xưa họ vẫn phải làm "khách hàng" trả tiền cước (Transit) cho Tier-1 để mượn đường cáp vận chuyển data. Tuy nhiên, lưu lượng Youtube và Cloud hiện nay chiếm tới hơn 20% toàn bộ Internet thế giới. Trả tiền cho Tier-1 quá đắt đỏ và bị kẹt xe. Thế là Google tự vác tàu đi xây luôn hệ thống cáp riêng xuyên đại dương (Ví dụ cáp quang Curie, Dunant) để Server của họ nói chuyện trực tiếp với nhau với độ trễ cực thấp mà không cần dựa dẫm vào Tier-1 nữa.

### 10.3. Bí mật Vật lý dưới đáy đại dương (Submarine Cables)
Nhiều người nghĩ cáp biển truyền dữ liệu bằng xung điện (Volt). Thực tế, xung điện (Volt) truyền trên cáp đồng gặp phải kẻ thù chí mạng là **Điện trở**. Đi vài chục cây số là tín hiệu điện bị hao mòn (Attenuation) suy yếu về 0.

- **Dữ liệu truyền bằng Ánh sáng:** Để đi qua Thái Bình Dương xa hàng vạn kilomet, cáp biển sử dụng các sợi quang học làm từ **Thủy tinh siêu tinh khiết (Ultra-pure silica glass)** mỏng hơn sợi tóc người. Dữ liệu (0 và 1) được truyền đi bằng các nhịp chớp tắt của đèn Laser. Ánh sáng này phản xạ toàn phần bên trong sợi thủy tinh để lao đi với vận tốc cực khủng.
- **Nghịch lý của dòng điện 10.000 Volt:** Dù thủy tinh siêu trong suốt, ánh sáng đi được khoảng 50-100km dưới đáy biển vẫn bị mờ đi. Người ta bắt buộc phải đặt các Cục kích sáng (Optical Amplifiers / Repeaters) dọc đường để hút ánh sáng yếu vào và bắn ánh sáng mạnh ra. **Và để nuôi sống các cục Repeater này ở độ sâu 4.000 mét dưới đáy biển, lớp vỏ của sợi cáp quang chứa một ống đồng khổng lồ truyền dòng điện một chiều (DC) lên tới 10.000 Volt!** 

> **Kết luận:** Dưới đáy biển, **Dữ liệu chạy bằng Ánh sáng Laser, còn Dòng điện 10.000 Volt chạy song song dọc theo cáp chỉ để làm duy nhất một việc: Cấp nguồn nuôi hệ thống kích sáng (Repeater) dọc đường!**

---

## 11. Giải mã tệp .pem của AWS và Thuật toán Băm HMAC-SHA256

### 11.1. Tệp key.pem của AWS thực chất chứa cái gì?
Khi bạn khởi tạo một máy ảo EC2 trên AWS, AWS yêu cầu bạn tải về một tệp key.pem để dùng cho việc SSH (Remote Connect). Vậy tại sao không nhập User/Password như bình thường (ví dụ Telnet) mà phải dùng cái file này?
- **Vấn đề của Password:** Nếu dùng Password, Hacker có thể dùng Tool Brute-force (đoán mật khẩu tự động 10,000 lần/giây) rà trúng mật khẩu của bạn. Hơn nữa, truyền mật khẩu qua mạng (như Telnet cũ) cực kỳ dễ bị nghe lén. SSH (Secure Shell) sinh ra để khai tử Password bằng sức mạnh Toán học (Mã hóa Bất đối xứng RSA hoặc ED25519).
- **Vậy tệp .pem chứa gì?** Chữ PEM viết tắt của *Privacy Enhanced Mail* - thực chất nó chỉ là một định dạng file text đơn thuần để lưu trữ khóa mã hóa.
Khi bạn bấm nút "Create Key Pair", AWS tự động chạy thuật toán sinh ra một cặp Khóa sinh đôi: **Public Key** và **Private Key**.
  - **Public Key:** AWS bí mật nhét sẵn cái khóa công khai này vào bên trong máy ảo EC2 (nằm ở đường dẫn ~/.ssh/authorized_keys).
  - **Private Key:** AWS mã hóa nó dưới định dạng văn bản Base64, nhét vào tệp key.pem và đưa cho bạn tải về. Bên trong tệp này, nếu bạn mở bằng Notepad, nó sẽ là một chuỗi ký tự rác khổng lồ nằm giữa hai dòng chữ: -----BEGIN RSA PRIVATE KEY----- và -----END RSA PRIVATE KEY-----.

**Luồng hoạt động CHUẨN XÁC của SSH (Sử dụng Chữ ký số - Digital Signature):**
Rất nhiều người lầm tưởng Server sẽ dùng Public Key để mã hóa bài toán rồi bắt Client giải. Nhưng KHÔNG, SSH Authentication dùng cơ chế **Chữ ký số (Ký bằng Private Key, Xác minh bằng Public Key)**:

1. **Client Gõ cửa:** Lệnh `ssh` của bạn kết nối và báo với EC2: *"Tôi muốn login bằng cái Public Key này nè, anh coi trong kho có không?"*
2. **Server Kiểm tra:** Server lôi file `~/.ssh/authorized_keys` ra tìm. *"À có Public Key này. Nhưng làm sao tao biết mầy là chủ nhân thực sự? Tao sẽ gửi cho mầy một chuỗi số ngẫu nhiên (Challenge), hãy dùng chìa khóa của mầy KÝ TÊN vào đây để chứng minh đi!"*. Server gửi chuỗi Challenge trần trụi về cho Client.
3. **Client Ký tên (Tạo Chữ ký số):** Máy tính của bạn lập tức đọc cái **Private Key** (từ tệp `.pem`). Nó băm cái chuỗi Challenge kia ra, rồi dùng Private Key **MÃ HÓA** cục băm đó. Kết quả sinh ra một khối dữ liệu gọi là **Chữ ký số (Digital Signature)**, rồi ném ngược lên Server.
4. **Server Xác minh:** Server nhận được Chữ ký số. Nó lôi ngay cái **Public Key** (lưu sẵn trong `authorized_keys`) ra để **GIẢI MÃ** cái chữ ký đó.
   - *Phép màu Toán học:* Định lý RSA quy định, thứ gì bị khóa bởi Private Key, thì CHỈ CÓ Public Key sinh đôi với nó mới mở ra được.
   - *Kết luận:* Server giải mã thành công và đối chiếu thấy khớp 100% với chuỗi ban đầu $\rightarrow$ Server chốt hạ: *"Trong vũ trụ này, chỉ có kẻ nắm giữ Private Key thật mới có thể tạo ra được cái chữ ký mà Public Key của tao mở được!"*. 
   - BÙM! Mở cổng kết nối. Bạn được login thành công mà Hacker có bắt lén trọn vẹn gói tin trên mạng cũng vô dụng, vì Hacker vĩnh viễn không thể làm giả Chữ ký số nếu không có file `.pem` trong tay!

### 11.2. Mã hóa Một chiều: HMAC-SHA256
Bạn đang thắc mắc thuật toán băm (Hashing) một chiều như HMAC-SHA256 hoạt động ra sao, có phải là dịch ra số ASCII rồi chạy công thức không? Chính xác là vậy, nhưng phức tạp và tàn bạo hơn nhiều!

**A. SHA-256 (Secure Hash Algorithm 256-bit) là gì?**
Nó **KHÔNG PHẢI LÀ MÃ HÓA** (Encryption). Mã hóa (như RSA/AES) là có đi có lại: Gói vào được thì tháo ra được. SHA-256 là **HÀM BĂM MỘT CHIỀU (Hashing)**: Tức là cho 1 con bò đi qua máy xay thịt, ra được cây xúc xích. Bạn không bao giờ có thể từ cây xúc xích đó nhào nặn ngược lại thành con bò.

**Cơ chế hoạt động của SHA-256:**
1. **Dịch bit:** Máy tính biến file chữ, file ảnh, file video thành dải nhị phân (0 và 1) khổng lồ dựa trên bảng mã (ví dụ ASCII).
2. **Băm nát:** Khối dữ liệu đó được chặt ra thành từng khúc dài bằng nhau (512-bit/khúc).
3. **Cối xay ma thuật (64 vòng lặp):** Máy tính đưa từng khúc 512-bit đó vào một "cối xay toán học". Nó sẽ thực hiện **64 vòng lặp** với các phép toán nhị phân cực kỳ man rợ như: **AND, OR, XOR (Dịch bit, đảo bit)** xen kẽ liên tục. Cái đống 0 và 1 bị trộn nát tới mức không còn nhận ra hình thù ban đầu.
4. **Đầu ra duy nhất:** Cuối cùng, dù bạn ném vào đó 1 chữ "A" hay ném nguyên bộ phim 4K nặng 50GB, cái lỗ output của SHA-256 luôn luôn ọt ra một chuỗi cố định gồm đúng 256 số 0 và 1 (Quy đổi ra chuỗi 64 ký tự chữ và số hệ Hex).
-> *Hiệu ứng Cánh bướm (Avalanche effect):* Bạn chỉ cần thay một chữ "a" thường thành "A" hoa trong bộ phim 50GB, toàn bộ 64 ký tự đầu ra sẽ thay đổi hoàn toàn khác biệt.

**B. Vậy thêm chữ "HMAC" vào SHA-256 để làm gì?**
Chữ HMAC viết tắt của *Hash-based Message Authentication Code*.
Giả sử bạn gửi một tin nhắn *"Chuyển 5 tỷ cho anh Trank"* và đính kèm cái mã Hash SHA-256 của tin nhắn đó. Hacker ở giữa mạng chụp được, hắn tráo tin nhắn thành *"Chuyển 5 tỷ cho Hacker"*, rồi hắn tự chạy SHA-256 sinh ra cái mã Hash mới ghép vào. Thế là toang!

**HMAC** ra đời để chống trò này. Để tính HMAC-SHA256, hai bên (Client và Server) phải có chung 1 cái **Khóa Bí Mật (Secret Key)**.
- Khi tính toán, máy tính không chỉ băm cái Data, mà nó sẽ băm cái Data **trộn chung** với cái Khóa Bí Mật thông qua 2 lớp đệm toán học (gọi là ipad và opad).
- Tức là: Output = SHA256(SecretKey + SHA256(SecretKey + Data)) (viết tóm tắt cho dễ hiểu).
- Hacker chặn giữa đường, tráo Data, muốn tính lại mã HMAC mới nhưng chịu chết vì... **hắn không biết cái Secret Key là gì để đưa vào cối xay!** 
-> HMAC vừa có khả năng băm một chiều (chống dịch ngược) của SHA-256, vừa chứng minh được "Danh tính người gửi" (Authentication) nhờ cái Secret Key bí mật.

### 11.3. Ứng dụng thực tế: Mối tình giữa JWT (JSON Web Token) và HMAC-SHA256
Rất nhiều Lập trình viên hay nhầm lẫn giữa JWT và thuật toán HMAC. Thực chất, **JWT là một tấm thẻ thông hành (Ứng dụng)**, và nó thường **sử dụng HMAC-SHA256 (Công cụ)** để làm chức năng "đóng dấu chống làm giả" (Signature).

Một token JWT hoàn chỉnh được ghép lại từ 3 phần, cách nhau bởi dấu chấm: `Header.Payload.Signature`.

**A. Tại sao lại phải băm Header và Payload ra Base64?**
- Có một sự hiểu lầm lớn: Base64 KHÔNG PHẢI LÀ MÃ HÓA (Encryption), cũng KHÔNG PHẢI LÀ BĂM (Hashing). Chữ "băm ra Base64" là sai bản chất kỹ thuật. Base64 thực chất là **Mã hóa chuỗi (Encoding)**.
- **Lý do phải Encode:** Dữ liệu Payload gốc là một chuỗi JSON (ví dụ: `{"user": "trank", "role": "admin"}`). Chuỗi JSON này chứa các ký tự đặc biệt như ngoặc nhọn `{ }`, dấu phẩy `,`, dấu nháy kép `"`, khoảng trắng, v.v. Nếu bạn ném trực tiếp cục JSON này lên thanh URL của trình duyệt hoặc nhét vào HTTP Header (như `Authorization: Bearer {...}`), các thiết bị mạng trung gian (như Router, Load Balancer) sẽ đọc không hiểu các ký tự đặc biệt này và lập tức đánh rớt gói tin.
- **Giải pháp:** Base64Url sẽ lấy cái chuỗi JSON rắc rối kia, nhào nặn biến nó thành một chuỗi văn bản an toàn chỉ bao gồm các ký tự thân thiện với mạng lưới: `A-Z, a-z, 0-9, -, _`. Nhờ vậy, cái Token có thể bay vèo vèo qua mọi ngóc ngách của Internet mà không bị vỡ định dạng. 
-> *Chốt lại: Việc dịch sang Base64 chỉ nhằm mục đích "Giao hàng an toàn qua mạng HTTP", hoàn toàn không có tác dụng bảo mật (vì bất kỳ ai cũng có thể dịch ngược Base64 về lại JSON bằng công cụ online).*

**B. Cách Server nhào nặn ra cái JWT (Mục đích bảo mật nằm ở Signature)**
1. **Header:** Server khai báo thuật toán sẽ dùng, ví dụ `{"alg": "HS256"}` (HS256 chính là HMAC-SHA256). Đem Encode Base64 $\rightarrow$ Ra cục số 1.
2. **Payload:** Server nhét thông tin User vào, ví dụ `{"user": "trank", "role": "admin"}`. Đem Encode Base64 $\rightarrow$ Ra cục số 2.
3. **Signature (Chữ ký):** Đây là lúc HMAC-SHA256 ra sân. 
   - Server lấy Cục 1 ghép với Cục 2 (Header + Payload).
   - Server lấy cái **SECRET KEY** (Khóa bí mật giấu sâu trong file `.env` của Backend).
   - Đút cả 2 thứ trên vào cối xay: `HMAC-SHA256 ( Secret_Key, (Cục 1 + "." + Cục 2) )`.
   - Kết quả văng ra chính là Cục 3 (**Signature**).

**C. Tại sao Hacker không thể làm giả JWT?**
Cái cục `Header.Payload.Signature` được ném cho Client lưu. Lúc này Hacker có thể bắt được cục JWT, lên trang jwt.io dịch ngược phần Payload Base64 ra và sửa chữ `"user"` thành `"admin"`.
Nhưng khi Hacker gửi cục JWT giả này lên Server:
- Server tách cái `Signature_Cũ` ra.
- Server lấy cái `Header` và `Payload_Của_Hacker` đút chung vào cối xay HMAC với cái **Secret Key** (chỉ Server mới biết) để tính ra `Signature_Mới`.
- Đem so sánh: `Signature_Mới` khác hoàn toàn `Signature_Cũ` đi kèm (do Hiệu ứng cánh bướm của hàm băm).
- Server chốt hạ: *"Dữ liệu Payload đã bị sửa đổi trên đường bay!"* $\rightarrow$ Đuổi cổ Hacker (Lỗi 401 Unauthorized).

**Kết luận tối thượng:** Hacker có thể nhìn thấy nội dung JWT, có thể sửa nội dung JWT, nhưng **vĩnh viễn không thể làm giả được Signature** vì Hacker không biết cái Secret Key là gì để bỏ vào thuật toán băm HMAC-SHA256!
