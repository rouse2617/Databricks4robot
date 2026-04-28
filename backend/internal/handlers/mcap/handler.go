package mcap

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type Handler struct {
	repo repository.McapFileRepository
}

func New(repo repository.McapFileRepository) *Handler {
	return &Handler{repo: repo}
}

// POST /api/v1/mcap/upload/finalize
// Called by SDK after GCS PUT completes. Triggers async summary (indexer-worker).
func (h *Handler) FinalizeUpload(c *gin.Context) {
	var req struct {
		McapFileID string `json:"mcap_file_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	// Phase 0: flip state to summarized immediately (no real footer parse yet).
	// Phase 0.5: this triggers Pub/Sub → indexer-worker → real MCAP footer parse.
	if err := h.repo.UpdateIngestState(c.Request.Context(), req.McapFileID, models.IngestStateSummarized); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mcap_file_id": req.McapFileID,
		"ingest_state": models.IngestStateSummarized,
	})
}

// GET /api/v1/mcap/:id/messages
// Placeholder — real iteration requires go-mcap; deferred to Phase 0.5.
func (h *Handler) IterMessages(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"messages": []any{},
		"note":     "MCAP message iteration available in Phase 0.5",
	})
}

// GET /api/v1/mcap-files
func (h *Handler) ListFiles(c *gin.Context) {
	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	ingestState := c.Query("ingest_state")
	owner := c.Query("owner")

	items, total, err := h.repo.List(c.Request.Context(), page, pageSize, ingestState, owner)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []*models.McapFile{}
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GET /api/v1/mcap-files/:id
func (h *Handler) GetFile(c *gin.Context) {
	f, err := h.repo.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if f == nil {
		httpresp.NotFound(c, "MCAP_FILE_NOT_FOUND", "mcap file not found")
		return
	}
	c.JSON(http.StatusOK, f)
}
