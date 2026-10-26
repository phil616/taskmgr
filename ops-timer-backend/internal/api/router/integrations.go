package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/service"
)

func (r *Router) dynamicOAuth(endpoint func(*handler.OAuthHandler, *gin.Context)) gin.HandlerFunc {
	return func(c *gin.Context) {
		current, _ := c.Get("oauth_service")
		oauth, _ := current.(*service.OAuthService)
		if oauth == nil {
			c.AbortWithStatus(404)
			return
		}
		endpoint(handler.NewOAuthHandler(oauth), c)
	}
}

// Resolve from PostgreSQL for every request: no stale per-process configuration
// or shared mutable OAuth state, including when running multiple replicas.
func (r *Router) integrationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		settings, err := r.mcpSettings.Get(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(503, gin.H{"message": "暂时无法读取服务设置"})
			return
		}
		oauth, err := r.mcpSettings.OAuth(settings)
		if err != nil {
			c.AbortWithStatusJSON(503, gin.H{"message": "授权服务设置无效"})
			return
		}
		c.Set("oauth_service", oauth)
		path := c.Request.URL.Path
		if path == "/.well-known/oauth-protected-resource"+settings.Path {
			if oauth == nil {
				c.AbortWithStatus(404)
				return
			}
			if c.Request.Method != http.MethodGet {
				c.AbortWithStatus(405)
				return
			}
			handler.NewOAuthHandler(oauth).ResourceMetadata(c)
			c.Abort()
			return
		}
		if path != settings.Path && path != settings.Path+"/config" {
			c.Next()
			return
		}
		c.Header("Cache-Control", "no-store")
		if !settings.Enabled || r.mcpHandler == nil {
			c.AbortWithStatus(404)
			return
		}
		mcp := r.mcpHandler.Configured(settings, oauth)
		switch {
		case path == settings.Path+"/config" && c.Request.Method == http.MethodGet:
			mcp.GetConfig(c)
		case path == settings.Path && c.Request.Method == http.MethodGet:
			mcp.GetInfo(c)
		case path == settings.Path && c.Request.Method == http.MethodPost:
			mcp.Handle(c)
		default:
			c.Status(405)
		}
		c.Abort()
	}
}
