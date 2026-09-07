# Quá trình build image — Podman/Buildah chi tiết

Tài liệu này giữ toàn bộ phần giải thích đã được gỡ ra khỏi 6 `Dockerfile`, cộng thêm
mô tả chi tiết cách Podman thực sự build một image và vì sao nó cache được.

Sơ đồ trực quan kèm theo: **`docs/diagrams/build-process.drawio`** (mở bằng draw.io / diagrams.net,
hoặc extension "Draw.io Integration" trong VS Code). Đó là **một sơ đồ liền mạch duy nhất**
— cuộn từ trên xuống là đọc hết, xem mục [§7](#7-bản-đồ-trong-file-drawio).

---

## Mục lục

1. [Kiến trúc: podman build thực chất là gì](#1-kiến-trúc-podman-build-thực-chất-là-gì)
2. [Sáu giai đoạn của một lần build](#2-sáu-giai-đoạn-của-một-lần-build)
3. [Vòng lặp per-instruction và cơ chế cache](#3-vòng-lặp-per-instruction-và-cơ-chế-cache)
4. [Cache mount — loại cache hoàn toàn khác](#4-cache-mount--loại-cache-hoàn-toàn-khác)
5. [Podman/Buildah so với BuildKit](#5-podmanbuildah-so-với-buildkit)
6. [Giải thích từng dòng Dockerfile của dự án](#6-giải-thích-từng-dòng-dockerfile-của-dự-án)
7. [Bản đồ trong file drawio](#7-bản-đồ-trong-file-drawio)
8. [Nguồn tham khảo](#8-nguồn-tham-khảo)

---

## 1. Kiến trúc: podman build thực chất là gì

Điều đầu tiên cần nắm: **Podman không tự build image.** Tài liệu chính thức nói rõ
`podman build` dùng code lấy từ dự án **Buildah**, và code Buildah đó tạo ra các
"Buildah container" cho những lệnh `RUN` trong container storage.

Nói cách khác, `podman build` chỉ là lớp vỏ CLI. Bên dưới là ba thư viện tách bạch:

| Thành phần                    | Vai trò                                                                   |
|-------------------------------|---------------------------------------------------------------------------|
| **buildah** (`imagebuildah`)  | Bộ điều phối build: đọc Containerfile, chạy từng chỉ thị, commit layer    |
| **containers/image**          | Kéo/đẩy/kiểm tra/ký image. Nói chuyện với registry, xử lý manifest        |
| **containers/storage**        | Lưu layer hệ thống file, image, container. Điều khiển driver `overlay`    |

Điểm khác biệt lớn nhất so với Docker: **không có daemon**. Docker cổ điển gửi build
context tới `dockerd` qua socket. Podman chạy build ngay trong tiến trình của chính nó,
với quyền của user đang gọi (rootless được).

Trong repo Buildah, hai file điều phối chính là:

- `imagebuildah/executor.go` — điều phối toàn bộ build, quản lý nhiều stage
- `imagebuildah/stage_executor.go` — chạy **một** stage: `Prepare`, `Execute`, `Run`, `Copy`, `Commit`

---

## 2. Sáu giai đoạn của một lần build

Đây là mô hình chia giai đoạn mà sơ đồ trang 1 trong file drawio vẽ lại.

### Giai đoạn 0 — Phân tích cú pháp

Containerfile được parse thành cây chỉ thị. Song song đó, `.containerignore` (hoặc `.dockerignore`) được đọc để quyết định **file nào thuộc build context**.

> Đây chính là chỗ `.dockerignore` của dự án phát huy tác dụng: nó loại 281MB rác
> (`docs/` 162MB, binary cũ ở gốc repo ~119MB) trước khi bất kỳ byte nào được đọc tới.

### Giai đoạn 1 — Lập kế hoạch stage

Containerfile bị cắt thành các stage theo mỗi lệnh `FROM`. Dockerfile của dự án có
2 stage: stage `builder` và stage `runtime`.

Buildah dựng đồ thị phụ thuộc giữa các stage — `COPY --from=builder` tạo một cạnh từ
stage runtime về stage builder. Cờ `--skip-unused-stages` (**bật mặc định**) sẽ bỏ qua
những stage mà stage đích không cần tới.

### Giai đoạn 2 — Giải tên và kéo base image

`FROM docker.io/library/golang:1.26-alpine` được giải tên qua `registries.conf`.

> **Lưu ý riêng của Podman:** Podman xử lý "short name" khác Docker. Gõ `golang:1.26-alpine`
> trơn, Podman sẽ hỏi anh chọn registry nào (hoặc lỗi nếu chạy không tương tác). Đó là
> lý do Dockerfile của dự án ghi **đầy đủ** `docker.io/library/...` — để chạy được cả
> trên Docker lẫn Podman mà không phụ thuộc cấu hình máy.

Nếu image chưa có trong local storage, `containers/image` kéo manifest, config JSON và
các layer blob về, giải nén vào `containers/storage`.

### Giai đoạn 3 — Tạo working container

Buildah tạo một **working container** từ base image (đúng như lệnh `buildah from`).

Trước khi đọc tiếp, cần hiểu **layer là gì** — vì đây là chỗ dễ hiểu nhầm nhất.

#### Layer không phải bản sao hệ thống file

Hiểu lầm phổ biến: tưởng mỗi layer là một bản chụp đầy đủ của cả hệ thống file.

Thực tế, **layer là bản ghi thay đổi (diff) so với layer nằm ngay dưới nó**, đóng gói
thành một file tar. Nó chỉ chứa: file mới thêm, file bị sửa, và dấu hiệu file bị xoá.
File nào không đụng tới thì không nằm trong layer đó.

Ví von cho dễ hình dung: giống **chồng giấy can** (giấy trong suốt) đặt lên nhau. Mỗi tờ
chỉ vẽ thêm phần của riêng nó, chỗ nào không vẽ thì để trong suốt. Nhìn từ trên xuống,
mắt thấy hình hợp nhất của tất cả các tờ.

Lấy đúng stage 2 của dự án làm ví dụ:

| Layer | Sinh ra bởi | Chứa đúng những gì |
|---|---|---|
| 1 | `FROM alpine:3.24` | toàn bộ alpine tối giản: `/bin/sh`, `/etc/passwd`, `/lib/...` (~8 MB) |
| 2 | `RUN apk add ...` | **chỉ** `/usr/share/ca-certificates/...`, `/usr/share/zoneinfo/...`, và bản mới của `/lib/apk/db/installed` |
| 3 | `COPY ... user_service_bin .` | **đúng một file**: `/app/user_service_bin` (~25 MB) |

Layer 3 không hề chứa `/bin` hay `/etc` — vì nó không đụng tới chúng.

Mỗi layer có một **`diffID`** = sha256 của chính file tar đó. Nội dung giống nhau thì
`diffID` giống nhau, nên hệ thống biết chắc hai layer là một. Đây là nền tảng của cache,
của việc chia sẻ layer giữa nhiều image, và của việc pull chỉ tải phần còn thiếu.

> Hệ quả thực tế: 6 service của dự án đều `FROM alpine:3.24`, nên cả 6 image **dùng chung
> một layer đáy** trên đĩa. 6 × 25 MB **không** tốn 150 MB.

Và tính chất quan trọng nhất: **layer là bất biến**. Đã tạo thì không sửa được nữa, muốn
đổi gì chỉ có cách chồng thêm layer mới lên trên.

#### OverlayFS — cái ghép các layer lại

`containers/storage` tạo một **layer đọc-ghi mới** chồng lên các layer chỉ-đọc của base
image. Với driver `overlay`, đây là một **overlay mount** — một filesystem của nhân Linux
nhận nhiều thư mục rồi trình bày ra thành một thư mục duy nhất:

| Thành phần | Là gì |
|---|---|
| `lowerdir` | **Nhiều** thư mục, **chỉ đọc**, có thứ tự — chính là các layer của image |
| `upperdir` | **Một** thư mục, **ghi được** — mọi thay đổi rơi vào đây |
| `workdir` | Chỗ nháp nhân cần để thao tác nguyên tử. Không bao giờ phải đụng tới |
| `merged` | Thứ container **nhìn thấy**. Chỉ là góc nhìn, không tốn thêm dung lượng |

Lệnh thật mà nhân nhận được có dạng:

```
mount -t overlay overlay \
  -o lowerdir=/l3:/l2:/l1,upperdir=/rw,workdir=/work \
  /merged
```

Vì `lowerdir` **chỉ đọc**, nảy sinh hai cơ chế đặc biệt. Hiểu hai cái này là hiểu hết
OverlayFS:

**Đọc** — nhân tìm lần lượt: `upperdir` trước, rồi từng `lowerdir` theo thứ tự. Gặp cái
nào trước thì trả về cái đó, dừng tìm. Đây là lý do layer 2 "đè" được file của layer 1:
file cũ vẫn còn nguyên bên dưới, chỉ là không ai thấy nữa.

**Ghi vào file đang nằm ở `lowerdir`** → **copy-up**. Không sửa tại chỗ được, nên nhân
phải chép nguyên cả file lên `upperdir` rồi mới sửa bản chép đó.

> Hệ quả đắt giá: sửa 1 byte của file 1 GB thì tốn nguyên 1 GB trong layer mới.

**Xoá file đang nằm ở `lowerdir`** → **whiteout**. Không xoá thật được, nên nhân ghi một
dấu che ở `upperdir` (OverlayFS dùng character device 0:0; trong tar của image là file
`.wh.<tên>`). Khi đọc, gặp dấu che thì coi như file không tồn tại — **nhưng file gốc vẫn
nằm nguyên ở layer dưới và vẫn tốn dung lượng**.

Điều này giải thích một chuyện rất thực tế — **vì sao Dockerfile của dự án viết
`apk --no-cache`**:

```dockerfile
# SAI — image KHÔNG nhỏ đi
RUN apk add ca-certificates tzdata
RUN rm -rf /var/cache/apk/*      # chỉ thêm dấu che vài byte; cache vẫn nằm ở layer trước

# ĐÚNG — cache không bao giờ được tạo ra
RUN apk --no-cache add ca-certificates tzdata
```

Nguyên tắc chung: **dọn dẹp phải nằm trong cùng lệnh `RUN` đã tạo ra rác.** Dọn ở lệnh
`RUN` sau là vô ích về mặt dung lượng.

#### Nối lại với việc build

Vậy "tạo working container" thực chất chỉ là: dựng một overlay mount mới, với các layer
của base image làm `lowerdir` và một thư mục rỗng mới làm `upperdir`.

Rồi mỗi lệnh `RUN`/`COPY` ghi vào `upperdir`. Hết một chỉ thị, Buildah "commit" — tức là
đóng gói `upperdir` thành tar để ra layer mới. Layer đó nhập vào `lowerdir` cho chỉ thị kế
tiếp, và một `upperdir` rỗng khác được tạo ra. Cứ thế lặp lại.

Sơ đồ trang **5** và **6** trong file drawio vẽ lại toàn bộ phần này.

### Giai đoạn 4 — Vòng lặp chỉ thị

Phần cốt lõi. Mỗi chỉ thị trong stage được xử lý tuần tự — chi tiết ở [§3](#3-vòng-lặp-per-instruction-và-cơ-chế-cache).

### Giai đoạn 5 — Commit image cuối

`commit` ghi image mới từ layer đọc-ghi của container cộng với các layer của base image.

Metadata (`ENV`, `CMD`, `EXPOSE`, `WORKDIR`, `LABEL`) được ghi vào **config JSON** của
image, **không tạo layer hệ thống file**. Đây là lý do các lệnh đó gần như miễn phí về
dung lượng.

Với multi-stage, chỉ image của stage cuối được gắn tag. Các stage trung gian bị bỏ,
trừ khi dùng `--save-stages`.

---

## 3. Vòng lặp per-instruction và cơ chế cache

Với **mỗi** chỉ thị, Buildah làm tuần tự sáu bước sau (sơ đồ trang 2 trong file drawio):

```
┌─ 1. Dựng chuỗi "createdBy" ─────────────────────────────────┐
│    Biểu diễn chuẩn hoá của chỉ thị. Ví dụ:                   │
│    "/bin/sh -c go mod download"                              │
└──────────────────────────────────────────────────────────────┘
                          │
┌─ 2. Nếu là COPY/ADD: băm nội dung ──────────────────────────┐
│    ContentDigester tính digest của ĐÚNG những file được copy │
│    → sửa 1 byte trong source là digest đổi                   │
└──────────────────────────────────────────────────────────────┘
                          │
┌─ 3. generateCacheKey ───────────────────────────────────────┐
│    Lấy history + diffIDs của image cha, kết hợp với          │
│    createdBy ở bước 1 (và digest ở bước 2)                   │
│    → ra cache key                                            │
└──────────────────────────────────────────────────────────────┘
                          │
┌─ 4. Tra cache ──────────────────────────────────────────────┐
│    Tìm image trung gian có history khớp                      │
└──────────────────────────────────────────────────────────────┘
              │                              │
          TRÚNG                          TRƯỢT
              │                              │
┌─────────────▼──────────┐   ┌───────────────▼─────────────────┐
│ 5a. Bỏ qua thực thi    │   │ 5b. Thực thi thật               │
│     Trỏ working        │   │     RUN  → chạy qua OCI runtime │
│     container sang     │   │            (crun/runc)          │
│     image đã cache     │   │     COPY → builder.Add()        │
└────────────────────────┘   └─────────────────────────────────┘
              │                              │
              └──────────────┬───────────────┘
                             │
┌─ 6. Commit layer trung gian (nếu --layers) ─────────────────┐
│    Image này thành cha của chỉ thị kế tiếp                   │
└──────────────────────────────────────────────────────────────┘
```

### Vì sao nó cache được — câu trả lời cốt lõi

Cache hoạt động được nhờ **ba tính chất cộng lại**:

**a. Layer là bất biến và định địa chỉ bằng nội dung.**
Mỗi layer có một `diffID` — hash của nội dung nó. Layer đã tạo thì không bao giờ đổi.

**b. Image là một chuỗi layer có thứ tự, cộng với history.**
History ghi lại `createdBy` của từng bước. Vì vậy, một image trung gian *tự nó mang theo*
bằng chứng đầy đủ về việc nó được tạo ra bởi chuỗi chỉ thị nào.

**c. Chỉ thị là hàm thuần theo (cha + input).**
Cùng một image cha, cùng một chỉ thị, cùng một nội dung file → chắc chắn ra cùng kết quả.

Kết hợp ba điều đó: nếu tìm được một image mà history khớp với chuỗi chỉ thị đã chạy tới
thời điểm này, thì layer của nó **chắc chắn** giống hệt thứ sẽ tạo ra nếu chạy lại. Thế thì
khỏi chạy.

### Đổ dây chuyền

Đây là tính chất quan trọng nhất về mặt thực hành:

> **Một chỉ thị trượt cache thì mọi chỉ thị SAU nó đều trượt theo.**

Lý do nằm ngay ở bước 3: cache key của chỉ thị N chứa history của image cha, tức là kết
quả của chỉ thị N−1. Cha đổi thì key đổi, dù bản thân chỉ thị N không đổi một ký tự nào.

**Đây chính là toàn bộ lý do Dockerfile của dự án tách làm hai lớp COPY:**

```dockerfile
COPY api/go.mod api/go.sum ./api/          # lớp 1 — hiếm đổi
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY user_service/go.mod user_service/go.sum ./user_service/
RUN ... go mod download                     # đắt, nhưng được cache

COPY api/ ./api/                            # lớp 2 — đổi liên tục
COPY pkg/ ./pkg/
COPY user_service/ ./user_service/
RUN ... go build ...
```

Sửa một dòng `.go`: digest ở lớp 2 đổi → `go build` chạy lại. Nhưng `go.mod`/`go.sum`
không đổi → `go mod download` **vẫn trúng cache**, khỏi tải lại toàn bộ thư viện.

Nếu gộp thành một `COPY . .` duy nhất thì mọi lần sửa code đều kéo theo tải lại sạch
toàn bộ dependency.

### Bật/tắt cache

| Lệnh | Mặc định `--layers` |
|---|---|
| `podman build` | **`true`** |
| `buildah bud` | **`false`** |

Ghi đè bằng biến môi trường `BUILDAH_LAYERS`. Tắt hẳn cache cho một lần build: `--no-cache`.

### Cache phân tán qua registry

`--cache-from` / `--cache-to` cho phép chia sẻ cache giữa nhiều máy. **Cả hai bị bỏ qua
nếu không có `--layers`.**

Cách Buildah làm khác Docker/BuildKit ở điểm căn bản: Buildah **kéo thẳng image trung gian
từ remote registry**, thay vì nhúng cache vào trong image như BuildKit. Cách này giống
kaniko, và có ưu điểm là không làm phình image gốc. Còn có `--cache-ttl` để bỏ qua cache
quá cũ (ví dụ `--cache-ttl=1h`).

---

## 4. Cache mount — loại cache hoàn toàn khác

Đây là chỗ dễ nhầm nhất, nên tách riêng ra.

```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/user_service_bin ./cmd
```

`RUN --mount` **không phải** cú pháp Dockerfile cổ điển. Nó gắn một thư mục vào container
**chỉ trong lúc lệnh RUN đó chạy**; chạy xong là tháo ra, và nội dung **không nằm lại
trong layer**.

Buildah hỗ trợ `--mount=type=cache` ổn định từ **v1.29.0** (đi kèm **Podman v4.4**).

### Nó nằm ở đâu

Với Podman, cache mount được lưu **trên máy host**, không nằm trong image storage:

```
$TMPDIR/buildah-cache/          # hoặc /var/tmp/buildah-cache nếu $TMPDIR chưa đặt
$TMPDIR/buildah-cache/<id>/     # khi dùng tuỳ chọn id=
```

Buildah tạo thư mục cache cha **riêng cho từng user** trên host.

### Hai loại cache, đừng lẫn

| | Layer cache | Cache mount |
|---|---|---|
| Lưu ở đâu | containers/storage (image) | thư mục trên host |
| Có vào image không | Có (là layer thật) | **Không bao giờ** |
| Ảnh hưởng cache key | Có | **Không** |
| `--cache-to` đẩy đi được | Có | **Không** |
| Tác dụng | Bỏ qua cả chỉ thị | Chỉ thị vẫn chạy, nhưng chạy nhanh |

Hai cơ chế này **bổ sung** cho nhau chứ không thay thế. Sửa một dòng `.go` là layer `RUN`
bắt buộc chạy lại (layer cache trượt) — nhưng nhờ cache mount, Go thấy `GOCACHE` còn
nguyên nên chỉ biên dịch lại đúng phần đổi thay vì dịch từ số 0.

Hai target là hai loại cache khác nhau của Go:

| Thư mục | Biến Go | Chứa gì | Tiết kiệm |
|---|---|---|---|
| `/go/pkg/mod` | `GOMODCACHE` | mã nguồn thư viện đã tải | băng thông |
| `/root/.cache/go-build` | `GOCACHE` | kết quả biên dịch từng package | CPU |

### Bẫy: đừng ghi output vào cache mount

Binary đi ra `/out/` chứ không phải chỗ khác — **có chủ ý**. `/out` là thư mục thường
trong layer, nên stage 2 mới `COPY --from=builder /out/...` được.

Nếu lỡ ghi vào `/go/pkg/mod`, tháo mount xong là file bay mất, mà lỗi lại nổ tận stage 2
dưới dạng "not found" — rất khó lần ngược ra nguyên nhân.

### Trên CI thì cache mount vô dụng

Cache mount **không** được `cache-to: type=gha` xuất đi — nó nằm trong state cục bộ của
máy build. GitHub Actions cấp runner mới tinh mỗi lần chạy, nên trên CI hai thư mục này
**luôn rỗng**: không hại gì, nhưng cũng không giúp gì.

Thứ tăng tốc CI là **layer cache**, nhờ layer `go mod download` đã tách riêng.

Cache mount phát huy tác dụng khi build **lặp lại trên cùng một máy**: máy anh chạy
`make docker-build`, hoặc self-hosted runner.

---

## 5. Podman/Buildah so với BuildKit

Dự án này chạm cả hai, nên cần phân biệt rõ:

- **CI** dùng `docker/build-push-action` → chạy trên **BuildKit**
- **EC2** dùng `podman-compose` → chạy trên **Buildah** (nhưng từ khi chuyển sang GHCR
  thì EC2 chỉ `pull`, không build nữa)

| | Buildah (podman build) | BuildKit (docker build) |
|---|---|---|
| Mô hình thực thi | Tuần tự, từng chỉ thị một | Đồ thị DAG (LLB) |
| Chạy song song | Không (trong một stage) | Có — các stage độc lập chạy song song |
| Daemon | Không cần | Cần buildkitd (nhúng trong dockerd) |
| Rootless | Có, thiết kế từ đầu | Có, cần cấu hình |
| `--cache-from` | Kéo image trung gian từ registry | Nhập cache blob (inline/registry/gha) |
| Cache mount | Thư mục trên host | Trong state của builder |

Hệ quả thực tế cho dự án: 6 image trong CI build **song song** nhờ `strategy.matrix` của
GitHub Actions, chứ không phải nhờ BuildKit. Nếu có lúc nào chạy `podman build` cho cả 6
service trên cùng một máy thì chúng sẽ chạy lần lượt.

---

## 6. Giải thích từng dòng Dockerfile của dự án

Đây là phần đã gỡ ra khỏi các `Dockerfile`.

### Vì sao build context là gốc repo

```dockerfile
# podman build -f user_service/Dockerfile -t user_service .
#                                                          ↑ context = gốc repo
```

Trong `user_service/go.mod` có:

```
replace github.com/logistic/api => ../api
replace github.com/logistic/pkg => ../pkg
```

Hai đường dẫn `../` đó nằm **ngoài** thư mục service. Nếu context là `./user_service`
thì Docker/Podman không gửi `api/` và `pkg/` cho builder — chúng đơn giản là không tồn
tại, và build chết ở stage 1.

### Stage 1 — builder

```dockerfile
FROM docker.io/library/golang:1.26-alpine AS builder
```
Ghi đầy đủ registry để Podman không phải hỏi short-name (xem [§2](#giai-đoạn-2--giải-tên-và-kéo-base-image)).

```dockerfile
ENV GOWORK=off \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
```

| Biến | Vì sao |
|---|---|
| `GOWORK=off` | Tắt workspace. Sách *Learning Go* (2nd ed., ch.10) nói `go.work` chỉ dành cho máy local. Trong container, mỗi module phải tự đứng bằng `go.mod` + `go.sum` của chính nó → build tái lập được, không dính lỗi `missing go.sum entry` do `go.work.sum` lệch pha. `go.work` cũng đã bị chặn trong `.dockerignore`. |
| `CGO_ENABLED=0` | Binary tĩnh. Alpine dùng musl chứ không phải glibc — tắt cgo thì binary không phụ thuộc libc của hệ thống, chạy được trên alpine trần. |
| `GOOS`/`GOARCH` | Ghim đích Linux/amd64, không phụ thuộc máy build. |

```dockerfile
COPY api/go.mod api/go.sum ./api/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY user_service/go.mod user_service/go.sum ./user_service/
```
Phải copy cả `go.mod` của `api` và `pkg` vì `replace` trỏ tới chúng — Go cần đọc `go.mod`
ở hai chỗ đó để tính phiên bản (MVS), dù chưa cần source.

```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
```
Tách riêng để tận dụng layer cache — xem phần [đổ dây chuyền](#đổ-dây-chuyền).

```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/user_service_bin ./cmd
```

Hai cờ, **đã đo thật** trên một chương trình mẫu:

| Build | Kích thước |
|---|---|
| Không cờ | 2.22 MB |
| `-trimpath -ldflags "-s -w"` | 1.50 MB (**−33%**) |

- **`-trimpath`** bỏ đường dẫn tuyệt đối của máy build khỏi binary. Thấy rõ nhất ở panic
  trace: `C:/Users/trank/.../sw_test/main.go` rút còn `swtest/main.go`. Nhờ vậy build tái
  lập được và không lộ cấu trúc thư mục máy build.

- **`-ldflags "-s -w"`** là cờ cho **linker**: `-s` bỏ symbol table, `-w` bỏ DWARF debug info.
  - **Mất:** không debug được bằng delve/gdb trên binary này.
  - **KHÔNG mất:** panic vẫn in đủ tên hàm **và** số dòng, vì thông tin đó nằm ở `pclntab`
    chứ không phải symbol table. Nhiều người hiểu nhầm chỗ này rồi ngại dùng `-s -w`.

Trace lấy từ chính binary đã strip:

```
panic: runtime error: index out of range [99] with length 3

goroutine 1 [running]:
main.level3(...)
	swtest/main.go:5
main.level2(...)
	swtest/main.go:6
main.main()
	swtest/main.go:10 +0x13
```

### Stage 2 — runtime

```dockerfile
FROM docker.io/library/alpine:3.24
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/user_service_bin .
EXPOSE 9004
CMD ["./user_service_bin"]
```

- `ca-certificates` — cần cho TLS khi gọi ra ngoài (gRPC, Cloudinary, Google OAuth)
- `tzdata` — cần để `time.LoadLocation("Asia/Ho_Chi_Minh")` không lỗi
- `apk --no-cache` — không để lại chỉ mục package trong layer

Toàn bộ Go toolchain (~400MB), mã nguồn và module cache **nằm lại ở stage 1**. Image cuối
chỉ còn alpine (~8MB) + binary tĩnh, khoảng **25MB**.

`EXPOSE` và `CMD` chỉ là metadata trong config JSON, không tạo layer.

---

## 7. Bản đồ trong file drawio

`docs/diagrams/build-process.drawio` là **một sơ đồ liền mạch duy nhất**, không chia trang. Cuộn từ
trên xuống là đọc hết — chín phần, mỗi phần là hệ quả trực tiếp của phần trước, nối
với nhau bằng mũi tên có ghi lý do chuyển tiếp.

| Phần | Nội dung |
|---|---|
| **I. Nền tảng: Go quản version thế nào** | Theo Jon Bodner, *Learning Go* ch.10 — ba tầng repo/module/package, SemVer + import compatibility rule, MVS, `go.mod` vs `go.sum`, proxy + checksum database, `replace`/`exclude`/`retract`, `/v2` trong import path, `go.work`, vendoring |
| **II. Bài toán: đưa module Go vào container** | Vì sao build context phải là gốc repo, luật `GOWORK=off`, vai trò `.dockerignore` |
| **III. Layer là gì** | Layer = diff chứ không phải bản sao. Ví von giấy can. Từng layer chứa đúng file nào. `diffID`. Tính bất biến |
| **IV. OverlayFS** | `lowerdir`/`upperdir`/`workdir`/`merged`, ba thao tác đọc–ghi–xoá, copy-up, whiteout, vì sao `apk --no-cache` |
| **V. Podman build: sáu giai đoạn** | Podman không tự build — ba thư viện bên dưới. Parse → kế hoạch stage → kéo base → working container → vòng lặp → commit |
| **VI. Vòng lặp chỉ thị và vì sao cache được** | Sáu bước, nhánh TRÚNG/TRƯỢT, ba tính chất khiến cache khả thi, đổ dây chuyền |
| **VII. Hai loại cache** | Layer cache vs cache mount — cái nào vào image, cái nào nằm trên host |
| **VIII. Dockerfile thật, từng dòng** | Cả hai stage, chú thích từng chỉ thị, hai cờ `go build` với số đo thật |
| **IX. Chạy ở đâu** | Podman/Buildah vs BuildKit, pipeline ba job, lệnh kiểm tra trước khi push |

Mở bằng https://app.diagrams.net rồi `File → Open`, hoặc cài extension
"Draw.io Integration" cho VS Code là sửa được ngay trong editor.

---

## 8. Nguồn tham khảo

- [podman-build — Podman documentation](https://docs.podman.io/en/latest/markdown/podman-build.1.html) — `--layers` mặc định `true`, quan hệ podman↔buildah, `--cache-from`/`--cache-to` cần `--layers`
- [buildah-build.1.md — containers/buildah](https://github.com/containers/buildah/blob/main/docs/buildah-build.1.md) — `buildah bud` mặc định `--layers=false`, `--cache-ttl`, `--skip-unused-stages`, cơ chế cache phân tán khác BuildKit
- [imagebuildah/stage_executor.go — containers/buildah](https://github.com/containers/buildah/blob/main/imagebuildah/stage_executor.go) — `Prepare`/`Execute`/`Run`/`Copy`/`Commit`, `ContentDigester`
- [Support for RUN --mount=type=cache · Issue #3452](https://github.com/containers/buildah/issues/3452) — lịch sử hỗ trợ cache mount
- [--mount=type=cache location · podman Discussion #15612](https://github.com/containers/podman/discussions/15612) — vị trí `$TMPDIR/buildah-cache`
- [Buildah v1.29.0 Release Announcement](https://buildah.io/releases/2023/01/27/Buildah-version-v1.29.0.html) — mốc ổn định cache mount, cache cha riêng theo user
- [Building container images with Buildah — Red Hat](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/building_running_and_managing_containers/assembly_building-container-images-with-buildah) — vai trò containers/image và containers/storage
