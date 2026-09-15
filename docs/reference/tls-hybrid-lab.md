# Thực hành mã hóa lai của TLS bằng Go

Bài này chạy tận tay ba bước ở [networking.md](networking.md) mục 5.2: client sinh khóa và bọc bằng
RSA, server mở gói, rồi hai bên nói chuyện bằng AES-GCM. Một "kẻ nghe lén" ngồi xem mọi byte trên
dây. Mất khoảng 45 phút, chỉ dùng thư viện chuẩn của Go.

## Trước khi bắt đầu

**Cách học:** mỗi phần có **Dự đoán**. Viết câu trả lời ra giấy *trước* khi chạy.

**Cần:** Go 1.21 trở lên cho demo chính, Go 1.24 trở lên cho phần tự làm cuối bài (`crypto/hkdf`),
`tcpdump`, quyền `sudo`, và **ba terminal**.

### Demo khác TLS thật ở đâu

Demo cố tình bỏ bớt để thấy rõ xương sống. Đừng mang nguyên đoạn code này đi dùng thật.

| Trong demo | Trong TLS thật |
|---|---|
| Server gửi Public Key trần, client tin luôn | Public Key nằm trong chứng thư, client kiểm tra chữ ký CA và tên miền trước (networking.md mục 5.3, bước 6–7) |
| Client sinh thẳng Session Key 32 byte | TLS 1.2 sinh Pre-Master Secret rồi trộn với `Client Random`, `Server Random`; TLS 1.3 dùng ECDHE |
| Một khóa dùng cho cả hai chiều | Mỗi chiều truyền một khóa riêng |
| Nonce ngẫu nhiên cho mỗi bản tin | Nonce dẫn xuất từ số thứ tự bản ghi |
| Tự chế khung: 4 byte độ dài + dữ liệu | Định dạng TLS record |
| Không có bản tin `Finished` | `Finished` kiểm tra toàn bộ quá trình bắt tay không bị sửa |

## Bước 1 — Tạo chương trình

```bash
mkdir -p /tmp/tls-demo && cd /tmp/tls-demo
go mod init tlsdemo
cat > main.go <<'EOF'
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const addr = "127.0.0.1:9443"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("dùng: go run . server   hoặc   go run . client")
	}
	if os.Args[1] == "server" {
		runServer()
	} else {
		runClient()
	}
}

func runServer() {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	must(err)
	pub, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	must(err)

	ln, err := net.Listen("tcp", addr)
	must(err)
	fmt.Println("[server] đang nghe", addr)
	conn, err := ln.Accept()
	must(err)
	defer conn.Close()

	fmt.Println("[server] Bước 0: gửi Public Key, Private Key giữ lại trong máy")
	send(conn, pub)

	wrapped := recv(conn)
	sessionKey, err := rsa.DecryptOAEP(sha256.New(), nil, priv, wrapped, nil)
	must(err)
	fmt.Println("[server] Bước 2: mở gói bằng Private Key → Session Key", short(sessionKey))

	req, err := open(sessionKey, recv(conn))
	must(err)
	fmt.Printf("[server] Bước 3: giải mã request → %q\n", req)

	body := []byte(`{"don_hang":123,"trang_thai":"dang_giao"}`)
	fmt.Printf("[server] Bước 3: mã hoá response %s bằng AES-GCM rồi gửi\n", body)
	send(conn, seal(sessionKey, body))
}

func runClient() {
	conn, err := net.Dial("tcp", addr)
	must(err)
	defer conn.Close()

	pubAny, err := x509.ParsePKIXPublicKey(recv(conn))
	must(err)
	serverPub := pubAny.(*rsa.PublicKey)
	fmt.Println("[máy 1] Bước 0: nhận Public Key của server")

	sessionKey := make([]byte, 32)
	rand.Read(sessionKey)
	fmt.Println("[máy 1] Bước 1: tự sinh Session Key AES-256 ngẫu nhiên →", short(sessionKey))

	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, serverPub, sessionKey, nil)
	must(err)
	fmt.Println("[máy 1] Bước 1: bọc Session Key bằng Public Key của server rồi gửi")
	send(conn, wrapped)

	req := []byte("GET /don-hang/123")
	fmt.Printf("[máy 1] Bước 3: mã hoá request %q bằng Session Key rồi gửi\n", req)
	send(conn, seal(sessionKey, req))

	resp, err := open(sessionKey, recv(conn))
	must(err)
	fmt.Printf("[máy 1] Bước 3: giải mã response → %s\n", resp)
}

func seal(key, plaintext []byte) []byte {
	gcm := newGCM(key)
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	return gcm.Seal(nonce, nonce, plaintext, nil)
}

func open(key, msg []byte) ([]byte, error) {
	gcm := newGCM(key)
	nonce, ciphertext := msg[:gcm.NonceSize()], msg[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func newGCM(key []byte) cipher.AEAD {
	block, err := aes.NewCipher(key)
	must(err)
	gcm, err := cipher.NewGCM(block)
	must(err)
	return gcm
}

func send(conn net.Conn, b []byte) {
	fmt.Printf("         [trên dây] %d byte: %s…\n", len(b), hex.EncodeToString(b[:min(len(b), 20)]))
	must(binary.Write(conn, binary.BigEndian, uint32(len(b))))
	_, err := conn.Write(b)
	must(err)
}

func recv(conn net.Conn) []byte {
	var n uint32
	must(binary.Read(conn, binary.BigEndian, &n))
	b := make([]byte, n)
	_, err := io.ReadFull(conn, b)
	must(err)
	return b
}

func short(b []byte) string { return hex.EncodeToString(b[:8]) + "…" }

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
EOF
go vet .
```

