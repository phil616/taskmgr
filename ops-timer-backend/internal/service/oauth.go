package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
)

const OAuthAccessLifetime = 15 * time.Minute
const OAuthGrantLifetime = 30 * 24 * time.Hour

var OAuthScopes = []string{"data:read", "data:write", "secrets:read", "secrets:write"}
var ErrOAuthRequest = errors.New("invalid_request")
var ErrOAuthGrant = errors.New("invalid_grant")
var ErrOAuthScope = errors.New("invalid_scope")
var ErrOAuthClient = errors.New("invalid_client")
var ErrOAuthTarget = errors.New("invalid_target")
var ErrOAuthRateLimit = errors.New("registration rate limit exceeded")
var pkceChallenge = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
var pkceVerifier = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

type OAuthService struct {
	db                  *gorm.DB
	auth                *AuthService
	Issuer              string
	Resource            string
	DynamicRegistration bool
	formKey             []byte
}

func NewOAuthService(db *gorm.DB, auth *AuthService, issuer, resourcePath, secret string, dynamic bool) (*OAuthService, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	u, err := url.Parse(issuer)
	if err != nil || !safeOAuthURL(u) || u.RawQuery != "" || (u.Path != "" && u.Path != "/") || len(secret) < 32 {
		return nil, fmt.Errorf("OAuth issuer must be an HTTPS origin (HTTP allowed for loopback development) and signing secret must be at least 32 bytes")
	}
	if !strings.HasPrefix(resourcePath, "/") || strings.HasPrefix(resourcePath, "//") || strings.ContainsAny(resourcePath, "?#\\%") || path.Clean(resourcePath) != resourcePath {
		return nil, ErrOAuthRequest
	}
	return &OAuthService{db: db, auth: auth, Issuer: issuer, Resource: issuer + resourcePath, formKey: []byte(secret), DynamicRegistration: dynamic}, nil
}

func safeOAuthURL(u *url.URL) bool {
	if u == nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))
}

func (s *OAuthService) RegisterClient(name string, redirects []string, ip string) (*model.OAuthClient, error) {
	if len(name) == 0 || len(name) > 128 || len(redirects) == 0 || len(redirects) > 10 {
		return nil, ErrOAuthRequest
	}
	for _, redirect := range redirects {
		u, err := url.Parse(redirect)
		if err != nil || !safeOAuthURL(u) || len(redirect) > 2048 {
			return nil, ErrOAuthRequest
		}
		if _, err := url.ParseQuery(u.RawQuery); err != nil {
			return nil, ErrOAuthRequest
		}
		for _, reserved := range []string{"code", "state", "iss", "error", "error_description"} {
			if u.Query().Has(reserved) {
				return nil, ErrOAuthRequest
			}
		}
	}
	client := &model.OAuthClient{ID: uuid.NewString(), Name: name, RedirectURIs: redirects, RegistrationIP: credentialDigest(ip)}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Serialize registration per IP across processes as well as goroutines.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", client.RegistrationIP).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.OAuthClient{}).Where("registration_ip = ? AND created_at > ?", client.RegistrationIP, time.Now().Add(-time.Hour)).Count(&count).Error; err != nil {
			return err
		}
		if count >= 20 {
			return ErrOAuthRateLimit
		}
		return tx.Create(client).Error
	})
	return client, err
}

type OAuthAuthorization struct {
	ClientID        string `json:"client_id"`
	RedirectURI     string `json:"redirect_uri"`
	ResponseType    string `json:"response_type"`
	Challenge       string `json:"code_challenge"`
	ChallengeMethod string `json:"code_challenge_method"`
	Scope           string `json:"scope"`
	State           string `json:"state"`
	Resource        string `json:"resource"`
	CSRF            string `json:"csrf"`
	Expires         int64  `json:"expires"`
}

func normalizeOAuthScope(scope string) (string, error) {
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		fields = []string{"data:read"}
	}
	for _, field := range fields {
		if !slices.Contains(OAuthScopes, field) {
			return "", ErrOAuthScope
		}
	}
	slices.Sort(fields)
	return strings.Join(slices.Compact(fields), " "), nil
}

