package model

import "time"

// OAuth credentials are stored as SHA-256 digests, never as bearer values.
type OAuthClient struct {
	ID             string    `gorm:"primaryKey;size:64" json:"client_id"`
	Name           string    `gorm:"size:128;not null" json:"client_name"`
	RedirectURIs   []string  `gorm:"serializer:json;type:text;not null" json:"redirect_uris"`
	RegistrationIP string    `gorm:"size:64;index" json:"-"`
	CreatedAt      time.Time `json:"-"`
}

type OAuthCode struct {
	Hash           string    `gorm:"primaryKey;size:64"`
	ClientID       string    `gorm:"size:64;not null"`
	UserID         string    `gorm:"size:36;not null"`
	CredentialHash string    `gorm:"size:64;not null"`
	RedirectURI    string    `gorm:"type:text;not null"`
	Challenge      string    `gorm:"size:43;not null"`
	Scope          string    `gorm:"size:256;not null"`
	Resource       string    `gorm:"type:text;not null"`
	ExpiresAt      time.Time `gorm:"index;not null"`
	Used           bool      `gorm:"not null;default:false"`
	GrantID        string    `gorm:"size:36"`
}

type OAuthGrant struct {
	ID             string    `gorm:"primaryKey;size:36" json:"id"`
	ClientID       string    `gorm:"size:64;not null" json:"client_id"`
	UserID         string    `gorm:"size:36;index;not null" json:"-"`
	CredentialHash string    `gorm:"size:64;not null" json:"-"`
	Scope          string    `gorm:"size:256;not null" json:"scope"`
	Resource       string    `gorm:"type:text;not null" json:"resource"`
	Revoked        bool      `gorm:"not null;default:false" json:"revoked"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `gorm:"index;not null" json:"expires_at"`
}

type OAuthToken struct {
	AccessHash      string    `gorm:"primaryKey;size:64"`
	RefreshHash     string    `gorm:"uniqueIndex;size:64;not null"`
	GrantID         string    `gorm:"size:36;index;not null"`
	Scope           string    `gorm:"size:256;not null"`
	AccessExpiresAt time.Time `gorm:"index;not null"`
	RefreshUsed     bool      `gorm:"not null;default:false"`
}
