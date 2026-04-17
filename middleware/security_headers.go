package middleware

import "github.com/gin-gonic/gin"

const permissionsPolicyHeaderValue = "camera=(), microphone=(), geolocation=()"

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", permissionsPolicyHeaderValue)
		c.Next()
	}
}
