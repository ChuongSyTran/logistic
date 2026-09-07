package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type collection struct {
	Info     info       `json:"info"`
	Auth     *auth      `json:"auth,omitempty"`
	Event    []event    `json:"event,omitempty"`
	Item     []item     `json:"item"`
	Variable []variable `json:"variable,omitempty"`
}

type info struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
}

type auth struct {
	Type   string     `json:"type"`
	Bearer []authAttr `json:"bearer,omitempty"`
}

type authAttr struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type event struct {
	Listen string `json:"listen"`
	Script script `json:"script"`
}

type script struct {
	Type string   `json:"type"`
	Exec []string `json:"exec"`
}

type item struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Auth        *auth    `json:"auth,omitempty"`
	Event       []event  `json:"event,omitempty"`
	Item        []item   `json:"item,omitempty"`
	Request     *request `json:"request,omitempty"`
}

type request struct {
	Method      string   `json:"method"`
	Header      []header `json:"header,omitempty"`
	Body        *body    `json:"body,omitempty"`
	URL         url      `json:"url"`
	Description string   `json:"description,omitempty"`
	Auth        *auth    `json:"auth,omitempty"`
}

type header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type body struct {
	Mode     string      `json:"mode"`
	Raw      string      `json:"raw,omitempty"`
	FormData []formField `json:"formdata,omitempty"`
	Options  *bodyOption `json:"options,omitempty"`
}

type bodyOption struct {
	Raw rawOption `json:"raw"`
}

type rawOption struct {
	Language string `json:"language"`
}

type formField struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Src         string `json:"src,omitempty"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
}

type url struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host"`
	Path  []string `json:"path"`
	Query []query  `json:"query,omitempty"`
}

type query struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type endpoint struct {
	name        string
	method      string
	path        string
	description string
	jsonBody    string
	formData    []formField
	queries     []query
	expect      int
	extraTests  []string
	captures    map[string]string
	noAuth      bool
}

const baseHost = "{{base_url}}"

