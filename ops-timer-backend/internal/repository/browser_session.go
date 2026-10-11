package repository

import (
	"gorm.io/gorm"
	"ops-timer-backend/internal/model"
	"time"
)

type BrowserSessionRepository struct{ db *gorm.DB }

func NewBrowserSessionRepository(db *gorm.DB) *BrowserSessionRepository {
	return &BrowserSessionRepository{db}
}
func (r *BrowserSessionRepository) Create(session *model.BrowserSession) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at <= ?", time.Now()).Delete(&model.BrowserSession{}).Error; err != nil {
			return err
		}
		return tx.Create(session).Error
	})
}
func (r *BrowserSessionRepository) Find(hash string) (*model.BrowserSession, error) {
	var session model.BrowserSession
	err := r.db.Where("token_hash = ? AND expires_at > ?", hash, time.Now()).First(&session).Error
	return &session, err
}
func (r *BrowserSessionRepository) Delete(hash string) error {
	return r.db.Where("token_hash = ?", hash).Delete(&model.BrowserSession{}).Error
}
