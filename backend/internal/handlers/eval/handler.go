package eval

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Handler serves eval/metrics endpoints.
type Handler struct {
	evalRepo  *postgres.EvalRepo
	metricReg *config.MetricRegistry
}

func New(evalRepo *postgres.EvalRepo, metricReg *config.MetricRegistry, _ repository.AssetEventRepository) *Handler {
	return &Handler{evalRepo: evalRepo, metricReg: metricReg}
}

// ─── K1: POST /api/v1/assets/:id/eval-results ────────────────────────────────

// ReportEvalResult writes an eval result and projects queryable metrics.
func (h *Handler) ReportEvalResult(c *gin.Context) {
	assetID := c.Param("id")
	start := time.Now()
	outcome := "error"
	defer func() {
		metrics.BackendEvalWriteRequestsTotal.WithLabelValues(outcome).Inc()
		metrics.BackendEvalWriteDurationSeconds.WithLabelValues(outcome).Observe(time.Since(start).Seconds())
	}()

	var req struct {
		TargetType       string         `json:"target_type"`
		TargetID         string         `json:"target_id"`
		EvalName         string         `json:"eval_name" binding:"required"`
		EvalVersion      string         `json:"eval_version" binding:"required"`
		ParameterVersion *string        `json:"parameter_version"`
		RunID            *string        `json:"run_id"`
		Status           string         `json:"status"`
		ResultPayload    map[string]any `json:"result_payload"`
		OutputURI        *string        `json:"output_uri"`
		SummaryURI       *string        `json:"summary_uri"`
		SourceType       string         `json:"source_type"`
		SourceName       *string        `json:"source_name"`
		SourceVersion    *string        `json:"source_version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		outcome = "invalid_argument"
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "ok"
	}
	if req.TargetType == "" {
		req.TargetType = "segment"
	}
	if req.SourceType == "" {
		req.SourceType = "algo"
	}
	if req.ResultPayload == nil {
		req.ResultPayload = map[string]any{}
	}

	// Build queryable key → metric_type map from registry
	queryableKeys := map[string]string{}
	unregisteredOrUnqueryable := 0
	for key := range req.ResultPayload {
		if def, ok := h.metricReg.Get(key); ok && def.Queryable {
			queryableKeys[key] = def.MetricType
			continue
		}
		unregisteredOrUnqueryable++
	}
	metrics.BackendEvalMetricKeysTotal.WithLabelValues("registered_queryable").Add(float64(len(queryableKeys)))
	metrics.BackendEvalMetricKeysTotal.WithLabelValues("unregistered_or_unqueryable").Add(float64(unregisteredOrUnqueryable))

	res, err := h.evalRepo.Write(c.Request.Context(), postgres.EvalResultWriteInput{
		AssetID:          assetID,
		TargetType:       req.TargetType,
		TargetID:         req.TargetID,
		EvalName:         req.EvalName,
		EvalVersion:      req.EvalVersion,
		ParameterVersion: req.ParameterVersion,
		RunID:            req.RunID,
		Status:           req.Status,
		ResultPayload:    req.ResultPayload,
		OutputURI:        req.OutputURI,
		SummaryURI:       req.SummaryURI,
		SourceType:       req.SourceType,
		SourceName:       req.SourceName,
		SourceVersion:    req.SourceVersion,
		QueryableKeys:    queryableKeys,
		RequestID:        c.GetHeader("X-Request-ID"),
	})
	if err != nil {
		outcome = "write_error"
		httpresp.Internal(c, err.Error())
		return
	}

	outcome = "ok"
	c.JSON(http.StatusCreated, res)
}

// ─── K2: GET /api/v1/assets/:id/eval-results ─────────────────────────────────

func (h *Handler) ListEvalResults(c *gin.Context) {
	assetID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	results, err := h.evalRepo.ListByAsset(c.Request.Context(), assetID, limit)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if results == nil {
		results = []*postgres.EvalResult{}
	}
	c.JSON(http.StatusOK, gin.H{"items": results, "total": len(results)})
}

// ─── K3: GET /api/v1/assets/:id/metrics ──────────────────────────────────────

func (h *Handler) ListMetrics(c *gin.Context) {
	assetID := c.Param("id")

	metrics, err := h.evalRepo.ListMetricsByAsset(c.Request.Context(), assetID)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if metrics == nil {
		metrics = []*postgres.AssetMetric{}
	}
	c.JSON(http.StatusOK, gin.H{"items": metrics, "asset_id": assetID})
}

// ─── K5: GET /api/v1/metrics/registry ────────────────────────────────────────

func (h *Handler) GetRegistry(c *gin.Context) {
	defs := h.metricReg.All()
	sort.Slice(defs, func(i, j int) bool { return defs[i].Key < defs[j].Key })
	c.JSON(http.StatusOK, gin.H{"items": defs})
}

// ─── K6: POST /api/v1/metrics:search ─────────────────────────────────────────

func (h *Handler) SearchByMetrics(c *gin.Context) {
	var req struct {
		Filters struct {
			LifecycleState string `json:"lifecycle_state"`
			Metrics        []struct {
				MetricKey string  `json:"metric_key" binding:"required"`
				Op        string  `json:"op" binding:"required"`
				Value     float64 `json:"value"`
			} `json:"metrics"`
		} `json:"filters"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	if len(req.Filters.Metrics) == 0 {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "filters.metrics must have at least one entry", nil)
		return
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	filters := make([]postgres.MetricFilter, 0, len(req.Filters.Metrics))
	for _, f := range req.Filters.Metrics {
		op := normalizeMetricOp(f.Op)
		if op == "" {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid metric op", map[string]any{
				"op":      f.Op,
				"allowed": []string{"eq", "gt", "gte", "lt", "lte", "=", ">", ">=", "<", "<="},
			})
			return
		}
		filters = append(filters, postgres.MetricFilter{
			MetricKey: f.MetricKey,
			Op:        op,
			Value:     f.Value,
		})
	}

	ids, total, err := h.evalRepo.SearchByMetrics(c.Request.Context(),
		filters,
		req.Filters.LifecycleState,
		req.Page, req.PageSize,
	)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, gin.H{
		"asset_ids": ids,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
	})
}

func normalizeMetricOp(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "eq", "=":
		return "="
	case "gt", ">":
		return ">"
	case "gte", ">=":
		return ">="
	case "lt", "<":
		return "<"
	case "lte", "<=":
		return "<="
	default:
		return ""
	}
}
