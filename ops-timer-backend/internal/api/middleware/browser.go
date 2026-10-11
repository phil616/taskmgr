package middleware

import (
	"github.com/gin-gonic/gin"
	"net/url"
)

// A mandatory custom header forces cross-origin browsers through CORS preflight.
// Also check Origin explicitly, including requests from untrusted same-site hosts.
func browserRequestAllowed(c *gin.Context, allowedOrigins []string) bool {
	if c.GetHeader("X-Taskmgr-Browser") != "1" {
		return false
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		return c.GetHeader("Sec-Fetch-Site") != "cross-site"
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return false
	}
	if parsed.Host == c.Request.Host {
		return true
	}
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func BrowserRequestGuard(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if !browserRequestAllowed(c, origins) {
			c.AbortWithStatus(403)
			return
		}
		c.Next()
	}
}
