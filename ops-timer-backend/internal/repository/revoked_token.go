package repository

import (
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"
	"time"

	"gorm.io/gorm"
)

type RevokedTokenRepository struct {
	db *gorm.DB
}

func NewRevokedTokenRepository(db *gorm.DB) *RevokedTokenRepository {
	return &RevokedTokenRepository{db: db}
}

func (r *RevokedTokenRepository) Add(token string, expiresAt time.Time) error {
	return r.db.Create(&model.RevokedToken{
		Token:     token,
		ExpiresAt: expiresAt,
	}).Error
}

func (r *RevokedTokenRepository) Exists(token string) bool {
	var count int64
	r.db.Model(&model.RevokedToken{}).Where("token = ? AND expires_at > ?", token, timeutil.Now()).Count(&count)
	return count > 0
}

func (r *RevokedTokenRepository) Cleanup() error {
	return r.db.Where("expires_at <= ?", timeutil.Now()).Delete(&model.RevokedToken{}).Error
}

func (r *RevokedTokenRepository) LoadAll() (map[string]time.Time, error) {
	var tokens []model.RevokedToken
	err := r.db.Where("expires_at > ?", timeutil.Now()).Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]time.Time, len(tokens))
	for _, t := range tokens {
		result[t.Token] = t.ExpiresAt
	}
	return result, nil
}
