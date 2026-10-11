package model

import "time"

const (
	TodoStatusPending    = "pending"
	TodoStatusInProgress = "in_progress"
	TodoStatusDone       = "done"
	TodoStatusCancelled  = "cancelled"
)

type Todo struct {
	ID          string     `gorm:"primaryKey;size:36" json:"id"`
	GroupID     *string    `gorm:"size:36;index" json:"group_id"`
	Title       string     `gorm:"size:256;not null" json:"title"`
	Description string     `gorm:"type:text" json:"description"`
	Status      string     `gorm:"size:20;not null;default:pending" json:"status"`
	Priority    string     `gorm:"size:10;not null;default:normal" json:"priority"`
	Importance  float64    `gorm:"not null;default:0;check:todo_importance_range,importance >= 0 AND importance <= 1" json:"importance"`
	Difficulty  float64    `gorm:"not null;default:0;check:todo_difficulty_range,difficulty >= 0 AND difficulty <= 1" json:"difficulty"`
	DueDate     *time.Time `gorm:"type:date" json:"due_date"`
	SortOrder   int        `gorm:"default:0" json:"sort_order"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Group *TodoGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

type TodoGroup struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Color     string    `gorm:"size:20" json:"color"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Todos     []Todo `gorm:"foreignKey:GroupID" json:"todos,omitempty"`
	TodoCount int64  `gorm:"-" json:"todo_count"`
}
