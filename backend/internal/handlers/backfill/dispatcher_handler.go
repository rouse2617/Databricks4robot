package backfill

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	uc "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/backfill"
)

// CYB-3679 — dispatcher online-tuning endpoints.

// GetDispatcherStatus handles GET /api/v1/dispatcher/clusters:
// configured-vs-effective per cluster, with suppression reason.
func (h *Handler) GetDispatcherStatus(c *gin.Context) {
	statuses, err := h.uc.DispatcherStatus(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"clusters": statuses})
}

// PutDispatcherConfig handles PUT /api/v1/dispatcher/clusters/:cluster.
func (h *Handler) PutDispatcherConfig(c *gin.Context) {
	var req struct {
		MaxConcurrency int     `json:"max_concurrency"`
		SubmitBatch    int     `json:"submit_batch"`
		RatePerSec     float64 `json:"rate_per_sec"`
		Paused         bool    `json:"paused"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	cfg := &models.DispatcherConfig{
		ClusterID:      c.Param("cluster"),
		MaxConcurrency: req.MaxConcurrency,
		SubmitBatch:    req.SubmitBatch,
		RatePerSec:     req.RatePerSec,
		Paused:         req.Paused,
		UpdatedBy:      middleware.GetUserEmail(c),
	}
	if err := h.uc.SaveDispatcherConfig(c.Request.Context(), cfg); err != nil {
		if errors.Is(err, uc.ErrInvalidDispatcherConfig) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"saved": cfg})
}

// DeleteDispatcherConfig handles DELETE /api/v1/dispatcher/clusters/:cluster
// (back to compiled defaults).
func (h *Handler) DeleteDispatcherConfig(c *gin.Context) {
	if err := h.uc.DeleteDispatcherConfig(c.Request.Context(), c.Param("cluster")); err != nil {
		if errors.Is(err, uc.ErrInvalidDispatcherConfig) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"deleted": c.Param("cluster")})
}
