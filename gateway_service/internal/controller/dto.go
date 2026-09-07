package controller

import (
	"time"

	pbnotification "github.com/logistic/api/logistic/notification_service/v1"
	pbuser "github.com/logistic/api/logistic/user_service/v1"
	pbvehicle "github.com/logistic/api/logistic/vehicle_service/v1"
	"github.com/logistic/pkg/uuidx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func rfc3339(ts *timestamppb.Timestamp) *string {
	if ts == nil || !ts.IsValid() {
		return nil
	}
	s := ts.AsTime().UTC().Format(time.RFC3339)
	return &s
}

type UserDTO struct {
	ID        string  `json:"id"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email"`
	FullName  string  `json:"full_name"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	AvatarURL string  `json:"avatar_url"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

func toUserDTO(u *pbuser.User) *UserDTO {
	if u == nil {
		return nil
	}
	return &UserDTO{
		ID:        uuidx.String(u.Id),
		Phone:     u.Phone,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		Status:    u.Status,
		AvatarURL: u.AvatarUrl,
		CreatedAt: rfc3339(u.CreatedAt),
		UpdatedAt: rfc3339(u.UpdatedAt),
	}
}

func toUserDTOs(list []*pbuser.User) []*UserDTO {
	out := make([]*UserDTO, 0, len(list))
	for _, u := range list {
		out = append(out, toUserDTO(u))
	}
	return out
}

type DriverProfileDTO struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	LicenseNumber string  `json:"license_number"`
	IDCard        string  `json:"id_card"`
	Rating        float64 `json:"rating"`
	TotalTrips    int32   `json:"total_trips"`
	KYCStatus     string  `json:"kyc_status"`
	KYCNote       string  `json:"kyc_note"`
	CreatedAt     *string `json:"created_at"`
	UpdatedAt     *string `json:"updated_at"`
}

func toDriverProfileDTO(p *pbuser.DriverProfile) *DriverProfileDTO {
	if p == nil {
		return nil
	}
	return &DriverProfileDTO{
		ID:            uuidx.String(p.Id),
		UserID:        uuidx.String(p.UserId),
		LicenseNumber: p.LicenseNumber,
		IDCard:        p.IdCard,
		Rating:        p.Rating,
		TotalTrips:    p.TotalTrips,
		KYCStatus:     p.KycStatus,
		KYCNote:       p.KycNote,
		CreatedAt:     rfc3339(p.CreatedAt),
		UpdatedAt:     rfc3339(p.UpdatedAt),
	}
}

func toDriverProfileDTOs(list []*pbuser.DriverProfile) []*DriverProfileDTO {
	out := make([]*DriverProfileDTO, 0, len(list))
	for _, p := range list {
		out = append(out, toDriverProfileDTO(p))
	}
	return out
}

type ShipperProfileDTO struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	CompanyName     string  `json:"company_name"`
	TaxCode         string  `json:"tax_code"`
	BusinessAddress string  `json:"business_address"`
	TotalOrders     int32   `json:"total_orders"`
	CreatedAt       *string `json:"created_at"`
	UpdatedAt       *string `json:"updated_at"`
}

func toShipperProfileDTO(p *pbuser.ShipperProfile) *ShipperProfileDTO {
	if p == nil {
		return nil
	}
	return &ShipperProfileDTO{
		ID:              uuidx.String(p.Id),
		UserID:          uuidx.String(p.UserId),
		CompanyName:     p.CompanyName,
		TaxCode:         p.TaxCode,
		BusinessAddress: p.BusinessAddress,
		TotalOrders:     p.TotalOrders,
		CreatedAt:       rfc3339(p.CreatedAt),
		UpdatedAt:       rfc3339(p.UpdatedAt),
	}
}

