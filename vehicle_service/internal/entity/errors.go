package entity

import "github.com/logistic/pkg/apperr"

var (
	ErrInvalidVehicleID  = apperr.InvalidArgument("INVALID_VEHICLE_ID", "vehicle id không hợp lệ")
	ErrInvalidDriverID   = apperr.InvalidArgument("INVALID_DRIVER_ID", "driver id không hợp lệ")
	ErrInvalidDocumentID = apperr.InvalidArgument("INVALID_DOCUMENT_ID", "document id không hợp lệ")
	ErrInvalidType       = apperr.InvalidArgument("INVALID_VEHICLE_TYPE", "vehicle_type phải là truck, van, bike, container hoặc trailer")
	ErrInvalidStatus     = apperr.InvalidArgument("INVALID_VEHICLE_STATUS", "status phải là active, maintenance hoặc inactive")
	ErrInvalidDocType    = apperr.InvalidArgument("INVALID_DOCUMENT_TYPE", "document_type phải là registration, inspection, insurance hoặc license")
	ErrInvalidReviewStat = apperr.InvalidArgument("INVALID_REVIEW_STATUS", "review_status phải là pending, approved hoặc rejected")
	ErrPlateRequired     = apperr.InvalidArgument("LICENSE_PLATE_REQUIRED", "biển số xe là bắt buộc")
	ErrFileURLRequired   = apperr.InvalidArgument("FILE_URL_REQUIRED", "file_url của giấy tờ là bắt buộc")
	ErrInvalidCoordinate = apperr.InvalidArgument("INVALID_COORDINATE", "toạ độ GPS không hợp lệ")
	ErrInvalidCapacity   = apperr.InvalidArgument("INVALID_CAPACITY", "sức chứa phải lớn hơn 0")

	ErrVehicleNotFound      = apperr.NotFound("VEHICLE_NOT_FOUND", "không tìm thấy phương tiện")
	ErrDocumentNotFound     = apperr.NotFound("DOCUMENT_NOT_FOUND", "không tìm thấy giấy tờ")
	ErrLocationNotFound     = apperr.NotFound("LOCATION_NOT_FOUND", "chưa có dữ liệu vị trí cho phương tiện này")
	ErrAvailabilityNotFound = apperr.NotFound("AVAILABILITY_NOT_FOUND", "tài xế chưa thiết lập trạng thái nhận đơn")

	ErrPlateAlreadyUsed = apperr.AlreadyExists("LICENSE_PLATE_ALREADY_USED", "biển số xe đã được đăng ký")

	ErrVehicleNotOwned      = apperr.PermissionDenied("VEHICLE_NOT_OWNED", "phương tiện không thuộc về tài xế này")
	ErrVehicleNotVerified   = apperr.FailedPrecondition("VEHICLE_NOT_VERIFIED", "phương tiện chưa được duyệt giấy tờ")
	ErrVehicleInMaintenance = apperr.FailedPrecondition("VEHICLE_IN_MAINTENANCE", "phương tiện đang bảo dưỡng, không thể nhận đơn")
	ErrDocAlreadyReviewed   = apperr.FailedPrecondition("DOCUMENT_ALREADY_REVIEWED", "giấy tờ đã được duyệt trước đó")
	ErrVehicleHasDocuments  = apperr.FailedPrecondition("VEHICLE_HAS_DOCUMENTS", "phải xoá giấy tờ trước khi xoá phương tiện")

	ErrDatabase = apperr.Internal("DATABASE_ERROR", "lỗi truy cập cơ sở dữ liệu")
	ErrGeoIndex = apperr.Unavailable("GEO_INDEX_UNAVAILABLE", "chỉ mục vị trí tạm thời không khả dụng")
)