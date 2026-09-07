package main

// Sơ đồ cho các tài liệu trong docs/services/ — mô tả BÊN TRONG một service:
// tầng, kho dữ liệu, phụ thuộc ra ngoài.

func gatewayService() Diagram {
	return Diagram{
		Name:    "gateway-service",
		Title:   "gateway_service",
		Caption: "Không có database, không có luật nghiệp vụ. Chỉ dịch HTTP ↔ gRPC, chặn quyền, chuẩn hoá lỗi.",
		Nodes: []Node{
			{ID: "in", Label: "HTTP từ Nginx", Kind: KindClient, Col: 1, Row: 0},
			{ID: "mw", Label: "Chuỗi middleware\nRequestID → Recovery → AccessLog\n→ IdentityContext → ErrorGuard", Kind: KindGateway, Col: 1, Row: 1, Span: 2},
			{ID: "route", Label: "delivery/http\ncây route /api/v1", Kind: KindGateway, Col: 1, Row: 2},
			{ID: "guard", Label: "RequireRole(\"admin\")\nchỉ nhóm /admin", Kind: KindNote, Col: 3, Row: 2},
			{ID: "ctrl", Label: "controller/\nbind JSON → gọi gRPC", Kind: KindGateway, Col: 1, Row: 3},
			{ID: "resp", Label: "response/\nEnvelope + Error", Kind: KindNote, Col: 0, Row: 3},

			{ID: "auth", Label: "auth_service", Kind: KindService, Col: 0, Row: 4},
			{ID: "user", Label: "user_service", Kind: KindService, Col: 1, Row: 4},
			{ID: "vehicle", Label: "vehicle_service", Kind: KindService, Col: 2, Row: 4},
			{ID: "matching", Label: "matching_service", Kind: KindService, Col: 3, Row: 4},
			{ID: "notif", Label: "notification_service", Kind: KindService, Col: 1, Row: 5},
			{ID: "media", Label: "media_service", Kind: KindService, Col: 2, Row: 5},
		},
		Edges: []Edge{
			{From: "in", To: "mw"},
			{From: "mw", To: "route"},
			{From: "route", To: "guard", Label: "nhóm admin"},
			{From: "route", To: "ctrl"},
			{From: "ctrl", To: "resp", Label: "mọi phản hồi"},
			{From: "ctrl", To: "auth", Label: "gRPC"},
			{From: "ctrl", To: "user", Label: "gRPC"},
			{From: "ctrl", To: "vehicle", Label: "gRPC"},
			{From: "ctrl", To: "matching", Label: "gRPC"},
			{From: "ctrl", To: "notif", Label: "gRPC"},
			{From: "ctrl", To: "media", Label: "gRPC"},
		},
	}
}

func authService() Diagram {
	return Diagram{
		Name:    "auth-service",
		Title:   "auth_service",
		Caption: "Danh tính và JWT. Dùng MySQL master-slave; mật khẩu và TOTP là trường Sensitive.",
		Nodes: []Node{
			{ID: "gw", Label: "gateway_service", Kind: KindGateway, Col: 1, Row: 0},
			{ID: "ctrl", Label: "controller/\n5 RPC", Kind: KindService, Col: 1, Row: 1},
			{ID: "rpc", Label: "Register · Login\nGetGoogleLoginURL\nGoogleCallback · VerifyToken", Kind: KindNote, Col: 2, Row: 1},
			{ID: "biz", Label: "biz/\nhash mật khẩu, ký JWT", Kind: KindService, Col: 1, Row: 2},
			{ID: "google", Label: "Google OAuth2", Kind: KindExternal, Col: 3, Row: 2},
			{ID: "repo", Label: "repo/", Kind: KindService, Col: 1, Row: 3},
			{ID: "master", Label: "MySQL master\nghi", Kind: KindStore, Col: 0, Row: 4},
			{ID: "slave", Label: "MySQL slave\nđọc", Kind: KindStore, Col: 2, Row: 4},
			{ID: "tbl", Label: "users\n(password, totp_secret = Sensitive)", Kind: KindNote, Col: 1, Row: 5},
		},
		Edges: []Edge{
			{From: "gw", To: "ctrl", Label: "gRPC :9001"},
			{From: "ctrl", To: "rpc"},
			{From: "ctrl", To: "biz"},
			{From: "biz", To: "google", Label: "đổi code lấy token"},
			{From: "biz", To: "repo"},
			{From: "repo", To: "master"},
			{From: "repo", To: "slave"},
			{From: "master", To: "tbl"},
		},
	}
}

