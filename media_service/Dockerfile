# Build context = gốc repo, KHÔNG phải ./media_service
#   podman build -f media_service/Dockerfile -t media_service .
# Giải thích từng bước + cơ chế cache: xem docs/operations/build-process.md

FROM docker.io/library/golang:1.26-alpine AS builder

ENV GOWORK=off \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /src

COPY api/go.mod api/go.sum ./api/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY media_service/go.mod media_service/go.sum ./media_service/

WORKDIR /src/media_service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

WORKDIR /src
COPY api/ ./api/
COPY pkg/ ./pkg/
COPY media_service/ ./media_service/

WORKDIR /src/media_service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/media_service_bin ./cmd


FROM docker.io/library/alpine:3.24

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/media_service_bin .

EXPOSE 9002

CMD ["./media_service_bin"]
