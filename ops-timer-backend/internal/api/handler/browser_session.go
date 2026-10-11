package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
	"time"
)

func (h *AuthHandler) setBrowserCookie(c *gin.Context, token string, remember bool, expires time.Time) {
	cookie := &http.Cookie{Name: service.BrowserCookieName, Value: token, Path: "/api/v1", HttpOnly: true, Secure: h.authService.SecureBrowserCookie(), SameSite: http.SameSiteStrictMode}
	if remember {
		cookie.MaxAge = int(service.RememberDuration.Seconds())
		cookie.Expires = expires
	}
	if token == "" {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(1, 0)
	}
	http.SetCookie(c.Writer, cookie)
}

func (h *AuthHandler) BrowserLogin(c *gin.Context) {
	var req struct {
		dto.LoginRequest
		RememberMe bool `json:"remember_me"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}
	token, user, expires, err := h.authService.BrowserLogin(&req.LoginRequest, req.RememberMe)
	if err != nil {
		if err == service.ErrAccountLocked {
			response.TooManyRequests(c, err.Error())
		} else if err == service.ErrInvalidCredentials {
			response.Unauthorized(c, response.CodeCredentialError, err.Error())
		} else {
			response.InternalError(c, "登录暂时不可用")
		}
		return
	}
	if previous, err := c.Cookie(service.BrowserCookieName); err == nil {
		if err := h.authService.RevokeBrowserSession(previous); err != nil {
			_ = h.authService.RevokeBrowserSession(token)
			response.InternalError(c, "无法替换旧会话，请重试")
			return
		}
	}
	h.setBrowserCookie(c, token, req.RememberMe, expires)
	response.Success(c, gin.H{"user": user, "expires_at": expires})
}

// Public so even an expired session can be explicitly cleared.
func (h *AuthHandler) BrowserLogout(c *gin.Context) {
	if token, err := c.Cookie(service.BrowserCookieName); err == nil {
		if err := h.authService.RevokeBrowserSession(token); err != nil {
			response.InternalError(c, "退出失败，请重试")
			return
		}
	}
	h.setBrowserCookie(c, "", false, time.Time{})
	response.Success(c, nil)
}
