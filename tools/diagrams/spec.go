// Package main sinh sơ đồ kiến trúc từ MỘT nguồn khai báo duy nhất.
//
// Vì sao không vẽ tay từng file .drawio: mỗi tài liệu cần một sơ đồ, và sơ đồ
// vẽ tay thì (a) mỗi cái một phong cách, (b) không ai nhớ cập nhật khi code đổi.
// Khai báo bằng Go thì cả .drawio lẫn .svg cùng sinh ra từ một chỗ, đổi một lần
// là hai định dạng cùng đổi.
//
// Sinh lại:  make diagrams
package main

// Kind quyết định màu sắc và hình dạng của một hộp.
type Kind string

const (
	KindClient   Kind = "client"    // app tài xế / chủ hàng / trình duyệt admin
	KindEdge     Kind = "edge"      // Nginx, load balancer
	KindGateway  Kind = "gateway"   // gateway_service
	KindService  Kind = "service"   // service nội bộ
	KindStore    Kind = "store"     // Postgres, MySQL, Redis
	KindBroker   Kind = "broker"    // RabbitMQ, Kafka, NATS
	KindExternal Kind = "external"  // dịch vụ bên thứ ba
	KindNote     Kind = "note"      // ghi chú giải thích
	KindLayer    Kind = "layer"     // tầng trong kiến trúc nội bộ service
)

// Node là một hộp trên lưới. Col/Row tính từ 0, ô (0,0) ở góc trên trái.
type Node struct {
	ID    string
	Label string // dùng "\n" để xuống dòng
	Kind  Kind
	Col   int
	Row   int
	Span  int // số cột chiếm ngang, mặc định 1
}

// Edge là mũi tên nối hai hộp.
type Edge struct {
	From, To string
	Label    string
	Dashed   bool // nét đứt: luồng bất đồng bộ / tuỳ chọn
}

// Diagram là một sơ đồ hoàn chỉnh. Name chính là tên file sinh ra, nên phải
// trùng tên gốc với file .md đi kèm (quy ước ở docs/README.md).
type Diagram struct {
	Name    string // vd: "matching-notification-flow" -> .drawio + .svg cùng tên
	Title   string
	Caption string
	Nodes   []Node
	Edges   []Edge
}

// diagrams là toàn bộ sơ đồ của repo. Mỗi tài liệu .md có đúng một mục ở đây.
func diagrams() []Diagram {
	return []Diagram{
		systemOverview(),
		serviceLayering(),
		gatewayService(),
		authService(),
		userService(),
		vehicleService(),
		matchingService(),
		notificationService(),
		mediaService(),
		walletService(),
	}
}

// ---------------------------------------------------------------------------
// KIẾN TRÚC TỔNG THỂ
// ---------------------------------------------------------------------------

