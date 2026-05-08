package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID attaches a request id to both gin.Context and the request's
// context.Context, and sets the X-Request-ID response header.
// If client sends X-Request-ID it is reused, otherwise generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)

		// Propagate into context.Context so usecase/repo layers can access it
		// via middleware.RequestIDFromContext(ctx) or middleware.L(ctx).
		ctx := WithRequestID(c.Request.Context(), rid)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
