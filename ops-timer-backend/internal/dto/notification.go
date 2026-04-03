package dto

import "time"

type NotificationResponse struct {
	ID          string     `json:"id"`
	UnitID      string     `json:"unit_id"`
	Level       string     `json:"level"`
	Message     string     `json:"message"`
	IsRead      bool       `json:"is_read"`
	TriggeredAt time.Time  `json:"triggered_at"`
	ReadAt      *time.Time `json:"read_at"`
	UnitTitle   string     `json:"unit_title,omitempty"`
}

type NotificationQueryParams struct {
	IsRead   *bool  `form:"is_read"`
	Level    string `form:"level"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}
