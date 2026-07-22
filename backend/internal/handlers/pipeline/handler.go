package pipeline

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	runKernel "github.com/CyberOrigin2077/cyber-databrew/internal/runtimeos/run"
	"github.com/CyberOrigin2077/cyber-databrew/internal/usecase/assetvalidation"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
	"github.com/google/uuid"
)

// BatchSubtaskReconciler materializes batch subtasks as pipeline runs for list views.
type BatchSubtaskReconciler interface {
	ReconcileSubtaskRuns(ctx context.Context, jobID string) error
	ReconcileItemByID(ctx context.Context, itemID string) (string, error)
	// SyncJob force-syncs a batch job's progress (terminal detection + once-only
	// completion notification). Kept on the interface for the reconcile backstop
	// path; the webhook hot path uses AdvanceItemForRun instead.
	SyncJob(ctx context.Context, jobID string) error
	// AdvanceItemForRun is the O(1) webhook hot path — advances one batch
	// child's item + increments the job counter atomically, without scanning
	// the whole batch. Replaces the O(N²) SyncJob-per-webhook cascade that
	// collapsed the DB connection pool on large batches.
	AdvanceItemForRun(ctx context.Context, run *models.PipelineRun) error
}

// Handler bundles the pipeline endpoints.
type Handler struct {
	uc        *pipelineUC.Usecase
	runs      runKernel.Service
	pricing   *pipelineUC.PricingConfig
	batchRuns BatchSubtaskReconciler
}

type runtimeConfigSelectionRequest struct {
	Mode           string `json:"mode"`
	ConfigID       string `json:"configId"`
	Version        int    `json:"version"`
	FileName       string `json:"fileName"`
	Content        string `json:"content"`
	MountPath      string `json:"mountPath"`
	TargetFilename string `json:"targetFilename"`
}

func (r *runtimeConfigSelectionRequest) toUsecase() *pipelineUC.RuntimeConfigSelection {
	if r == nil {
		return nil
	}
	return &pipelineUC.RuntimeConfigSelection{
		Mode:           r.Mode,
		ConfigID:       r.ConfigID,
		Version:        r.Version,
		FileName:       r.FileName,
		Content:        r.Content,
		MountPath:      r.MountPath,
		TargetFilename: r.TargetFilename,
	}
}

// New constructs a Handler. When pricingPath is non-empty the GCP pricing
// YAML is loaded at construction time for cost estimation.
func New(uc *pipelineUC.Usecase, pricingPath string, batchRuns BatchSubtaskReconciler) *Handler {
	var pricing *pipelineUC.PricingConfig
	if strings.TrimSpace(pricingPath) != "" {
		pricing, _ = pipelineUC.LoadPricing(pricingPath)
	}
	return &Handler{uc: uc, runs: runKernel.NewService(uc), pricing: pricing, batchRuns: batchRuns}
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
	page, pageSize := handlers.ParsePageParams(c.Query("page"), c.Query("page_size"))
	// CYB-3390: exclude_auto_drafts=true pushes the "hide auto-named drafts"
	// filter to the DB so pagination reflects the curated set rather than
	// requiring the client to page through auto-draft noise.
	excludeAuto := strings.EqualFold(strings.TrimSpace(c.Query("exclude_auto_drafts")), "true")
	filter := models.PipelineTemplateListFilter{
		Query:             strings.TrimSpace(c.Query("q")),
		Scope:             strings.TrimSpace(c.Query("scope")),
		Sort:              strings.TrimSpace(c.DefaultQuery("sort", "updated_at_desc")),
		Page:              page,
		PageSize:          pageSize,
		ExcludeAutoDrafts: excludeAuto,
	}
	items, total, err := h.uc.ListTemplatesPaged(c.Request.Context(), filter)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineTemplate{}
	}
	c.JSON(200, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
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
		Pipeline map[string]interface{}         `json:"pipeline" binding:"required"`
		Name     string                         `json:"name"`
		AssetIDs []string                       `json:"asset_ids"`
		TargetID string                         `json:"target_id"`
		Config   *runtimeConfigSelectionRequest `json:"configSelection"`
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

	dep, err := h.uc.Deploy(c.Request.Context(), req.Pipeline, req.Name, req.AssetIDs, pipelineUC.DeployOptions{DryRun: dryRun, TargetID: req.TargetID, Owner: middleware.GetUserEmail(c), ConfigSelection: req.Config.toUsecase()})
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
		Name     string                         `json:"name"`
		AssetIDs []string                       `json:"asset_ids"`
		TargetID string                         `json:"target_id"`
		Version  int                            `json:"version"`
		Config   *runtimeConfigSelectionRequest `json:"configSelection"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	dep, err := h.uc.DeployByTemplateID(c.Request.Context(), id, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID, TemplateVersion: req.Version, Owner: middleware.GetUserEmail(c), ConfigSelection: req.Config.toUsecase()})
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(201, dep)
}

// CreateRun handles POST /api/v1/pipeline-runs.
func (h *Handler) CreateRun(c *gin.Context) {
	var req struct {
		Pipeline map[string]interface{}         `json:"pipeline" binding:"required"`
		Name     string                         `json:"name"`
		AssetIDs []string                       `json:"asset_ids"`
		TargetID string                         `json:"target_id"`
		Config   *runtimeConfigSelectionRequest `json:"configSelection"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	run, err := h.runs.CreateRun(c.Request.Context(), req.Pipeline, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID, ConfigSelection: req.Config.toUsecase()})
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
		Name     string                         `json:"name"`
		AssetIDs []string                       `json:"asset_ids"`
		TargetID string                         `json:"target_id"`
		Version  int                            `json:"version"`
		Config   *runtimeConfigSelectionRequest `json:"configSelection"`
	}
	// Body is optional: empty body is fine, but a non-empty body that fails to
	// bind (malformed JSON, wrong content type) is a client error and must not
	// silently fall through to the usecase.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if len(req.AssetIDs) > 1 {
		runs, err := h.runs.CreateRunsByTemplateID(c.Request.Context(), id, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID, TemplateVersion: req.Version, Owner: middleware.GetUserEmail(c), ConfigSelection: req.Config.toUsecase()})
		if err != nil {
			mapDeployError(c, err)
			return
		}
		if runs == nil {
			runs = []models.PipelineRun{}
		}
		c.JSON(http.StatusCreated, gin.H{"items": runs, "total": len(runs)})
		return
	}
	run, err := h.runs.CreateRunByTemplateID(c.Request.Context(), id, req.Name, req.AssetIDs, pipelineUC.DeployOptions{TargetID: req.TargetID, TemplateVersion: req.Version, Owner: middleware.GetUserEmail(c), ConfigSelection: req.Config.toUsecase()})
	if err != nil {
		mapDeployError(c, err)
		return
	}
	c.JSON(http.StatusCreated, run)
}

