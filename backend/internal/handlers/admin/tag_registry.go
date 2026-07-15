package admin

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// TagRegistryHandler serves admin CRUD for managed tag definitions (CYB-3246
// Phase 2). Writes persist to the DB and then refresh the in-memory validation
// map on the shared *config.TagRegistry so new definitions take effect without
// a service restart. The validation hot path itself never touches the DB.
type TagRegistryHandler struct {
	repo repository.TagRegistryRepository
	reg  *config.TagRegistry
}

// NewTagRegistryHandler wires the repo and the shared in-memory registry.
func NewTagRegistryHandler(repo repository.TagRegistryRepository, reg *config.TagRegistry) *TagRegistryHandler {
	return &TagRegistryHandler{repo: repo, reg: reg}
}

// tagDefRequest is the create/update body. On update, key comes from the path.
type tagDefRequest struct {
	Key         string   `json:"key"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Values      []string `json:"values"`
	MaxLength   int      `json:"max_length"`
	Propagation string   `json:"propagation"`
}

func entryToTagDef(e *models.TagRegistryEntry) config.TagDef {
	return config.TagDef{
		Description: e.Description,
		Type:        e.Type,
		Values:      e.Values,
		MaxLength:   e.MaxLength,
		Propagation: e.Propagation,
	}
}

// SeedAndLoadTagRegistry seeds the DB from the YAML-loaded in-memory registry
// when the table is empty, then loads all definitions from the DB into the
// in-memory validation map (CYB-3246 Phase 2). Idempotent: existing keys are
// skipped. Called once at server startup. On any error the caller should keep
// the YAML-loaded in-memory registry (already populated) as a safe fallback.
func entriesToDefMap(entries []*models.TagRegistryEntry) map[string]config.TagDef {
	m := make(map[string]config.TagDef, len(entries))
	for _, e := range entries {
		m[e.Key] = entryToTagDef(e)
	}
	return m
}

// LoadManagedTags loads DB-backed tag definitions into the in-memory managed
// overlay (CYB-3246 Phase 2). The YAML baseline is left intact — if the table
// is empty or unavailable (e.g. a migration has not applied yet), baseline
// validation keeps working and this simply installs an empty overlay. Called
// once at server startup. There is intentionally NO seeding: the DB holds only
// admin-managed definitions layered on top of the YAML baseline, so startup
// never races the migration job and admin writes can never erase the baseline.
func LoadManagedTags(ctx context.Context, repo repository.TagRegistryRepository, reg *config.TagRegistry) error {
	entries, err := repo.List(ctx)
	if err != nil {
		return err
	}
	reg.SetManagedTags(entriesToDefMap(entries))
	return nil
}

// refreshRegistry reloads DB definitions into the managed overlay (leaving the
// YAML baseline intact) so admin writes take effect without a restart.
func (h *TagRegistryHandler) refreshRegistry(ctx context.Context) error {
	entries, err := h.repo.List(ctx)
	if err != nil {
		return err
	}
	h.reg.SetManagedTags(entriesToDefMap(entries))
	return nil
}

// normalizeAndValidate applies defaults and enforces the definition invariants,
// returning a user-facing error message when invalid.
func normalizeAndValidate(req *tagDefRequest) string {
	req.Key = strings.TrimSpace(req.Key)
	req.Type = strings.TrimSpace(req.Type)
	if req.Propagation == "" {
		req.Propagation = "none"
	}
	if req.Key == "" {
		return "key is required"
	}
	switch req.Type {
	case "enum":
		if len(req.Values) == 0 {
			return "enum type requires a non-empty values list"
		}
	case "string":
		if req.MaxLength < 0 {
			return "max_length must be >= 0"
		}
	default:
		return "type must be 'enum' or 'string'"
	}
	if req.Propagation != "none" && req.Propagation != "descendants" {
		return "propagation must be 'none' or 'descendants'"
	}
	return ""
}

// List returns the effective tag definitions: DB-managed entries (editable)
// plus YAML-baseline entries not overridden in the DB (read-only), each marked
// with `managed`. GET /admin/tag-registry
func (h *TagRegistryHandler) List(c *gin.Context) {
	dbEntries, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	out := make([]*models.TagRegistryEntry, 0, len(dbEntries))
	seen := make(map[string]struct{}, len(dbEntries))
	for _, e := range dbEntries {
		e.Managed = true
		if e.Values == nil {
			e.Values = []string{}
		}
		seen[e.Key] = struct{}{}
		out = append(out, e)
	}
	// Include YAML-baseline keys not overridden in the DB, as read-only rows.
	for k, def := range h.reg.GetBaseTags() {
		if _, ok := seen[k]; ok {
			continue
		}
		values := def.Values
		if values == nil {
			values = []string{}
		}
		out = append(out, &models.TagRegistryEntry{
			Key:         k,
			Description: def.Description,
			Type:        def.Type,
			Values:      values,
			MaxLength:   def.MaxLength,
			Propagation: def.Propagation,
			CreatedBy:   "yaml",
			Managed:     false,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// Create registers a new managed tag definition. POST /admin/tag-registry
func (h *TagRegistryHandler) Create(c *gin.Context) {
	var req tagDefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if msg := normalizeAndValidate(&req); msg != "" {
		httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, msg, nil)
		return
	}
	entry := &models.TagRegistryEntry{
		Key:         req.Key,
		Description: req.Description,
		Type:        req.Type,
		Values:      req.Values,
		MaxLength:   req.MaxLength,
		Propagation: req.Propagation,
		CreatedBy:   middleware.GetUserEmail(c),
	}
	created, err := h.repo.Create(c.Request.Context(), entry)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateTagKey) {
			httpresp.Conflict(c, httpresp.CodeTagKeyExists, "tag key already exists", map[string]any{"key": req.Key})
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	if err := h.refreshRegistry(c.Request.Context()); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	audit.Log(c.Request.Context(), "tag_registry.create", "tag_registry", []string{req.Key}, map[string]any{
		"type": req.Type, "values": req.Values, "propagation": req.Propagation,
	})
	c.JSON(http.StatusCreated, created)
}

// Update mutates an existing definition. PATCH /admin/tag-registry/:key
func (h *TagRegistryHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req tagDefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	req.Key = key
	if msg := normalizeAndValidate(&req); msg != "" {
		httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, msg, nil)
		return
	}
	entry := &models.TagRegistryEntry{
		Key:         key,
		Description: req.Description,
		Type:        req.Type,
		Values:      req.Values,
		MaxLength:   req.MaxLength,
		Propagation: req.Propagation,
	}
	updated, err := h.repo.Update(c.Request.Context(), entry)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if updated == nil {
		httpresp.NotFound(c, httpresp.CodeTagNotFound, "tag key not found")
		return
	}
	if err := h.refreshRegistry(c.Request.Context()); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	audit.Log(c.Request.Context(), "tag_registry.update", "tag_registry", []string{key}, map[string]any{
		"type": req.Type, "values": req.Values, "propagation": req.Propagation,
	})
	c.JSON(http.StatusOK, updated)
}

// Delete removes a definition. DELETE /admin/tag-registry/:key
func (h *TagRegistryHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	existing, err := h.repo.Get(c.Request.Context(), key)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if existing == nil {
		httpresp.NotFound(c, httpresp.CodeTagNotFound, "tag key not found")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), key); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if err := h.refreshRegistry(c.Request.Context()); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	audit.Log(c.Request.Context(), "tag_registry.delete", "tag_registry", []string{key}, nil)
	c.JSON(http.StatusOK, gin.H{"deleted": true, "key": key})
}
