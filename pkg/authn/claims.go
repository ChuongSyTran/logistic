package authn

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	TokenAccess  TokenType = "access"
	TokenRefresh TokenType = "refresh"
)

const (
	RoleDriver  = "driver"
	RoleShipper = "shipper"
	RoleAdmin   = "admin"
)

type Claims struct {
	Email string    `json:"email,omitempty"`
	Role  string    `json:"role,omitempty"`
	Type  TokenType `json:"typ"`

	jwt.RegisteredClaims
}

var (
	ErrTokenInvalid   = errors.New("authn: token không hợp lệ")
	ErrTokenExpired   = errors.New("authn: token đã hết hạn")
	ErrWrongTokenType = errors.New("authn: sai loại token")
	ErrWrongIssuer    = errors.New("authn: token không do hệ thống này phát hành")
	ErrNoKey          = errors.New("authn: thiếu khoá ký/xác thực")
)

const (
	DefaultAccessTTL  = 15 * time.Minute
	DefaultRefreshTTL = 7 * 24 * time.Hour
)

const Issuer = "logistic-auth"