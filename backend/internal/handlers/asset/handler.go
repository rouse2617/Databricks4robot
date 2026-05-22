package asset

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
	"github.com/CyberOrigin2077/cyber-databrew/internal/validate"
)

type Handler struct {
	uc           *assetUC.Usecase
	deliveryRepo repository.DeliveryRepository
	mcapRepo     repository.McapFileRepository
	pg           *postgres.Client
}

func New(uc *assetUC.Usecase, deliveryRepo repository.DeliveryRepository) *Handler {
	return &Handler{uc: uc, deliveryRepo: deliveryRepo}
}

// SetMcapRepo wires the mcap file repository for endpoints that need to join
// asset metadata with the underlying MCAP object (e.g. /assets/:id/mcap-locator).
// Optional: when nil, those endpoints return 503.
func (h *Handler) SetMcapRepo(repo repository.McapFileRepository) {
	h.mcapRepo = repo
}

// SetPG wires the postgres client for endpoints that need to run custom SQL
// queries (e.g. /assets/:id/lineage).
func (h *Handler) SetPG(pg *postgres.Client) {
	h.pg = pg
}

// Get returns a single asset by ID.
// @Summary      Get asset
// @Description  Get asset by ID
// @Tags         assets
// @Produce      json
// @Param        id path string true "Asset ID"
// @Success      200 {object} models.Asset
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	// Use GetAll so soft-deleted (archived) assets remain accessible,
	// matching the API contract: "assets still accessible via GET after soft delete".
	a, err := h.uc.GetAll(c.Request.Context(), assetID)
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, a)
}

// List returns assets with optional filters and pagination.
// Deprecated: asset list queries should use POST /api/v1/queries/run.
func (h *Handler) List(c *gin.Context) {
	filterStrs := c.QueryArray("filter")
	page, pageSize := handlers.ParsePageParams(c.Query("page"), c.Query("page_size"))
	sortBy := c.DefaultQuery("sort_by", "-created_at")

	// Backward compatibility: no filter params → use mcap_file_id.
	if len(filterStrs) == 0 {
		mcapFileID := c.Query("mcap_file_id")
		if mcapFileID != "" {
			filterStrs = append(filterStrs, "mcap_file_id:eq:"+mcapFileID)
		}
	}

	// Parse and validate filters.
	filters, err := filter.ParseFilters(filterStrs)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidFilter, err.Error(), nil)
		return
	}
	if err := filter.ValidateFilters(filters); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidFilter, err.Error(), nil)
		return
	}

	// Build WHERE clause.
	wc, err := filter.BuildWhereClause(filters, 1)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidFilter, err.Error(), nil)
		return
	}

	orderBy, _, err := filter.ResolveSortBy(sortBy, 1)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidFilter, err.Error(), nil)
		return
	}

	items, total, err := h.uc.ListWithFilters(c.Request.Context(), wc.SQL, wc.Args, page, pageSize, orderBy)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

type eventQuery struct {
	beforeSeq *int64
	afterSeq  *int64
	startTime *time.Time
	endTime   *time.Time
	limit     int
}

func parseEventQuery(c *gin.Context) (eventQuery, error) {
	beforeSeq, err := parseOptionalInt64(c.Query("cursor"))
	if err != nil {
		return eventQuery{}, fmt.Errorf("invalid cursor: %w", err)
	}
	if beforeAlias := c.Query("before_event_seq"); beforeAlias != "" {
		beforeSeq, err = parseOptionalInt64(beforeAlias)
		if err != nil {
			return eventQuery{}, fmt.Errorf("invalid before_event_seq: %w", err)
		}
	}
	afterSeq, err := parseOptionalInt64(c.Query("after_event_seq"))
	if err != nil {
		return eventQuery{}, fmt.Errorf("invalid after_event_seq: %w", err)
	}
	startTime, err := parseOptionalRFC3339(c.Query("start_time"))
	if err != nil {
		return eventQuery{}, fmt.Errorf("invalid start_time: %w", err)
	}
	endTime, err := parseOptionalRFC3339(c.Query("end_time"))
	if err != nil {
		return eventQuery{}, fmt.Errorf("invalid end_time: %w", err)
	}
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return eventQuery{}, fmt.Errorf("start_time must be <= end_time")
	}
	limit, err := parseBoundedInt(c.Query("limit"), 50, 1, 200)
	if err != nil {
		return eventQuery{}, fmt.Errorf("invalid limit: %w", err)
	}
	return eventQuery{
		beforeSeq: beforeSeq,
		afterSeq:  afterSeq,
		startTime: startTime,
		endTime:   endTime,
		limit:     limit,
	}, nil
}

