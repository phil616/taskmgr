package model

import "time"

// Only a digest of the random browser credential is stored in the database.
type BrowserSession struct {
	TokenHash      string    `gorm:"primaryKey;size:64"`
	UserID         string    `gorm:"size:36;not null;index"`
	CredentialHash string    `gorm:"size:64;not null"`
	ExpiresAt      time.Time `gorm:"not null;index"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}
