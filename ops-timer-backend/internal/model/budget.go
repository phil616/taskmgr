package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	WalletTypeBank    = "bank"
	WalletTypeCash    = "cash"
	WalletTypeCredit  = "credit"
	WalletTypeAlipay  = "alipay"
	WalletTypeWechat  = "wechat"
	WalletTypeOther   = "other"

	TransactionTypeIncome  = "income"
	TransactionTypeExpense = "expense"
	TransactionTypeTransfer = "transfer"
)

// Wallet 钱包/账户
type Wallet struct {
	ID          string         `gorm:"primaryKey;size:36" json:"id"`
	Name        string         `gorm:"size:64;not null" json:"name"`
	Type        string         `gorm:"size:20;not null;default:bank" json:"type"`
	Balance     float64        `gorm:"not null;default:0" json:"balance"`
	Currency    string         `gorm:"size:10;not null;default:CNY" json:"currency"`
	Color       string         `gorm:"size:20" json:"color"`
	Icon        string         `gorm:"size:64" json:"icon"`
	Description string         `gorm:"size:512" json:"description"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

// BudgetCategory 收支分类
type BudgetCategory struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Type      string    `gorm:"size:20;not null" json:"type"` // income / expense / both
	Color     string    `gorm:"size:20" json:"color"`
	Icon      string    `gorm:"size:64" json:"icon"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	IsSystem  bool      `gorm:"default:false" json:"is_system"` // 系统内置分类不可删除
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Transaction 收支记录
type Transaction struct {
	ID              string          `gorm:"primaryKey;size:36" json:"id"`
	WalletID        string          `gorm:"size:36;not null;index" json:"wallet_id"`
	CategoryID      *string         `gorm:"size:36;index" json:"category_id"`
	Type            string          `gorm:"size:20;not null" json:"type"` // income / expense / transfer
	Amount          float64         `gorm:"not null" json:"amount"`
	Note            string          `gorm:"size:512" json:"note"`
	Tags            JSONStringArray `gorm:"type:text" json:"tags"`
	TransactionAt   time.Time       `gorm:"not null;index" json:"transaction_at"`
	// 转账目标钱包（仅 transfer 类型使用）
	ToWalletID      *string         `gorm:"size:36;index" json:"to_wallet_id"`
	DeletedAt       gorm.DeletedAt  `gorm:"index" json:"-"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	Wallet   *Wallet         `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
	Category *BudgetCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	ToWallet *Wallet         `gorm:"foreignKey:ToWalletID" json:"to_wallet,omitempty"`
}
