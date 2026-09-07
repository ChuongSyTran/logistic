package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserProfile struct {
	Id       uuid.UUID
	Email    string
	FullName *string
	Avatar   *string

	Role      string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

type UserRegister struct {
	Email    string
	FullName string
	Password string
	GoogleID string
	Role     string
}

type UserLogin struct {
	Email    string
	Password string
}

type AuthTokenPair struct {
	AccessToken  string
	RefreshToken string

	ExpiresAt int64
}

type RefreshSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	UsedAt    *time.Time
}

func (s *RefreshSession) IsUsable(now time.Time) bool {
	if s == nil || s.RevokedAt != nil {
		return false
	}
	return now.Before(s.ExpiresAt)
}