type AddressDTO struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	Label        string  `json:"label"`
	ContactName  string  `json:"contact_name"`
	ContactPhone string  `json:"contact_phone"`
	Line1        string  `json:"line1"`
	Ward         string  `json:"ward"`
	District     string  `json:"district"`
	City         string  `json:"city"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	AddressType  string  `json:"address_type"`
	IsDefault    bool    `json:"is_default"`
	CreatedAt    *string `json:"created_at"`
	UpdatedAt    *string `json:"updated_at"`
}

func toAddressDTO(a *pbuser.Address) *AddressDTO {
	if a == nil {
		return nil
	}
	return &AddressDTO{
		ID:           uuidx.String(a.Id),
		UserID:       uuidx.String(a.UserId),
		Label:        a.Label,
		ContactName:  a.ContactName,
		ContactPhone: a.ContactPhone,
		Line1:        a.Line1,
		Ward:         a.Ward,
		District:     a.District,
		City:         a.City,
		Latitude:     a.Latitude,
		Longitude:    a.Longitude,
		AddressType:  a.AddressType,
		IsDefault:    a.IsDefault,
		CreatedAt:    rfc3339(a.CreatedAt),
		UpdatedAt:    rfc3339(a.UpdatedAt),
	}
}

func toAddressDTOs(list []*pbuser.Address) []*AddressDTO {
	out := make([]*AddressDTO, 0, len(list))
	for _, a := range list {
		out = append(out, toAddressDTO(a))
	}
	return out
}

type UserDeviceDTO struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	DeviceToken string  `json:"device_token"`
	Platform    string  `json:"platform"`
	DeviceName  string  `json:"device_name"`
	IsActive    bool    `json:"is_active"`
	LastSeenAt  *string `json:"last_seen_at"`
	CreatedAt   *string `json:"created_at"`
}

func toUserDeviceDTO(d *pbuser.UserDevice) *UserDeviceDTO {
	if d == nil {
		return nil
	}
	return &UserDeviceDTO{
		ID:          uuidx.String(d.Id),
		UserID:      uuidx.String(d.UserId),
		DeviceToken: d.DeviceToken,
		Platform:    d.Platform,
		DeviceName:  d.DeviceName,
		IsActive:    d.IsActive,
		LastSeenAt:  rfc3339(d.LastSeenAt),
		CreatedAt:   rfc3339(d.CreatedAt),
	}
}

func toUserDeviceDTOs(list []*pbuser.UserDevice) []*UserDeviceDTO {
	out := make([]*UserDeviceDTO, 0, len(list))
	for _, d := range list {
		out = append(out, toUserDeviceDTO(d))
	}
	return out
}

type PaginationDTO struct {
	Page       int32 `json:"page"`
	PageSize   int32 `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int32 `json:"total_pages"`
}

func toUserPaginationDTO(p *pbuser.Pagination) *PaginationDTO {
	if p == nil {
		return nil
	}
	return &PaginationDTO{Page: p.Page, PageSize: p.PageSize, TotalItems: p.TotalItems, TotalPages: p.TotalPages}
}

func toVehiclePaginationDTO(p *pbvehicle.Pagination) *PaginationDTO {
	if p == nil {
		return nil
	}
	return &PaginationDTO{Page: p.Page, PageSize: p.PageSize, TotalItems: p.TotalItems, TotalPages: p.TotalPages}
}

func toNotifPaginationDTO(p *pbnotification.Pagination) *PaginationDTO {
	if p == nil {
		return nil
	}
	return &PaginationDTO{Page: p.Page, PageSize: p.PageSize, TotalItems: p.TotalItems, TotalPages: p.TotalPages}
}

type VehicleDTO struct {
	ID                 string  `json:"id"`
	DriverID           string  `json:"driver_id"`
	LicensePlate       string  `json:"license_plate"`
	Brand              string  `json:"brand"`
	Model              string  `json:"model"`
	ManufactureYear    int32   `json:"manufacture_year"`
	VehicleType        string  `json:"vehicle_type"`
	CapacityWeightKg   float64 `json:"capacity_weight_kg"`
	CapacityVolumeCbm  float64 `json:"capacity_volume_cbm"`
	Status             string  `json:"status"`
	VerificationStatus string  `json:"verification_status"`
	CreatedAt          *string `json:"created_at"`
	UpdatedAt          *string `json:"updated_at"`
}

func toVehicleDTO(v *pbvehicle.Vehicle) *VehicleDTO {
	if v == nil {
		return nil
	}
	return &VehicleDTO{
		ID:                 uuidx.String(v.Id),
		DriverID:           uuidx.String(v.DriverId),
		LicensePlate:       v.LicensePlate,
		Brand:              v.Brand,
		Model:              v.Model,
		ManufactureYear:    v.ManufactureYear,
		VehicleType:        v.VehicleType,
		CapacityWeightKg:   v.CapacityWeightKg,
		CapacityVolumeCbm:  v.CapacityVolumeCbm,
		Status:             v.Status,
		VerificationStatus: v.VerificationStatus,
		CreatedAt:          rfc3339(v.CreatedAt),
		UpdatedAt:          rfc3339(v.UpdatedAt),
	}
}

func toVehicleDTOs(list []*pbvehicle.Vehicle) []*VehicleDTO {
	out := make([]*VehicleDTO, 0, len(list))
	for _, v := range list {
		out = append(out, toVehicleDTO(v))
	}
	return out
}

