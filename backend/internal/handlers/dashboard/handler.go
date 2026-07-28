// Package dashboard hosts HTTP handlers for the DataBrew dashboard.
// Read-only aggregations — no writes, no cross-service orchestration.
package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	dashboardUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/dashboard"
)

// Handler serves the dashboard endpoints.
type Handler struct {
	uc *dashboardUC.Usecase
}

// New constructs a dashboard Handler.
func New(uc *dashboardUC.Usecase) *Handler { return &Handler{uc: uc} }

// DurationDistribution returns the fleet-wide asset-duration histogram + overall
// stats for the dashboard's 数据时长分布 card (CYB-4303).
//
// @Summary      Dashboard duration distribution
// @Description  Fleet-wide duration histogram (5 fixed buckets) + mean/min/max/p50/p90
// @Tags         dashboard
// @Produce      json
// @Param        asset_type query string false "Optional asset_type filter (empty = all types)"
// @Success      200 {object} models.DurationDistribution
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /dashboard/duration-distribution [get]
func (h *Handler) DurationDistribution(c *gin.Context) {
	// asset_type is optional; the usecase trims whitespace. A nonsense value
	// simply returns total_assets=0 rather than 400 — the dashboard is
	// intentionally permissive so a fresh install with no data still renders.
	assetType := c.Query("asset_type")

	dist, err := h.uc.DurationDistribution(c.Request.Context(), assetType)
	if err != nil {
		c.Error(err)
		httpresp.Internal(c, "failed to load duration distribution")
		return
	}
	c.JSON(http.StatusOK, dist)
}