func userService() Diagram {
	return Diagram{
		Name:    "user-service",
		Title:   "user_service",
		Caption: "5 bảng. Redis db0 làm cache-aside: ghi Postgres trước, xoá key sau.",
		Nodes: []Node{
			{ID: "gw", Label: "gateway_service", Kind: KindGateway, Col: 1, Row: 0},
			{ID: "ctrl", Label: "controller/\n21 RPC (15 client + 6 admin)", Kind: KindService, Col: 1, Row: 1},
			{ID: "biz", Label: "biz/\nmustBeRole · địa chỉ mặc định duy nhất\n· chặn duyệt KYC hai lần", Kind: KindService, Col: 1, Row: 2, Span: 2},
			{ID: "repo", Label: "repo/\ncache-aside + invalidate", Kind: KindService, Col: 1, Row: 3},
			{ID: "redis", Label: "Redis db0\nprefix \"user\"", Kind: KindStore, Col: 3, Row: 3},
			{ID: "pg", Label: "Postgres master + slave", Kind: KindStore, Col: 0, Row: 4},
			{ID: "t1", Label: "users", Kind: KindNote, Col: 0, Row: 5},
			{ID: "t2", Label: "driver_profiles\ndriver ⟷ KYC", Kind: KindNote, Col: 1, Row: 5},
			{ID: "t3", Label: "shipper_profiles", Kind: KindNote, Col: 2, Row: 5},
			{ID: "t4", Label: "addresses\nsổ địa chỉ + toạ độ", Kind: KindNote, Col: 3, Row: 5},
			{ID: "t5", Label: "user_devices\npush token", Kind: KindNote, Col: 3, Row: 6},
		},
		Edges: []Edge{
			{From: "gw", To: "ctrl", Label: "gRPC :9004"},
			{From: "ctrl", To: "biz"},
			{From: "biz", To: "repo"},
			{From: "repo", To: "redis", Label: "đọc / xoá"},
			{From: "repo", To: "pg", Label: "nguồn sự thật"},
			{From: "pg", To: "t1"},
			{From: "pg", To: "t2"},
			{From: "pg", To: "t3"},
			{From: "pg", To: "t4"},
			{From: "t4", To: "t5"},
		},
	}
}

func vehicleService() Diagram {
	return Diagram{
		Name:    "vehicle-service",
		Title:   "vehicle_service",
		Caption: "Nguồn dữ liệu \"xe đang chạy\" cho matching. Redis db1 giữ chỉ mục GEO.",
		Nodes: []Node{
			{ID: "gw", Label: "gateway_service", Kind: KindGateway, Col: 1, Row: 0},
			{ID: "matching", Label: "matching_service\nSearchNearbyVehicles", Kind: KindService, Col: 3, Row: 0},
			{ID: "ctrl", Label: "controller/\n19 RPC (14 client + 5 admin)", Kind: KindService, Col: 2, Row: 1},
			{ID: "biz", Label: "biz/\nchặn xe chưa duyệt lên online\n· kẹp sức chứa khai báo", Kind: KindService, Col: 2, Row: 2, Span: 2},
			{ID: "repo", Label: "repo/\nghi Postgres TRƯỚC, đồng bộ GEO SAU", Kind: KindService, Col: 2, Row: 3, Span: 2},
			{ID: "geo", Label: "Redis db1\nsorted set GEO xe online", Kind: KindStore, Col: 4, Row: 4},
			{ID: "pg", Label: "Postgres master + slave", Kind: KindStore, Col: 1, Row: 4},
			{ID: "t1", Label: "vehicles\nstatus ≠ verification_status", Kind: KindNote, Col: 0, Row: 5},
			{ID: "t2", Label: "vehicle_documents\nđăng kiểm, bảo hiểm…", Kind: KindNote, Col: 1, Row: 5},
			{ID: "t3", Label: "vehicle_locations\n1 dòng / xe, ghi đè", Kind: KindNote, Col: 2, Row: 5},
			{ID: "t4", Label: "driver_availabilities\nsức chứa CÒN TRỐNG", Kind: KindNote, Col: 3, Row: 5},
			{ID: "fb", Label: "Redis chết → quét zone_id\n+ haversine trong Go", Kind: KindNote, Col: 4, Row: 5},
		},
		Edges: []Edge{
			{From: "gw", To: "ctrl", Label: "gRPC :9005"},
			{From: "matching", To: "ctrl", Label: "gRPC"},
			{From: "ctrl", To: "biz"},
			{From: "biz", To: "repo"},
			{From: "repo", To: "pg"},
			{From: "repo", To: "geo", Label: "GEOADD / ZREM"},
			{From: "pg", To: "t1"},
			{From: "pg", To: "t2"},
			{From: "pg", To: "t3"},
			{From: "pg", To: "t4"},
			{From: "geo", To: "fb", Label: "dự phòng", Dashed: true},
		},
	}
}

