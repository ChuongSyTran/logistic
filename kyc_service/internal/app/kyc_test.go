package app

import (
	"context"
	"errors"
	"testing"

	"kyc_service/internal/entity"

	"github.com/google/uuid"
)

type mockRepo struct {
	doc     *entity.KycDocument
	count   int64
	listErr error
}

func (m *mockRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.KycDocument, error) {
	if m.doc == nil || m.doc.UserID != userID {
		return nil, entity.ErrKycNotFound
	}
	return m.doc, nil
}

func (m *mockRepo) Upsert(ctx context.Context, param *entity.SubmitKYCParam) (*entity.KycDocument, error) {
	m.doc = &entity.KycDocument{
		ID:             uuid.New(),
		UserID:         param.UserID,
		IDCardNumber:   param.IDCardNumber,
		LicenseNumber:  param.LicenseNumber,
		Status:         param.Status,
		Note:           param.Note,
	}
	return m.doc, nil
}

func (m *mockRepo) Review(ctx context.Context, param *entity.ReviewKYCParam) (*entity.KycDocument, error) {
	if m.doc == nil || m.doc.UserID != param.UserID {
		return nil, entity.ErrKycNotFound
	}
	if param.Approved {
		m.doc.Status = entity.KycApproved
	} else {
		m.doc.Status = entity.KycRejected
	}
	m.doc.Note = param.Note
	m.doc.ReviewerID = &param.ReviewerID
	return m.doc, nil
}

func (m *mockRepo) ListPending(ctx context.Context, page, pageSize int) ([]entity.KycDocument, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	if m.doc != nil && m.doc.Status == entity.KycPending {
		return []entity.KycDocument{*m.doc}, 1, nil
	}
	return []entity.KycDocument{}, 0, nil
}

func (m *mockRepo) CountPending(ctx context.Context) (int64, error) {
	return m.count, nil
}

func TestSubmitKYC_Validation(t *testing.T) {
	repo := &mockRepo{}
	useCase := NewKycUseCase(repo)

	// Case 1: Nil UserID
	_, err := useCase.SubmitKYC(context.Background(), &entity.SubmitKYCParam{
		UserID: uuid.Nil,
	})
	if !errors.Is(err, entity.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}

	// Case 2: Invalid status
	_, err = useCase.SubmitKYC(context.Background(), &entity.SubmitKYCParam{
		UserID: uuid.New(),
		Status: "unknown_status",
	})
	if !errors.Is(err, entity.ErrInvalidKycStatus) {
		t.Fatalf("expected ErrInvalidKycStatus, got %v", err)
	}

	// Case 3: Success with default status pending
	uID := uuid.New()
	doc, err := useCase.SubmitKYC(context.Background(), &entity.SubmitKYCParam{
		UserID:        uID,
		IDCardNumber:  "0123456789",
		LicenseNumber: "B2-9999",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if doc.Status != entity.KycPending {
		t.Fatalf("expected status pending, got %s", doc.Status)
	}
}

func TestReviewKYC_StateMachine(t *testing.T) {
	uID := uuid.New()
	adminID := uuid.New()

	repo := &mockRepo{
		doc: &entity.KycDocument{
			ID:     uuid.New(),
			UserID: uID,
			Status: entity.KycPending,
		},
	}
	useCase := NewKycUseCase(repo)

	// Duyệt lần 1: thành công
	reviewed, err := useCase.ReviewKYC(context.Background(), &entity.ReviewKYCParam{
		UserID:     uID,
		Approved:   true,
		ReviewerID: adminID,
		Note:       "hồ sơ hợp lệ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reviewed.Status != entity.KycApproved {
		t.Fatalf("expected approved status, got %s", reviewed.Status)
	}

	// Duyệt lần 2: phải lỗi KYC_ALREADY_REVIEWED
	_, err = useCase.ReviewKYC(context.Background(), &entity.ReviewKYCParam{
		UserID:     uID,
		Approved:   false,
		ReviewerID: adminID,
		Note:       "duyệt lại",
	})
	if !errors.Is(err, entity.ErrKycAlreadyReviewed) {
		t.Fatalf("expected ErrKycAlreadyReviewed, got %v", err)
	}
}
