# Quan sát hệ thống với OpenTelemetry

Trace đi từ lúc request chạm gateway tới lúc rời khỏi service cuối cùng.

---

## Thành phần

| Thành phần | Vai trò |
|---|---|
| `pkg/tracer` | Khởi tạo OTLP exporter, gán `service.name`, cài propagator |
| `otelgin` | Mở span gốc cho mỗi request HTTP ở gateway |
| `otelgrpc` | Nối span giữa gateway và service nội bộ qua gRPC metadata |
| Jaeger | Nhận trace qua OTLP gRPC, hiển thị tại `:16686` |

```
client ──HTTP──► gateway_service ──gRPC──► service nội bộ
                      │                          │
                      └────── OTLP :4317 ────────┘
                                  ▼
                               jaeger
```

## Xem trace

```bash
docker compose up -d jaeger
```

Giao diện: <http://localhost:16686>

Endpoint xuất trace được cấu hình bằng `OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4317`.
Khi biến này rỗng, SDK dùng mặc định `localhost:4317`; trong container không có
tiến trình nào nghe ở đó nên toàn bộ span bị loại bỏ. `BatchSpanProcessor` không
ghi log khi gửi thất bại, nên triệu chứng duy nhất là giao diện Jaeger trống.

> **Production.** Jaeger all-in-one giữ trace trong bộ nhớ và mất khi restart.
> Môi trường thật cần OTel Collector đứng trước một backend có lưu trữ, kèm lấy
> mẫu. Giữ 100% span ở nhịp báo GPS sẽ tốn hơn chính hệ thống đang được đo.

## Span gốc

Span gốc mở tại middleware đầu tiên của gateway:

```go
engine.Use(
    otelgin.Middleware("gateway_service"),
    middleware.RequestID(),
    ...
)
```

Vị trí này quyết định phạm vi trace: toàn bộ tầng HTTP — bind, validate, xác
thực, phân quyền — nằm trong span gốc, và mọi lời gọi gRPC phát sinh trong một
request đều là span con của cùng một cha.

`otelgin` cũng đọc header `traceparent` nếu client gửi lên, nên trace bắt đầu từ
app di động nối liền được với trace phía máy chủ.

## Một mã truy vết duy nhất

`X-Request-ID` lấy giá trị từ `trace_id` của span đang chạy:

```go
if sc := trace.SpanContextFromContext(ctx.Request.Context()); sc.IsValid() {
    id = sc.TraceID().String()
}
```

Mã này xuất hiện ở ba nơi cho cùng một request: header phản hồi, dòng access log,
và trường `request_id` trong body lỗi. Cầm một mã tra được cả log lẫn trace.

Khi chưa có tracer (chạy local không bật Jaeger), gateway sinh UUID v7 thay thế.

Client gửi sẵn `X-Request-ID` thì gateway dùng lại, giữ chuỗi truy vết xuyên
nhiều hệ thống.

## Span ở từng service

| Service | Tracer | Instrumentation | Chain interceptor |
|---|---|---|---|
| gateway_service | có | otelgin + otelgrpc client | middleware HTTP riêng |
| auth_service | có | otelgrpc server | có |
| user_service | có | otelgrpc server | có |
| vehicle_service | có | otelgrpc server | có |
| matching_service | có | otelgrpc server | có |
| media_service | có | otelgrpc server | có |
| notification_service | có | otelgrpc server | có |
| wallet_service | có | otelgrpc server | có |

## Hình dạng một trace

```
gateway_service  POST /api/v1/vehicles/{id}/location          [span gốc]
├─ verify JWT (~31µs, không chạm mạng, không chạm DB)
└─ vehicle_service  VehicleService/ReportLocation             [span con]
   ├─ redis GEOADD
   └─ postgres UPDATE vehicle_locations
```

Hai mức đầu có sẵn từ instrumentation tự động. Hai mức trong cùng chưa được
instrument: trace cho biết "gọi vehicle_service mất 40ms" nhưng không cho biết
40ms đó tiêu vào Redis hay Postgres.

## Gắn OTel vào service mới

```go
// 1. Khởi tạo, giữ lại hàm shutdown
shutdownTracer, err := tracer.InitTracer("tên_service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))

// 2. Gắn vào gRPC server
grpcServer := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
    middleware.ChainForService("tên_service"),
)

// 3. Flush lúc tắt
func (a *App) Stop() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    a.shutdown(ctx)
}
```

Bước 3 không bỏ được: `BatchSpanProcessor` gom span theo lô, nên các span cuối
cùng mất theo tiến trình nếu không flush.

## Phạm vi chưa phủ

| Hạng mục | Trạng thái |
|---|---|
| Trace context qua RabbitMQ/Kafka | chưa — trace đứt tại ranh giới hàng đợi, consumer mở trace mới |
| Span cho truy vấn SQL (`ent`) | chưa |
| Lấy mẫu | chưa — hiện giữ 100% span |
| Metrics (RED theo endpoint) | chưa — chỉ suy ra được từ log |

Hạng mục đầu ảnh hưởng lớn nhất, vì toàn bộ luồng thông báo đi qua hàng đợi.