func parseOptionalRFC3339(v string) (*time.Time, error) {
	raw := strings.TrimSpace(v)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func writeEventList(c *gin.Context, res *assetUC.ListEventsResult, limit int) {
	resp := gin.H{"items": res.Items, "limit": limit}
	if res.NextCursor != nil {
		resp["next_cursor"] = *res.NextCursor
	}
	c.JSON(200, resp)
}

// ListEvents returns the asset event timeline ordered by event_seq DESC.
// @Summary      List asset events
// @Description  List generic asset events with optional event_type/algo_key filters
// @Tags         assets
// @Produce      json
// @Param        id               path  string true  "Asset ID"
// @Param        event_type       query []string false "Repeated event type filter; supports wildcard suffix like algo_*"
// @Param        algo_key         query string false "Filter event_payload.algo_key"
// @Param        cursor           query int64  false "Fetch older events with event_seq < cursor"
// @Param        before_event_seq query int64  false "Alias of cursor"
// @Param        after_event_seq  query int64  false "Fetch events with event_seq > after_event_seq"
// @Param        start_time       query string false "Lower bound of occurred_at (RFC3339)"
// @Param        end_time         query string false "Upper bound of occurred_at (RFC3339)"
// @Param        limit            query int    false "Page size" default(50)
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id}/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	q, err := parseEventQuery(c)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid event query", map[string]any{"error": err.Error()})
		return
	}

	var eventTypes []string
	var eventTypePatterns []string
	for _, raw := range c.QueryArray("event_type") {
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "*") {
			eventTypePatterns = append(eventTypePatterns, strings.ReplaceAll(raw, "*", "%"))
			continue
		}
		eventTypes = append(eventTypes, raw)
	}

	res, err := h.uc.ListEvents(c.Request.Context(), assetID, assetUC.ListEventsInput{
		EventTypes:        eventTypes,
		EventTypePatterns: eventTypePatterns,
		AlgoKey:           c.Query("algo_key"),
		BeforeEventSeq:    q.beforeSeq,
		AfterEventSeq:     q.afterSeq,
		StartTime:         q.startTime,
		EndTime:           q.endTime,
		Limit:             q.limit,
	})
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	writeEventList(c, res, q.limit)
}

