package model

import (
	"time"

	"gorm.io/gorm"
)

type Secret struct {
	ID          string          `gorm:"primaryKey;size:36" json:"id"`
	Name        string          `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Value       string          `gorm:"type:text;not null" json:"value"`
	Description string          `gorm:"size:2048" json:"description"`
	Tags        JSONStringArray `gorm:"type:text" json:"tags"`
	ProjectID   *string         `gorm:"size:36;index" json:"project_id"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

const (
	SecretActionCreated   = "created"
	SecretActionRead      = "read"
	SecretActionUpdated   = "updated"
	SecretActionDeleted   = "deleted"
	SecretActionValueRead = "value_read"
	SecretActionListed    = "listed"
)

type SecretAuditLog struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SecretID  string    `gorm:"size:36;index;not null" json:"secret_id"`
	Action    string    `gorm:"size:20;not null" json:"action"`
	UserID    string    `gorm:"size:36;not null" json:"user_id"`
	Username  string    `gorm:"size:32;not null" json:"username"`
	IPAddress string    `gorm:"size:45" json:"ip_address"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
