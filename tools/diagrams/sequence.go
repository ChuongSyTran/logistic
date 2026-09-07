package main

// Mô hình SEQUENCE DIAGRAM theo UML, dùng cho docs/flows/.
//
// Vì sao flow phải là sequence chứ không phải sơ đồ hộp: một nghiệp vụ có THỨ TỰ
// THỜI GIAN. Sơ đồ hộp - mũi tên diễn tả được "ai nối với ai" nhưng làm mất ba
// thứ quan trọng nhất của một luồng:
//
//	1. Thứ tự các bước (bước nào trước, bước nào sau).
//	2. Phân biệt gọi đồng bộ / giá trị trả về / phát sự kiện bất đồng bộ.
//	3. Nhánh rẽ (alt/else) và vòng lặp.
//
// Sơ đồ cấu trúc bên trong một service (docs/services/) thì ngược lại — ở đó
// không có trục thời gian, nên vẫn dùng sơ đồ thành phần trong spec.go.

// MsgKind quyết định kiểu nét vẽ và đầu mũi tên, theo quy ước UML.
type MsgKind int

const (
	// Sync: gọi đồng bộ, chờ kết quả. Nét liền, đầu mũi tên đặc.
	Sync MsgKind = iota
	// Return: giá trị trả về. Nét đứt, đầu mũi tên mảnh.
	Return
	// Async: phát rồi đi tiếp, không chờ. Nét liền, đầu mũi tên mảnh.
	Async
	// Self: tự gọi chính mình (tính toán nội bộ, kiểm tra điều kiện).
	Self
)

type Lifeline struct {
	ID    string
	Label string // "\n" để xuống dòng
	Kind  Kind
}

type Message struct {
	From, To string
	Label    string
	Kind     MsgKind
	// Note hiện thành ghi chú nhỏ bên phải message, dùng cho chi tiết kỹ thuật
	// không đáng đưa vào nhãn mũi tên.
	Note string
}

// Fragment là khung alt/opt/loop bao quanh một dải message (đánh số từ 0).
//
// From/To là chỉ số message ĐẦU và CUỐI nằm trong khung (bao gồm cả hai đầu).
// Else là chỉ số message mở đầu nhánh thứ hai; -1 nghĩa là không có nhánh else.
type Fragment struct {
	Type      string // "alt" | "opt" | "loop" | "par"
	Label     string // điều kiện của nhánh thứ nhất
	ElseLabel string // điều kiện của nhánh else
	From, To  int
	Else      int
}

type Sequence struct {
	Name      string // trùng tên gốc với file .md
	Title     string
	Caption   string
	Lifelines []Lifeline
	Messages  []Message
	Fragments []Fragment
}

// sequences là toàn bộ sequence diagram của repo — mỗi tài liệu trong
// docs/flows/ có đúng một mục ở đây.
func sequences() []Sequence {
	return []Sequence{
		seqMatchingNotification(),
		seqDriverOnboarding(),
		seqShipperOrder(),
		seqDriverLocation(),
		seqAuthentication(),
		seqErrorHandling(),
	}
}

// ---------------------------------------------------------------------------

