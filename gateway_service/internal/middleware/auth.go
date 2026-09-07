package middleware

import (
	"strings"

	"gateway_service/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/logistic/pkg/authn"
)

const (
	HeaderUserID   = "X-User-Id"
	HeaderUserRole = "X-User-Role"
	HeaderEmail    = "X-User-Email"

	CtxIdentity = "ctx_identity"
	CtxUserID   = "ctx_user_id"
	CtxUserRole = "ctx_user_role"
)

func StripClientIdentity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		for _, h := range []string{HeaderUserID, HeaderUserRole, HeaderEmail} {
			ctx.Request.Header.Del(h)
		}
		ctx.Next()
	}
}

type Authenticator struct {
	verifier *authn.Verifier
}

func NewAuthenticator(verifier *authn.Verifier) *Authenticator {
	return &Authenticator{verifier: verifier}
}

func (a *Authenticator) Required() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, err := a.parse(ctx)
		if err != nil {
			respondAuthError(ctx, err)
			return
		}
		a.store(ctx, claims)
		ctx.Next()
	}
}

func (a *Authenticator) Optional() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		raw := extractToken(ctx)
		if raw == "" {
			ctx.Next()
			return
		}
		claims, err := a.verifier.VerifyAccess(raw)
		if err != nil {
			respondAuthError(ctx, err)
			return
		}
		a.store(ctx, claims)
		ctx.Next()
	}
}

func (a *Authenticator) parse(ctx *gin.Context) (*authn.Claims, error) {
	raw := extractToken(ctx)
	if raw == "" {
		return nil, authn.ErrTokenInvalid
	}
	return a.verifier.VerifyAccess(raw)
}

func (a *Authenticator) store(ctx *gin.Context, claims *authn.Claims) {
	userID, err := claims.SubjectUUID()
	if err != nil {
		respondAuthError(ctx, authn.ErrTokenInvalid)
		return
	}

	id := authn.Identity{UserID: userID, Role: claims.Role, Email: claims.Email}
	ctx.Set(CtxIdentity, id)
	ctx.Set(CtxUserID, userID.String())
	ctx.Set(CtxUserRole, claims.Role)

	ctx.Request = ctx.Request.WithContext(authn.WithIdentity(ctx.Request.Context(), id))

	ctx.Request.Header.Set(HeaderUserID, userID.String())
	ctx.Request.Header.Set(HeaderUserRole, claims.Role)
	ctx.Request.Header.Set(HeaderEmail, claims.Email)
}

func extractToken(ctx *gin.Context) string {
	if h := ctx.GetHeader("Authorization"); h != "" {
		if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
			return strings.TrimSpace(h[7:])
		}
		return ""
	}
	if c, err := ctx.Cookie("access_token"); err == nil {
		return c
	}
	return ""
}

func respondAuthError(ctx *gin.Context, err error) {
	switch err {
	case authn.ErrTokenExpired:

		ctx.AbortWithStatusJSON(401, response.ErrorBody{
			Error: response.ErrorDetail{
				Code:    "TOKEN_EXPIRED",
				Message: "phiên đã hết hạn, hãy làm mới token",
			},
			RequestID: ctx.GetString(response.HeaderRequestID),
		})
	default:
		response.Unauthorized(ctx, "cần đăng nhập để truy cập")
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(ctx *gin.Context) {
		id, ok := CurrentIdentity(ctx)
		if !ok || id.IsZero() {
			response.Unauthorized(ctx, "thiếu thông tin xác thực")
			return
		}
		if _, found := allowed[id.Role]; !found {
			response.Forbidden(ctx, "tài khoản không có quyền truy cập khu vực này")
			return
		}
		ctx.Next()
	}
}

func CurrentIdentity(ctx *gin.Context) (authn.Identity, bool) {
	v, exists := ctx.Get(CtxIdentity)
	if !exists {
		return authn.Identity{}, false
	}
	id, ok := v.(authn.Identity)
	return id, ok
}

func CurrentUserID(ctx *gin.Context) string {
	id, ok := CurrentIdentity(ctx)
	if !ok {
		return ""
	}
	return id.UserID.String()
}

func CurrentUserUUID(ctx *gin.Context) uuid.UUID {
	id, ok := CurrentIdentity(ctx)
	if !ok {
		return uuid.Nil
	}
	return id.UserID
}

func IsAdmin(ctx *gin.Context) bool {
	id, ok := CurrentIdentity(ctx)
	return ok && id.Role == authn.RoleAdmin
}