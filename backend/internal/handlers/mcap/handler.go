package mcap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type Handler struct {
	repo         repository.McapFileRepository
	tx           repository.TxRunner
	eventRepo    repository.AssetEventRepository
	assetRepo    repository.AssetRepository    // CYB-1217: 1:1 raw_mcap asset creation
	assetTagRepo repository.AssetTagRepository // CYB-3797: auto-extract metadata → tag on ingest
	bytesSrc     BytesSource
	nowFn        func() time.Time
}

func New(repo repository.McapFileRepository) *Handler {
	return &Handler{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

// SetTxRunner wires a transaction runner so MCAP writes and outbox appends can
// commit atomically.
func (h *Handler) SetTxRunner(tx repository.TxRunner) {
	h.tx = tx
}

// SetEventRepo wires optional asset_events appends for MCAP mutations.
func (h *Handler) SetEventRepo(eventRepo repository.AssetEventRepository) {
	h.eventRepo = eventRepo
}

// SetAssetRepo wires an asset repository so CreateFile can insert a
// placeholder raw_mcap asset in the same transaction (CYB-1217: 1:1).
func (h *Handler) SetAssetRepo(assetRepo repository.AssetRepository) {
	h.assetRepo = assetRepo
}

// SetAssetTagRepo wires the asset-tag repository so CreateFile can extract
// known metadata fields (vibecap_tasks / source_platform / location.address)
// into `task` / `source` / `city` tag rows in the same transaction as the
// mcap and raw_mcap-asset writes (CYB-3797). Unset ⇒ the extract step is
// silently skipped, preserving old behavior for callers that haven't wired
// the repo.
func (h *Handler) SetAssetTagRepo(assetTagRepo repository.AssetTagRepository) {
	h.assetTagRepo = assetTagRepo
}

// SetBytesSource wires a byte source for GET /mcap-files/:id/bytes.
// If unset, that endpoint returns 503.
func (h *Handler) SetBytesSource(src BytesSource) {
	h.bytesSrc = src
}

func (h *Handler) withTx(ctx context.Context, fn func(context.Context) error) error {
	if h.tx != nil {
		return h.tx.WithTx(ctx, fn)
	}
	if txRunner, ok := h.repo.(repository.TxRunner); ok {
		return txRunner.WithTx(ctx, fn)
	}
	return fn(ctx)
}

func (h *Handler) appendMcapEvent(ctx context.Context, eventType, mcapFileID, requestID string, payload map[string]any) error {
	if h.eventRepo == nil {
		return nil
	}
	body, _ := json.Marshal(payload)
	return h.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:            eventType,
		AggregateType:        "mcap_file",
		PayloadSchemaVersion: "v1",
		McapFileID:           mcapFileID,
		EventSource:          "backend",
		RequestID:            requestID,
		EventPayload:         body,
	})
}

