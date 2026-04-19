package model

import (
	"time"

	"gorm.io/gorm"
)

type Note struct {
	ID        string          `gorm:"primaryKey;size:36" json:"id"`
	GroupID   *string         `gorm:"size:36;index" json:"group_id"`
	Title     string          `gorm:"size:256;not null" json:"title"`
	Content   string          `gorm:"type:text;not null" json:"content"`
	Tags      JSONStringArray `gorm:"type:text" json:"tags"`
	DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"`
	CreatedAt time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	Group *NoteGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

type NoteGroup struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Color     string    `gorm:"size:20" json:"color"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Notes     []Note `gorm:"foreignKey:GroupID" json:"notes,omitempty"`
	NoteCount int64  `gorm:"-" json:"note_count"`
}