func (s *OAuthService) ValidateAuthorization(a *OAuthAuthorization) (*model.OAuthClient, error) {
	var client model.OAuthClient
	if len(a.ClientID) > 64 || s.db.First(&client, "id = ?", a.ClientID).Error != nil {
		return nil, ErrOAuthClient
	}
	// Exact matching, including query strings and loopback ports.
	if !slices.Contains(client.RedirectURIs, a.RedirectURI) {
		return nil, ErrOAuthRequest
	}
	if a.ResponseType != "code" || a.ChallengeMethod != "S256" || !pkceChallenge.MatchString(a.Challenge) || len(a.State) > 2048 {
		return nil, ErrOAuthRequest
	}
	if a.Resource == "" {
		a.Resource = s.Resource
	}
	if a.Resource != s.Resource {
		return nil, ErrOAuthTarget
	}
	scope, err := normalizeOAuthScope(a.Scope)
	if err != nil {
		return nil, err
	}
	a.Scope = scope
	return &client, nil
}

func oauthRandom(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// Sign the entire consent request so POST cannot change what was displayed.
// The random CSRF cookie is also included in the signature and expires in 10 minutes.
func (s *OAuthService) SignConsent(a *OAuthAuthorization) (string, string, error) {
	csrf, err := oauthRandom("")
	if err != nil {
		return "", "", err
	}
	a.CSRF = csrf
	a.Expires = time.Now().Add(10 * time.Minute).Unix()
	raw, err := json.Marshal(a)
	if err != nil {
		return "", "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + s.sign(payload), csrf, nil
}

func (s *OAuthService) sign(payload string) string {
	mac := hmac.New(sha256.New, s.formKey)
	mac.Write([]byte("taskmgr-oauth-consent:" + payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *OAuthService) VerifyConsent(form, csrf string) (*OAuthAuthorization, error) {
	parts := strings.Split(form, ".")
	if len(form) > 16384 || len(parts) != 2 || !hmac.Equal([]byte(parts[1]), []byte(s.sign(parts[0]))) {
		return nil, ErrOAuthRequest
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrOAuthRequest
	}
	var a OAuthAuthorization
	if json.Unmarshal(raw, &a) != nil || a.Expires <= time.Now().Unix() || len(csrf) != 43 || subtle.ConstantTimeCompare([]byte(csrf), []byte(a.CSRF)) != 1 {
		return nil, ErrOAuthRequest
	}
	return &a, nil
}

func (s *OAuthService) Authorize(a *OAuthAuthorization, username, password string) (string, error) {
	if _, err := s.ValidateAuthorization(a); err != nil {
		return "", err
	}
	if len(username) > 32 || len(password) > 72 || username == "" || password == "" {
		return "", ErrInvalidCredentials
	}
	user, err := s.auth.authenticate(&dto.LoginRequest{Username: username, Password: password})
	if err != nil {
		return "", err
	}
	code, err := oauthRandom("ot_code_")
	if err != nil {
		return "", err
	}
	record := &model.OAuthCode{Hash: credentialDigest(code), ClientID: a.ClientID, UserID: user.ID, CredentialHash: credentialDigest(user.PasswordHash), RedirectURI: a.RedirectURI, Challenge: a.Challenge, Scope: a.Scope, Resource: a.Resource, ExpiresAt: time.Now().Add(5 * time.Minute)}
	return code, s.db.Create(record).Error
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func (s *OAuthService) issue(tx *gorm.DB, grant *model.OAuthGrant, scope string) (*OAuthTokenResponse, error) {
	access, err := oauthRandom("ot_at_")
	if err != nil {
		return nil, err
	}
	refresh, err := oauthRandom("ot_rt_")
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expires := now.Add(OAuthAccessLifetime)
	if grant.ExpiresAt.Before(expires) {
		expires = grant.ExpiresAt
	}
	record := &model.OAuthToken{AccessHash: credentialDigest(access), RefreshHash: credentialDigest(refresh), GrantID: grant.ID, Scope: scope, AccessExpiresAt: expires}
	if err := tx.Create(record).Error; err != nil {
		return nil, err
	}
	return &OAuthTokenResponse{AccessToken: access, TokenType: "Bearer", ExpiresIn: int(expires.Sub(now).Seconds()), RefreshToken: refresh, Scope: scope}, nil
}

func (s *OAuthService) ExchangeCode(clientID, code, redirect, verifier, resource string) (*OAuthTokenResponse, error) {
	if !pkceVerifier.MatchString(verifier) || len(code) > 128 {
		return nil, ErrOAuthGrant
	}
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	var result *OAuthTokenResponse
	invalid := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var c model.OAuthCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "hash = ?", credentialDigest(code)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOAuthGrant
			}
			return err
		}
		if c.ClientID != clientID || c.RedirectURI != redirect || subtle.ConstantTimeCompare([]byte(c.Challenge), []byte(challenge)) != 1 || (resource != "" && resource != c.Resource) || c.Resource != s.Resource {
			return ErrOAuthGrant
		}
		if c.Used {
			invalid = true
			// A replay with all original bindings revokes the issued token family.
			return tx.Model(&model.OAuthGrant{}).Where("id = ?", c.GrantID).Update("revoked", true).Error
		}
		if !c.ExpiresAt.After(time.Now()) {
			return ErrOAuthGrant
		}
		var user model.User
		if tx.First(&user, "id = ?", c.UserID).Error != nil || credentialDigest(user.PasswordHash) != c.CredentialHash {
			return ErrOAuthGrant
		}
		grant := &model.OAuthGrant{ID: uuid.NewString(), ClientID: c.ClientID, UserID: c.UserID, CredentialHash: c.CredentialHash, Scope: c.Scope, Resource: c.Resource, ExpiresAt: time.Now().Add(OAuthGrantLifetime)}
		if err := tx.Create(grant).Error; err != nil {
			return err
		}
		if err := tx.Model(&c).Updates(map[string]any{"used": true, "grant_id": grant.ID}).Error; err != nil {
			return err
		}
		var err error
		result, err = s.issue(tx, grant, c.Scope)
		return err
	})
	if invalid && err == nil {
		err = ErrOAuthGrant
	}
	return result, err
}

func (s *OAuthService) Refresh(clientID, refresh, scope, resource string) (*OAuthTokenResponse, error) {
	if len(refresh) > 128 {
		return nil, ErrOAuthGrant
	}
	var result *OAuthTokenResponse
	invalid := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var token model.OAuthToken
		if err := tx.First(&token, "refresh_hash = ?", credentialDigest(refresh)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOAuthGrant
			}
			return err
		}
		// Lock the family first, then re-read the token: parallel refreshes cannot both succeed.
		var grant model.OAuthGrant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&grant, "id = ?", token.GrantID).Error; err != nil {
			return err
		}
		if err := tx.First(&token, "access_hash = ?", token.AccessHash).Error; err != nil {
			return err
		}
		if grant.ClientID != clientID || grant.Revoked || !grant.ExpiresAt.After(time.Now()) || grant.Resource != s.Resource || (resource != "" && resource != grant.Resource) {
			return ErrOAuthGrant
		}
		if token.RefreshUsed {
			invalid = true
			return tx.Model(&grant).Update("revoked", true).Error
		}
		var user model.User
		if tx.First(&user, "id = ?", grant.UserID).Error != nil || credentialDigest(user.PasswordHash) != grant.CredentialHash {
			return ErrOAuthGrant
		}
		if scope == "" {
			scope = token.Scope
		} else {
			var err error
			scope, err = normalizeOAuthScope(scope)
			if err != nil {
				return err
			}
			for _, field := range strings.Fields(scope) {
				if !slices.Contains(strings.Fields(token.Scope), field) {
					return ErrOAuthScope
				}
			}
		}
		if err := tx.Model(&token).Update("refresh_used", true).Error; err != nil {
			return err
		}
		var err error
		result, err = s.issue(tx, &grant, scope)
		return err
	})
	if invalid && err == nil {
		err = ErrOAuthGrant
	}
	return result, err
}

