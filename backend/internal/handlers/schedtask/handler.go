// Package schedtask (handlers) exposes CRUD + lifecycle endpoints for the
// scheduled-task rules (CYB-3744). Wired under /api/v1/scheduled-tasks in
// backend/routes/routes.go behind the same admin-authenticated group as other
// admin endpoints.
//
// Response shape is deliberately consistent with existing collection handlers
// (`gin.H{"items": [...], "total": N}`) so the frontend can reuse its generic
// list components without special-casing.
package schedtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/schedtask"
)

// Repo is the persistence surface the handler needs. Concrete impl is
// *postgres.ScheduledTaskRepo. Kept as an interface so tests can mock without
// pgx.
type Repo interface {
	List(ctx context.Context) ([]schedtask.Rule, error)
	Get(ctx context.Context, id string) (*schedtask.Rule, error)
	Create(ctx context.Context, rule *schedtask.Rule) error
	Update(ctx context.Context, rule *schedtask.Rule) error
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	RequestRunNow(ctx context.Context, id string) error
}

// Handler wires HTTP endpoints to the repo. Business logic beyond validation
// (which fields are required per trigger_mode etc.) lives in the schedtask
// usecase / repo; the handler is deliberately thin.
type Handler struct {
	repo Repo
}

func New(repo Repo) *Handler { return &Handler{repo: repo} }

// ─── DTOs ─────────────────────────────────────────────────────

// ruleRequest is the create/update payload. All optional-when-absent fields
// use pointers or zero-of-type so we can tell "unset" from "empty".
type ruleRequest struct {
	Name            string          `json:"name"`
	Enabled         *bool           `json:"enabled,omitempty"`
	TemplateID      string          `json:"templateId"`
	TemplateVersion *int            `json:"templateVersion,omitempty"`
	TargetID        string          `json:"targetId"`
	Scheduling      json.RawMessage `json:"scheduling,omitempty"`
	SourceType      string          `json:"sourceType"`
	SourceConfig    json.RawMessage `json:"sourceConfig,omitempty"`
	TriggerMode     string          `json:"triggerMode"`
	TriggerConfig   json.RawMessage `json:"triggerConfig,omitempty"`
}

// runNowRequest is the manual-run payload. All fields optional; when empty,
// re-run the rule with its own definition.
//
// This lets 立即运行 double as an ad-hoc backfill (mirrors grace-sync's --date/
// --from-to/--ids), without editing the rule itself.
type runNowRequest struct {
	Mode string `json:"mode,omitempty"` // "range" | "ids" — overrides the rule's mode for this run only
	// TODO(v2): also accept ad-hoc trigger_config here. For v1, run-now uses
	// the rule's own trigger_config; edit the rule to change it. Keeping the
	// request DTO in place so we don't break the URL when we add fields.
}

// ─── endpoints ────────────────────────────────────────────────

