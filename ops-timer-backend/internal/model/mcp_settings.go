package model

import "time"

// MCPSettings is a singleton. Credentials remain in their existing stores.
type MCPSettings struct {
	ID                       uint      `gorm:"primaryKey" json:"-"`
	Enabled                  bool      `json:"enabled"`
	Path                     string    `json:"path"`
	ExternalURL              string    `json:"external_url"`
	ServerName               string    `json:"server_name"`
	ServerVersion            string    `json:"server_version"`
	TimeoutSeconds           int       `json:"timeout_seconds"`
	GenericToolsEnabled      bool      `json:"generic_tools_enabled"`
	SensitiveToolsEnabled    bool      `json:"sensitive_tools_enabled"`
	APITokenEnabled          bool      `json:"api_token_enabled"`
	OAuthEnabled             bool      `json:"oauth_enabled"`
	OAuthIssuer              string    `json:"oauth_issuer"`
	OAuthDynamicRegistration bool      `json:"oauth_dynamic_registration"`
	UpdatedBy                string    `json:"updated_by"`
	UpdatedAt                time.Time `json:"updated_at"`
}