func main() {
	out := flag.String("o", "logistic.postman_collection.json", "đường dẫn file collection")
	flag.Parse()

	c := buildCollection()

	blob, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	blob = append(blob, 0x0a)

	if err := os.WriteFile(*out, blob, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("đã ghi %s (%d nhóm, %d request)\n", *out, len(c.Item), countRequests(c.Item))
}

func buildCollection() collection {
	return collection{
		Info: info{
			PostmanID:   "8f2c41d0-6b3a-4e19-9d77-0a5c1e2b7f34",
			Name:        "Logistics OS — Gateway API",
			Description: collectionDescription,
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Auth: bearerAuth(),
		Event: []event{{
			Listen: "prerequest",
			Script: script{Type: "text/javascript", Exec: []string{
				"const required = ['base_url'];",
				"required.forEach(function (key) {",
				"    if (!pm.variables.get(key)) {",
				"        throw new Error('Thiếu biến ' + key + '. Chọn environment \"Logistics OS — Local\" trước khi chạy.');",
				"    }",
				"});",
			}},
		}},
		Variable: []variable{
			{Key: "base_url", Value: "http://localhost:8080", Type: "string"},
			{Key: "access_token", Value: "", Type: "string"},
			{Key: "refresh_token", Value: "", Type: "string"},
			{Key: "admin_access_token", Value: "", Type: "string"},
			{Key: "user_id", Value: "", Type: "string"},
			{Key: "address_id", Value: "", Type: "string"},
			{Key: "device_id", Value: "", Type: "string"},
			{Key: "vehicle_id", Value: "", Type: "string"},
			{Key: "document_id", Value: "", Type: "string"},
			{Key: "notification_id", Value: "", Type: "string"},
			{Key: "template_id", Value: "", Type: "string"},
			{Key: "bid_id", Value: "", Type: "string"},
			{Key: "ask_id", Value: "", Type: "string"},
			{Key: "driver_id", Value: "", Type: "string"},
			{Key: "public_id", Value: "", Type: "string"},
			{Key: "test_email", Value: "shipper.demo@logistic.vn", Type: "string"},
			{Key: "test_password", Value: "Matkhau@12345", Type: "string"},
			{Key: "test_phone", Value: "0900000001", Type: "string"},
		},
		Item: []item{
			folder("00 — Sức khoẻ hệ thống", healthEndpoints(), healthDescription),
			folder("01 — Auth", authEndpoints(), authDescription),
			folder("02 — User", userEndpoints(), userDescription),
			folder("03 — Sổ địa chỉ", addressEndpoints(), addressDescription),
			folder("04 — Thiết bị nhận push", deviceEndpoints(), deviceDescription),
			folder("05 — Phương tiện", vehicleEndpoints(), vehicleDescription),
			folder("06 — Giấy tờ xe", vehicleDocEndpoints(), vehicleDocDescription),
			folder("07 — Vị trí & nhận đơn", locationEndpoints(), locationDescription),
			folder("08 — Matching", matchingEndpoints(), matchingDescription),
			folder("09 — Thông báo", notificationEndpoints(), notificationDescription),
			folder("10 — Media", mediaEndpoints(), mediaDescription),
			adminFolder(),
			folder("99 — Kiểm thử bảo mật", securityEndpoints(), securityDescription),
		},
	}
}

func countRequests(items []item) int {
	n := 0
	for _, it := range items {
		if it.Request != nil {
			n++
		}
		n += countRequests(it.Item)
	}
	return n
}

func bearerAuth() *auth {
	return &auth{Type: "bearer", Bearer: []authAttr{
		{Key: "token", Value: "{{access_token}}", Type: "string"},
	}}
}

func adminAuth() *auth {
	return &auth{Type: "bearer", Bearer: []authAttr{
		{Key: "token", Value: "{{admin_access_token}}", Type: "string"},
	}}
}

func noAuth() *auth {
	return &auth{Type: "noauth"}
}

func folder(name string, eps []endpoint, description string) item {
	return item{Name: name, Description: description, Item: toItems(eps)}
}

func adminFolder() item {
	return item{
		Name:        "11 — Admin",
		Description: adminDescription,
		Auth:        adminAuth(),
		Item: []item{
			folder("Người dùng", adminUserEndpoints(), ""),
			folder("KYC", adminKycEndpoints(), ""),
			folder("Phương tiện", adminVehicleEndpoints(), ""),
			folder("Giấy tờ xe", adminDocEndpoints(), ""),
			folder("Thông báo", adminNotificationEndpoints(), ""),
			folder("Mẫu thông báo", adminTemplateEndpoints(), ""),
		},
	}
}

func toItems(eps []endpoint) []item {
	items := make([]item, 0, len(eps))
	for _, ep := range eps {
		items = append(items, toItem(ep))
	}
	return items
}

func toItem(ep endpoint) item {
	it := item{
		Name:    ep.name,
		Request: buildRequest(ep),
		Event:   []event{{Listen: "test", Script: script{Type: "text/javascript", Exec: buildTests(ep)}}},
	}
	return it
}

func buildRequest(ep endpoint) *request {
	segments := strings.Split(strings.TrimPrefix(ep.path, "/"), "/")

	raw := baseHost + ep.path
	if len(ep.queries) > 0 {
		parts := make([]string, 0, len(ep.queries))
		for _, q := range ep.queries {
			if q.Disabled {
				continue
			}
			parts = append(parts, q.Key+"="+q.Value)
		}
		if len(parts) > 0 {
			raw += "?" + strings.Join(parts, "&")
		}
	}

	req := &request{
		Method:      ep.method,
		Description: ep.description,
		URL: url{
			Raw:   raw,
			Host:  []string{baseHost},
			Path:  segments,
			Query: ep.queries,
		},
	}

	if ep.noAuth {
		req.Auth = noAuth()
	}

	switch {
	case ep.jsonBody != "":
		req.Header = []header{{Key: "Content-Type", Value: "application/json", Type: "text"}}
		req.Body = &body{
			Mode:    "raw",
			Raw:     ep.jsonBody,
			Options: &bodyOption{Raw: rawOption{Language: "json"}},
		}
	case len(ep.formData) > 0:
		req.Body = &body{Mode: "formdata", FormData: ep.formData}
	}

	return req
}

func buildTests(ep endpoint) []string {
	expect := ep.expect
	if expect == 0 {
		expect = 200
	}

	lines := []string{
		fmt.Sprintf("pm.test('mã trạng thái là %d', function () {", expect),
		fmt.Sprintf("    pm.response.to.have.status(%d);", expect),
		"});",
		"",
		"pm.test('phản hồi là JSON đúng khuôn', function () {",
		"    const json = pm.response.json();",
		"    if (pm.response.code >= 400) {",
		"        pm.expect(json).to.have.property('error');",
		"        pm.expect(json.error).to.have.property('code');",
		"    } else {",
		"        pm.expect(json).to.be.an('object');",
		"    }",
		"});",
		"",
		"pm.test('có mã truy vết để tra log và Jaeger', function () {",
		"    pm.expect(pm.response.headers.has('X-Request-ID')).to.be.true;",
		"});",
	}

	if len(ep.captures) > 0 {
		lines = append(lines, "", "if (pm.response.code < 400) {", "    const json = pm.response.json();")
		for _, v := range sortedKeys(ep.captures) {
			expr := ep.captures[v]
			lines = append(lines,
				fmt.Sprintf("    if (%s) { pm.environment.set('%s', %s); }", expr, v, expr))
		}
		lines = append(lines, "}")
	}

	if len(ep.extraTests) > 0 {
		lines = append(lines, "")
		lines = append(lines, ep.extraTests...)
	}

	return lines
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

const collectionDescription = `Bộ sưu tập đầy đủ cho gateway của Logistics OS: 69 endpoint dưới /api/v1 cộng health check, kèm 5 request kiểm thử bảo mật.

## Chuẩn bị

1. Import cả file này và file environment "Logistics OS — Local".
2. Chọn environment ở góc trên bên phải Postman.
3. Chạy nhóm "01 — Auth > Đăng nhập" một lần: access_token và refresh_token được lưu tự động, mọi request sau tự đính kèm.

## Thứ tự chạy

Các nhóm được đánh số theo đúng trình tự nghiệp vụ, và request sau dùng biến do request trước lưu lại (user_id, vehicle_id, bid_id...). Chạy tuần tự từ 00 tới 99 bằng Collection Runner là đi hết một vòng nghiệp vụ.

Nhóm "11 — Admin" cần biến admin_access_token — đăng nhập bằng một tài khoản có role admin rồi gán tay, vì API công khai không tạo được tài khoản admin.

## Mỗi request kiểm gì

- Mã trạng thái đúng như thiết kế (kể cả các mã lỗi có chủ đích ở nhóm 99).
- Phản hồi đúng khuôn: thành công có "data", lỗi có "error.code".
- Có header X-Request-ID — mã này chính là trace_id trên Jaeger.

File này được SINH RA từ tools/postman. Sửa endpoint thì sửa ở đó rồi chạy "make postman", đừng sửa tay file JSON.`

const healthDescription = "Không cần token. Dùng để kiểm tra gateway đã lên chưa trước khi chạy các nhóm còn lại."

const authDescription = `Nhóm CÔNG KHAI duy nhất — đây là nơi cấp token.

auth_service giữ private key và ký token; gateway chỉ có public key nên verify được mà không ký được.

Chạy "Đăng nhập" trước tiên: nó lưu access_token, refresh_token và user_id vào environment.`

const userDescription = `Hồ sơ người dùng và hồ sơ nghiệp vụ theo vai trò (tài xế / chủ hàng).

Mọi endpoint ở đây chỉ thao tác được trên dữ liệu của CHÍNH mình, trừ khi token có vai trò admin.`

const addressDescription = "Sổ địa chỉ lấy hàng / giao hàng. Có sẵn toạ độ nên lúc tạo đơn không phải geocode lại."

const deviceDescription = "Thiết bị nhận push. notification_service đọc bảng này để biết đẩy thông báo tới đâu."

const vehicleDescription = `Phương tiện của tài xế.

driver_id luôn lấy từ token, không nhận từ body — không ai đăng ký được xe đứng tên người khác.`

const vehicleDocDescription = "Đăng kiểm, bảo hiểm, giấy phép. Xe chỉ được verified khi mọi giấy tờ bắt buộc đã approved và còn hạn."

const locationDescription = `Báo vị trí GPS và bật/tắt nhận đơn.

Các RPC vị trí dùng hạn chờ ngắn hơn mức chung (2s thay vì 5s): bản tin GPS tới muộn thì đã vô nghĩa vì xe đã ở chỗ khác.`

const matchingDescription = `Lõi ghép đơn theo mô hình Bid/Ask.

Trình tự: chủ hàng đăng Bid → tài xế đăng Ask → tài xế báo giá (Offer) → chủ hàng chốt hoặc từ chối.

Chốt xong, matching_service phát sự kiện qua RabbitMQ và notification_service báo cho cả hai bên.`

const notificationDescription = "Hộp thư thông báo. Thông báo do consumer RabbitMQ sinh ra, không có endpoint tạo cho client."

const mediaDescription = "Tải file lên Cloudinary. Nhớ chọn file thật ở tab Body > form-data trước khi gửi."

const adminDescription = `Toàn bộ nhóm này yêu cầu token có vai trò admin.

Đặt biến admin_access_token trước khi chạy. Tài khoản admin không tạo được qua API công khai — dùng công cụ nội bộ hoặc sửa trực tiếp cột role trong database.`

const securityDescription = `Các request CỐ TÌNH sai, dùng để chứng minh lớp bảo vệ hoạt động.

Mọi request ở đây PHẢI thất bại. Nếu có cái nào trả 200 thì đó là lỗ hổng thật.

Chạy nhóm này sau khi đã đăng nhập bằng tài khoản thường (không phải admin).`
