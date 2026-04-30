package asset

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"data-platform/internal/audit"
	"data-platform/internal/filter"
	"data-platform/internal/httpresp"
	"data-platform/internal/models"
	"data-platform/internal/repository"
	assetUC "data-platform/internal/usecase/asset"
	"data-platform/internal/validate"
)

type Handler struct {
	uc           *assetUC.Usecase
	deliveryRepo repository.DeliveryRepository
}

func New(uc *assetUC.Usecase, deliveryRepo repository.DeliveryRepository) *Handler {
	return &Handler{uc: uc, deliveryRepo: deliveryRepo}
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
	a, err := h.uc.Get(c.Request.Context(), c.Param("id"))
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
// @Summary      List assets
// @Description  List assets with optional filters, sorting, and pagination
// @Tags         assets
// @Produce      json
// @Param        filter   query []string false "Filter expressions (field:op:value)"
// @Param        sort_by  query string   false "Sort field (prefix - for desc)" default(-created_at)
// @Param        page     query int      false "Page number" default(1)
// @Param        page_size query int     false "Page size" default(20)
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets [get]
func (h *Handler) List(c *gin.Context) {
	filterStrs := c.QueryArray("filter")
	page, pageSize := parsePageParams(c.Query("page"), c.Query("page_size"))
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
		httpresp.Internal(c, err.Error())
		return
	}

	orderBy, err := filter.ResolveSortBy(sortBy)
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
	return eventQuery{
		beforeSeq: beforeSeq,
		afterSeq:  afterSeq,
		limit:     parseBoundedInt(c.Query("limit"), 50, 1, 200),
	}, nil
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
// @Param        limit            query int    false "Page size" default(50)
// @Success      200 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /assets/{id}/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
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

	res, err := h.uc.ListEvents(c.Request.Context(), c.Param("id"), assetUC.ListEventsInput{
		EventTypes:        eventTypes,
		EventTypePatterns: eventTypePatterns,
		AlgoKey:           c.Query("algo_key"),
		BeforeEventSeq:    q.beforeSeq,
		AfterEventSeq:     q.afterSeq,
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
	var req struct {
		Key   string `json:"key" binding:"required" label:"标签 Key"`
		Value string `json:"value" binding:"required" label:"标签值"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	a, err := h.uc.UpsertTag(c.Request.Context(), c.Param("id"), assetUC.UpsertTagInput{
		Key:   req.Key,
		Value: req.Value,
	})
	if err != nil {
		switch {
		case errors.Is(err, assetUC.ErrNotFound):
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		case errors.Is(err, assetUC.ErrInvalidTag):
			httpresp.Unprocessable(c, httpresp.CodeInvalidTag, err.Error(), nil)
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
	a, err := h.uc.DeleteTag(c.Request.Context(), c.Param("id"), c.Param("key"))
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
	q, err := parseEventQuery(c)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid event query", map[string]any{"error": err.Error()})
		return
	}
	res, err := h.uc.ListTagHistory(c.Request.Context(), c.Param("id"), assetUC.ListEventsInput{
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
		AssetID          string            `json:"asset_id" label:"资产ID"`
		McapFileID       string            `json:"mcap_file_id" binding:"required" label:"MCAP文件ID"`
		StartTimestampNs int64             `json:"start_timestamp_ns" binding:"required,gt=0" label:"起始时间戳"`
		EndTimestampNs   int64             `json:"end_timestamp_ns" binding:"required,gt=0" label:"结束时间戳"`
		Reviewer         string            `json:"reviewer" binding:"required" label:"审核人"`
		Owner            string            `json:"owner" label:"所有者"`
		SegType          string            `json:"type" label:"片段类型"`
		AssetType        string            `json:"asset_type" label:"资产类型"`
		Status           string            `json:"status" label:"旧生命周期状态"`
		LifecycleState   string            `json:"lifecycle_state" label:"新生命周期状态"`
		Env              string            `json:"env" label:"环境"`
		Task             string            `json:"task" label:"任务"`
		Tags             map[string]string `json:"tags"`
		Files            map[string]string `json:"files"`
		Metadata         map[string]interface{} `json:"metadata"`
		LifecycleMeta    map[string]interface{} `json:"lifecycle_meta"`
		RetentionTier    string            `json:"retention_tier"`
		ExpireAt         *time.Time        `json:"expire_at"`
		StorageURI       string            `json:"storage_uri"`
		ThumbURI         string            `json:"thumb_uri"`
		AssetLevel       int               `json:"asset_level"`
		ParentAssetID    string            `json:"parent_asset_id"`
		RootAssetID      string            `json:"root_asset_id"`
		SegmentIndex     *int              `json:"segment_index"`
		ParentStartOffsetMs *int64         `json:"parent_start_offset_ms"`
		ParentEndOffsetMs   *int64         `json:"parent_end_offset_ms"`
		SplitMethod      string            `json:"split_method"`
		SplitAlgoName    string            `json:"split_algo_name"`
		SplitAlgoVersion string            `json:"split_algo_version"`
		SplitRunID       string            `json:"split_run_id"`
		SplitReason      string            `json:"split_reason"`
		DeliveryCount    int               `json:"delivery_count"`
		LastDeliveredAt  *time.Time        `json:"last_delivered_at"`
		LastDeliveredTo  string            `json:"last_delivered_to"`
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
		AssetID:          req.AssetID,
		McapFileID:       req.McapFileID,
		StartTimestampNs: req.StartTimestampNs,
		EndTimestampNs:   req.EndTimestampNs,
		Reviewer:         req.Reviewer,
		Owner:            req.Owner,
		SegType:          req.SegType,
		AssetType:        req.AssetType,
		Status:           req.Status,
		LifecycleState:   req.LifecycleState,
		Env:              req.Env,
		Task:             req.Task,
		Tags:             req.Tags,
		Files:            req.Files,
		Metadata:         req.Metadata,
		LifecycleMeta:    req.LifecycleMeta,
		RetentionTier:    req.RetentionTier,
		ExpireAt:         req.ExpireAt,
		StorageURI:       req.StorageURI,
		ThumbURI:         req.ThumbURI,
		AssetLevel:       req.AssetLevel,
		ParentAssetID:    req.ParentAssetID,
		RootAssetID:      req.RootAssetID,
		SegmentIndex:     req.SegmentIndex,
		ParentStartOffsetMs: req.ParentStartOffsetMs,
		ParentEndOffsetMs:   req.ParentEndOffsetMs,
		SplitMethod:      req.SplitMethod,
		SplitAlgoName:    req.SplitAlgoName,
		SplitAlgoVersion: req.SplitAlgoVersion,
		SplitRunID:       req.SplitRunID,
		SplitReason:      req.SplitReason,
		DeliveryCount:    req.DeliveryCount,
		LastDeliveredAt:  req.LastDeliveredAt,
		LastDeliveredTo:  req.LastDeliveredTo,
	})
	if err != nil {
		switch {
		case errors.Is(err, assetUC.ErrMcapFileIDRequired), errors.Is(err, assetUC.ErrInvalidRange):
			httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), nil)
		case errors.Is(err, assetUC.ErrInvalidTag):
			httpresp.Unprocessable(c, httpresp.CodeInvalidTag, err.Error(), nil)
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, a)
}

// PATCH /api/v1/assets/:id
func (h *Handler) Update(c *gin.Context) {
	var req struct {
		Status   *string           `json:"status"`
		Reviewer *string           `json:"reviewer"`
		Owner    *string           `json:"owner"`
		Tags     map[string]string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	a, err := h.uc.Update(c.Request.Context(), c.Param("id"), assetUC.UpdateInput{
		Status:   req.Status,
		Reviewer: req.Reviewer,
		Owner:    req.Owner,
		Tags:     req.Tags,
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
		audit.Log(c.Request.Context(), "asset.batch_tag", "asset", []string{c.Param("id")}, map[string]any{"tags": req.Tags})
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
	assetID := c.Param("id")
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
	page, pageSize := parsePageParams(c.Query("page"), c.Query("page_size"))

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

func parsePageParams(pageStr, pageSizeStr string) (int, int) {
	page := 1
	pageSize := 20
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if s, err := strconv.Atoi(pageSizeStr); err == nil && s > 0 && s <= 200 {
			pageSize = s
		}
	}
	return page, pageSize
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

func parseBoundedInt(raw string, fallback, min, max int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < min || v > max {
		return fallback
	}
	return v
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

	items := make([]*models.Asset, 0, len(req.AssetIDs))
	result, err := h.uc.BatchGet(c.Request.Context(), req.AssetIDs)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	items = result
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
