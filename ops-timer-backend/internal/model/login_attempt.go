package model

import "time"

type LoginAttempt struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	Username string    `gorm:"uniqueIndex;size:32;not null"`
	Count    int       `gorm:"not null;default:0"`
	LockedAt time.Time `gorm:"index"`
}
