package pipeline

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// CreateBatchRun handles POST /api/v1/runs/batch
func (h *Handler) StopBatchRun(c *gin.Context) {
	batchID := strings.TrimSpace(c.Param("batchId"))
	if batchID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "batchId is required", nil)
		return
	}

	owner := middleware.GetUserEmail(c)
	stopped, failed, err := h.uc.StopBatchRuns(c.Request.Context(), batchID, owner)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "batch stop initiated", "stopped": stopped, "failed": failed})
}

func (h *Handler) CreateBatchRun(c *gin.Context) {
	var req struct {
		TemplateID     string   `json:"template_id"`
		AssetIDs       []string `json:"asset_ids"`
		TargetID       string   `json:"target_id"`
		Version        int      `json:"version"`
		Name           string   `json:"name"`
		MaxConcurrency int      `json:"max_concurrency"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.TemplateID) == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "template_id is required", nil)
		return
	}
	if len(req.AssetIDs) == 0 {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset_ids must not be empty", nil)
		return
	}

	if req.TargetID == "" {
		req.TargetID = "default"
	}

	owner := middleware.GetUserEmail(c)
	job, err := h.uc.CreateBatchJob(c.Request.Context(), req.TemplateID, req.Name, req.AssetIDs, req.TargetID, req.Version, req.MaxConcurrency, owner)
	if err != nil {
		if errors.Is(err, pipelineUC.ErrTemplateNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		if errors.Is(err, pipelineUC.ErrInvalidArgument) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"batch_id":    job.ID,
		"status":      job.Status,
		"asset_count": job.TotalCount,
		"template_id": job.TemplateID,
		"created_at":  job.CreatedAt,
	})
}

// GetBatchStatus handles GET /api/v1/runs/batch/:batchId
func (h *Handler) GetBatchStatus(c *gin.Context) {
	batchID := strings.TrimSpace(c.Param("batchId"))
	if batchID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "batchId is required", nil)
		return
	}

	job, err := h.uc.GetBatchJobStatus(c.Request.Context(), batchID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if job == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "batch job not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"batch_id":        job.ID,
		"status":          job.Status,
		"name":            job.Name,
		"asset_count":     job.TotalCount,
		"completed_count": job.CompletedCount,
		"failed_count":    job.FailedCount,
		"template_id":     job.TemplateID,
		"created_at":      job.CreatedAt,
	})
}

// ListBatchDLQ handles GET /api/v1/runs/batch/:batchId/dlq — the batch's
// dead-lettered items (status=failed) with their recorded reasons (CYB-3678).
func (h *Handler) ListBatchDLQ(c *gin.Context) {
	batchID := c.Param("batchId")
	items, err := h.uc.ListBatchDLQ(c.Request.Context(), batchID)
	if err != nil {
		httpresp.Internal(c, "failed to list batch dlq")
		return
	}
	type dlqItem struct {
		ID           string `json:"id"`
		AssetID      string `json:"asset_id"`
		ErrorMessage string `json:"error_message,omitempty"`
		WorkflowName string `json:"workflow_name,omitempty"`
	}
	out := make([]dlqItem, 0, len(items))
	for _, it := range items {
		d := dlqItem{ID: it.ID, AssetID: it.AssetID}
		if it.ErrorMessage != nil {
			d.ErrorMessage = *it.ErrorMessage
		}
		if it.WorkflowName != nil {
			d.WorkflowName = *it.WorkflowName
		}
		out = append(out, d)
	}
	c.JSON(200, gin.H{"batch_id": batchID, "count": len(out), "items": out})
}

// RetryBatchDLQ handles POST /api/v1/runs/batch/:batchId/dlq:retry — revives
// the batch's failed items (fresh attempt cap) and kicks the submitter. The
// revived items rejoin the NORMAL dispatch pipeline and are subject to the
// same per-cluster governor — a mass retry can never become a storm.
func (h *Handler) RetryBatchDLQ(c *gin.Context) {
	batchID := c.Param("batchId")
	n, err := h.uc.RetryBatchDLQ(c.Request.Context(), batchID)
	if err != nil {
		httpresp.Internal(c, "failed to retry batch dlq")
		return
	}
	c.JSON(200, gin.H{"batch_id": batchID, "revived": n})
}
