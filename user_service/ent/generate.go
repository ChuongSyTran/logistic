package ent

// Ghim đúng version ent trong lệnh generate.
//
// Không dùng `go run entgo.io/ent/cmd/ent` trần: khi đó Go phải giải cmd/ent từ
// go.mod của chính service, mà `go mod tidy` lại xoá các dependency chỉ dùng cho
// codegen (cobra, tablewriter...) vì không file .go nào import chúng. Dạng có
// @version chạy tách biệt khỏi module hiện tại nên không dính vòng luẩn quẩn đó.
//go:generate go run entgo.io/ent/cmd/ent@v0.14.6 generate ./schema
