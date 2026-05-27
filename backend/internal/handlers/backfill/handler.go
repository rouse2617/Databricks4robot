package backfill

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
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
		Name       string   `json:"name" binding:"required"`
		TemplateID string   `json:"templateId" binding:"required"`
		AssetIDs   []string `json:"assetIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	job, err := h.uc.CreateBackfill(c.Request.Context(), req.Name, req.TemplateID, req.AssetIDs)
	if err != nil {
		httpresp.Internal(c, err.Error())
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

	job, items, err := h.uc.GetJob(c.Request.Context(), id)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if job == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "backfill job not found")
		return
	}

	if items == nil {
		items = []models.BackfillItem{}
	}
	c.JSON(200, gin.H{"job": job, "items": items})
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
		mapError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "retrying"})
}

func mapError(c *gin.Context, err error) {
	if errors.Is(err, errors.New("not found")) {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		return
	}
	httpresp.Internal(c, err.Error())
}
