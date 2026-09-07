package persistence

import (
	"context"
	"time"

	"kyc_service/ent"
	"kyc_service/ent/kyc"
	"kyc_service/internal/entity"

	"github.com/google/uuid"
)

type KycRepo struct {
	client *ent.Client
}

func NewKycRepo(client *ent.Client) *KycRepo {
	return &KycRepo{client: client}
}

func (r *KycRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.KycDocument, error) {
	record, err := r.client.Kyc.Query().
		Where(kyc.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, wrapError(err, entity.ErrKycNotFound)
	}
	return toEntity(record), nil
}

func (r *KycRepo) Upsert(ctx context.Context, param *entity.SubmitKYCParam) (*entity.KycDocument, error) {
	existing, err := r.client.Kyc.Query().
		Where(kyc.UserIDEQ(param.UserID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, wrapError(err, nil)
	}

	if ent.IsNotFound(err) {
		create := r.client.Kyc.Create().
			SetUserID(param.UserID).
			SetStatus(kyc.Status(param.Status)).
			SetNote(param.Note)

		if param.IDCardNumber != "" {
			create.SetIDCardNumber(param.IDCardNumber)
		}
		if param.LicenseNumber != "" {
			create.SetLicenseNumber(param.LicenseNumber)
		}
		if param.IDCardFrontURL != "" {
			create.SetIDCardFrontURL(param.IDCardFrontURL)
		}
		if param.IDCardBackURL != "" {
			create.SetIDCardBackURL(param.IDCardBackURL)
		}
		if param.LicenseFrontURL != "" {
			create.SetLicenseFrontURL(param.LicenseFrontURL)
		}
		if param.LicenseBackURL != "" {
			create.SetLicenseBackURL(param.LicenseBackURL)
		}

		created, cErr := create.Save(ctx)
		if cErr != nil {
			return nil, wrapError(cErr, nil)
		}
		return toEntity(created), nil
	}

	update := existing.Update().
		SetStatus(kyc.Status(param.Status)).
		SetNote(param.Note)

	if param.IDCardNumber != "" {
		update.SetIDCardNumber(param.IDCardNumber)
	}
	if param.LicenseNumber != "" {
		update.SetLicenseNumber(param.LicenseNumber)
	}
	if param.IDCardFrontURL != "" {
		update.SetIDCardFrontURL(param.IDCardFrontURL)
	}
	if param.IDCardBackURL != "" {
		update.SetIDCardBackURL(param.IDCardBackURL)
	}
	if param.LicenseFrontURL != "" {
		update.SetLicenseFrontURL(param.LicenseFrontURL)
	}
	if param.LicenseBackURL != "" {
		update.SetLicenseBackURL(param.LicenseBackURL)
	}

	updated, uErr := update.Save(ctx)
	if uErr != nil {
		return nil, wrapError(uErr, nil)
	}
	return toEntity(updated), nil
}

func (r *KycRepo) Review(ctx context.Context, param *entity.ReviewKYCParam) (*entity.KycDocument, error) {
	status := kyc.StatusRejected
	if param.Approved {
		status = kyc.StatusApproved
	}

	now := time.Now()
	n, err := r.client.Kyc.Update().
		Where(kyc.UserIDEQ(param.UserID)).
		SetStatus(status).
		SetNote(param.Note).
		SetReviewedBy(param.ReviewerID).
		SetReviewedAt(now).
		Save(ctx)
	if err != nil {
		return nil, wrapError(err, nil)
	}
	if n == 0 {
		return nil, entity.ErrKycNotFound
	}

	return r.GetByUserID(ctx, param.UserID)
}

func (r *KycRepo) ListPending(ctx context.Context, page, pageSize int) ([]entity.KycDocument, int64, error) {
	query := r.client.Kyc.Query().Where(kyc.StatusEQ(kyc.StatusPending))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, wrapError(err, nil)
	}

	offset := (page - 1) * pageSize
	records, err := query.
		Order(ent.Desc(kyc.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, wrapError(err, nil)
	}

	res := make([]entity.KycDocument, len(records))
	for i, rec := range records {
		res[i] = *toEntity(rec)
	}
	return res, int64(total), nil
}

func (r *KycRepo) CountPending(ctx context.Context) (int64, error) {
	count, err := r.client.Kyc.Query().Where(kyc.StatusEQ(kyc.StatusPending)).Count(ctx)
	if err != nil {
		return 0, wrapError(err, nil)
	}
	return int64(count), nil
}

func toEntity(k *ent.Kyc) *entity.KycDocument {
	if k == nil {
		return nil
	}
	doc := &entity.KycDocument{
		ID:              k.ID,
		UserID:          k.UserID,
		IDCardFrontURL:  k.IDCardFrontURL,
		IDCardBackURL:   k.IDCardBackURL,
		LicenseFrontURL: k.LicenseFrontURL,
		LicenseBackURL:  k.LicenseBackURL,
		Status:          string(k.Status),
		Note:            k.Note,
		CreatedAt:       k.CreatedAt,
		UpdatedAt:       k.UpdatedAt,
	}
	if k.IDCardNumber != nil {
		doc.IDCardNumber = *k.IDCardNumber
	}
	if k.LicenseNumber != nil {
		doc.LicenseNumber = *k.LicenseNumber
	}
	if k.ReviewedBy != nil {
		doc.ReviewerID = k.ReviewedBy
	}
	if k.ReviewedAt != nil {
		doc.ReviewedAt = k.ReviewedAt
	}
	return doc
}
