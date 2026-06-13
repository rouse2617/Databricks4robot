package pipeline

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// Handler bundles the pipeline endpoints.
type Handler struct {
	uc      *pipelineUC.Usecase
	pricing *pipelineUC.PricingConfig
}

// New constructs a Handler. When pricingPath is non-empty the GCP pricing
// YAML is loaded at construction time for cost estimation.
func New(uc *pipelineUC.Usecase, pricingPath string) *Handler {
	var pricing *pipelineUC.PricingConfig
	if strings.TrimSpace(pricingPath) != "" {
		pricing, _ = pipelineUC.LoadPricing(pricingPath)
	}
	return &Handler{uc: uc, pricing: pricing}
}

// SaveTemplate handles POST /api/v1/pipelines.
func (h *Handler) SaveTemplate(c *gin.Context) {
	var req struct {
		Name     string                 `json:"name" binding:"required"`
		Pipeline map[string]interface{} `json:"pipeline" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	t, err := h.uc.SaveTemplate(c.Request.Context(), req.Name, req.Pipeline, "dev", middleware.GetUserEmail(c))
	if err != nil {
		if errors.Is(err, pipelineUC.ErrInvalidArgument) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, t)
}

// Promote handles POST /api/v1/pipelines/:id/promote.
func (h *Handler) Promote(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template id is required", nil)
		return
	}
	t, err := h.uc.Promote(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrTemplateNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, t)
}

// ListTemplates handles GET /api/v1/pipelines.
func (h *Handler) ListTemplates(c *gin.Context) {
	items, err := h.uc.ListTemplates(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineTemplate{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetTemplate handles GET /api/v1/pipelines/:id.
func (h *Handler) GetTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	t, err := h.uc.GetTemplate(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if t == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "template not found")
		return
	}
	c.JSON(200, t)
}

// DeleteTemplate handles DELETE /api/v1/pipelines/:id.
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	if err := h.uc.DeleteTemplate(c.Request.Context(), id, middleware.GetUserEmail(c)); err != nil {
		if errors.Is(err, pipelineUC.ErrProdLocked) || errors.Is(err, pipelineUC.ErrTemplateNotOwned) {
			httpresp.Error(c, http.StatusForbidden, httpresp.CodeInvalidArgument, err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(204)
}

// SetActiveVersion handles PATCH /api/v1/pipelines/:id/active-version.
func (h *Handler) SetActiveVersion(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template id is required", nil)
		return
	}
	var req struct {
		ActiveVersion int `json:"activeVersion"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if req.ActiveVersion < 0 {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "activeVersion must be >= 0", nil)
		return
	}
	if err := h.uc.SetActiveVersion(c.Request.Context(), id, req.ActiveVersion); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"message": "ok", "activeVersion": req.ActiveVersion})
}

// ListVersions handles GET /api/v1/pipelines/:id/versions.
func (h *Handler) ListVersions(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template id is required", nil)
		return
	}

	items, err := h.uc.ListVersions(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineTemplate{}
	}
	c.JSON(200, gin.H{"items": items})
}

// Deploy handles POST /api/v1/deploy.
func (h *Handler) Deploy(c *gin.Context) {
	var req struct {
		Pipeline map[string]interface{} `json:"pipeline" binding:"required"`
		Name     string                 `json:"name"`
		AssetIDs []string               `json:"asset_ids"`
		TargetID string                 `json:"target_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	dryRun := false
	if raw := strings.TrimSpace(c.Query("dryRun")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid dryRun query value", map[string]any{"error": err.Error()})
			return
		}
		dryRun = v
	}

	dep, err := h.uc.Deploy(c.Request.Context(), req.Pipeline, req.Name, req.AssetIDs, pipelineUC.DeployOptions{DryRun: dryRun, TargetID: req.TargetID, Owner: middleware.GetUserEmail(c)})
	if err != nil {
		mapDeployError(c, err)
		return
	}
	if dryRun {
		c.JSON(http.StatusOK, dep)
		return
	}
	c.JSON(http.StatusCreated, dep)
}

// DeployByTemplate handles POST /api/v1/deploy/template/:id.
func (h *Handler) DeployByTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template id is required", nil)
		return
	}

	var req struct {
		Name     string   `json:"name"`
		AssetIDs []string `json:"asset_ids"`
		TargetID string   `json:"target_id"`
		Version  int      `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	dep, err := h.uc.DeployByTemplateID(c.Request.Context(), id, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID, TemplateVersion: req.Version, Owner: middleware.GetUserEmail(c)})
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(201, dep)
}

