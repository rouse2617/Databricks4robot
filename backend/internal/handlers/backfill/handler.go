package backfill

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/backfill"
)

// Handler bundles the backfill endpoints.
type Handler struct {
	uc *uc.Usecase
}

// New constructs a Handler.
func New(uc *uc.Usecase) *Handler { return &Handler{uc: uc} }

// CreateJob handles POST /api/v1/backfill.
func (h *Handler) CreateJob(c *gin.Context) {
	var req struct {
		Name            string   `json:"name" binding:"required"`
		TemplateID      string   `json:"templateId" binding:"required"`
		AssetIDs        []string `json:"assetIds" binding:"required,min=1"`
		TemplateVersion int      `json:"templateVersion,omitempty"`
		PilotCount      int      `json:"pilotCount,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	job, err := h.uc.CreateBackfill(c.Request.Context(), req.Name, req.TemplateID, req.AssetIDs, uc.CreateBackfillOptions{
		TemplateVersion: req.TemplateVersion,
		PilotCount:      req.PilotCount,
	})
	if err != nil {
		mapCreateJobError(c, err)
		return
	}
	c.JSON(201, job)
}

// ListJobs handles GET /api/v1/backfill.
func (h *Handler) ListJobs(c *gin.Context) {
	items, err := h.uc.ListJobs(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.BackfillJob{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetJob handles GET /api/v1/backfill/:id.
func (h *Handler) GetJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	job, err := h.uc.GetJob(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if job == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "backfill job not found")
		return
	}

	c.JSON(200, gin.H{"job": job})
}

// PauseJob handles POST /api/v1/backfill/:id/pause.
func (h *Handler) PauseJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.PauseJob(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"status": "paused"})
}

// ResumeJob handles POST /api/v1/backfill/:id/resume.
func (h *Handler) ResumeJob(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.ResumeJob(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"status": "resumed"})
}

// RetryFailed handles POST /api/v1/backfill/:id/retry-failed.
func (h *Handler) RetryFailed(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.RetryFailed(c.Request.Context(), id); err != nil {
		mapBackfillError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "retrying"})
}

// GetNodeSummary handles GET /api/v1/backfill/:id/node-summary.
func (h *Handler) GetNodeSummary(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	summary, err := h.uc.GetBatchNodeSummary(c.Request.Context(), id)
	if err != nil {
		mapBackfillError(c, err)
		return
	}
	c.JSON(200, summary)
}

// ListNodeFailures handles GET /api/v1/backfill/:id/node-failures.
func (h *Handler) ListNodeFailures(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	nodeID := strings.TrimSpace(c.Query("pipelineNodeId"))
	if id == "" || nodeID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id and pipelineNodeId are required", nil)
		return
	}
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("pageSize"), 20)
	statuses := parseCSV(c.Query("status"))
	result, err := h.uc.ListNodeFailures(c.Request.Context(), id, repository.BatchNodeFailureFilter{
		PipelineNodeID: nodeID,
		Statuses:       statuses,
		Page:           page,
		PageSize:       pageSize,
		Query:          strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		mapBackfillError(c, err)
		return
	}
	c.JSON(200, result)
}

// Rerun handles POST /api/v1/backfill/:id/rerun.
func (h *Handler) Rerun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	var req struct {
		Scope           string   `json:"scope"`
		TemplateID      string   `json:"templateId"`
		TemplateVersion int      `json:"templateVersion"`
		ItemIDs         []string `json:"itemIds"`
		AssetIDs        []string `json:"assetIds"`
		PipelineNodeID  string   `json:"pipelineNodeId"`
		DryRun          bool     `json:"dryRun"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	result, err := h.uc.Rerun(c.Request.Context(), id, uc.RerunRequest{
		Scope:           req.Scope,
		TemplateID:      req.TemplateID,
		TemplateVersion: req.TemplateVersion,
		ItemIDs:         req.ItemIDs,
		AssetIDs:        req.AssetIDs,
		PipelineNodeID:  req.PipelineNodeID,
		DryRun:          req.DryRun,
	})
	if err != nil {
		mapBackfillError(c, err)
		return
	}
	c.JSON(200, result)
}

// ContinueFull handles POST /api/v1/backfill/:id/continue-full.
func (h *Handler) ContinueFull(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.ContinueFull(c.Request.Context(), id); err != nil {
		mapBackfillError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "running"})
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return fallback
	}
	return v
}

func mapBackfillError(c *gin.Context, err error) {
	if errors.Is(err, uc.ErrNotFound) {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		return
	}
	if errors.Is(err, uc.ErrInvalidRerunScope) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	httpresp.Internal(c, err.Error())
}

func mapCreateJobError(c *gin.Context, err error) {
	if errors.Is(err, uc.ErrTooManyAssets) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if errors.Is(err, assetvalidation.ErrInvalidAssetIDs) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	httpresp.Internal(c, err.Error())
}