func (h *Handler) createFileTx(ctx context.Context, f *models.McapFile, requestID string) error {
	return h.withTx(ctx, func(txCtx context.Context) error {
		if err := h.repo.Set(txCtx, f); err != nil {
			return err
		}
		// CYB-1217: create placeholder raw_mcap asset with same ID (1:1
		// extension). The asset will be updated later via POST /api/v1/assets.
		if h.assetRepo != nil {
			now := h.nowFn()
			// CYB-3715: mirror the 7 mcap-file producer-identity fields
			// onto the raw_mcap asset so /queries/run can filter/facet
			// them without joining to mcap_files or asset_tags. source_platform
			// is lifted from metadata.source_platform (CYB-3714 keeps it on
			// tags.source too — both paths coexist).
			var srcPlatform string
			if f.Metadata != nil {
				if v, ok := f.Metadata["source_platform"].(string); ok {
					srcPlatform = v
				}
			}
			placeholder := &models.Asset{
				AssetID:          f.McapFileID,
				McapFileID:       f.McapFileID,
				StartTimestampNs: f.StartTimestampNs,
				EndTimestampNs:   f.EndTimestampNs,
				DurationMs:       f.FileDurationMs,
				Owner:            f.Owner,
				AssetType:        "raw_mcap",
				LifecycleState:   "created",
				RetentionTier:    f.RetentionTier,
				ExpireAt:         f.ExpireAt,
				TenantID:         f.TenantID,
				ProjectID:        f.ProjectID,
				Metadata:         map[string]interface{}{},
				Files:            map[string]string{},
				CameraModel:      f.CameraModel,
				GraceVideoID:     f.GraceVideoID, // CYB-4011: mirror onto raw_mcap asset
				DeviceID:         f.DeviceID,
				CollectorID:      f.CollectorID,
				SceneID:          f.SceneID,
				DataSource:       f.DataSource,
				CollectionMethod: f.CollectionMethod,
				SourcePlatform:   srcPlatform,
				CreatedAt:        now,
				UpdatedAt:        now,
				Version:          1,
			}
			if err := h.assetRepo.InsertNew(txCtx, placeholder); err != nil {
				return err
			}
			// CYB-3797: auto-extract known metadata fields (vibecap_tasks /
			// source_platform / location.address) into task / source / city
			// tags so newly uploaded mcap arrive already tagged, eliminating
			// the "backfill chases moving target" pattern of CYB-3714.
			// Unknown metadata keys are ignored; shape mismatches skip the
			// individual tag with a WARN log but do not abort the mcap
			// create. Only runs when assetTagRepo is wired.
			if h.assetTagRepo != nil {
				for _, tag := range extractMetadataTags(placeholder.AssetID, f.TenantID, f.ProjectID, f.Metadata) {
					if err := h.assetTagRepo.Upsert(txCtx, tag); err != nil {
						return err
					}
				}
			}
			// CYB-3297 Phase D: emit an asset-scoped event so the ES subscriber
			// indexes the placeholder raw_mcap immediately. The mcap_file_created
			// event below carries no asset_id and is dropped by the subscriber, so
			// without this a newly ingested raw_mcap is invisible in search until
			// the next full reindex.
			if h.eventRepo != nil {
				assetBody, _ := json.Marshal(map[string]any{
					"asset_id":     placeholder.AssetID,
					"asset_type":   "raw_mcap",
					"mcap_file_id": placeholder.McapFileID,
				})
				if err := h.eventRepo.Append(txCtx, repository.AssetEventAppendInput{
					EventType:     "asset_created",
					AggregateType: "asset",
					AssetID:       placeholder.AssetID,
					McapFileID:    placeholder.McapFileID,
					TenantID:      placeholder.TenantID,
					ProjectID:     placeholder.ProjectID,
					EventSource:   "backend",
					RequestID:     requestID,
					EventPayload:  assetBody,
				}); err != nil {
					return err
				}
			}
		}
		return h.appendMcapEvent(txCtx, "mcap_file_created", f.McapFileID, requestID, map[string]any{
			"mcap_file_id": f.McapFileID,
			"ingest_state": f.IngestState,
			"gcs_path":     f.GCSPath,
			"size_bytes":   f.SizeBytes,
		})
	})
}

// maxMcapFileIDRetries is the maximum number of attempts to allocate a unique
// auto-generated mcap_file_id before giving up.
const maxMcapFileIDRetries = 16

// uniqueViolationKind classifies a Postgres unique_violation (23505) raised by
// createFileTx into either "hash" (raw_hash_md5 collision — unresolvable by
// picking a new mcap_file_id) or "id" (mcap_file_id / asset_id collision —
// resolvable by retrying with a new auto-generated ID). Returns the empty
// string when err is not a 23505 or the constraint is unrecognized (caller
// should treat as an unclassified server error).
func uniqueViolationKind(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return ""
	}
	switch pgErr.ConstraintName {
	case "uq_mcap_files_hash_md5":
		return "hash"
	case "mcap_files_pkey", "assets_pkey":
		return "id"
	}
	return "other"
}

