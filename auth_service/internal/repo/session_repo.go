package repo

import (
	"context"
	"fmt"
	"time"

	"auth_service/ent"
	"auth_service/ent/refreshtoken"
	"auth_service/internal/biz"
	"auth_service/internal/entity"

	"github.com/google/uuid"
)

type sessionRepoImpl struct {
	client *ent.Client
}

func NewSessionRepo(client *ent.Client) biz.SessionRepo {
	return &sessionRepoImpl{client: client}
}

func (r *sessionRepoImpl) Create(ctx context.Context, s entity.RefreshSession) error {
	_, err := r.client.RefreshToken.Create().
		SetID(s.ID).
		SetUserID(s.UserID).
		SetExpiresAt(s.ExpiresAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("repo session create: %w", err)
	}
	return nil
}

func (r *sessionRepoImpl) Get(ctx context.Context, id uuid.UUID) (*entity.RefreshSession, error) {
	row, err := r.client.RefreshToken.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("repo session get: %w", biz.ErrSessionRevoked)
		}
		return nil, fmt.Errorf("repo session get: %w", err)
	}

	return &entity.RefreshSession{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: row.RevokedAt,
		UsedAt:    row.UsedAt,
	}, nil
}

func (r *sessionRepoImpl) MarkUsed(ctx context.Context, id uuid.UUID) error {
	affected, err := r.client.RefreshToken.Update().
		Where(
			refreshtoken.IDEQ(id),
			refreshtoken.UsedAtIsNil(),
			refreshtoken.RevokedAtIsNil(),
		).
		SetUsedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("repo session markUsed: %w", err)
	}
	if affected == 0 {
		return biz.ErrSessionRevoked
	}
	return nil
}

func (r *sessionRepoImpl) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.client.RefreshToken.Update().
		Where(refreshtoken.IDEQ(id), refreshtoken.RevokedAtIsNil()).
		SetRevokedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("repo session revoke: %w", err)
	}
	return nil
}

func (r *sessionRepoImpl) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.client.RefreshToken.Update().
		Where(refreshtoken.UserIDEQ(userID), refreshtoken.RevokedAtIsNil()).
		SetRevokedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("repo session revokeAll: %w", err)
	}
	return nil
}

func (r *sessionRepoImpl) DeleteExpired(ctx context.Context) (int, error) {
	n, err := r.client.RefreshToken.Delete().
		Where(refreshtoken.ExpiresAtLT(time.Now())).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("repo session deleteExpired: %w", err)
	}
	return n, nil
}