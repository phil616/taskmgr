package service

import (
	"crypto/sha256"
	"encoding/hex"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"time"
)

const BrowserCookieName = "taskmgr_session"
const RememberDuration = 14 * 24 * time.Hour

func credentialDigest(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) BrowserLogin(req *dto.LoginRequest, remember bool) (string, *dto.UserResponse, time.Time, error) {
	user, err := s.authenticate(req)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	token, err := auth.GenerateAPIToken()
	if err != nil {
		return "", nil, time.Time{}, err
	}
	duration := 24 * time.Hour
	if remember {
		duration = RememberDuration
	}
	expires := time.Now().Add(duration)
	session := &model.BrowserSession{TokenHash: credentialDigest(token), UserID: user.ID, CredentialHash: credentialDigest(user.PasswordHash), ExpiresAt: expires}
	if err := s.sessionRepo.Create(session); err != nil {
		return "", nil, time.Time{}, err
	}
	profile := s.toUserResponse(user)
	return token, &profile, expires, nil
}

func (s *AuthService) ValidateBrowserSession(token string) (*model.User, error) {
	if len(token) != 64 {
		return nil, auth.ErrInvalidToken
	}
	session, err := s.sessionRepo.Find(credentialDigest(token))
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	user, err := s.userRepo.FindByID(session.UserID)
	if err != nil || session.CredentialHash != credentialDigest(user.PasswordHash) {
		return nil, auth.ErrInvalidToken
	}
	return user, nil
}

func (s *AuthService) RevokeBrowserSession(token string) error {
	return s.sessionRepo.Delete(credentialDigest(token))
}
func (s *AuthService) SecureBrowserCookie() bool { return s.cfg.CookieSecure }
