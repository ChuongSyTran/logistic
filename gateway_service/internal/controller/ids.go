package controller

import (
	"gateway_service/internal/middleware"
	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/logistic/pkg/uuidx"
)

func pathID(ctx *gin.Context, keys ...string) ([]byte, bool) {
	for _, key := range keys {
		raw := ctx.Param(key)
		if raw == "" {
			continue
		}
		id, err := uuidx.Parse(raw)
		if err != nil {
			response.BadRequest(ctx, "INVALID_ID", "tham số "+key+" phải là UUID hợp lệ")
			return nil, false
		}
		return id, true
	}

	response.BadRequest(ctx, "MISSING_ID", "thiếu định danh trong đường dẫn")
	return nil, false
}

func bodyID(ctx *gin.Context, field, raw string, required bool) ([]byte, bool) {
	if raw == "" {
		if required {
			response.BadRequest(ctx, "MISSING_"+upperSnake(field), "thiếu "+field)
			return nil, false
		}
		return nil, true
	}

	id, err := uuidx.Parse(raw)
	if err != nil {
		response.BadRequest(ctx, "INVALID_"+upperSnake(field), field+" phải là UUID hợp lệ")
		return nil, false
	}
	return id, true
}

func selfID(ctx *gin.Context) []byte {
	return uuidx.ToBytes(middleware.CurrentUserUUID(ctx))
}

func resolveOwnID(ctx *gin.Context, keys ...string) ([]byte, bool) {
	for _, key := range keys {
		if raw := ctx.Param(key); raw != "" {
			id, err := uuidx.Parse(raw)
			if err != nil {
				response.BadRequest(ctx, "INVALID_ID", "tham số "+key+" phải là UUID hợp lệ")
				return nil, false
			}
			return id, true
		}
	}

	if raw := ctx.Query("user_id"); raw != "" {
		id, err := uuidx.Parse(raw)
		if err != nil {
			response.BadRequest(ctx, "INVALID_ID", "user_id phải là UUID hợp lệ")
			return nil, false
		}
		return id, true
	}

	self := selfID(ctx)
	if uuidx.IsZero(self) {
		response.Unauthorized(ctx, "không xác định được người dùng")
		return nil, false
	}
	return self, true
}

func requireSelfOrAdmin(ctx *gin.Context, targetID []byte) bool {
	if middleware.IsAdmin(ctx) {
		return true
	}

	self := selfID(ctx)
	if uuidx.IsZero(self) {
		response.Unauthorized(ctx, "cần đăng nhập")
		return false
	}
	if uuidx.String(self) != uuidx.String(targetID) {
		response.Forbidden(ctx, "không thể thao tác trên dữ liệu của người dùng khác")
		return false
	}
	return true
}

func upperSnake(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c-32)
		case c == '-' || c == ' ':
			out = append(out, '_')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}

func queryID(ctx *gin.Context, key string) ([]byte, bool) {
	raw := ctx.Query(key)
	if raw == "" {
		return nil, true
	}
	id, err := uuidx.Parse(raw)
	if err != nil {
		response.BadRequest(ctx, "INVALID_"+upperSnake(key), key+" phải là UUID hợp lệ")
		return nil, false
	}
	return id, true
}