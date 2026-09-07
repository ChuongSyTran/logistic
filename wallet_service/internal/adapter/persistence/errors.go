package persistence

import (
	"fmt"

	"wallet_service/ent"
	"wallet_service/internal/entity"
)

// translate là ranh giới duy nhất mà từ vựng lỗi của ent được phép tồn tại.
// Mọi thứ đi ra khỏi package này chỉ còn lỗi nghiệp vụ, nên tầng app không cần
// import wallet_service/ent để hiểu chuyện gì đã xảy ra.
func translate(err error, notFound error) error {
	switch {
	case err == nil:
		return nil
	case ent.IsNotFound(err):
		return fmt.Errorf("%w: %v", notFound, err)
	case ent.IsConstraintError(err):
		return fmt.Errorf("%w: %v", entity.ErrMessageAlreadyProcessed, err)
	default:
		return err
	}
}
