package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID attaches a request id to context and response header.
// If client sends X-Request-ID it is reused, otherwise generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

