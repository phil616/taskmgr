package dto

import "time"

// ==================== Wallet 钱包 ====================

type CreateWalletRequest struct {
	Name        string  `json:"name" binding:"required,max=64"`
	Type        string  `json:"type" binding:"omitempty,oneof=bank cash credit alipay wechat other"`
	Balance     float64 `json:"balance"`
	Currency    string  `json:"currency" binding:"omitempty,max=10"`
	Color       string  `json:"color" binding:"omitempty,max=20"`
	Icon        string  `json:"icon" binding:"omitempty,max=64"`
	Description string  `json:"description" binding:"omitempty,max=512"`
	IsDefault   bool    `json:"is_default"`
	SortOrder   *int    `json:"sort_order"`
}

type UpdateWalletRequest struct {
	Name        string  `json:"name" binding:"omitempty,max=64"`
	Type        string  `json:"type" binding:"omitempty,oneof=bank cash credit alipay wechat other"`
	Color       *string `json:"color" binding:"omitempty,max=20"`
	Icon        *string `json:"icon" binding:"omitempty,max=64"`
	Description *string `json:"description" binding:"omitempty,max=512"`
	IsDefault   *bool   `json:"is_default"`
	SortOrder   *int    `json:"sort_order"`
}

type WalletResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Balance        float64   `json:"balance"`
	Currency       string    `json:"currency"`
	Color          string    `json:"color"`
	Icon           string    `json:"icon"`
	Description    string    `json:"description"`
	IsDefault      bool      `json:"is_default"`
	SortOrder      int       `json:"sort_order"`
	TotalIncome    float64   `json:"total_income"`
	TotalExpense   float64   `json:"total_expense"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ==================== BudgetCategory 分类 ====================

type CreateCategoryRequest struct {
	Name      string `json:"name" binding:"required,max=64"`
	Type      string `json:"type" binding:"required,oneof=income expense both"`
	Color     string `json:"color" binding:"omitempty,max=20"`
	Icon      string `json:"icon" binding:"omitempty,max=64"`
	SortOrder *int   `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name      string  `json:"name" binding:"omitempty,max=64"`
	Type      string  `json:"type" binding:"omitempty,oneof=income expense both"`
	Color     *string `json:"color" binding:"omitempty,max=20"`
	Icon      *string `json:"icon" binding:"omitempty,max=64"`
	SortOrder *int    `json:"sort_order"`
}

type CategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Color     string    `json:"color"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	IsSystem  bool      `json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ==================== Transaction 收支记录 ====================

type CreateTransactionRequest struct {
	WalletID      string   `json:"wallet_id" binding:"required"`
	CategoryID    *string  `json:"category_id"`
	ProjectID     *string  `json:"project_id"`
	Type          string   `json:"type" binding:"required,oneof=income expense transfer"`
	Amount        float64  `json:"amount" binding:"required,gt=0"`
	Note          string   `json:"note" binding:"omitempty,max=512"`
	Tags          []string `json:"tags"`
	TransactionAt string   `json:"transaction_at" binding:"required"`
	ToWalletID    *string  `json:"to_wallet_id"`
}

type UpdateTransactionRequest struct {
	CategoryID    *string  `json:"category_id"`
	ProjectID     *string  `json:"project_id"`
	Amount        float64  `json:"amount" binding:"omitempty,gt=0"`
	Note          *string  `json:"note" binding:"omitempty,max=512"`
	Tags          []string `json:"tags"`
	TransactionAt string   `json:"transaction_at"`
}

type TransactionQueryParams struct {
	WalletID   string  `form:"wallet_id"`
	CategoryID string  `form:"category_id"`
	ProjectID  string  `form:"project_id"`
	Type       string  `form:"type"`
	StartDate  string  `form:"start_date"`
	EndDate    string  `form:"end_date"`
	MinAmount  float64 `form:"min_amount"`
	MaxAmount  float64 `form:"max_amount"`
	Keyword    string  `form:"keyword"`
	Page       int     `form:"page,default=1"`
	PageSize   int     `form:"page_size,default=20"`
}

type TransactionResponse struct {
	ID              string            `json:"id"`
	WalletID        string            `json:"wallet_id"`
	WalletName      string            `json:"wallet_name"`
	CategoryID      *string           `json:"category_id"`
	CategoryName    string            `json:"category_name"`
	CategoryIcon    string            `json:"category_icon"`
	CategoryColor   string            `json:"category_color"`
	ProjectID       *string           `json:"project_id"`
	ProjectName     string            `json:"project_name"`
	Type            string            `json:"type"`
	Amount          float64           `json:"amount"`
	Note            string            `json:"note"`
	Tags            []string          `json:"tags"`
	TransactionAt   time.Time         `json:"transaction_at"`
	ToWalletID      *string           `json:"to_wallet_id"`
	ToWalletName    string            `json:"to_wallet_name"`
	CreatedAt       time.Time         `json:"created_at"`
}

// ==================== 统计汇总 ====================

type WalletStatRequest struct {
	WalletID  string `form:"wallet_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type WalletStatResponse struct {
	TotalIncome     float64              `json:"total_income"`
	TotalExpense    float64              `json:"total_expense"`
	NetAmount       float64              `json:"net_amount"`
	TransactionCount int                 `json:"transaction_count"`
	CategoryStats   []CategoryStatItem   `json:"category_stats"`
}

type CategoryStatItem struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CategoryIcon string  `json:"category_icon"`
	CategoryColor string `json:"category_color"`
	Type         string  `json:"type"`
	Total        float64 `json:"total"`
	Count        int64   `json:"count"`
}
