package repository

import (
	"ops-timer-backend/internal/model"

	"gorm.io/gorm"
)

type SecretRepository struct {
	db *gorm.DB
}

func NewSecretRepository(db *gorm.DB) *SecretRepository {
	return &SecretRepository{db: db}
}

func (r *SecretRepository) Create(secret *model.Secret) error {
	return r.db.Create(secret).Error
}

func (r *SecretRepository) FindByID(id string) (*model.Secret, error) {
	var secret model.Secret
	err := r.db.Preload("Project").First(&secret, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *SecretRepository) FindByName(name string) (*model.Secret, error) {
	var secret model.Secret
	err := r.db.Preload("Project").First(&secret, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *SecretRepository) List(name, tag, projectID string, page, pageSize int) ([]model.Secret, int64, error) {
	var secrets []model.Secret
	var total int64

	query := r.db.Model(&model.Secret{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if tag != "" {
		query = query.Where("tags LIKE ?", "%\""+tag+"\"%")
	}
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	query.Count(&total)

	err := query.Preload("Project").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&secrets).Error

	return secrets, total, err
}

func (r *SecretRepository) Update(secret *model.Secret) error {
	return r.db.Save(secret).Error
}

func (r *SecretRepository) Delete(id string) error {
	return r.db.Delete(&model.Secret{}, "id = ?", id).Error
}

// SecretAuditLogRepository

type SecretAuditLogRepository struct {
	db *gorm.DB
}

func NewSecretAuditLogRepository(db *gorm.DB) *SecretAuditLogRepository {
	return &SecretAuditLogRepository{db: db}
}

func (r *SecretAuditLogRepository) Create(log *model.SecretAuditLog) error {
	return r.db.Create(log).Error
}

func (r *SecretAuditLogRepository) ListBySecret(secretID, action string, page, pageSize int) ([]model.SecretAuditLog, int64, error) {
	var logs []model.SecretAuditLog
	var total int64

	query := r.db.Model(&model.SecretAuditLog{})
	if secretID != "" {
		query = query.Where("secret_id = ?", secretID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	query.Count(&total)

	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&logs).Error

	return logs, total, err
}
