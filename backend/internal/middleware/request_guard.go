package middleware

import (
	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
)

// RequestGuard rejects requests with excessively long URIs or bodies
// to prevent abuse and avoid unexpected 500 errors from downstream components.
func RequestGuard(maxURILen int) gin.HandlerFunc {
	if maxURILen <= 0 {
		maxURILen = 2048
	}
	return func(c *gin.Context) {
		if len(c.Request.RequestURI) > maxURILen {
			httpresp.Error(c, 414, httpresp.CodeURITooLong, "request URI exceeds maximum length", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
