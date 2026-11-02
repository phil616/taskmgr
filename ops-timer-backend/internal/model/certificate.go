package model

import "time"

// Attachment stores the original public certificate file. Content is exported only
// by the backup service; API responses use a separate DTO.
type Attachment struct {
	ID           string        `gorm:"primaryKey;size:36" json:"id"`
	Filename     string        `gorm:"size:255;not null" json:"filename"`
	ContentType  string        `gorm:"size:128;not null" json:"content_type"`
	Size         int64         `json:"size"`
	Content      []byte        `gorm:"type:bytea;not null" json:"content"`
	CreatedAt    time.Time     `json:"created_at"`
	Certificates []Certificate `gorm:"foreignKey:AttachmentID" json:"-"`
}

type AttachmentLink struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	AttachmentID string    `gorm:"size:36;not null;uniqueIndex:attachment_owner" json:"attachment_id"`
	OwnerType    string    `gorm:"size:20;not null;uniqueIndex:attachment_owner;index:attachment_link_owner" json:"owner_type"`
	OwnerID      string    `gorm:"size:36;not null;uniqueIndex:attachment_owner;index:attachment_link_owner" json:"owner_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type Certificate struct {
	ID              string          `gorm:"primaryKey;size:36" json:"id"`
	AttachmentID    string          `gorm:"size:36;not null;index" json:"attachment_id"`
	Type            string          `gorm:"size:10;not null" json:"type"`
	CommonName      string          `gorm:"type:text" json:"common_name"`
	Subject         string          `gorm:"type:text" json:"subject"`
	Issuer          string          `gorm:"type:text" json:"issuer"`
	SerialNumber    string          `gorm:"type:text" json:"serial_number"`
	Fingerprint     string          `gorm:"size:128;index" json:"fingerprint"`
	Algorithm       string          `gorm:"size:128" json:"algorithm"`
	DNSNames        JSONStringArray `gorm:"type:text" json:"dns_names"`
	Emails          JSONStringArray `gorm:"type:text" json:"emails"`
	UserIDs         JSONStringArray `gorm:"type:text" json:"user_ids"`
	NotBefore       time.Time       `json:"not_before"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	Revoked         bool            `json:"revoked"`
	CountdownUnitID *string         `gorm:"size:36" json:"countdown_unit_id"`
}
