package entity

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("auth: email already registered")
	ErrInvalidCredentials  = errors.New("auth: invalid email or password")
	ErrTokenGenerationFail = errors.New("auth: failed to generate auth token")
	ErrInvalidToken        = errors.New("auth: invalid token")
	ErrSessionRevoked      = errors.New("auth: phiên đăng nhập đã bị thu hồi")
)
