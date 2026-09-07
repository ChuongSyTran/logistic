package app

import (
	"context"

	"kyc_service/internal/entity"

	"github.com/google/uuid"
)

type KycRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.KycDocument, error)
	Upsert(ctx context.Context, param *entity.SubmitKYCParam) (*entity.KycDocument, error)
	Review(ctx context.Context, param *entity.ReviewKYCParam) (*entity.KycDocument, error)
	ListPending(ctx context.Context, page, pageSize int) ([]entity.KycDocument, int64, error)
	CountPending(ctx context.Context) (int64, error)
}
