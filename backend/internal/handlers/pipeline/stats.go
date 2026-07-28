package pipeline

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

// GetStats handles GET /api/v1/pipelines/stats
// Returns user's pipeline usage statistics for smart grouping (e.g. "most used")
func (h *Handler) GetStats(c *gin.Context) {
	userEmail := middleware.GetUserEmail(c)
	if userEmail == "" {
		httpresp.Unauthorized(c, httpresp.CodeUnauthorized, "user not authenticated")
		return
	}

	// Parse window query param (default: "30d")
	window := c.DefaultQuery("window", "30d")
	if !isValidWindow(window) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid window parameter", map[string]any{
			"valid_windows": []string{"7d", "30d", "60d", "90d"},
		})
		return
	}

	stats, err := h.uc.GetUserPipelineStats(c.Request.Context(), userEmail, window)
	if err != nil {
		httpresp.Internal(c, "failed to fetch pipeline stats: "+err.Error())
		return
	}

	c.JSON(200, stats)
}

// isValidWindow validates the time window parameter
func isValidWindow(window string) bool {
	validWindows := map[string]bool{
		"7d":  true,
		"30d": true,
		"60d": true,
		"90d": true,
	}
	return validWindows[window]
}

// UpdatePipeline handles PUT /api/v1/pipelines/:id with version conflict detection
func (h *Handler) UpdatePipeline(c *gin.Context) {
	pipelineID := c.Param("id")
	if pipelineID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "pipeline id is required", nil)
		return
	}

	var req struct {
		Pipeline    map[string]interface{} `json:"pipeline"`
		BaseVersion int                    `json:"baseVersion"`
		Note        string                 `json:"note,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	role, _ := c.Get(middleware.CtxKeyRole)
	isAdmin, _ := role.(string)
	updated, err := h.uc.UpdateTemplate(
		c.Request.Context(),
		pipelineID,
		req.Pipeline,
		req.BaseVersion,
		middleware.GetUserEmail(c),
		isAdmin == "admin" || middleware.GetUserEmail(c) == "sdk",
	)
	if err != nil {
		switch {
		case errors.Is(err, pipelineUC.ErrVersionConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error":          "version_conflict",
				"message":        "Pipeline has been modified by another user",
				"currentVersion": updated.Version,
				"requestedBase":  req.BaseVersion,
			})
		case errors.Is(err, pipelineUC.ErrTemplateNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "pipeline not found")
		case errors.Is(err, pipelineUC.ErrProdLocked), errors.Is(err, pipelineUC.ErrTemplateNotOwned):
			httpresp.Error(c, http.StatusForbidden, httpresp.CodeInvalidArgument, err.Error(), nil)
		case errors.Is(err, pipelineUC.ErrInvalidArgument):
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		default:
			httpresp.Internal(c, "failed to update pipeline: "+err.Error())
		}
		return
	}
	c.JSON(http.StatusCreated, updated)
}

// PipelineStatsResponse is the response body for GetStats
type PipelineStatsResponse struct {
	UserStats       *models.PipelineUserStats        `json:"userStats"`
	Recommendations []*models.PipelineRecommendation `json:"recommendations"`
	FetchedAt       time.Time                        `json:"fetchedAt"`
}