// POST /api/v1/mcap-files
func (h *Handler) CreateFile(c *gin.Context) {
	var req struct {
		McapFileID       string                 `json:"mcap_file_id"`
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
		GraceVideoID     string                 `json:"grace_video_id"` // CYB-4011
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
	req.McapFileID = strings.TrimSpace(req.McapFileID)
	if req.McapFileID != "" && !id.ValidateMcapFileID(req.McapFileID) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "mcap_file_id must be exactly 8 alphanumeric characters", nil)
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
		GraceVideoID:     req.GraceVideoID, // CYB-4011
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

	autoID := req.McapFileID == ""
	if !autoID {
		err := h.createFileTx(c.Request.Context(), f, c.GetHeader("X-Request-ID"))
		if err != nil {
			// AssetRepo.InsertNew wraps assets_pkey unique_violation into the
			// ErrDuplicateAssetID sentinel, so the assets_pkey case inside
			// uniqueViolationKind is unreachable from the auto-derived raw_mcap
			// asset path. Catch the sentinel explicitly.
			if errors.Is(err, repository.ErrDuplicateAssetID) {
				httpresp.Conflict(c, httpresp.CodeDuplicateMcapFileID, "mcap_file_id already exists", nil)
				return
			}
			switch uniqueViolationKind(err) {
			case "hash":
				httpresp.Conflict(c, httpresp.CodeDuplicateHash, "raw_hash_md5 already exists", nil)
				return
			case "id":
				httpresp.Conflict(c, httpresp.CodeDuplicateMcapFileID, "mcap_file_id already exists", nil)
				return
			}
			httpresp.Internal(c, err.Error())
			return
		}
		c.JSON(http.StatusCreated, f)
		return
	}

	for range maxMcapFileIDRetries {
		gid, genErr := id.GenerateMcapFileID()
		if genErr != nil {
			httpresp.Internal(c, genErr.Error())
			return
		}
		f.McapFileID = gid
		err := h.createFileTx(c.Request.Context(), f, c.GetHeader("X-Request-ID"))
		if err == nil {
			c.JSON(http.StatusCreated, f)
			return
		}
		// Sentinel from AssetRepo means the auto-picked mcap_file_id already
		// has a raw_mcap asset with the same id — retry with a fresh id, same
		// as the pgconn "id" case below.
		if errors.Is(err, repository.ErrDuplicateAssetID) {
			continue
		}
		switch uniqueViolationKind(err) {
		case "hash":
			httpresp.Conflict(c, httpresp.CodeDuplicateHash, "raw_hash_md5 already exists", nil)
			return
		case "id":
			continue
		}
		httpresp.Internal(c, err.Error())
		return
	}
	httpresp.Internal(c, "failed to allocate unique mcap_file_id")
}

// POST /api/v1/mcap/upload/finalize and POST /api/v1/mcap-files/:id/finalize
// Called by SDK after GCS PUT completes. Triggers async summary (indexer-worker).
func (h *Handler) FinalizeUpload(c *gin.Context) {
	mcapFileID := strings.TrimSpace(c.Param("id"))
	if mcapFileID == "" {
		// Fallback: read from body (old route /mcap/upload/finalize)
		var req struct {
			McapFileID string `json:"mcap_file_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid request body", map[string]any{"error": err.Error()})
			return
		}
		mcapFileID = strings.TrimSpace(req.McapFileID)
	}
	if mcapFileID == "" || !id.ValidateMcapFileID(mcapFileID) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "mcap_file_id must be exactly 8 alphanumeric characters", nil)
		return
	}

	// Phase 0: flip state to summarized immediately (no real footer parse yet).
	// Phase 0.5: this triggers Pub/Sub → indexer-worker → real MCAP footer parse.
	if err := h.withTx(c.Request.Context(), func(txCtx context.Context) error {
		if err := h.repo.UpdateIngestState(txCtx, mcapFileID, models.IngestStateSummarized); err != nil {
			return err
		}
		return h.appendMcapEvent(txCtx, "mcap_upload_finalized", mcapFileID, c.GetHeader("X-Request-ID"), map[string]any{
			"mcap_file_id": mcapFileID,
			"ingest_state": models.IngestStateSummarized,
		})
	}); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mcap_file_id": mcapFileID,
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
