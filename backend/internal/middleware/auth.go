package middleware

import (
	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
)

// StaticTokenAuth is a Phase 0 placeholder.
// Replace with OIDC/JWT in Phase 0.5.
func StaticTokenAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetHeader("X-Grace-Token")
		if got == "" {
			got = c.GetHeader("Authorization")
		}
		if got == "" {
			if cookie, err := c.Cookie("grace_session"); err == nil {
				got = cookie
			}
		}
		if got != token && got != "Bearer "+token {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}