// GetLineage returns the upstream/downstream lineage of an asset.
func (h *Handler) GetLineage(c *gin.Context) {
	assetID := c.Param("id")
	if assetID == "" || !id.ValidateAssetID(assetID) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "valid asset_id is required", nil)
		return
	}

	if h.pg == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available", nil)
		return
	}

	type algoEntry struct {
		AlgoName    string `json:"algo_name"`
		AlgoVersion string `json:"algo_version"`
		Status      string `json:"status"`
		RunID       string `json:"run_id,omitempty"`
		OutputURI   string `json:"output_uri,omitempty"`
	}
	type deliveryEntry struct {
		DeliveryID  string `json:"delivery_id"`
		CustomerID  string `json:"customer_id"`
		DeliveredAt string `json:"delivered_at,omitempty"`
	}
	type evalEntry struct {
		EvalName    string  `json:"eval_name"`
		MetricKey   string  `json:"metric_key"`
		MetricValue float64 `json:"metric_value,omitempty"`
	}

	ctx := c.Request.Context()

	// Upstream: MCAP file
	var mcapFileID, mcapURI, ingestState string
	err := h.pg.QueryRow(ctx, `
		SELECT mcap_file_id, COALESCE(storage_uri,''), COALESCE(ingest_state,'')
		FROM mcap_files
		WHERE mcap_file_id = (SELECT mcap_file_id FROM assets WHERE asset_id = $1)
	`, assetID).Scan(&mcapFileID, &mcapURI, &ingestState)
	upstream := gin.H{}
	if err == nil && mcapFileID != "" {
		upstream["mcap_file_id"] = mcapFileID
		upstream["mcap_uri"] = mcapURI
		upstream["ingest_state"] = ingestState
	}

	// Downstream: algo results
	algoRows, _ := h.pg.Query(ctx, `
		SELECT algo_name, algo_version, status, COALESCE(run_id,''), COALESCE(output_uri,'')
		FROM asset_algo_latest
		WHERE asset_id = $1
		ORDER BY algo_name
	`, assetID)
	algos := []algoEntry{}
	if algoRows != nil {
		defer algoRows.Close()
		for algoRows.Next() {
			var a algoEntry
			if err := algoRows.Scan(&a.AlgoName, &a.AlgoVersion, &a.Status, &a.RunID, &a.OutputURI); err == nil {
				algos = append(algos, a)
			}
		}
	}

	// Downstream: deliveries
	delRows, _ := h.pg.Query(ctx, `
		SELECT d.delivery_id, d.customer_id, d.delivered_at
		FROM delivery_items di
		JOIN deliveries d ON d.delivery_id = di.delivery_id
		WHERE di.asset_id = $1
		ORDER BY d.delivered_at DESC
		LIMIT 20
	`, assetID)
	deliveries := []deliveryEntry{}
	if delRows != nil {
		defer delRows.Close()
		for delRows.Next() {
			var d deliveryEntry
			var deliveredAt *time.Time
			if err := delRows.Scan(&d.DeliveryID, &d.CustomerID, &deliveredAt); err == nil {
				if deliveredAt != nil {
					d.DeliveredAt = deliveredAt.Format(time.RFC3339)
				}
				deliveries = append(deliveries, d)
			}
		}
	}

	// Downstream: eval results
	evalRows, _ := h.pg.Query(ctx, `
		SELECT eval_name, metric_key, COALESCE(metric_value,0)
		FROM asset_eval_results
		WHERE asset_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, assetID)
	evals := []evalEntry{}
	if evalRows != nil {
		defer evalRows.Close()
		for evalRows.Next() {
			var e evalEntry
			if err := evalRows.Scan(&e.EvalName, &e.MetricKey, &e.MetricValue); err == nil {
				evals = append(evals, e)
			}
		}
	}

	c.JSON(200, gin.H{
		"asset_id": assetID,
		"upstream": upstream,
		"downstream": gin.H{
			"algo_results": algos,
			"deliveries":   deliveries,
			"eval_results": evals,
		},
	})
}

// Timeline is an alias of ListEvents for clients that consume a timeline-named
// endpoint while preserving the same query/filter semantics.
func (h *Handler) Timeline(c *gin.Context) {
	h.ListEvents(c)
}

// ListGlobalEvents returns recent events across all assets for operational
// visibility. Falls back to the last 24h when no time filter is provided.
func (h *Handler) ListGlobalEvents(c *gin.Context) {
	q, err := parseEventQuery(c)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid event query", map[string]any{"error": err.Error()})
		return
	}
	if q.limit <= 0 {
		q.limit = 100
	}

	var eventTypes []string
	var eventTypePatterns []string
	for _, raw := range c.QueryArray("event_type") {
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "*") {
			eventTypePatterns = append(eventTypePatterns, strings.ReplaceAll(raw, "*", "%"))
			continue
		}
		eventTypes = append(eventTypes, raw)
	}

	// Default to last 24h when no explicit time range is provided.
	startTime := q.startTime
	if startTime == nil && q.endTime == nil {
		t := time.Now().UTC().Add(-24 * time.Hour)
		startTime = &t
	}

	res, err := h.uc.ListGlobalEvents(c.Request.Context(), assetUC.ListEventsInput{
		EventTypes:        eventTypes,
		EventTypePatterns: eventTypePatterns,
		BeforeEventSeq:    q.beforeSeq,
		AfterEventSeq:     q.afterSeq,
		StartTime:         startTime,
		EndTime:           q.endTime,
		Limit:             q.limit,
	})
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	writeEventList(c, res, q.limit)
}

// UpsertTag creates or updates a single tag on an asset.
// @Summary      Upsert asset tag
// @Description  Create or update one tag entry under asset_tags and append tag_upserted
// @Tags         assets
// @Accept       json
// @Produce      json
// @Param        id   path string true "Asset ID"
// @Param        body body object true "Tag upsert request"
// @Success      200 {object} models.Asset
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      422 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id}/tags [post]
func (h *Handler) UpsertTag(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	var req struct {
		Key           string `json:"key" binding:"required" label:"标签 Key"`
		Value         string `json:"value" binding:"required" label:"标签值"`
		SourceType    string `json:"source_type"`
		SourceName    string `json:"source_name"`
		SourceVersion string `json:"source_version"`
		RunID         string `json:"run_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	a, err := h.uc.UpsertTag(c.Request.Context(), assetID, assetUC.UpsertTagInput{
		Key:           req.Key,
		Value:         req.Value,
		SourceType:    req.SourceType,
		SourceName:    req.SourceName,
		SourceVersion: req.SourceVersion,
		RunID:         req.RunID,
	})
	if err != nil {
		switch {
		case errors.Is(err, assetUC.ErrNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		case errors.Is(err, assetUC.ErrInvalidTag):
			httpresp.Unprocessable(c, httpresp.CodeInvalidTag, err.Error(), nil)
		case errors.Is(err, assetUC.ErrTagSourceInvalid):
			httpresp.Unprocessable(c, httpresp.CodeTagSourceInvalid, err.Error(), nil)
		case errors.Is(err, assetUC.ErrTagImmutable):
			httpresp.Conflict(c, httpresp.CodeTagImmutable, err.Error(), nil)
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(200, a)
}

// DeleteTag removes a single tag from an asset.
// @Summary      Delete asset tag
// @Description  Delete one tag entry under asset_tags and append tag_deleted when present
// @Tags         assets
// @Produce      json
// @Param        id  path string true "Asset ID"
// @Param        key path string true "Tag key"
// @Success      200 {object} models.Asset
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id}/tags/{key} [delete]
func (h *Handler) DeleteTag(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	a, err := h.uc.DeleteTag(c.Request.Context(), assetID, c.Param("key"), c.Query("source_type"))
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, a)
}