type VehicleDocumentDTO struct {
	ID             string  `json:"id"`
	VehicleID      string  `json:"vehicle_id"`
	DocumentType   string  `json:"document_type"`
	DocumentNumber string  `json:"document_number"`
	FileURL        string  `json:"file_url"`
	IssuedAt       *string `json:"issued_at"`
	ExpiresAt      *string `json:"expires_at"`
	ReviewStatus   string  `json:"review_status"`
	ReviewNote     string  `json:"review_note"`
	CreatedAt      *string `json:"created_at"`
}

func toVehicleDocumentDTO(d *pbvehicle.VehicleDocument) *VehicleDocumentDTO {
	if d == nil {
		return nil
	}
	return &VehicleDocumentDTO{
		ID:             uuidx.String(d.Id),
		VehicleID:      uuidx.String(d.VehicleId),
		DocumentType:   d.DocumentType,
		DocumentNumber: d.DocumentNumber,
		FileURL:        d.FileUrl,
		IssuedAt:       rfc3339(d.IssuedAt),
		ExpiresAt:      rfc3339(d.ExpiresAt),
		ReviewStatus:   d.ReviewStatus,
		ReviewNote:     d.ReviewNote,
		CreatedAt:      rfc3339(d.CreatedAt),
	}
}

func toVehicleDocumentDTOs(list []*pbvehicle.VehicleDocument) []*VehicleDocumentDTO {
	out := make([]*VehicleDocumentDTO, 0, len(list))
	for _, d := range list {
		out = append(out, toVehicleDocumentDTO(d))
	}
	return out
}