type OAuthIdentity struct {
	UserID   string
	Username string
	ClientID string
	GrantID  string
	Scope    string
}

func (s *OAuthService) ValidateAccess(token string) (*OAuthIdentity, error) {
	if !strings.HasPrefix(token, "ot_at_") || len(token) != 49 {
		return nil, ErrOAuthGrant
	}
	var record model.OAuthToken
	if s.db.First(&record, "access_hash = ? AND access_expires_at > ?", credentialDigest(token), time.Now()).Error != nil {
		return nil, ErrOAuthGrant
	}
	var grant model.OAuthGrant
	if s.db.First(&grant, "id = ? AND revoked = false AND expires_at > ?", record.GrantID, time.Now()).Error != nil || grant.Resource != s.Resource {
		return nil, ErrOAuthGrant
	}
	var user model.User
	if s.db.First(&user, "id = ?", grant.UserID).Error != nil || credentialDigest(user.PasswordHash) != grant.CredentialHash {
		return nil, ErrOAuthGrant
	}
	return &OAuthIdentity{UserID: user.ID, Username: user.Username, ClientID: grant.ClientID, GrantID: grant.ID, Scope: record.Scope}, nil
}

func (s *OAuthService) RevokeToken(clientID, token string) error {
	var record model.OAuthToken
	err := s.db.Where("access_hash = ? OR refresh_hash = ?", credentialDigest(token), credentialDigest(token)).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.db.Model(&model.OAuthGrant{}).Where("id = ? AND client_id = ?", record.GrantID, clientID).Update("revoked", true).Error
}

