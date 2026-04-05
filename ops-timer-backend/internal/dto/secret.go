package dto

import "time"

type CreateSecretRequest struct {
	Name        string   `json:"name" binding:"required,max=128"`
	Value       string   `json:"value" binding:"required"`
	Description string   `json:"description" binding:"max=2048"`
	Tags        []string `json:"tags"`
	ProjectID   *string  `json:"project_id"`
}

type UpdateSecretRequest struct {
	Name        *string  `json:"name" binding:"omitempty,max=128"`
	Value       *string  `json:"value"`
	Description *string  `json:"description" binding:"omitempty,max=2048"`
	Tags        []string `json:"tags"`
	ProjectID   *string  `json:"project_id"`
}

type SecretResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	ProjectID   *string   `json:"project_id"`
	ProjectName string    `json:"project_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SecretBriefResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	ProjectID   *string   `json:"project_id"`
	ProjectName string    `json:"project_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SecretQueryParams struct {
	Name      string `form:"name"`
	Tag       string `form:"tag"`
	ProjectID string `form:"project_id"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
}

type SecretAuditLogResponse struct {
	ID        string    `json:"id"`
	SecretID  string    `json:"secret_id"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

type SecretAuditQueryParams struct {
	SecretID string `form:"secret_id"`
	Action   string `form:"action"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}
