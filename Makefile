# ==============================================================================
# Variables & Configuration
# ==============================================================================
PROTO_DIR = api
PROTO_GEN_OUT = internal/dto/gen/go

ENT_DIR = ./ent
ENT_SCHEMA_DIR = $(ENT_DIR)/schema

# Metadata
SERVICE_NAME = goBackend

.PHONY: all help api clean-api db clean-db update-db build-all

# ==============================================================================
# Global Targets
# ==============================================================================

## all: Dọn dẹp và tái tạo toàn bộ Code (API & DB)
all: clean-api clean-db api db
	@echo "==> [$(SERVICE_NAME)] All artifacts generated successfully."

## help: Hiển thị hướng dẫn sử dụng
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# API & Protobuf Layer
# ==============================================================================

## api: Generate Go & gRPC code từ file .proto
api: clean-api
	@echo "==> Generating Protobuf code and Swagger JSON..."
	@mkdir -p $(PROTO_GEN_OUT)
	@mkdir -p matching_service/docs
	protoc -I=$(PROTO_DIR) -I=. \
		--go_out=$(PROTO_GEN_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_GEN_OUT) --go-grpc_opt=paths=source_relative \
		--openapiv2_out=matching_service/docs --openapiv2_opt=logtostderr=true \
		$(shell find $(PROTO_DIR) -name "*.proto")
	@echo "    Protobuf and Swagger generation complete."

## clean-api: Xóa các file .pb.go cũ để tránh xung đột
clean-api:
	@echo "==> Cleaning old Protobuf files..."
	@if [ -d "$(PROTO_GEN_OUT)" ]; then \
		find $(PROTO_GEN_OUT) -type f -name '*.pb.go' -delete; \
	fi
	@echo "    Cleanup complete."

# ==============================================================================
# Database & Ent Layer
# ==============================================================================

.PHONY: ent ent-new

ent:
	go generate ./ent

ent-new:
	go run -mod=mod entgo.io/ent/cmd/ent new ${name}


# ==============================================================================
# Auth — khoá ký JWT
# ==============================================================================

.PHONY: auth-keys auth-keys-show

## auth-keys: Sinh cặp khoá RSA cho JWT (private cho auth_service, public cho gateway)
auth-keys:
	@bash scripts/generate-jwt-keys.sh

# ==============================================================================
# Swagger
# ==============================================================================

.PHONY: swagger

## swagger: Sinh lại tài liệu API từ annotation trong controller
swagger:
	@swag init -g cmd/main.go -d gateway_service -o gateway_service/docs --parseDependency --parseInternal
	@echo "    swagger.json / swagger.yaml / docs.go đã cập nhật"

# ==============================================================================
# Postman
# ==============================================================================

.PHONY: postman

## postman: Sinh lại logistic.postman_collection.json từ tools/postman
postman:
	@go run ./tools/postman -o logistic.postman_collection.json
	@go test ./tools/postman/

## auth-keys-show: In dấu vân tay khoá để đối chiếu hai bên có khớp không
# Triệu chứng khi lệch khoá là "mọi token bỗng dưng không hợp lệ" — rất khó đoán
# nguyên nhân, nên có sẵn một lệnh đối chiếu là đáng.
auth-keys-show:
	@echo "public key trong secrets/:"
	@openssl rsa -pubin -in secrets/jwt_public.pem -outform DER 2>/dev/null | openssl dgst -sha256 | cut -d" " -f2
	@echo "public key suy ra từ private key:"
	@openssl rsa -in secrets/jwt_private.pem -pubout -outform DER 2>/dev/null | openssl dgst -sha256 | cut -d" " -f2



# ==============================================================================
# Modules / Build Verification
# ==============================================================================

GO_SERVICES = auth_service gateway_service matching_service media_service notification_service user_service vehicle_service wallet_service kyc_service

.PHONY: verify-modules tidy-modules docker-build

## verify-modules: Kiểm tra MỖI service tự build được khi KHÔNG có go.work
# Đây là cái bẫy sách "Learning Go" (2nd ed., ch.10) cảnh báo: go.work gộp chung
# module graph nên ở máy local cái gì cũng chạy, nhưng Docker build với GOWORK=off
# chỉ đọc go.mod + go.sum RIÊNG của từng module. Thiếu hash trong go.sum là vỡ
# ngay trên CI. Chạy lệnh này TRƯỚC khi push để bắt lỗi tại chỗ.
# (CI cũng chạy đúng bước này trong job build-push.)
verify-modules:
	@for s in $(GO_SERVICES); do \
		printf "==> %-18s " $$s; \
		( cd $$s && GOWORK=off GOFLAGS=-mod=readonly CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
			go build -o /dev/null ./cmd ) && echo "OK (standalone)"; \
	done
	@echo "==> Tất cả module đều build được mà không cần go.work."

