package repository

import (
	"time"
)

// TokenBlacklistStoreAdapter adapts RevokedTokenRepository to the
// auth.TokenBlacklistStore interface.
type TokenBlacklistStoreAdapter struct {
	repo *RevokedTokenRepository
}

func NewTokenBlacklistStoreAdapter(repo *RevokedTokenRepository) *TokenBlacklistStoreAdapter {
	return &TokenBlacklistStoreAdapter{repo: repo}
}

func (a *TokenBlacklistStoreAdapter) Add(token string, expiresAt time.Time) error {
	return a.repo.Add(token, expiresAt)
}

func (a *TokenBlacklistStoreAdapter) Exists(token string) bool {
	return a.repo.Exists(token)
}

func (a *TokenBlacklistStoreAdapter) Cleanup() error {
	return a.repo.Cleanup()
}

func (a *TokenBlacklistStoreAdapter) LoadAll() (map[string]time.Time, error) {
	return a.repo.LoadAll()
}
