package main

func healthEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Health check",
			method:      "GET",
			path:        "/healthz",
			description: "Không cần token. Docker healthcheck và load balancer gọi endpoint này.",
			noAuth:      true,
			extraTests: []string{
				"pm.test('trạng thái ok', function () {",
				"    pm.expect(pm.response.json().status).to.eql('ok');",
				"});",
			},
		},
		{
			name:        "Swagger UI",
			method:      "GET",
			path:        "/swagger/index.html",
			description: "Tài liệu API sinh từ annotation trong controller. Mở bằng trình duyệt để xem đầy đủ.",
			noAuth:      true,
			extraTests: []string{
				"pm.test('trả về trang HTML', function () {",
				"    pm.expect(pm.response.headers.get('Content-Type')).to.include('text/html');",
				"});",
			},
		},
	}
}

func authEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Đăng ký tài khoản",
			method:      "POST",
			path:        "/api/v1/auth/register",
			description: "Công khai. Vai trò chỉ nhận driver hoặc shipper — admin không tạo được qua API.",
			noAuth:      true,
			expect:      201,
			jsonBody: `{
    "email": "{{test_email}}",
    "full_name": "Nguyễn Văn Chủ Hàng",
    "password": "{{test_password}}",
    "role": "shipper"
}`,
			captures: map[string]string{"user_id": "json.data && json.data.user && json.data.user.id"},
		},
		{
			name:        "Đăng nhập",
			method:      "POST",
			path:        "/api/v1/auth/login",
			description: "Lưu access_token và refresh_token vào environment cho mọi request sau.",
			noAuth:      true,
			jsonBody: `{
    "email": "{{test_email}}",
    "password": "{{test_password}}"
}`,
			captures: map[string]string{
				"access_token":  "json.data && json.data.access_token",
				"refresh_token": "json.data && json.data.refresh_token",
				"user_id":       "json.data && json.data.user && json.data.user.id",
			},
			extraTests: []string{
				"pm.test('token là JWT ba phần', function () {",
				"    const token = pm.response.json().data.access_token;",
				"    pm.expect(token.split('.')).to.have.lengthOf(3);",
				"});",
				"",
				"pm.test('expires_at là mốc thời gian trong tương lai', function () {",
				"    const exp = pm.response.json().data.expires_at;",
				"    pm.expect(exp).to.be.a('number');",
				"    pm.expect(exp * 1000).to.be.above(Date.now());",
				"});",
				"",
				"pm.test('không lộ mật khẩu trong phản hồi', function () {",
				"    pm.expect(pm.response.text()).to.not.include(pm.environment.get('test_password'));",
				"});",
			},
		},
		{
			name:        "Thông tin tài khoản hiện tại",
			method:      "GET",
			path:        "/api/v1/auth/me",
			description: "Cần access token. Trả về profile của chủ nhân token.",
			extraTests: []string{
				"pm.test('id là UUID canonical, không phải base64', function () {",
				"    const id = pm.response.json().data.user.id;",
				"    pm.expect(id).to.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/);",
				"});",
			},
		},
		{
			name:        "Làm mới phiên",
			method:      "POST",
			path:        "/api/v1/auth/refresh",
			description: "Refresh token dùng MỘT LẦN. Mỗi lần gọi trả cặp token mới; token cũ hết hiệu lực ngay.",
			noAuth:      true,
			jsonBody: `{
    "refresh_token": "{{refresh_token}}"
}`,
			captures: map[string]string{
				"access_token":  "json.data && json.data.access_token",
				"refresh_token": "json.data && json.data.refresh_token",
			},
		},
		{
			name:        "Đăng xuất",
			method:      "POST",
			path:        "/api/v1/auth/logout",
			description: "Thu hồi phiên refresh. Access token đang cầm vẫn sống tới khi hết hạn (tối đa 15 phút).",
			jsonBody: `{
    "refresh_token": "{{refresh_token}}"
}`,
		},
		{
			name:        "Google OAuth2 — lấy URL đăng nhập",
			method:      "GET",
			path:        "/api/v1/auth/google/login",
			description: "Trả redirect 307 sang màn hình consent của Google, kèm cookie oauth_state chống CSRF.",
			noAuth:      true,
			expect:      307,
		},
		{
			name:        "Google OAuth2 — callback",
			method:      "GET",
			path:        "/api/v1/auth/google/callback",
			description: "Google gọi lại endpoint này. Gọi tay sẽ trả 400 vì thiếu state hợp lệ — đó là hành vi đúng.",
			noAuth:      true,
			expect:      400,
			queries: []query{
				{Key: "state", Value: "state-khong-hop-le", Description: "Phải khớp cookie oauth_state"},
				{Key: "code", Value: "authorization-code-tu-google"},
			},
		},
	}
}

func userEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Đăng ký người dùng (user_service)",
			method:      "POST",
			path:        "/api/v1/users/register",
			description: "Công khai. Khác /auth/register: tạo hồ sơ nghiệp vụ kèm số điện thoại và vai trò.",
			noAuth:      true,
			expect:      201,
			jsonBody: `{
    "phone": "{{test_phone}}",
    "email": "{{test_email}}",
    "password": "{{test_password}}",
    "role": "shipper",
    "full_name": "Nguyễn Văn Chủ Hàng"
}`,
			captures: map[string]string{"user_id": "json.data && json.data.id"},
		},
		{
			name:        "Xem hồ sơ",
			method:      "GET",
			path:        "/api/v1/users/{{user_id}}",
			description: "Chỉ xem được hồ sơ của chính mình, trừ khi là admin.",
		},
		{
			name:   "Cập nhật hồ sơ",
			method: "PUT",
			path:   "/api/v1/users/{{user_id}}",
			jsonBody: `{
    "full_name": "Nguyễn Văn Chủ Hàng (đã sửa)",
    "email": "{{test_email}}",
    "avatar_url": "https://cdn.logistic.vn/avatar/demo.png"
}`,
		},
		{
			name:   "Xem hồ sơ tài xế",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}/driver-profile",
		},
		{
			name:   "Cập nhật hồ sơ tài xế",
			method: "PUT",
			path:   "/api/v1/users/{{user_id}}/driver-profile",
			jsonBody: `{
    "license_number": "79B1-234567",
    "id_card": "079201001234"
}`,
		},
		{
			name:   "Xem hồ sơ chủ hàng",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}/shipper-profile",
		},
		{
			name:   "Cập nhật hồ sơ chủ hàng",
			method: "PUT",
			path:   "/api/v1/users/{{user_id}}/shipper-profile",
			jsonBody: `{
    "company_name": "Công ty TNHH Vận Tải Demo",
    "tax_code": "0312345678",
    "business_address": "12 Nguyễn Văn Linh, Quận 7, TP.HCM"
}`,
		},
		{
			name:        "Nộp hồ sơ KYC",
			method:      "PUT",
			path:        "/api/v1/users/{{user_id}}/kyc",
			description: "Tài xế nộp hồ sơ của chính mình. Việc DUYỆT nằm ở nhóm Admin và cần vai trò admin.",
			jsonBody: `{
    "kyc_status": "pending",
    "note": "Đã nộp ảnh GPLX hai mặt"
}`,
		},
	}
}

func addressEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Thêm địa chỉ",
			method: "POST",
			path:   "/api/v1/users/{{user_id}}/addresses",
			expect: 201,
			jsonBody: `{
    "label": "Kho Quận 7",
    "contact_name": "Trần Thị Kho",
    "contact_phone": "0900000002",
    "line1": "Lô A2, KCN Tân Thuận",
    "ward": "Tân Thuận Đông",
    "district": "Quận 7",
    "city": "TP.HCM",
    "latitude": 10.7379,
    "longitude": 106.7226,
    "address_type": "pickup",
    "is_default": true
}`,
			captures: map[string]string{"address_id": "json.data && json.data.address && json.data.address.id"},
		},
		{
			name:   "Danh sách địa chỉ",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}/addresses",
			queries: []query{
				{Key: "address_type", Value: "pickup", Description: "pickup | delivery | both"},
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:   "Cập nhật địa chỉ",
			method: "PUT",
			path:   "/api/v1/addresses/{{address_id}}",
			jsonBody: `{
    "label": "Kho Quận 7 (mới)",
    "contact_name": "Trần Thị Kho",
    "contact_phone": "0900000002",
    "line1": "Lô A5, KCN Tân Thuận",
    "ward": "Tân Thuận Đông",
    "district": "Quận 7",
    "city": "TP.HCM",
    "latitude": 10.7381,
    "longitude": 106.7231,
    "address_type": "both",
    "is_default": true
}`,
		},
		{
			name:   "Xoá địa chỉ",
			method: "DELETE",
			path:   "/api/v1/addresses/{{address_id}}",
		},
	}
}

func deviceEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Đăng ký thiết bị nhận push",
			method:      "POST",
			path:        "/api/v1/users/{{user_id}}/devices",
			description: "Chỉ đăng ký được thiết bị cho chính mình — nếu không, người khác nhận được push của bạn.",
			expect:      201,
			jsonBody: `{
    "device_token": "fcm-token-demo-0001",
    "platform": "android",
    "device_name": "Samsung A54 của tài xế"
}`,
			captures: map[string]string{"device_id": "json.data && json.data.device && json.data.device.id"},
		},
		{
			name:   "Danh sách thiết bị",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}/devices",
		},
		{
			name:   "Xoá thiết bị",
			method: "DELETE",
			path:   "/api/v1/devices/{{device_id}}",
		},
	}
}

func vehicleEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Đăng ký phương tiện",
			method:      "POST",
			path:        "/api/v1/vehicles",
			description: "driver_id lấy từ token, không nhận từ body — không ai đăng ký xe đứng tên người khác được.",
			expect:      201,
			jsonBody: `{
    "license_plate": "51C-999.88",
    "brand": "Hyundai",
    "model": "Mighty EX8",
    "manufacture_year": 2022,
    "vehicle_type": "truck",
    "capacity_weight_kg": 8000,
    "capacity_volume_cbm": 24.5
}`,
			captures: map[string]string{"vehicle_id": "json.data && json.data.id"},
		},
		{
			name:   "Danh sách phương tiện",
			method: "GET",
			path:   "/api/v1/vehicles",
			queries: []query{
				{Key: "status", Value: "active", Description: "active | maintenance | inactive"},
				{Key: "vehicle_type", Value: "truck"},
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:   "Chi tiết phương tiện",
			method: "GET",
			path:   "/api/v1/vehicles/{{vehicle_id}}",
		},
		{
			name:   "Cập nhật phương tiện",
			method: "PUT",
			path:   "/api/v1/vehicles/{{vehicle_id}}",
			jsonBody: `{
    "brand": "Hyundai",
    "model": "Mighty EX8 GTL",
    "manufacture_year": 2023,
    "vehicle_type": "truck",
    "capacity_weight_kg": 8500,
    "capacity_volume_cbm": 26
}`,
		},
		{
			name:   "Đổi trạng thái phương tiện",
			method: "PUT",
			path:   "/api/v1/vehicles/{{vehicle_id}}/status",
			jsonBody: `{
    "status": "maintenance"
}`,
		},
		{
			name:        "Tìm xe gần một điểm",
			method:      "POST",
			path:        "/api/v1/vehicles/nearby",
			description: "Chạy trên chỉ mục Redis GEO. Đây cũng là API matching_service dùng nội bộ.",
			jsonBody: `{
    "latitude": 10.7769,
    "longitude": 106.7009,
    "radius_km": 10,
    "min_weight_kg": 1000,
    "min_volume_cbm": 5,
    "vehicle_type": "truck",
    "limit": 20
}`,
		},
		{
			name:   "Xoá phương tiện",
			method: "DELETE",
			path:   "/api/v1/vehicles/{{vehicle_id}}",
		},
	}
}

func vehicleDocEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Tải lên giấy tờ xe",
			method: "POST",
			path:   "/api/v1/vehicles/{{vehicle_id}}/documents",
			expect: 201,
			jsonBody: `{
    "document_type": "registration",
    "document_number": "DK-51C-99988",
    "file_url": "https://cdn.logistic.vn/docs/dangky-51c99988.pdf"
}`,
			captures: map[string]string{"document_id": "json.data && json.data.document && json.data.document.id"},
		},
		{
			name:   "Danh sách giấy tờ của xe",
			method: "GET",
			path:   "/api/v1/vehicles/{{vehicle_id}}/documents",
			queries: []query{
				{Key: "review_status", Value: "pending", Description: "pending | approved | rejected"},
			},
		},
		{
			name:   "Xoá giấy tờ xe",
			method: "DELETE",
			path:   "/api/v1/vehicle-documents/{{document_id}}",
		},
	}
}

func locationEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Tài xế báo vị trí GPS",
			method:      "POST",
			path:        "/api/v1/vehicles/{{vehicle_id}}/location",
			description: "App tài xế gọi định kỳ. Hạn chờ ngắn hơn mức chung (2s) vì bản tin GPS mau hỏng.",
			jsonBody: `{
    "latitude": 10.7769,
    "longitude": 106.7009,
    "heading": 135.5,
    "speed_kph": 42.3
}`,
		},
		{
			name:   "Vị trí hiện tại của xe",
			method: "GET",
			path:   "/api/v1/vehicles/{{vehicle_id}}/location",
		},
		{
			name:        "Bật/tắt nhận đơn",
			method:      "POST",
			path:        "/api/v1/drivers/{{user_id}}/availability",
			description: "Bật thì xe vào chỉ mục tìm kiếm của matching; tắt thì gỡ ra.",
			jsonBody: `{
    "vehicle_id": "{{vehicle_id}}",
    "is_online": true,
    "available_weight_kg": 6000,
    "available_volume_cbm": 18,
    "current_lat": 10.7769,
    "current_lng": 106.7009
}`,
		},
		{
			name:   "Trạng thái nhận đơn của tài xế",
			method: "GET",
			path:   "/api/v1/drivers/{{user_id}}/availability",
		},
	}
}

func matchingEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Chủ hàng đăng đơn cần xe (Bid)",
			method:      "POST",
			path:        "/api/v1/matching/bids",
			description: "Sau khi lưu, matching_service tìm tài xế phù hợp và phát sự kiện qua RabbitMQ.",
			jsonBody: `{
    "shipper_id": "{{user_id}}",
    "shipper_phone": "{{test_phone}}",
    "shipper_mail": "{{test_email}}",
    "consignee_id": "",
    "consignee_phone": "0900000003",
    "consignee_mail": "nguoinhan@logistic.vn",
    "origin": {
        "latitude": 10.7769,
        "longitude": 106.7009,
        "zone_id": "hcm-q1"
    },
    "destination": {
        "latitude": 10.9804,
        "longitude": 106.6519,
        "zone_id": "binhduong-thuandau"
    },
    "volume_m3": 12.5,
    "weight_kg": 3200,
    "max_price": 4500000,
    "cargo_value": 120000000,
    "required_deposit": 500000,
    "desired_deposit": 300000
}`,
			captures: map[string]string{"bid_id": "json.data && json.data.bid_id"},
		},
		{
			name:        "Tài xế đăng chuyến còn chỗ (Ask)",
			method:      "POST",
			path:        "/api/v1/matching/asks",
			description: "Cần token của tài xế. Dùng biến driver_id nếu chạy bằng tài khoản khác.",
			jsonBody: `{
    "driver_id": "{{user_id}}",
    "driver_phone": "0900000004",
    "driver_mail": "taixe@logistic.vn",
    "vehicle_id": "{{vehicle_id}}",
    "current_location": {
        "latitude": 10.7769,
        "longitude": 106.7009,
        "zone_id": "hcm-q1"
    },
    "destination": {
        "latitude": 10.9804,
        "longitude": 106.6519,
        "zone_id": "binhduong-thuandau"
    },
    "available_volume_m3": 20,
    "available_weight_kg": 6000,
    "min_price": 3800000,
    "desired_deposit": 300000
}`,
			captures: map[string]string{"ask_id": "json.data && json.data.ask_id"},
		},
		{
			name:   "Tài xế báo giá",
			method: "POST",
			path:   "/api/v1/matching/offers",
			jsonBody: `{
    "bid_id": "{{bid_id}}",
    "ask_id": "{{ask_id}}",
    "desired_price": 4200000
}`,
		},
		{
			name:        "Chủ hàng từ chối báo giá",
			method:      "POST",
			path:        "/api/v1/matching/offers/reject",
			description: "Đơn quay lại trạng thái PENDING, các tài xế khác tiếp tục báo giá được.",
			jsonBody: `{
    "bid_id": "{{bid_id}}",
    "ask_id": "{{ask_id}}",
    "reason": "Giá cao hơn ngân sách"
}`,
		},
		{
			name:        "Chủ hàng chốt xe",
			method:      "POST",
			path:        "/api/v1/matching/matches/accept",
			description: "Chốt xong, matching_service phát matching.match.found; notification_service báo cho cả hai bên.",
			jsonBody: `{
    "bid_id": "{{bid_id}}",
    "ask_id": "{{ask_id}}",
    "consensus_price": 4200000,
    "consensus_deposit": 400000,
    "shipper_signature": "chu-ky-so-demo"
}`,
		},
	}
}

func notificationEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Hộp thư thông báo",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}/notifications",
			queries: []query{
				{Key: "type", Value: "", Description: "match_found | driver_candidate | offer_received...", Disabled: true},
				{Key: "unread_only", Value: "false"},
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
			captures: map[string]string{
				"notification_id": "json.data && json.data.notifications && json.data.notifications.length > 0 && json.data.notifications[0].id",
			},
		},
		{
			name:        "Số thông báo chưa đọc",
			method:      "GET",
			path:        "/api/v1/users/{{user_id}}/notifications/unread-count",
			description: "App gọi ở mọi màn hình để vẽ chấm đỏ; con số này được cache trên Redis.",
		},
		{
			name:   "Chi tiết một thông báo",
			method: "GET",
			path:   "/api/v1/notifications/{{notification_id}}",
		},
		{
			name:   "Đánh dấu đã đọc",
			method: "PUT",
			path:   "/api/v1/notifications/{{notification_id}}/read",
		},
		{
			name:   "Đánh dấu tất cả đã đọc",
			method: "PUT",
			path:   "/api/v1/users/{{user_id}}/notifications/read-all",
		},
		{
			name:   "Xoá thông báo",
			method: "DELETE",
			path:   "/api/v1/notifications/{{notification_id}}",
		},
		{
			name:   "Xem cài đặt nhận thông báo",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}/notification-preferences",
		},
		{
			name:   "Cập nhật cài đặt nhận thông báo",
			method: "PUT",
			path:   "/api/v1/users/{{user_id}}/notification-preferences",
			jsonBody: `{
    "in_app_enabled": true,
    "push_enabled": true,
    "email_enabled": false,
    "sms_enabled": false,
    "match_events_enabled": true,
    "promotion_enabled": false,
    "quiet_hours_start": "22:00",
    "quiet_hours_end": "07:00"
}`,
		},
	}
}

func mediaEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Tải file lên",
			method:      "POST",
			path:        "/api/v1/media/upload",
			description: "Chọn file thật ở tab Body > form-data trước khi gửi. Trả về public_id dùng cho lệnh xoá.",
			formData: []formField{
				{Key: "file", Type: "file", Src: "", Description: "Ảnh giấy tờ hoặc avatar"},
				{Key: "folder", Type: "text", Value: "vehicle-documents"},
				{Key: "prefix", Type: "text", Value: "demo"},
			},
			captures: map[string]string{"public_id": "json.data && json.data.public_id"},
		},
		{
			name:   "Xoá file",
			method: "DELETE",
			path:   "/api/v1/media/files/{{public_id}}",
		},
	}
}

func adminUserEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Danh sách người dùng",
			method: "GET",
			path:   "/api/v1/admin/users",
			queries: []query{
				{Key: "role", Value: "", Description: "driver | shipper | admin", Disabled: true},
				{Key: "status", Value: "", Description: "active | banned | suspended", Disabled: true},
				{Key: "keyword", Value: "", Description: "tìm theo phone/email/tên", Disabled: true},
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:   "Thống kê người dùng",
			method: "GET",
			path:   "/api/v1/admin/users/stats",
		},
		{
			name:   "Khoá / mở tài khoản",
			method: "PUT",
			path:   "/api/v1/admin/users/{{user_id}}/status",
			jsonBody: `{
    "status": "suspended",
    "reason": "Nghi ngờ gian lận cước"
}`,
		},
		{
			name:   "Xoá người dùng",
			method: "DELETE",
			path:   "/api/v1/admin/users/{{user_id}}",
		},
	}
}

func adminKycEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Hàng đợi duyệt KYC",
			method: "GET",
			path:   "/api/v1/admin/kyc/pending",
			queries: []query{
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:        "Duyệt / từ chối KYC",
			method:      "PUT",
			path:        "/api/v1/admin/kyc/{{user_id}}/review",
			description: "reviewer_id lấy từ token của admin đang đăng nhập, không nhận từ body.",
			jsonBody: `{
    "approved": true,
    "note": "Giấy tờ hợp lệ, ảnh rõ nét"
}`,
		},
	}
}

func adminVehicleEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Danh sách toàn bộ phương tiện",
			method: "GET",
			path:   "/api/v1/admin/vehicles",
			queries: []query{
				{Key: "status", Value: "", Disabled: true},
				{Key: "verification_status", Value: "pending", Description: "pending | verified | rejected"},
				{Key: "vehicle_type", Value: "", Disabled: true},
				{Key: "keyword", Value: "", Disabled: true},
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:   "Thống kê phương tiện",
			method: "GET",
			path:   "/api/v1/admin/vehicles/stats",
		},
		{
			name:   "Duyệt phương tiện",
			method: "PUT",
			path:   "/api/v1/admin/vehicles/{{vehicle_id}}/verify",
			jsonBody: `{
    "approved": true,
    "note": "Đủ giấy tờ, đăng kiểm còn hạn"
}`,
		},
	}
}

func adminDocEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Hàng đợi duyệt giấy tờ",
			method: "GET",
			path:   "/api/v1/admin/vehicle-documents/pending",
			queries: []query{
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:   "Duyệt giấy tờ xe",
			method: "PUT",
			path:   "/api/v1/admin/vehicle-documents/{{document_id}}/review",
			jsonBody: `{
    "approved": true,
    "note": "Đăng ký xe khớp biển số"
}`,
		},
	}
}

func adminNotificationEndpoints() []endpoint {
	return []endpoint{
		{
			name:   "Danh sách toàn bộ thông báo",
			method: "GET",
			path:   "/api/v1/admin/notifications",
			queries: []query{
				{Key: "user_id", Value: "", Description: "lọc theo người nhận", Disabled: true},
				{Key: "type", Value: "", Disabled: true},
				{Key: "status", Value: "", Description: "pending | sent | failed | read", Disabled: true},
				{Key: "page", Value: "1"},
				{Key: "page_size", Value: "20"},
			},
		},
		{
			name:   "Thống kê thông báo",
			method: "GET",
			path:   "/api/v1/admin/notifications/stats",
		},
		{
			name:        "Gửi thông báo thủ công",
			method:      "POST",
			path:        "/api/v1/admin/notifications/send",
			description: "Để trống user_ids và đặt broadcast_role thì gửi cho toàn bộ người dùng thuộc vai trò đó.",
			jsonBody: `{
    "user_ids": ["{{user_id}}"],
    "broadcast_role": "",
    "type": "system_announcement",
    "channel": "in_app",
    "title": "Bảo trì hệ thống",
    "body": "Hệ thống bảo trì từ 23:00 đến 01:00 ngày mai.",
    "data": "{\"deep_link\":\"/notifications\"}"
}`,
		},
	}
}

func adminTemplateEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Danh sách mẫu thông báo",
			method:      "GET",
			path:        "/api/v1/admin/notification-templates",
			description: "Mẫu quyết định câu chữ thông báo gửi cho người dùng, sửa được mà không cần deploy lại.",
			queries: []query{
				{Key: "channel", Value: "", Description: "in_app | push | email | sms", Disabled: true},
				{Key: "locale", Value: "vi"},
			},
		},
		{
			name:        "Tạo mẫu thông báo",
			method:      "POST",
			path:        "/api/v1/admin/notification-templates",
			description: "Placeholder dạng {{driver_name}} được thay bằng dữ liệu thật lúc gửi.",
			expect:      201,
			jsonBody: `{
    "code": "MATCH_FOUND_SHIPPER",
    "name": "Chủ hàng — đã tìm được xe",
    "channel": "in_app",
    "locale": "vi",
    "title_template": "Đã tìm được xe cho đơn hàng của bạn",
    "body_template": "Tài xế {{driver_name}} nhận đơn với giá {{price}} đ. Biển số {{license_plate}}.",
    "is_active": true
}`,
			captures: map[string]string{"template_id": "json.data && json.data.template && json.data.template.id"},
		},
		{
			name:   "Sửa mẫu thông báo",
			method: "PUT",
			path:   "/api/v1/admin/notification-templates/{{template_id}}",
			jsonBody: `{
    "name": "Chủ hàng — đã tìm được xe (bản 2)",
    "title_template": "Xe đã nhận đơn {{bid_code}}",
    "body_template": "Tài xế {{driver_name}} ({{license_plate}}) nhận đơn với giá {{price}} đ.",
    "is_active": true
}`,
		},
		{
			name:   "Xoá mẫu thông báo",
			method: "DELETE",
			path:   "/api/v1/admin/notification-templates/{{template_id}}",
		},
	}
}

func securityEndpoints() []endpoint {
	return []endpoint{
		{
			name:        "Header vai trò tự khai KHÔNG vào được khu admin",
			method:      "GET",
			path:        "/api/v1/admin/users",
			description: "Gửi X-User-Role: admin mà không có token. Phải trả 401 — gateway xoá header danh tính do client gửi.",
			noAuth:      true,
			expect:      401,
			extraTests: []string{
				"pm.test('bị từ chối bằng mã UNAUTHENTICATED', function () {",
				"    pm.expect(pm.response.json().error.code).to.eql('UNAUTHENTICATED');",
				"});",
			},
		},
		{
			name:        "Token rác bị từ chối",
			method:      "GET",
			path:        "/api/v1/vehicles",
			description: "Token không phải JWT hợp lệ. Phải trả 401, không được coi như khách vãng lai.",
			noAuth:      true,
			expect:      401,
		},
		{
			name:   "Không có token thì không vào được endpoint người dùng",
			method: "GET",
			path:   "/api/v1/users/{{user_id}}",
			noAuth: true,
			expect: 401,
		},
		{
			name:        "Token người dùng thường KHÔNG vào được khu admin",
			method:      "GET",
			path:        "/api/v1/admin/users/stats",
			description: "Dùng access_token của tài khoản shipper/driver. Phải trả 403 PERMISSION_DENIED.",
			expect:      403,
			extraTests: []string{
				"pm.test('bị từ chối bằng mã PERMISSION_DENIED', function () {",
				"    pm.expect(pm.response.json().error.code).to.eql('PERMISSION_DENIED');",
				"});",
			},
		},
		{
			name:        "ID sai định dạng bị chặn tại gateway",
			method:      "GET",
			path:        "/api/v1/vehicles/khong-phai-uuid",
			description: "Gateway parse UUID trước khi gọi xuống gRPC, nên id rác dừng ngay tại biên.",
			expect:      400,
			extraTests: []string{
				"pm.test('báo lỗi INVALID_ID', function () {",
				"    pm.expect(pm.response.json().error.code).to.eql('INVALID_ID');",
				"});",
			},
		},
	}
}
