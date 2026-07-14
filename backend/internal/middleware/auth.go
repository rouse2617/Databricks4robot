package middleware

import (
	"crypto/subtle"
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

// tokenEquals compares two shared-secret tokens in constant time to prevent
// timing side channels leaking the configured secret one byte at a time. An
// optional "Bearer " prefix on `got` is stripped for RFC 6750 clients. Empty
// `want` (misconfiguration) always fails closed — an unconfigured token must
// never authenticate anyone. Length is checked first because
// subtle.ConstantTimeCompare requires equal-length inputs and returns 0
// (rather than constant-time false) for length mismatch.
func tokenEquals(got, want string) bool {
	if want == "" {
		return false
	}
	got = strings.TrimPrefix(got, "Bearer ")
	if len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// GetUserEmail extracts the authenticated user's email from the Gin context.
// Returns "legacy" when no email is set (backward compat for token auth).
func GetUserEmail(c *gin.Context) string {
	if email, ok := c.Get(CtxKeyEmail); ok {
		if s, ok := email.(string); ok && s != "" {
			return s
		}
	}
	return "legacy"
}

// ArgoWebhookAuth authenticates the Argo workflow status-push webhook (CYB-3058)
// via a dedicated token in the X-Databrew-Webhook-Token header, compared in
// constant time. It is intentionally separate from user/JWT auth: the caller is
// the Argo controller, not a user, so it must not receive broad API scope. An
// empty configured token disables the endpoint (always unauthorized).
func ArgoWebhookAuth(token string) gin.HandlerFunc {
	want := strings.TrimSpace(token)
	return func(c *gin.Context) {
		got := strings.TrimSpace(c.GetHeader("X-Databrew-Webhook-Token"))
		if want == "" || got == "" || len(got) != len(want) ||
			subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

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
		if !tokenEquals(got, token) {
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
//  1. X-Databrew-Token header → static token (legacy SDK path)
//  2. Authorization: Bearer <jwt> → JWT
//  3. databrew_session cookie → JWT
//
// On success, user_email and user_role are set in the gin context.
func JWTAuth(staticToken, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Legacy static token via header → SDK backward compat.
		if header := c.GetHeader("X-Databrew-Token"); header != "" {
			if !tokenEquals(header, staticToken) {
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
			if cookie, err := c.Cookie("databrew_session"); err == nil {
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
		if !tokenEquals(tokenStr, staticToken) {
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
		if !tokenEquals(got, adminToken) {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminTokenOrAdminRole guards READ-ONLY admin endpoints. It allows the request
// when the already-authenticated principal has the admin role (e.g. an
// ADMIN_EMAILS web session resolved by Authenticate on the parent group), and
// otherwise falls back to the static-token check (AdminTokenAuth).
//
// This lets web admins view read-only admin data (e.g. reindex history) without
// the static admin token, WITHOUT widening destructive admin/internal endpoints,
// which must keep using AdminTokenAuth directly (CYB-3229).
//
// Must be mounted under a group that already ran Authenticate (which populates the
// Principal); when no admin-role principal is present it degrades to the token
// check, so token/DatabrewToken callers keep working unchanged.
func AdminTokenOrAdminRole(adminToken, databrewToken, env string) gin.HandlerFunc {
	tokenAuth := AdminTokenAuth(adminToken, databrewToken, env)
	return func(c *gin.Context) {
		if p, ok := GetPrincipal(c); ok && p.Role == "admin" {
			c.Next()
			return
		}
		tokenAuth(c)
	}
}
