# Khởi tạo Ent ORM cho service mới

> Tài liệu này hướng dẫn từng bước khởi tạo **[entgo.io/ent](https://entgo.io)** ORM cho một service mới trong project Logistics OS.
> Pattern được rút từ `matching_service` — service đầu tiên đã tích hợp ent thành công.

---

## Mục Lục

1. [Yêu Cầu Trước Khi Bắt Đầu](#1-yêu-cầu-trước-khi-bắt-đầu)
2. [Bước 1 — Cài Đặt Ent CLI](#2-bước-1--cài-đặt-ent-cli)
3. [Bước 2 — Khởi Tạo Schema Đầu Tiên](#3-bước-2--khởi-tạo-schema-đầu-tiên)
4. [Bước 3 — Định Nghĩa Fields, Edges, Mixin](#4-bước-3--định-nghĩa-fields-edges-mixin)
5. [Bước 4 — Generate Code](#5-bước-4--generate-code)
6. [Bước 5 — Kết Nối Database & Auto Migrate](#6-bước-5--kết-nối-database--auto-migrate)
7. [Bước 6 — Tạo Mixin Chuẩn (Audit + SoftDelete)](#7-bước-6--tạo-mixin-chuẩn-audit--softdelete)
8. [Cấu Trúc Thư Mục Sau Khi Init](#8-cấu-trúc-thư-mục-sau-khi-init)
9. [Lệnh Tham Khảo Nhanh](#9-lệnh-tham-khảo-nhanh)

---

## 1. Yêu Cầu Trước Khi Bắt Đầu

- **Go** >= 1.22 (project hiện dùng Go 1.26.4)
- Service đã có `go.mod` riêng (mỗi service là một Go module độc lập)
- Database **PostgreSQL** hoặc **MySQL** (project dùng cả hai, tùy service)

---

## 2. Bước 1 — Cài Đặt Ent CLI

```bash
# Chạy tại root folder của service (vd: ./user_service)
go install entgo.io/ent/cmd/ent@latest
```

Kiểm tra cài thành công:

```bash
ent version
# Output: entgo.io/ent v0.14.x
```

---

## 3. Bước 2 — Khởi Tạo Schema Đầu Tiên

```bash
# cd vào thư mục service
cd <service_name>

# Init schema, ent sẽ tạo thư mục ent/schema/ với file Go mẫu
ent new <EntityName>

# Ví dụ cho user_service:
ent new User
ent new UserProfile
```

> **Lưu ý:** Tên entity dùng **PascalCase số ít** (User, Vehicle, Wallet), KHÔNG dùng số nhiều.
> Ngoại lệ: Nếu tên nghiệp vụ buộc phải số nhiều (vd: `Asks`, `Bids` trong matching_service), vẫn chấp nhận được.

---

## 4. Bước 3 — Định Nghĩa Fields, Edges, Mixin

### 4.1. Fields (Cột trong DB)

Mở file `ent/schema/<entity>.go` vừa tạo, thêm fields:

```go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "github.com/google/uuid"
)

type User struct {
    ent.Schema
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        // Dùng UUID v7 cho Primary Key (có tính chất thời gian, sắp xếp tốt hơn)
        field.UUID("id", uuid.UUID{}).
            Default(func() uuid.UUID { return uuid.Must(uuid.NewV7()) }),

        field.String("email").
            Unique().
            NotEmpty(),

        field.String("full_name").
            MaxLen(100),

        field.String("password_hash"),

        field.String("avatar").
            Optional().
            Nillable(),

        field.Int8("status").
            Default(1).
            Comment("1: Active, 2: Inactive, 3: Banned"),
    }
}
```

### 4.2. Edges (Quan Hệ Giữa Các Entity)

```go
import "entgo.io/ent/schema/edge"

func (User) Edges() []ent.Edge {
    return []ent.Edge{
        // One-to-Many: 1 User có nhiều Vehicles
        edge.To("vehicles", Vehicle.Type),

        // One-to-One: 1 User có 1 Wallet
        edge.To("wallet", Wallet.Type).Unique(),
    }
}
```

### 4.3. Mixin (Tái sử dụng Fields chung)

```go
func (User) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixin.AuditMixin{},      // created_at, created_by, updated_at, updated_by
        mixin.SoftDeleteMixin{}, // deleted_at (soft delete pattern)
    }
}
```

---

## 5. Bước 4 — Generate Code

### 5.1. Tạo file `ent/generate.go`

```go
package ent

//go:generate go run entgo.io/ent/cmd/ent generate --feature sql/execquery,intercept ./schema
```

> **Giải thích features:**
> - `sql/execquery`: Cho phép chạy raw SQL query qua ent client
> - `intercept`: Bật Interceptor pattern (cần cho SoftDelete filter tự động)

### 5.2. Chạy Generate

```bash
cd <service_name>
go generate ./ent
```

Sau khi chạy, ent sẽ tạo ra toàn bộ code CRUD trong `ent/`:

```
ent/
├── client.go          # Ent Client — entry point cho mọi thao tác DB
├── ent.go             # Base types và helpers
├── tx.go              # Transaction support
├── mutation.go        # Mutation builders (auto-generated)
├── generate.go        # Go generate directive
├── runtime.go         # Runtime validators
├── schema/            # ⬅️ Thư mục BẠN TỰ VIẾT — schema definitions
│   ├── user.go
│   └── mixin/
│       ├── audit.go
│       └── soft_delete.go
├── user.go            # Generated: User entity struct
├── user_create.go     # Generated: Create builder
├── user_query.go      # Generated: Query builder
├── user_update.go     # Generated: Update builder
├── user_delete.go     # Generated: Delete builder
├── hook/              # Generated: Hook interfaces
├── intercept/         # Generated: Interceptor interfaces
├── migrate/           # Generated: Auto migration
├── predicate/         # Generated: Where predicates
└── enttest/           # Generated: Test helpers
```

---

## 6. Bước 5 — Kết Nối Database & Auto Migrate

```go
package main

import (
    "context"
    "log"

    "<service_name>/ent"

    // Driver cho PostgreSQL
    _ "github.com/lib/pq"
    // Hoặc driver cho MySQL
    // _ "github.com/go-sql-driver/mysql"
)

func main() {
    // PostgreSQL
    client, err := ent.Open("postgres",
        "host=localhost port=5432 user=admin password=secret dbname=logistics sslmode=disable",
    )
    if err != nil {
        log.Fatalf("failed opening connection to database: %v", err)
    }
    defer client.Close()

    // Auto Migration — tạo bảng nếu chưa có, cập nhật schema nếu thay đổi
    if err := client.Schema.Create(context.Background()); err != nil {
        log.Fatalf("failed creating schema resources: %v", err)
    }

    log.Println("Database schema migrated successfully!")
}
```

> ⚠️ **Production**: Không dùng Auto Migrate trên production.
> Sử dụng **Atlas** (tích hợp sẵn với ent) để tạo migration files có phiên bản:
> ```bash
> atlas migrate diff migration_name \
>   --dir "file://ent/migrate/migrations" \
>   --to "ent://ent/schema" \
>   --dev-url "postgres://admin:secret@localhost:5432/dev?sslmode=disable"
> ```

---

## 7. Bước 6 — Tạo Mixin Chuẩn (Audit + SoftDelete)

Project sử dụng 2 mixin chung cho mọi entity. Copy từ `matching_service` hoặc tạo mới:

### 7.1. AuditMixin (`ent/schema/mixin/audit.go`)

```go
package mixin

import (
    "context"
    "time"

    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/mixin"
    "github.com/google/uuid"
)

type AuditMixin struct {
    mixin.Schema
}

func (AuditMixin) Fields() []ent.Field {
    return []ent.Field{
        field.Time("created_at").Default(time.Now).Immutable(),
        field.UUID("created_by", uuid.UUID{}).Default(func() uuid.UUID {
            return uuid.Must(uuid.NewV7())
        }),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
        field.UUID("updated_by", uuid.UUID{}).Default(func() uuid.UUID {
            return uuid.Must(uuid.NewV7())
        }),
        field.Bool("is_deleted").Default(false),
    }
}

func (AuditMixin) Hooks() []ent.Hook {
    return []ent.Hook{
        func(next ent.Mutator) ent.Mutator {
            return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
                // TODO: Inject user_id từ context vào created_by / updated_by
                return next.Mutate(ctx, m)
            })
        },
    }
}
```

### 7.2. SoftDeleteMixin (`ent/schema/mixin/soft_delete.go`)

```go
package mixin

import (
    "context"
    "fmt"
    "time"

    "entgo.io/ent"
    "entgo.io/ent/dialect/sql"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/mixin"
)

type SoftDeleteMixin struct {
    mixin.Schema
}

func (SoftDeleteMixin) Fields() []ent.Field {
    return []ent.Field{
        field.Time("deleted_at").Optional(),
    }
}

type softDeleteKey struct{}

// SkipSoftDelete cho phép bypass filter soft-delete (dùng khi cần query cả record đã xóa)
func SkipSoftDelete(parent context.Context) context.Context {
    return context.WithValue(parent, softDeleteKey{}, true)
}

func (d SoftDeleteMixin) Interceptors() []ent.Interceptor {
    return []ent.Interceptor{
        ent.InterceptFunc(func(next ent.Querier) ent.Querier {
            return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
                if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
                    return next.Query(ctx, q)
                }
                if f, ok := q.(interface{ WhereP(...func(*sql.Selector)) }); ok {
                    d.P(f)
                }
                return next.Query(ctx, q)
            })
        }),
    }
}

func (d SoftDeleteMixin) Hooks() []ent.Hook {
    return []ent.Hook{
        func(next ent.Mutator) ent.Mutator {
            return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
                if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
                    return next.Mutate(ctx, m)
                }

                // UPDATE: thêm WHERE deleted_at IS NULL
                if m.Op().Is(ent.OpUpdateOne | ent.OpUpdate) {
                    if f, ok := m.(interface{ WhereP(...func(*sql.Selector)) }); ok {
                        d.P(f)
                    }
                    return next.Mutate(ctx, m)
                }

                // DELETE → chuyển thành UPDATE deleted_at = now()
                if m.Op().Is(ent.OpDeleteOne | ent.OpDelete) {
                    if f, ok := m.(interface{ WhereP(...func(*sql.Selector)) }); ok {
                        d.P(f)
                    }
                    mx, ok := m.(interface{ SetOp(ent.Op) })
                    if !ok {
                        return nil, fmt.Errorf("unexpected mutation type %T", m)
                    }
                    mx.SetOp(ent.OpUpdate)
                    if err := m.SetField("deleted_at", time.Now()); err != nil {
                        return nil, err
                    }
                    return next.Mutate(ctx, m)
                }

                return next.Mutate(ctx, m)
            })
        },
    }
}

func (d SoftDeleteMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
    w.WhereP(sql.FieldIsNull(d.Fields()[0].Descriptor().Name))
}
```

---

## 8. Cấu Trúc Thư Mục Sau Khi Init

```
<service_name>/
├── cmd/
│   ├── main.go
│   └── app.go
├── ent/                          # ⬅️ Toàn bộ ent ORM
│   ├── generate.go               # go:generate directive
│   ├── schema/                   # Schema definitions (bạn viết)
│   │   ├── <entity>.go
│   │   └── mixin/
│   │       ├── audit.go
│   │       └── soft_delete.go
│   ├── client.go                 # (generated)
│   ├── ent.go                    # (generated)
│   ├── tx.go                     # (generated)
│   ├── mutation.go               # (generated)
│   ├── <entity>.go               # (generated)
│   ├── <entity>_create.go        # (generated)
│   ├── <entity>_query.go         # (generated)
│   ├── <entity>_update.go        # (generated)
│   ├── <entity>_delete.go        # (generated)
│   ├── hook/                     # (generated)
│   ├── intercept/                # (generated)
│   ├── migrate/                  # (generated)
│   ├── predicate/                # (generated)
│   ├── runtime/                  # (generated)
│   └── enttest/                  # (generated)
├── internal/
│   ├── handler/
│   ├── repository/
│   └── service/
├── go.mod
└── go.sum
```

---

## 9. Lệnh Tham Khảo Nhanh

| Thao Tác | Lệnh |
|---|---|
| Cài ent CLI | `go install entgo.io/ent/cmd/ent@latest` |
| Tạo entity mới | `ent new <EntityName>` |
| Generate code | `go generate ./ent` |
| Thêm dependency ent | `go get entgo.io/ent@latest` |
| Thêm driver PostgreSQL | `go get github.com/lib/pq` |
| Thêm driver MySQL | `go get github.com/go-sql-driver/mysql` |
| Thêm UUID support | `go get github.com/google/uuid` |
| Tạo Atlas migration | `atlas migrate diff <name> --dir "file://ent/migrate/migrations" --to "ent://ent/schema" --dev-url "<db_url>"` |

---

## Tham Khảo

- **Ent Official Docs:** https://entgo.io/docs/getting-started
- **Atlas Migration:** https://entgo.io/docs/versioned-migrations
- **Matching Service (Reference):** `./matching_service/ent/schema/` — xem cách project hiện tại implement ent
