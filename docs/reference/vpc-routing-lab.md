# Thực hành route table, NAT và NACL bằng network namespace

Lý thuyết ở [aws-architecture.md](aws-architecture.md) mục 4 chỉ đọng lại khi tận mắt thấy packet
bị vứt, bị đổi địa chỉ, hay chết ở chiều về. Bài này dựng một "VPC thu nhỏ" ngay trên máy Linux,
không tốn tiền AWS, mất khoảng 60 phút.

## Trước khi bắt đầu

**Network namespace** là một ngăn xếp mạng riêng trong cùng một kernel: có interface, bảng route
và firewall của riêng nó. Mỗi namespace trong bài đóng vai một máy. Dây nối giữa hai namespace là
một cặp **veth** — sợi cáp ảo có hai đầu.

**An toàn:** mọi lệnh chỉ tác động lên các namespace do bài này tạo ra. Route, firewall và
interface của máy bạn không bị đụng tới. Xong bài, hoặc làm hỏng giữa chừng, thì chạy mục
*Dọn dẹp* ở cuối rồi làm lại từ bước 0.

**Cách học:** mỗi bước có phần **Dự đoán**. Viết câu trả lời ra giấy *trước* khi chạy lệnh. Chỗ
dự đoán sai chính là chỗ mô hình trong đầu bạn đang lệch.

### Lab giống và khác AWS ở đâu

| Trong lab | Trên AWS | Khác biệt cần nhớ |
|---|---|---|
| Namespace `vpcr` | Router ngầm `.1` của VPC | AWS chạy trong hypervisor; lab dùng Linux thường |
| Bảng 100, 101, 102 và `ip rule iif …` | Main route table, route table public/private, association | Linux phải xoá luật tra bảng `main` mặc định để packet không khớp thì dừng, giống VPC |
| Tự gõ hai dòng `10.99.x.0/24 dev …` vào **mỗi** bảng | Dòng `local` AWS tự thêm vào mọi route table, không xoá được | Trên AWS **không bao giờ** phải tạo route để các subnet thấy nhau; lab phải gõ tay vì Linux không tự làm |
| `vpcr` trả `Network is unreachable` | VPC vứt packet im lặng | Trên AWS ứng dụng chờ tới timeout thay vì lỗi ngay |
| NAT 1:1 bằng nftables trên `vpcr` | Internet Gateway + Elastic IP | Linux làm bằng conntrack; IGW thật không cần state. Hành vi quan sát được giống nhau |
| Namespace `natgw` cắm vào subnet public | NAT Gateway chế độ zonal (mặc định) | Giống cả vị trí lẫn cơ chế: PAT có state. Chế độ regional không cần public subnet — xem aws-architecture.md mục 4.5 |
| Chain nftables trên `vpcr` | NACL / Security Group | Security Group thật nằm ở ENI của từng máy, không nằm ở router |

Dải `10.99.0.0/16` cố tình khác `192.168.0.0/20` của bản vẽ mục tiêu, để không lộ đáp án bài chia
CIDR và không trùng mạng nhà. `198.51.100.0/24` là dải dành riêng cho tài liệu, đóng vai Internet.

### Sơ đồ

```text
 subnet public 10.99.1.0/24
 ┌─────────────────────┐
 │ ec2a   10.99.1.10   │──┐
 │ natgw  10.99.1.20   │──┤ br-pub
 └─────────────────────┘  │  (natgw thêm ở bước 5)
                          │
                  ┌───────┴────────┐  r-inet 198.51.100.1          ┌─────────────────┐
                  │ vpcr           │───────────────────────────────│ inet            │
                  │ router .1      │  + .101 = Elastic IP của ec2a │ 198.51.100.10   │
                  │ 10.99.1.1      │  + .102 = Elastic IP của natgw│ (vai Internet)  │
                  │ 10.99.2.1      │                               └─────────────────┘
                  └───────┬────────┘
                          │ r-b
 subnet private 10.99.2.0/24
 ┌─────────────────────┐  │
 │ ec2b   10.99.2.10   │──┘
 └─────────────────────┘
```

