package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/api/middleware"
	"ops-timer-backend/internal/api/router"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"
)

func TestOAuthFlow(t *testing.T) {
	db := testPostgres(t)
	// Use several independent connections with the isolated schema set at connect
	// time: the concurrent refresh test must exercise PostgreSQL row locks.
	var schema string
	if err := db.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
		t.Fatal(err)
	}
	dsn := os.Getenv("TASK_MANAGER_TEST_DATABASE_DSN")
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		dsn += " search_path=" + schema
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(4)
	t.Cleanup(func() { pool.Close() })
	jwt := auth.NewJWTManager(strings.Repeat("s", 32), 24, nil)
	authSvc := service.NewAuthService(repository.NewUserRepository(db), repository.NewBrowserSessionRepository(db), repository.NewLoginAttemptRepository(db), jwt, &config.AuthConfig{LoginLockAttempts: 5, LoginLockMinutes: 15})
	if err := authSvc.EnsureAdminExists("admin", "OAuthTestPassword123"); err != nil {
		t.Fatal(err)
	}
	svc, err := service.NewOAuthService(db, authSvc, "https://tasks.example", "/mcp", strings.Repeat("s", 32), true)
	if err != nil {
		t.Fatal(err)
	}
	oh := handler.NewOAuthHandler(svc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/oauth/authorize", oh.Authorize)
	r.POST("/oauth/authorize", oh.Authorize)
	r.POST("/oauth/register", oh.Register)
	r.POST("/oauth/token", oh.Token)
	r.POST("/oauth/revoke", oh.Revoke)
	r.GET("/.well-known/oauth-authorization-server", oh.Metadata)
	r.GET("/.well-known/oauth-protected-resource", oh.ResourceMetadata)
	protected := r.Group("/api/v1", middleware.AuthMiddleware(jwt, authSvc, nil, svc))
	for _, path := range []string{"/todos", "/secrets/key/value", "/auth/token", "/backup/export"} {
		protected.GET(path, func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	}
	protected.POST("/todos", func(c *gin.Context) { c.Status(201) })
	protected.GET("/oauth/grants", oh.ListGrants)
	protected.DELETE("/oauth/grants/:id", oh.RevokeGrant)
	backend := httptest.NewServer(r)
	defer backend.Close()
	mcp := handler.NewMCPHandler(handler.MCPHandlerConfig{AuthService: authSvc, OAuthService: svc, BaseURL: backend.URL})
	r.POST("/mcp", mcp.Handle)
	r.GET("/mcp/config", mcp.GetConfig)
	request := func(method, path, body, contentType, token string, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "https://tasks.example"+path, strings.NewReader(body))
		req.Header.Set("Accept", "application/json, text/event-stream")
		if method == http.MethodPost && path == "/oauth/authorize" {
			req.Header.Set("Origin", "https://tasks.example")
			req.Header.Set("Sec-Fetch-Site", "same-origin")
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s status %d, want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	request("GET", "/.well-known/oauth-authorization-server", "", "", "", nil, 200)
	metadata := request("GET", "/.well-known/oauth-protected-resource", "", "", "", nil, 200)
	if !strings.Contains(metadata.Body.String(), `"resource":"https://tasks.example/mcp"`) {
		t.Fatal(metadata.Body.String())
	}
	unauthorized := request("POST", "/mcp", `{}`, "application/json", "", nil, 401)
	if !strings.Contains(unauthorized.Header().Get("WWW-Authenticate"), "resource_metadata=") {
		t.Fatal("missing OAuth challenge")
	}
	registration := request("POST", "/oauth/register", `{"client_name":"Test Agent <script>","redirect_uris":["http://127.0.0.1:43123/callback?existing=1"],"token_endpoint_auth_method":"none"}`, "application/json", "", nil, 201)
	var registered struct {
		ID string `json:"client_id"`
	}
	if err := json.Unmarshal(registration.Body.Bytes(), &registered); err != nil || registered.ID == "" {
		t.Fatal("no client ID", err)
	}
	clientID, redirect := registered.ID, "http://127.0.0.1:43123/callback?existing=1"
	verifier := strings.Repeat("v", 43)
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	formPattern := regexp.MustCompile(`name="request" value="([^"]+)"`)
	authorization := func(scope string) (string, *http.Cookie, string) {
		t.Helper()
		q := url.Values{"client_id": {clientID}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {scope}, "state": {"csrf-state"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}, "resource": {svc.Resource}}
		page := request("GET", "/oauth/authorize?"+q.Encode(), "", "", "", nil, 200)
		if page.Header().Get("Referrer-Policy") != "same-origin" {
			t.Fatal("consent page would cause browsers to POST Origin: null")
		}
		if strings.Contains(page.Body.String(), "<script>") || !strings.Contains(page.Body.String(), "&lt;script&gt;") {
			t.Fatal("unescaped client name")
		}
		form := formPattern.FindStringSubmatch(page.Body.String())
		if len(form) != 2 {
			t.Fatal("missing consent form")
		}
		cookie := page.Result().Cookies()[0]
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/oauth/authorize" {
			t.Fatalf("unsafe csrf cookie: %+v", cookie)
		}
		values := url.Values{"request": {html.UnescapeString(form[1])}, "username": {"admin"}, "password": {"OAuthTestPassword123"}, "decision": {"approve"}}
		return values.Encode(), cookie, q.Encode()
	}
	getCode := func(scope string) string {
		t.Helper()
		form, cookie, _ := authorization(scope)
		response := request("POST", "/oauth/authorize", form, "application/x-www-form-urlencoded", "", cookie, 303)
		location, err := url.Parse(response.Header().Get("Location"))
		if err != nil || location.Query().Get("state") != "csrf-state" || location.Query().Get("iss") != svc.Issuer || location.Query().Get("existing") != "1" {
			t.Fatal("bad redirect", location, err)
		}
		return location.Query().Get("code")
	}
	exchange := func(code, verify string, status int) service.OAuthTokenResponse {
		t.Helper()
		form := url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "code": {code}, "code_verifier": {verify}, "resource": {svc.Resource}}
		response := request("POST", "/oauth/token", form.Encode(), "application/x-www-form-urlencoded", "", nil, status)
		var tokens service.OAuthTokenResponse
		if err := json.Unmarshal(response.Body.Bytes(), &tokens); err != nil {
			t.Fatal(err)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("token response cacheable")
		}
		return tokens
	}
	form, cookie, query := authorization("data:read")
	request("POST", "/oauth/authorize", form, "application/x-www-form-urlencoded", "", nil, 400)
	request("POST", "/oauth/authorize", strings.Replace(form, "request=", "request=bad", 1), "application/x-www-form-urlencoded", "", cookie, 400)
	deny, _ := url.ParseQuery(form)
	for _, headers := range []struct{ origin, fetchSite string }{
		{"https://evil.example", "cross-site"},
		{"null", "same-origin"},
		{"https://tasks.example", "cross-site"},
	} {
		originRequest := httptest.NewRequest("POST", "https://tasks.example/oauth/authorize", strings.NewReader(form))
		originRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		originRequest.Header.Set("Origin", headers.origin)
		originRequest.Header.Set("Sec-Fetch-Site", headers.fetchSite)
		originRequest.AddCookie(cookie)
		originResponse := httptest.NewRecorder()
		r.ServeHTTP(originResponse, originRequest)
		if originResponse.Code != 403 {
			t.Fatalf("accepted untrusted form origin: %+v", headers)
		}
	}
	deny.Set("decision", "deny")
	denied := request("POST", "/oauth/authorize", deny.Encode(), "application/x-www-form-urlencoded", "", cookie, 303)
	if !strings.Contains(denied.Header().Get("Location"), "error=access_denied") {
		t.Fatal("denial did not return error")
	}
	request("GET", "/oauth/authorize?"+strings.Replace(query, "S256", "plain", 1), "", "", "", nil, 400)
	request("GET", "/oauth/authorize?"+query+"&client_id=duplicate", "", "", "", nil, 400)
	invalidTarget, _ := url.ParseQuery(query)
	invalidTarget.Set("resource", "https://evil.example/mcp")
	request("GET", "/oauth/authorize?"+invalidTarget.Encode(), "", "", "", nil, 400)
	invalidTarget.Set("resource", svc.Resource)
	invalidTarget.Set("redirect_uri", "https://evil.example/callback")
	request("GET", "/oauth/authorize?"+invalidTarget.Encode(), "", "", "", nil, 400)
	code := getCode("data:read")
	exchange(code, "", 400)
	exchange(code, strings.Repeat("x", 43), 400)
	tokens := exchange(code, verifier, 200)
	if tokens.ExpiresIn != 900 || tokens.TokenType != "Bearer" || tokens.RefreshToken == "" {
		t.Fatal("invalid token response")
	}
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 200)
	request("POST", "/api/v1/todos", "{}", "application/json", tokens.AccessToken, nil, 403)
	request("GET", "/api/v1/secrets/key/value", "", "", tokens.AccessToken, nil, 403)
	request("GET", "/api/v1/auth/token", "", "", tokens.AccessToken, nil, 403)
	request("GET", "/api/v1/backup/export", "", "", tokens.AccessToken, nil, 403)
	list := request("POST", "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "application/json", tokens.AccessToken, nil, 200)
	if strings.Contains(list.Body.String(), `"name":"auth_get_token"`) || strings.Contains(list.Body.String(), `"name":"todo_create"`) || !strings.Contains(list.Body.String(), `"name":"todo_list"`) {
		t.Fatal("tool list did not respect scopes")
	}
	call := request("POST", "/mcp", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"todo_list","arguments":{}}}`, "application/json", tokens.AccessToken, nil, 200)
	if strings.Contains(call.Body.String(), `"isError":true`) || !strings.Contains(call.Body.String(), `"ok":true`) {
		t.Fatal(call.Body.String())
	}
	var user model.User
	if err := db.First(&user).Error; err != nil {
		t.Fatal(err)
	}
	attack, _ := json.Marshal(gin.H{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": gin.H{"name": "backend_request", "arguments": gin.H{"method": "GET", "path": "/api/v1/auth/token", "headers": gin.H{"X-API-Token": user.APIToken}}}})
	blocked := request("POST", "/mcp", string(attack), "application/json", tokens.AccessToken, nil, 200)
	if !strings.Contains(blocked.Body.String(), "error") || strings.Contains(blocked.Body.String(), user.APIToken) {
		t.Fatal("proxy bypass")
	}
	var stored model.OAuthToken
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.AccessHash == tokens.AccessToken || stored.RefreshHash == tokens.RefreshToken || len(stored.AccessHash) != 64 {
		t.Fatal("stored plaintext token")
	}
	rotated, err := svc.Refresh(clientID, tokens.RefreshToken, "", svc.Resource)
	if err != nil || rotated.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh did not rotate", err)
	}
	if _, err := svc.Refresh(clientID, rotated.RefreshToken, "data:write", svc.Resource); err != service.ErrOAuthScope {
		t.Fatal("refresh escalated scope", err)
	}
	if _, err := svc.Refresh("wrong-client", tokens.RefreshToken, "", svc.Resource); err == nil {
		t.Fatal("refresh accepted wrong client")
	}
	if _, err := svc.Refresh(clientID, tokens.RefreshToken, "", svc.Resource); err != service.ErrOAuthGrant {
		t.Fatal("refresh replay accepted", err)
	}
	request("GET", "/api/v1/todos", "", "", rotated.AccessToken, nil, 401)
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 401)
	code = getCode("data:read data:write")
	tokens = exchange(code, verifier, 200)
	request("POST", "/api/v1/todos", "{}", "application/json", tokens.AccessToken, nil, 201)
	exchange(code, verifier, 400)
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 401)
	tokens = exchange(getCode("secrets:read"), verifier, 200)
	request("GET", "/api/v1/secrets/key/value", "", "", tokens.AccessToken, nil, 200)
	// Revocation is idempotent and bound to the issuing client.
	wrongRevoke := url.Values{"client_id": {"wrong"}, "token": {tokens.AccessToken}}
	request("POST", "/oauth/revoke", wrongRevoke.Encode(), "application/x-www-form-urlencoded", "", nil, 200)
	request("GET", "/api/v1/secrets/key/value", "", "", tokens.AccessToken, nil, 200)
	wrongRevoke.Set("client_id", clientID)
	request("POST", "/oauth/revoke", wrongRevoke.Encode(), "application/x-www-form-urlencoded", "", nil, 200)
	request("GET", "/api/v1/secrets/key/value", "", "", tokens.AccessToken, nil, 401)
	request("POST", "/oauth/revoke", wrongRevoke.Encode(), "application/x-www-form-urlencoded", "", nil, 200)
	// The account owner can inspect/revoke grants using a legacy credential.
	tokens = exchange(getCode("data:read"), verifier, 200)
	identity, err := svc.ValidateAccess(tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	ownerJWT, err := jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/v1/oauth/grants", "", "", tokens.AccessToken, nil, 403)
	request("GET", "/api/v1/oauth/grants", "", "", ownerJWT, nil, 200)
	request("DELETE", "/api/v1/oauth/grants/"+identity.GrantID, "", "", ownerJWT, nil, 204)
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 401)
	// Concurrent refreshes yield at most one response; replay revokes the family.
	tokens = exchange(getCode("data:read"), verifier, 200)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Refresh(clientID, tokens.RefreshToken, "", svc.Resource)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success, failed := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else {
			failed++
		}
	}
	if success != 1 || failed != 1 {
		t.Fatalf("concurrent refresh: %d success, %d failed", success, failed)
	}
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 401)
	// Expiry applies to the grant as well as the individual access token.
	tokens = exchange(getCode("data:read data:write"), verifier, 200)
	narrowed, err := svc.Refresh(clientID, tokens.RefreshToken, "data:read", svc.Resource)
	if err != nil {
		t.Fatal(err)
	}
	request("POST", "/api/v1/todos", "{}", "application/json", narrowed.AccessToken, nil, 403)
	if _, err := svc.Refresh(clientID, narrowed.RefreshToken, "data:write", svc.Resource); err != service.ErrOAuthScope {
		t.Fatal("downscoped refresh regained privileges", err)
	}
	grantIdentity, err := svc.ValidateAccess(narrowed.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.OAuthGrant{}).Where("id = ?", grantIdentity.GrantID).Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/v1/todos", "", "", narrowed.AccessToken, nil, 401)
	if _, err := svc.Refresh(clientID, narrowed.RefreshToken, "", svc.Resource); err == nil {
		t.Fatal("refreshed expired grant")
	}
	// Explicit expiry and password change invalidate delegated credentials.
	tokens = exchange(getCode("data:read"), verifier, 200)
	if err := db.Model(&model.OAuthToken{}).Where("grant_id <> ?", "none").Update("access_expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 401)
	tokens = exchange(getCode("data:read"), verifier, 200)
	if err := authSvc.ChangePassword(user.ID, &dto.ChangePasswordRequest{OldPassword: "OAuthTestPassword123", NewPassword: "NewPassword456"}); err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/v1/todos", "", "", tokens.AccessToken, nil, 401)
	if _, err := svc.Refresh(clientID, tokens.RefreshToken, "", svc.Resource); err == nil {
		t.Fatal("refresh survived password change")
	}
	if err := svc.CleanupExpired(); err != nil {
		t.Fatal(err)
	}
	// Exercise production route registration, including discovery aliases.
	production := router.NewRouter(&router.RouterConfig{AuthHandler: handler.NewAuthHandler(authSvc, nil), OAuthHandler: oh, OAuthService: svc, AuthService: authSvc, JWTManager: jwt, Logger: zap.NewNop(), MCPHandler: mcp}).Setup()
	for _, path := range []string{"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource/mcp", "/mcp/config", "/mcp/info"} {
		w := httptest.NewRecorder()
		production.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("production route %s: %d", path, w.Code)
		}
	}
}
