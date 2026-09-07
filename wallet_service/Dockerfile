# Build context = gốc repo, KHÔNG phải ./wallet_service
#   podman build -f wallet_service/Dockerfile -t wallet_service .
# Giải thích từng bước + cơ chế cache: xem docs/operations/build-process.md

FROM docker.io/library/golang:1.26-alpine AS builder

ENV GOWORK=off \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /src

COPY api/go.mod api/go.sum ./api/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY wallet_service/go.mod wallet_service/go.sum ./wallet_service/

WORKDIR /src/wallet_service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

WORKDIR /src
COPY api/ ./api/
COPY pkg/ ./pkg/
COPY wallet_service/ ./wallet_service/

WORKDIR /src/wallet_service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/wallet_service_bin ./cmd


FROM docker.io/library/alpine:3.24

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/wallet_service_bin .

EXPOSE 9007

CMD ["./wallet_service_bin"]
