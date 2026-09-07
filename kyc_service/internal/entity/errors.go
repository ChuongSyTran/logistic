package entity

import "github.com/logistic/pkg/apperr"

var (
	ErrInvalidUserID    = apperr.InvalidArgument("INVALID_USER_ID", "user id không hợp lệ")
	ErrInvalidKycStatus = apperr.InvalidArgument("INVALID_KYC_STATUS", "kyc_status phải là pending, approved hoặc rejected")

	ErrKycNotFound        = apperr.NotFound("KYC_NOT_FOUND", "không tìm thấy hồ sơ KYC")
	ErrLicenseAlreadyUsed = apperr.AlreadyExists("LICENSE_ALREADY_USED", "số bằng lái đã được dùng bởi tài xế khác")
	ErrIDCardAlreadyUsed  = apperr.AlreadyExists("ID_CARD_ALREADY_USED", "số CCCD đã được dùng bởi tài xế khác")

	ErrKycAlreadyReviewed = apperr.FailedPrecondition("KYC_ALREADY_REVIEWED", "hồ sơ KYC đã được duyệt trước đó")
	ErrKycNotApproved     = apperr.FailedPrecondition("KYC_NOT_APPROVED", "hồ sơ KYC chưa được duyệt")

	ErrDatabase = apperr.Internal("DATABASE_ERROR", "lỗi truy cập cơ sở dữ liệu")
)