// ListTagHistory returns tag_upserted and tag_deleted events for an asset.
// @Summary      List asset tag history
// @Description  List tag-specific events sourced from asset_events
// @Tags         assets
// @Produce      json
// @Param        id               path  string true  "Asset ID"
// @Param        cursor           query int64  false "Fetch older events with event_seq < cursor"
// @Param        before_event_seq query int64  false "Alias of cursor"
// @Param        after_event_seq  query int64  false "Fetch events with event_seq > after_event_seq"
// @Param        limit            query int    false "Page size" default(50)
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id}/tags/history [get]
func (h *Handler) ListTagHistory(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	q, err := parseEventQuery(c)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid event query", map[string]any{"error": err.Error()})
		return
	}
	res, err := h.uc.ListTagHistory(c.Request.Context(), assetID, assetUC.ListEventsInput{
		BeforeEventSeq: q.beforeSeq,
		AfterEventSeq:  q.afterSeq,
		Limit:          q.limit,
	})
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	writeEventList(c, res, q.limit)
}

// Create creates a new asset.
// @Summary      Create asset
// @Description  Create a new asset segment from an MCAP file
// @Tags         assets
// @Accept       json
// @Produce      json
// @Param        body body object true "Create asset request"
// @Success      201 {object} models.Asset
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      422 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets [post]
func (h *Handler) Create(c *gin.Context) {
	var req struct {
		AssetID             string                 `json:"asset_id" label:"资产ID"`
		LogicalAssetID      string                 `json:"logical_asset_id" label:"逻辑资产ID"`
		McapFileID          string                 `json:"mcap_file_id" binding:"required" label:"MCAP文件ID"`
		StartTimestampNs    int64                  `json:"start_timestamp_ns" binding:"required,gt=0" label:"起始时间戳"`
		EndTimestampNs      int64                  `json:"end_timestamp_ns" binding:"required,gt=0" label:"结束时间戳"`
		Reviewer            string                 `json:"reviewer" binding:"required" label:"审核人"`
		Owner               string                 `json:"owner" label:"所有者"`
		SegType             string                 `json:"type" label:"片段类型"`
		AssetType           string                 `json:"asset_type" label:"资产类型"`
		Status              string                 `json:"status" label:"旧生命周期状态"`
		LifecycleState      string                 `json:"lifecycle_state" label:"新生命周期状态"`
		Env                 string                 `json:"env" label:"环境"`
		Task                string                 `json:"task" label:"任务"`
		Tags                map[string]string      `json:"tags"`
		Files               map[string]string      `json:"files"`
		Metadata            map[string]interface{} `json:"metadata"`
		LifecycleMeta       map[string]interface{} `json:"lifecycle_meta"`
		RetentionTier       string                 `json:"retention_tier"`
		ExpireAt            *time.Time             `json:"expire_at"`
		StorageURI          string                 `json:"storage_uri"`
		ThumbURI            string                 `json:"thumb_uri"`
		AssetLevel          int                    `json:"asset_level"`
		ParentAssetID       string                 `json:"parent_asset_id"`
		RootAssetID         string                 `json:"root_asset_id"`
		SegmentIndex        *int                   `json:"segment_index"`
		ParentStartOffsetMs *int64                 `json:"parent_start_offset_ms"`
		ParentEndOffsetMs   *int64                 `json:"parent_end_offset_ms"`
		SplitMethod         string                 `json:"split_method"`
		SplitAlgoName       string                 `json:"split_algo_name"`
		SplitAlgoVersion    string                 `json:"split_algo_version"`
		SplitRunID          string                 `json:"split_run_id"`
		SplitReason         string                 `json:"split_reason"`
		DeliveryCount       int                    `json:"delivery_count"`
		LastDeliveredAt     *time.Time             `json:"last_delivered_at"`
		LastDeliveredTo     string                 `json:"last_delivered_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if valErr := validate.ValidateStruct(&req); valErr != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, valErr.Error(), nil)
		return
	}
	a, err := h.uc.Create(c.Request.Context(), assetUC.CreateInput{
		AssetID:             req.AssetID,
		LogicalAssetID:      req.LogicalAssetID,
		McapFileID:          req.McapFileID,
		StartTimestampNs:    req.StartTimestampNs,
		EndTimestampNs:      req.EndTimestampNs,
		Reviewer:            req.Reviewer,
		Owner:               req.Owner,
		SegType:             req.SegType,
		AssetType:           req.AssetType,
		Status:              req.Status,
		LifecycleState:      req.LifecycleState,
		Env:                 req.Env,
		Task:                req.Task,
		Tags:                req.Tags,
		Files:               req.Files,
		Metadata:            req.Metadata,
		LifecycleMeta:       req.LifecycleMeta,
		RetentionTier:       req.RetentionTier,
		ExpireAt:            req.ExpireAt,
		StorageURI:          req.StorageURI,
		ThumbURI:            req.ThumbURI,
		AssetLevel:          req.AssetLevel,
		ParentAssetID:       req.ParentAssetID,
		RootAssetID:         req.RootAssetID,
		SegmentIndex:        req.SegmentIndex,
		ParentStartOffsetMs: req.ParentStartOffsetMs,
		ParentEndOffsetMs:   req.ParentEndOffsetMs,
		SplitMethod:         req.SplitMethod,
		SplitAlgoName:       req.SplitAlgoName,
		SplitAlgoVersion:    req.SplitAlgoVersion,
		SplitRunID:          req.SplitRunID,
		SplitReason:         req.SplitReason,
		DeliveryCount:       req.DeliveryCount,
		LastDeliveredAt:     req.LastDeliveredAt,
		LastDeliveredTo:     req.LastDeliveredTo,
	})
	if err != nil {
		switch {
		case errors.Is(err, assetUC.ErrMcapFileIDRequired), errors.Is(err, assetUC.ErrInvalidRange):
			httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), nil)
		case errors.Is(err, assetUC.ErrInvalidTag):
			httpresp.Unprocessable(c, httpresp.CodeInvalidTag, err.Error(), nil)
		case errors.Is(err, assetUC.ErrLogicalAssetNotFound), errors.Is(err, assetUC.ErrLogicalAssetTypeMismatch):
			httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), nil)
		case errors.Is(err, assetUC.ErrInvalidMcapFileID), errors.Is(err, assetUC.ErrMcapFileNotFound):
			httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		case errors.Is(err, assetUC.ErrInvalidAssetID):
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		case errors.Is(err, assetUC.ErrAssetIDTaken):
			httpresp.Conflict(c, httpresp.CodeDuplicateAssetID, err.Error(), nil)
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, a)
}

// PATCH /api/v1/assets/:id
func (h *Handler) Update(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	var req struct {
		Status         *string           `json:"status"`
		LifecycleState *string           `json:"lifecycle_state"`
		Reviewer       *string           `json:"reviewer"`
		Owner          *string           `json:"owner"`
		Tags           map[string]string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	a, err := h.uc.Update(c.Request.Context(), assetID, assetUC.UpdateInput{
		Status:         req.Status,
		LifecycleState: req.LifecycleState,
		Reviewer:       req.Reviewer,
		Owner:          req.Owner,
		Tags:           req.Tags,
	})
	if err != nil {
		switch {
		case errors.Is(err, assetUC.ErrNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		case errors.Is(err, assetUC.ErrInvalidTag):
			httpresp.Unprocessable(c, httpresp.CodeInvalidTag, err.Error(), nil)
		case errors.Is(err, repository.ErrOptimisticLock):
			httpresp.Conflict(c, httpresp.CodeConcurrentConflict,
				"asset was modified concurrently; reload and retry", nil)
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	if len(req.Tags) > 0 {
		audit.Log(c.Request.Context(), "asset.batch_tag", "asset", []string{assetID}, map[string]any{"tags": req.Tags})
	}
	c.JSON(200, a)
}

// Delete soft-deletes an asset.
// @Summary      Delete asset
// @Description  Soft delete an asset (status = archived)
// @Tags         assets
// @Produce      json
// @Param        id path string true "Asset ID"
// @Success      200 {object} object
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	if err := h.uc.Delete(c.Request.Context(), assetID); err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	audit.Log(c.Request.Context(), "asset.delete", "asset", []string{assetID}, nil)
	c.JSON(200, gin.H{"deleted": true, "asset_id": assetID})
}

// POST /internal/commit-segments
func (h *Handler) CommitSegments(c *gin.Context) {
	var req struct {
		McapFileID string     `json:"mcap_file_id" binding:"required"`
		Ranges     [][2]int64 `json:"ranges" binding:"required"`
		Reviewer   string     `json:"reviewer" binding:"required"`
		Owner      string     `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}

	created, err := h.uc.CommitSegments(c.Request.Context(), assetUC.CommitSegmentsInput{
		McapFileID: req.McapFileID,
		Ranges:     req.Ranges,
		Reviewer:   req.Reviewer,
		Owner:      req.Owner,
	})
	if err != nil {
		switch {
		case errors.Is(err, assetUC.ErrMcapFileIDRequired), errors.Is(err, assetUC.ErrInvalidRange):
			httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), map[string]any{"partial": created})
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, gin.H{"created": created, "count": len(created)})
}