func seqMatchingNotification() Sequence {
	return Sequence{
		Name:    "matching-notification-flow",
		Title:   "Luồng ghép xe và thông báo",
		Caption: "Chủ hàng nhận phản hồi ngay ở bước 9; phần thông báo chạy bất đồng bộ sau đó.",
		Lifelines: []Lifeline{
			{ID: "shipper", Label: "Chủ hàng", Kind: KindClient},
			{ID: "gw", Label: "gateway_service", Kind: KindGateway},
			{ID: "matching", Label: "matching_service", Kind: KindService},
			{ID: "pgM", Label: "Postgres\n(matching)", Kind: KindStore},
			{ID: "mq", Label: "RabbitMQ\nlogistic.events", Kind: KindBroker},
			{ID: "notif", Label: "notification_service", Kind: KindService},
			{ID: "pgN", Label: "Postgres\n(notification)", Kind: KindStore},
			{ID: "redis", Label: "Redis\nbộ đếm chưa đọc", Kind: KindStore},
			{ID: "driver", Label: "Tài xế", Kind: KindClient},
		},
		Messages: []Message{
			{From: "shipper", To: "gw", Label: "POST /api/v1/matching/bids", Kind: Sync},
			{From: "gw", To: "matching", Label: "SubmitBid (gRPC)", Kind: Sync},
			{From: "matching", To: "pgM", Label: "INSERT bids (PENDING)", Kind: Sync},
			{From: "pgM", To: "matching", Label: "bid_id", Kind: Return},
			{From: "matching", To: "pgM", Label: "FindAskForBid", Kind: Sync, Note: "cùng zone · đủ tải · giá ≤ max_price"},
			{From: "pgM", To: "matching", Label: "[]Ask (ứng viên)", Kind: Return},
			{From: "matching", To: "mq", Label: "publish matching.driver.candidates_found", Kind: Async},
			{From: "matching", To: "gw", Label: "bid_id, status", Kind: Return},
			{From: "gw", To: "shipper", Label: "200 { bid_id }", Kind: Return},

			{From: "mq", To: "notif", Label: "giao message (prefetch 20)", Kind: Async},
			{From: "notif", To: "pgN", Label: "GetOrCreatePreference (mỗi tài xế)", Kind: Sync},
			{From: "pgN", To: "notif", Label: "NotificationPreference", Kind: Return},
			{From: "notif", To: "notif", Label: "lọc theo cài đặt + giờ yên lặng", Kind: Self},

			{From: "notif", To: "pgN", Label: "BEGIN · INSERT processed_events", Kind: Sync},
			{From: "notif", To: "pgN", Label: "INSERT notifications (bulk) · COMMIT", Kind: Sync},
			{From: "notif", To: "redis", Label: "DEL unread:<user_id>", Kind: Async},
			{From: "notif", To: "mq", Label: "ACK", Kind: Async},
			{From: "notif", To: "driver", Label: "push \"có đơn hàng phù hợp gần bạn\"", Kind: Async},

			{From: "notif", To: "mq", Label: "ACK (bỏ qua, không tạo gì)", Kind: Async},
		},
		Fragments: []Fragment{
			{
				Type:      "alt",
				Label:     "event_id chưa có trong processed_events",
				ElseLabel: "event_id đã xử lý — broker giao lại",
				From:      13, To: 18, Else: 18,
			},
		},
	}
}

func seqDriverOnboarding() Sequence {
	return Sequence{
		Name:    "driver-onboarding-flow",
		Title:   "Luồng tài xế gia nhập hệ thống",
		Caption: "Năm bước, hai lần admin duyệt. Xe chỉ vào chỉ mục tìm kiếm ở bước cuối.",
		Lifelines: []Lifeline{
			{ID: "driver", Label: "Tài xế", Kind: KindClient},
			{ID: "gw", Label: "gateway_service", Kind: KindGateway},
			{ID: "user", Label: "user_service", Kind: KindService},
			{ID: "vehicle", Label: "vehicle_service", Kind: KindService},
			{ID: "admin", Label: "Admin", Kind: KindClient},
			{ID: "redis", Label: "Redis GEO", Kind: KindStore},
		},
		Messages: []Message{
			{From: "driver", To: "gw", Label: "POST /auth/register (role=driver)", Kind: Sync},
			{From: "gw", To: "user", Label: "RegisterUser(id = id danh tính)", Kind: Sync},
			{From: "user", To: "user", Label: "tạo users + driver_profiles RỖNG", Kind: Self, Note: "license/id_card = NULL, không phải \"\""},
			{From: "user", To: "gw", Label: "user_id", Kind: Return},
			{From: "gw", To: "driver", Label: "201 Created", Kind: Return},

			{From: "driver", To: "gw", Label: "PUT /users/{id}/driver-profile", Kind: Sync},
			{From: "gw", To: "user", Label: "UpdateDriverProfile", Kind: Sync},
			{From: "user", To: "user", Label: "mustBeRole(driver)", Kind: Self},
			{From: "user", To: "gw", Label: "kyc_status = pending", Kind: Return},

			{From: "admin", To: "gw", Label: "GET /admin/kyc/pending", Kind: Sync},
			{From: "admin", To: "gw", Label: "PUT /admin/kyc/{id}/review", Kind: Sync},
			{From: "gw", To: "user", Label: "AdminReviewKYC", Kind: Sync},
			{From: "user", To: "admin", Label: "422 KYC_ALREADY_REVIEWED", Kind: Return},
			{From: "user", To: "admin", Label: "kyc_status = approved", Kind: Return},

			{From: "driver", To: "gw", Label: "POST /vehicles", Kind: Sync},
			{From: "gw", To: "vehicle", Label: "RegisterVehicle", Kind: Sync},
			{From: "vehicle", To: "driver", Label: "verification_status = pending", Kind: Return},
			{From: "driver", To: "gw", Label: "POST /vehicles/{id}/documents", Kind: Sync},
			{From: "admin", To: "gw", Label: "PUT /admin/vehicles/{id}/verify", Kind: Sync},
			{From: "gw", To: "vehicle", Label: "AdminVerifyVehicle → verified", Kind: Sync},

			{From: "driver", To: "gw", Label: "POST /drivers/{id}/availability", Kind: Sync},
			{From: "gw", To: "vehicle", Label: "SetDriverAvailability(is_online=true)", Kind: Sync},
			{From: "vehicle", To: "vehicle", Label: "kiểm: đúng chủ · đã duyệt · không bảo dưỡng · toạ độ hợp lệ", Kind: Self},
			{From: "vehicle", To: "redis", Label: "GEOADD geo:online <vehicle_id>", Kind: Sync},
			{From: "vehicle", To: "driver", Label: "200 — sẵn sàng nhận đơn", Kind: Return},
			{From: "vehicle", To: "driver", Label: "422 VEHICLE_NOT_VERIFIED", Kind: Return},
		},
		Fragments: []Fragment{
			{Type: "alt", Label: "hồ sơ đã duyệt trước đó", ElseLabel: "còn pending", From: 12, To: 13, Else: 13},
			{Type: "alt", Label: "đạt cả 4 điều kiện", ElseLabel: "không đạt", From: 23, To: 25, Else: 25},
		},
	}
}

