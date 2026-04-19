package repository

import (
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"strings"

	"gorm.io/gorm"
)

type UnitRepository struct {
	db *gorm.DB
}

func NewUnitRepository(db *gorm.DB) *UnitRepository {
	return &UnitRepository{db: db}
}

func (r *UnitRepository) Create(unit *model.Unit) error {
	return r.db.Create(unit).Error
}

func (r *UnitRepository) FindByID(id string) (*model.Unit, error) {
	var unit model.Unit
	err := r.db.First(&unit, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &unit, nil
}

type UnitFilter struct {
	Types     []string
	Status    string
	ProjectID string
	Tags      []string
	Priority  string
	Q         string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

func (r *UnitRepository) List(filter UnitFilter) ([]model.Unit, int64, error) {
	var units []model.Unit
	var total int64

	query := r.db.Model(&model.Unit{})

	if len(filter.Types) > 0 {
		query = query.Where("type IN ?", filter.Types)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ProjectID != "" {
		if filter.ProjectID == "none" {
			query = query.Where("project_id IS NULL")
		} else {
			query = query.Where("project_id = ?", filter.ProjectID)
		}
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.Q != "" {
		search := "%" + filter.Q + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", search, search)
	}
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query = query.Where("tags LIKE ?", "%\""+tag+"\"%")
		}
	}

	query.Count(&total)

	orderClause := r.buildOrderClause(filter.SortBy, filter.SortOrder)
	err := query.Order(orderClause).
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).
		Find(&units).Error

	return units, total, err
}

func (r *UnitRepository) buildOrderClause(sortBy, sortOrder string) string {
	order := "desc"
	if strings.ToLower(sortOrder) == "asc" {
		order = "asc"
	}

	switch sortBy {
	case "priority":
		return "CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 WHEN 'low' THEN 3 END " + order
	case "updated_at":
		return "updated_at " + order
	case "created_at":
		return "created_at " + order
	default:
		return "created_at DESC"
	}
}

func (r *UnitRepository) Update(unit *model.Unit) error {
	return r.db.Save(unit).Error
}

func (r *UnitRepository) Delete(id string) error {
	return r.db.Delete(&model.Unit{}, "id = ?", id).Error
}

func (r *UnitRepository) ListByProjectID(projectID string, page, pageSize int) ([]model.Unit, int64, error) {
	var units []model.Unit
	var total int64

	query := r.db.Model(&model.Unit{}).Where("project_id = ?", projectID)
	query.Count(&total)

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&units).Error

	return units, total, err
}

func (r *UnitRepository) CountByStatus() (map[string]int64, error) {
	type Result struct {
		Status string
		Count  int64
	}
	var results []Result
	err := r.db.Model(&model.Unit{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	m := make(map[string]int64)
	for _, r := range results {
		m[r.Status] = r.Count
	}
	return m, nil
}

func (r *UnitRepository) CountByProjectAndStatus(projectID, status string) (int64, error) {
	var count int64
	query := r.db.Model(&model.Unit{}).Where("project_id = ?", projectID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *UnitRepository) CountExpiring(days int) (int64, error) {
	var count int64
	now := timeutil.Now()
	deadline := now.AddDate(0, 0, days)
	err := r.db.Model(&model.Unit{}).
		Where("type = ? AND status = ? AND target_time <= ? AND target_time > ?",
			model.UnitTypeTimeCountdown, model.UnitStatusActive, deadline, now).
		Count(&count).Error
	return count, err
}

func (r *UnitRepository) CountExpired() (int64, error) {
	var count int64
	err := r.db.Model(&model.Unit{}).
		Where("type = ? AND status = ? AND target_time <= ?",
			model.UnitTypeTimeCountdown, model.UnitStatusActive, timeutil.Now()).
		Count(&count).Error
	return count, err
}

func (r *UnitRepository) FindActiveUnits() ([]model.Unit, error) {
	var units []model.Unit
	err := r.db.Where("status = ?", model.UnitStatusActive).Find(&units).Error
	return units, err
}
