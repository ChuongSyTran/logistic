package persistence

import (
	"fmt"
	"matching_service/ent"
	"matching_service/internal/entity"
)

func wrapError(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *ent.NotFoundError:
		return fmt.Errorf("%w: %v", entity.ErrRecordNotFound, err)
	case *ent.ConstraintError:
		return fmt.Errorf("%w: %v", entity.ErrInlavidInput, err)
	case *ent.ValidationError:
		return fmt.Errorf("%w: %v", entity.ErrValidationFailed, err)
	default:
		return fmt.Errorf("%w: db execution failed - %v", entity.ErrInternalServer, err)
	}
}