### Chuẩn bị

Cần Linux, quyền `sudo`, `iproute2`, `nftables`, `curl`, `python3`:

```bash
ip -V && nft -v && curl --version | head -1 && python3 --version
```

Mở **hai terminal**. Toàn bộ bài chạy ở terminal 1; terminal 2 chỉ để xem log của "Internet".

Chạy ở terminal 1, một lần cho cả buổi:

```bash
sudo -v
hit() { sudo ip netns exec "$1" curl -sS -o /dev/null -m 3 -w "%{http_code}\n" "$2"; }
```

`hit <máy> <url>` gọi HTTP từ máy đó và in mã trả về: `200` là tới nơi, `000` là không có phản hồi
HTTP nào. Dòng lỗi của curl in ngay phía trên cho biết vì sao.

## Bước 0 — Dựng mạng

**Tạo bốn máy và nối dây:**

```bash
for n in ec2a ec2b vpcr inet; do sudo ip netns add $n; done

sudo ip link add a-eth0 netns ec2a type veth peer name r-a    netns vpcr
sudo ip link add b-eth0 netns ec2b type veth peer name r-b    netns vpcr
sudo ip link add i-eth0 netns inet type veth peer name r-inet netns vpcr
for n in ec2a ec2b vpcr inet; do sudo ip -n $n link set lo up; done
```

`ip -n <máy> …` chạy lệnh `ip` bên trong namespace đó.

**Gán IP và default gateway cho từng máy:**

```bash
sudo ip -n ec2a addr add 10.99.1.10/24 dev a-eth0 && sudo ip -n ec2a link set a-eth0 up
sudo ip -n ec2a route add default via 10.99.1.1
sudo ip -n ec2b addr add 10.99.2.10/24 dev b-eth0 && sudo ip -n ec2b link set b-eth0 up
sudo ip -n ec2b route add default via 10.99.2.1
sudo ip -n inet addr add 198.51.100.10/24 dev i-eth0 && sudo ip -n inet link set i-eth0 up
```

Trên AWS, DHCP cấp sẵn IP và default gateway `.1` cho mỗi EC2; ở đây ta tự gán.

**Dựng router của VPC:**

```bash
sudo ip -n vpcr link add br-pub type bridge
sudo ip -n vpcr link set r-a master br-pub
sudo ip -n vpcr addr add 10.99.1.1/24 dev br-pub
sudo ip -n vpcr addr add 10.99.2.1/24 dev r-b
sudo ip -n vpcr addr add 198.51.100.1/24 dev r-inet
for i in br-pub r-a r-b r-inet; do sudo ip -n vpcr link set $i up; done
sudo ip netns exec vpcr sysctl -qw net.ipv4.ip_forward=1 net.ipv4.conf.all.rp_filter=2
```

- Subnet public là một **bridge**, vì ở bước 5 NAT Gateway sẽ cắm thêm vào đây.
- `ip_forward=1` biến namespace thành router.
- `rp_filter=2` (chế độ lỏng) cần vì bài có đường đi vòng qua NAT Gateway: gói trả về đi vào router
  từ subnet public, trong khi địa chỉ nguồn của nó nằm phía Internet. Chế độ chặt sẽ vứt gói đó.

**Main route table — chỉ có `local`:**

```bash
sudo ip -n vpcr route add 10.99.1.0/24 dev br-pub table 100
sudo ip -n vpcr route add 10.99.2.0/24 dev r-b    table 100
sudo ip -n vpcr rule add iif br-pub lookup 100  pref 1000
sudo ip -n vpcr rule add iif r-b    lookup 100  pref 1001
sudo ip -n vpcr rule add iif r-inet lookup main pref 2000
sudo ip -n vpcr rule add iif lo     lookup main pref 2001
sudo ip -n vpcr rule del pref 32766
```

Đọc từng dòng:

