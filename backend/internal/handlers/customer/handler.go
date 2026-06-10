package customer

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

var customerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{2,31}$`)

var (
	ErrInvalidCustomerID = errors.New("customer_id must match ^[a-z][a-z0-9_-]{2,31}$")
	ErrCustomerNotFound  = errors.New("customer not found")
)

type Handler struct {
	repo repository.CustomerRepository
}

func New(repo repository.CustomerRepository) *Handler {
	return &Handler{repo: repo}
}

func ValidateCustomerID(id string) error {
	id = strings.TrimSpace(id)
	if !customerIDPattern.MatchString(id) {
		return ErrInvalidCustomerID
	}
	return nil
}

// Create registers a new customer.
// @Summary      Create customer
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        body body object true "Create customer request"
// @Success      201 {object} models.Customer
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /customers [post]
func (h *Handler) Create(c *gin.Context) {
	var req struct {
		CustomerID     string                 `json:"customer_id" binding:"required"`
		DisplayName    string                 `json:"display_name" binding:"required"`
		LegalName      string                 `json:"legal_name"`
		Status         string                 `json:"status"`
		Region         string                 `json:"region"`
		SLATier        string                 `json:"sla_tier"`
		AccountOwner   string                 `json:"account_owner"`
		ComplianceTags []interface{}          `json:"compliance_tags"`
		ExcludeTags    []interface{}          `json:"exclude_tags"`
		Metadata       map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	if err := ValidateCustomerID(req.CustomerID); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	cust := &models.Customer{
		CustomerID:     req.CustomerID,
		DisplayName:    strings.TrimSpace(req.DisplayName),
		LegalName:      strings.TrimSpace(req.LegalName),
		Status:         strings.TrimSpace(req.Status),
		Region:         strings.TrimSpace(req.Region),
		SLATier:        strings.TrimSpace(req.SLATier),
		AccountOwner:   strings.TrimSpace(req.AccountOwner),
		ComplianceTags: req.ComplianceTags,
		ExcludeTags:    req.ExcludeTags,
		Metadata:       req.Metadata,
		RowVersion:     1,
	}
	if cust.Status == "" {
		cust.Status = "active"
	}
	if cust.SLATier == "" {
		cust.SLATier = "standard"
	}
	if err := h.repo.Insert(c.Request.Context(), cust); err != nil {
		if errors.Is(err, repository.ErrDuplicateCustomerID) {
			httpresp.Conflict(c, httpresp.CodeInvalidArgument, "customer_id already exists", nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	out, err := h.repo.Get(c.Request.Context(), cust.CustomerID)
	if err != nil || out == nil {
		httpresp.Internal(c, "customer created but could not be loaded")
		return
	}
	c.JSON(201, out)
}

// Get returns one customer by id.
// @Summary      Get customer
// @Tags         customers
// @Produce      json
// @Param        customer_id path string true "Customer ID"
// @Success      200 {object} models.Customer
// @Failure      404 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /customers/{customer_id} [get]
func (h *Handler) Get(c *gin.Context) {
	customerID := strings.TrimSpace(c.Param("customer_id"))
	if customerID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "customer_id is required", nil)
		return
	}
	cust, err := h.repo.Get(c.Request.Context(), customerID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if cust == nil {
		httpresp.NotFound(c, "CUSTOMER_NOT_FOUND", "customer not found")
		return
	}
	c.JSON(200, cust)
}

// Update patches customer fields.
// @Summary      Update customer
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        customer_id path string true "Customer ID"
// @Param        body body object true "Patch customer request"
// @Success      200 {object} models.Customer
// @Failure      404 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Security     DatabrewToken
// @Router       /customers/{customer_id} [patch]
func (h *Handler) Update(c *gin.Context) {
	customerID := strings.TrimSpace(c.Param("customer_id"))
	if customerID == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "customer_id is required", nil)
		return
	}
	existing, err := h.repo.Get(c.Request.Context(), customerID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if existing == nil {
		httpresp.NotFound(c, "CUSTOMER_NOT_FOUND", "customer not found")
		return
	}
	var req struct {
		DisplayName    *string                `json:"display_name"`
		LegalName      *string                `json:"legal_name"`
		Status         *string                `json:"status"`
		Region         *string                `json:"region"`
		SLATier        *string                `json:"sla_tier"`
		AccountOwner   *string                `json:"account_owner"`
		ComplianceTags []interface{}          `json:"compliance_tags"`
		ExcludeTags    []interface{}          `json:"exclude_tags"`
		Metadata       map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if req.DisplayName != nil {
		existing.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.LegalName != nil {
		existing.LegalName = strings.TrimSpace(*req.LegalName)
	}
	if req.Status != nil {
		existing.Status = strings.TrimSpace(*req.Status)
	}
	if req.Region != nil {
		existing.Region = strings.TrimSpace(*req.Region)
	}
	if req.SLATier != nil {
		existing.SLATier = strings.TrimSpace(*req.SLATier)
	}
	if req.AccountOwner != nil {
		existing.AccountOwner = strings.TrimSpace(*req.AccountOwner)
	}
	if req.ComplianceTags != nil {
		existing.ComplianceTags = req.ComplianceTags
	}
	if req.ExcludeTags != nil {
		existing.ExcludeTags = req.ExcludeTags
	}
	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}
	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		if errors.Is(err, repository.ErrOptimisticLock) {
			httpresp.Conflict(c, httpresp.CodeConcurrentConflict, "customer was modified concurrently", nil)
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	out, err := h.repo.Get(c.Request.Context(), customerID)
	if err != nil || out == nil {
		httpresp.Internal(c, "customer updated but could not be loaded")
		return
	}
	c.JSON(200, out)
}

// List returns customers with optional filters and cursor pagination.
// @Summary      List customers
// @Tags         customers
// @Produce      json
// @Param        status    query string false "Filter by status"
// @Param        sla_tier  query string false "Filter by SLA tier"
// @Param        region    query string false "Filter by region"
// @Param        limit     query int    false "Page size (default 50, max 200)"
// @Param        cursor    query string false "Cursor: customer_id to start after"
// @Success      200 {object} object
// @Security     DatabrewToken
// @Router       /customers [get]
func (h *Handler) List(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	slaTier := strings.TrimSpace(c.Query("sla_tier"))
	region := strings.TrimSpace(c.Query("region"))
	cursor := strings.TrimSpace(c.Query("cursor"))

	limit := 50
	if s := c.Query("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	items, err := h.repo.List(c.Request.Context(), status, slaTier, region, limit, cursor)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []*models.Customer{}
	}

	nextCursor := ""
	if len(items) == limit {
		nextCursor = items[len(items)-1].CustomerID
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"limit":       limit,
		"next_cursor": nextCursor,
	})
}
