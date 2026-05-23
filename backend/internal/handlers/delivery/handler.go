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
	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	"github.com/CyberOrigin2077/cyber-databrew/internal/handlers"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type Handler struct {
	repo         repository.DeliveryRepository
	idemRepo     repository.IdempotencyRepository
	customerRepo repository.CustomerRepository
	eventRepo    repository.AssetEventRepository
	ruleEngine   *deliveryrules.Engine
}

func New(repo repository.DeliveryRepository, idemRepo repository.IdempotencyRepository, customerRepo repository.CustomerRepository, eventRepo ...repository.AssetEventRepository) *Handler {
	var evt repository.AssetEventRepository
	if len(eventRepo) > 0 {
		evt = eventRepo[0]
	}
	return &Handler{repo: repo, idemRepo: idemRepo, customerRepo: customerRepo, eventRepo: evt}
}

// SetRuleEngine enables pre-delivery compliance checks (CYB-1020).
func (h *Handler) SetRuleEngine(e *deliveryrules.Engine) {
	h.ruleEngine = e
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
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	if req.CustomerID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "customer_id is required", nil)
		return
	}
	if h.customerRepo != nil {
		ok, err := h.customerRepo.Exists(c.Request.Context(), req.CustomerID)
		if err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
		if !ok {
			httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, "customer not found", map[string]any{"customer_id": req.CustomerID})
			return
		}
	}
	if h.ruleEngine != nil {
		violations, err := h.ruleEngine.Check(c.Request.Context(), req.CustomerID, req.AssetIDs)
		if err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
		if len(violations) > 0 {
			httpresp.Unprocessable(c, httpresp.CodeDeliveryRuleFailed,
				"one or more assets failed delivery rules",
				map[string]any{"violations": violations},
			)
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
	customerID := strings.TrimSpace(c.Query("customer_id"))

	items, total, err := h.repo.List(c.Request.Context(), page, pageSize, status, customerID)
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

// ─── C2 (two-step) delivery workflow ────────────────────────────────────────

// HandleDraft creates a new delivery in "pending" status without running
// the rule engine. Items are added later via HandleAddItems, then committed
// via HandleCommitC2.
func (h *Handler) HandleDraft(c *gin.Context) {
	var req struct {
		CustomerID string   `json:"customer_id" binding:"required"`
		AssetIDs   []string `json:"asset_ids"`
		ContractID string   `json:"contract_id"`
		Note       string   `json:"note"`
		Owner      string   `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	if req.CustomerID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "customer_id is required", nil)
		return
	}
	if h.customerRepo != nil {
		ok, err := h.customerRepo.Exists(c.Request.Context(), req.CustomerID)
		if err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
		if !ok {
			httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, "customer not found", map[string]any{"customer_id": req.CustomerID})
			return
		}
	}
	for _, raw := range req.AssetIDs {
		if !id.ValidateAssetID(raw) {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset_ids must be 8 alphanumeric characters", map[string]any{"asset_id": raw})
			return
		}
	}

	now := time.Now()
	requestedBy := c.GetHeader("X-Request-ID") // fallback; prefer body field
	if requestedBy == "" {
		requestedBy = req.Owner
	}
	d := &models.Delivery{
		DeliveryID:  uuid.NewString(),
		CustomerID:  req.CustomerID,
		Status:      models.DeliveryStatusPending,
		ContractID:  req.ContractID,
		Note:        req.Note,
		Owner:       req.Owner,
		RequestedBy: requestedBy,
		AssetCount:  len(req.AssetIDs),
		CreatedAt:   now,
	}

	if err := h.commitDelivery(c.Request.Context(), d, req.AssetIDs, c.GetHeader("X-Request-ID")); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	audit.Log(c.Request.Context(), "delivery.draft", "delivery", []string{d.DeliveryID}, map[string]any{
		"customer_id": req.CustomerID,
		"asset_count": len(req.AssetIDs),
	})
	c.JSON(http.StatusCreated, gin.H{
		"delivery_id": d.DeliveryID,
		"customer_id": d.CustomerID,
		"status":      d.Status,
		"asset_count": d.AssetCount,
		"created_at":  d.CreatedAt,
	})
}

// HandleAddItems batch-adds asset items to a pending delivery.
func (h *Handler) HandleAddItems(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}
	var req struct {
		AssetIDs []string `json:"asset_ids" binding:"required,min=1"`
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

	d, err := h.repo.Get(c.Request.Context(), deliveryID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if d == nil {
		httpresp.NotFound(c, httpresp.CodeDeliveryNotFound, "delivery not found")
		return
	}
	if d.Status != models.DeliveryStatusPending {
		httpresp.Unprocessable(c, httpresp.CodeInvalidState,
			"can only add items to pending deliveries",
			map[string]any{"current_status": d.Status})
		return
	}

	// Append items via WriteIndexes (idempotent ON CONFLICT).
	for _, assetID := range req.AssetIDs {
		if err := h.repo.WriteIndexes(c.Request.Context(), assetID, d); err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
	}

	// Refresh delivery to reflect new item count.
	d.AssetCount += len(req.AssetIDs)
	if err := h.repo.Update(c.Request.Context(), d, d.Version); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"delivery_id": d.DeliveryID,
		"customer_id": d.CustomerID,
		"status":      d.Status,
		"asset_count": d.AssetCount,
		"version":     d.Version,
	})
}

// HandleCommitC2 commits a pending delivery after re-running the rule engine.
// Enforce modes: block → reject violations; warn/tag_only → commit with warnings.
func (h *Handler) HandleCommitC2(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}
	var req struct {
		ExpectedRevision int64  `json:"expected_revision"`
		ApprovedBy       string `json:"approved_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
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
	if d.Status != models.DeliveryStatusPending {
		httpresp.Unprocessable(c, httpresp.CodeInvalidState,
			"can only commit pending deliveries",
			map[string]any{"current_status": d.Status})
		return
	}
	if d.Version != req.ExpectedRevision {
		httpresp.Conflict(c, httpresp.CodeConcurrentConflict,
			"delivery has been modified since you last read it",
			map[string]any{"expected_revision": req.ExpectedRevision, "current_version": d.Version})
		return
	}

	// Collect asset IDs from delivery_items for rule evaluation.
	items, err := h.repo.ListItems(c.Request.Context(), deliveryID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	var assetIDs []string
	for _, item := range items {
		assetIDs = append(assetIDs, item.AssetID)
	}

	// Run rule engine at commit time.
	var warnings []deliveryrules.Violation
	if h.ruleEngine != nil && len(assetIDs) > 0 {
		violations, err := h.ruleEngine.CheckAll(c.Request.Context(), d.CustomerID, assetIDs)
		if err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
		if len(violations) > 0 {
			// Check if any violation has enforce_mode "block".
			hasBlock := false
			for _, v := range violations {
				if v.EnforceMode == "block" {
					hasBlock = true
					break
				}
			}
			if hasBlock {
				httpresp.Unprocessable(c, httpresp.CodeDeliveryRuleFailed,
					"one or more assets failed delivery rules",
					map[string]any{"violations": violations},
				)
				return
			}
			// warn / tag_only → commit with warnings
			warnings = violations
		}
	}

	now := time.Now()
	d.Status = models.DeliveryStatusDelivered
	d.DeliveredAt = &now
	d.CompletedAt = &now
	d.ApprovedBy = req.ApprovedBy

	if err := h.repo.Update(c.Request.Context(), d, req.ExpectedRevision); err != nil {
		if err == repository.ErrOptimisticLock {
			httpresp.Conflict(c, httpresp.CodeConcurrentConflict,
				"delivery has been modified concurrently",
				map[string]any{"expected_revision": req.ExpectedRevision})
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}

	// Write indexes for all asset items (delivery_count, last_delivered_at, etc.)
	for _, assetID := range assetIDs {
		if err := h.repo.WriteIndexes(c.Request.Context(), assetID, d); err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
	}

	audit.Log(c.Request.Context(), "delivery.commit_c2", "delivery", []string{d.DeliveryID}, map[string]any{
		"customer_id": d.CustomerID,
		"asset_count": len(assetIDs),
	})

	body := gin.H{
		"delivery_id":  d.DeliveryID,
		"customer_id":  d.CustomerID,
		"status":       d.Status,
		"delivered_at": d.DeliveredAt,
		"approved_by":  d.ApprovedBy,
		"asset_count":  d.AssetCount,
		"version":      d.Version,
	}
	if len(warnings) > 0 {
		body["warnings"] = warnings
	}
	c.JSON(http.StatusOK, body)
}

// ─── Delivery operations: cancel / retry / ack ──────────────────────────────

// HandleCancel cancels a pending or delivered delivery (CYB-1104).
// POST /api/v1/deliveries/:id/cancel
func (h *Handler) HandleCancel(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}
	var req struct {
		CancelledBy  string `json:"cancelled_by"`
		CancelReason string `json:"cancel_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
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

	// Validate allowed statuses: pending or delivered.
	if d.Status != models.DeliveryStatusPending && d.Status != models.DeliveryStatusDelivered {
		httpresp.Unprocessable(c, httpresp.CodeInvalidState,
			"can only cancel pending or delivered deliveries",
			map[string]any{"current_status": d.Status})
		return
	}

	// Use state machine for pending→cancelled transition.
	if d.Status == models.DeliveryStatusPending {
		if err := models.ValidateTransition(d.Status, models.DeliveryStatusCancelled); err != nil {
			httpresp.Unprocessable(c, httpresp.CodeInvalidStateTransition, err.Error(), nil)
			return
		}
	}

	now := time.Now()
	d.Status = models.DeliveryStatusCancelled
	d.CancelledAt = &now
	d.CancelledBy = req.CancelledBy
	d.CancelReason = req.CancelReason

	if err := h.repo.Update(c.Request.Context(), d, d.Version); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	audit.Log(c.Request.Context(), "delivery.cancel", "delivery", []string{d.DeliveryID}, map[string]any{
		"customer_id":   d.CustomerID,
		"cancelled_by":  req.CancelledBy,
		"cancel_reason": req.CancelReason,
	})

	c.JSON(http.StatusOK, gin.H{
		"delivery_id":   d.DeliveryID,
		"status":        d.Status,
		"cancelled_at":  d.CancelledAt,
		"cancelled_by":  d.CancelledBy,
		"cancel_reason": d.CancelReason,
	})
}

// HandleRetry creates a new delivery from a failed or cancelled one (CYB-1105).
// POST /api/v1/deliveries/:id/retry
func (h *Handler) HandleRetry(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}

	old, err := h.repo.Get(c.Request.Context(), deliveryID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if old == nil {
		httpresp.NotFound(c, httpresp.CodeDeliveryNotFound, "delivery not found")
		return
	}

	// Only allow retry from failed or cancelled deliveries.
	if old.Status != models.DeliveryStatusFailed && old.Status != models.DeliveryStatusCancelled {
		httpresp.Unprocessable(c, httpresp.CodeInvalidState,
			"can only retry failed or cancelled deliveries",
			map[string]any{"current_status": old.Status})
		return
	}

	// Fetch items from old delivery.
	items, err := h.repo.ListItems(c.Request.Context(), deliveryID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	now := time.Now()
	d := &models.Delivery{
		DeliveryID:  uuid.NewString(),
		CustomerID:  old.CustomerID,
		Status:      models.DeliveryStatusPending,
		ContractID:  old.ContractID,
		Note:        old.Note,
		Owner:       old.Owner,
		AssetCount:  len(items),
		CreatedAt:   now,
	}

	var assetIDs []string
	for _, item := range items {
		assetIDs = append(assetIDs, item.AssetID)
	}

	if err := h.commitDelivery(c.Request.Context(), d, assetIDs, c.GetHeader("X-Request-ID")); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	// Reference the original delivery for traceability.
	audit.Log(c.Request.Context(), "delivery.retry", "delivery", []string{d.DeliveryID, old.DeliveryID}, map[string]any{
		"customer_id":      d.CustomerID,
		"original_delivery": old.DeliveryID,
		"asset_count":      d.AssetCount,
	})

	c.JSON(http.StatusCreated, gin.H{
		"delivery_id":      d.DeliveryID,
		"customer_id":      d.CustomerID,
		"status":           d.Status,
		"asset_count":      d.AssetCount,
		"original_delivery": old.DeliveryID,
	})
}

// HandleAck acknowledges a delivered delivery (CYB-1106).
// POST /api/v1/deliveries/:id/ack
func (h *Handler) HandleAck(c *gin.Context) {
	deliveryID := c.Param("id")
	if _, err := uuid.Parse(deliveryID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid delivery_id: must be a valid UUID", nil)
		return
	}
	var req struct {
		AcknowledgedBy string `json:"acknowledged_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	req.AcknowledgedBy = strings.TrimSpace(req.AcknowledgedBy)
	if req.AcknowledgedBy == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "acknowledged_by is required", nil)
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

	if d.Status != models.DeliveryStatusDelivered {
		httpresp.Unprocessable(c, httpresp.CodeInvalidState,
			"can only acknowledge delivered deliveries",
			map[string]any{"current_status": d.Status})
		return
	}

	now := time.Now()
	d.AcknowledgedAt = &now
	d.AcknowledgedBy = req.AcknowledgedBy

	if err := h.repo.Update(c.Request.Context(), d, d.Version); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}

	audit.Log(c.Request.Context(), "delivery.ack", "delivery", []string{d.DeliveryID}, map[string]any{
		"customer_id":     d.CustomerID,
		"acknowledged_by": req.AcknowledgedBy,
	})

	c.JSON(http.StatusOK, gin.H{
		"delivery_id":      d.DeliveryID,
		"status":           d.Status,
		"acknowledged_at":  d.AcknowledgedAt,
		"acknowledged_by":  d.AcknowledgedBy,
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