- Bảng `100` đóng vai **main route table** của VPC: chỉ hai dòng, tương đương dòng `local`. Trên
  AWS, dòng `local` do AWS tự thêm vào mọi route table và không xoá được; ở lab phải gõ tay, và
  bước 4, 5 cũng phải chép lại hai dòng này vào bảng mới vì cùng lý do.
- `rule iif br-pub lookup 100` nghĩa là "packet đi ra từ subnet public thì tra bảng 100" — đây chính
  là **association**. Hai subnet chưa được gắn bảng riêng nên cùng rơi vào bảng main.
- Luật `32766` mặc định của Linux là "mọi packet tra bảng `main`". Xoá nó để packet từ subnet không
  khớp bảng của mình thì **dừng**, không lén rơi xuống bảng khác — đúng như VPC.
- Hai luật `2000`, `2001` dành cho traffic đến từ phía Internet và của chính router.

**Bật dịch vụ web trên các máy:**

```bash
sudo ip netns exec ec2a python3 -m http.server 8080 --directory /tmp >/dev/null 2>&1 &
sudo ip netns exec ec2b python3 -m http.server 8080 --directory /tmp >/dev/null 2>&1 &
sudo ip netns exec ec2b python3 -m http.server 9000 --directory /tmp >/dev/null 2>&1 &
sleep 1
```

Ở **terminal 2**, bật "Internet" và để nó chạy. Mỗi request tới sẽ in một dòng log kèm IP nguồn:

```bash
sudo ip netns exec inet python3 -m http.server 80 --bind 198.51.100.10 --directory /tmp
```

## Bước 1 — Hệ điều hành chỉ biết `.1`

**Dự đoán:** bảng route bên trong `ec2a` có mấy dòng? Có dòng nào nói về Internet, NAT hay IGW không?

```bash
sudo ip -n ec2a route
```

**Quan sát:** chỉ có `default via 10.99.1.1` và dòng của subnet mình. Máy không biết gì về NAT hay
IGW — nó đưa mọi thứ ngoài subnet cho `.1`. Mọi quyết định tiếp theo nằm ở router. Đây là "tầng 1"
trong [aws-architecture.md](aws-architecture.md) mục 4.3.

## Bước 2 — Main route table chỉ có `local`

**Dự đoán:** `ec2a` gọi Internet được không? Nếu không, packet chết ở đâu, và terminal 2 có thấy gì?

```bash
hit ec2a http://198.51.100.10/
sudo ip -n vpcr route get 198.51.100.10 from 10.99.1.10 iif br-pub
```

**Quan sát:** `000`, router trả `Network is unreachable`, terminal 2 im lặng. Packet tới được `.1`
nhưng bảng 100 không có dòng nào khớp nên bị vứt ngay tại router.

`ip route get … iif …` là cách hỏi thẳng router: "một packet như thế này, đi vào từ interface này,
thì được chuyển đi đâu?" Dùng nó mỗi khi nghi ngờ routing.

## Bước 3 — Hai subnet gọi nhau không cần thêm route

**Dự đoán:** chưa có dòng route nào "tới `ec2b`". `ec2a` gọi `ec2b` được không? Chiều về tra bảng nào?

```bash
hit ec2a http://10.99.2.10:8080/
hit ec2b http://10.99.1.10:8080/
sudo ip -n vpcr route get 10.99.2.10 from 10.99.1.10 iif br-pub
sudo ip -n vpcr route get 10.99.1.10 from 10.99.2.10 iif r-b
```

**Quan sát:** cả hai chiều `200`. Chiều đi ra `dev r-b`, tra theo `iif br-pub`; chiều về ra
`dev br-pub`, tra theo `iif r-b`. **Mỗi chiều tra bảng của subnet nơi packet rời đi**, và dòng
`local` có sẵn trong bảng nên không cần thêm gì.

### Thử phá: xoá dòng `local`

**Dự đoán:** xoá dòng dẫn tới subnet B khỏi bảng 100. `ec2a` còn gọi được `ec2b` không?

```bash
sudo ip -n vpcr route del 10.99.2.0/24 dev r-b table 100
hit ec2a http://10.99.2.10:8080/
```

