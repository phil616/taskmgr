package model

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:32;not null" json:"username"`
	IsAdmin      bool      `gorm:"not null;default:false" json:"is_admin"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	DisplayName  string    `gorm:"size:64;not null" json:"display_name"`
	Email        string    `gorm:"size:128" json:"email"`
	APIToken     string    `gorm:"uniqueIndex;size:64;not null" json:"-"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
