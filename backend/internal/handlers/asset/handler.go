package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
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
	pgq          assetSQLQuerier
}

type assetSQLRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type assetSQLQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (assetSQLRows, error)
}

type assetPostgresQuerier struct {
	pg *postgres.Client
}

func (q assetPostgresQuerier) Query(ctx context.Context, sql string, args ...any) (assetSQLRows, error) {
	if q.pg == nil {
		return nil, fmt.Errorf("postgres client is not configured")
	}
	return q.pg.Query(ctx, sql, args...)
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
	if pg == nil {
		h.pgq = nil
	} else {
		h.pgq = assetPostgresQuerier{pg: pg}
	}
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
// @Security     DatabrewToken
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
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
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
// @Security     DatabrewToken
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
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
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
	body, err := h.buildLineageResponse(c.Request.Context(), assetID)
	if err != nil {
		slog.Warn("lineage response partial", "asset_id", assetID, "error", err)
	}
	c.JSON(200, body)
}

// GetProvenance returns version history, revisions, and structural lineage for an asset.
// @Summary      Get asset provenance
// @Description  Version timeline and revision list for the logical asset family, plus lineage snapshot
// @Tags         assets
// @Produce      json
// @Param        id path string true "Asset ID"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/provenance [get]
func (h *Handler) GetProvenance(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	res, err := h.uc.GetProvenance(c.Request.Context(), assetID)
	if err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	lineage, err := h.buildLineageResponse(c.Request.Context(), assetID)
	if err != nil {
		slog.Warn("provenance lineage partial", "asset_id", assetID, "error", err)
	}
	c.JSON(200, gin.H{
		"asset_id":         res.AssetID,
		"logical_asset_id": res.LogicalAssetID,
		"revisions":        res.Revisions,
		"version_history":  res.VersionHistory,
		"lineage":          lineage,
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
// @Security     DatabrewToken
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
		if !mapAssetError(c, err) {
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
// @Security     DatabrewToken
// @Router       /assets/{id}/tags/{key} [delete]
func (h *Handler) DeleteTag(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	a, err := h.uc.DeleteTag(c.Request.Context(), assetID, c.Param("key"), c.Query("source_type"))
	if err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
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
// @Security     DatabrewToken
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
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
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
// @Security     DatabrewToken
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
		// CYB-1164: hierarchy violation check before sentinel switch.
		var hv *deliveryrules.HierarchyViolation
		if errors.As(err, &hv) {
			httpresp.Unprocessable(c, httpresp.CodeHierarchyViolation, hv.Error(), map[string]any{
				"invariant":  hv.Invariant,
				"asset_type": hv.AssetType,
				"parent_id":  hv.ParentID,
				"expected":   hv.Expected,
			})
			return
		}
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, a)
}

// createChildAssetRequest is the shared request body for all layered endpoints.
type createChildAssetRequest struct {
	StartTimestampNs int64                  `json:"start_timestamp_ns" binding:"required,gt=0" label:"start_timestamp_ns"`
	EndTimestampNs   int64                  `json:"end_timestamp_ns" binding:"required,gt=0" label:"end_timestamp_ns"`
	SplitMethod      string                 `json:"split_method"`
	SplitRunID       string                 `json:"split_run_id"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// createChildAsset is the shared handler logic for all layered endpoints.
func (h *Handler) createChildAsset(c *gin.Context, assetType string) {
	parentID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	var req createChildAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if valErr := validate.ValidateStruct(&req); valErr != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, valErr.Error(), nil)
		return
	}
	a, err := h.uc.CreateChildAsset(c.Request.Context(), assetUC.CreateChildAssetInput{
		AssetType:        assetType,
		ParentAssetID:    parentID,
		StartTimestampNs: req.StartTimestampNs,
		EndTimestampNs:   req.EndTimestampNs,
		SplitMethod:      req.SplitMethod,
		SplitRunID:       req.SplitRunID,
		Metadata:         req.Metadata,
	})
	if err != nil {
		var hv *deliveryrules.HierarchyViolation
		if errors.As(err, &hv) {
			httpresp.Unprocessable(c, httpresp.CodeHierarchyViolation, hv.Error(), map[string]any{
				"invariant":  hv.Invariant,
				"asset_type": hv.AssetType,
				"parent_id":  hv.ParentID,
				"expected":   hv.Expected,
			})
			return
		}
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, a)
}

// POST /api/v1/assets/:id/clips
func (h *Handler) CreateClip(c *gin.Context) { h.createChildAsset(c, "clip") }

// POST /api/v1/assets/:id/actions
func (h *Handler) CreateAction(c *gin.Context) { h.createChildAsset(c, "action") }

// POST /api/v1/assets/:id/frames
func (h *Handler) CreateFrame(c *gin.Context) { h.createChildAsset(c, "frame") }

// POST /api/v1/assets/:id/tasks
func (h *Handler) CreateTask(c *gin.Context) { h.createChildAsset(c, "task") }

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
		if !mapAssetError(c, err) {
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
// @Security     DatabrewToken
// @Router       /assets/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	if err := h.uc.Delete(c.Request.Context(), assetID); err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
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
		if errors.Is(err, assetUC.ErrMcapFileIDRequired) || errors.Is(err, assetUC.ErrInvalidRange) {
			httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), map[string]any{"partial": created})
		} else if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, gin.H{"created": created, "count": len(created)})
}

// GET /api/v1/assets/:id/deliveries
func (h *Handler) ListDeliveries(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
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
	end := 0
	if start >= len(ids) {
		items = []string{}
	} else {
		end = start + pageSize
		if end > len(ids) {
			end = len(ids)
		}
		items = ids[start:end]
	}

	// Compute next_token based on whether there are more pages.
	var nextToken string
	if end < len(ids) {
		nextToken = strconv.Itoa(page + 1)
	}

	c.JSON(200, gin.H{
		"items":      items,
		"asset_id":   assetID,
		"page":       page,
		"page_size":  pageSize,
		"next_token": nextToken,
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
// @Security     DatabrewToken
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

	// Deduplicate asset IDs to avoid duplicate items in response.
	seen := make(map[string]struct{}, len(req.AssetIDs))
	deduped := make([]string, 0, len(req.AssetIDs))
	for _, id := range req.AssetIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			deduped = append(deduped, id)
		}
	}
	req.AssetIDs = deduped

	result, err := h.uc.BatchGet(c.Request.Context(), req.AssetIDs)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	items := result
	c.JSON(200, gin.H{"items": items})
}

// PromoteRevision creates a new revision of an asset (B-route promote).
// @Summary      Promote asset revision
// @Description  Create a new revision of an asset within a logical asset family
// @Tags         assets
// @Accept       json
// @Produce      json
// @Param        id   path string true "Source Asset ID"
// @Param        body body object true "Promote request"
// @Success      201 {object} models.Asset
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      422 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/revisions [post]
func (h *Handler) PromoteRevision(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	var req struct {
		LogicalAssetID string `json:"logical_asset_id"`
		RevisionOf     string `json:"revision_of"`
		Owner          string `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	a, err := h.uc.Promote(c.Request.Context(), assetID, assetUC.PromoteInput{
		LogicalAssetID: req.LogicalAssetID,
		RevisionOf:     req.RevisionOf,
		Owner:          req.Owner,
	})
	if err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, a)
}

// GetCurrentForLogical returns the current revision for a logical asset.
// @Summary      Get current logical asset
// @Description  Get the current revision for a logical asset
// @Tags         logical-assets
// @Produce      json
// @Param        id path string true "Logical Asset ID"
// @Success      200 {object} asset.CurrentAssetResponse
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /logical-assets/{id}/current [get]
func (h *Handler) GetCurrentForLogical(c *gin.Context) {
	logicalAssetID := strings.TrimSpace(c.Param("id"))
	if logicalAssetID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "logical_asset_id is required", nil)
		return
	}
	res, err := h.uc.GetCurrentForLogical(c.Request.Context(), logicalAssetID)
	if err != nil {
		if errors.Is(err, assetUC.ErrLogicalAssetNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "logical asset not found")
		} else if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, "no current revision found")
		} else if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(200, res)
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