**Quan sát:** `000` — bảng của subnet A không còn đường tới subnet B. Trên AWS kịch bản này
**không thể xảy ra**, vì AWS không cho xoá dòng `local`. Đó là lý do trong một VPC, các subnet
luôn thấy nhau mà bạn không phải làm gì.

Trả lại như cũ:

```bash
sudo ip -n vpcr route add 10.99.2.0/24 dev r-b table 100
hit ec2a http://10.99.2.10:8080/
```

## Bước 4 — Biến subnet A thành public

Trên AWS, subnet thành public khi được gắn một route table có `0.0.0.0/0 → igw`, và máy muốn ra
Internet phải có public IP hoặc Elastic IP.

**Tự làm:** tạo bảng `101` đóng vai route table public.

1. Bảng cần những dòng nào? Gợi ý: hai dòng `local` giống bảng 100, cộng một dòng mặc định đi ra
   `r-inet` qua `198.51.100.10`.
2. Associate subnet public với bảng 101 bằng cách thay luật `pref 1000`.

<details>
<summary>Lời giải</summary>

```bash
sudo ip -n vpcr route add 10.99.1.0/24 dev br-pub table 101
sudo ip -n vpcr route add 10.99.2.0/24 dev r-b    table 101
sudo ip -n vpcr route add default via 198.51.100.10 dev r-inet onlink table 101
sudo ip -n vpcr rule del pref 1000
sudo ip -n vpcr rule add iif br-pub lookup 101 pref 1000
```

`onlink` báo cho kernel rằng next-hop nằm ngay trên dây `r-inet`, không cần tra thêm để tìm nó.

</details>

**Gắn Elastic IP cho `ec2a`** — Internet Gateway ánh xạ 1:1 `198.51.100.101 ↔ 10.99.1.10`:

```bash
sudo ip -n vpcr addr add 198.51.100.101/32 dev r-inet
sudo ip netns exec vpcr nft -f - <<'EOF'
table ip igw {
  chain pre  { type nat hook prerouting  priority -100; }
  chain post { type nat hook postrouting priority 100; }
}
add rule ip igw pre  ip daddr 198.51.100.101 dnat to 10.99.1.10
add rule ip igw post ip saddr 10.99.1.10 oifname "r-inet" snat to 198.51.100.101
EOF
```

**Dự đoán — trả lời cả bốn rồi mới chạy:**

1. `ec2a` ra Internet được chưa? Terminal 2 thấy IP nguồn nào?
2. Internet gọi ngược vào `ec2a` qua `198.51.100.101` được không?
3. `ip addr` bên trong `ec2a` có hiện `198.51.100.101` không?
4. `ec2b` ra Internet được chưa?

```bash
hit ec2a http://198.51.100.10/
hit inet http://198.51.100.101:8080/
sudo ip -n ec2a addr show a-eth0
hit ec2b http://198.51.100.10/
```

**Quan sát:**

1. `200`, terminal 2 thấy `198.51.100.101` — Internet chỉ thấy Elastic IP.
2. `200` — ánh xạ 1:1 mở cả hai chiều.
3. **Không.** Hệ điều hành không bao giờ biết public IP của chính nó; việc dịch xảy ra bên ngoài máy
   ([network-to-vpc.md](network-to-vpc.md) mục 3.4).
4. `000` — subnet private vẫn dùng bảng 100.

## Bước 5 — Subnet B là private, đi ra qua NAT Gateway

**Tạo NAT Gateway và cắm nó vào subnet public:**

```bash
sudo ip netns add natgw
sudo ip link add n-eth0 netns natgw type veth peer name r-nat netns vpcr
sudo ip -n vpcr link set r-nat master br-pub && sudo ip -n vpcr link set r-nat up
sudo ip -n natgw link set lo up
sudo ip -n natgw addr add 10.99.1.20/24 dev n-eth0 && sudo ip -n natgw link set n-eth0 up
sudo ip -n natgw route add default via 10.99.1.1
sudo ip netns exec natgw sysctl -qw net.ipv4.ip_forward=1
sudo ip netns exec natgw nft -f - <<'EOF'
table ip nat {
  chain post {
    type nat hook postrouting priority 100;
    ip saddr 10.99.2.0/24 oifname "n-eth0" masquerade
  }
}
EOF
```