func matchingService() Diagram {
	return Diagram{
		Name:    "matching-service",
		Title:   "matching_service",
		Caption: "Lõi ghép đơn. Ba broker phục vụ ba nhu cầu khác nhau, không thay thế nhau.",
		Nodes: []Node{
			{ID: "gw", Label: "gateway_service", Kind: KindGateway, Col: 1, Row: 0},
			{ID: "ctrl", Label: "controller/\nSubmitBid · SubmitAsk\nSubmitOffer · RejectOffer · AcceptMatch", Kind: KindService, Col: 1, Row: 1, Span: 2},
			{ID: "engine", Label: "biz/MatchingEngine\nvòng thương lượng", Kind: KindService, Col: 1, Row: 2},
			{ID: "spatial", Label: "GeoHashEngine\ntính zone_id", Kind: KindNote, Col: 0, Row: 2},
			{ID: "wallet", Label: "wallet_service\nCheckBalance", Kind: KindService, Col: 3, Row: 2},

			{ID: "repo", Label: "repo/\nPostGIS ST_DWithin", Kind: KindService, Col: 1, Row: 3},
			{ID: "pg", Label: "Postgres master + slave\nasks · bids · matches", Kind: KindStore, Col: 0, Row: 4},

			{ID: "notifier", Label: "biz.Notifier", Kind: KindNote, Col: 2, Row: 3},
			{ID: "rabbit", Label: "RabbitMQ\nthông báo BỀN", Kind: KindBroker, Col: 3, Row: 4},
			{ID: "kafka", Label: "Kafka\nnhật ký + đặt cọc", Kind: KindBroker, Col: 2, Row: 4},
			{ID: "nats", Label: "NATS JetStream\nrealtime app đang mở", Kind: KindBroker, Col: 1, Row: 4},
			{ID: "noop", Label: "RabbitMQ chết →\nNoopNotifier, vẫn ghép đơn", Kind: KindNote, Col: 3, Row: 5},
		},
		Edges: []Edge{
			{From: "gw", To: "ctrl", Label: "gRPC :9003"},
			{From: "ctrl", To: "engine"},
			{From: "engine", To: "spatial"},
			{From: "engine", To: "wallet", Label: "trước khi chốt"},
			{From: "engine", To: "repo"},
			{From: "repo", To: "pg"},
			{From: "engine", To: "notifier"},
			{From: "notifier", To: "rabbit", Dashed: true},
			{From: "engine", To: "kafka", Dashed: true},
			{From: "engine", To: "nats", Dashed: true},
			{From: "rabbit", To: "noop", Label: "dự phòng", Dashed: true},
		},
	}
}

func notificationService() Diagram {
	return Diagram{
		Name:    "notification-service",
		Title:   "notification_service",
		Caption: "Hai cửa vào: gRPC để ĐỌC hộp thư, RabbitMQ để SINH thông báo.",
		Nodes: []Node{
			{ID: "gw", Label: "gateway_service", Kind: KindGateway, Col: 0, Row: 0},
			{ID: "rabbit", Label: "RabbitMQ\nbinding matching.#", Kind: KindBroker, Col: 3, Row: 0},

			{ID: "ctrl", Label: "controller/\n15 RPC — chỉ ĐỌC + admin", Kind: KindService, Col: 0, Row: 1},
			{ID: "consumer", Label: "consumer/\nMatchingConsumer", Kind: KindService, Col: 3, Row: 1},
			{ID: "ack", Label: "ACK: hỏng / trùng / lạ\nNACK: lỗi tạm thời", Kind: KindNote, Col: 4, Row: 1},

			{ID: "engine", Label: "biz/NotificationEngine\nMỘT nơi duy nhất giữ luật", Kind: KindService, Col: 1, Row: 2, Span: 2},
			{ID: "pref", Label: "lọc theo cài đặt\n+ giờ yên lặng", Kind: KindNote, Col: 3, Row: 2},

			{ID: "repo", Label: "repo/\nCreateWithEventGuard", Kind: KindService, Col: 1, Row: 3, Span: 2},
			{ID: "pg", Label: "Postgres", Kind: KindStore, Col: 0, Row: 4},
			{ID: "redis", Label: "Redis db2\nbộ đếm chưa đọc", Kind: KindStore, Col: 3, Row: 4},

			{ID: "t1", Label: "notifications\n1 dòng / người nhận", Kind: KindNote, Col: 0, Row: 5},
			{ID: "t2", Label: "notification_templates\nsửa câu chữ không cần deploy", Kind: KindNote, Col: 1, Row: 5},
			{ID: "t3", Label: "notification_preferences", Kind: KindNote, Col: 2, Row: 5},
			{ID: "t4", Label: "processed_events\nchống xử lý trùng", Kind: KindNote, Col: 3, Row: 5},
		},
		Edges: []Edge{
			{From: "gw", To: "ctrl", Label: "gRPC :9006"},
			{From: "rabbit", To: "consumer", Dashed: true},
			{From: "consumer", To: "ack"},
			{From: "ctrl", To: "engine"},
			{From: "consumer", To: "engine", Label: "DispatchEvent"},
			{From: "engine", To: "pref"},
			{From: "engine", To: "repo"},
			{From: "repo", To: "pg"},
			{From: "repo", To: "redis", Label: "xoá bộ đếm"},
			{From: "pg", To: "t1"},
			{From: "pg", To: "t2"},
			{From: "pg", To: "t3"},
			{From: "pg", To: "t4"},
		},
	}
}

