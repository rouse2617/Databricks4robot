package asset

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"data-platform/internal/httpresp"
	"data-platform/internal/models"
	assetUC "data-platform/internal/usecase/asset"
)

type Handler struct {
	uc *assetUC.Usecase
}

func New(uc *assetUC.Usecase) *Handler {
	return &Handler{uc: uc}
}

// GET /api/v1/assets/:id
func (h *Handler) Get(c *gin.Context) {
	a, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, "ASSET_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, a)
}

// GET /api/v1/assets?mcap_file_id=<id>
func (h *Handler) List(c *gin.Context) {
	assets, err := h.uc.ListByMcapFile(c.Request.Context(), c.Query("mcap_file_id"))
	if err != nil {
		if errors.Is(err, assetUC.ErrMcapFileIDRequired) {
			httpresp.BadRequest(c, "INVALID_ARGUMENT", err.Error(), nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	page, pageSize := parsePageParams(c.Query("page"), c.Query("page_size"))
	items := paginateAssets(assets, page, pageSize)
	c.JSON(200, gin.H{"items": items, "total": len(assets), "page": page, "page_size": pageSize})
}

// POST /api/v1/assets
func (h *Handler) Create(c *gin.Context) {
	var req struct {
		McapFileID       string            `json:"mcap_file_id" binding:"required"`
		StartTimestampNs int64             `json:"start_timestamp_ns" binding:"required"`
		EndTimestampNs   int64             `json:"end_timestamp_ns" binding:"required"`
		Reviewer         string            `json:"reviewer" binding:"required"`
		Owner            string            `json:"owner"`
		SegType          string            `json:"type"`
		Env              string            `json:"env"`
		Task             string            `json:"task"`
		Tags             map[string]string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid request body", map[string]any{"error": err.Error()})
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
			httpresp.Unprocessable(c, "INVALID_STATE", err.Error(), nil)
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
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	a, err := h.uc.Update(c.Request.Context(), c.Param("id"), assetUC.UpdateInput{
		Status:   req.Status,
		Reviewer: req.Reviewer,
		Owner:    req.Owner,
		Tags:     req.Tags,
	})
	if err != nil {
		if errors.Is(err, assetUC.ErrNotFound) {
			httpresp.NotFound(c, "ASSET_NOT_FOUND", err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, a)
}

// DELETE /api/v1/assets/:id  → soft delete (status = archived)
func (h *Handler) Delete(c *gin.Context) {
	if err := h.uc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"deleted": true, "asset_id": c.Param("id")})
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
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid request body", map[string]any{"error": err.Error()})
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
			httpresp.Unprocessable(c, "INVALID_STATE", err.Error(), map[string]any{"partial": created})
		default:
			httpresp.Internal(c, err.Error())
		}
		return
	}
	c.JSON(201, gin.H{"created": created, "count": len(created)})
}

// GET /api/v1/assets/:id/deliveries
func (h *Handler) ListDeliveries(c *gin.Context) {
	c.JSON(200, gin.H{"items": []any{}, "asset_id": c.Param("id"), "page": 1, "page_size": 20, "next_token": ""})
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