func seqShipperOrder() Sequence {
	return Sequence{
		Name:    "shipper-order-flow",
		Title:   "Luồng đặt đơn tới khi chốt xe",
		Caption: "Vòng thương lượng: bid → offer → accept hoặc reject. Cọc được kiểm trước, đóng băng sau.",
		Lifelines: []Lifeline{
			{ID: "shipper", Label: "Chủ hàng", Kind: KindClient},
			{ID: "gw", Label: "gateway_service", Kind: KindGateway},
			{ID: "matching", Label: "matching_service", Kind: KindService},
			{ID: "wallet", Label: "wallet_service", Kind: KindService},
			{ID: "nats", Label: "NATS JetStream", Kind: KindBroker},
			{ID: "kafka", Label: "Kafka", Kind: KindBroker},
			{ID: "mq", Label: "RabbitMQ", Kind: KindBroker},
			{ID: "notif", Label: "notification_service", Kind: KindService},
			{ID: "driver", Label: "Tài xế", Kind: KindClient},
		},
		Messages: []Message{
			{From: "shipper", To: "gw", Label: "POST /matching/bids", Kind: Sync},
			{From: "gw", To: "matching", Label: "SubmitBid", Kind: Sync},
			{From: "matching", To: "mq", Label: "matching.driver.candidates_found", Kind: Async},
			{From: "mq", To: "notif", Label: "giao message", Kind: Async},
			{From: "notif", To: "driver", Label: "push \"có đơn gần bạn\"", Kind: Async},

			{From: "driver", To: "gw", Label: "POST /matching/offers (giá 3.2tr)", Kind: Sync},
			{From: "gw", To: "matching", Label: "SubmitOffer", Kind: Sync},
			{From: "matching", To: "nats", Label: "publish matching.offers.{bid_id}", Kind: Async},
			{From: "nats", To: "matching", Label: "consumer → ProcessOfferQueue", Kind: Async, Note: "hàng đợi để nhiều báo giá cùng đơn xử lý tuần tự"},
			{From: "matching", To: "matching", Label: "bid.status = NEGOTIATING", Kind: Self, Note: "khoá mềm: tài xế khác bị từ chối ngay"},
			{From: "matching", To: "mq", Label: "matching.offer.received", Kind: Async},
			{From: "mq", To: "notif", Label: "giao message", Kind: Async},
			{From: "notif", To: "shipper", Label: "push \"có báo giá mới\"", Kind: Async},

			{From: "shipper", To: "gw", Label: "POST /matching/offers/reject", Kind: Sync},
			{From: "gw", To: "matching", Label: "RejectOffer", Kind: Sync},
			{From: "matching", To: "matching", Label: "bid.status = PENDING (mở lại)", Kind: Self},
			{From: "matching", To: "mq", Label: "matching.offer.rejected", Kind: Async},

			{From: "shipper", To: "gw", Label: "POST /matching/matches/accept", Kind: Sync},
			{From: "gw", To: "matching", Label: "AcceptMatch", Kind: Sync},
			{From: "matching", To: "wallet", Label: "CheckBalance(shipper_id)", Kind: Sync},
			{From: "wallet", To: "matching", Label: "balance", Kind: Return},
			{From: "matching", To: "shipper", Label: "422 INSUFFICIENT_BALANCE", Kind: Return},
			{From: "matching", To: "kafka", Label: "wallet.hold_deposit (đóng băng cọc 10%)", Kind: Async},
			{From: "matching", To: "matching", Label: "INSERT matches → bid/ask = MATCHED", Kind: Self, Note: "ghi hợp đồng trước, lật trạng thái sau"},
			{From: "matching", To: "mq", Label: "matching.match.found", Kind: Async},
			{From: "mq", To: "notif", Label: "giao message", Kind: Async},
			{From: "notif", To: "shipper", Label: "push \"đã tìm được xe\"", Kind: Async},
			{From: "notif", To: "driver", Label: "push \"bạn nhận được đơn\"", Kind: Async},
		},
		Fragments: []Fragment{
			{Type: "alt", Label: "chủ hàng từ chối giá", ElseLabel: "chủ hàng chốt xe", From: 13, To: 27, Else: 17},
			{Type: "alt", Label: "số dư không đủ", ElseLabel: "đủ tiền cọc", From: 21, To: 27, Else: 22},
		},
	}
}

