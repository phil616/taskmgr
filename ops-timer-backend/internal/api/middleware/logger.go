package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func sanitizeLogValue(s string) string {
	r := strings.NewReplacer("\n", "", "\r", "", "\t", " ")
	return r.Replace(s)
}

func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", sanitizeLogValue(path)),
			zap.String("query", sanitizeLogValue(query)),
			zap.Duration("latency", latency),
			zap.String("ip", sanitizeLogValue(c.ClientIP())),
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
