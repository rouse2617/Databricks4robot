package apikey

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/auth"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

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
