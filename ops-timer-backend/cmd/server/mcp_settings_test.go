package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/api/router"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/pkg/email"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"
)

func TestMCPAdminSettingsLiveConfiguration(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	secret := strings.Repeat("s", 32)
	jwt := auth.NewJWTManager(secret, 24, nil)
	users := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(users, repository.NewBrowserSessionRepository(db), repository.NewLoginAttemptRepository(db), jwt, &config.AuthConfig{LoginLockAttempts: 5, LoginLockMinutes: 15})
	if err := authSvc.EnsureAdminExists("original", "SettingsTestPassword123"); err != nil {
		t.Fatal(err)
	}
	admin, _ := users.FindByUsername("original")
	// Model an upgrade: the existing account has no role and its name is mutable.
	admin.IsAdmin = false
	if err := users.Update(admin); err != nil {
		t.Fatal(err)
	}
	ordinary := &model.User{ID: uuid.NewString(), Username: "admin", PasswordHash: admin.PasswordHash, DisplayName: "Member", APIToken: "ordinary-token"}
	if err := users.Create(ordinary); err != nil {
		t.Fatal(err)
	}
	if err := authSvc.EnsureAdminExists("admin", "ignored"); err != nil {
		t.Fatal(err)
	}
	admin, _ = users.FindByID(admin.ID)
	ordinary, _ = users.FindByID(ordinary.ID)
	if !admin.IsAdmin || ordinary.IsAdmin {
		t.Fatal("upgrade assigned admin role using username rather than original identity")
	}
	settings := service.NewMCPSettingsService(db, authSvc, secret)
	cfg := &config.Config{MCP: config.MCPConfig{Enabled: true, Path: "/mcp", ServerName: "old-name", ServerVersion: "2.0.0", TimeoutSeconds: 30}, OAuth: config.OAuthConfig{DynamicRegistration: true}}
	if err := settings.Initialize(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	// A second replica resolves the same database settings without restart.
	replica := service.NewMCPSettingsService(db, authSvc, secret)
	backend := httptest.NewUnstartedServer(nil)
	defer backend.Close()
	r := router.NewRouter(&router.RouterConfig{
		AuthHandler: handler.NewAuthHandler(authSvc, email.NewService(&config.SMTPConfig{})),
		AuthService: authSvc, JWTManager: jwt, Logger: zap.NewNop(), MCPSettings: settings,
		MCPHandler: handler.NewMCPHandler(handler.MCPHandlerConfig{AuthService: authSvc, BaseURL: "http://" + backend.Listener.Addr().String()}),
	}).Setup()
	backend.Config.Handler = r
	backend.Start()
	adminJWT, _ := jwt.GenerateToken(admin.ID, admin.Username)
	memberJWT, _ := jwt.GenerateToken(ordinary.ID, ordinary.Username)
	call := func(method, path, body, credential string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "https://tasks.example"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if strings.HasPrefix(credential, "api:") {
			req.Header.Set("X-API-Token", strings.TrimPrefix(credential, "api:"))
		} else if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s got %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	const adminPath = "/api/v1/admin/mcp-settings"
	call("GET", adminPath, "", "", 401)
	call("GET", adminPath, "", "api:"+admin.APIToken, 403)
	call("GET", adminPath, "", memberJWT, 403)
	call("GET", adminPath, "", adminJWT, 200)
	call("PUT", adminPath, `{}`, adminJWT, 400)
	call("PUT", adminPath, `null`, adminJWT, 400)
	call("PUT", adminPath, `{}`, memberJWT, 403)
	// Real browser sessions work; cookies without the CSRF custom header do not.
	cookieToken, _, _, err := authSvc.BrowserLogin(&dto.LoginRequest{Username: admin.Username, Password: "SettingsTestPassword123"}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, trusted := range []bool{false, true} {
		req := httptest.NewRequest("GET", "https://tasks.example"+adminPath, nil)
		req.AddCookie(&http.Cookie{Name: service.BrowserCookieName, Value: cookieToken})
		if trusted {
			req.Header.Set("X-Taskmgr-Browser", "1")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		want := 403
		if trusted {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("browser admin status: %d want %d", w.Code, want)
		}
	}
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	call("POST", "/mcp", initialize, "api:"+admin.APIToken, 200)
	call("POST", "/mcp", initialize, admin.APIToken, 200)
	call("POST", "/mcp", initialize, adminJWT, 401)
	// API tokens, even belonging to an administrator, cannot use the MCP proxy
	// to read or alter administrator settings.
	proxy := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"backend_request","arguments":{"method":"GET","path":"/api/v1/admin/mcp-settings"}}}`
	blocked := call("POST", "/mcp", proxy, "api:"+admin.APIToken, 200)
	if !strings.Contains(blocked.Body.String(), `"isError":true`) {
		t.Fatal("MCP proxy accessed admin settings", blocked.Body.String())
	}
	current, _ := settings.Get(ctx)
	current.Path, current.ExternalURL, current.ServerName = "/agents/mcp", "https://tasks.example", "new-name"
	save := func() {
		t.Helper()
		body, _ := json.Marshal(current)
		call("PUT", adminPath, string(body), adminJWT, 200)
	}
	save()
	call("GET", "/mcp/config", "", "", 404)
	call("POST", "/mcp", initialize, "api:"+admin.APIToken, 404)
	info := call("GET", current.Path+"/config", "", "", 200)
	if !strings.Contains(info.Body.String(), "https://tasks.example/agents/mcp") {
		t.Fatal(info.Body.String())
	}
	result := call("POST", current.Path, initialize, "api:"+admin.APIToken, 200)
	if !strings.Contains(result.Body.String(), `"name":"new-name"`) {
		t.Fatal("server info did not update", result.Body.String())
	}
	// Stale environment values must not overwrite admin settings on restart.
	cfg.MCP.Path, cfg.OAuth.Issuer = "/stale", "invalid-legacy-value"
	if err := replica.Initialize(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	persisted, _ := replica.Get(ctx)
	if persisted.Path != current.Path || persisted.UpdatedBy != admin.ID {
		t.Fatal("restart lost settings or audit identity")
	}
	current.APITokenEnabled, current.OAuthEnabled = false, true
	current.OAuthIssuer, current.OAuthDynamicRegistration = "https://tasks.example", false
	save()
	metadata := call("GET", "/.well-known/oauth-authorization-server", "", "", 200)
	if strings.Contains(metadata.Body.String(), "registration_endpoint") {
		t.Fatal("disabled dynamic registration still advertised")
	}
	call("GET", "/.well-known/oauth-protected-resource"+current.Path, "", "", 200)
	call("POST", "/oauth/register", `{}`, "", 403)
	unauth := call("POST", current.Path, initialize, "api:"+admin.APIToken, 401)
	if !strings.Contains(unauth.Header().Get("WWW-Authenticate"), "https://tasks.example/.well-known/oauth-protected-resource") {
		t.Fatal("missing dynamic OAuth challenge")
	}
	call("POST", current.Path, initialize, admin.APIToken, 401)
	call("GET", "/api/v1/auth/profile", "", "api:"+admin.APIToken, 200)
	info = call("GET", current.Path+"/config", "", "", 200)
	if strings.Contains(info.Body.String(), "X-API-Token") {
		t.Fatal("OAuth config contains legacy token instructions")
	}
	oauth, err := replica.OAuth(current)
	if err != nil {
		t.Fatal(err)
	}
	client, err := oauth.RegisterClient("Agent", []string{"http://127.0.0.1:43123/callback"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("v", 43)
	hash := sha256.Sum256([]byte(verifier))
	a := &service.OAuthAuthorization{ClientID: client.ID, RedirectURI: client.RedirectURIs[0], ResponseType: "code", ChallengeMethod: "S256", Challenge: base64.RawURLEncoding.EncodeToString(hash[:]), Scope: "data:read"}
	code, err := oauth.Authorize(a, admin.Username, "SettingsTestPassword123")
	if err != nil {
		t.Fatal(err)
	}
	token, err := oauth.ExchangeCode(client.ID, code, client.RedirectURIs[0], verifier, oauth.Resource)
	if err != nil {
		t.Fatal(err)
	}
	call("POST", current.Path, initialize, token.AccessToken, 200)
	call("GET", "/api/v1/auth/profile", "", token.AccessToken, 200)
	call("GET", adminPath, "", token.AccessToken, 403)
	// Disabling OAuth immediately rejects existing credentials at both entry points.
	current.OAuthEnabled, current.APITokenEnabled = false, true
	save()
	call("POST", current.Path, initialize, token.AccessToken, 401)
	call("GET", "/api/v1/auth/profile", "", token.AccessToken, 401)
	call("GET", "/.well-known/oauth-authorization-server", "", "", 404)
	call("POST", "/oauth/token", `{}`, "", 404)
	current.Enabled = false
	save()
	call("POST", current.Path, initialize, "api:"+admin.APIToken, 404)
	call("GET", current.Path+"/config", "", "", 404)
	call("GET", adminPath, "", adminJWT, 200)
	// Reading settings fails closed, without falling back to environment policy.
	if err := db.Delete(&model.MCPSettings{}, 1).Error; err != nil {
		t.Fatal(err)
	}
	call("POST", current.Path, initialize, "api:"+admin.APIToken, 503)
	call("GET", "/health", "", "", 200)
}
