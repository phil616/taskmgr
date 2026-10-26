package repository

import (
	"ops-timer-backend/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id string) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByAPIToken(token string) (*model.User, error) {
	var user model.User
	err := r.db.Where("api_token = ?", token).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

func (r *UserRepository) EnsureInitialAdmin() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.User{}).Where("is_admin = true").Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		var first model.User
		if err := tx.Order("created_at ASC, id ASC").First(&first).Error; err != nil {
			return err
		}
		return tx.Model(&first).Update("is_admin", true).Error
	})
}

// FindAllWithEmail 返回所有设置了邮箱的用户
func (r *UserRepository) FindAllWithEmail() ([]model.User, error) {
	var users []model.User
	err := r.db.Where("email != ''").Find(&users).Error
	return users, err
}

// FindByEmail 根据邮箱查找用户（不区分大小写）
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
