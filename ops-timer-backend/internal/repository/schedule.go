package repository

import (
	"ops-timer-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type ScheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Create(schedule *model.Schedule) error {
	return r.db.Create(schedule).Error
}

func (r *ScheduleRepository) GetByID(id string) (*model.Schedule, error) {
	var s model.Schedule
	err := r.db.Preload("Resources").Where("id = ?", id).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ScheduleRepository) Update(schedule *model.Schedule) error {
	return r.db.Save(schedule).Error
}

func (r *ScheduleRepository) Delete(id string) error {
	return r.db.Delete(&model.Schedule{}, "id = ?", id).Error
}

// List 按日期范围查询日程，返回总数
func (r *ScheduleRepository) List(startDate, endDate time.Time, status string, page, pageSize int) ([]model.Schedule, int64, error) {
	var schedules []model.Schedule
	var total int64

	query := r.db.Model(&model.Schedule{})

	// 与区间有交集的日程：start_time < endDate AND end_time > startDate
	if !startDate.IsZero() && !endDate.IsZero() {
		query = query.Where("start_time < ? AND end_time > ?", endDate, startDate)
	} else if !startDate.IsZero() {
		query = query.Where("end_time > ?", startDate)
	} else if !endDate.IsZero() {
		query = query.Where("start_time < ?", endDate)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Preload("Resources").
		Order("start_time ASC").
		Offset(offset).Limit(pageSize).
		Find(&schedules).Error

	return schedules, total, err
}

// AddResource 给日程添加关联资源
func (r *ScheduleRepository) AddResource(res *model.ScheduleResource) error {
	return r.db.Create(res).Error
}

// RemoveResource 删除关联资源
func (r *ScheduleRepository) RemoveResource(scheduleID, resourceID string) error {
	return r.db.Delete(&model.ScheduleResource{}, "id = ? AND schedule_id = ?", resourceID, scheduleID).Error
}

// GetResourceByID 查询单个关联资源
func (r *ScheduleRepository) GetResourceByID(id string) (*model.ScheduleResource, error) {
	var res model.ScheduleResource
	if err := r.db.Where("id = ?", id).First(&res).Error; err != nil {
		return nil, err
	}
	return &res, nil
}