`masquerade` là PAT: đổi địa chỉ nguồn của mọi packet từ subnet private thành IP của chính NAT
Gateway, và ghi nhớ từng kết nối để trả gói về đúng máy.

**Gắn Elastic IP cho NAT Gateway** — y như bước 4, chỉ khác cặp địa chỉ:

```bash
sudo ip -n vpcr addr add 198.51.100.102/32 dev r-inet
sudo ip netns exec vpcr nft -f - <<'EOF'
add rule ip igw pre  ip daddr 198.51.100.102 dnat to 10.99.1.20
add rule ip igw post ip saddr 10.99.1.20 oifname "r-inet" snat to 198.51.100.102
EOF
```

**Tự làm:** tạo bảng `102` đóng vai route table private, rồi associate subnet private (`pref 1001`)
với nó. So với bảng 101, bảng này khác đúng một chỗ: dòng mặc định đi đâu?

<details>
<summary>Lời giải</summary>

```bash
sudo ip -n vpcr route add 10.99.1.0/24 dev br-pub table 102
sudo ip -n vpcr route add 10.99.2.0/24 dev r-b    table 102
sudo ip -n vpcr route add default via 10.99.1.20 dev br-pub table 102
sudo ip -n vpcr rule del pref 1001
sudo ip -n vpcr rule add iif r-b lookup 102 pref 1001
```

Bảng 101 trỏ dòng mặc định ra cửa Internet, bảng 102 trỏ vào NAT Gateway. **Target khác nhau nên
phải là hai bảng** — đúng lý do ở [aws-architecture.md](aws-architecture.md) mục 4.5.

</details>

**Dự đoán:**

1. `ec2b` ra Internet được chưa? Terminal 2 thấy IP nguồn nào — `10.99.2.10`, `10.99.1.20` hay
   `198.51.100.102`?
2. Internet có cách nào gọi vào `ec2b` không?

```bash
hit ec2b http://198.51.100.10/
sudo ip -n vpcr route get 198.51.100.10 from 10.99.2.10 iif r-b
hit inet http://198.51.100.102:8080/
```

**Quan sát:**

1. `200`, terminal 2 thấy `198.51.100.102`. Có hai lần đổi địa chỉ: NAT Gateway đổi `10.99.2.10`
   thành IP của nó (`10.99.1.20`), rồi Internet Gateway đổi tiếp thành Elastic IP.
2. Không. Internet chỉ biết `198.51.100.102`, và NAT Gateway không có luật nào đưa kết nối mới vào
   trong — `000`.

### Thử phá: NAT Gateway phụ thuộc route table của subnet chứa nó

**Dự đoán:** cho subnet public quay về bảng 100 (không có dòng ra Internet). `ec2b` ở subnet khác
và vẫn dùng bảng 102 — nó còn ra Internet được không?

```bash
sudo ip -n vpcr rule del pref 1000
sudo ip -n vpcr rule add iif br-pub lookup 100 pref 1000
hit ec2b http://198.51.100.10/
```

**Quan sát:** `000`. Packet của `ec2b` vẫn tới được NAT Gateway, nhưng NAT Gateway nằm trong một
subnet không có đường ra. Đây là lý do NAT Gateway chế độ zonal **phải** đặt trong public subnet. Chế độ regional tránh được bẫy này
bằng một route table riêng có sẵn đường ra IGW.

Trả lại như cũ:

```bash
sudo ip -n vpcr rule del pref 1000
sudo ip -n vpcr rule add iif br-pub lookup 101 pref 1000
hit ec2b http://198.51.100.10/
```

### Thử phá: NAT Gateway chết, route table vẫn còn

Mô phỏng AZ chứa NAT Gateway bị sập bằng cách rút dây của `natgw` khỏi subnet public. Bảng 102
**không bị đụng tới**.

