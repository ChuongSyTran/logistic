# Quy trình triển khai và vận hành

Tài liệu này chuẩn hóa quy trình (Workflow) triển khai hạ tầng từ số không (Zero) đến khi vận hành toàn diện một hệ thống phần mềm phân tán (Distributed System) trên nền tảng Cloud. Quy trình được thiết kế theo tư duy Cơ sở hạ tầng dưới dạng Mã (Infrastructure as Code - IaC).

---

## GIAI ĐOẠN 1: KHỞI TẠO TIỀN ĐỀ (PRE-REQUISITES)
Giai đoạn chuẩn bị định danh, quyền hạn và khóa mã hóa trước khi tương tác với Cloud API.

1. **Quản trị Tên miền (Domain Management):** 
   - Đăng ký tên miền (Ví dụ: `glolog.dev`) và cấu hình Name Server (NS) chuyển giao quyền quản lý về Cloudflare.
   - Cấp phát **API Token** từ Cloudflare Dashboard với quyền hạn can thiệp vào DNS Record để phục vụ Terraform.
2. **Quản trị Cloud Provider (Ví dụ: AWS / GCP):**
   - Khởi tạo **Key Pair (Chìa khóa SSH)** định dạng `.pem`. Đây là phương thức xác thực phi đối xứng (Asymmetric Cryptography) bắt buộc để truy cập máy chủ (Vô hiệu hóa xác thực bằng mật khẩu truyền thống).
   - Thiết lập IAM User, cấp quyền Programmatic Access và tải bộ Credentials (`Access Key ID` & `Secret Access Key`).
3. **Cấu hình Môi trường Local:**
   - Liên kết môi trường phát triển cục bộ với Cloud API thông qua CLI (`aws configure`).

---

## GIAI ĐOẠN 2: CẤP PHÁT HẠ TẦNG TỰ ĐỘNG (INFRASTRUCTURE PROVISIONING)
Sử dụng Terraform để định nghĩa hạ tầng dưới dạng mã nguồn (IaC). Trình tự logic quy hoạch tài nguyên yêu cầu tuân thủ nguyên lý Đồ thị phụ thuộc (Directed Acyclic Graph - DAG).

1. **Định nghĩa Kiến trúc Mạng (Networking & Security):**
   - Thiết lập Data Source để truy vấn tự động ID của OS Image (AMI) mới nhất (Ví dụ: Ubuntu 24.04 LTS).
   - Khởi tạo Security Group (Firewall). Giới hạn luồng Ingress tại Port 22 (Quản trị mạng riêng) và Port 80/443 (Giao thông Web).
2. **Khởi tạo Nút Điện toán (Compute Instance):**
   - Cấp phát Máy ảo. Khai báo ràng buộc chặt chẽ (Binding) với ID của Security Group và tên của Key Pair (SSH).
   - Định nghĩa quy mô Block Storage (Ví dụ: 30GB, chuẩn `gp3`).
3. **Phân giải Tên miền (DNS Configuration):**
   - Tạo DNS Record (A Record) trên hệ thống Cloudflare.
   - Ánh xạ động Public IP của máy ảo vừa được Terraform cấp phát sang Subdomain. Kích hoạt cờ Proxy (Edge Reverse Proxy).
4. **Thực thi Pipeline (CI/CD Local):**
   - Lệnh `terraform init`: Tải thư viện Provider API.
   - Lệnh `terraform plan`: Phân tích DAG và in ra Execution Plan.
   - Lệnh `terraform apply`: Thực thi gọi Cloud API để cấp phát toàn bộ hạ tầng thực tế.

---

## GIAI ĐOẠN 3: CẤU HÌNH MÔI TRƯỜNG NODE (CONFIGURATION MANAGEMENT)
Sau khi phần cứng và mạng lưới đã được Provider cấp phát thành công, tiến hành truy cập máy chủ để cấu hình hệ điều hành và cài đặt Runtime Dependencies.

