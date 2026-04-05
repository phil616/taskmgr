package model

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID          string         `gorm:"primaryKey;size:36" json:"id"`
	Title       string         `gorm:"size:128;not null" json:"title"`
	Description string         `gorm:"size:10000" json:"description"`
	Status      string         `gorm:"size:20;not null;default:active" json:"status"`
	Color       string         `gorm:"size:20" json:"color"`
	Icon        string         `gorm:"size:64" json:"icon"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	MaxBudget   float64        `gorm:"default:0" json:"max_budget"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Units        []Unit        `gorm:"foreignKey:ProjectID" json:"units,omitempty"`
	Transactions []Transaction `gorm:"foreignKey:ProjectID" json:"transactions,omitempty"`
}
