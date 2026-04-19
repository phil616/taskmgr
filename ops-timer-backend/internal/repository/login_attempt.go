package repository

import (
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoginAttemptRepository struct {
	db *gorm.DB
}

func NewLoginAttemptRepository(db *gorm.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

func (r *LoginAttemptRepository) Get(username string) (*model.LoginAttempt, error) {
	var attempt model.LoginAttempt
	err := r.db.Where("username = ?", username).First(&attempt).Error
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

func (r *LoginAttemptRepository) Increment(username string, lockThreshold int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var attempt model.LoginAttempt
		err := tx.Where("username = ?", username).First(&attempt).Error

		if err == gorm.ErrRecordNotFound {
			attempt = model.LoginAttempt{Username: username, Count: 1}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "username"}},
				DoUpdates: clause.Assignments(map[string]interface{}{"count": gorm.Expr("count + 1")}),
			}).Create(&attempt).Error
		}
		if err != nil {
			return err
		}

		attempt.Count++
		if attempt.Count >= lockThreshold {
			attempt.LockedAt = timeutil.Now()
		}
		return tx.Save(&attempt).Error
	})
}

func (r *LoginAttemptRepository) Reset(username string) error {
	return r.db.Where("username = ?", username).Delete(&model.LoginAttempt{}).Error
}