func seqDriverLocation() Sequence {
	return Sequence{
		Name:    "driver-location-flow",
		Title:   "Luồng GPS và tìm xe đang chạy",
		Caption: "Phần trên: tài xế ping GPS. Phần dưới: matching hỏi xe quanh điểm lấy hàng.",
		Lifelines: []Lifeline{
			{ID: "app", Label: "App tài xế", Kind: KindClient},
			{ID: "nginx", Label: "Nginx", Kind: KindEdge},
			{ID: "gw", Label: "gateway_service", Kind: KindGateway},
			{ID: "vehicle", Label: "vehicle_service", Kind: KindService},
			{ID: "pg", Label: "Postgres", Kind: KindStore},
			{ID: "redis", Label: "Redis GEO", Kind: KindStore},
			{ID: "matching", Label: "matching_service", Kind: KindService},
		},
		Messages: []Message{
			{From: "app", To: "nginx", Label: "POST /vehicles/{id}/location", Kind: Sync, Note: "vài giây một lần, mỗi xe"},
			{From: "nginx", To: "nginx", Label: "limit_req gps_zone 60r/s", Kind: Self, Note: "ngưỡng riêng, cao gấp đôi API thường"},
			{From: "nginx", To: "gw", Label: "proxy_pass", Kind: Sync},
			{From: "gw", To: "vehicle", Label: "ReportLocation", Kind: Sync},
			{From: "vehicle", To: "vehicle", Label: "IsValidCoordinate + ComputeZoneID", Kind: Self},
			{From: "vehicle", To: "app", Label: "400 INVALID_COORDINATE", Kind: Return},

			{From: "vehicle", To: "pg", Label: "UPSERT vehicle_locations (ghi đè 1 dòng)", Kind: Sync},
			{From: "vehicle", To: "pg", Label: "UPDATE driver_availabilities (toạ độ, zone)", Kind: Sync},
			{From: "vehicle", To: "pg", Label: "SELECT is_online", Kind: Sync},
			{From: "pg", To: "vehicle", Label: "is_online", Kind: Return},
			{From: "vehicle", To: "redis", Label: "GEOADD geo:online", Kind: Async, Note: "chỉ khi tài xế đang bật nhận đơn"},
			{From: "vehicle", To: "app", Label: "200 { zone_id }", Kind: Return},

			{From: "matching", To: "vehicle", Label: "SearchNearbyVehicles (gRPC)", Kind: Sync},
			{From: "vehicle", To: "redis", Label: "GEOSEARCH bán kính, limit×3", Kind: Sync},
			{From: "redis", To: "vehicle", Label: "[(vehicle_id, khoảng cách)] gần→xa", Kind: Return},
			{From: "vehicle", To: "pg", Label: "SELECT vehicles WHERE id IN (...)", Kind: Sync, Note: "gộp 1 câu, tránh N+1"},
			{From: "vehicle", To: "pg", Label: "SELECT availabilities WHERE vehicle_id IN (...)", Kind: Sync},
			{From: "vehicle", To: "vehicle", Label: "lọc: verified · active · đúng loại · đủ tải", Kind: Self},
			{From: "vehicle", To: "pg", Label: "quét zone lân cận + haversine trong Go", Kind: Sync, Note: "đường dự phòng, chậm hơn"},
			{From: "vehicle", To: "matching", Label: "[]NearbyVehicle", Kind: Return},
		},
		Fragments: []Fragment{
			{Type: "alt", Label: "toạ độ không hợp lệ", ElseLabel: "toạ độ hợp lệ", From: 5, To: 11, Else: 6},
			{Type: "alt", Label: "Redis khả dụng", ElseLabel: "Redis lỗi — fallback", From: 13, To: 18, Else: 18},
		},
	}
}

