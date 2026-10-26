package middleware

import (
	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/service"
)

// Integration credentials cannot change server policy, even for an admin user.
func AdminMiddleware(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		method := c.GetString("auth_method")
		if method != "cookie" && method != "jwt" {
			c.AbortWithStatusJSON(403, gin.H{"message": "请使用管理员网页登录或登录令牌管理系统设置"})
			return
		}
		user, err := auth.GetProfile(c.GetString("user_id"))
		if err != nil || !user.IsAdmin {
			c.AbortWithStatusJSON(403, gin.H{"message": "仅管理员可管理系统设置"})
			return
		}
		c.Next()
	}
}
