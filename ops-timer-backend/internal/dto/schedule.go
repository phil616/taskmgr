package dto

import "time"

// ---------- 请求 ----------

type CreateScheduleRequest struct {
	Title          string   `json:"title" binding:"required,max=256"`
	Description    string   `json:"description"`
	StartTime      string   `json:"start_time" binding:"required"`
	EndTime        string   `json:"end_time" binding:"required"`
	AllDay         bool     `json:"all_day"`
	Color          string   `json:"color" binding:"omitempty,max=20"`
	Location       string   `json:"location" binding:"omitempty,max=256"`
	Status         string   `json:"status" binding:"omitempty,oneof=planned in_progress completed cancelled"`
	RecurrenceType string   `json:"recurrence_type" binding:"omitempty,oneof=none daily weekly monthly yearly"`
	RecurrenceEnd  *string  `json:"recurrence_end"`
	Tags           []string `json:"tags"`
}

type UpdateScheduleRequest struct {
	Title          string   `json:"title" binding:"omitempty,max=256"`
	Description    *string  `json:"description"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	AllDay         *bool    `json:"all_day"`
	Color          *string  `json:"color" binding:"omitempty,max=20"`
	Location       *string  `json:"location" binding:"omitempty,max=256"`
	Status         string   `json:"status" binding:"omitempty,oneof=planned in_progress completed cancelled"`
	RecurrenceType string   `json:"recurrence_type" binding:"omitempty,oneof=none daily weekly monthly yearly"`
	RecurrenceEnd  *string  `json:"recurrence_end"`
	Tags           []string `json:"tags"`
}

type ScheduleQueryParams struct {
	StartDate string `form:"start_date"` // YYYY-MM-DD，范围起始
	EndDate   string `form:"end_date"`   // YYYY-MM-DD，范围结束
	Status    string `form:"status"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=100"`
}

type AddScheduleResourceRequest struct {
	ResourceType string `json:"resource_type" binding:"required,oneof=project todo unit"`
	ResourceID   string `json:"resource_id" binding:"required"`
	Note         string `json:"note" binding:"omitempty,max=256"`
}

// ---------- 响应 ----------

type ScheduleResourceResponse struct {
	ID            string    `json:"id"`
	ScheduleID    string    `json:"schedule_id"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	Note          string    `json:"note"`
	ResourceTitle string    `json:"resource_title,omitempty"`
	ResourceColor string    `json:"resource_color,omitempty"`
	ResourceStatus string   `json:"resource_status,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type ScheduleResponse struct {
	ID             string                      `json:"id"`
	Title          string                      `json:"title"`
	Description    string                      `json:"description"`
	StartTime      time.Time                   `json:"start_time"`
	EndTime        time.Time                   `json:"end_time"`
	AllDay         bool                        `json:"all_day"`
	Color          string                      `json:"color"`
	Location       string                      `json:"location"`
	Status         string                      `json:"status"`
	RecurrenceType string                      `json:"recurrence_type"`
	RecurrenceEnd  *time.Time                  `json:"recurrence_end"`
	Tags           []string                    `json:"tags"`
	Resources      []ScheduleResourceResponse  `json:"resources"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}
