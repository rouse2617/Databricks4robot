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
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
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
		TargetID        string   `json:"targetId,omitempty"`
		TemplateVersion int      `json:"templateVersion,omitempty"`
		PilotCount      int      `json:"pilotCount,omitempty"`
		Config          *struct {
			Mode           string `json:"mode"`
			ConfigID       string `json:"configId"`
			Version        int    `json:"version"`
			FileName       string `json:"fileName"`
			Content        string `json:"content"`
			MountPath      string `json:"mountPath"`
			TargetFilename string `json:"targetFilename"`
		} `json:"configSelection,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	job, err := h.uc.CreateBackfill(c.Request.Context(), req.Name, req.TemplateID, req.AssetIDs, uc.CreateBackfillOptions{
		TargetID:        req.TargetID,
		TemplateVersion: req.TemplateVersion,
		PilotCount:      req.PilotCount,
		ConfigSelection: func() *pipelineUC.RuntimeConfigSelection {
			if req.Config == nil {
				return nil
			}
			return &pipelineUC.RuntimeConfigSelection{
				Mode:           req.Config.Mode,
				ConfigID:       req.Config.ConfigID,
				Version:        req.Config.Version,
				FileName:       req.Config.FileName,
				Content:        req.Config.Content,
				MountPath:      req.Config.MountPath,
				TargetFilename: req.Config.TargetFilename,
			}
		}(),
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
	var body struct {
		StopRunning bool `json:"stopRunning"`
	}
	_ = c.ShouldBindJSON(&body)
	result, err := h.uc.PauseJob(c.Request.Context(), id, uc.PauseJobOptions{
		StopRunning: body.StopRunning,
	})
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, result)
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

// GetItemAttempts handles GET /api/v1/backfill/:id/attempts.
func (h *Handler) GetItemAttempts(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("id"))
	itemID := strings.TrimSpace(c.Query("itemId"))
	assetID := strings.TrimSpace(c.Query("assetId"))
	if jobID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if itemID == "" && assetID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "itemId or assetId is required", nil)
		return
	}
	result, err := h.uc.GetItemAttempts(c.Request.Context(), jobID, itemID, assetID)
	if err != nil {
		mapBackfillError(c, err)
		return
	}
	c.JSON(200, result)
}

// ValidateAssets handles POST /api/v1/backfill/validate-assets.
func (h *Handler) ValidateAssets(c *gin.Context) {
	var req struct {
		AssetIDs []string `json:"assetIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	result, err := h.uc.ValidateAssets(c.Request.Context(), req.AssetIDs)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
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

// UploadResult handles POST /api/v1/backfill/results.
func (h *Handler) UploadResult(c *gin.Context) {
	var req struct {
		AssetID  string         `json:"assetId" binding:"required"`
		ReportID string         `json:"reportId" binding:"required"`
		Version  string         `json:"version" binding:"required"`
		Manifest map[string]any `json:"manifest"`
		Result   map[string]any `json:"result"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	out, err := h.uc.UploadResult(c.Request.Context(), uc.UploadResultInput{
		AssetID:  req.AssetID,
		ReportID: req.ReportID,
		Version:  req.Version,
		Manifest: req.Manifest,
		Result:   req.Result,
	})
	if err != nil {
		mapUploadResultError(c, err)
		return
	}
	c.JSON(201, out)
}

func mapUploadResultError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, uc.ErrAssetNotFound):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
	case errors.Is(err, uc.ErrReportManifestMissing):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
	case errors.Is(err, uc.ErrManifestMismatch),
		errors.Is(err, uc.ErrPayloadTooLarge),
		errors.Is(err, uc.ErrInvalidUploadRequest):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	default:
		httpresp.Internal(c, err.Error())
	}
}

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
