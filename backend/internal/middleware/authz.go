package middleware

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Context keys.
const (
	// CtxKeyPrincipal holds the authenticated *Principal*.
	CtxKeyPrincipal = "principal"
	// CtxKeyScopes holds the principal's scopes ([]string). Kept as a separate
	// key for back-compat with GetScopes callers.
	CtxKeyScopes = "user_scopes"
)

// Auth methods (Principal.AuthMethod). Extend this list as new credential
// types are added (e.g. "oidc", "mtls", "service-account").
const (
	AuthMethodJWT         = "jwt"
	AuthMethodAPIKey      = "apikey" // pragma: allowlist secret
	AuthMethodStaticToken = "static-token"
)

// Principal is the authenticated identity plus authorization scopes, produced
// by Authenticate REGARDLESS of credential type (API key / JWT / static token,
// and in future OIDC / mTLS / external IdP). Downstream code depends on the
// Principal + its scopes, NOT on how the caller authenticated.
//
// Extension point: to support a more formal auth scheme later, add a branch in
// Authenticate (or a pluggable Authenticator) that yields a Principal — the
// scope-based authorization layer (RequireScope) and every handler stay
// unchanged, because they only ever read Principal/scopes.
type Principal struct {
	Subject    string   // email (user) or "apikey:<name>" / "sdk" (service)
	Type       string   // "user" | "service"
	Role       string   // role label (admin / user / service)
	Scopes     []string // authorization scopes; "*" is a wildcard = all
	AuthMethod string   // how the principal authenticated (see AuthMethod* consts)
}

// setPrincipal stores the Principal plus the legacy context keys that existing
// helpers (GetUserEmail, GetScopes) still read — so the abstraction is additive
// and nothing downstream breaks.
func setPrincipal(c *gin.Context, p Principal) {
	c.Set(CtxKeyPrincipal, p)
	c.Set(CtxKeyEmail, p.Subject)
	c.Set(CtxKeyRole, p.Role)
	c.Set(CtxKeyScopes, p.Scopes)
}

// GetPrincipal returns the authenticated Principal, if any.
func GetPrincipal(c *gin.Context) (Principal, bool) {
	if v, ok := c.Get(CtxKeyPrincipal); ok {
		if p, ok := v.(Principal); ok {
			return p, true
		}
	}
	return Principal{}, false
}

// ScopesForRole maps a role to a scope set, so web users and machine (API key)
// callers are authorized through the SAME scope check. Isolated here so it can
// later be replaced by scopes sourced from an IdP / policy store.
func ScopesForRole(role string) []string {
	switch role {
	case "admin":
		return []string{"*"}
	default: // "user" and anything else → standard app scopes
		return []string{"assets:read", "assets:write", "pipeline:run"}
	}
}

// GetScopes returns the principal's scopes from context.
func GetScopes(c *gin.Context) []string {
	if v, ok := c.Get(CtxKeyScopes); ok {
		if ss, ok := v.([]string); ok {
			return ss
		}
	}
	return nil
}

// HasScope reports whether scopes grant `want` ("*" is a wildcard = all).
func HasScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == "*" || s == want {
			return true
		}
	}
	return false
}

// RequireScope aborts with 403 unless the authenticated principal has `scope`.
// Must run after Authenticate (which populates the scopes).
func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasScope(GetScopes(c), scope) {
			c.Next()
			return
		}
		httpresp.Error(c, 403, httpresp.CodeUnauthorized, "missing required scope: "+scope, nil)
		c.Abort()
	}
}

// Authenticate is the unified authentication middleware. From a single
// normalized credential (X-Databrew-Token header, Authorization Bearer, or
// databrew_session cookie), it recognizes:
//
//   - API key      (dbk_<prefix>_<secret>) → looked up in the store, scoped
//   - static token (legacy SDK, admin)     → all scopes
//   - JWT          (web email-login)        → scopes derived from role
//
// Every path yields a Principal via setPrincipal, so RequireScope and handlers
// work uniformly. keys may be nil (API key path disabled).
func Authenticate(staticToken, jwtSecret string, keys repository.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		cred := strings.TrimSpace(c.GetHeader("X-Databrew-Token"))
		if cred == "" {
			cred = strings.TrimSpace(c.GetHeader("Authorization"))
		}
		if cred == "" {
			if ck, err := c.Cookie("databrew_session"); err == nil {
				cred = strings.TrimSpace(ck)
			}
		}
		cred = strings.TrimSpace(strings.TrimPrefix(cred, "Bearer "))
		if cred == "" {
			httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// 1) API key
		if auth.IsAPIKey(cred) {
			if keys == nil {
				httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "api keys not enabled")
				c.Abort()
				return
			}
			prefix, secret, ok := auth.ParseAPIKey(cred)
			if !ok {
				httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "malformed api key")
				c.Abort()
				return
			}
			k, err := keys.FindByPrefix(c.Request.Context(), prefix)
			if err != nil || k == nil || k.Status != "active" ||
				(k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now())) ||
				!auth.VerifyAPIKeySecret(secret, k.SecretHash) {
				httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "invalid api key")
				c.Abort()
				return
			}
			setPrincipal(c, Principal{
				Subject:    "apikey:" + apiKeyLabel(k.Name, k.Owner, k.KeyPrefix),
				Type:       "service",
				Role:       "service",
				Scopes:     k.Scopes,
				AuthMethod: AuthMethodAPIKey,
			})
			id := k.ID
			go func() { _ = keys.TouchLastUsed(context.Background(), id) }()
			c.Next()
			return
		}

		// 2) Legacy static token → admin (backward compat, never sunset here)
		if staticToken != "" && subtle.ConstantTimeCompare([]byte(cred), []byte(staticToken)) == 1 {
			setPrincipal(c, Principal{
				Subject:    "sdk",
				Type:       "service",
				Role:       "admin",
				Scopes:     ScopesForRole("admin"),
				AuthMethod: AuthMethodStaticToken,
			})
			c.Next()
			return
		}

		// 3) JWT (web session)
		if claims, err := auth.VerifyToken(jwtSecret, cred); err == nil {
			setPrincipal(c, Principal{
				Subject:    claims.Email,
				Type:       "user",
				Role:       claims.Role,
				Scopes:     ScopesForRole(claims.Role),
				AuthMethod: AuthMethodJWT,
			})
			c.Next()
			return
		}

		httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "unauthorized")
		c.Abort()
	}
}

func apiKeyLabel(name, owner, prefix string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	if strings.TrimSpace(owner) != "" {
		return owner
	}
	return prefix
}
