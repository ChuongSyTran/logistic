package persistence

import (
	"context"
	"errors"
	"strings"

	"kyc_service/ent"
	"kyc_service/internal/entity"

	"github.com/logistic/pkg/apperr"
)

func wrapError(err error, notFound *apperr.Error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return apperr.New(apperr.KindTimeout, "REQUEST_CANCELLED", "yêu cầu đã bị huỷ").WithCause(err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apperr.New(apperr.KindTimeout, "REQUEST_TIMEOUT", "yêu cầu quá thời gian chờ").WithCause(err)
	}

	if ent.IsNotFound(err) {
		if notFound == nil {
			notFound = entity.ErrKycNotFound
		}
		return notFound.WithCause(err)
	}

	if ent.IsConstraintError(err) {
		return mapConstraint(err)
	}

	if ent.IsValidationError(err) {
		return apperr.InvalidArgument("VALIDATION_FAILED", "dữ liệu không hợp lệ").WithCause(err)
	}

	if ent.IsNotSingular(err) {
		return apperr.Conflict("NOT_SINGULAR", "truy vấn trả về nhiều hơn một bản ghi").WithCause(err)
	}

	return entity.ErrDatabase.WithCause(err)
}

func mapConstraint(err error) error {
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "license_number"):
		return entity.ErrLicenseAlreadyUsed.WithCause(err)
	case strings.Contains(msg, "id_card"):
		return entity.ErrIDCardAlreadyUsed.WithCause(err)
	default:
		return apperr.Conflict("CONSTRAINT_VIOLATION", "dữ liệu vi phạm ràng buộc của hệ thống").WithCause(err)
	}
}
