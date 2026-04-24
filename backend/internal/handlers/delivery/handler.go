package delivery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"data-platform/internal/httpresp"
	"data-platform/internal/models"
	"data-platform/internal/repository"
)

type Handler struct {
	repo     repository.DeliveryRepository
	idemRepo repository.IdempotencyRepository
}

func New(repo repository.DeliveryRepository, idemRepo repository.IdempotencyRepository) *Handler {
	return &Handler{repo: repo, idemRepo: idemRepo}
}

// POST /api/v1/deliveries
func (h *Handler) Commit(c *gin.Context) {
	var req struct {
		AssetIDs   []string `json:"asset_ids" binding:"required,min=1"`
		CustomerID string   `json:"customer_id" binding:"required"`
		ContractID string   `json:"contract_id"`
		Note       string   `json:"note"`
		Owner      string   `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "INVALID_ARGUMENT", "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	idemKey := c.GetHeader("Idempotency-Key")
	if idemKey == "" {
		httpresp.BadRequest(c, "MISSING_IDEMPOTENCY_KEY", "Idempotency-Key header is required", nil)
		return
	}
	hash := hashDeliveryRequest(req)
	if rec, err := h.idemRepo.Get(c.Request.Context(), "deliveries_commit", idemKey); err == nil && rec != nil {
		if rec.RequestHash != hash {
			httpresp.Conflict(c, "IDEMPOTENCY_CONFLICT", "same idempotency key used with different payload", nil)
			return
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Response, &body)
		c.JSON(rec.StatusCode, body)
		return
	}

	now := time.Now()
	d := &models.Delivery{
		DeliveryID:  uuid.NewString(),
		CustomerID:  req.CustomerID,
		Status:      models.DeliveryStatusDelivered,
		DeliveredAt: &now,
		ContractID:  req.ContractID,
		Note:        req.Note,
		Owner:       req.Owner,
		AssetCount:  len(req.AssetIDs),
		CreatedAt:   now,
	}

	if err := h.repo.Set(c.Request.Context(), d); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	// Write secondary indexes for each asset
	for _, assetID := range req.AssetIDs {
		if err := h.repo.WriteIndexes(c.Request.Context(), assetID, d); err != nil {
			c.Header("X-Warning", "index write partial failure: "+err.Error())
		}
	}

	body := gin.H{
		"delivery_id": d.DeliveryID,
		"customer_id": d.CustomerID,
		"asset_count": d.AssetCount,
		"delivered_at": d.DeliveredAt,
		"status":      d.Status,
	}
	c.JSON(http.StatusCreated, body)
	respBytes, _ := json.Marshal(body)
	_ = h.idemRepo.Save(c.Request.Context(), &repository.IdempotencyRecord{
		Scope: "deliveries_commit", Key: idemKey, RequestHash: hash, StatusCode: http.StatusCreated, Response: respBytes,
	})
}

// GET /api/v1/deliveries/:id
func (h *Handler) Get(c *gin.Context) {
	d, err := h.repo.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if d == nil {
		httpresp.NotFound(c, "DELIVERY_NOT_FOUND", "delivery not found")
		return
	}
	c.JSON(http.StatusOK, d)
}

// GET /api/v1/customers/:customer_id/deliveries
func (h *Handler) ListByCustomer(c *gin.Context) {
	ids, err := h.repo.ListByCustomer(c.Request.Context(), c.Param("customer_id"))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	page, pageSize := parsePageParams(c.Query("page"), c.Query("page_size"))
	items, nextToken := paginateIDs(ids, page, pageSize)
	c.JSON(http.StatusOK, gin.H{
		"delivery_ids": items,
		"total":       len(ids),
		"page":        page,
		"page_size":   pageSize,
		"next_token":  nextToken,
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

func paginateIDs(ids []string, page, pageSize int) ([]string, string) {
	if len(ids) == 0 {
		return []string{}, ""
	}
	start := (page - 1) * pageSize
	if start >= len(ids) {
		return []string{}, ""
	}
	end := start + pageSize
	if end > len(ids) {
		end = len(ids)
	}
	nextToken := ""
	if end < len(ids) {
		nextToken = strconv.Itoa(page + 1)
	}
	return ids[start:end], nextToken
}

func hashDeliveryRequest(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