## tidy-modules: Chạy go mod tidy cho từng module, KHÔNG qua workspace
# Phải tắt go.work, nếu không tidy sẽ tính theo module graph gộp của workspace
# và ghi ra go.sum thiếu/thừa so với lúc build độc lập trong Docker.
tidy-modules:
	@for s in api pkg $(GO_SERVICES); do \
		echo "==> go mod tidy: $$s"; \
		( cd $$s && GOWORK=off go mod tidy ); \
	done

## docker-build: Build thử toàn bộ image ở local (context = gốc repo)
docker-build:
	@for s in $(GO_SERVICES); do \
		echo "==> docker build $$s"; \
		docker build -f $$s/Dockerfile -t $$(echo $$s | tr '_' '-'):dev . ; \
	done

# ==============================================================================
# Code generation cho các service dùng ent + goverter
# ==============================================================================
# Cả hai công cụ đều được GHIM VERSION trong lệnh `go run ...@vX.Y.Z`:
#  - ent   : `go mod tidy` xoá dependency chỉ dùng cho codegen (cobra, tablewriter)
#            vì không file .go nào import chúng, nên dạng không ghim sẽ hỏng.
#  - goverter: v1.9.0 kéo golang.org/x/tools v0.25.0 — bản này KHÔNG biên dịch
#            được với Go 1.26 ("invalid array length"). v1.9.4 thì chạy tốt.

# CỐ Ý thiếu matching_service và wallet_service: hai service này sinh ent bằng
# cờ --feature riêng, khai trong <service>/ent/generate.go (matching dùng
# sql/execquery,intercept; wallet dùng sql/modifier cho SELECT ... FOR UPDATE).
# Regen chúng bằng `go generate ./ent` trong thư mục service, KHÔNG phải ent-all.
ENT_SERVICES = notification_service user_service vehicle_service kyc_service
MAPPER_SERVICES = matching_service notification_service user_service vehicle_service wallet_service

ENT_VERSION = v0.14.6
GOVERTER_VERSION = v1.9.4

.PHONY: ent-all mapper-all gen-all

## ent-all: Sinh lại toàn bộ code ent (dao) từ schema của từng service
ent-all:
	@for s in $(ENT_SERVICES); do \
		echo "==> ent generate: $$s"; \
		( cd $$s && GOWORK=off go run entgo.io/ent/cmd/ent@$(ENT_VERSION) generate ./ent/schema ); \
	done

## mapper-all: Sinh lại toàn bộ mapper goverter (dao <-> entity <-> dto)
mapper-all:
	@for s in $(MAPPER_SERVICES); do \
		echo "==> goverter gen: $$s"; \
		( cd $$s && GOWORK=off go run github.com/jmattheis/goverter/cmd/goverter@$(GOVERTER_VERSION) gen ./internal/mapper ); \
	done

## gen-all: proto + ent + mapper. Chạy sau khi sửa .proto hoặc ent/schema
gen-all: proto-buf ent-all mapper-all
	@echo "==> Đã sinh lại toàn bộ code generate."

.PHONY: proto-buf
## proto-buf: Sinh code Go + swagger từ .proto bằng buf (dùng remote plugin)
proto-buf:
	@echo "==> buf generate..."
	@( cd api && buf lint && buf generate )
	@echo "    Xong. Nhớ chạy 'make tidy-modules' nếu thêm import mới."

# ==============================================================================
# Tài liệu
# ==============================================================================

.PHONY: docs-lint

## docs-lint: Kiểm tra liên kết chết + quy ước đặt tên trong docs/
# Tài liệu không được biên dịch nên link chết là lỗi im lặng. Chạy trước khi push
# nếu có đổi tên hoặc di chuyển file tài liệu.
docs-lint:
	@GOWORK=off go run ./tools/doclint

.PHONY: diagrams

## diagrams: Sinh lại .drawio + .svg + trang HTML từ tools/diagrams/
# Mỗi tài liệu trong architecture/, services/, flows/ có đúng một sơ đồ cùng tên.
# Sửa spec trong tools/diagrams/ rồi chạy lệnh này — đừng sửa tay file sinh ra.
diagrams:
	@GOWORK=off go run ./tools/diagrams
