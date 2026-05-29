package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

// Context keys for user info injected by JWT auth.
const (
	CtxKeyEmail = "user_email"
	CtxKeyRole  = "user_role"
)

// StaticTokenAuth is a Phase 0 placeholder.
func StaticTokenAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetHeader("X-Databrew-Token")
		if got == "" {
			got = c.GetHeader("Authorization")
		}
		if got == "" {
			if cookie, err := c.Cookie("databrew_session"); err == nil {
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

// JWTAuth returns a Gin middleware that authenticates via JWT.
//
// Auth source priority:
//  1. X-Grace-Token header → static token (legacy SDK path)
//  2. Authorization: Bearer <jwt> → JWT
//  3. grace_session cookie → JWT
//
// On success, user_email and user_role are set in the gin context.
func JWTAuth(staticToken, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Legacy static token via header → SDK backward compat.
		if header := c.GetHeader("X-Grace-Token"); header != "" {
			if header != staticToken && header != "Bearer "+staticToken {
				httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
				c.Abort()
				return
			}
			c.Set(CtxKeyEmail, "sdk")
			c.Set(CtxKeyRole, "admin")
			c.Next()
			return
		}

		// JWT from Authorization header or cookie.
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			if cookie, err := c.Cookie("grace_session"); err == nil {
				tokenStr = cookie
			}
		}
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		if tokenStr == "" {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		claims, jwtErr := auth.VerifyToken(jwtSecret, tokenStr)
		if jwtErr == nil {
			c.Set(CtxKeyEmail, claims.Email)
			c.Set(CtxKeyRole, claims.Role)
			c.Next()
			return
		}

		// Not a valid JWT — fall back to legacy static token.
		if tokenStr != staticToken {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, fmt.Sprintf("invalid token: %v", jwtErr))
			c.Abort()
			return
		}
		c.Set(CtxKeyEmail, "sdk")
		c.Set(CtxKeyRole, "admin")
		c.Next()
	}
}

// AdminTokenAuth checks for an admin-specific token via X-Admin-Token header.
// When adminToken is empty, non-production environments fall back to databrewToken;
// production must configure ADMIN_TOKEN (routes should also be unmounted).
func AdminTokenAuth(adminToken, databrewToken, env string) gin.HandlerFunc {
	if adminToken == "" {
		if env == "production" {
			return func(c *gin.Context) {
				httpresp.Error(c, http.StatusForbidden, httpresp.CodeUnauthorized, "admin routes require ADMIN_TOKEN in production", nil)
				c.Abort()
			}
		}
		return StaticTokenAuth(databrewToken)
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
