package dto

import (
	"ops-timer-backend/internal/model"
	"time"
)

type AttachmentQuery struct {
	OwnerType string `form:"owner_type" binding:"omitempty,oneof=project unit todo note schedule"`
	OwnerID   string `form:"owner_id" binding:"omitempty,uuid"`
	Q         string `form:"q" binding:"max=256"`
	Type      string `form:"type" binding:"omitempty,oneof=x509 gpg"`
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
}

type AttachmentOwner struct {
	OwnerType string `json:"owner_type" binding:"required,oneof=project unit todo note schedule"`
	OwnerID   string `json:"owner_id" binding:"required,uuid"`
}

type CertificateResponse struct {
	model.Certificate
	Status           string   `json:"status"`
	RemainingSeconds *float64 `json:"remaining_seconds"`
}

type AttachmentResponse struct {
	ID           string                `json:"id"`
	Filename     string                `json:"filename"`
	ContentType  string                `json:"content_type"`
	Size         int64                 `json:"size"`
	CreatedAt    time.Time             `json:"created_at"`
	Certificates []CertificateResponse `json:"certificates"`
}
