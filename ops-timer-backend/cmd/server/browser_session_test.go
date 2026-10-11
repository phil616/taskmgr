package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/api/middleware"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"
)

func TestBrowserSessions(t *testing.T) {
	db := testPostgres(t)
	jwt := auth.NewJWTManager(strings.Repeat("s", 32), 24, nil)
	svc := service.NewAuthService(repository.NewUserRepository(db), repository.NewBrowserSessionRepository(db), repository.NewLoginAttemptRepository(db), jwt, &config.AuthConfig{CookieSecure: true, LoginLockAttempts: 5, LoginLockMinutes: 15})
	if err := svc.EnsureAdminExists("admin", "TestPassword123"); err != nil {
		t.Fatal(err)
	}
	h := handler.NewAuthHandler(svc, nil)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CORSMiddleware([]string{"https://app.example.com"}))
	browser := r.Group("/api/v1/auth/browser", middleware.BrowserRequestGuard([]string{"https://app.example.com"}))
	browser.POST("/login", h.BrowserLogin)
	browser.POST("/logout", h.BrowserLogout)
	secured := r.Group("/api/v1", middleware.AuthMiddleware(jwt, svc, []string{"https://app.example.com"}))
	secured.GET("/auth/profile", h.GetProfile)
	secured.PUT("/auth/password", h.ChangePassword)
	request := func(method, path, body string, cookie *http.Cookie, origin, header string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "https://api.example.com"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.Header.Set("X-Taskmgr-Browser", header)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s status %d, expected %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w
	}
	login := func(remember bool, old *http.Cookie) *http.Cookie {
		t.Helper()
		body := `{"username":"admin","password":"TestPassword123","remember_me":false}`
		if remember {
			body = strings.Replace(body, "false", "true", 1)
		}
		w := request("POST", "/api/v1/auth/browser/login", body, old, "https://app.example.com", "1", 200)
		cookies := w.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("cookies: %v", cookies)
		}
		cookie := cookies[0]
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api/v1" || cookie.Domain != "" {
			t.Fatalf("unsafe cookie: %+v", cookie)
		}
		if remember {
			if cookie.MaxAge != 14*24*3600 || time.Until(cookie.Expires) < 14*24*time.Hour-time.Minute {
				t.Fatalf("wrong remember expiry: %+v", cookie)
			}
		} else if cookie.MaxAge != 0 || !cookie.Expires.IsZero() {
			t.Fatal("unchecked login persisted cookie")
		}
		var bodyJSON map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &bodyJSON); err != nil {
			t.Fatal(err)
		}
		data := bodyJSON["data"].(map[string]any)
		if _, exists := data["token"]; exists {
			t.Fatal("exposed browser credential to javascript")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("login response cacheable")
		}
		return cookie
	}
	request("POST", "/api/v1/auth/browser/login", `{}`, nil, "https://evil.example.com", "1", 403)
	request("POST", "/api/v1/auth/browser/login", `{}`, nil, "https://app.example.com", "", 403)
	request("POST", "/api/v1/auth/browser/login", `{"username":"admin","password":"wrong123"}`, nil, "https://app.example.com", "1", 401)
	remembered := login(true, nil)
	var stored model.BrowserSession
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.TokenHash == remembered.Value || len(stored.TokenHash) != 64 {
		t.Fatal("session not hashed")
	}
	request("GET", "/api/v1/auth/profile", "", remembered, "https://app.example.com", "1", 200)
	request("GET", "/api/v1/auth/profile", "", remembered, "https://evil.example.com", "1", 403)
	request("GET", "/api/v1/auth/profile", "", remembered, "https://app.example.com", "", 403)
	session := login(false, remembered)
	request("GET", "/api/v1/auth/profile", "", remembered, "https://app.example.com", "1", 401)
	w := request("POST", "/api/v1/auth/browser/logout", "", session, "https://app.example.com", "1", 200)
	if w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not clear cookie")
	}
	request("GET", "/api/v1/auth/profile", "", session, "https://app.example.com", "1", 401)
	expired := login(true, nil)
	if err := db.Model(&model.BrowserSession{}).Where("expires_at > ?", time.Now()).Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/v1/auth/profile", "", expired, "https://app.example.com", "1", 401)
	first := login(true, nil)
	second := login(true, nil)
	request("PUT", "/api/v1/auth/password", `{"old_password":"TestPassword123","new_password":"NewPassword456"}`, first, "https://app.example.com", "1", 200)
	request("GET", "/api/v1/auth/profile", "", first, "https://app.example.com", "1", 401)
	request("GET", "/api/v1/auth/profile", "", second, "https://app.example.com", "1", 401)
}
