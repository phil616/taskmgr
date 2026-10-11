package middleware

import (
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtManager *auth.JWTManager, authService *service.AuthService, allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try JWT first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, err := jwtManager.ValidateToken(parts[1])
				if err == nil {
					c.Set("user_id", claims.UserID)
					c.Set("username", claims.Username)
					c.Set("token", parts[1])
					c.Next()
					return
				}
			}
		}

		// Try API Token
		apiToken := c.GetHeader("X-API-Token")
		if apiToken != "" {
			user, err := authService.FindByAPIToken(apiToken)
			if err == nil {
				c.Set("user_id", user.ID)
				c.Set("username", user.Username)
				c.Next()
				return
			}
		}

		if token, err := c.Cookie(service.BrowserCookieName); err == nil {
			if !browserRequestAllowed(c, allowedOrigins) {
				c.AbortWithStatus(403)
				return
			}
			if user, err := authService.ValidateBrowserSession(token); err == nil {
				c.Set("user_id", user.ID)
				c.Set("username", user.Username)
				c.Header("Cache-Control", "no-store")
				c.Next()
				return
			}
		}

		response.Unauthorized(c, response.CodeTokenMissing, "认证信息缺失或无效")
		c.Abort()
	}
}