Đọc code theo đúng các bước ở networking.md mục 5.2:

- **Bước 0** — `runServer` sinh cặp khóa RSA 2048 và gửi phần công khai. Private Key không bao giờ rời hàm.
- **Bước 1** — `runClient` lấy 32 byte từ `crypto/rand`, bọc bằng `rsa.EncryptOAEP`.
- **Bước 2** — server mở bằng `rsa.DecryptOAEP`.
- **Bước 3** — `seal` / `open`: mỗi bản tin là `nonce ‖ ciphertext ‖ tag`. `gcm.Seal(nonce, nonce, …)`
  nối ciphertext và tag vào ngay sau nonce.
- `send` / `recv` tự chế khung "4 byte độ dài + dữ liệu", vì TCP là một dòng byte liền, không biết
  bản tin bắt đầu và kết thúc ở đâu.

## Bước 2 — Bật server và kẻ nghe lén

**Terminal 1** — server:

```bash
cd /tmp/tls-demo && go run . server
```

**Terminal 3** — kẻ nghe lén, xem mọi byte đi qua cổng 9443:

```bash
sudo tcpdump -i lo -nn -X 'tcp port 9443'
```

**Dự đoán, trước khi sang bước 3:**

1. Terminal 3 có hiện chuỗi `GET /don-hang/123` hay `dang_giao` không?
2. Session Key 32 byte, vậy khối bọc khóa trên dây dài bao nhiêu byte?
3. Request `GET /don-hang/123` dài 17 byte. Trên dây nó dài bao nhiêu?

## Bước 3 — Máy 1 gửi request

**Terminal 2** — máy 1:

```bash
cd /tmp/tls-demo && go run . client
```

**Quan sát** — các chuỗi hex đổi sau mỗi lần chạy, còn cấu trúc thì giữ nguyên:

```text
[máy 1] Bước 0: nhận Public Key của server
[máy 1] Bước 1: tự sinh Session Key AES-256 ngẫu nhiên → f5199089b00b000f…
[máy 1] Bước 1: bọc Session Key bằng Public Key của server rồi gửi
         [trên dây] 256 byte: 43518bc4ec68f22493c0b5bff70d0dc46c9ad835…
[máy 1] Bước 3: mã hoá request "GET /don-hang/123" bằng Session Key rồi gửi
         [trên dây] 45 byte: f49e4a08e0ce73af935a5e4080124cee0065a62a…
[máy 1] Bước 3: giải mã response → {"don_hang":123,"trang_thai":"dang_giao"}
```

```text
[server] đang nghe 127.0.0.1:9443
[server] Bước 0: gửi Public Key, Private Key giữ lại trong máy
         [trên dây] 294 byte: 30820122300d06092a864886f70d010101050003…
[server] Bước 2: mở gói bằng Private Key → Session Key f5199089b00b000f…
[server] Bước 3: giải mã request → "GET /don-hang/123"
[server] Bước 3: mã hoá response {"don_hang":123,"trang_thai":"dang_giao"} bằng AES-GCM rồi gửi
         [trên dây] 69 byte: b58a766d038ad5140a11c0fb6a744894a95f1d39…
```

1. **Terminal 3 không có chuỗi nào đọc được.** Các gói mang dữ liệu lần lượt có `length 4, 294, 4,
   256, 4, 45, 4, 69` — mỗi khung đi sau 4 byte độ dài. Phần ASCII bên phải của `tcpdump` chỉ là dấu
   chấm và ký tự lộn xộn.
2. **Khối bọc khóa dài 256 byte**, dù Session Key chỉ 32 byte — RSA-OAEP luôn ra đúng độ dài của khóa
   RSA (2048 bit = 256 byte).
3. **Request 17 byte thành 45 byte** = 12 byte nonce + 17 byte ciphertext + 16 byte tag. Response 41 byte
   thành 69 byte theo đúng công thức đó.
4. Hai terminal in **cùng một Session Key**, dù nó chưa từng đi trên dây ở dạng rõ.

Server tự thoát sau một kết nối. Muốn chạy lại thì bật lại terminal 1 trước.

## Thử phá 1 — Sửa 1 byte trên đường đi

Giả lập kẻ đứng giữa lật một bit trong response. Trong `runClient`, thay dòng:

```go
	resp, err := open(sessionKey, recv(conn))
```