func (s *OAuthService) ListGrants(userID string) ([]model.OAuthGrant, error) {
	var grants []model.OAuthGrant
	err := s.db.Where("user_id = ? AND revoked = false AND expires_at > ?", userID, time.Now()).Order("created_at DESC").Find(&grants).Error
	return grants, err
}

func (s *OAuthService) RevokeGrant(userID, grantID string) error {
	return s.db.Model(&model.OAuthGrant{}).Where("id = ? AND user_id = ?", grantID, userID).Update("revoked", true).Error
}

// Keep consumed refresh tokens until their grant expires so reuse remains detectable.
func (s *OAuthService) CleanupExpired() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Where("expires_at < ? AND (used = false OR expires_at < ?)", now, now.Add(-OAuthGrantLifetime)).Delete(&model.OAuthCode{}).Error; err != nil {
			return err
		}
		expired := tx.Model(&model.OAuthGrant{}).Select("id").Where("expires_at < ? OR revoked = true", now)
		if err := tx.Where("grant_id IN (?)", expired).Delete(&model.OAuthToken{}).Error; err != nil {
			return err
		}
		return tx.Where("expires_at < ?", now).Delete(&model.OAuthGrant{}).Error
	})
}

// RequiredOAuthScope is a deny-by-default policy shared by REST and MCP.
// Account management, credential minting and whole-database backups are never delegated.
func RequiredOAuthScope(method, path string) string {
	read := method == "GET" || method == "HEAD"
	if path == "/api/v1/auth/profile" && read {
		return "data:read"
	}
	if !strings.HasPrefix(path, "/api/v1/") || strings.ContainsAny(path, "\\%?#") || strings.Contains(path, "..") {
		return ""
	}
	segment := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")[0]
	if segment == "secrets" || segment == "secret-audit-logs" {
		if read {
			return "secrets:read"
		}
		return "secrets:write"
	}
	if !slices.Contains([]string{"units", "projects", "todos", "todo-groups", "notes", "note-groups", "note-drafts", "notifications", "schedules", "wallets", "budget", "transactions"}, segment) {
		return ""
	}
	if read {
		return "data:read"
	}
	return "data:write"
}

func OAuthAllows(scope, method, path string) bool {
	required := RequiredOAuthScope(method, path)
	return required != "" && slices.Contains(strings.Fields(scope), required)
}

func (s *OAuthService) Challenge(oauthError, scope string) string {
	header := `Bearer resource_metadata="` + s.Issuer + `/.well-known/oauth-protected-resource", scope="` + scope + `"`
	if oauthError != "" {
		header += `, error="` + oauthError + `"`
	}
	return header
}