// List handles GET /api/v1/scheduled-tasks?enabled=&sourceType=&q=&page=&pageSize=.
//
// The repo returns all rules ordered by name; filter + paginate in-handler.
// For thousands-of-rules scale we'd push filters into SQL, but the design
// contract for this table is dozens at most (one per data-source × pipeline
// combination), so keeping it simple here.
func (h *Handler) List(c *gin.Context) {
	rules, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	// Filters.
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	enabled := c.Query("enabled") // "" | "true" | "false"
	sourceType := c.Query("sourceType")
	filtered := make([]schedtask.Rule, 0, len(rules))
	for _, r := range rules {
		if q != "" && !strings.Contains(strings.ToLower(r.Name), q) && !strings.Contains(strings.ToLower(r.ID), q) {
			continue
		}
		if enabled == "true" && !r.Enabled {
			continue
		}
		if enabled == "false" && r.Enabled {
			continue
		}
		if sourceType != "" && !strings.EqualFold(r.SourceType, sourceType) {
			continue
		}
		filtered = append(filtered, r)
	}
	page, pageSize := parsePaging(c)
	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	c.JSON(http.StatusOK, gin.H{
		"items":    filtered[start:end],
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Get handles GET /api/v1/scheduled-tasks/:id.
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	rule, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if rule == nil {
		httpresp.NotFound(c, httpresp.CodeScheduledTaskNotFound, "scheduled task not found")
		return
	}
	c.JSON(http.StatusOK, rule)
}

// Create handles POST /api/v1/scheduled-tasks.
func (h *Handler) Create(c *gin.Context) {
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if err := validateRuleRequest(req, true); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	rule := requestToRule(req, "")
	rule.ID = "sched_" + uuid.New().String()
	rule.CreatedBy = middleware.GetUserEmail(c)
	rule.CreatedAt = time.Now().UTC()
	rule.UpdatedAt = rule.CreatedAt
	if err := h.repo.Create(c.Request.Context(), &rule); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// Update handles PUT /api/v1/scheduled-tasks/:id.
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if existing == nil {
		httpresp.NotFound(c, httpresp.CodeScheduledTaskNotFound, "scheduled task not found")
		return
	}
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if err := validateRuleRequest(req, false); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	rule := requestToRule(req, id)
	// Preserve create metadata; only mutable fields are replaced.
	rule.CreatedBy = existing.CreatedBy
	rule.CreatedAt = existing.CreatedAt
	if err := h.repo.Update(c.Request.Context(), &rule); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	// Return the fresh row so the frontend doesn't have to re-fetch.
	fresh, _ := h.repo.Get(c.Request.Context(), id)
	if fresh != nil {
		c.JSON(http.StatusOK, fresh)
		return
	}
	c.JSON(http.StatusOK, rule)
}

// Delete handles DELETE /api/v1/scheduled-tasks/:id.
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// Pause handles POST /api/v1/scheduled-tasks/:id/pause.
func (h *Handler) Pause(c *gin.Context) {
	h.setEnabled(c, false)
}

// Resume handles POST /api/v1/scheduled-tasks/:id/resume.
func (h *Handler) Resume(c *gin.Context) {
	h.setEnabled(c, true)
}

func (h *Handler) setEnabled(c *gin.Context, enabled bool) {
	id := c.Param("id")
	if err := h.repo.SetEnabled(c.Request.Context(), id, enabled); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	rule, _ := h.repo.Get(c.Request.Context(), id)
	if rule == nil {
		httpresp.NotFound(c, httpresp.CodeScheduledTaskNotFound, "scheduled task not found")
		return
	}
	c.JSON(http.StatusOK, rule)
}

// RunNow handles POST /api/v1/scheduled-tasks/:id/run-now.
//
// v1: marks the rule to run on the next scheduler tick regardless of interval.
// Optional payload lets a caller override the mode for that single run in a
// future version (see runNowRequest TODO).
func (h *Handler) RunNow(c *gin.Context) {
	id := c.Param("id")
	// Body is optional in v1 (we ignore mode/params for now, but accept &
	// validate the shape so the field name is booked).
	if c.Request.ContentLength > 0 {
		var req runNowRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
			return
		}
		if req.Mode != "" && req.Mode != "range" && req.Mode != "ids" {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "unsupported mode; v1 accepts only \"range\" or \"ids\" (unused)", nil)
			return
		}
	}
	if err := h.repo.RequestRunNow(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	rule, _ := h.repo.Get(c.Request.Context(), id)
	if rule == nil {
		httpresp.NotFound(c, httpresp.CodeScheduledTaskNotFound, "scheduled task not found")
		return
	}
	c.JSON(http.StatusAccepted, rule)
}

// ─── helpers ──────────────────────────────────────────────────

func parsePaging(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 50
	if v := c.Query("page"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &pageSize); err != nil || pageSize < 1 {
			pageSize = 50
		}
	}
	if pageSize > 500 {
		pageSize = 500
	}
	return
}

// validateRuleRequest rejects payloads the DB CHECK would reject anyway, but
// with a friendlier 400 instead of a 500 from the DB constraint. Also
// validates trigger_mode + source_type v1 whitelists so the UI form errors
// stay predictable.
func validateRuleRequest(req ruleRequest, forCreate bool) error {
	if forCreate && strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.TemplateID) == "" {
		return errors.New("templateId is required")
	}
	if strings.TrimSpace(req.TargetID) == "" {
		return errors.New("targetId is required")
	}
	switch strings.ToLower(strings.TrimSpace(req.SourceType)) {
	case "rest":
	default:
		return fmt.Errorf("unsupported sourceType %q (v1: rest)", req.SourceType)
	}
	switch schedtask.TriggerMode(strings.ToLower(strings.TrimSpace(req.TriggerMode))) {
	case schedtask.TriggerIncremental, schedtask.TriggerRolling, schedtask.TriggerRange, schedtask.TriggerIDs:
	default:
		return fmt.Errorf("unsupported triggerMode %q", req.TriggerMode)
	}
	return nil
}

func requestToRule(req ruleRequest, id string) schedtask.Rule {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return schedtask.Rule{
		ID:              id,
		Name:            strings.TrimSpace(req.Name),
		Enabled:         enabled,
		TemplateID:      strings.TrimSpace(req.TemplateID),
		TemplateVersion: req.TemplateVersion,
		TargetID:        strings.TrimSpace(req.TargetID),
		Scheduling:      req.Scheduling,
		SourceType:      strings.ToLower(strings.TrimSpace(req.SourceType)),
		SourceConfig:    req.SourceConfig,
		TriggerMode:     schedtask.TriggerMode(strings.ToLower(strings.TrimSpace(req.TriggerMode))),
		TriggerConfig:   req.TriggerConfig,
	}
}
