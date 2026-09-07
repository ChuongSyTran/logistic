# Elasticsearch — ghi chú từ nguyên lý tới vận hành

> Tài liệu dành cho người **mới học Elasticsearch (ES)**, đi từ bản chất → nguyên lý & thuật toán bên trong → khi nào nên dùng → cách triển khai thực tế trong dự án Logistics.
> Đọc hết là hiểu **ES làm gì, chạy ra sao, mạnh yếu chỗ nào**, chứ không chỉ biết copy câu query.

**Mục lục**

| Phần | Nội dung | Câu hỏi trả lời |
|---|---|---|
| [I](#phần-i--what--elasticsearch-là-cái-gì) | Bản chất, hệ sinh thái, mô hình dữ liệu | **WHAT** |
| [II](#phần-ii--why--vì-sao-elasticsearch-tồn-tại) | Bài toán mà database thường bó tay | **WHY** |
| [III](#phần-iii--nguyên-lý--thuật-toán-lõi) | Inverted index, BM25, segment, phân tán, các thuật toán | **HOW (bên trong)** |
| [IV](#phần-iv--where--dùng-ở-đâu) | Use case & vị trí trong kiến trúc | **WHERE** |
| [V](#phần-v--when--khi-nào-nên-và-khi-nào-đừng) | Tiêu chí quyết định, so sánh đối thủ | **WHEN** |
| [VI](#phần-vi--how--triển-khai-thực-tế) | Đồng bộ dữ liệu, Query DSL, code trong repo | **HOW (thực hành)** |
| [VII](#phần-vii--ưu-điểm) | Điểm mạnh | **ADVANTAGES** |
| [VIII](#phần-viii--nhược-điểm--cạm-bẫy) | Điểm yếu & cái giá phải trả | **DISADVANTAGES** |
| [IX](#phần-ix--lộ-trình-học--tra-cứu-nhanh) | Lộ trình học, cheat sheet | |

---
---

# PHẦN I — WHAT — Elasticsearch là cái gì?

## 1.1. Định nghĩa một câu

> **Elasticsearch là một search engine phân tán, lưu trữ document JSON, xây trên nền thư viện Apache Lucene, giao tiếp hoàn toàn qua HTTP/JSON.**

Bóc tách từng vế:

| Vế | Nghĩa là gì |
|---|---|
| **Search engine** | Sinh ra để **tìm kiếm và xếp hạng theo độ liên quan**, không phải để giữ dữ liệu chuẩn xác tuyệt đối |
| **Phân tán** | Dữ liệu tự động chia nhỏ ra nhiều máy, tự nhân bản, tự cân bằng khi thêm/bớt máy |
| **Document JSON** | Đơn vị dữ liệu là một object JSON, không phải một dòng bảng |
| **Trên nền Lucene** | Toàn bộ phần lõi tìm kiếm là của Lucene. ES là lớp áo phân tán + API bọc ngoài |
| **HTTP/JSON** | Không có driver riêng, không có ngôn ngữ query riêng kiểu SQL — chỉ là REST API |

## 1.2. Cây phả hệ — ai là ai trong hệ sinh thái

```
┌─────────────────────────────────────────────────────────┐
│  Kibana        │ Giao diện web: dashboard, biểu đồ, dev tool
├─────────────────────────────────────────────────────────┤
│  Logstash / Beats / Fluentd  │ Đường ống bơm dữ liệu vào
├─────────────────────────────────────────────────────────┤
│  ELASTICSEARCH               │ ◄── Chúng ta ở đây
│   • Cluster, Node, Shard, Replica  (lớp PHÂN TÁN)
│   • REST API, Query DSL, Mapping   (lớp GIAO TIẾP)
├─────────────────────────────────────────────────────────┤
│  APACHE LUCENE               │ Lõi tìm kiếm (Java)
│   • Inverted Index, Segment, BM25, Analyzer
│   • Chạy trên MỘT máy, không biết gì về mạng
├─────────────────────────────────────────────────────────┤
│  Ổ đĩa (segment files)                                  │
└─────────────────────────────────────────────────────────┘
```

**Ví dụ dễ hiểu:** Lucene là **động cơ xe**, cực mạnh nhưng trần trụi, một mình nó chẳng chở được ai. Elasticsearch là **cả chiếc xe hoàn chỉnh** — lắp động cơ Lucene vào, thêm vô lăng (REST API), thêm hệ thống nhiều bánh phối hợp (phân tán), thêm bảng điều khiển. Kibana là **cái màn hình giải trí** gắn thêm.

Hiểu điều này rất quan trọng: mỗi khi đọc về "inverted index", "segment", "BM25" — **đó là Lucene**. Còn "shard", "replica", "cluster", "index API" — **đó là Elasticsearch**.

## 1.3. Lịch sử ngắn (và một drama về giấy phép cần biết)

| Mốc | Sự kiện |
|---|---|
| 1999 | Doug Cutting viết **Lucene** (Java), sau này ông cũng là cha đẻ Hadoop |
| 2004 | Shay Banon viết **Compass** — bọc Lucene cho dễ xài |
| 2010 | Shay viết lại từ đầu thành **Elasticsearch**, thêm khả năng phân tán |
| 2015 | Bộ **ELK Stack** (Elasticsearch + Logstash + Kibana) bùng nổ trong giới DevOps |
| 2021 | Elastic **đổi giấy phép** từ Apache 2.0 sang SSPL/Elastic License → AWS **fork ra OpenSearch** |
| 2024 | Elastic bổ sung lựa chọn **AGPLv3**, chính thức được coi là open source trở lại |

> **Thực tế cần nắm:** hiện có **hai nhánh song song** — **Elasticsearch** (của Elastic) và **OpenSearch** (của AWS, fork từ ES 7.10). API gần như giống hệt nhau. Nếu deploy trên AWS thì dịch vụ managed thường là OpenSearch. Dự án mình đang dùng client `go-elasticsearch/v8`, tức nhánh Elastic chính chủ.

## 1.4. Mô hình dữ liệu — đối chiếu với SQL

```
CLUSTER  (cả cụm máy)
   └── INDEX  "transactions"          ≈ Table
         ├── MAPPING                  ≈ Schema (DDL)
         ├── SHARD 0 ─ replica 0      ≈ Partition + bản sao
         ├── SHARD 1 ─ replica 1
         └── DOCUMENT                 ≈ Row
               └── FIELD              ≈ Column
```

| Elasticsearch | SQL | Ghi chú quan trọng |
|---|---|---|
| Index | Table | Từ ES 7.0 **không còn "type"** — mỗi index chỉ chứa một loại document |
| Document | Row | Là JSON, **có thể lồng nhau**, có mảng — không phẳng như SQL |
| Field | Column | Một field có thể được index **nhiều kiểu cùng lúc** (multi-field) |
| Mapping | Schema | **Tạo rồi gần như không sửa được** — đây là điểm đau lớn nhất |
| `_id` | Primary key | Nếu không đặt, ES tự sinh chuỗi ngẫu nhiên |
| Query DSL | SQL | JSON lồng nhau. ES cũng có SQL API nhưng hạn chế |
| Shard | Partition | Số shard **chốt cứng lúc tạo index** |

## 1.5. Elasticsearch KHÔNG phải là gì — chống ngộ nhận

Đây là mục quan trọng nhất cho người mới. Ba ngộ nhận kinh điển:

**❌ "ES là database, dùng thay MySQL được."**
Không. ES **không có transaction đa document**, không có `ROLLBACK`, không đảm bảo ACID xuyên nhiều bản ghi. Không nên để số dư ví chỉ tồn tại trong ES. Với dữ liệu tài chính, nên xem ES là bản sao phục vụ đọc.

**❌ "ES trả kết quả tức thì sau khi ghi."**
Không. ES là **Near Real-Time** — mặc định ~1 giây sau khi ghi mới tìm thấy. Chi tiết ở [mục 3.5](#35-vòng-đời-ghi--refresh-flush-merge).

**❌ "ES có JOIN như SQL."**
Gần như không. ES có `nested` và `join` field nhưng đắt đỏ và hạn chế. Triết lý của ES là **denormalize** — nhồi sẵn dữ liệu cần thiết vào chung một document, chấp nhận trùng lặp để đổi lấy tốc độ.

---
---

# PHẦN II — WHY — Vì sao Elasticsearch tồn tại?

## 2.1. Bài toán mà B-Tree bó tay

Giả sử có bảng `transactions` 50 triệu dòng và cần tìm giao dịch có mô tả chứa `"tiền cọc"`:

```sql
SELECT * FROM transactions WHERE description LIKE '%tiền cọc%' ORDER BY created_at DESC;
```

**Vì sao câu này chết?** Index B-Tree của SQL sắp xếp dữ liệu **theo thứ tự từ trái sang phải của chuỗi**. Nó tra được `LIKE 'tiền%'` (biết chỗ bắt đầu để nhảy tới), nhưng `LIKE '%tiền%'` thì **không có điểm neo** → buộc phải quét toàn bộ 50 triệu dòng, so khớp từng chuỗi một.

**Ví dụ dễ hiểu:** B-Tree là **danh bạ điện thoại** — sắp theo tên nên tra "Nguyễn Văn A" cực nhanh. Nhưng hỏi *"ai có chữ 'Văn' trong tên?"* thì phải lật từng trang từ đầu tới cuối.

Và đó mới chỉ là bài toán dễ nhất. Năm bài toán sau thì SQL gần như **không có lời giải**:

| Bài toán | Vì sao SQL bó tay |
|---|---|
| **Xếp hạng theo độ liên quan** | `WHERE` chỉ trả lời đúng/sai, không trả lời "kết quả nào **hợp lý nhất**" |
| **Tìm trên nhiều cột cùng lúc** có trọng số khác nhau | Phải viết `OR` chồng chất, không cách nào ưu tiên "khớp ở tiêu đề quan trọng gấp 3 lần khớp ở mô tả" |
| **Chịu lỗi chính tả** | Gõ `"tiềnn cọc"` → SQL trả về 0 dòng |
| **Đếm gom nhóm nhiều chiều tức thì** (facet) | `GROUP BY` trên 50 triệu dòng = vài giây, ES trả về trong mili-giây |
| **Gợi ý gõ tới đâu ra tới đó** | Mỗi ký tự người dùng gõ là một câu `LIKE` mới |

## 2.2. Lời giải: lật ngược chỉ mục lại

Thay vì hỏi *"dòng này chứa từ nào?"*, ES lật ngược câu hỏi thành *"từ này nằm ở những dòng nào?"*.

```
DỮ LIỆU GỐC                          INVERTED INDEX (chỉ mục ngược)
┌────┬──────────────────────┐        ┌──────────┬─────────────┐
│ ID │ description          │        │ Term     │ Postings    │
├────┼──────────────────────┤        ├──────────┼─────────────┤
│ 1  │ đóng băng tiền cọc   │  ───►  │ băng     │ [1]         │
│ 2  │ hoàn tiền cọc        │        │ cọc      │ [1, 2]      │
│ 3  │ nạp tiền vào ví      │        │ đóng     │ [1]         │
└────┴──────────────────────┘        │ hoàn     │ [2]         │
                                     │ nạp      │ [3]         │
                                     │ tiền     │ [1, 2, 3]   │
                                     │ ví       │ [3]         │
                                     └──────────┴─────────────┘
```

Tìm `"tiền cọc"` → lấy giao của hai danh sách: `{1,2,3} ∩ {1,2}` = `{1, 2}`. **Không hề đụng vào dữ liệu gốc.**

Độ phức tạp thay đổi về chất:

| | Full scan (SQL `LIKE %x%`) | Inverted Index |
|---|---|---|
| Tra từ khoá | `O(N)` — N = tổng số dòng | `O(log T)` — T = số **từ khoá khác nhau** |
| Đặc điểm | Càng nhiều dữ liệu càng chậm tuyến tính | Từ vựng tăng chậm hơn nhiều so với số dòng |

> **Đây là cốt lõi:** dữ liệu tăng từ 1 triệu lên 100 triệu dòng, nhưng **số từ tiếng Việt khác nhau gần như không đổi**. Nên inverted index gần như không chậm đi.

## 2.3. Vậy sao không dùng `FULLTEXT` của MySQL / `tsvector` của Postgres?

Dùng được, và với dữ liệu nhỏ thì **nên dùng** (xem [Phần V](#phần-v--when--khi-nào-nên-và-khi-nào-đừng)). Nhưng chúng thiếu:

- Không phân tán ngang được — hết cỡ một máy là hết đường
- Chấm điểm liên quan yếu hơn nhiều (Postgres `ts_rank` thua xa BM25)
- Không có aggregation nhiều tầng tốc độ cao
- Không có fuzzy/phonetic/synonym/highlight/percolator sẵn dùng
- **Quan trọng nhất:** chạy chung tài nguyên với đường ghi. Câu search nặng sẽ giành CPU/IO với luồng ghi tiền thật.

---
---

# PHẦN III — NGUYÊN LÝ & THUẬT TOÁN LÕI

> Đây là phần dài nhất và cũng đáng giá nhất. Hiểu phần này thì mọi hành vi "kỳ lạ" của ES đều trở nên hiển nhiên.

## 3.1. Analysis Pipeline — nơi mọi thứ bắt đầu

Trước khi vào inverted index, **mọi văn bản đều bị băm nhỏ**. Quy trình gọi là **Analysis**, gồm 3 chặng:

```
  "Đóng Băng Tiền Cọc <b>2024</b>!"
              │
              ▼
   ① CHAR FILTER  (lọc ký tự thô)
      • gỡ thẻ HTML, thay thế ký tự
              │  "Đóng Băng Tiền Cọc 2024!"
              ▼
   ② TOKENIZER  (cắt thành token) ── BẮT BUỘC, chỉ được 1
      • standard: cắt theo ranh giới từ, bỏ dấu câu
              │  ["Đóng","Băng","Tiền","Cọc","2024"]
              ▼
   ③ TOKEN FILTER  (gọt từng token) ── xếp chồng nhiều lớp
      • lowercase  → ["đóng","băng","tiền","cọc","2024"]
      • stop       → bỏ từ vô nghĩa ("và","của","the","is")
      • stemmer    → "running"→"run", "cats"→"cat"
      • synonym    → "xe tải" ⇄ "truck"
      • ascii_folding → "tiền" → "tien"
              │
              ▼
      TOKEN CUỐI CÙNG → ghi vào Inverted Index
```

### Nguyên tắc VÀNG của Analysis

> **Câu truy vấn phải đi qua CÙNG một analyzer với lúc ghi dữ liệu.**

Vì sao? Lúc ghi, `"Đóng Băng"` được lưu thành `["đóng","băng"]`. Nếu lúc tìm, câu query **không** qua analyzer, ES sẽ đi tìm token `"Đóng Băng"` (viết hoa, còn nguyên cụm) — **không có trong index** → 0 kết quả.

**Đây chính là nguyên nhân số 1 khiến người mới học ES ngồi debug cả buổi.** Và nó dẫn thẳng tới cặp khái niệm sau.

### `text` vs `keyword` — cặp đôi gây nhầm lẫn nhất

| | `text` | `keyword` |
|---|---|---|
| Qua analyzer? | **CÓ** | **KHÔNG** — giữ nguyên si |
| `"Đóng băng tiền cọc"` lưu thành | `["đóng","băng","tiền","cọc"]` | `["Đóng băng tiền cọc"]` — **1 token** |
| Query đi kèm | `match`, `match_phrase` | `term`, `terms` |
| Dùng cho | Văn bản tự do cần tìm mò | ID, mã trạng thái, email, tag |
| Sort / Aggregate được? | ❌ (phải bật `fielddata`, rất ngốn RAM) | ✅ (có doc values) |

Nhớ một câu duy nhất:

> ### 🔑 **`term` đi với `keyword`, `match` đi với `text`.** Ghép chéo = 0 kết quả.

### Với tiếng Việt

`standard` tokenizer cắt theo **khoảng trắng và ranh giới Unicode**, nên `"tiền cọc"` thành 2 token riêng `["tiền","cọc"]`. Nó **không hiểu** đây là một từ ghép có nghĩa. Hệ quả: tìm `"cọc tiền"` vẫn ra kết quả vì cả 2 token đều khớp.

Muốn chuẩn tiếng Việt cần plugin tách từ (`vi_analyzer`, `coccoc-tokenizer`). Với dự án ví tiền — mô tả giao dịch ngắn và theo mẫu cố định — `standard` là đủ dùng.

API `_analyze` giúp quan sát trực tiếp: cho người dùng thấy tận mắt một câu bị cắt ra sao:

```bash
curl "localhost:9200/_analyze?pretty" -H 'Content-Type: application/json' -d '{"analyzer":"standard","text":"Đóng băng tiền cọc nhận cuốc"}'
```

## 3.2. Cấu trúc lưu trữ — các thuật toán nén

Inverted index không phải cái `HashMap` ngây thơ. Lucene dùng loạt cấu trúc dữ liệu chuyên biệt:

### A. Term Dictionary — **FST (Finite State Transducer)**

Từ điển từ khoá được nén bằng **FST** — một dạng automaton hữu hạn trạng thái, **gộp chung tiền tố và hậu tố** của các từ:

```
    Lưu 4 từ: "tien", "tiec", "tiem", "tiep"
    Cách ngây thơ: 4 × 4 ký tự = 16 ô nhớ
    FST:  t ─► i ─► e ─┬─► n
                       ├─► c        chỉ 7 nút!
                       ├─► m
                       └─► p
```

**Tính chất:**
- Tỉ lệ nén rất cao → **toàn bộ term dictionary nằm gọn trong RAM**
- Tra cứu `O(độ dài từ)` — **không phụ thuộc vào số lượng từ trong index**
- Duyệt được theo thứ tự từ điển → hỗ trợ tìm theo tiền tố, range trên chuỗi

FST cũng chính là thứ giúp **Completion Suggester** (gợi ý autocomplete) nhanh tới mức phi lý.

### B. Postings List — **Delta Encoding + Frame of Reference**

Danh sách document ID cho mỗi term có thể dài hàng triệu phần tử. Lucene nén hai tầng:

```
Doc IDs gốc:      [ 73,  75,  80,  91,  95, 300 ]
① Delta encoding: [ 73,   2,   5,  11,   4, 205 ]   ← chỉ lưu HIỆU SỐ
② Frame of Ref:   chia lô 128 số, mỗi lô tìm số lớn nhất
                  rồi đóng gói bit-packed vừa khít
                  (số nhỏ chỉ tốn 3-5 bit thay vì 32 bit)
```

Kết quả: postings list co lại còn **khoảng 1/4 kích thước gốc**. Ít byte đọc từ đĩa = nhanh hơn, và cache được nhiều hơn.

### C. **Skip List** — nhảy cóc khi giao hai danh sách

Khi tìm `"tiền AND cọc"`, ES phải giao 2 postings list. Nếu `"tiền"` có 1 triệu doc mà `"cọc"` chỉ có 100 doc, duyệt tuần tự cả triệu là quá phí.

Lucene gắn **skip list nhiều tầng** vào postings, cho phép **nhảy thẳng** tới vị trí gần đúng — giống mục lục chương trong sách, khỏi lật từng trang.

### D. **Roaring Bitmap** — cache cho filter

Kết quả của một mệnh đề `filter` được cache dưới dạng **Roaring Bitmap** — cấu trúc bitmap nén thích ứng, tự chọn cách biểu diễn tối ưu theo mật độ dữ liệu.

> **Đây là lý do kỹ thuật vì sao `filter` nhanh hơn `must`:** kết quả `filter` chỉ là đúng/sai (bitmap), cache lại tái dùng được cho mọi request sau. Còn `must` phải tính điểm số cho từng document, không cache được.

### E. **Doc Values** — cấu trúc cột cho sort & aggregate

Inverted index trả lời cực nhanh *"term này ở doc nào?"*, nhưng lại **rất tệ** ở câu hỏi ngược: *"doc số 42 có giá trị field `balance` bằng bao nhiêu?"* — mà sort và aggregate cần đúng câu hỏi ngược này.

Nên Lucene ghi **thêm một cấu trúc thứ hai** gọi là **doc values** — lưu theo **cột** (columnar), trên đĩa, được OS cache:

```
INVERTED INDEX (để TÌM)          DOC VALUES (để SORT / GOM NHÓM)
  term → [doc1, doc5, ...]         doc1 → 50000
                                   doc2 → 12000
                                   doc3 → 78000
```

| | Inverted Index | Doc Values |
|---|---|---|
| Hướng | term → docs | doc → value |
| Phục vụ | `match`, `term`, `range` | `sort`, `aggregations`, `script` |
| Bật mặc định cho | mọi field index | mọi field **trừ `text`** |

> **Vì sao không sort được trên field `text`?** Vì `text` **không có doc values** (nội dung đã bị băm thành token, không còn giá trị nguyên bản). Muốn sort/aggregate trên văn bản, bắt buộc phải có sub-field `keyword` — chính là kỹ thuật **multi-field** dùng trong dự án:
> ```json
> "reference_id": { "type": "text", "fields": { "keyword": { "type": "keyword" } } }
> ```
> Field `reference_id` để tìm mò, `reference_id.keyword` để lọc/sort/gom nhóm.

## 3.3. Thuật toán chấm điểm — trái tim của "độ liên quan"

Đây là thứ phân biệt **search engine** với **database**. SQL trả lời "đúng/sai", ES trả lời **"hợp lý tới mức nào"**.

### Bước 1 — Boolean model: lọc ứng viên

Trước tiên ES dùng logic tập hợp (`AND`/`OR`/`NOT` trên các postings list) để tìm ra tập document **có khả năng** khớp. Nhanh, rẻ, không tính toán gì.

### Bước 2 — Tính điểm liên quan

#### TF-IDF — nền móng cổ điển

Hai trực giác đơn giản:

- **TF (Term Frequency)** — từ khoá xuất hiện càng nhiều lần trong một document, document đó càng liên quan.
- **IDF (Inverse Document Frequency)** — từ khoá càng **hiếm** trên toàn bộ index thì càng **quý**.

> Từ `"tiền"` xuất hiện trong 90% giao dịch → gần như vô giá trị phân biệt.
> Từ `"escrow"` chỉ có trong 20 giao dịch → cực kỳ có giá trị.

```
score(term, doc) = TF(term, doc) × IDF(term)
                       ▲               ▲
              đếm trong 1 doc    độ hiếm toàn index
```

Cộng thêm **Field-length norm**: khớp 1 từ trong tiêu đề 5 chữ ý nghĩa hơn nhiều so với khớp trong bài viết 5000 chữ.

#### BM25 — mặc định của ES từ phiên bản 5.0

BM25 (Best Matching 25) sửa hai khuyết điểm chí tử của TF-IDF:

**Khuyết điểm 1 — TF tăng tuyến tính vô hạn.** Với TF-IDF, document nhắc từ khoá 100 lần được điểm gấp 100 lần document nhắc 1 lần. Vô lý — đó là spam. BM25 áp dụng **đường cong bão hoà (saturation)**: nhắc từ lần thứ 2 tăng điểm nhiều, lần thứ 20 gần như không tăng nữa.

```
điểm ▲
     │        ╭──────────────  BM25 (bão hoà)
     │      ╭─╯
     │    ╭─╯        ╱  TF-IDF (tuyến tính, dễ bị spam)
     │  ╭─╯        ╱
     │╭─╯       ╱
     └────────────────────────► số lần xuất hiện
```

**Khuyết điểm 2 — chuẩn hoá độ dài quá thô.** BM25 cho tham số điều chỉnh mức độ phạt document dài.

Công thức đầy đủ:

```
                                    f(q,D) · (k₁ + 1)
BM25(q,D) = IDF(q) · ─────────────────────────────────────────
                     f(q,D) + k₁ · (1 − b + b · |D| / avgdl)

  f(q,D) = số lần từ q xuất hiện trong document D
  |D|    = độ dài document D    ;  avgdl = độ dài trung bình
  k₁     = 1.2  → điều khiển tốc độ bão hoà (k₁ nhỏ = bão hoà sớm)
  b      = 0.75 → mức phạt document dài (b=0 tắt hẳn, b=1 phạt tối đa)

  IDF(q) = ln( 1 + (N − n + 0.5) / (n + 0.5) )
           N = tổng số document  ;  n = số document chứa q
```

**Khi nào cần chỉnh `k₁`, `b`?** Gần như không bao giờ với ứng dụng nghiệp vụ thông thường. Chỉ đụng vào khi làm search engine cho nội dung dài (bài báo, tài liệu) và đo được rằng document dài đang bị thiệt thòi bất công.

#### Block-Max WAND — thuật toán lấy top-K nhanh

Người dùng chỉ xem 10 kết quả đầu, nhưng có thể có 5 triệu document khớp. Chấm điểm hết 5 triệu rồi lấy 10 là quá phí.

Lucene 8+ dùng **Block-Max WAND** (Weak AND): mỗi lô postings được lưu sẵn **điểm tối đa có thể đạt được**. Trong lúc duyệt, nếu điểm trần của cả một lô còn thấp hơn document thứ 10 hiện tại, thuật toán **bỏ qua nguyên lô** không thèm tính.

> Đây là lý do lấy top-10 nhanh hơn hẳn top-1000, và cũng là lý do `"track_total_hits": false` (mặc định chỉ đếm chính xác tới 10.000) làm query nhanh hơn — ES được phép bỏ qua sớm thay vì đếm cho đủ.

#### Vector Search & HNSW (ES 8.x — thế hệ mới)

Từ ES 8.0 có kiểu `dense_vector` và truy vấn `knn`, phục vụ **semantic search** và **RAG**: thay vì khớp từ khoá, so khớp **ý nghĩa** qua embedding.

Thuật toán đằng sau là **HNSW (Hierarchical Navigable Small World)** — đồ thị nhiều tầng, tìm láng giềng gần nhất **xấp xỉ** với độ phức tạp gần `O(log N)` thay vì `O(N)` nếu duyệt cạn.

```
Tầng 2 (thưa):   A ─────────────── F        ← nhảy xa, định vị nhanh
Tầng 1:          A ─── C ───── E ── F
Tầng 0 (đầy đủ): A─B─C─D─E─F─G─H─I─J        ← tinh chỉnh chính xác
```

Xu hướng hiện nay là **hybrid search**: kết hợp BM25 (khớp từ khoá chính xác) với kNN (khớp ngữ nghĩa), gộp điểm bằng **RRF (Reciprocal Rank Fusion)**.

### Các thuật toán khác hay gặp

| Nhu cầu | Thuật toán | Ghi chú |
|---|---|---|
| Chịu lỗi chính tả | **Levenshtein Automaton** | `fuzziness: AUTO`, tối đa 2 ký tự sai. Automaton hoá nên nhanh, không so từng cặp |
| Autocomplete | **FST** + edge n-gram | Completion Suggester |
| Tìm đoạn giữa chuỗi | **N-gram** | Cắt `"tiền"` → `["ti","ie","en"]`. Index phình to, dùng có cân nhắc |
| Tìm theo cách đọc | **Soundex / Metaphone** | "Nguyen" ≈ "Nguyên" |
| Đếm giá trị duy nhất | **HyperLogLog++** | `cardinality` agg — xấp xỉ, sai số ~0.5%, RAM cố định |
| Tính phân vị | **T-Digest** | `percentiles` agg — xấp xỉ, chính xác cao ở hai đuôi (p99) |
| Tìm từ khoá bất thường | **JLH score** | `significant_terms` — "từ nào nổi bật khác thường trong nhóm này" |
| Gom kết quả theo nhóm | Bucket aggregation | Nền tảng của bộ lọc facet trong e-commerce |

> **Lưu ý về "xấp xỉ":** `cardinality` và `percentiles` **cố tình** trả kết quả gần đúng để đổi lấy RAM cố định và tốc độ. Nếu cần con số chính xác tuyệt đối (ví dụ tổng tiền để đối soát kế toán) → **phải hỏi MySQL**, không hỏi ES.

## 3.4. Segment — nền tảng của mọi hành vi "kỳ lạ"

Một shard **không phải một file duy nhất**, mà là tập hợp nhiều **segment**:

```
   SHARD (= 1 Lucene index)
   ├── segment_1  (bất biến)  ← inverted index + doc values + stored fields
   ├── segment_2  (bất biến)
   ├── segment_3  (bất biến)
   ├── ...
   └── commit point  (danh sách segment đang có hiệu lực)
```

### Nguyên tắc cốt lõi: **SEGMENT LÀ BẤT BIẾN (immutable)**

Đã ghi ra đĩa thì **không bao giờ sửa**. Điều này nghe kỳ quặc nhưng mang lại 4 lợi ích lớn:

1. **Không cần khoá khi đọc** — nhiều luồng đọc song song thoải mái, không lock
2. **Cache vĩnh viễn hợp lệ** — file không đổi thì cache không bao giờ bẩn
3. **Nén mạnh tay** — biết trước toàn bộ dữ liệu nên nén tối ưu được
4. **OS page cache phát huy tối đa** — file tĩnh nằm lì trong RAM của hệ điều hành

**Cái giá phải trả:**

| Thao tác | Thực chất diễn ra |
|---|---|
| **Xoá** | Không xoá thật. Chỉ đánh dấu vào file `.del` (**tombstone** — bia mộ). Document vẫn nằm đó, chỉ bị lọc ra khi trả kết quả |
| **Sửa** | = **Xoá + Chèn mới**. Bản cũ thành tombstone, bản mới ghi vào segment mới |
| **Dọn rác thật sự** | Chỉ xảy ra khi **merge** segment |

> Đây là lý do index bị update liên tục sẽ **phình to hơn dữ liệu thật rất nhiều** — đầy xác tombstone chưa được dọn.

## 3.5. Vòng đời ghi — Refresh, Flush, Merge

```
  ① Client gửi request index document
                │
                ▼
  ┌─────────────────────────────────────────────────────────┐
  │  IN-MEMORY BUFFER          │   TRANSLOG (ghi đĩa ngay)  │
  │  (chưa tìm kiếm được)      │   ← đảm bảo không mất dữ liệu│
  └──────────┬─────────────────┴────────────────────────────┘
             │
             │  ② REFRESH — mặc định mỗi 1 GIÂY
             ▼     buffer → segment mới (nằm ở OS cache)
  ┌─────────────────────────────┐
  │  SEGMENT MỚI → TÌM ĐƯỢC RỒI │  ◄── đây chính là "Near Real-Time"
  └──────────┬──────────────────┘
             │
             │  ③ FLUSH — khi translog đầy (~512MB) hoặc định kỳ 30 phút
             ▼     fsync segment xuống đĩa thật + xoá translog
  ┌─────────────────────────────┐
  │  SEGMENT BỀN VỮNG TRÊN ĐĨA  │
  └──────────┬──────────────────┘
             │
             │  ④ MERGE — chạy ngầm liên tục
             ▼     gộp segment nhỏ thành lớn + dọn sạch tombstone
  ┌─────────────────────────────┐
  │  ÍT SEGMENT HƠN → TÌM NHANH │
  └─────────────────────────────┘
```

### ② REFRESH — nguồn gốc của "Near Real-Time"

> **Document vừa ghi KHÔNG tìm thấy ngay. Phải đợi refresh (mặc định 1 giây).**

Ba lựa chọn khi index:

| `refresh` | Hành vi | Dùng khi |
|---|---|---|
| `"false"` | Ghi xong trả về luôn, ~1s sau mới tìm thấy | **Mặc định — chuẩn cho production** |
| `"wait_for"` | Chặn tới chu kỳ refresh kế tiếp mới trả về | UI cần thấy kết quả ngay sau khi ghi |
| `"true"` | Ép refresh ngay lập tức | **Chỉ dùng khi viết test** |

**Vì sao `"true"` giết hiệu năng?** Mỗi lần refresh sinh ra **một segment mới**. Ép refresh mỗi document = đẻ ra hàng nghìn segment vụn → merge chạy điên cuồng, ngốn sạch I/O và CPU.

Với index kiểu log chỉ ghi không đọc ngay, tăng `index.refresh_interval` lên `30s` là mẹo tối ưu throughput cực kỳ hiệu quả.

### ① TRANSLOG — vì sao ghi 1 giây/lần mà không mất dữ liệu?

Nếu chỉ dựa vào refresh mỗi giây thì mất điện = mất trắng 1 giây dữ liệu. Nên ES ghi **song song** vào **Translog** (transaction log — giống WAL/binlog của database) **ngay lập tức, xuống đĩa thật**.

Mặc định `index.translog.durability: request` — **fsync sau mỗi request** trước khi báo thành công. Máy sập, lúc khởi động lại ES replay translog để dựng lại phần chưa flush.

Đổi sang `async` (fsync mỗi 5 giây) sẽ nhanh hơn đáng kể nhưng **có thể mất 5 giây dữ liệu**. Với log/metric thì chấp nhận được; với dữ liệu nghiệp vụ thì không.

### ④ MERGE — dọn dẹp ngầm

Segment nhỏ nhiều = tìm kiếm chậm (phải hỏi từng segment rồi gộp kết quả). ES chạy **TieredMergePolicy** ngầm: gộp các segment cùng cỡ thành segment lớn hơn, **đồng thời vứt bỏ tombstone**.

Merge ngốn I/O rất nặng. Với index **đã ngừng ghi** (ví dụ log tháng trước), chạy `_forcemerge?max_num_segments=1` sẽ nén về 1 segment → truy vấn nhanh hơn và tốn ít đĩa hơn. **Không nên force merge index đang được ghi.**

## 3.6. Kiến trúc phân tán

### Node, Shard, Replica

```
        CLUSTER "logistic-es"
   ┌──────────────┬──────────────┬──────────────┐
   │   NODE A     │   NODE B     │   NODE C     │
   ├──────────────┼──────────────┼──────────────┤
   │  P0  ██      │  P1  ██      │  P2  ██      │  P = Primary shard
   │  R2  ░░      │  R0  ░░      │  R1  ░░      │  R = Replica shard
   └──────────────┴──────────────┴──────────────┘
          ▲
     Index "transactions": 3 primary + 1 replica mỗi shard
     Quy tắc sắt: replica KHÔNG BAO GIỜ nằm cùng node với primary của nó
```

| Khái niệm | Vai trò |
|---|---|
| **Primary shard** | Bản gốc. Mọi thao tác ghi đi qua đây trước |
| **Replica shard** | Bản sao. Vừa dự phòng khi node chết, **vừa gánh tải đọc** |
| **Coordinating node** | Node nhận request, phân phát tới các shard, gộp kết quả trả về |
| **Master node** | Quản lý cluster state (index nào, shard ở đâu). **Không** đụng vào dữ liệu |

> **Replica không chỉ để dự phòng — nó nhân đôi khả năng đọc.** Tăng replica là một cách mở rộng throughput đọc.

### Routing — document đi về shard nào?

```
shard_number = hash(_routing) % number_of_primary_shards
                      ▲
              mặc định _routing = _id của document
```

> ### ⚠️ Đây là lý do **KHÔNG THỂ đổi số primary shard** sau khi tạo index.
> Đổi mẫu số thì công thức chia lấy dư ra kết quả khác → toàn bộ document cũ "biến mất" vì ES đi tìm sai shard. Muốn đổi phải `_reindex` sang index mới (hoặc dùng Split/Shrink API với ràng buộc chặt).

### Truy vấn phân tán — hai pha Query then Fetch

```
  ① QUERY PHASE
     Coordinating node ──► gửi query tới TẤT CẢ shard liên quan
     Mỗi shard tự tìm, tự chấm điểm, trả về CHỈ [doc_id + score]
     (nhẹ, chưa lấy nội dung)
                │
                ▼
     Coordinator gộp tất cả, sắp xếp, chọn ra top-K toàn cục
                │
  ② FETCH PHASE
                ▼
     Coordinator ──► hỏi lại đúng những shard chứa top-K
                     để lấy `_source` thật
                │
                ▼
              Trả về client
```

**Hai hệ quả quan trọng:**

1. **Deep pagination cực đắt.** Muốn lấy trang 1000 (`from=10000, size=10`), **mỗi shard** phải trả về 10.010 kết quả cho coordinator gộp. 5 shard = 50.050 bản ghi phải xử lý trong RAM chỉ để lấy ra 10 dòng. Đây chính là lý do có trần cứng:

   > **`index.max_result_window` mặc định = 10.000.** `from + size` vượt qua là ES ném lỗi thẳng.

   Giải pháp đi sâu: **`search_after`** (phân trang bằng con trỏ, dùng giá trị sort của document cuối trang trước) — không giới hạn, chi phí không đổi theo độ sâu. Cần ảnh chụp nhất quán thì kèm **PIT (Point In Time)**. Scroll API đã lỗi thời.

2. **Điểm số có thể lệch nhẹ giữa các shard.** IDF được tính **cục bộ trên từng shard**, không phải toàn cục. Index nhỏ mà chia nhiều shard, phân bố dữ liệu lệch → thứ hạng hơi sai. Sửa bằng `search_type=dfs_query_then_fetch` (thu thập thống kê toàn cục trước, tốn thêm một vòng mạng).

### Mô hình nhất quán & xử lý xung đột

| Đặc tính | ES |
|---|---|
| Transaction đa document | ❌ **Không có** |
| ACID trên **một** document | ✅ Có |
| Nhất quán giữa primary ↔ replica | **Eventual** — replica có độ trễ |
| Chống ghi đè song song | ✅ **Optimistic concurrency** qua `_seq_no` + `_primary_term` |

Cơ chế optimistic: client đọc document kèm `_seq_no`, khi ghi lại gửi kèm điều kiện `if_seq_no=...&if_primary_term=...`. Nếu ai đó đã sửa trước → ES trả **409 Conflict**, client tự quyết định thử lại. Đây là kiểu **khoá lạc quan**, khác hẳn `SELECT ... FOR UPDATE` (khoá bi quan) của MySQL.

### Bầu chọn Master & Split-brain

Master node được bầu qua thuật toán đồng thuận **dựa trên quorum** (từ ES 7.0 là Zen2, thiết kế theo hướng Raft). Quy tắc: cần **quá bán** master-eligible node đồng ý.

> **Vì sao luôn phải có SỐ LẺ master node (3, 5)?** Nếu chỉ có 2 node mà đứt mạng giữa chúng, mỗi bên tự cho mình là master → **split-brain**, hai bên ghi dữ liệu khác nhau, gộp lại là thảm hoạ. Với 3 node, bên nào giữ được 2 node mới thắng, bên còn lại tự động ngừng phục vụ. Đây là bài học xương máu của ES thời kỳ đầu.

---
---

# PHẦN IV — WHERE — Dùng ở đâu?

## 4.1. Sáu nhóm use case kinh điển

| # | Use case | Ví dụ | Vì sao ES thắng |
|---|---|---|---|
| 1 | **Full-text search** | Tìm sản phẩm, tra cứu tài liệu | Đúng nghề gốc: BM25, fuzzy, synonym, highlight |
| 2 | **Log & Observability** | ELK Stack, APM, tracing | Ghi rất nhiều/đọc ít, dữ liệu theo thời gian, aggregation nhanh |
| 3 | **Security Analytics (SIEM)** | Phát hiện xâm nhập | Truy vấn tương quan tức thì trên khối log khổng lồ |
| 4 | **E-commerce facet** | Lọc theo hãng/giá/màu + đếm số lượng | Aggregation nhiều tầng trong mili-giây |
| 5 | **Geo search** | "Tài xế trong bán kính 5km" | `geo_point`, `geo_distance` dùng cây BKD |
| 6 | **Vector / RAG** | Semantic search, chatbot doanh nghiệp | `dense_vector` + HNSW + hybrid với BM25 |

## 4.2. Vị trí trong kiến trúc — luôn là công dân hạng hai

```
   ┌──────────────────────────────────────────────────────┐
   │              ĐƯỜNG GHI  (Command)                    │
   │   Client ──► Service ──► MySQL/Postgres              │
   │                          ↑ SOURCE OF TRUTH           │
   │                          │ ACID, transaction, ràng buộc│
   └──────────────────────────┼───────────────────────────┘
                              │ đồng bộ (mục 6.1)
                              ▼
   ┌──────────────────────────────────────────────────────┐
   │              ĐƯỜNG ĐỌC  (Query)                      │
   │   Client ──► Service ──► Elasticsearch               │
   │                          ↑ READ MODEL                │
   │                          │ có thể mất, dựng lại được │
   └──────────────────────────────────────────────────────┘
```

Mô hình này gọi là **CQRS (Command Query Responsibility Segregation)**.

> ### 🔑 Nguyên tắc bất di bất dịch
> **Mất sạch cluster ES phải là sự cố có thể khắc phục bằng cách re-index lại từ database — không mất một đồng nào, không mất một bản ghi nghiệp vụ nào.**
> Nếu xoá ES mà mất dữ liệu vĩnh viễn → kiến trúc đã sai từ gốc.

Ngoại lệ duy nhất: hệ thống log/metric thuần tuý, nơi ES **chính là** kho lưu trữ và mất log cũ là chấp nhận được.

## 4.3. Trong dự án Logistics này

Hiện tại **chỉ `wallet_service` dùng ES** ([`internal/search/`](../../wallet_service/internal/search)). Các chỗ khác có thể cân nhắc:

| Service | Bài toán tiềm năng | Nên dùng ES? |
|---|---|---|
| `wallet_service` | Tra cứu lịch sử giao dịch, lọc đa điều kiện | ✅ **Đang dùng** |
| `user_service` | Tìm tài xế/chủ hàng theo tên, sđt, biển số | ✅ Hợp — nhiều field, cần tìm gần đúng |
| `vehicle_service` | Tìm xe theo tải trọng, loại, khu vực | ⚠️ Lọc số thuần tuý — MySQL index là đủ |
| `matching_service` | Ghép chuyến theo vị trí + tải trọng + giá | ❌ **Không** — cần chính xác tuyệt đối và độ trễ cực thấp. Đang dùng Redis/in-memory là đúng |
| Toàn hệ thống | Gom log tập trung (ELK) | ✅ Rất đáng làm khi lên production |

---
---

# PHẦN V — WHEN — Khi nào nên, và khi nào ĐỪNG

## 5.1. NÊN dùng khi có đủ các dấu hiệu

- ✅ Dữ liệu **trên 1 triệu bản ghi** và còn tăng
- ✅ Cần tìm **văn bản tự do** (`LIKE '%...%'` đang giết database)
- ✅ Cần **xếp hạng theo độ liên quan**, không chỉ lọc đúng/sai
- ✅ Cần **facet/aggregation nhiều chiều** phản hồi tức thì
- ✅ Tỉ lệ **đọc áp đảo ghi**, và chấp nhận trễ ~1 giây
- ✅ Cần **mở rộng ngang** vượt giới hạn một máy
- ✅ Có người **vận hành được** cluster (đây là điều kiện hay bị bỏ quên nhất)

## 5.2. ĐỪNG dùng khi

- ❌ Dưới **500 nghìn bản ghi** → Postgres `tsvector` + GIN index là quá đủ, và rẻ hơn nhiều lần
- ❌ Cần **transaction/ACID** trên dữ liệu đó
- ❌ Cần **join nhiều bảng** phức tạp
- ❌ Cần đọc thấy dữ liệu **ngay lập tức** sau khi ghi (dưới 1 giây)
- ❌ Cần **con số chính xác tuyệt đối** để đối soát tài chính
- ❌ Chỉ để **thay thế cache** → Redis đúng việc hơn nhiều
- ❌ Đội chưa có ai vận hành được → cluster ES vỡ lúc 2 giờ sáng là cơn ác mộng có thật

> **Câu hỏi tự vấn trước khi quyết định:**
> *"Đã đánh index đúng cho MySQL/Postgres chưa? Đã thử `tsvector` chưa? Nếu chưa mà đã nhảy sang ES, gần như chắc chắn đang dùng dao mổ trâu giết gà."*

## 5.3. So sánh các lựa chọn

| Tiêu chí | MySQL `FULLTEXT` | Postgres `tsvector` | **Elasticsearch** | Meilisearch / Typesense | Algolia (SaaS) |
|---|---|---|---|---|---|
| Chi phí vận hành | Không (đã có sẵn) | Không | **Cao** | Thấp | Không (trả tiền) |
| Chấm điểm liên quan | Yếu | Khá | **Xuất sắc** | Tốt | Xuất sắc |
| Mở rộng ngang | ❌ | ❌ | **✅** | Hạn chế | ✅ |
| Aggregation/Facet | Yếu | Trung bình | **Xuất sắc** | Cơ bản | Tốt |
| Vector/Semantic | ❌ | ✅ (pgvector) | **✅ (HNSW)** | Một phần | ✅ |
| Ngưỡng dữ liệu hợp lý | < 1 triệu | < 10 triệu | **Không giới hạn** | < 10 triệu | Theo gói tiền |
| ACID | ✅ | ✅ | ❌ | ❌ | ❌ |
| Độ khó bắt đầu | Dễ | Dễ | **Khó** | Rất dễ | Rất dễ |

**Tóm tắt quyết định:**
- Dữ liệu nhỏ, đã có Postgres → **`tsvector`**
- Cần search đẹp, không muốn vận hành, ngân sách ổn → **Algolia / Meilisearch Cloud**
- Dữ liệu lớn, cần aggregation mạnh, cần toàn quyền kiểm soát → **Elasticsearch**
- Chỉ cần lọc chính xác theo cột → **index B-Tree của database, đừng nghĩ gì thêm**

---
---

# PHẦN VI — HOW — Triển khai thực tế

## 6.1. Bài toán khó nhất: đồng bộ Database → Elasticsearch

Đây mới là chỗ dự án thực tế hay vỡ trận, chứ không phải viết query.

### Bốn chiến lược

#### ① Dual Write — ghi thẳng hai nơi

```go
tx.Commit()                    // ghi MySQL
esEngine.IndexWallet(doc)      // rồi ghi ES
```

| ✅ | ❌ |
|---|---|
| Đơn giản, làm 5 phút xong | **Không nguyên tử** — ES fail là dữ liệu lệch vĩnh viễn, không ai biết |

Chỉ chấp nhận được khi: lệch dữ liệu không gây hậu quả nghiêm trọng **và** có job đối soát định kỳ. **Đây là cách dự án đang dùng.**

#### ② Transactional Outbox — chuẩn mực công nghiệp

```
   ┌─ TRANSACTION MySQL ─────────────────────┐
   │  UPDATE wallets  SET balance = ...      │
   │  INSERT INTO outbox (payload, status)   │  ← cùng 1 transaction!
   └─────────────────────────────────────────┘
                    │ commit
                    ▼
        Worker đọc bảng outbox ──► đẩy sang ES ──► đánh dấu đã xử lý
```

| ✅ | ❌ |
|---|---|
| **Không bao giờ mất sự kiện** — outbox và dữ liệu cùng nằm trong một transaction. Retry thoải mái | Thêm một bảng + một worker. Trễ vài trăm ms |

Repo **đã có sẵn Kafka và `concurrencyUtils.Worker`** → hạ tầng để làm outbox đã có đủ, chỉ thiếu bảng outbox và worker đẩy.

#### ③ CDC (Change Data Capture) — Debezium

Đọc thẳng **binlog** của MySQL, biến mọi thay đổi thành sự kiện Kafka.

| ✅ | ❌ |
|---|---|
| **Code nghiệp vụ không cần biết ES tồn tại**. Bắt được cả thay đổi do sửa tay trong DB | Thêm hạ tầng nặng (Debezium + Kafka Connect). Vận hành phức tạp |

#### ④ Batch Reindex định kỳ

Job chạy đêm quét toàn bộ DB đẩy lại vào ES.

| ✅ | ❌ |
|---|---|
| Đơn giản nhất, tự chữa lành mọi sai lệch | Dữ liệu cũ tới cả ngày |

> **Khuyến nghị thực tế:** dùng **② Outbox** làm đường chính + **④ Batch reindex** chạy hàng đêm làm lưới an toàn đối soát. Đây là combo mà hầu hết hệ thống tài chính nghiêm túc đang dùng.

### Nguyên tắc chống trùng lặp

> **Luôn đặt `DocumentID` = ID của entity trong database.**

Vì `_id` trùng thì ES **ghi đè** (upsert), nên mọi thao tác index trở nên **idempotent** — chạy lại 100 lần vẫn ra đúng một document. Nếu để ES tự sinh `_id`, mỗi lần retry sẽ đẻ ra một bản trùng mới. Dự án đang làm đúng điểm này.

## 6.2. Query DSL — bộ khung cần thuộc

### `bool` query — 4 mệnh đề phải nhớ

```json
{ "query": { "bool": {
    "must":     [ ... ],   // AND — CÓ tính điểm
    "filter":   [ ... ],   // AND — KHÔNG tính điểm, ĐƯỢC CACHE
    "should":   [ ... ],   // OR  — có tính điểm
    "must_not": [ ... ]    // NOT — không tính điểm
}}}
```

| Mệnh đề | SQL | Tính `_score`? | Cache? |
|---|---|---|---|
| `must` | `AND` | ✅ | ❌ |
| `filter` | `AND` | ❌ | ✅ **nhanh hơn nhiều** |
| `should` | `OR` | ✅ | ❌ |
| `must_not` | `NOT` | ❌ | ✅ |

> ### 🔑 Quy tắc tối ưu quan trọng nhất
> **Điều kiện nào chỉ trả lời đúng/sai (trạng thái, khoảng ngày, ID, loại) → bỏ vào `filter`.**
> **Chỉ thứ thực sự cần xếp hạng (văn bản người dùng gõ) mới để trong `must`.**
> Chuyển đúng chỗ có thể nhanh gấp nhiều lần nhờ Roaring Bitmap cache.

### Các loại query hay dùng

| Query | Công dụng | Đi với kiểu field |
|---|---|---|
| `match` | Full-text, có analyze | `text` |
| `match_phrase` | Đúng cụm, đúng thứ tự | `text` |
| `match_phrase_prefix` | Cụm + từ cuối là tiền tố (search-as-you-type) | `text` |
| `multi_match` | Tìm trên nhiều field cùng lúc | `text` |
| `term` / `terms` | Khớp **chính xác**, không analyze | `keyword`, số, ngày |
| `range` | `gte` `gt` `lte` `lt` | số, ngày |
| `exists` | Field có tồn tại không | mọi kiểu |
| `wildcard` / `regexp` | Ký tự đại diện — **rất chậm**, tránh dùng | `keyword` |
| `fuzzy` | Chịu lỗi chính tả (Levenshtein) | `text` |
| `knn` | Tìm theo ngữ nghĩa (vector) | `dense_vector` |

## 6.3. Implement trong `wallet_service`

**File nguồn:** [`wallet_service/internal/search/elasticsearch.go`](../../wallet_service/internal/search/elasticsearch.go) — 380 dòng, là toàn bộ phần ES của repo.
**Client:** `github.com/elastic/go-elasticsearch/v8 v8.19.7`

### Bản đồ luồng

```
 gRPC SearchWallets
        │
        ▼
 controller/wallet_controller.go:102   pb.Req ──► search.SearchWalletParams
        │
        ▼
 search/elasticsearch.go
   buildWalletQuery()  → dựng Query DSL bằng map[string]any
   doSearch[T]()       → gọi HTTP, bóc _source
        │
        ▼
 mapper.ESWalletToProto()  → pb.WalletInfo

 ĐƯỜNG GHI:
 biz/wallet_biz.go:99 ─► mapper.EntityToESWallet() ─► esEngine.IndexWallet()
```

| File | Vai trò |
|---|---|
| [`internal/search/elasticsearch.go`](../../wallet_service/internal/search/elasticsearch.go) | Document model, mapping, query builder, client |
| [`internal/di/injection.go:51-63`](../../wallet_service/internal/di/injection.go) | Khởi tạo client, tạo index lúc boot, cho phép chạy khi ES chết |
| [`internal/biz/wallet_biz.go`](../../wallet_service/internal/biz/wallet_biz.go) | Đẩy dữ liệu sang ES sau khi ghi MySQL |
| [`internal/controller/wallet_controller.go:100-182`](../../wallet_service/internal/controller/wallet_controller.go) | Nhận gRPC, gọi search, map ra proto |

### A. Document Model

```go
// elasticsearch.go:18
type WalletDocument struct {
	ID        string    `json:"id"`         // uuid dạng STRING, không phải []byte
	UserID    string    `json:"user_id"`
	UserType  uint8     `json:"user_type"`
	Balance   int64     `json:"balance"`    // tiền để số nguyên, KHÔNG dùng float
	Currency  string    `json:"currency"`
	Status    uint8     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

Ba điểm đáng học:

1. **Tag `json` phải khớp CHÍNH XÁC tên field trong mapping.** Sai một ký tự → ES bật dynamic mapping tạo field mới, query không bao giờ khớp.
2. **UUID để `string`, không để `[]byte`.** Go mã hoá `[]byte` sang JSON thành **base64** → không tra cứu bằng mắt được. Đây là lý do mapper có `UUIDToString`.
3. **Document model tách rời Entity** — chỉ mang field cần cho tìm kiếm. Nhẹ index = nhanh.

### B. Mapping

```go
// elasticsearch.go:118 — EnsureIndices()
"id":           { "type": "keyword" },
"user_id":      { "type": "keyword" },
"user_type":    { "type": "byte" },
"balance":      { "type": "long" },
"created_at":   { "type": "date" }
// transactions:
"reference_id": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
"description":  { "type": "text", "analyzer": "standard" }
```

Đối chiếu với [mục 3.1](#31-analysis-pipeline--nơi-mọi-thứ-bắt-đầu) và [3.2](#32-cấu-trúc-lưu-trữ--các-thuật-toán-nén): `wallet_id` để `keyword` nên `term` query khớp chuẩn; `reference_id` dùng multi-field nên vừa tìm mò vừa lọc chính xác được.

`createIndexIfNotExists` là **idempotent** — chạy lại bao nhiêu lần cũng an toàn.

**Bài học về error handling của client ES (rất dễ sai):**

```go
if err != nil    { ... }  // ← CHỈ bắt lỗi TẦNG MẠNG (đứt cáp, sai host, timeout)
if res.IsError() { ... }  // ← bắt lỗi NGHIỆP VỤ (HTTP 4xx/5xx: sai mapping, 401)
```

> ES trả HTTP 400 "mapping sai bét" thì `err` vẫn **`nil`**! Phải check `res.IsError()` mới bắt được. **Luôn check cả hai**, và luôn `defer res.Body.Close()` — bỏ sót là rò rỉ connection.

### C. `doSearch[T]` — generic helper

```go
// elasticsearch.go:327
func doSearch[T any](ctx context.Context, client *elasticsearch.Client, index string,
                     query map[string]any, page, pageSize int) (*SearchResult[T], error)
```

Dùng Go generics nên **một hàm phục vụ cả `WalletDocument` lẫn `TransactionDocument`** mà vẫn type-safe. Thêm index mới chỉ cần viết thêm `buildXxxQuery()`.

Cấu trúc response ES cần thuộc:

```json
{
  "took": 5,
  "hits": {
    "total": { "value": 42, "relation": "eq" },
    "hits": [ { "_id": "...", "_score": 1.2, "_source": { ...document gốc... } } ]
  }
}
```

- `hits.hits[]._source` — document JSON gốc mình đã ghi, đây là thứ duy nhất cần lấy
- `_score` — sẽ là `null` khi đã khai `sort` (ES không buồn tính nữa)
- `total.relation` — `"eq"` (chính xác) hoặc `"gte"` (ít nhất). Mặc định chỉ đếm chính xác tới 10.000; muốn chính xác tuyệt đối phải thêm `"track_total_hits": true`

### D. Graceful degradation trong DI

```go
// di/injection.go:57
esEngine, err := search.NewElasticSearchEngine(esAddresses)
if err != nil {
	log.Printf("Failed to init elasticsearch: %v. Search will be disabled.", err)
	esEngine = nil                    // ← không kill service
} else {
	esEngine.EnsureIndices(ctx)
}
```

Triết lý rất đáng học: **ES sập → nạp/rút/chuyển tiền vẫn chạy bình thường**, chỉ riêng API search trả `codes.Unimplemented`. Với hệ thống tài chính, để dịch vụ phụ trợ kéo sập ví tiền là không chấp nhận được.

Cấu hình qua `WALLET_SERVICE_ES_ADDRESSES` (mặc định `http://localhost:9200`), nhiều node cách nhau bằng dấu phẩy → client tự round-robin và loại node chết.

Chạy ES để nghịch thử (repo **chưa có** service ES trong docker-compose):

```bash
docker run -d --name es-logistic -p 9200:9200 -e "discovery.type=single-node" -e "xpack.security.enabled=false" docker.elastic.co/elasticsearch/elasticsearch:8.19.0
```

## 6.4. Các vấn đề đang tồn tại trong code hiện tại

Xếp theo mức độ nghiêm trọng:

### 🔴 A. Index ES nằm BÊN TRONG transaction MySQL

[`biz/wallet_biz.go:180`](../../wallet_service/internal/biz/wallet_biz.go) (`HoldDeposit`, lặp ở dòng 289, 373):

```go
return uc.uow.Do(ctx, func(ctxTx context.Context) error {
	...
	if uc.esEngine != nil {
		_ = uc.esEngine.IndexWallet(ctx, ...)      // ← 4 lượt HTTP giữa transaction
		_ = uc.esEngine.IndexTransaction(ctx, ...)
	}
	return nil
})
```

1. **Giữ row lock trong lúc chờ mạng.** Bốn round-trip HTTP diễn ra khi các dòng ví **vẫn đang bị `SELECT ... FOR UPDATE` khoá**. Ví escrow là điểm nghẽn chung của mọi giao dịch → ES lag 200ms là khoá thêm 200ms. Công thức tạo deadlock và sập throughput.
2. **Sai lệch dữ liệu khi rollback.** Transaction fail *sau* khi ES đã ghi → MySQL rollback nhưng **ES giữ nguyên giao dịch chưa từng tồn tại**.

**Sửa:** gom document vào slice, đẩy ES **sau khi `uow.Do()` trả `nil`** — đúng như `Deposit` (dòng 99) đang làm.

### 🟠 B. Bộ lọc ngày tháng là code chết

Proto [`wallet_messages.proto:79`](../../api/logistic/wallet_service/v1/wallet_messages.proto) có `from_time`/`to_time`, `buildTransactionQuery` đã viết sẵn logic `range` (dòng 306). Nhưng [`wallet_controller.go:145`](../../wallet_service/internal/controller/wallet_controller.go) **không map `req.FromTime` → `params.From`**. Client gửi khoảng thời gian lên, server lặng lẽ bỏ qua.

### 🟠 C. Nuốt trọn lỗi khi index

```go
_ = uc.esEngine.IndexWallet(ctx, ...)   // lỗi bị vứt sạch, không log
```

ES trả 400 hay đứt mạng — **không ai biết**. ES lệch dần khỏi MySQL không có cảnh báo nào. Tối thiểu phải log + metric; bài bản thì làm **outbox** ([mục 6.1](#61-bài-toán-khó-nhất-đồng-bộ-database--elasticsearch)) — repo đã có sẵn Kafka.

### 🟡 D. Các điểm còn lại

| Vấn đề | Vị trí | Hậu quả |
|---|---|---|
| Ping lúc boot không check `res.IsError()` | `elasticsearch.go:105` | ES trả 401 vẫn log "Connected", mọi lệnh index fail âm thầm |
| `EnsureIndices` bỏ qua error trả về | `di/injection.go:62` | Tạo index fail nhưng service vẫn khởi động |
| Chưa dùng **Bulk API** | `wallet_biz.go` | `RefundContract` bắn 7 request tuần tự thay vì 1 lần `_bulk` |
| Điều kiện lọc để trong `must` | `elasticsearch.go:246-255` | Mất cơ hội cache bitmap ([mục 6.2](#62-query-dsl--bộ-khung-cần-thuộc)) |
| Client thiếu cấu hình production | `elasticsearch.go:95` | Không auth, không TLS, **không timeout transport**, không retry |
| Mapping `uint8` (Go) → `byte` (ES) | `elasticsearch.go:123` | ES `byte` là **có dấu (-128..127)**, Go `uint8` là 0..255. Mã ≥128 sẽ bị ES từ chối → nên đổi sang `short` |
| `doSearch` sửa trực tiếp map đầu vào | `elasticsearch.go:329` | Chưa gây lỗi (map luôn tạo mới) nhưng dễ sinh bug về sau |
| `go.mod` đánh dấu `// indirect` sai | `wallet_service/go.mod:10` | Chạy `go mod tidy` là sạch |
| Chưa khai `number_of_shards`/`replicas` | `elasticsearch.go:118` | Single-node sẽ ở trạng thái **yellow** vĩnh viễn (replica không có chỗ đặt) — bình thường, không phải lỗi |

## 6.5. Vận hành production — checklist

| Hạng mục | Khuyến nghị |
|---|---|
| **Kích cỡ shard** | **10–50GB mỗi shard**. Nhỏ quá = phí tài nguyên quản lý, lớn quá = phục hồi chậm |
| **Số shard** | Bắt đầu ít thôi. Không đổi được, nhưng thừa shard cũng tốn kém. Ước lượng theo dung lượng 1–2 năm tới |
| **JVM Heap** | 50% RAM máy, **và không bao giờ vượt 31GB** — vượt ngưỡng ~32GB, JVM mất cơ chế nén con trỏ (compressed oops), tự nhiên tốn RAM hơn mà chậm đi |
| **Nửa RAM còn lại** | **Để dành cho OS page cache** — đây là thứ giúp Lucene đọc segment nhanh. Đừng cấp hết cho heap |
| **Master node** | **Số lẻ (3)**, tách riêng khỏi data node, để tránh split-brain |
| **ILM (Index Lifecycle Mgmt)** | Với dữ liệu theo thời gian: hot → warm → cold → delete. Tự động xoay vòng index |
| **Force merge** | Chỉ cho index **đã ngừng ghi** |
| **Snapshot** | Sao lưu định kỳ ra S3 — dù ES chỉ là read model thì re-index toàn bộ vẫn rất tốn thời gian |
| **Giám sát** | JVM heap %, số segment, độ trễ merge, độ trễ ghi, tỉ lệ reject của thread pool |

---
---

# PHẦN VII — ƯU ĐIỂM

| # | Ưu điểm | Giải thích |
|---|---|---|
| 1 | **Tốc độ full-text phi thường** | Inverted index + FST + nén postings → tìm trong hàng trăm triệu document vẫn dưới 100ms |
| 2 | **Xếp hạng theo độ liên quan** | BM25 sẵn dùng, tuỳ chỉnh sâu được (boost, function_score, rescore) |
| 3 | **Mở rộng ngang thực thụ** | Thêm node là tự cân bằng shard, gần như không cần can thiệp |
| 4 | **Aggregation cực mạnh** | Gom nhóm nhiều tầng trên khối dữ liệu lớn trong mili-giây nhờ doc values |
| 5 | **Schema linh hoạt** | Dynamic mapping cho phép ném JSON vào là chạy — rất hợp với log |
| 6 | **Hệ sinh thái trưởng thành** | Kibana, Beats, Logstash, APM, hàng trăm plugin, cộng đồng khổng lồ |
| 7 | **API thuần HTTP/JSON** | Ngôn ngữ nào cũng gọi được, debug bằng `curl` |
| 8 | **Sẵn sàng cao** | Replica + tự bầu master + tự phục hồi shard |
| 9 | **Đa năng vượt tìm kiếm** | Log, metric, APM, SIEM, geo, vector/RAG trong cùng một hệ |
| 10 | **Near real-time đủ tốt** | ~1 giây là chấp nhận được với hầu hết nghiệp vụ |

---
---

# PHẦN VIII — NHƯỢC ĐIỂM & CẠM BẪY

| # | Nhược điểm | Chi tiết & cách giảm đau |
|---|---|---|
| 1 | **Không ACID, không transaction đa document** | Không bao giờ dùng làm nguồn chân lý cho dữ liệu tài chính. → Luôn để sau một database thật |
| 2 | **Mapping gần như không sửa được** | Đổi kiểu field = phải tạo index mới + `_reindex` toàn bộ. → Dùng **alias** ngay từ đầu để đổi index không cần sửa code |
| 3 | **Ngốn RAM khủng khiếp** | JVM heap + page cache. Cluster nghiêm túc cần ít nhất 8–16GB mỗi node. → Không bao giờ vượt 31GB heap |
| 4 | **Deep pagination sập** | `from + size > 10.000` là lỗi. → `search_after` + PIT |
| 5 | **Đồng bộ dữ liệu là bài toán riêng** | Dual write sẽ lệch dữ liệu. → Outbox hoặc CDC + đối soát định kỳ |
| 6 | **Eventual consistency** | Ghi xong ~1s mới thấy; replica còn trễ hơn. → Thiết kế UI chấp nhận điều này |
| 7 | **Không có JOIN** | `nested` tạo document Lucene riêng cho mỗi phần tử (chậm, tốn); `join` field bắt buộc cùng shard và query rất chậm. → **Denormalize**, chấp nhận trùng dữ liệu |
| 8 | **Mapping explosion** | Dynamic mapping gặp JSON có key động (`user_123: ...`) sẽ đẻ ra hàng nghìn field, giới hạn mặc định 1000 field rồi vỡ. → Tắt dynamic mapping cho index nghiệp vụ |
| 9 | **Chi phí vận hành cao** | Cần người biết tune JVM, đọc hot threads, xử lý shard đỏ. → Cân nhắc dịch vụ managed |
| 10 | **Aggregation xấp xỉ** | `cardinality` (HyperLogLog++) và `percentiles` (T-Digest) **không chính xác tuyệt đối**. → Số liệu đối soát tài chính phải hỏi database |
| 11 | **Số shard chốt cứng** | Chọn sai lúc đầu là phải reindex. → Ước lượng kỹ, dùng Split/Shrink API nếu buộc phải đổi |
| 12 | **Vấn đề giấy phép** | Từ 2021 không còn thuần Apache 2.0; có nhánh OpenSearch song song. → Kiểm tra ràng buộc pháp lý trước khi bán sản phẩm |
| 13 | **Ghi chậm hơn database** | Mỗi document phải qua analyze + đảo index. Ghi nhiều nhỏ lẻ rất tệ. → **Luôn dùng `_bulk`** |
| 14 | **Bảo mật mặc định lỏng** | Bản cũ mở toang không auth (nguyên nhân của vô số vụ lộ dữ liệu). ES 8.x đã bật security mặc định. → **Không bao giờ để cổng 9200 lộ ra Internet** |

---
---

# PHẦN IX — Lộ trình học & tra cứu nhanh

## 9.1. Lộ trình cho người mới

| Chặng | Mục tiêu | Việc cần làm |
|---|---|---|
| **1. Cảm nhận** | Chạy được, ném dữ liệu vào, tìm ra | Docker chạy ES + Kibana, dùng Dev Tools gõ thử |
| **2. Analysis** | Hiểu vì sao query không ra kết quả | Nghịch API `_analyze` cho chán. Nắm chắc `text` vs `keyword` |
| **3. Mapping** | Thiết kế index đúng ngay từ đầu | Tự khai mapping, cố tình khai sai để xem lỗi thế nào |
| **4. Query DSL** | Viết được truy vấn thật | `bool` + `must`/`filter`, `match` vs `term`, `range` |
| **5. Relevance** | Hiểu vì sao kết quả này đứng trên | Dùng API `_explain` xem ES chấm điểm ra sao |
| **6. Aggregation** | Làm được facet, dashboard | `terms`, `date_histogram`, agg lồng nhau |
| **7. Vận hành** | Không sợ cluster đỏ | Shard, replica, `_cat` API, ILM, snapshot |
| **8. Nâng cao** | | `search_after`, percolator, vector/kNN, hybrid search |

**Ba API nên xem trước:**

| API | Trả lời câu hỏi |
|---|---|
| `_analyze` | "Câu này bị cắt thành token gì?" — **giải 90% ca query không ra kết quả** |
| `_explain` | "Vì sao document này được điểm 3.7?" |
| `_cat/*` | "Cluster đang khoẻ không? Shard nằm đâu?" |

## 9.2. Cheat sheet

**Xem tình trạng cluster & index**
```bash
curl "localhost:9200/_cat/indices?v&s=store.size:desc"
```

**Xem mapping của index**
```bash
curl "localhost:9200/wallets/_mapping?pretty"
```

**Xem một câu bị cắt token ra sao — dùng nhiều nhất**
```bash
curl "localhost:9200/_analyze?pretty" -H 'Content-Type: application/json' -d '{"analyzer":"standard","text":"Đóng băng tiền cọc nhận cuốc"}'
```

**Truy vấn đầy đủ: full-text + filter + sort + phân trang**
```bash
curl "localhost:9200/transactions/_search?pretty" -H 'Content-Type: application/json' -d '{"query":{"bool":{"must":[{"multi_match":{"query":"tiền cọc","fields":["reference_id","description"],"type":"phrase_prefix"}}],"filter":[{"term":{"status":1}},{"range":{"created_at":{"gte":"2026-01-01T00:00:00Z"}}}]}},"sort":[{"created_at":{"order":"desc"}}],"from":0,"size":10}'
```

**Hỏi ES vì sao document này được chấm điểm như vậy**
```bash
curl "localhost:9200/transactions/_explain/DOC_ID?pretty" -H 'Content-Type: application/json' -d '{"query":{"match":{"description":"tiền cọc"}}}'
```

**Aggregation: đếm giao dịch theo ngày**
```bash
curl "localhost:9200/transactions/_search?pretty" -H 'Content-Type: application/json' -d '{"size":0,"aggs":{"theo_ngay":{"date_histogram":{"field":"created_at","calendar_interval":"day"}}}}'
```

## 9.3. Hai mươi điều cần khắc cốt ghi tâm

**Về bản chất**
1. ES là **read model**, database mới là chân lý. Mất ES = re-index, không mất dữ liệu.
2. ES = **Lucene** (lõi tìm kiếm) + lớp **phân tán** + **REST API**.
3. ES **không có transaction đa document**, không có JOIN đúng nghĩa.

**Về mapping & analysis**
4. **`term` đi với `keyword`, `match` đi với `text`.** Ghép chéo = 0 kết quả.
5. Câu query phải qua **cùng analyzer** với lúc ghi.
6. **Mapping đã tạo thì không sửa được** — phải reindex. Dùng **alias** ngay từ đầu.
7. Không sort/aggregate được trên `text` — phải có sub-field `keyword`.
8. Dynamic mapping tiện nhưng nguy hiểm — tắt đi với index nghiệp vụ.

**Về nguyên lý bên trong**
9. **Segment là bất biến.** Xoá = tombstone, sửa = xoá + chèn mới.
10. **Refresh 1 giây** = Near Real-Time. `refresh:"true"` chỉ dùng trong test.
11. **Translog** giữ cho dữ liệu không mất giữa hai lần flush.
12. **BM25** có bão hoà TF nên chống spam từ khoá tốt hơn TF-IDF.
13. **Doc values** (cột) phục vụ sort/agg, **inverted index** phục vụ tìm kiếm.
14. Số **primary shard không đổi được** vì routing dùng phép chia lấy dư.

**Về code**
15. Luôn check **cả `err` lẫn `res.IsError()`**, và luôn `defer res.Body.Close()`.
16. **`DocumentID` = ID entity** → index trở nên idempotent.
17. Điều kiện lọc bỏ vào **`filter`**, không phải `must` — được cache.
18. **`from + size` trần 10.000** — sâu hơn thì dùng `search_after`.
19. **Luôn dùng `_bulk`** khi ghi nhiều document.
20. **Tránh gọi ES bên trong transaction database.**
