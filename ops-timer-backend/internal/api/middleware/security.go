package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware 设置常见安全响应头，防止常见 Web 攻击
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 禁止浏览器 MIME 嗅探，防止 content-type 混淆攻击
		c.Header("X-Content-Type-Options", "nosniff")
		// 禁止页面被嵌入 iframe，防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		// 启用 XSS 过滤（老浏览器兼容）
		c.Header("X-XSS-Protection", "1; mode=block")
		// 控制 Referer 信息，防止敏感路径泄露
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// 禁止使用部分浏览器 API（最小权限原则）
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// 后端纯 API 服务，CSP 限制资源来源
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// 移除服务器版本信息（信息最小化）
		c.Header("Server", "")
		c.Next()
	}
}