1. **Xác thực Mạng Khách (SSH Authentication):**
   - **SSH (Secure Shell) là gì?** 
     - Đây là giao thức mạng chuẩn công nghiệp dùng để đăng nhập và điều khiển máy chủ từ xa (Remote Login) một cách bảo mật. 
     - Khác với HTTP truyền dữ liệu hiển thị Web, SSH truyền các **câu lệnh Terminal**. Nó chạy mặc định ở **Port 22**.
     - Nhờ mã hóa mạnh (Asymmetric Cryptography với file `.pem`), dù hacker có đứng giữa chặn mạng cũng chỉ thấy các đoạn mã vô nghĩa, không thể biết bạn đang gõ lệnh gì hay ăn cắp mật khẩu.
   - **Cú pháp Lệnh SSH:** 
     - Mở Terminal tại thư mục chứa file chìa khóa (`.pem`) và gõ lệnh:
       ```bash
       ssh -i "logistic-key.pem" ubuntu@<PUBLIC_IP_CỦA_EC2>
       ```
     - *Phân tích tham số:*
       - `ssh`: Tiện ích dòng lệnh Secure Shell Client.
       - `-i "logistic-key.pem"`: Chỉ định đường dẫn tới Identity File (Private Key) để thực hiện quá trình xác thực phi đối xứng (Asymmetric Authentication).
       - `ubuntu`: Người dùng mặc định (Default User) của hệ điều hành Ubuntu trên AWS.
       - `<PUBLIC_IP_CỦA_EC2>`: Địa chỉ Public IPv4 của máy chủ. Yêu cầu kết nối trực tiếp vào IP thay vì FQDN (`api.glolog.dev`) nhằm bypass lớp Cloudflare Proxy đang giới hạn lưu lượng theo chuẩn HTTP/HTTPS.
2. **Cập nhật Nhân Hệ điều hành (System Patching):**
   - Cú pháp: `sudo apt update && sudo apt upgrade -y`
   - *Đặc tả:* 
     - `apt update`: Đồng bộ hóa metadata của các gói phần mềm từ Package Repository của Ubuntu.
     - `apt upgrade -y`: Thực thi cập nhật hệ thống và tự động chấp nhận (`-y`) các bản vá bảo mật mới nhất nhằm giảm thiểu bề mặt tấn công hệ điều hành.
3. **Thiết lập Môi trường Chạy ứng dụng (Runtime Setup):**
   - Cú pháp: `sudo apt install -y nginx docker.io docker-compose git`
   - *Đặc tả:*
     - `nginx`: Cài đặt Web Server để đảm nhiệm vai trò Edge Reverse Proxy cục bộ.
     - `docker.io` & `docker-compose`: Triển khai nền tảng Containerization nhằm tiêu chuẩn hóa và cô lập môi trường runtime của ứng dụng.
     - `git`: Cài đặt Version Control Client phục vụ đồng bộ mã nguồn.
4. **Triển khai Mã nguồn (Source Code Synchronization):**
   - Cú pháp: `git clone <URL_REPO> && cd <PROJECT_DIR>`
   - *Đặc tả:* Đồng bộ mã nguồn nguyên bản từ Remote Repository về môi trường triển khai (Local) của máy chủ.

## GIAI ĐOẠN 4: TỰ ĐỘNG HÓA TRIỂN KHAI ỨNG DỤNG (APPLICATION CI/CD PIPELINE)
Khác với phương pháp cổ điển là phải SSH thủ công vào server để `git clone` và sửa file `.env` bằng tay, dự án này áp dụng luồng tự động hóa 100% bằng **GitHub Actions** (được định nghĩa tại job `deploy-app` trong file `terraform.yml`).

Quy trình tự động hóa đưa Source Code lên EC2 và khởi chạy được thực hiện tuần tự qua các bước sau:

1. **Định vị Máy chủ (Retrieve EC2 IP):**
   - *Cú pháp lệnh:* `IP=$(terraform output -raw server_logistic_ip)`
   - *Mục đích:* Dùng Terraform xuất ra địa chỉ IP Public của máy EC2 vừa được tạo ở Giai đoạn 2. Lưu IP này vào biến môi trường của GitHub Runner (`GITHUB_ENV`) để các bước sau biết đích đến để kết nối SSH.

