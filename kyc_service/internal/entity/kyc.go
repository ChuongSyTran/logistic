package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	KycPending  = "pending"
	KycApproved = "approved"
	KycRejected = "rejected"
)

func IsValidKycStatus(s string) bool {
	return s == KycPending || s == KycApproved || s == KycRejected
}

type KycDocument struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	IDCardNumber    string     `json:"id_card_number"`
	LicenseNumber   string     `json:"license_number"`
	IDCardFrontURL  string     `json:"id_card_front_url"`
	IDCardBackURL   string     `json:"id_card_back_url"`
	LicenseFrontURL string     `json:"license_front_url"`
	LicenseBackURL  string     `json:"license_back_url"`
	Status          string     `json:"status"`
	Note            string     `json:"note"`
	ReviewerID      *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type SubmitKYCParam struct {
	UserID          uuid.UUID
	IDCardNumber    string
	LicenseNumber   string
	IDCardFrontURL  string
	IDCardBackURL   string
	LicenseFrontURL string
	LicenseBackURL  string
	Status          string
	Note            string
}

type ReviewKYCParam struct {
	UserID     uuid.UUID
	Approved   bool
	Note       string
	ReviewerID uuid.UUID
}

type KycPagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

type ListPendingKYCResult struct {
	Items      []KycDocument
	Pagination KycPagination
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

func NormalizePaging(page, pageSize int) (int, int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	offset := (page - 1) * pageSize
	return page, pageSize, offset
}

func BuildPagination(page, pageSize int, total int64) KycPagination {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 && total > 0 {
		totalPages = 1
	}
	return KycPagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}
}