// RecordView increments view_count and updates last_viewed_at for an asset (CYB-1095).
// @Summary      Record asset view
// @Description  Increment view counter and update last_viewed_at in asset_usage_stats
// @Tags         assets
// @Produce      json
// @Param        id path string true "Asset ID"
// @Success      200 {object} object
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/view [post]
func (h *Handler) RecordView(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	if err := h.uc.RecordAssetView(c.Request.Context(), assetID); err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(200, gin.H{"asset_id": assetID, "viewed": true})
}

// ToggleFavorite flips the favorite state for an asset (CYB-1096).
// @Summary      Toggle asset favorite
// @Description  Toggle favorite and update favorite_count in asset_usage_stats
// @Tags         assets
// @Produce      json
// @Param        id path string true "Asset ID"
// @Success      200 {object} object
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/favorite [post]
func (h *Handler) ToggleFavorite(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}
	newCount, err := h.uc.ToggleFavorite(c.Request.Context(), assetID)
	if err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(200, gin.H{"asset_id": assetID, "favorite_count": newCount, "is_favorited": newCount > 0})
}

// sseEvent represents one event in the SSE stream.
type sseEvent struct {
	EventID    string          `json:"event_id"`
	EventSeq   int64           `json:"event_seq"`
	EventType  string          `json:"event_type"`
	AssetID    string          `json:"asset_id,omitempty"`
	Payload    json.RawMessage `json:"event_payload"`
	OccurredAt time.Time       `json:"occurred_at"`
}