func systemOverview() Diagram {
	return Diagram{
		Name:    "system-overview",
		Title:   "Kiến trúc tổng thể",
		Caption: "Client chỉ chạm được Nginx và gateway. Service nội bộ nằm trong mạng logistic_net.",
		Nodes: []Node{
			{ID: "driver", Label: "App tài xế", Kind: KindClient, Col: 0, Row: 0},
			{ID: "shipper", Label: "App chủ hàng", Kind: KindClient, Col: 1, Row: 0},
			{ID: "admin", Label: "Trang quản trị", Kind: KindClient, Col: 2, Row: 0},

			{ID: "nginx", Label: "Nginx\n:80 / :443", Kind: KindEdge, Col: 1, Row: 1},
			{ID: "gw", Label: "gateway_service\n:8080", Kind: KindGateway, Col: 1, Row: 2},

			{ID: "auth", Label: "auth_service\n:9001", Kind: KindService, Col: 0, Row: 3},
			{ID: "user", Label: "user_service\n:9004", Kind: KindService, Col: 1, Row: 3},
			{ID: "vehicle", Label: "vehicle_service\n:9005", Kind: KindService, Col: 2, Row: 3},
			{ID: "matching", Label: "matching_service\n:9003", Kind: KindService, Col: 3, Row: 3},

			{ID: "notif", Label: "notification_service\n:9006", Kind: KindService, Col: 0, Row: 4},
			{ID: "media", Label: "media_service\n:9002", Kind: KindService, Col: 1, Row: 4},
			{ID: "wallet", Label: "wallet_service\n:9007", Kind: KindService, Col: 2, Row: 4},

			{ID: "redis", Label: "Redis\ncache + GEO", Kind: KindStore, Col: 3, Row: 4},
			{ID: "rabbit", Label: "RabbitMQ\nlogistic.events", Kind: KindBroker, Col: 3, Row: 5},
			{ID: "kafka", Label: "Kafka\nnhật ký sự kiện", Kind: KindBroker, Col: 2, Row: 5},
			{ID: "nats", Label: "NATS JetStream\nrealtime", Kind: KindBroker, Col: 1, Row: 5},
			{ID: "pg", Label: "Postgres / MySQL\nmaster + slave", Kind: KindStore, Col: 0, Row: 5},
		},
		Edges: []Edge{
			{From: "driver", To: "nginx", Label: "HTTPS"},
			{From: "shipper", To: "nginx", Label: "HTTPS"},
			{From: "admin", To: "nginx", Label: "HTTPS"},
			{From: "nginx", To: "gw", Label: "proxy"},
			{From: "gw", To: "auth", Label: "gRPC"},
			{From: "gw", To: "user", Label: "gRPC"},
			{From: "gw", To: "vehicle", Label: "gRPC"},
			{From: "gw", To: "matching", Label: "gRPC"},
			{From: "gw", To: "notif", Label: "gRPC"},
			{From: "gw", To: "media", Label: "gRPC"},
			{From: "matching", To: "rabbit", Label: "phát sự kiện", Dashed: true},
			{From: "rabbit", To: "notif", Label: "tiêu thụ", Dashed: true},
			{From: "matching", To: "kafka", Label: "nhật ký", Dashed: true},
			{From: "kafka", To: "wallet", Label: "đặt cọc", Dashed: true},
			{From: "vehicle", To: "redis", Label: "GEO"},
			{From: "user", To: "redis", Label: "cache"},
			{From: "matching", To: "nats", Label: "realtime", Dashed: true},
		},
	}
}

func serviceLayering() Diagram {
	return Diagram{
		Name:    "service-layering",
		Title:   "Phân tầng bên trong mỗi service",
		Caption: "Mũi tên chỉ chiều phụ thuộc. biz không biết gì về gRPC lẫn cơ sở dữ liệu.",
		Nodes: []Node{
			{ID: "cmd", Label: "cmd/\nkhởi động + interceptor", Kind: KindLayer, Col: 1, Row: 0},
			{ID: "di", Label: "internal/di/\nnơi DUY NHẤT ráp mọi thứ", Kind: KindLayer, Col: 1, Row: 1},
			{ID: "ctrl", Label: "internal/controller/\nvỏ gRPC", Kind: KindLayer, Col: 1, Row: 2},
			{ID: "biz", Label: "internal/biz/\nluật nghiệp vụ", Kind: KindLayer, Col: 1, Row: 3},
			{ID: "repoif", Label: "biz.Repo (interface)\nkhai báo ở phía tiêu thụ", Kind: KindNote, Col: 2, Row: 3},
			{ID: "repo", Label: "internal/repo/\ncài đặt truy cập dữ liệu", Kind: KindLayer, Col: 1, Row: 4},
			{ID: "ent", Label: "ent/ (dao)\nPostgres / MySQL", Kind: KindStore, Col: 0, Row: 5},
			{ID: "redis", Label: "Redis\ncache / GEO", Kind: KindStore, Col: 2, Row: 5},
			{ID: "mapper", Label: "internal/mapper/\ngoverter sinh phần thân", Kind: KindNote, Col: 0, Row: 3},
			{ID: "entity", Label: "internal/entity/\nmodel nghiệp vụ", Kind: KindNote, Col: 0, Row: 2},
		},
		Edges: []Edge{
			{From: "cmd", To: "di"},
			{From: "di", To: "ctrl", Label: "tiêm"},
			{From: "ctrl", To: "biz", Label: "gọi"},
			{From: "biz", To: "repoif", Label: "phụ thuộc"},
			{From: "repo", To: "repoif", Label: "cài đặt", Dashed: true},
			{From: "repo", To: "ent"},
			{From: "repo", To: "redis"},
			{From: "mapper", To: "entity", Label: "dao ↔ entity ↔ dto", Dashed: true},
		},
	}
}
