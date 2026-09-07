package biz

import (
	"context"

	"auth_service/internal/entity"

	"github.com/google/uuid"
)

type AuthRepo interface {
	FindByEmail(ctx context.Context, email string) (*entity.UserProfile, string, error)

	FindByID(ctx context.Context, id uuid.UUID) (*entity.UserProfile, error)

	Save(ctx context.Context, user entity.UserRegister, hashedPassword string) (*entity.UserProfile, error)

	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type SessionRepo interface {
	Create(ctx context.Context, session entity.RefreshSession) error
	Get(ctx context.Context, id uuid.UUID) (*entity.RefreshSession, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error

	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error

	DeleteExpired(ctx context.Context) (int, error)
}