var assetEventStreamPollInterval = 5 * time.Second

// HandleEventsStream streams asset events as Server-Sent Events (CYB-1099).
// Supports Last-Event-ID header for reconnection — the client sends the last
// event_seq it received and the server replays from that point onward.
//
// @Summary      Stream asset events (SSE)
// @Description  SSE endpoint that polls asset_events for new rows belonging to this asset.
//
//	Supports Last-Event-ID for reconnection (value must be an event_seq integer).
//
// @Tags         assets
// @Produce      text/event-stream
// @Param        id path string true "Asset ID"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /assets/{id}/events/stream [get]
func (h *Handler) HandleEventsStream(c *gin.Context) {
	assetID, ok := handlers.RequirePathAssetID(c)
	if !ok {
		return
	}

	lastEventIDStr := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	var afterSeq *int64
	if lastEventIDStr != "" {
		v, err := strconv.ParseInt(lastEventIDStr, 10, 64)
		if err != nil || v < 0 {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid Last-Event-ID", map[string]any{
				"error": "Last-Event-ID must be a non-negative event_seq integer",
			})
			return
		}
		afterSeq = &v
	}

	if _, err := h.uc.GetAll(c.Request.Context(), assetID); err != nil {
		if !mapAssetError(c, err) {
			httpresp.Internal(c, err.Error())
		}
		return
	}

	// SSE headers.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		// Wait for new events by polling.
		events := h.pollAssetEvents(c.Request.Context(), assetID, afterSeq)
		if events == nil {
			// Context cancelled or query failed — stop streaming.
			return false
		}

		for _, ev := range events {
			payloadJSON, _ := json.Marshal(ev.Payload)
			if payloadJSON == nil {
				payloadJSON = []byte("{}")
			}

			// SSE format: "id: <event_seq>\nevent: <event_type>\ndata: <json>\n\n"
			line := fmt.Sprintf("id: %d\nevent: %s\ndata: {\"event_id\":%q,\"event_seq\":%d,\"event_type\":%q,\"asset_id\":%q,\"event_payload\":%s,\"occurred_at\":%q}\n\n",
				ev.EventSeq, ev.EventType,
				ev.EventID, ev.EventSeq, ev.EventType, ev.AssetID,
				string(payloadJSON),
				ev.OccurredAt.Format(time.RFC3339Nano),
			)

			if _, err := io.WriteString(w, line); err != nil {
				return false
			}

			// Update the cursor so subsequent polls skip past this event.
			afterSeq = &ev.EventSeq
		}

		// Send keepalive comment every poll cycle.
		if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
			return false
		}

		return true
	})
}

