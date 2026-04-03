package service

import (
	"errors"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/repository"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrAccountLocked      = errors.New("账户已被锁定，请稍后再试")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrOldPasswordWrong   = errors.New("旧密码错误")
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
	cfg        *config.AuthConfig

	loginAttempts map[string]*loginAttempt
	mu            sync.RWMutex
}

type loginAttempt struct {
	Count    int
	LockedAt time.Time
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager, cfg *config.AuthConfig) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		jwtManager:    jwtManager,
		cfg:           cfg,
		loginAttempts: make(map[string]*loginAttempt),
	}
}

func (s *AuthService) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	s.mu.RLock()
	attempt, exists := s.loginAttempts[req.Username]
	s.mu.RUnlock()

	if exists && attempt.Count >= s.cfg.LoginLockAttempts {
		if time.Since(attempt.LockedAt) < time.Duration(s.cfg.LoginLockMinutes)*time.Minute {
			return nil, ErrAccountLocked
		}
		s.mu.Lock()
		delete(s.loginAttempts, req.Username)
		s.mu.Unlock()
	}

	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		s.recordFailedAttempt(req.Username)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.recordFailedAttempt(req.Username)
		return nil, ErrInvalidCredentials
	}

	s.mu.Lock()
	delete(s.loginAttempts, req.Username)
	s.mu.Unlock()

	token, err := s.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		User:  s.toUserResponse(user),
	}, nil
}

func (s *AuthService) recordFailedAttempt(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	attempt, exists := s.loginAttempts[username]
	if !exists {
		attempt = &loginAttempt{}
		s.loginAttempts[username] = attempt
	}
	attempt.Count++
	if attempt.Count >= s.cfg.LoginLockAttempts {
		attempt.LockedAt = time.Now()
	}
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
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}
}
