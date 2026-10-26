package middleware

import (
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtManager *auth.JWTManager, authService *service.AuthService, allowedOrigins []string, oauthServices ...*service.OAuthService) gin.HandlerFunc {
	var oauth *service.OAuthService
	if len(oauthServices) > 0 {
		oauth = oauthServices[0]
	}
	return func(c *gin.Context) {
		// Each request uses one snapshot when administrators update settings.
		oauth := oauth
		if current, exists := c.Get("oauth_service"); exists {
			oauth, _ = current.(*service.OAuthService)
		}
		// Delegated OAuth credentials are checked before unrestricted legacy credentials.
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				if strings.HasPrefix(parts[1], "ot_") {
					if oauth != nil {
						identity, err := oauth.ValidateAccess(parts[1])
						if err == nil {
							if !service.OAuthAllows(identity.Scope, c.Request.Method, c.Request.URL.Path) {
								c.Header("WWW-Authenticate", oauth.Challenge("insufficient_scope", service.RequiredOAuthScope(c.Request.Method, c.Request.URL.Path)))
								c.AbortWithStatusJSON(403, gin.H{"error": "insufficient_scope"})
								return
							}
							c.Set("user_id", identity.UserID)
							c.Set("auth_method", "oauth")
							c.Set("username", identity.Username)
							c.Set("oauth_client_id", identity.ClientID)
							c.Set("oauth_grant_id", identity.GrantID)
							c.Header("Cache-Control", "no-store")
							c.Next()
							return
						}
						c.Header("WWW-Authenticate", oauth.Challenge("invalid_token", "data:read"))
					}
					c.AbortWithStatusJSON(401, gin.H{"error": "invalid_token"})
					return
				}
				claims, err := jwtManager.ValidateToken(parts[1])
				if err == nil {
					c.Set("auth_method", "jwt")
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
				c.Set("auth_method", "api_token")
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
				c.Set("auth_method", "cookie")
				c.Set("user_id", user.ID)
				c.Set("username", user.Username)
				c.Header("Cache-Control", "no-store")
				c.Next()
				return
			}
		}

		if oauth != nil {
			c.Header("WWW-Authenticate", oauth.Challenge("", "data:read"))
		}
		response.Unauthorized(c, response.CodeTokenMissing, "认证信息缺失或无效")
		c.Abort()
	}
}
