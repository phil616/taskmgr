package repository

import (
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(n *model.Notification) error {
	return r.db.Create(n).Error
}

func (r *NotificationRepository) FindByID(id string) (*model.Notification, error) {
	var n model.Notification
	err := r.db.First(&n, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepository) List(isRead *bool, level string, page, pageSize int) ([]model.Notification, int64, error) {
	var notifications []model.Notification
	var total int64

	query := r.db.Model(&model.Notification{})
	if isRead != nil {
		query = query.Where("is_read = ?", *isRead)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}

	query.Count(&total)

	err := query.Order("triggered_at DESC").
		Preload("Unit", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title")
		}).
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *NotificationRepository) MarkAsRead(id string) error {
	now := timeutil.Now()
	return r.db.Model(&model.Notification{}).Where("id = ?", id).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now}).Error
}

func (r *NotificationRepository) MarkAllAsRead() error {
	now := timeutil.Now()
	return r.db.Model(&model.Notification{}).Where("is_read = ?", false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now}).Error
}

func (r *NotificationRepository) UnreadCount() (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).Where("is_read = ?", false).Count(&count).Error
	return count, err
}

func (r *NotificationRepository) Delete(id string) error {
	return r.db.Delete(&model.Notification{}, "id = ?", id).Error
}

func (r *NotificationRepository) ExistsTodayForUnit(unitID, level string) (bool, error) {
	var count int64
	today := timeutil.StartOfDay(timeutil.Now())
	err := r.db.Model(&model.Notification{}).
		Where("unit_id = ? AND level = ? AND triggered_at >= ?", unitID, level, today).
		Count(&count).Error
	return count > 0, err
}
