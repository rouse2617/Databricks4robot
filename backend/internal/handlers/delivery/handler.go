package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/audit"
	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type Handler struct {
	repo      repository.DeliveryRepository
	idemRepo  repository.IdempotencyRepository
	eventRepo repository.AssetEventRepository
}

func New(repo repository.DeliveryRepository, idemRepo repository.IdempotencyRepository, eventRepo ...repository.AssetEventRepository) *Handler {
	var evt repository.AssetEventRepository
	if len(eventRepo) > 0 {
		evt = eventRepo[0]
	}
	return &Handler{repo: repo, idemRepo: idemRepo, eventRepo: evt}
}

func (h *Handler) appendDeliveryEvents(ctx context.Context, d *models.Delivery, assetIDs []string, requestID string) error {
	if h.eventRepo == nil {
		return nil
	}
	payload := map[string]any{
		"delivery_id":  d.DeliveryID,
		"customer_id":  d.CustomerID,
		"status":       d.Status,
		"asset_count":  d.AssetCount,
		"delivered_at": d.DeliveredAt,
	}
	body, _ := json.Marshal(payload)
	for _, assetID := range assetIDs {
		if err := h.eventRepo.Append(ctx, repository.AssetEventAppendInput{
			EventType:            "delivery_committed",
			AggregateType:        "delivery",
			PayloadSchemaVersion: "v1",
			AssetID:              assetID,
			TenantID:             d.TenantID,
			ProjectID:            d.ProjectID,
			EventSource:          "backend",
			RequestID:            requestID,
			EventPayload:         body,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) commitDelivery(ctx context.Context, d *models.Delivery, assetIDs []string, requestID string) error {
	writeFn := func(txCtx context.Context) error {
		if err := h.repo.Set(txCtx, d); err != nil {
			return err
		}
		for _, assetID := range assetIDs {
			if err := h.repo.WriteIndexes(txCtx, assetID, d); err != nil {
				return err
			}
		}
		return h.appendDeliveryEvents(txCtx, d, assetIDs, requestID)
	}

	if txRunner, ok := h.repo.(repository.TxRunner); ok {
		return txRunner.WithTx(ctx, writeFn)
	}
	return writeFn(ctx)
}

// Commit creates a new delivery.
// @Summary      Commit delivery
// @Description  Create a new delivery for a set of assets
// @Tags         deliveries
// @Accept       json
// @Produce      json
// @Param        Idempotency-Key header string true "Idempotency key"
// @Param        body body object true "Commit delivery request"
// @Success      201 {object} object
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Failure      500 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /deliveries [post]
func (h *Handler) Commit(c *gin.Context) {
	var req struct {
		AssetIDs   []string `json:"asset_ids" binding:"required,min=1"`
		CustomerID string   `json:"customer_id" binding:"required"`
		ContractID string   `json:"contract_id"`
		Note       string   `json:"note"`
		Owner      string   `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	for _, raw := range req.AssetIDs {
		if !id.ValidateAssetID(raw) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset_ids must be 8 alphanumeric characters", map[string]any{"asset_id": raw})
			return
		}
	}
	idemKey := c.GetHeader("Idempotency-Key")
	if idemKey == "" {
		httpresp.BadRequest(c, httpresp.CodeMissingIdempotencyKey, "Idempotency-Key header is required", nil)
		return
	}
	hash := hashDeliveryRequest(req)
	if rec, err := h.idemRepo.Get(c.Request.Context(), "deliveries_commit", idemKey); err == nil && rec != nil {
		if rec.RequestHash != hash {
			httpresp.Conflict(c, httpresp.CodeIdempotencyConflict, "same idempotency key used with different payload", nil)
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

	if err := h.commitDelivery(c.Request.Context(), d, req.AssetIDs, c.GetHeader("X-Request-ID")); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	body := gin.H{
		"delivery_id":  d.DeliveryID,
		"customer_id":  d.CustomerID,
		"asset_count":  d.AssetCount,
		"delivered_at": d.DeliveredAt,
		"status":       d.Status,
	}
	audit.Log(c.Request.Context(), "delivery.commit", "delivery", []string{d.DeliveryID}, map[string]any{
		"customer_id": req.CustomerID,
		"asset_ids":   req.AssetIDs,
		"asset_count": len(req.AssetIDs),
	})
	respBytes, _ := json.Marshal(body)
	if err := h.idemRepo.Save(c.Request.Context(), &repository.IdempotencyRecord{
		Scope: "deliveries_commit", Key: idemKey, RequestHash: hash, StatusCode: http.StatusCreated, Response: respBytes,
	}); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, body)
}

// GET /api/v1/deliveries
func (h *Handler) List(c *gin.Context) {
	page, pageSize := handlers.ParsePageParams(c.Query("page"), c.Query("page_size"))
	status := c.Query("status")

	items, total, err := h.repo.List(c.Request.Context(), page, pageSize, status)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []*models.Delivery{}
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GET /api/v1/deliveries/:id
func (h *Handler) Get(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}
	d, err := h.repo.Get(c.Request.Context(), deliveryID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if d == nil {
		httpresp.NotFound(c, httpresp.CodeDeliveryNotFound, "delivery not found")
		return
	}
	c.JSON(http.StatusOK, d)
}

// GET /api/v1/deliveries/:id/items
func (h *Handler) ListItems(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}
	d, err := h.repo.Get(c.Request.Context(), deliveryID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if d == nil {
		httpresp.NotFound(c, httpresp.CodeDeliveryNotFound, "delivery not found")
		return
	}

	items, err := h.repo.ListItems(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []*models.DeliveryItem{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /api/v1/customers/:customer_id/deliveries
func (h *Handler) ListByCustomer(c *gin.Context) {
	customerID := strings.TrimSpace(c.Param("customer_id"))
	if customerID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "customer_id is required", nil)
		return
	}
	ids, err := h.repo.ListByCustomer(c.Request.Context(), customerID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	page, pageSize := handlers.ParsePageParams(c.Query("page"), c.Query("page_size"))
	items, nextToken := paginateIDs(ids, page, pageSize)
	c.JSON(http.StatusOK, gin.H{
		"delivery_ids": items,
		"total":        len(ids),
		"page":         page,
		"page_size":    pageSize,
		"next_token":   nextToken,
	})
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
