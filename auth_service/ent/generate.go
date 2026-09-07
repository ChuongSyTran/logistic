package ent

// Feature sql/execquery bật client.QueryContext/ExecContext — cần cho lượt
// "SELECT 1" kiểm tra kết nối lúc khởi động ở internal/adapter/persistence.
//go:generate go run entgo.io/ent/cmd/ent@v0.14.6 generate --feature sql/execquery ./schema