bằng:

```go
	msg := recv(conn)
	msg[len(msg)-1] ^= 1
	resp, err := open(sessionKey, msg)
```

**Dự đoán:** client in ra response bị sai một ký tự, hay chuyện gì khác?

Bật lại server ở terminal 1, rồi chạy client ở terminal 2:

```bash
cd /tmp/tls-demo && go run . client
```

**Quan sát:** client dừng với lỗi `cipher: message authentication failed`. GCM không trả về dữ liệu
sai — nó từ chối hẳn, vì tag không còn khớp. Mã hóa không có phần kiểm tra này (như AES-CBC trần) sẽ
âm thầm trả về dữ liệu rác, và ứng dụng có thể xử lý tiếp trên dữ liệu đó.

Trả code về như cũ trước khi làm tiếp.

## Thử phá 2 — Gửi cùng một request hai lần

**Dự đoán:** chạy client hai lần với cùng request `GET /don-hang/123`. Chuỗi hex của khối 45 byte hai
lần có giống nhau không?

Chạy server rồi client, lặp lại một lần nữa, và so hai dòng `[trên dây] 45 byte`.

**Quan sát:** khác hoàn toàn. Có hai lý do chồng lên nhau: mỗi lần chạy sinh một Session Key mới, và
mỗi bản tin dùng một nonce ngẫu nhiên mới. Nếu cùng khóa **và** cùng nonce thì cùng plaintext sẽ ra
cùng ciphertext — kẻ nghe lén chỉ cần so gói là biết hai request giống nhau, dù không đọc được nội dung.

## Tự làm — Nâng lên kiểu TLS 1.3 (ECDHE)

**Mục tiêu:** bỏ hẳn RSA. Mỗi bên sinh một cặp khóa tạm, trao phần công khai cho nhau, tự tính ra
cùng một bí mật chung, rồi dẫn xuất khóa AES từ bí mật đó. Hàm `seal`, `open`, `send`, `recv` giữ
nguyên. Cơ chế nằm ở networking.md mục 5.2, phần "Vì sao TLS 1.3 bỏ cách bọc khóa bằng RSA".

**Gợi ý 1 — luồng:** server gửi public key tạm của nó trước; client nhận, rồi gửi public key tạm của
mình. Không bên nào gửi bí mật chung.

**Gợi ý 2 — thư viện chuẩn cần dùng:**

- `crypto/ecdh`: `ecdh.X25519().GenerateKey(rand.Reader)` sinh cặp khóa tạm;
  `priv.PublicKey().Bytes()` lấy phần công khai để gửi; `ecdh.X25519().NewPublicKey(b)` dựng lại
  public key của bên kia từ byte nhận được; `priv.ECDH(peerPub)` tính bí mật chung.
- `crypto/hkdf` (Go 1.24+): `hkdf.Key(sha256.New, secret, nil, "tls-demo", 32)` biến bí mật chung
  thành khóa AES-256.

**Đạt khi:**

- Hai terminal in cùng 8 byte đầu của khóa.
- Terminal 3 không còn khối 256 byte nào; thay vào đó là hai khối 32 byte (public key tạm X25519).
- Request và response vẫn dài 45 và 69 byte.

<details>
<summary>Gợi ý cuối — hàm dẫn xuất khóa</summary>

```go
func derive(priv *ecdh.PrivateKey, peer *ecdh.PublicKey) []byte {
	secret, err := priv.ECDH(peer)
	must(err)
	key, err := hkdf.Key(sha256.New, secret, nil, "tls-demo", 32)
	must(err)
	return key
}
```

</details>

Bản bạn vừa viết vẫn chưa an toàn: server chưa **ký** gì cả, nên kẻ đứng giữa vẫn tráo được public key
tạm. TLS 1.3 thật dùng Private Key trong chứng thư để ký vào bản tin bắt tay.

## Dọn dẹp

Dừng `tcpdump` ở terminal 3 bằng `Ctrl+C`, rồi:

```bash
rm -rf /tmp/tls-demo
```

## Câu hỏi sau lab

Trả lời không nhìn lại bài:

1. Session Key chỉ 32 byte, vì sao khối bọc khóa luôn dài 256 byte?
2. Request 17 byte thành 45 byte trên dây. 28 byte thêm vào là gì? Request dài 1.000 byte thì trên dây
   dài bao nhiêu?
3. Demo bỏ qua việc kiểm tra Public Key. Một kẻ ngồi giữa máy 1 và server sẽ làm gì, cụ thể từng bước,
   để đọc được request?
4. Giả sử server dùng lại cùng một cặp khóa RSA suốt nhiều năm như server thật, và bạn đã lưu lại toàn
   bộ gói của bản RSA. Hôm nay Private Key bị lộ: bạn mở được các gói cũ không? Với bản ECDHE thì sao?
5. Trong demo, phép toán nào tốn CPU nhất, và nó chạy ở máy nào? Điều đó nói gì về nginx của repo này
   khi có nhiều kết nối HTTPS mới cùng lúc?
