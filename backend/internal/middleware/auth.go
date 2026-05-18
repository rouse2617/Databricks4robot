package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
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

// AdminTokenAuth checks for an admin-specific token via X-Admin-Token header.
// When adminToken is empty, it falls back to StaticTokenAuth with graceToken
// for backward compatibility.
func AdminTokenAuth(adminToken, graceToken string) gin.HandlerFunc {
	if adminToken == "" {
		return StaticTokenAuth(graceToken)
	}
	return func(c *gin.Context) {
		got := c.GetHeader("X-Admin-Token")
		if got == "" {
			got = c.GetHeader("Authorization")
		}
		if got != adminToken && got != "Bearer "+adminToken {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}
