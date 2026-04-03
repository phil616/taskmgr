package repository

import (
	"ops-timer-backend/internal/model"

	"gorm.io/gorm"
)

type BudgetCategoryRepository struct {
	db *gorm.DB
}

func NewBudgetCategoryRepository(db *gorm.DB) *BudgetCategoryRepository {
	return &BudgetCategoryRepository{db: db}
}

func (r *BudgetCategoryRepository) Create(c *model.BudgetCategory) error {
	return r.db.Create(c).Error
}

func (r *BudgetCategoryRepository) FindByID(id string) (*model.BudgetCategory, error) {
	var c model.BudgetCategory
	if err := r.db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *BudgetCategoryRepository) List(catType string) ([]model.BudgetCategory, error) {
	var list []model.BudgetCategory
	q := r.db.Order("sort_order ASC, created_at ASC")
	if catType != "" {
		q = q.Where("type = ? OR type = 'both'", catType)
	}
	return list, q.Find(&list).Error
}

func (r *BudgetCategoryRepository) Update(c *model.BudgetCategory) error {
	return r.db.Save(c).Error
}

func (r *BudgetCategoryRepository) Delete(id string) error {
	return r.db.Delete(&model.BudgetCategory{}, "id = ?", id).Error
}

// CountByCategory 查询某分类的交易数量（删除前使用）
func (r *BudgetCategoryRepository) CountByCategory(id string) int64 {
	var count int64
	r.db.Model(&model.Transaction{}).Where("category_id = ? AND deleted_at IS NULL", id).Count(&count)
	return count
}