// GET /api/v1/assets/:id/deliveries
func (h *Handler) ListDeliveries(c *gin.Context) {
	assetID := c.Param("id")
	page, pageSize := handlers.ParsePageParams(c.Query("page"), c.Query("page_size"))

	ids, err := h.deliveryRepo.ListByAsset(c.Request.Context(), assetID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if ids == nil {
		ids = []string{}
	}

	// Paginate the delivery ID list.
	start := (page - 1) * pageSize
	var items []string
	if start >= len(ids) {
		items = []string{}
	} else {
		end := start + pageSize
		if end > len(ids) {
			end = len(ids)
		}
		items = ids[start:end]
	}

	c.JSON(200, gin.H{
		"items":      items,
		"asset_id":   assetID,
		"page":       page,
		"page_size":  pageSize,
		"next_token": "",
	})
}

func parseOptionalInt64(raw string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseBoundedInt(raw string, fallback, min, max int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", raw, err)
	}
	if v < min || v > max {
		return 0, fmt.Errorf("value %d out of range [%d, %d]", v, min, max)
	}
	return v, nil
}

// BatchGet returns multiple assets by their IDs.
// @Summary      Batch get assets
// @Description  Retrieve multiple assets by ID in a single request (max 100)
// @Tags         assets
// @Accept       json
// @Produce      json
// @Param        body body object true "Batch get request"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets:batch_get [post]
func (h *Handler) BatchGet(c *gin.Context) {
	var req struct {
		AssetIDs []string `json:"asset_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if len(req.AssetIDs) == 0 {
		c.JSON(200, gin.H{"items": []any{}})
		return
	}
	if len(req.AssetIDs) > 100 {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset_ids exceeds maximum of 100", nil)
		return
	}

	result, err := h.uc.BatchGet(c.Request.Context(), req.AssetIDs)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	items := result
	c.JSON(200, gin.H{"items": items})
}

func paginateAssets(items []*models.Asset, page, pageSize int) []*models.Asset {
	if len(items) == 0 {
		return []*models.Asset{}
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []*models.Asset{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
