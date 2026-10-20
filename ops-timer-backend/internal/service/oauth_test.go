package service

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOAuthPolicy(t *testing.T) {
	for _, tc := range []struct {
		scope, method, path string
		allowed             bool
	}{
		{"data:read", "GET", "/api/v1/todos", true},
		{"data:read", "POST", "/api/v1/todos", false},
		{"data:write", "POST", "/api/v1/todos", true},
		{"data:write", "GET", "/api/v1/todos", false},
		{"data:read data:write", "GET", "/api/v1/secrets/key/value", false},
		{"secrets:read", "GET", "/api/v1/secrets/key/value", true},
		{"secrets:read", "GET", "/api/v1/secret-audit-logs", true},
		{"secrets:read", "DELETE", "/api/v1/secrets/key", false},
		{"secrets:write", "DELETE", "/api/v1/secrets/key", true},
		{"data:read", "GET", "/api/v1/auth/profile", true},
		{"data:read data:write secrets:read secrets:write", "GET", "/api/v1/auth/token", false},
		{"data:read data:write secrets:read secrets:write", "POST", "/api/v1/auth/token/regenerate", false},
		{"data:read data:write", "PUT", "/api/v1/auth/password", false},
		{"data:read data:write", "PUT", "/api/v1/auth/profile", false},
		{"data:read data:write", "POST", "/api/v1/auth/login", false},
		{"data:read data:write", "GET", "/api/v1/backup/export", false},
		{"data:read data:write", "POST", "/api/v1/backup/import", false},
		{"data:write", "POST", "/api/v1/oauth/clients", false},
		{"data:read", "GET", "/api/v1/future-feature", false},
		{"data:read", "GET", "/api/v1/todos/../auth/token", false},
		{"data:read", "GET", "/api/v1/todos/%2e%2e/auth/token", false},
	} {
		if got := OAuthAllows(tc.scope, tc.method, tc.path); got != tc.allowed {
			t.Errorf("%s %s with %s: %v", tc.method, tc.path, tc.scope, got)
		}
	}
}

func TestOAuthConsentIntegrity(t *testing.T) {
	s, err := NewOAuthService(nil, nil, "https://tasks.example", "/mcp", strings.Repeat("s", 32), true)
	if err != nil {
		t.Fatal(err)
	}
	a := &OAuthAuthorization{ClientID: "client", RedirectURI: "https://client.example/callback", Scope: "data:read"}
	form, csrf, err := s.SignConsent(a)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := s.VerifyConsent(form, csrf)
	if err != nil || verified.ClientID != a.ClientID {
		t.Fatal("valid consent rejected", err)
	}
	for _, invalid := range []string{"", "tampered", form + "a", strings.Replace(form, ".", ".a", 1)} {
		if _, err := s.VerifyConsent(invalid, csrf); err == nil {
			t.Fatal("accepted invalid form")
		}
	}
	if _, err := s.VerifyConsent(form, strings.Repeat("x", 43)); err == nil {
		t.Fatal("accepted another browser's cookie")
	}
	a.Expires = time.Now().Add(-time.Second).Unix()
	// Re-sign an expired request to distinguish expiry checks from signature checks.
	raw, _ := json.Marshal(a)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	if _, err := s.VerifyConsent(payload+"."+s.sign(payload), csrf); err == nil {
		t.Fatal("accepted expired form")
	}
	s2, _ := NewOAuthService(nil, nil, "https://tasks.example", "/mcp", strings.Repeat("t", 32), true)
	if _, err := s2.VerifyConsent(form, csrf); err == nil {
		t.Fatal("accepted signature from another key")
	}
}

func TestOAuthURLValidation(t *testing.T) {
	for _, raw := range []string{"https://client.example/callback", "http://127.0.0.1:43123/callback", "http://[::1]:123/callback", "http://localhost:123/callback"} {
		u, _ := url.Parse(raw)
		if !safeOAuthURL(u) {
			t.Errorf("rejected %s", raw)
		}
	}
	for _, raw := range []string{"http://client.example/callback", "https://user:pass@client.example/callback", "https://client.example/callback#fragment", "javascript:alert(1)", "/callback"} {
		u, _ := url.Parse(raw)
		if safeOAuthURL(u) {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, issuer := range []string{"http://tasks.example", "https://tasks.example/path", "https://tasks.example?x=1", "https://tasks.example/#fragment"} {
		if _, err := NewOAuthService(nil, nil, issuer, "/mcp", strings.Repeat("s", 32), true); err == nil {
			t.Errorf("accepted issuer %s", issuer)
		}
	}
}