**Dự đoán:** bảng 102 vẫn còn nguyên dòng `default via 10.99.1.20`. `ec2b` còn ra Internet được không?

```bash
sudo ip -n vpcr link set r-nat down
hit ec2b http://198.51.100.10/
sudo ip -n vpcr route show table 102
```

**Quan sát:** `000`, trong khi bảng 102 vẫn in `default via 10.99.1.20 dev br-pub`. Route table sống
sót — và chính vì sống sót, nó tiếp tục gửi traffic vào một NAT đã chết; không có gì tự đổi hướng.
Giả sử có thêm NAT thứ hai ở AZ khác mà không bảng nào trỏ tới, nó cũng không cứu được `ec2b`. Đây là
lý do mỗi AZ cần một route table private riêng, trỏ về NAT của chính AZ đó
([aws-architecture.md](aws-architecture.md) mục 4.5).

Trả lại như cũ:

```bash
sudo ip -n vpcr link set r-nat up
sleep 1
hit ec2b http://198.51.100.10/
```

## Bước 6 — Dòng cụ thể nhất thắng

**Dự đoán:** thêm vào bảng 102 một dòng vứt mọi packet tới đúng `198.51.100.10/32`, trong khi dòng
`default` vẫn còn. `ec2b` còn gọi được không?

```bash
sudo ip -n vpcr route add blackhole 198.51.100.10/32 table 102
hit ec2b http://198.51.100.10/
sudo ip -n vpcr route show table 102
```

**Quan sát:** `000`. Cả `/32` lẫn `/0` đều khớp, `/32` dài hơn nên thắng (longest prefix match).

```bash
sudo ip -n vpcr route del blackhole 198.51.100.10/32 table 102
hit ec2b http://198.51.100.10/
```

## Bước 7 — NACL không nhớ trạng thái: chiều về chết

Kiểm tra trước khi có NACL:

```bash
hit ec2a http://10.99.2.10:9000/
```

Dựng một "NACL" cho chiều đi vào subnet private: chỉ cho TCP 8080, chặn hết phần còn lại.

```bash
sudo ip netns exec vpcr nft -f - <<'EOF'
table inet nacl {
  chain vao_b {
    type filter hook forward priority 0; policy accept;
    oifname "r-b" tcp dport 8080 counter accept
    oifname "r-b" counter drop
  }
}
EOF
```

**Dự đoán:**

1. `ec2a` gọi `ec2b:8080` được không? Gọi `ec2b:9000` thì sao?
2. `ec2b` tự gọi ra Internet thì sao? Nhớ rằng luật chỉ lọc chiều *đi vào* subnet B.

```bash
hit ec2a http://10.99.2.10:8080/
hit ec2a http://10.99.2.10:9000/
hit ec2b http://198.51.100.10/
sudo ip netns exec vpcr nft list chain inet nacl vao_b
```

**Quan sát:**

1. `8080` → `200`. `9000` → `000` kèm `timed out`: gói bị vứt im lặng, không phải bị từ chối.
2. `000` kèm `timed out`, terminal 2 im lặng, và bộ đếm của dòng drop tăng lên.

Terminal 2 im lặng **không** có nghĩa là gói chưa tới Internet — chỉ là request HTTP chưa kịp được
gửi. Nhìn xuống tầng TCP ngay tại "Internet":

```bash
sudo ip netns exec inet timeout 6 tcpdump -lni i-eth0 -c 6 tcp port 80 & TD=$!
sleep 1; hit ec2b http://198.51.100.10/; wait $TD
```

`Flags [S]` là SYN của `ec2b` (đã qua NAT, nên nguồn là `198.51.100.102`) tới Internet. `Flags [S.]`
là SYN-ACK Internet trả lời. Cả hai lặp đi lặp lại: SYN-ACK quay về, đi vào subnet B tới một **cổng
ngẫu nhiên** (ephemeral port) của `ec2b` — không phải 8080 — nên bị dòng drop chặn, `ec2b` không
bao giờ nhận được, và gửi lại SYN. Bắt tay TCP không bao giờ xong nên HTTP chưa từng được gửi.
Chiều đi thông, chiều về chết.

