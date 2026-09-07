package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// @title 				Logistics OS — Gateway API
// @version 			1.0
// @description 		API Gateway trung tâm cho hệ thống Logistics OS. Nhận toàn bộ HTTP request từ client và phân luồng tới các microservice nội bộ qua gRPC.
// @description
// @description 		**Các service được proxy:**
// @description 		- **Auth** (`/api/v1/auth/*`) — Đăng ký, đăng nhập, OAuth2 Google, xác thực token
// @description 		- **User** (`/api/v1/users/*`) — Quản lý người dùng, hồ sơ tài xế/chủ hàng, KYC
// @description 		- **Vehicle** (`/api/v1/vehicles/*`) — Đăng ký & quản lý phương tiện, báo GPS, tìm xe gần
// @description 		- **Matching** (`/api/v1/matching/*`) — Ghép đơn theo mô hình Bid/Ask
// @description 		- **Notification** (`/api/v1/notifications/*`) — Hộp thư thông báo
// @description 		- **Media** (`/api/v1/media/*`) — Tải file lên Cloudinary
// @host 				localhost:8080
// @BasePath 			/
func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize Gateway App: %v", err)
	}

	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start Gateway Service: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Stop()
}