2. **Khởi tạo thư mục làm việc (Workspace Setup):**
   - *Mục đích:* Đăng nhập SSH vào EC2 và chạy lệnh `mkdir -p /home/ubuntu/logistic` để đảm bảo thư mục gốc dự án luôn tồn tại trước khi chép code sang.

3. **Chuyển giao Mã nguồn (Secure Copy - SCP):**
   - *Cú pháp lệnh:* Sử dụng plugin `appleboy/scp-action`. Truyền vào IP, Username (`ubuntu`), Private Key (`EC2_SSH_PRIVATE_KEY`), và danh sách các thư mục mã nguồn (`source: "api/,gateway_service/,docker-compose.yml..."`).
   - *Mục đích:* Thay vì cài Git trên EC2 và cấu hình xác thực lằng nhằng để pull code, Bot GitHub sẽ "đóng gói" các thư mục cần thiết và gửi thẳng qua giao thức bảo mật SCP vào thư mục `/home/ubuntu/logistic` trên EC2. Máy EC2 hoàn toàn không cần lưu giữ thông tin đăng nhập GitHub của bạn.

4. **Sinh cấu hình động (Dynamic `.env` Injection):**
   - *Cú pháp lệnh:* Mở kết nối SSH (`appleboy/ssh-action`) và dùng lệnh `cat << 'EOF' > .env` kết hợp với `${{ secrets.X }}`.
   - *Mục đích:* File `.env` chứa các thông tin cực kỳ nhạy cảm (Mật khẩu Database, Secret Key...) luôn bị cấm đưa lên GitHub (chặn bởi `.gitignore`). Do đó, Bot GitHub sẽ lấy các biến an toàn đã được bạn thiết lập sẵn trong **GitHub Secrets** và "tiêm" (inject) trực tiếp vào file `.env` ngay trên không gian mạng của EC2.

5. **Khởi chạy Hệ thống Container (Orchestration):**
   - *Cú pháp lệnh:* `sudo podman-compose up -d --build`
   - *Mục đích:* Chờ cho `cloud-init` cài đặt xong Podman. Kích hoạt Podman đọc file `docker-compose.yml`, tự động tải các base image (Postgres, NATS, Kafka...), tiến hành Build source code GoLang cho từng Microservice nội bộ, nạp các biến môi trường từ file `.env` vừa tạo, và khởi chạy toàn bộ mạng lưới dưới dạng các tiến trình chạy ngầm (Daemon mode `-d`).

---

## GIAI ĐOẠN 5: BẢO TRÌ & THU HỒI TÀI NGUYÊN (MAINTENANCE & TEARDOWN)
- Khi hoàn tất phiên làm việc, giai đoạn kiểm thử, hoặc cần hủy bỏ hạ tầng, thực thi thu hồi toàn bộ tài nguyên bằng lệnh `terraform destroy`.
- Trình biên dịch Terraform sẽ đối chiếu với tệp tin State (Trạng thái), tự động đảo ngược cấu trúc phụ thuộc và hủy bỏ (De-provision) toàn bộ máy ảo, tường lửa, và bản ghi DNS để triệt tiêu chi phí Cloud.

---

## PHỤ LỤC: CHEATSHEET LỆNH CONTAINER CLI (PODMAN/DOCKER)
Danh sách các câu lệnh chẩn đoán (Diagnostics) cốt lõi dành cho DevOps Engineer:

- **Thâm nhập Không gian Container (Exec/Dive):**
  - Mục đích: Chẩn đoán file hệ thống, cấu hình biến môi trường bên trong Namespace cách ly.
  - Cú pháp: `podman exec -it <container_name> /bin/sh` (hoặc `/bin/bash`).
- **Theo dõi Nhật ký Giao dịch (Tail Logs):**
  - Theo dõi realtime: `podman logs -f <container_name>`
- **Kiểm tra trạng thái Vòng đời (Process Status):**
  - Xem Container đang hoạt động: `podman ps`
  - Xem toàn bộ lịch sử Container: `podman ps -a`
- **Giải phóng Bộ nhớ & Rác Hệ thống (Pruning):**
  - Dọn dẹp cache build, Image mồ côi: `podman system prune -f`