// pollAssetEvents queries asset_events for rows with event_seq > afterSeq
// belonging to the given asset. Returns nil when the context is cancelled or
// the query fails.
func (h *Handler) pollAssetEvents(ctx context.Context, assetID string, afterSeq *int64) []sseEvent {
	if h.pgq == nil {
		if !sleepAssetEventStreamPoll(ctx) {
			return nil
		}
		return []sseEvent{}
	}

	args := []interface{}{assetID}
	where := "asset_id = $1"

	if afterSeq != nil {
		args = append(args, *afterSeq)
		where = fmt.Sprintf("asset_id = $1 AND event_seq > $%d", len(args))
	}

	q := `SELECT event_id, event_seq, event_type,
  COALESCE(asset_id::text, ''),
  event_payload,
  occurred_at
FROM asset_events
WHERE ` + where + `
ORDER BY event_seq ASC
LIMIT 50`

	rows, err := h.pgq.Query(ctx, q, args...)
	if err != nil {
		// If context cancelled, return nil to signal stop.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		// On transient error, sleep and return empty (don't kill the stream).
		if !sleepAssetEventStreamPoll(ctx) {
			return nil
		}
		return []sseEvent{}
	}
	defer rows.Close()

	var events []sseEvent
	for rows.Next() {
		var ev sseEvent
		if err := rows.Scan(
			&ev.EventID, &ev.EventSeq, &ev.EventType,
			&ev.AssetID,
			&ev.Payload,
			&ev.OccurredAt,
		); err != nil {
			return []sseEvent{}
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return []sseEvent{}
	}

	// If no new events, sleep before polling again.
	if len(events) == 0 {
		if !sleepAssetEventStreamPoll(ctx) {
			return nil
		}
	}

	return events
}

func sleepAssetEventStreamPoll(ctx context.Context) bool {
	timer := time.NewTimer(assetEventStreamPollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// ratingsHistoryRevision represents one logical asset revision and its rating
// metrics for the ratings-history endpoint.
type ratingsHistoryRevision struct {
	AssetID        string                 `json:"asset_id"`
	Revision       int64                  `json:"revision"`
	IsCurrent      bool                   `json:"is_current"`
	LifecycleState string                 `json:"lifecycle_state"`
	CreatedAt      time.Time              `json:"created_at"`
	Ratings        []ratingsHistoryMetric `json:"ratings"`
}

type ratingsHistoryMetric struct {
	MetricKey        string    `json:"metric_key"`
	MetricType       string    `json:"metric_type"`
	MetricUnit       *string   `json:"metric_unit,omitempty"`
	MetricValue      *float64  `json:"metric_value,omitempty"`
	MetricValueInt   *int64    `json:"metric_value_int,omitempty"`
	MetricValueText  *string   `json:"metric_value_text,omitempty"`
	MetricValueBool  *bool     `json:"metric_value_bool,omitempty"`
	TargetType       string    `json:"target_type"`
	TargetID         string    `json:"target_id"`
	EvalName         string    `json:"eval_name"`
	EvalVersion      string    `json:"eval_version"`
	ParameterVersion *string   `json:"parameter_version,omitempty"`
	RunID            *string   `json:"run_id,omitempty"`
	SourceType       string    `json:"source_type"`
	SourceName       *string   `json:"source_name,omitempty"`
	Confidence       *float64  `json:"confidence,omitempty"`
	RecordedAt       time.Time `json:"recorded_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// HandleRatingsHistory returns all revisions of a logical asset with their
// associated rating.* metric rows over time (CYB-1100).
//
// @Summary      Get ratings history for a logical asset
// @Description  Query all revisions of a logical asset, return rating fields over time
// @Tags         logical-assets
// @Produce      json
// @Param        id path string true "Logical Asset ID"
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /logical-assets/{id}/ratings-history [get]
func (h *Handler) HandleRatingsHistory(c *gin.Context) {
	logicalAssetID := strings.TrimSpace(c.Param("id"))
	if logicalAssetID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "logical_asset_id is required", nil)
		return
	}
	if !id.ValidateAssetID(logicalAssetID) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "logical_asset_id must be an 8-character alphanumeric id", nil)
		return
	}
	if h == nil || h.pgq == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, "PG_DISABLED", "postgres not available", nil)
		return
	}

	const q = `
SELECT a.asset_id,
       COALESCE(a.revision, 0) AS revision,
       COALESCE(a.is_current, FALSE) AS is_current,
       COALESCE(a.lifecycle_state, '') AS lifecycle_state,
       a.created_at,
       m.metric_key,
       m.metric_type,
       m.metric_unit,
       m.metric_value,
       m.metric_value_int,
       m.metric_value_text,
       m.metric_value_bool,
       m.target_type,
       m.target_id,
       m.eval_name,
       m.eval_version,
       m.parameter_version,
       m.run_id,
       m.source_type,
       m.source_name,
       m.confidence,
       m.recorded_at,
       m.updated_at
FROM assets a
LEFT JOIN asset_metrics m ON m.asset_id = a.asset_id
  AND m.metric_key LIKE 'rating.%'
WHERE a.logical_asset_id = $1
  AND a.is_deleted = FALSE
ORDER BY a.revision ASC NULLS LAST,
         a.created_at ASC,
         m.metric_key ASC NULLS LAST,
         m.source_type ASC NULLS LAST,
         m.source_name ASC NULLS LAST,
         m.recorded_at ASC NULLS LAST`

	rows, err := h.pgq.Query(c.Request.Context(), q, logicalAssetID)
	if err != nil {
		httpresp.Internal(c, "database query failed: "+err.Error())
		return
	}
	defer rows.Close()

	items := make([]ratingsHistoryRevision, 0)
	itemIndexByAssetID := make(map[string]int)
	for rows.Next() {
		var (
			assetID        string
			revision       int64
			isCurrent      bool
			lifecycleState string
			createdAt      time.Time
			metric         ratingsHistoryMetric
			metricKey      *string
			metricType     *string
			targetType     *string
			targetID       *string
			evalName       *string
			evalVersion    *string
			sourceType     *string
			recordedAt     *time.Time
			updatedAt      *time.Time
		)

		if err := rows.Scan(
			&assetID, &revision, &isCurrent, &lifecycleState, &createdAt,
			&metricKey, &metricType, &metric.MetricUnit,
			&metric.MetricValue, &metric.MetricValueInt, &metric.MetricValueText, &metric.MetricValueBool,
			&targetType, &targetID, &evalName, &evalVersion,
			&metric.ParameterVersion, &metric.RunID,
			&sourceType, &metric.SourceName, &metric.Confidence,
			&recordedAt, &updatedAt,
		); err != nil {
			httpresp.Internal(c, "scan failed: "+err.Error())
			return
		}

		itemIndex, ok := itemIndexByAssetID[assetID]
		if !ok {
			items = append(items, ratingsHistoryRevision{
				AssetID:        assetID,
				Revision:       revision,
				IsCurrent:      isCurrent,
				LifecycleState: lifecycleState,
				CreatedAt:      createdAt,
				Ratings:        []ratingsHistoryMetric{},
			})
			itemIndex = len(items) - 1
			itemIndexByAssetID[assetID] = itemIndex
		}

		if metricKey == nil {
			continue
		}
		metric.MetricKey = *metricKey
		if metricType != nil {
			metric.MetricType = *metricType
		}
		if targetType != nil {
			metric.TargetType = *targetType
		}
		if targetID != nil {
			metric.TargetID = *targetID
		}
		if evalName != nil {
			metric.EvalName = *evalName
		}
		if evalVersion != nil {
			metric.EvalVersion = *evalVersion
		}
		if sourceType != nil {
			metric.SourceType = *sourceType
		}
		if recordedAt != nil {
			metric.RecordedAt = *recordedAt
		}
		if updatedAt != nil {
			metric.UpdatedAt = *updatedAt
		}
		items[itemIndex].Ratings = append(items[itemIndex].Ratings, metric)
	}
	if err := rows.Err(); err != nil {
		httpresp.Internal(c, "row iteration error: "+err.Error())
		return
	}
	if len(items) == 0 {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "logical asset not found")
		return
	}

	c.JSON(200, gin.H{
		"logical_asset_id": logicalAssetID,
		"items":            items,
		"count":            len(items),
	})
}
