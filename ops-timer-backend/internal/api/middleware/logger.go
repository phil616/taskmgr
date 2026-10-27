package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func sanitizeLogValue(s string) string {
	r := strings.NewReplacer("\n", "", "\r", "", "\t", " ")
	return r.Replace(s)
}

func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.NewString()
		// Preserve only well-formed opaque IDs provided by the ingress or parent request.
		if parsed, err := uuid.Parse(c.GetHeader("X-Request-ID")); err == nil {
			requestID = parsed.String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("request_id", requestID),
			zap.Int("response_bytes", max(0, c.Writer.Size())),
			zap.String("content_type", c.Writer.Header().Get("Content-Type")),
			zap.String("method", c.Request.Method),
			zap.String("path", sanitizeLogValue(path)),
			zap.Duration("latency", latency),
			zap.String("ip", sanitizeLogValue(c.ClientIP())),
		}
		if method := c.GetString("mcp_method"); method != "" {
			fields = append(fields, zap.String("mcp_method", sanitizeLogValue(method)), zap.String("mcp_tool", c.GetString("mcp_tool")), zap.String("mcp_error_category", c.GetString("mcp_error_category")), zap.Int("mcp_backend_status", c.GetInt("mcp_backend_status")), zap.Any("mcp_backend_latency", c.MustGet("mcp_backend_latency")))
		}
		if clientID := c.GetString("oauth_client_id"); clientID != "" {
			fields = append(fields, zap.String("oauth_client_id", clientID), zap.String("oauth_grant_id", c.GetString("oauth_grant_id")))
		}

		if status >= 500 {
			logger.Error("server error", fields...)
		} else if status >= 400 {
			logger.Warn("client error", fields...)
		} else {
			logger.Info("request", fields...)
		}
	}
}
