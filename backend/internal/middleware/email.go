package middleware

import (
	"github.com/gin-gonic/gin"
)

// emailKey is the Gin context key for the user email.
const emailKey = "user_email"

// UserEmail extracts the X-User-Email header and stores it in the Gin context.
// The header is set by the Python SDK (or any client) for audit trail purposes.
// If the header is empty, the middleware passes through without setting the key
// so handlers can distinguish "not provided" from "provided and empty".
func UserEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := c.GetHeader("X-User-Email")
		if email != "" {
			c.Set(emailKey, email)
		}
		c.Next()
	}
}

// UserEmailFromContext retrieves the user email previously stored by UserEmail middleware.
// Returns empty string if not set.
func UserEmailFromContext(c *gin.Context) string {
	if v, ok := c.Get(emailKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
