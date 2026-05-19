package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

// SearchOutboxStatsResponse is returned by GET /api/v1/admin/search/outbox-stats.
type SearchOutboxStatsResponse struct {
	PublishStateCounts map[string]int64 `json:"publish_state_counts"`
	OutboxDLQRows      int64            `json:"outbox_dlq_rows"`
}

// SearchOutboxStats returns read-only counts of asset_events by publish_state
// plus rows in outbox_dlq (archived permanently failed events).
func (h *Handler) SearchOutboxStats(c *gin.Context) {
	if h.events == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"asset_events repository not configured", nil)
		return
	}
	ctx := c.Request.Context()
	counts, err := h.events.PublishStateCounts(ctx)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	var dlqRows int64
	if h.dlq != nil {
		n, err := h.dlq.Count(ctx)
		if err != nil {
			httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
			return
		}
		dlqRows = n
	}
	c.JSON(http.StatusOK, SearchOutboxStatsResponse{
		PublishStateCounts: counts,
		OutboxDLQRows:      dlqRows,
	})
}
