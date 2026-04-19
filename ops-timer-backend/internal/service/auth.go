package service

import (
	"errors"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/pkg/timeutil"
	"ops-timer-backend/internal/repository"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrAccountLocked      = errors.New("账户已被锁定，请稍后再试")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrOldPasswordWrong   = errors.New("旧密码错误")
)

type AuthService struct {
	userRepo         *repository.UserRepository
	loginAttemptRepo *repository.LoginAttemptRepository
	jwtManager       *auth.JWTManager
	cfg              *config.AuthConfig
}

func NewAuthService(
	userRepo *repository.UserRepository,
	loginAttemptRepo *repository.LoginAttemptRepository,
	jwtManager *auth.JWTManager,
	cfg *config.AuthConfig,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		loginAttemptRepo: loginAttemptRepo,
		jwtManager:       jwtManager,
		cfg:              cfg,
	}
}

func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	attempt, err := s.loginAttemptRepo.Get(req.Username)
	if err == nil && attempt.Count >= s.cfg.LoginLockAttempts {
		if time.Since(attempt.LockedAt) < time.Duration(s.cfg.LoginLockMinutes)*time.Minute {
			return nil, ErrAccountLocked
		}
		_ = s.loginAttemptRepo.Reset(req.Username)
	}

	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		_ = s.loginAttemptRepo.Increment(req.Username, s.cfg.LoginLockAttempts)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		_ = s.loginAttemptRepo.Increment(req.Username, s.cfg.LoginLockAttempts)
		return nil, ErrInvalidCredentials
	}

	_ = s.loginAttemptRepo.Reset(req.Username)

	token, err := s.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User:  s.toUserResponse(user),
	}, nil
}

func (s *AuthService) Logout(tokenStr string) {
	s.jwtManager.RevokeToken(tokenStr)
}

func (s *AuthService) GetProfile(userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	resp := s.toUserResponse(user)
	return &resp, nil
}

func (s *AuthService) UpdateProfile(userID string, req *dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	user.Email = req.Email

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	resp := s.toUserResponse(user)
	return &resp, nil
}

func (s *AuthService) ChangePassword(userID string, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return ErrOldPasswordWrong
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return s.userRepo.Update(user)
}

func (s *AuthService) GetAPIToken(userID string) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}
	return user.APIToken, nil
}

func (s *AuthService) RegenerateAPIToken(userID string) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	token, err := auth.GenerateAPIToken()
	if err != nil {
		return "", err
	}

	user.APIToken = token
	if err := s.userRepo.Update(user); err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) EnsureAdminExists(username, password string) error {
	count, err := s.userRepo.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	apiToken, err := auth.GenerateAPIToken()
	if err != nil {
		return err
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  "Admin",
		APIToken:     apiToken,
	}

	return s.userRepo.Create(user)
}

func (s *AuthService) FindByAPIToken(token string) (*model.User, error) {
	return s.userRepo.FindByAPIToken(token)
}

func (s *AuthService) toUserResponse(user *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		CreatedAt:   timeutil.Normalize(user.CreatedAt).Format(time.RFC3339),
		UpdatedAt:   timeutil.Normalize(user.UpdatedAt).Format(time.RFC3339),
	}
}

// Ensure LoginAttemptRepository satisfies usage by gorm.ErrRecordNotFound
var _ error = gorm.ErrRecordNotFound
