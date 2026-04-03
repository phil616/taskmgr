package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	ScheduleStatusPlanned    = "planned"
	ScheduleStatusInProgress = "in_progress"
	ScheduleStatusCompleted  = "completed"
	ScheduleStatusCancelled  = "cancelled"

	RecurrenceNone    = "none"
	RecurrenceDaily   = "daily"
	RecurrenceWeekly  = "weekly"
	RecurrenceMonthly = "monthly"
	RecurrenceYearly  = "yearly"

	ResourceTypeProject = "project"
	ResourceTypeTodo    = "todo"
	ResourceTypeUnit    = "unit"
)

// Schedule 日程事件
type Schedule struct {
	ID             string          `gorm:"primaryKey;size:36" json:"id"`
	Title          string          `gorm:"size:256;not null" json:"title"`
	Description    string          `gorm:"type:text" json:"description"`
	StartTime      time.Time       `gorm:"not null;index" json:"start_time"`
	EndTime        time.Time       `gorm:"not null" json:"end_time"`
	AllDay         bool            `gorm:"default:false" json:"all_day"`
	Color          string          `gorm:"size:20" json:"color"`
	Location       string          `gorm:"size:256" json:"location"`
	Status         string          `gorm:"size:20;not null;default:planned" json:"status"`
	RecurrenceType string          `gorm:"size:20;not null;default:none" json:"recurrence_type"`
	RecurrenceEnd  *time.Time      `json:"recurrence_end"`
	Tags           JSONStringArray `gorm:"type:text" json:"tags"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Resources []ScheduleResource `gorm:"foreignKey:ScheduleID" json:"resources,omitempty"`
}

// ScheduleResource 日程关联资源（多态关联 project/todo/unit）
type ScheduleResource struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	ScheduleID   string    `gorm:"size:36;not null;index" json:"schedule_id"`
	ResourceType string    `gorm:"size:20;not null" json:"resource_type"`
	ResourceID   string    `gorm:"size:36;not null" json:"resource_id"`
	Note         string    `gorm:"size:256" json:"note"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}
