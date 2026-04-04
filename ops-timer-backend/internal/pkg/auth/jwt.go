package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrTokenBlacklisted = errors.New("token has been revoked")
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// TokenBlacklistStore abstracts persistent storage for revoked JWT tokens.
type TokenBlacklistStore interface {
	Add(token string, expiresAt time.Time) error
	Exists(token string) bool
	Cleanup() error
	LoadAll() (map[string]time.Time, error)
}

type JWTManager struct {
	secret      []byte
	expiryHours int
	blacklist   map[string]time.Time
	store       TokenBlacklistStore
	mu          sync.RWMutex
}

func NewJWTManager(secret string, expiryHours int, store TokenBlacklistStore) *JWTManager {
	m := &JWTManager{
		secret:      []byte(secret),
		expiryHours: expiryHours,
		blacklist:   make(map[string]time.Time),
		store:       store,
	}
	if store != nil {
		if persisted, err := store.LoadAll(); err == nil {
			m.blacklist = persisted
		}
	}
	return m
}

func (m *JWTManager) GenerateToken(userID, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	m.mu.RLock()
	if _, ok := m.blacklist[tokenStr]; ok {
		m.mu.RUnlock()
		return nil, ErrTokenBlacklisted
	}
	m.mu.RUnlock()

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (m *JWTManager) RevokeToken(tokenStr string) {
	expiresAt := time.Now().Add(time.Duration(m.expiryHours) * time.Hour)

	m.mu.Lock()
	m.blacklist[tokenStr] = expiresAt
	m.mu.Unlock()

	if m.store != nil {
		_ = m.store.Add(tokenStr, expiresAt)
	}
}

func (m *JWTManager) CleanupBlacklist() {
	m.mu.Lock()
	now := time.Now()
	for token, expiry := range m.blacklist {
		if now.After(expiry) {
			delete(m.blacklist, token)
		}
	}
	m.mu.Unlock()

	if m.store != nil {
		_ = m.store.Cleanup()
	}
}

func GenerateAPIToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
