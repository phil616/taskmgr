package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/service"
)

func (r *Router) dynamicOAuth(endpoint func(*handler.OAuthHandler, *gin.Context)) gin.HandlerFunc {
	return func(c *gin.Context) {
		oauth, ok := r.resolveOAuth(c)
		if !ok {
			return
		}
		if oauth == nil {
			c.AbortWithStatus(404)
			return
		}
		endpoint(handler.NewOAuthHandler(oauth), c)
	}
}

// Resolve dynamic MCP paths only after registered routes have been considered.
// Configuration remains request-scoped, including when running multiple replicas.
func (r *Router) integrationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// MCP settings cannot claim API paths, including nonexistent REST routes.
		if strings.HasPrefix(path, "/api/") {
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
		if path == "/.well-known/oauth-protected-resource"+settings.Path {
			if oauth == nil {
				c.AbortWithStatus(404)
				return
			}
			if c.Request.Method != http.MethodGet {
				c.Header("Allow", "GET")
				c.AbortWithStatus(405)
				return
			}
			handler.NewOAuthHandler(oauth).ResourceMetadata(c)
			c.Abort()
			return
		}
		if path != settings.Path && path != settings.Path+"/config" && path != settings.Path+"/info" {
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
		case path == settings.Path+"/info" && c.Request.Method == http.MethodGet:
			mcp.GetInfo(c)
		case path == settings.Path:
			mcp.Handle(c)
		default:
			c.Header("Allow", "GET")
			c.Status(405)
		}
		c.Abort()
	}
}

// OAuth configuration is resolved only at OAuth endpoints or when an OAuth
// credential is actually presented to REST. Other authentication modes remain
// independent of the MCP settings table.
func (r *Router) resolveOAuth(c *gin.Context) (*service.OAuthService, bool) {
	if current, ok := c.Get("oauth_service"); ok {
		oauth, _ := current.(*service.OAuthService)
		return oauth, true
	}
	settings, err := r.mcpSettings.Get(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"message": "暂时无法读取服务设置"})
		return nil, false
	}
	oauth, err := r.mcpSettings.OAuth(settings)
	if err != nil {
		c.AbortWithStatusJSON(503, gin.H{"message": "授权服务设置无效"})
		return nil, false
	}
	c.Set("oauth_service", oauth)
	return oauth, true
}
func (r *Router) oauthCredentialsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fields := strings.Fields(c.GetHeader("Authorization"))
		if len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") && strings.HasPrefix(fields[1], "ot_") {
			if _, ok := r.resolveOAuth(c); !ok {
				return
			}
		}
		c.Next()
	}
}
