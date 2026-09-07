package app

import (
	"context"

	"kyc_service/internal/entity"

	"github.com/google/uuid"
)

type KycUseCase interface {
	SubmitKYC(ctx context.Context, param *entity.SubmitKYCParam) (*entity.KycDocument, error)
	GetKYC(ctx context.Context, userID uuid.UUID) (*entity.KycDocument, error)
	ListPendingKYC(ctx context.Context, page, pageSize int) (*entity.ListPendingKYCResult, error)
	ReviewKYC(ctx context.Context, param *entity.ReviewKYCParam) (*entity.KycDocument, error)
	CountPendingKYC(ctx context.Context) (int64, error)
}

type kycImpl struct {
	repo KycRepository
}

func NewKycUseCase(repo KycRepository) KycUseCase {
	return &kycImpl{repo: repo}
}

func (k *kycImpl) SubmitKYC(ctx context.Context, param *entity.SubmitKYCParam) (*entity.KycDocument, error) {
	if param.UserID == uuid.Nil {
		return nil, entity.ErrInvalidUserID
	}
	if param.Status == "" {
		param.Status = entity.KycPending
	}
	if !entity.IsValidKycStatus(param.Status) {
		return nil, entity.ErrInvalidKycStatus.WithDetail("status", param.Status)
	}

	return k.repo.Upsert(ctx, param)
}

func (k *kycImpl) GetKYC(ctx context.Context, userID uuid.UUID) (*entity.KycDocument, error) {
	if userID == uuid.Nil {
		return nil, entity.ErrInvalidUserID
	}
	return k.repo.GetByUserID(ctx, userID)
}

func (k *kycImpl) ListPendingKYC(ctx context.Context, page, pageSize int) (*entity.ListPendingKYCResult, error) {
	page, pageSize, _ = entity.NormalizePaging(page, pageSize)
	items, total, err := k.repo.ListPending(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &entity.ListPendingKYCResult{
		Items:      items,
		Pagination: entity.BuildPagination(page, pageSize, total),
	}, nil
}

func (k *kycImpl) ReviewKYC(ctx context.Context, param *entity.ReviewKYCParam) (*entity.KycDocument, error) {
	if param.UserID == uuid.Nil {
		return nil, entity.ErrInvalidUserID
	}

	current, err := k.repo.GetByUserID(ctx, param.UserID)
	if err != nil {
		return nil, err
	}

	// Máy trạng thái: Chỉ hồ sơ pending mới được duyệt. Duyệt hai lần là lỗi nghiệp vụ.
	if current.Status != entity.KycPending {
		return nil, entity.ErrKycAlreadyReviewed.WithDetail("current_status", current.Status)
	}

	return k.repo.Review(ctx, param)
}

func (k *kycImpl) CountPendingKYC(ctx context.Context) (int64, error) {
	return k.repo.CountPending(ctx)
}
