package asset

import (
	"errors"
	"strconv"

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
		McapFileID       string            `json:"mcap_file_id" binding:"required" label:"MCAP文件ID"`
		StartTimestampNs int64             `json:"start_timestamp_ns" binding:"required,gt=0" label:"起始时间戳"`
		EndTimestampNs   int64             `json:"end_timestamp_ns" binding:"required,gt=0" label:"结束时间戳"`
		Reviewer         string            `json:"reviewer" binding:"required" label:"审核人"`
		Owner            string            `json:"owner" label:"所有者"`
		SegType          string            `json:"type" label:"片段类型"`
		Env              string            `json:"env" label:"环境"`
		Task             string            `json:"task" label:"任务"`
		Tags             map[string]string `json:"tags"`
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
		McapFileID:       req.McapFileID,
		StartTimestampNs: req.StartTimestampNs,
		EndTimestampNs:   req.EndTimestampNs,
		Reviewer:         req.Reviewer,
		Owner:            req.Owner,
		SegType:          req.SegType,
		Env:              req.Env,
		Task:             req.Task,
		Tags:             req.Tags,
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