func seqAuthentication() Sequence {
	return Sequence{
		Name:    "authentication-flow",
		Title:   "Luồng xác thực và phân quyền",
		Caption: "Hai đường đăng nhập, rồi tới bước kiểm quyền khi gọi endpoint quản trị.",
		Lifelines: []Lifeline{
			{ID: "user", Label: "Người dùng", Kind: KindClient},
			{ID: "gw", Label: "gateway_service", Kind: KindGateway},
			{ID: "auth", Label: "auth_service", Kind: KindService},
			{ID: "usersvc", Label: "user_service", Kind: KindService},
			{ID: "google", Label: "Google OAuth2", Kind: KindExternal},
			{ID: "svc", Label: "service nội bộ", Kind: KindService},
		},
		Messages: []Message{
			{From: "user", To: "gw", Label: "POST /auth/register", Kind: Sync},
			{From: "gw", To: "auth", Label: "Register → id danh tính", Kind: Sync},
			{From: "gw", To: "usersvc", Label: "RegisterUser(id = id danh tính)", Kind: Sync, Note: "cùng id thì token và hồ sơ trỏ về một người"},
			{From: "gw", To: "user", Label: "201 Created", Kind: Return},

			{From: "user", To: "gw", Label: "POST /auth/login", Kind: Sync},
			{From: "gw", To: "auth", Label: "Login(email, password)", Kind: Sync},
			{From: "auth", To: "gw", Label: "TokenPair · hoặc Unauthenticated", Kind: Return},
			{From: "gw", To: "user", Label: "200 { access_token } · hoặc 401", Kind: Return},

			{From: "user", To: "gw", Label: "GET /auth/google/login", Kind: Sync},
			{From: "gw", To: "auth", Label: "GetGoogleLoginURL(state)", Kind: Sync},
			{From: "gw", To: "gw", Label: "Set-Cookie oauth_state (HttpOnly, 5 phút)", Kind: Self},
			{From: "gw", To: "user", Label: "307 → accounts.google.com", Kind: Return},
			{From: "user", To: "google", Label: "đăng nhập + đồng ý", Kind: Sync},
			{From: "google", To: "gw", Label: "GET /callback?code=…&state=…", Kind: Sync},
			{From: "gw", To: "gw", Label: "so state trên URL ↔ cookie", Kind: Self, Note: "cookie chỉ có trên máy người khởi tạo"},
			{From: "gw", To: "user", Label: "400 INVALID_OAUTH_STATE", Kind: Return},
			{From: "gw", To: "auth", Label: "GoogleCallback(code)", Kind: Sync},
			{From: "auth", To: "google", Label: "đổi code lấy token (client_secret)", Kind: Sync},
			{From: "auth", To: "gw", Label: "TokenPair", Kind: Return},
			{From: "gw", To: "user", Label: "Set-Cookie + 307 về frontend", Kind: Return},

			{From: "user", To: "gw", Label: "GET /api/v1/admin/users  (Bearer)", Kind: Sync},
			{From: "gw", To: "gw", Label: "IdentityContext: đọc X-User-Id / X-User-Role", Kind: Self},
			{From: "gw", To: "gw", Label: "RequireRole(\"admin\") — gắn ở CẤP GROUP", Kind: Self},
			{From: "gw", To: "user", Label: "403 PERMISSION_DENIED", Kind: Return},
			{From: "gw", To: "svc", Label: "AdminListUsers (gRPC)", Kind: Sync},
			{From: "svc", To: "user", Label: "200 { users, pagination }", Kind: Return},
		},
		Fragments: []Fragment{
			{Type: "alt", Label: "đăng nhập bằng email + mật khẩu", ElseLabel: "đăng nhập bằng Google OAuth2", From: 4, To: 19, Else: 8},
			{Type: "alt", Label: "state lệch hoặc thiếu cookie", ElseLabel: "state khớp", From: 15, To: 19, Else: 16},
			{Type: "alt", Label: "role ≠ admin", ElseLabel: "role = admin", From: 23, To: 25, Else: 24},
		},
	}
}

