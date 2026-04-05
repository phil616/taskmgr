package repository

import (
	"ops-timer-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(t *model.Transaction) error {
	return r.db.Create(t).Error
}

func (r *TransactionRepository) FindByID(id string) (*model.Transaction, error) {
	var t model.Transaction
	err := r.db.Preload("Category").Preload("Wallet").Preload("ToWallet").Preload("Project").
		Where("id = ?", id).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type TransactionFilter struct {
	WalletID   string
	CategoryID string
	ProjectID  string
	TxType     string
	StartDate  time.Time
	EndDate    time.Time
	MinAmount  float64
	MaxAmount  float64
	Keyword    string
	Page       int
	PageSize   int
}

func (r *TransactionRepository) List(f *TransactionFilter) ([]model.Transaction, int64, error) {
	var list []model.Transaction
	var total int64

	q := r.db.Model(&model.Transaction{})
	if f.WalletID != "" {
		q = q.Where("wallet_id = ?", f.WalletID)
	}
	if f.CategoryID != "" {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.ProjectID != "" {
		q = q.Where("project_id = ?", f.ProjectID)
	}
	if f.TxType != "" {
		q = q.Where("type = ?", f.TxType)
	}
	if !f.StartDate.IsZero() {
		q = q.Where("transaction_at >= ?", f.StartDate)
	}
	if !f.EndDate.IsZero() {
		q = q.Where("transaction_at <= ?", f.EndDate)
	}
	if f.MinAmount > 0 {
		q = q.Where("amount >= ?", f.MinAmount)
	}
	if f.MaxAmount > 0 {
		q = q.Where("amount <= ?", f.MaxAmount)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("note LIKE ?", like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (f.Page - 1) * f.PageSize
	err := q.Preload("Category").Preload("Wallet").Preload("ToWallet").Preload("Project").
		Order("transaction_at DESC, created_at DESC").
		Offset(offset).Limit(f.PageSize).
		Find(&list).Error
	return list, total, err
}

func (r *TransactionRepository) Update(t *model.Transaction) error {
	return r.db.Save(t).Error
}

func (r *TransactionRepository) Delete(id string) error {
	return r.db.Delete(&model.Transaction{}, "id = ?", id).Error
}

// StatByWallet 汇总某钱包在日期范围内的收入/支出
type TxStat struct {
	Type  string
	Total float64
	Count int64
}

func (r *TransactionRepository) StatByWallet(walletID string, start, end time.Time) []TxStat {
	var results []TxStat
	q := r.db.Model(&model.Transaction{}).
		Select("type, COALESCE(SUM(amount), 0) as total, COUNT(*) as count").
		Group("type")
	if walletID != "" {
		q = q.Where("wallet_id = ?", walletID)
	}
	if !start.IsZero() {
		q = q.Where("transaction_at >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("transaction_at <= ?", end)
	}
	q.Scan(&results)
	return results
}

// StatByCategory 按分类汇总
type CategoryStat struct {
	CategoryID   *string
	Total        float64
	Count        int64
	TxType       string
}

func (r *TransactionRepository) StatByProject(projectID string) []TxStat {
	var results []TxStat
	q := r.db.Model(&model.Transaction{}).
		Select("type, COALESCE(SUM(amount), 0) as total, COUNT(*) as count").
		Where("project_id = ?", projectID).
		Group("type")
	q.Scan(&results)
	return results
}

func (r *TransactionRepository) ClearProjectTransactions(projectID string) error {
	return r.db.Model(&model.Transaction{}).Where("project_id = ?", projectID).
		Update("project_id", nil).Error
}

func (r *TransactionRepository) StatByCategory(walletID string, start, end time.Time) []CategoryStat {
	var results []CategoryStat
	q := r.db.Model(&model.Transaction{}).
		Select("category_id, type as tx_type, COALESCE(SUM(amount), 0) as total, COUNT(*) as count").
		Group("category_id, type").
		Where("type IN ?", []string{model.TransactionTypeIncome, model.TransactionTypeExpense})
	if walletID != "" {
		q = q.Where("wallet_id = ?", walletID)
	}
	if !start.IsZero() {
		q = q.Where("transaction_at >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("transaction_at <= ?", end)
	}
	q.Scan(&results)
	return results
}
