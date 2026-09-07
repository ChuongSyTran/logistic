package errors

import "github.com/logistic/pkg/apperr"

var (
	ErrInvalidUserID     = apperr.InvalidArgument("INVALID_USER_ID", "user id không hợp lệ")
	ErrInvalidNotifID    = apperr.InvalidArgument("INVALID_NOTIFICATION_ID", "notification id không hợp lệ")
	ErrInvalidTemplateID = apperr.InvalidArgument("INVALID_TEMPLATE_ID", "template id không hợp lệ")
	ErrInvalidChannel    = apperr.InvalidArgument("INVALID_CHANNEL", "channel phải là in_app, push, email hoặc sms")
	ErrInvalidRole       = apperr.InvalidArgument("INVALID_ROLE", "recipient_role phải là driver, shipper hoặc admin")
	ErrInvalidStatus     = apperr.InvalidArgument("INVALID_STATUS", "status phải là pending, sent, failed hoặc read")
	ErrTitleRequired     = apperr.InvalidArgument("TITLE_REQUIRED", "tiêu đề thông báo là bắt buộc")
	ErrBodyRequired      = apperr.InvalidArgument("BODY_REQUIRED", "nội dung thông báo là bắt buộc")
	ErrCodeRequired      = apperr.InvalidArgument("TEMPLATE_CODE_REQUIRED", "mã template là bắt buộc")
	ErrNoRecipient       = apperr.InvalidArgument("NO_RECIPIENT", "phải chỉ định user_ids hoặc broadcast_role")

	ErrNotificationNotFound = apperr.NotFound("NOTIFICATION_NOT_FOUND", "không tìm thấy thông báo")
	ErrTemplateNotFound     = apperr.NotFound("TEMPLATE_NOT_FOUND", "không tìm thấy template")
	ErrPreferenceNotFound   = apperr.NotFound("PREFERENCE_NOT_FOUND", "không tìm thấy cài đặt thông báo")

	ErrTemplateCodeExists = apperr.AlreadyExists("TEMPLATE_CODE_EXISTS", "mã template đã tồn tại cho kênh và ngôn ngữ này")

	ErrNotificationNotOwned = apperr.PermissionDenied("NOTIFICATION_NOT_OWNED", "thông báo không thuộc về người dùng này")

	ErrDatabase = apperr.Internal("DATABASE_ERROR", "lỗi truy cập cơ sở dữ liệu")
	ErrBroker   = apperr.Unavailable("BROKER_UNAVAILABLE", "hàng đợi thông báo tạm thời không khả dụng")
)