func seqErrorHandling() Sequence {
	return Sequence{
		Name:    "error-handling-flow",
		Title:   "Luồng lỗi xuyên tầng",
		Caption: "Ví dụ: đăng ký trùng số điện thoại. Bốn lần biến đổi, chi tiết SQL dừng lại ở log.",
		Lifelines: []Lifeline{
			{ID: "client", Label: "Client", Kind: KindClient},
			{ID: "gw", Label: "gateway_service", Kind: KindGateway},
			{ID: "icept", Label: "ErrorInterceptor\n(pkg/middleware)", Kind: KindGateway},
			{ID: "ctrl", Label: "controller", Kind: KindService},
			{ID: "biz", Label: "biz", Kind: KindService},
			{ID: "repo", Label: "repo", Kind: KindService},
			{ID: "pg", Label: "Postgres", Kind: KindStore},
			{ID: "log", Label: "Log nội bộ", Kind: KindNote},
		},
		Messages: []Message{
			{From: "client", To: "gw", Label: "POST /users/register (phone đã tồn tại)", Kind: Sync},
			{From: "gw", To: "icept", Label: "RegisterUser (gRPC)", Kind: Sync},
			{From: "icept", To: "ctrl", Label: "chuyển tiếp", Kind: Sync},
			{From: "ctrl", To: "biz", Label: "RegisterUser(param)", Kind: Sync},
			{From: "biz", To: "repo", Label: "CreateUser(entity)", Kind: Sync},
			{From: "repo", To: "pg", Label: "INSERT INTO users", Kind: Sync},
			{From: "pg", To: "repo", Label: "duplicate key: users_phone_key", Kind: Return},
			{From: "repo", To: "repo", Label: "wrapError → apperr{AlreadyExists, PHONE_ALREADY_USED}", Kind: Self, Note: "đọc tên index để biết CỘT nào trùng"},
			{From: "repo", To: "biz", Label: "*apperr.Error", Kind: Return},
			{From: "biz", To: "ctrl", Label: "return err — KHÔNG bọc thêm", Kind: Return},
			{From: "ctrl", To: "icept", Label: "return err — KHÔNG có status.Errorf", Kind: Return},
			{From: "icept", To: "icept", Label: "status.New(code) + ErrorInfo{Reason}", Kind: Self},
			{From: "icept", To: "log", Label: "câu SQL, tên bảng dừng lại ở đây", Kind: Async},
			{From: "icept", To: "gw", Label: "gRPC ALREADY_EXISTS + ErrorInfo", Kind: Return},
			{From: "gw", To: "gw", Label: "response.Error → HTTP 409", Kind: Self},
			{From: "gw", To: "client", Label: "409 { code: PHONE_ALREADY_USED, request_id }", Kind: Return},
		},
	}
}
