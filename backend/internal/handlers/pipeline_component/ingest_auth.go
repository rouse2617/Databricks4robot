package pipeline_component

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
)

// ReleaseIngestAuth accepts a CI-only token for release sync while preserving
// normal user/API authentication for the same endpoint.
func ReleaseIngestAuth(ciToken, staticToken, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if acceptsCIToken(c, ciToken) {
			c.Set(middleware.CtxKeyEmail, "ci")
			c.Set(middleware.CtxKeyRole, "component-release-ingest")
			c.Next()
			return
		}

		if header := strings.TrimSpace(c.GetHeader("X-Databrew-Token")); header != "" {
			if !constantTimeTokenEqual(header, staticToken) && !constantTimeTokenEqual(strings.TrimPrefix(header, "Bearer "), staticToken) {
				httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
				c.Abort()
				return
			}
			c.Set(middleware.CtxKeyEmail, "sdk")
			c.Set(middleware.CtxKeyRole, "admin")
			c.Next()
			return
		}

		tokenStr := strings.TrimSpace(c.GetHeader("Authorization"))
		if tokenStr == "" {
			if cookie, err := c.Cookie("databrew_session"); err == nil {
				tokenStr = strings.TrimSpace(cookie)
			}
		}
		tokenStr = strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer "))
		if tokenStr == "" {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		if claims, err := auth.VerifyToken(jwtSecret, tokenStr); err == nil {
			c.Set(middleware.CtxKeyEmail, claims.Email)
			c.Set(middleware.CtxKeyRole, claims.Role)
			c.Next()
			return
		}
		if constantTimeTokenEqual(tokenStr, staticToken) {
			c.Set(middleware.CtxKeyEmail, "sdk")
			c.Set(middleware.CtxKeyRole, "admin")
			c.Next()
			return
		}
		httpresp.Unauthorized(c, httpresp.CodeUnauthorized, fmt.Sprintf("invalid token: %s", "signature is invalid"))
		c.Abort()
	}
}

func acceptsCIToken(c *gin.Context, ciToken string) bool {
	ciToken = strings.TrimSpace(ciToken)
	if ciToken == "" {
		return false
	}
	header := strings.TrimSpace(c.GetHeader("X-Databrew-CI-Token"))
	if constantTimeTokenEqual(header, ciToken) || constantTimeTokenEqual(strings.TrimPrefix(header, "Bearer "), ciToken) {
		return true
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	return constantTimeTokenEqual(strings.TrimPrefix(authHeader, "Bearer "), ciToken)
}

func constantTimeTokenEqual(got, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got == "" || want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
