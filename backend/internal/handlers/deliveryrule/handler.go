package deliveryrule

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type Handler struct {
	repo         repository.DeliveryRuleRepository
	customerRepo repository.CustomerRepository
}

func New(repo repository.DeliveryRuleRepository, customerRepo repository.CustomerRepository) *Handler {
	return &Handler{repo: repo, customerRepo: customerRepo}
}

// POST /api/v1/delivery-rules
func (h *Handler) Create(c *gin.Context) {
	var req struct {
		Name        string          `json:"name" binding:"required"`
		Owner       string          `json:"owner" binding:"required"`
		CustomerID  string          `json:"customer_id"`
		QueryDSL    json.RawMessage `json:"query_dsl" binding:"required"`
		DSLVersion  string          `json:"dsl_version"`
		EnforceMode string          `json:"enforce_mode"`
		RatingScope string          `json:"rating_scope"`
		IsActive    *bool           `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if _, err := deliveryrules.ParseQueryDSL(req.QueryDSL); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	if req.CustomerID != "" && h.customerRepo != nil {
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
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	rule := &models.DeliveryRule{
		Name:        req.Name,
		Owner:       req.Owner,
		CustomerID:  req.CustomerID,
		QueryDSL:    req.QueryDSL,
		DSLVersion:  req.DSLVersion,
		EnforceMode: req.EnforceMode,
		RatingScope: req.RatingScope,
		IsActive:    active,
	}
	if err := h.repo.Insert(c.Request.Context(), rule); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// GET /api/v1/delivery-rules?customer_id=
func (h *Handler) List(c *gin.Context) {
	customerID := strings.TrimSpace(c.Query("customer_id"))
	items, err := h.repo.List(c.Request.Context(), customerID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []*models.DeliveryRule{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