**Sửa bằng cách mở dải ephemeral port:**

```bash
sudo ip netns exec vpcr nft -f - <<'EOF'
flush chain inet nacl vao_b
add rule inet nacl vao_b oifname "r-b" tcp dport 8080 counter accept
add rule inet nacl vao_b oifname "r-b" tcp dport 1024-65535 counter accept
add rule inet nacl vao_b oifname "r-b" counter drop
EOF
```

**Dự đoán:** `ec2b` ra Internet được lại chưa? Còn `ec2a` gọi `ec2b:9000` thì sao?

```bash
hit ec2b http://198.51.100.10/
hit ec2a http://10.99.2.10:9000/
```

**Quan sát:** cả hai `200`. Mở dải ephemeral port để cứu chiều về thì cũng **vô tình mở luôn mọi
dịch vụ chạy ở cổng cao** như 9000. Đó là cái giá của một firewall không nhớ trạng thái.

## Bước 8 — Security Group nhớ trạng thái

Thay dòng mở dải cổng bằng dòng "cho qua mọi gói thuộc một kết nối đã tồn tại":

```bash
sudo ip netns exec vpcr nft -f - <<'EOF'
flush chain inet nacl vao_b
add rule inet nacl vao_b oifname "r-b" tcp dport 8080 counter accept
add rule inet nacl vao_b oifname "r-b" ct state established,related counter accept
add rule inet nacl vao_b oifname "r-b" counter drop
EOF
```

**Dự đoán:** ba lệnh dưới đây ra gì?

```bash
hit ec2b http://198.51.100.10/
hit ec2a http://10.99.2.10:8080/
hit ec2a http://10.99.2.10:9000/
```

**Quan sát:** `200`, `200`, `000`. Gói trả lời được nhận ra là thuộc kết nối do `ec2b` mở, nên tự
được qua mà không cần mở dải cổng nào — và `9000` vẫn đóng. Đó là "stateful" của Security Group.

Tuỳ chọn — xem tận mắt "trí nhớ" của NAT Gateway (cần cài gói `conntrack`):

```sh
sudo apt install conntrack
sudo ip netns exec natgw conntrack -L
```

## Dọn dẹp

Dừng terminal 2 bằng `Ctrl+C`, rồi ở terminal 1:

```bash
for n in ec2a ec2b inet natgw vpcr; do sudo ip netns pids $n | xargs -r sudo kill; done
sleep 1
for n in ec2a ec2b inet natgw vpcr; do sudo ip netns del $n; done
sudo ip netns list
```

Lệnh cuối không in gì là đã sạch. Xoá namespace thì veth, bridge, bảng route và luật nftables bên
trong biến mất theo.

## Câu hỏi sau lab

Trả lời không nhìn lại bài:

1. Ở bước 3, vì sao `ec2a` gọi được `ec2b` dù chưa ai thêm route "tới `ec2b`"?
2. Bảng 101 và 102 khác nhau đúng ở dòng nào, và vì sao khác biệt đó buộc phải có hai bảng?
3. Ở phần thử phá của bước 5, `ec2b` không hề đổi bảng — vậy vì sao nó mất Internet?
4. Ở bước 7, vì sao chiều về chết, và vì sao cách sửa bằng dải ephemeral port lại mở thêm lỗ hổng?
5. Trên AWS, lệnh `ip rule add iif br-pub lookup 101 pref 1000` tương ứng với resource Terraform nào?
6. Ở bước 2, lab báo lỗi ngay còn AWS để ứng dụng chờ tới timeout. Khi debug trên AWS, điều đó
   khiến bạn dễ nhầm lỗi routing với loại lỗi nào?
7. Đóng vai kẻ tấn công đã chiếm được `ec2a`. Dùng kết quả của bước 3, bước 5 và bước 8, giải
   thích: vì sao subnet private chặn được Internet nhưng không chặn được bạn, và thứ gì mới thật
   sự chặn được bạn?