// ListExecutionTargets handles GET /api/v1/execution-targets.
func (h *Handler) CreateExecutionTarget(c *gin.Context) {
	var t models.ExecutionTarget
	if err := c.ShouldBindJSON(&t); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if err := h.uc.CreateExecutionTarget(c.Request.Context(), &t); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *Handler) UpdateExecutionTarget(c *gin.Context) {
	id := c.Param("id")
	var t models.ExecutionTarget
	if err := c.ShouldBindJSON(&t); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	t.ID = id
	if err := h.uc.UpdateExecutionTarget(c.Request.Context(), &t); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handler) DeleteExecutionTarget(c *gin.Context) {
	id := c.Param("id")
	if err := h.uc.DeleteExecutionTarget(c.Request.Context(), id); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) ListExecutionTargets(c *gin.Context) {
	items, err := h.uc.ListExecutionTargets(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"items": items})
}

// ListRuntimeMounts handles GET /api/v1/pipeline/runtime-mounts.
func (h *Handler) ListRuntimeMounts(c *gin.Context) {
	catalog, err := h.uc.ListRuntimeMounts(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, catalog)
}

// ListRuns handles GET /api/v1/pipeline-runs.
func (h *Handler) ListRuns(c *gin.Context) {
	refreshActive := strings.EqualFold(c.Query("refresh"), "true") || c.Query("refresh") == "1"
	summaryView := strings.EqualFold(c.Query("view"), "summary")
	batchJobID := strings.TrimSpace(c.Query("batchJobId"))
	excludeBatch := strings.EqualFold(c.Query("excludeBatch"), "true") || c.Query("excludeBatch") == "1"
	// CYB-3392b: excludeBatchParents=true keeps batch children (so the ui
	// can show a "跳批次" badge on child rows) but hides the aggregate
	// batch parent row from the mixed "单次执行" tab.
	excludeBatchParents := strings.EqualFold(c.Query("excludeBatchParents"), "true") || c.Query("excludeBatchParents") == "1"
	statusFilter := strings.TrimSpace(c.Query("status"))
	query := strings.TrimSpace(c.Query("q"))
	createdBy := strings.TrimSpace(c.Query("createdBy"))
	pipelineNodeID := strings.TrimSpace(c.Query("pipelineNodeId"))
	nodeStatus := strings.TrimSpace(c.Query("nodeStatus"))
	page, _ := strconv.Atoi(strings.TrimSpace(c.Query("page")))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(c.Query("pageSize")))

	// Hard cap pageSize to prevent runaway queries
	// Default to 50, max 200 (hard cap against runaway queries)
	if pageSize <= 0 {
		pageSize = 50
	} else if pageSize > 200 {
		pageSize = 200
	}
	if page <= 0 {
		page = 1
	}

	var (
		items []models.PipelineRun
		total int
		err   error
	)
	filter := models.PipelineRunListFilter{
		BatchJobID:          batchJobID,
		ExcludeBatch:        excludeBatch,
		ExcludeBatchParents: excludeBatchParents,
		Status:              statusFilter,
		Query:               query,
		CreatedBy:           createdBy,
		PipelineNodeID:      pipelineNodeID,
		NodeStatus:          nodeStatus,
		Page:                page,
		PageSize:            pageSize,
		RefreshActive:       refreshActive,
		SummaryOnly:         summaryView,
	}
	if batchJobID != "" && refreshActive && h.batchRuns != nil {
		_ = h.batchRuns.ReconcileSubtaskRuns(c.Request.Context(), batchJobID)
	}
	items, total, err = h.runs.ListRunSummaries(c.Request.Context(), filter)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineRun{}
	}
	if summaryView {
		resp := gin.H{"items": items, "total": total}
		if page > 0 || pageSize > 0 || batchJobID != "" || excludeBatch {
			if page < 1 {
				page = 1
			}
			if pageSize < 1 {
				pageSize = 20
			}
			resp["page"] = page
			resp["pageSize"] = pageSize
		}
		c.JSON(200, resp)
		return
	}
	for i := range items {
		items[i].TotalEstimatedCost = pipelineUC.ComputeRunCost(&items[i], h.pricing)
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

// GetRunByWorkflowName handles GET /api/v1/pipeline-runs/by-workflow/:workflowName.
func (h *Handler) GetRunByWorkflowName(c *gin.Context) {
	workflowName := strings.TrimSpace(c.Param("workflowName"))
	if workflowName == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "workflowName is required", nil)
		return
	}
	run, err := h.runs.GetRunByWorkflowName(c.Request.Context(), workflowName)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if run == nil {
		run, _ = h.runs.GetRun(c.Request.Context(), workflowName)
	}
	if run == nil && h.batchRuns != nil {
		if runID, err := h.batchRuns.ReconcileItemByID(c.Request.Context(), workflowName); err == nil && runID != "" {
			run, _ = h.runs.GetRun(c.Request.Context(), runID)
		}
		if run == nil {
			run, _ = h.runs.GetRunByWorkflowName(c.Request.Context(), workflowName)
		}
	}
	if run == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
		return
	}
	run.TotalEstimatedCost = pipelineUC.ComputeRunCost(run, h.pricing)
	c.JSON(200, run)
}

