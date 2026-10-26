package service

import (
	"ops-timer-backend/internal/model"
	"testing"
)

func TestMCPSettingsValidation(t *testing.T) {
	valid := model.MCPSettings{Enabled: true, Path: "/agents/mcp", ServerName: "tasks", ServerVersion: "1.0", TimeoutSeconds: 30, APITokenEnabled: true}
	for _, tc := range []struct {
		name   string
		change func(*model.MCPSettings)
	}{
		{"no auth", func(s *model.MCPSettings) { s.APITokenEnabled = false }},
		{"missing issuer", func(s *model.MCPSettings) { s.OAuthEnabled = true }},
		{"api collision", func(s *model.MCPSettings) { s.Path = "/api/v1" }},
		{"oauth collision", func(s *model.MCPSettings) { s.Path = "/oauth/authorize" }},
		{"health collision", func(s *model.MCPSettings) { s.Path = "/health/config" }},
		{"discovery collision", func(s *model.MCPSettings) { s.Path = "/.well-known/x" }},
		{"escaped path", func(s *model.MCPSettings) { s.Path = "/mcp%2Fconfig" }},
		{"traversal", func(s *model.MCPSettings) { s.Path = "/mcp/../api" }},
		{"timeout", func(s *model.MCPSettings) { s.TimeoutSeconds = 0 }},
		{"large timeout", func(s *model.MCPSettings) { s.TimeoutSeconds = 301 }},
		{"empty name", func(s *model.MCPSettings) { s.ServerName = " " }},
		{"http issuer", func(s *model.MCPSettings) { s.OAuthIssuer = "http://example.com" }},
		{"path issuer", func(s *model.MCPSettings) { s.OAuthIssuer = "https://example.com/path" }},
		{"query issuer", func(s *model.MCPSettings) { s.OAuthIssuer = "https://example.com?" }},
		{"userinfo", func(s *model.MCPSettings) { s.ExternalURL = "https://user:pass@example.com" }},
		{"fragment", func(s *model.MCPSettings) { s.ExternalURL = "https://example.com/#x" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := valid
			tc.change(&s)
			if ValidateMCPSettings(&s) == nil {
				t.Fatal("accepted invalid settings")
			}
		})
	}
	for _, issuer := range []string{"https://example.com", "http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:8080"} {
		s := valid
		s.OAuthEnabled, s.OAuthIssuer, s.APITokenEnabled = true, issuer, false
		if err := ValidateMCPSettings(&s); err != nil {
			t.Fatal(issuer, err)
		}
	}
	valid.Enabled, valid.APITokenEnabled = false, false
	if err := ValidateMCPSettings(&valid); err != nil {
		t.Fatal("disabled MCP should allow disabling all auth:", err)
	}
}
