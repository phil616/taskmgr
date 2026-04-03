package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const defaultMaxBodyBytes = 2 << 20 // 2 MiB

// BodySizeLimitMiddleware 限制请求体最大大小，防止超大 Payload 耗尽内存（DoS 防护）
func BodySizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBodyBytes
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