type VehicleLocationDTO struct {
	VehicleID  string  `json:"vehicle_id"`
	DriverID   string  `json:"driver_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Heading    float64 `json:"heading"`
	SpeedKph   float64 `json:"speed_kph"`
	ZoneID     string  `json:"zone_id"`
	RecordedAt *string `json:"recorded_at"`
}

func toVehicleLocationDTO(l *pbvehicle.VehicleLocation) *VehicleLocationDTO {
	if l == nil {
		return nil
	}
	return &VehicleLocationDTO{
		VehicleID:  uuidx.String(l.VehicleId),
		DriverID:   uuidx.String(l.DriverId),
		Latitude:   l.Latitude,
		Longitude:  l.Longitude,
		Heading:    l.Heading,
		SpeedKph:   l.SpeedKph,
		ZoneID:     l.ZoneId,
		RecordedAt: rfc3339(l.RecordedAt),
	}
}

type DriverAvailabilityDTO struct {
	ID                 string  `json:"id"`
	DriverID           string  `json:"driver_id"`
	VehicleID          string  `json:"vehicle_id"`
	IsOnline           bool    `json:"is_online"`
	AvailableWeightKg  float64 `json:"available_weight_kg"`
	AvailableVolumeCbm float64 `json:"available_volume_cbm"`
	CurrentLat         float64 `json:"current_lat"`
	CurrentLng         float64 `json:"current_lng"`
	ZoneID             string  `json:"zone_id"`
	UpdatedAt          *string `json:"updated_at"`
}

func toDriverAvailabilityDTO(a *pbvehicle.DriverAvailability) *DriverAvailabilityDTO {
	if a == nil {
		return nil
	}
	return &DriverAvailabilityDTO{
		ID:                 uuidx.String(a.Id),
		DriverID:           uuidx.String(a.DriverId),
		VehicleID:          uuidx.String(a.VehicleId),
		IsOnline:           a.IsOnline,
		AvailableWeightKg:  a.AvailableWeightKg,
		AvailableVolumeCbm: a.AvailableVolumeCbm,
		CurrentLat:         a.CurrentLat,
		CurrentLng:         a.CurrentLng,
		ZoneID:             a.ZoneId,
		UpdatedAt:          rfc3339(a.UpdatedAt),
	}
}

type NotificationDTO struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	RecipientRole string  `json:"recipient_role"`
	Type          string  `json:"type"`
	Channel       string  `json:"channel"`
	Title         string  `json:"title"`
	Body          string  `json:"body"`
	Data          string  `json:"data"`
	RefType       string  `json:"ref_type"`
	RefID         string  `json:"ref_id"`
	IsRead        bool    `json:"is_read"`
	Status        string  `json:"status"`
	ReadAt        *string `json:"read_at"`
	CreatedAt     *string `json:"created_at"`
}

func toNotificationDTO(n *pbnotification.Notification) *NotificationDTO {
	if n == nil {
		return nil
	}
	return &NotificationDTO{
		ID:            uuidx.String(n.Id),
		UserID:        uuidx.String(n.UserId),
		RecipientRole: n.RecipientRole,
		Type:          n.Type,
		Channel:       n.Channel,
		Title:         n.Title,
		Body:          n.Body,
		Data:          n.Data,
		RefType:       n.RefType,
		RefID:         uuidx.String(n.RefId),
		IsRead:        n.IsRead,
		Status:        n.Status,
		ReadAt:        rfc3339(n.ReadAt),
		CreatedAt:     rfc3339(n.CreatedAt),
	}
}

func toNotificationDTOs(list []*pbnotification.Notification) []*NotificationDTO {
	out := make([]*NotificationDTO, 0, len(list))
	for _, n := range list {
		out = append(out, toNotificationDTO(n))
	}
	return out
}

type NotificationTemplateDTO struct {
	ID            string  `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Channel       string  `json:"channel"`
	Locale        string  `json:"locale"`
	TitleTemplate string  `json:"title_template"`
	BodyTemplate  string  `json:"body_template"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     *string `json:"created_at"`
	UpdatedAt     *string `json:"updated_at"`
}

func toNotificationTemplateDTO(t *pbnotification.NotificationTemplate) *NotificationTemplateDTO {
	if t == nil {
		return nil
	}
	return &NotificationTemplateDTO{
		ID:            uuidx.String(t.Id),
		Code:          t.Code,
		Name:          t.Name,
		Channel:       t.Channel,
		Locale:        t.Locale,
		TitleTemplate: t.TitleTemplate,
		BodyTemplate:  t.BodyTemplate,
		IsActive:      t.IsActive,
		CreatedAt:     rfc3339(t.CreatedAt),
		UpdatedAt:     rfc3339(t.UpdatedAt),
	}
}

func toNotificationTemplateDTOs(list []*pbnotification.NotificationTemplate) []*NotificationTemplateDTO {
	out := make([]*NotificationTemplateDTO, 0, len(list))
	for _, t := range list {
		out = append(out, toNotificationTemplateDTO(t))
	}
	return out
}

type NotificationPreferenceDTO struct {
	ID                 string  `json:"id"`
	UserID             string  `json:"user_id"`
	InAppEnabled       bool    `json:"in_app_enabled"`
	PushEnabled        bool    `json:"push_enabled"`
	EmailEnabled       bool    `json:"email_enabled"`
	SMSEnabled         bool    `json:"sms_enabled"`
	MatchEventsEnabled bool    `json:"match_events_enabled"`
	PromotionEnabled   bool    `json:"promotion_enabled"`
	QuietHoursStart    string  `json:"quiet_hours_start"`
	QuietHoursEnd      string  `json:"quiet_hours_end"`
	UpdatedAt          *string `json:"updated_at"`
}

func toNotificationPreferenceDTO(p *pbnotification.NotificationPreference) *NotificationPreferenceDTO {
	if p == nil {
		return nil
	}
	return &NotificationPreferenceDTO{
		ID:                 uuidx.String(p.Id),
		UserID:             uuidx.String(p.UserId),
		InAppEnabled:       p.InAppEnabled,
		PushEnabled:        p.PushEnabled,
		EmailEnabled:       p.EmailEnabled,
		SMSEnabled:         p.SmsEnabled,
		MatchEventsEnabled: p.MatchEventsEnabled,
		PromotionEnabled:   p.PromotionEnabled,
		QuietHoursStart:    p.QuietHoursStart,
		QuietHoursEnd:      p.QuietHoursEnd,
		UpdatedAt:          rfc3339(p.UpdatedAt),
	}
}

func uuidString(b []byte) string { return uuidx.String(b) }

type NearbyVehicleDTO struct {
	VehicleID          string  `json:"vehicle_id"`
	DriverID           string  `json:"driver_id"`
	LicensePlate       string  `json:"license_plate"`
	VehicleType        string  `json:"vehicle_type"`
	DistanceKm         float64 `json:"distance_km"`
	AvailableWeightKg  float64 `json:"available_weight_kg"`
	AvailableVolumeCbm float64 `json:"available_volume_cbm"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
}

func toNearbyVehicleDTOs(list []*pbvehicle.NearbyVehicle) []*NearbyVehicleDTO {
	out := make([]*NearbyVehicleDTO, 0, len(list))
	for _, v := range list {
		if v == nil {
			continue
		}
		out = append(out, &NearbyVehicleDTO{
			VehicleID:          uuidx.String(v.VehicleId),
			DriverID:           uuidx.String(v.DriverId),
			LicensePlate:       v.LicensePlate,
			VehicleType:        v.VehicleType,
			DistanceKm:         v.DistanceKm,
			AvailableWeightKg:  v.AvailableWeightKg,
			AvailableVolumeCbm: v.AvailableVolumeCbm,
			Latitude:           v.Latitude,
			Longitude:          v.Longitude,
		})
	}
	return out
}