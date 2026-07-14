package apikey

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// PrivilegedScopes are scopes that MUST NOT be granted through a short-lived
// JWT session (e.g. admin JWT obtained via email-login has no proof of email
// ownership and lasts 24h). Requests asking for these scopes are only allowed
// when the caller authenticated with a static admin token, which requires
// proof-of-possession of the shared secret.
var PrivilegedScopes = map[string]struct{}{
	"*":              {}, // wildcard = every scope, indefinitely
	"apikeys:manage": {}, // can self-perpetuate: mint further keys with any scope
}

// Handler manages API keys (create / list / revoke). Mounted under admin auth.
type Handler struct {
	repo repository.APIKeyRepository
}

// New constructs a Handler.
func New(repo repository.APIKeyRepository) *Handler { return &Handler{repo: repo} }

// Create handles POST /api/v1/admin/api-keys. Returns the plaintext key ONCE.
func (h *Handler) Create(c *gin.Context) {
	var req struct {
		Name      string     `json:"name"`
		Owner     string     `json:"owner"`
		Scopes    []string   `json:"scopes" binding:"required,min=1"`
		ExpiresAt *time.Time `json:"expiresAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "name/owner optional, scopes required (min 1)", map[string]any{"error": err.Error()})
		return
	}
	// SECURITY: reject privileged scopes unless the caller used the static
	// admin token. A JWT-authenticated admin (email-login) is only a
	// time-limited operator session and must not be able to mint a
	// long-lived key that outlives its own 24h TTL. Without this check any
	// email-login admin JWT could mint a wildcard-scope key that survives
	// JWT expiration.
	principal, _ := middleware.GetPrincipal(c)
	for _, s := range req.Scopes {
		if s = strings.TrimSpace(s); s == "" {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "scope entries must be non-empty", nil)
			return
		}
		if _, priv := PrivilegedScopes[s]; priv && principal.AuthMethod != middleware.AuthMethodStaticToken {
			httpresp.Error(c, http.StatusForbidden, "FORBIDDEN",
				"issuing scope "+s+" requires the static admin token; a JWT session cannot mint long-lived privileged keys",
				nil)
			return
		}
	}
	full, prefix, secretHash, err := auth.GenerateAPIKey()
	if err != nil {
		httpresp.Internal(c, "failed to generate key")
		return
	}
	k := &models.APIKey{
		KeyPrefix:  prefix,
		SecretHash: secretHash,
		Name:       strings.TrimSpace(req.Name),
		Owner:      strings.TrimSpace(req.Owner),
		Scopes:     req.Scopes,
		Status:     "active",
		ExpiresAt:  req.ExpiresAt,
		CreatedBy:  middleware.GetUserEmail(c),
	}
	if err := h.repo.Create(c.Request.Context(), k); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, gin.H{
		"id":        k.ID,
		"key":       full, // shown ONCE; only the hash is stored
		"keyPrefix": k.KeyPrefix,
		"name":      k.Name,
		"owner":     k.Owner,
		"scopes":    k.Scopes,
		"expiresAt": k.ExpiresAt,
		"note":      "store this key now — it will not be shown again",
	})
}

// List handles GET /api/v1/admin/api-keys. SecretHash is never serialized.
func (h *Handler) List(c *gin.Context) {
	items, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.APIKey{}
	}
	c.JSON(200, gin.H{"items": items})
}

// Revoke handles DELETE /api/v1/admin/api-keys/:id.
func (h *Handler) Revoke(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.repo.Revoke(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"status": "revoked", "id": id})
}
