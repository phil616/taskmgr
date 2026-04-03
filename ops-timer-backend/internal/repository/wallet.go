package repository

import (
	"ops-timer-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type WalletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) Create(w *model.Wallet) error {
	return r.db.Create(w).Error
}

func (r *WalletRepository) FindByID(id string) (*model.Wallet, error) {
	var w model.Wallet
	if err := r.db.Where("id = ?", id).First(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) List() ([]model.Wallet, error) {
	var list []model.Wallet
	err := r.db.Order("sort_order ASC, created_at ASC").Find(&list).Error
	return list, err
}

func (r *WalletRepository) Update(w *model.Wallet) error {
	return r.db.Save(w).Error
}

func (r *WalletRepository) Delete(id string) error {
	return r.db.Delete(&model.Wallet{}, "id = ?", id).Error
}

// ClearDefault 清除所有钱包的默认标记
func (r *WalletRepository) ClearDefault() error {
	return r.db.Model(&model.Wallet{}).Where("is_default = true").Update("is_default", false).Error
}

// UpdateBalance 更新钱包余额
func (r *WalletRepository) UpdateBalance(id string, delta float64) error {
	return r.db.Model(&model.Wallet{}).Where("id = ?", id).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error
}

// SumByType 查询某钱包在指定时间范围内的收入/支出总额
// start/end 为零值时不限时间范围
func (r *WalletRepository) SumByType(walletID, txType string, start, end time.Time) float64 {
	var total float64
	q := r.db.Model(&model.Transaction{}).
		Where("wallet_id = ? AND type = ? AND deleted_at IS NULL", walletID, txType)
	if !start.IsZero() {
		q = q.Where("transaction_at >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("transaction_at < ?", end)
	}
	q.Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total
}
