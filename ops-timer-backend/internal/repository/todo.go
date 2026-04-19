package repository

import (
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"

	"gorm.io/gorm"
)

type TodoRepository struct {
	db *gorm.DB
}

func NewTodoRepository(db *gorm.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(todo *model.Todo) error {
	return r.db.Create(todo).Error
}

func (r *TodoRepository) FindByID(id string) (*model.Todo, error) {
	var todo model.Todo
	err := r.db.First(&todo, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

type TodoFilter struct {
	Status   string
	Priority string
	GroupID  string
	DueDate  string
	Page     int
	PageSize int
}

func (r *TodoRepository) List(filter TodoFilter) ([]model.Todo, int64, error) {
	var todos []model.Todo
	var total int64

	query := r.db.Model(&model.Todo{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.GroupID != "" {
		if filter.GroupID == "none" {
			query = query.Where("group_id IS NULL")
		} else {
			query = query.Where("group_id = ?", filter.GroupID)
		}
	}
	if filter.DueDate != "" {
		t, err := timeutil.ParseDate(filter.DueDate)
		if err == nil {
			query = query.Where("due_date <= ?", t)
		}
	}

	query.Count(&total)

	err := query.Order("sort_order ASC, created_at DESC").
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).
		Find(&todos).Error

	return todos, total, err
}

func (r *TodoRepository) Update(todo *model.Todo) error {
	return r.db.Save(todo).Error
}

func (r *TodoRepository) Delete(id string) error {
	return r.db.Delete(&model.Todo{}, "id = ?", id).Error
}

func (r *TodoRepository) BatchUpdateStatus(ids []string, status string) error {
	now := timeutil.Now()
	updates := map[string]interface{}{"status": status, "updated_at": now}
	if status == model.TodoStatusDone {
		updates["completed_at"] = &now
	}
	return r.db.Model(&model.Todo{}).Where("id IN ?", ids).Updates(updates).Error
}

func (r *TodoRepository) BatchDelete(ids []string) error {
	return r.db.Where("id IN ?", ids).Delete(&model.Todo{}).Error
}

func (r *TodoRepository) ClearGroupTodos(groupID string) error {
	return r.db.Model(&model.Todo{}).Where("group_id = ?", groupID).
		Update("group_id", nil).Error
}

func (r *TodoRepository) CountPending() (int64, error) {
	var count int64
	err := r.db.Model(&model.Todo{}).
		Where("status IN ?", []string{model.TodoStatusPending, model.TodoStatusInProgress}).
		Count(&count).Error
	return count, err
}