// CreateRun handles POST /api/v1/pipeline-runs.
func (h *Handler) CreateRun(c *gin.Context) {
	var req struct {
		Pipeline map[string]interface{} `json:"pipeline" binding:"required"`
		Name     string                 `json:"name"`
		AssetIDs []string               `json:"asset_ids"`
		TargetID string                 `json:"target_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	run, err := h.uc.CreateRun(c.Request.Context(), req.Pipeline, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID})
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(http.StatusCreated, run)
}

// CreateRunByTemplate handles POST /api/v1/pipeline-runs/template/:id.
func (h *Handler) CreateRunByTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template id is required", nil)
		return
	}
	var req struct {
		Name     string   `json:"name"`
		AssetIDs []string `json:"asset_ids"`
		TargetID string   `json:"target_id"`
		Version  int      `json:"version"`
	}
	// Body is optional: empty body is fine, but a non-empty body that fails to
	// bind (malformed JSON, wrong content type) is a client error and must not
	// silently fall through to the usecase.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	run, err := h.uc.CreateRunByTemplateID(c.Request.Context(), id, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID, TemplateVersion: req.Version, Owner: middleware.GetUserEmail(c)})
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(http.StatusCreated, run)
}

// ListExecutionTargets handles GET /api/v1/execution-targets.
func (h *Handler) ListExecutionTargets(c *gin.Context) {
	items, err := h.uc.ListExecutionTargets(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"items": items})
}

// ListRuns handles GET /api/v1/pipeline-runs.
func (h *Handler) ListRuns(c *gin.Context) {
	refreshActive := strings.EqualFold(c.Query("refresh"), "true") || c.Query("refresh") == "1"
	summaryView := strings.EqualFold(c.Query("view"), "summary")

	var (
		items []models.PipelineRun
		err   error
	)
	if summaryView {
		items, err = h.uc.ListRunSummaries(c.Request.Context())
	} else {
		items, err = h.uc.ListRuns(c.Request.Context(), refreshActive)
	}
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineRun{}
	}
	if summaryView {
		for i := range items {
			items[i].TotalEstimatedCost = nil
		}
	} else {
		for i := range items {
			items[i].TotalEstimatedCost = pipelineUC.ComputeRunCost(&items[i], h.pricing)
		}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetRun handles GET /api/v1/pipeline-runs/:id.
func (h *Handler) GetRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	run, err := h.uc.GetRun(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if run == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
		return
	}
	run.TotalEstimatedCost = pipelineUC.ComputeRunCost(run, h.pricing)
	c.JSON(200, run)
}

// ListRunEvents handles GET /api/v1/pipeline-runs/:id/events.
func (h *Handler) ListRunEvents(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 || v > 500 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "limit must be between 1 and 500", nil)
			return
		}
		limit = v
	}
	var cursor int64
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v < 0 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "cursor must be a non-negative sequence", nil)
			return
		}
		cursor = v
	}
	var from *time.Time
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		v, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "from must be RFC3339", nil)
			return
		}
		from = &v
	}
	var to *time.Time
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		v, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "to must be RFC3339", nil)
			return
		}
		to = &v
	}
	result, err := h.uc.ListRunEvents(c.Request.Context(), id, models.PipelineRunEventListOptions{
		Limit:       limit,
		Cursor:      cursor,
		SubjectType: strings.TrimSpace(c.Query("subjectType")),
		EventType:   strings.TrimSpace(c.Query("eventType")),
		Status:      strings.TrimSpace(c.Query("status")),
		Query:       strings.TrimSpace(c.Query("q")),
		From:        from,
		To:          to,
	})
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, result)
}

// ListRunAssetNodes handles GET /api/v1/pipeline-runs/:id/asset-nodes.
func (h *Handler) ListRunAssetNodes(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 || v > 500 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "limit must be between 1 and 500", nil)
			return
		}
		limit = v
	}
	result, err := h.uc.ListRunAssetNodes(c.Request.Context(), id, models.PipelineRunAssetNodeListOptions{
		Limit:   limit,
		Cursor:  strings.TrimSpace(c.Query("cursor")),
		AssetID: strings.TrimSpace(c.Query("assetId")),
		NodeID:  strings.TrimSpace(c.Query("nodeId")),
		Status:  strings.TrimSpace(c.Query("status")),
		OrderBy: strings.TrimSpace(c.Query("orderBy")),
	})
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, result)
}

// GetRunCostSummary handles GET /api/v1/pipeline-runs/:id/cost-summary.
func (h *Handler) GetRunCostSummary(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	result, err := h.uc.GetRunCostSummary(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, result)
}

// GetRunWatcherStatus handles GET /api/v1/pipeline-runs/watcher/status.
func (h *Handler) GetRunWatcherStatus(c *gin.Context) {
	state, err := h.uc.GetRunWatcherStatus(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, state)
}

// RetryRun handles POST /api/v1/pipeline-runs/:id/retry.
func (h *Handler) RetryRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	run, err := h.uc.RetryRun(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		mapDeployError(c, err)
		return
	}
	c.JSON(http.StatusCreated, run)
}

// StopRun handles POST /api/v1/pipeline-runs/:id/stop.
func (h *Handler) StopRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.StopRun(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"message": "pipeline run stopped"})
}

// DeleteRun handles DELETE /api/v1/pipeline-runs/:id.
func (h *Handler) DeleteRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.DeleteRun(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// ListDeployments handles GET /api/v1/deployments.
func (h *Handler) ListDeployments(c *gin.Context) {
	items, err := h.uc.ListDeployments(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineDeployment{}
	}
	c.JSON(200, gin.H{"items": items})
}

// GetDeployment handles GET /api/v1/deployments/:id.
func (h *Handler) GetDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	d, err := h.uc.GetDeployment(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if d == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "deployment not found")
		return
	}
	c.JSON(200, d)
}

// DeleteDeployment handles DELETE /api/v1/deployments/:id.
func (h *Handler) DeleteDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}

	if err := h.uc.DeleteDeployment(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.Status(204)
}

func mapDeployError(c *gin.Context, err error) {
	var assetErr *assetvalidation.ValidationError
	if errors.As(err, &assetErr) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, assetErr.Message(), assetErr.Details())
		return
	}
	if errors.Is(err, pipelineUC.ErrTemplateNotFound) {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		return
	}
	if errors.Is(err, pipelineUC.ErrAssetNotFound) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if errors.Is(err, pipelineUC.ErrInvalidArgument) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if errors.Is(err, pipelineUC.ErrExecutionTargetNotFound) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if errors.Is(err, pipelineUC.ErrWorkflowUnavailable) {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, err.Error(), nil)
		return
	}
	httpresp.Internal(c, err.Error())
}

// RetryDeployment handles POST /api/v1/deployments/:id/retry.
func (h *Handler) RetryDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	dep, err := h.uc.RetryDeployment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, dep)
}

// StopDeployment handles POST /api/v1/deployments/:id/stop.
func (h *Handler) StopDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.uc.StopDeployment(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"message": "workflow stopped"})
}

// SaveFromDeployment handles POST /api/v1/deployments/:id/save-template.
func (h *Handler) SaveFromDeployment(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "deployment id is required", nil)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)

	t, err := h.uc.SaveFromDeployment(c.Request.Context(), id, req.Name)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, t)
}

// GetResourceUsage handles GET /api/v1/deployments/:id/resources.
func (h *Handler) GetResourceUsage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "deployment id is required", nil)
		return
	}

	report, err := h.uc.GetResourceUsage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, report)
}

// GetWorkflowResourceUsage handles GET /api/v1/workflows/:name/resources.
func (h *Handler) GetWorkflowResourceUsage(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "workflow name is required", nil)
		return
	}

	report, err := h.uc.GetWorkflowResourceUsage(c.Request.Context(), name)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) || errors.Is(err, pipelineUC.ErrWorkflowUnavailable) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, report)
}

// GetWorkflowNodeResourceUsage handles GET /api/v1/workflows/:name/nodes/:nodeId/resources.
func (h *Handler) GetWorkflowNodeResourceUsage(c *gin.Context) {
	name := strings.TrimSpace(c.Param("name"))
	nodeID := strings.TrimSpace(c.Param("nodeId"))
	if name == "" || nodeID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "workflow name and nodeId are required", nil)
		return
	}

	report, err := h.uc.GetWorkflowNodeResourceUsage(c.Request.Context(), name, nodeID)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) || errors.Is(err, pipelineUC.ErrWorkflowUnavailable) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, report)
}

// DiffTemplates handles GET /api/v1/pipelines/:id1/diff/:id2.
func (h *Handler) DiffTemplates(c *gin.Context) {
	id1 := strings.TrimSpace(c.Param("id"))
	id2 := strings.TrimSpace(c.Param("id2"))
	if id1 == "" || id2 == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "both id1 and id2 are required", nil)
		return
	}

	diff, err := h.uc.DiffTemplates(c.Request.Context(), id1, id2)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrTemplateNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, diff)
}

// RegisterOutput handles POST /api/v1/pipeline-assets.
// Pipeline containers call back to register processing results as new assets.
func (h *Handler) RegisterOutput(c *gin.Context) {
	var req pipelineUC.RegisterPipelineOutputInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if req.DeploymentID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "deployment_id is required", nil)
		return
	}
	asset, err := h.uc.RegisterOutput(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(201, asset)
}

// GetLineage handles GET /api/v1/assets/:id/pipeline-lineage.
func (h *Handler) GetLineage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset id is required", nil)
		return
	}
	lineage, err := h.uc.GetLineage(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, lineage)
}
