package mcap

import (
	"net/http"
	"strconv"
	"time"

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

// POST /api/v1/mcap-files
func (h *Handler) CreateFile(c *gin.Context) {
	var req struct {
		McapFileID       string                 `json:"mcap_file_id" binding:"required"`
		GCSPath          string                 `json:"gcs_path"`
		McapURI          string                 `json:"mcap_uri"`
		SizeBytes        int64                  `json:"size_bytes"`
		RawHashMD5       string                 `json:"raw_hash_md5"`
		RawHashSHA256    string                 `json:"raw_hash_sha256"`
		IngestState      string                 `json:"ingest_state"`
		FileDurationMs   int64                  `json:"file_duration_ms"`
		StartTimestampNs int64                  `json:"start_timestamp_ns"`
		EndTimestampNs   int64                  `json:"end_timestamp_ns"`
		ChannelCount     int                    `json:"channel_count"`
		ChunkCount       int                    `json:"chunk_count"`
		VendorID         string                 `json:"vendor_id"`
		CollectorID      string                 `json:"collector_id"`
		TaskID           string                 `json:"task_id"`
		DeviceID         string                 `json:"device_id"`
		CameraModel      string                 `json:"camera_model"`
		DataSource       string                 `json:"data_source"`
		LocationID       string                 `json:"location_id"`
		SceneID          string                 `json:"scene_id"`
		EnvironmentID    string                 `json:"environment_id"`
		CollectionMethod string                 `json:"collection_method"`
		Owner            string                 `json:"owner"`
		RetentionTier    string                 `json:"retention_tier"`
		ExpireAt         *time.Time             `json:"expire_at"`
		Metadata         map[string]interface{} `json:"metadata"`
		ProcessState     map[string]string      `json:"process_state"`
		TenantID         string                 `json:"tenant_id"`
		ProjectID        string                 `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	gcsPath := req.GCSPath
	if gcsPath == "" {
		gcsPath = req.McapURI
	}
	if req.IngestState == "" {
		req.IngestState = string(models.IngestStatePending)
	}
	f := &models.McapFile{
		McapFileID:       req.McapFileID,
		GCSPath:          gcsPath,
		SizeBytes:        req.SizeBytes,
		RawHashMD5:       req.RawHashMD5,
		RawHashSHA256:    req.RawHashSHA256,
		IngestState:      models.IngestState(req.IngestState),
		FileDurationMs:   req.FileDurationMs,
		StartTimestampNs: req.StartTimestampNs,
		EndTimestampNs:   req.EndTimestampNs,
		ChannelCount:     req.ChannelCount,
		ChunkCount:       req.ChunkCount,
		VendorID:         req.VendorID,
		CollectorID:      req.CollectorID,
		TaskID:           req.TaskID,
		DeviceID:         req.DeviceID,
		CameraModel:      req.CameraModel,
		DataSource:       req.DataSource,
		LocationID:       req.LocationID,
		SceneID:          req.SceneID,
		EnvironmentID:    req.EnvironmentID,
		CollectionMethod: req.CollectionMethod,
		Owner:            req.Owner,
		RetentionTier:    req.RetentionTier,
		ExpireAt:         req.ExpireAt,
		Metadata:         req.Metadata,
		ProcessState:     req.ProcessState,
		TenantID:         req.TenantID,
		ProjectID:        req.ProjectID,
	}
	if err := h.repo.Set(c.Request.Context(), f); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, f)
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
