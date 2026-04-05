package dto

import "time"

type CreateProjectRequest struct {
	Title       string  `json:"title" binding:"required,max=128"`
	Description string  `json:"description" binding:"max=10000"`
	Status      string  `json:"status" binding:"omitempty,oneof=active completed archived"`
	Color       string  `json:"color"`
	Icon        string  `json:"icon"`
	SortOrder   *int    `json:"sort_order"`
	MaxBudget   float64 `json:"max_budget"`
}

type UpdateProjectRequest struct {
	Title       string   `json:"title" binding:"omitempty,max=128"`
	Description *string  `json:"description"`
	Status      string   `json:"status" binding:"omitempty,oneof=active completed archived"`
	Color       *string  `json:"color"`
	Icon        *string  `json:"icon"`
	SortOrder   *int     `json:"sort_order"`
	MaxBudget   *float64 `json:"max_budget"`
}

type ProjectResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	SortOrder   int       `json:"sort_order"`
	MaxBudget   float64   `json:"max_budget"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	UnitStats   *ProjectUnitStats   `json:"unit_stats,omitempty"`
	BudgetStats *ProjectBudgetStats `json:"budget_stats,omitempty"`
}

type ProjectUnitStats struct {
	ActiveCount    int64 `json:"active_count"`
	ExpiringCount  int64 `json:"expiring_count"`
	CompletedCount int64 `json:"completed_count"`
	TotalCount     int64 `json:"total_count"`
}

type ProjectBudgetStats struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetAmount    float64 `json:"net_amount"`
	MaxBudget    float64 `json:"max_budget"`
	Remaining    float64 `json:"remaining"`
	UsageRate    float64 `json:"usage_rate"`
	TxCount      int64   `json:"tx_count"`
}

type ProjectQueryParams struct {
	Status   string `form:"status"`
	SortBy   string `form:"sort_by"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}