// GetRun handles GET /api/v1/pipeline-runs/:id.
func (h *Handler) GetRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	run, err := h.runs.GetRun(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if run == nil && h.batchRuns != nil {
		if runID, err := h.batchRuns.ReconcileItemByID(c.Request.Context(), id); err == nil && runID != "" {
			run, err = h.runs.GetRun(c.Request.Context(), runID)
			if err != nil {
				httpresp.Internal(c, err.Error())
				return
			}
		}
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
	result, err := h.runs.ListRunEvents(c.Request.Context(), id, models.PipelineRunEventListOptions{
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
	result, err := h.runs.ListRunAssetNodes(c.Request.Context(), id, models.PipelineRunAssetNodeListOptions{
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
	result, err := h.runs.GetRunCostSummary(c.Request.Context(), id)
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

// ListRunNodes handles GET /api/v1/runs/:id/nodes.
func (h *Handler) ListRunNodes(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	items, err := h.runs.ListRunNodes(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []models.PipelineRunNode{}
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// ListRunInputs handles GET /api/v1/runs/:id/inputs.
func (h *Handler) ListRunInputs(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	result, err := h.runs.ListRunInputs(c.Request.Context(), id)
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

// ListRunOutputs handles GET /api/v1/runs/:id/outputs.
func (h *Handler) ListRunOutputs(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	result, err := h.runs.ListRunOutputs(c.Request.Context(), id)
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

// ListRunChildren handles GET /api/v1/runs/:id/children.
func (h *Handler) ListRunChildren(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	page, _ := strconv.Atoi(strings.TrimSpace(c.Query("page")))
	pageSize, _ := strconv.Atoi(strings.TrimSpace(c.Query("pageSize")))
	// CYB-3822: optional server-side status filter — batch-export UX selects
	// "仅成功 / 仅失败" and passes Argo TitleCase ("Succeeded" / "Failed") so
	// the repo can drop non-matching rows before serialization instead of
	// forcing the frontend to fetch every child then discard 95%.
	status := strings.TrimSpace(c.Query("status"))
	result, err := h.runs.ListRunChildren(c.Request.Context(), id, models.PipelineRunListFilter{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline run not found")
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	for i := range result.Items {
		result.Items[i].Manifest = nil
		result.Items[i].PipelineJSON = nil
	}
	c.JSON(200, result)
}

// GetRunRuntime handles GET /api/v1/runs/:id/runtime.
func (h *Handler) GetRunRuntime(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	result, err := h.runs.GetRunRuntime(c.Request.Context(), id)
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
	state, err := h.runs.GetRunWatcherStatus(c.Request.Context())
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
	run, err := h.runs.RetryRun(c.Request.Context(), id)
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

// runWebhookRequest is the poke payload sent by the Argo workflow exit hook
// (CYB-3058). The phase is a hint only; DataBrew re-reads the workflow for the
// authoritative state, so a forged/replayed call cannot inject false status.
type runWebhookRequest struct {
	WorkflowName string `json:"workflowName"`
	Namespace    string `json:"namespace"`
	UID          string `json:"uid"`
	Phase        string `json:"phase"`
}

// HandleRunWebhook handles POST /api/v1/pipeline-runs/webhook. It receives an
// Argo workflow exit-hook poke and refreshes the corresponding run from Argo.
// Idempotent: repeated deliveries converge on the authoritative state.
func (h *Handler) HandleRunWebhook(c *gin.Context) {
	var req runWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid webhook payload", nil)
		return
	}
	if strings.TrimSpace(req.WorkflowName) == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "workflowName is required", nil)
		return
	}
	run, err := h.uc.RefreshRunFromWorkflowByName(c.Request.Context(), req.WorkflowName, req.UID)
	if err != nil {
		httpresp.Internal(c, "failed to refresh run from workflow")
		return
	}
	if run == nil {
		httpresp.NotFound(c, "RUN_NOT_FOUND", "no run found for workflow")
		return
	}
	// Advance the child item's ledger row + increment the parent job's counter
	// in one atomic CTE. O(1) per event — batch size does not matter. The old
	// path (SyncJob → full-batch aggregate on every terminal exit hook) was
	// O(batch) per event: a 6k-item batch produced 24s slow queries that
	// swamped the DB connection pool. Finalization + completion notification
	// still happen — the reconcile backstop (StartJobReconciler, 60s) is the
	// single place that judges "batch done" now, so at most one finalize check
	// per job per minute regardless of how many child webhooks arrived.
	if h.batchRuns != nil {
		if err := h.batchRuns.AdvanceItemForRun(c.Request.Context(), run); err != nil {
			slog.Warn("run webhook: advance batch item failed", "runID", run.ID, "err", err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"runId": run.ID, "status": run.Status})
}

// isTerminalRunStatus reports whether an Argo run phase is terminal.
func isTerminalRunStatus(status string) bool {
	switch status {
	case "Succeeded", "Failed", "Error":
		return true
	default:
		return false
	}
}

// RetryRunRuntime handles POST /api/v1/runs/:id/retry.
func (h *Handler) RetryRunRuntime(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	run, err := h.runs.RuntimeRetryRun(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		mapDeployError(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

// ResubmitRun handles POST /api/v1/runs/:id/resubmit.
func (h *Handler) ResubmitRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	run, err := h.runs.ResubmitRun(c.Request.Context(), id)
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

// RerunRun handles POST /api/v1/runs/:id/rerun.
func (h *Handler) RerunRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	run, err := h.runs.RerunRun(c.Request.Context(), id)
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
	if err := h.runs.StopRun(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		mapDeployError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "pipeline run stopped"})
}

// SuspendRun handles POST /api/v1/runs/:id/suspend.
func (h *Handler) SuspendRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.runs.SuspendRun(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		mapDeployError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "run suspend submitted"})
}

// ResumeRun handles POST /api/v1/runs/:id/resume.
func (h *Handler) ResumeRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.runs.ResumeRun(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		mapDeployError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "run resume submitted"})
}

// TerminateRun handles POST /api/v1/runs/:id/terminate.
func (h *Handler) TerminateRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.runs.TerminateRun(c.Request.Context(), id); err != nil {
		if errors.Is(err, pipelineUC.ErrDeploymentNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		mapDeployError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "run terminate submitted"})
}

// DeleteRun handles DELETE /api/v1/pipeline-runs/:id.
func (h *Handler) DeleteRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "id is required", nil)
		return
	}
	if err := h.runs.DeleteRun(c.Request.Context(), id); err != nil {
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