func mediaService() Diagram {
	return Diagram{
		Name:    "media-service",
		Title:   "media_service",
		Caption: "Service không trạng thái, không database. Chỉ chuyển tiếp file lên kho lưu trữ ngoài.",
		Nodes: []Node{
			{ID: "gw", Label: "gateway_service\nmultipart/form-data", Kind: KindGateway, Col: 1, Row: 0},
			{ID: "ctrl", Label: "controller/\nUploadFile · DeleteFile", Kind: KindService, Col: 1, Row: 1},
			{ID: "iface", Label: "storage.FileStorage\n(interface)", Kind: KindNote, Col: 2, Row: 2},
			{ID: "impl", Label: "storage/cloudinary/\ncài đặt cụ thể", Kind: KindService, Col: 1, Row: 2},
			{ID: "cloud", Label: "Cloudinary", Kind: KindExternal, Col: 1, Row: 3},
			{ID: "url", Label: "Trả về public_id + url\nlưu ở service khác\n(vd: vehicle_documents.file_url)", Kind: KindNote, Col: 0, Row: 3},
		},
		Edges: []Edge{
			{From: "gw", To: "ctrl", Label: "gRPC :9002"},
			{From: "ctrl", To: "impl"},
			{From: "impl", To: "iface", Label: "cài đặt", Dashed: true},
			{From: "impl", To: "cloud", Label: "HTTPS"},
			{From: "cloud", To: "url"},
		},
	}
}

func walletService() Diagram {
	return Diagram{
		Name:    "wallet-service",
		Title:   "wallet_service",
		Caption: "CHƯA có trong docker-compose. Tiền bạc nên dùng Unit of Work để mọi bút toán vào cùng một transaction.",
		Nodes: []Node{
			{ID: "matching", Label: "matching_service", Kind: KindService, Col: 0, Row: 0},
			{ID: "kafka", Label: "Kafka\nwallet.hold_deposit", Kind: KindBroker, Col: 2, Row: 0},
			{ID: "ctrl", Label: "controller/\nGetBalance · Deposit · Transfer\nSearchWallets · SearchTransactions", Kind: KindService, Col: 0, Row: 1, Span: 2},
			{ID: "consumer", Label: "broker/kafka\nconsumer", Kind: KindService, Col: 2, Row: 1},
			{ID: "biz", Label: "biz/\nHoldDeposit · ReleaseAndPay\nTransferMoney", Kind: KindService, Col: 1, Row: 2, Span: 2},
			{ID: "uow", Label: "UnitOfWork\nmọi bút toán cùng 1 transaction", Kind: KindNote, Col: 3, Row: 2},
			{ID: "repo", Label: "repository/", Kind: KindService, Col: 1, Row: 3},
			{ID: "mysql", Label: "MySQL\nwallets · transactions", Kind: KindStore, Col: 0, Row: 4},
			{ID: "es", Label: "Elasticsearch\ntra cứu giao dịch", Kind: KindStore, Col: 2, Row: 4},
			{ID: "dedupe", Label: "processed_messages\nchống xử lý trùng Kafka", Kind: KindNote, Col: 3, Row: 4},
		},
		Edges: []Edge{
			{From: "matching", To: "ctrl", Label: "gRPC :9007"},
			{From: "kafka", To: "consumer", Dashed: true},
			{From: "ctrl", To: "biz"},
			{From: "consumer", To: "biz", Label: "HoldDeposit"},
			{From: "biz", To: "uow"},
			{From: "biz", To: "repo"},
			{From: "repo", To: "mysql"},
			{From: "repo", To: "es", Label: "đồng bộ sau khi ghi"},
			{From: "consumer", To: "dedupe"},
		},
	}
}
