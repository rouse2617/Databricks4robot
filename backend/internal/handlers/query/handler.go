package query

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	corees "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/metrics"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	esexec "github.com/CyberOrigin2077/cyber-databrew/internal/queryexec/elasticsearch"
	pgexec "github.com/CyberOrigin2077/cyber-databrew/internal/queryexec/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

type Handler struct {
	assetUC       *assetUC.Usecase
	fieldRegistry *config.QueryFieldRegistry
	planner       *queryplan.PGBridgePlanner
	pgExecutor    *pgexec.Executor
	esExecutor    *esexec.Executor
	savedQueries  *postgres.SavedQueryRepo
}

func New(assetUsecase *assetUC.Usecase, fieldRegistry *config.QueryFieldRegistry, esClient *corees.Client, savedQueries *postgres.SavedQueryRepo) *Handler {
	return &Handler{
		assetUC:       assetUsecase,
		fieldRegistry: fieldRegistry,
		planner:       queryplan.NewPGBridgePlanner(esClient != nil),
		pgExecutor:    pgexec.New(),
		esExecutor:    esexec.New(esClient),
		savedQueries:  savedQueries,
	}
}

func applyIncludeHistoryQueryParam(c *gin.Context, req *queryir.QueryRequest) {
	if c.Query("include_history") == "true" {
		req.Scope.IncludeHistory = true
	}
}

// Validate validates and compiles query IR without executing it.
func (h *Handler) Validate(c *gin.Context) {
	var req queryir.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	applyIncludeHistoryQueryParam(c, &req)
	compiled, err := h.compileAndValidate(c.Request.Context(), req)
	if err != nil {
		writeQueryError(c, err)
		return
	}
	fieldCapabilities := h.fieldCapabilitiesFor(req.Scope.Resource, compiled.FieldCapabilities)
	c.JSON(200, gin.H{
		"valid":              true,
		"normalized_query":   compiled.NormalizedQuery,
		"warnings":           compiled.Warnings,
		"field_capabilities": fieldCapabilities,
		"debug_plan":         compiled.DebugPlan,
	})
}

// Run validates and executes query IR on assets.
func (h *Handler) Run(c *gin.Context) {
	started := time.Now()
	outcome := "ok"
	defer func() {
		metrics.QueryRunRequestsTotal.WithLabelValues(outcome).Inc()
		metrics.QueryRunDurationSeconds.WithLabelValues(outcome).Observe(time.Since(started).Seconds())
	}()

	var req queryir.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		outcome = "bad_request"
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	applyIncludeHistoryQueryParam(c, &req)
	compileStarted := time.Now()
	plan, compiled, err := h.compileForRun(c.Request.Context(), req)
	if err != nil {
		outcome = "compile_error"
		metrics.QueryRunPhaseDurationSeconds.WithLabelValues("compile_validate", "error").Observe(time.Since(compileStarted).Seconds())
		writeQueryError(c, err)
		return
	}
	metrics.QueryRunPhaseDurationSeconds.WithLabelValues("compile_validate", "ok").Observe(time.Since(compileStarted).Seconds())

	items, total, err := h.executeCompiledRun(c.Request.Context(), plan, compiled)
	if err != nil {
		outcome = "execute_error"
		httpresp.Internal(c, err.Error())
		return
	}
	metrics.QueryRunCandidateIDs.Observe(float64(len(compiled.CandidateAssetIDs)))
	offset, limit := (compiled.Page-1)*compiled.PageSize, compiled.PageSize
	columnDefs := buildResultColumns(req.Select.Fields)
	c.JSON(200, gin.H{
		"items":       items,
		"total":       total,
		"page":        compiled.Page,
		"page_size":   compiled.PageSize,
		"columns":     req.Select.Fields,
		"column_defs": columnDefs,
		"offset":      offset,
		"limit":       limit,
		"facets":      compiled.Facets,
		"warnings":    compiled.Warnings,
		"debug_plan":  compiled.DebugPlan,
	})
}

func (h *Handler) compileAndValidate(ctx context.Context, req queryir.QueryRequest) (*queryir.CompiledQuery, error) {
	plan, err := h.planner.Plan(req)
	if err != nil {
		return nil, err
	}
	compiled, err := h.pgExecutor.Compile(plan)
	if err != nil {
		return nil, err
	}
	if (plan.UseESRecall || plan.UseESFacets) && h.esExecutor != nil {
		body, err := h.esExecutor.Compile(plan, false)
		if err != nil {
			compiled.Warnings = append(compiled.Warnings, "elasticsearch compile unsupported; fell back to postgres-only execution")
		} else {
			phaseLabel := "es_recall"
			if !plan.UseESRecall && plan.UseESFacets {
				phaseLabel = "es_facet"
			}
			esRecallStarted := time.Now()
			esCompiled, err := h.esExecutor.Execute(ctx, body, plan.UseESRecall)
			if err != nil {
				metrics.QueryRunPhaseDurationSeconds.WithLabelValues(phaseLabel, "error").Observe(time.Since(esRecallStarted).Seconds())
				compiled.Warnings = append(compiled.Warnings, "elasticsearch unavailable; fell back to postgres-only execution")
			} else {
				metrics.QueryRunPhaseDurationSeconds.WithLabelValues(phaseLabel, "ok").Observe(time.Since(esRecallStarted).Seconds())
				if len(esCompiled.Warnings) > 0 {
					compiled.Warnings = append(compiled.Warnings, esCompiled.Warnings...)
				}
				if plan.UseESRecall {
					// When ES recall returns 0 candidates we silently fall back to a
					// PG scan to defend against index lag. We do NOT surface a
					// user-facing warning here — most "0 results" cases are
					// genuinely empty filters (e.g. lifecycle_state=processing
					// when nothing is processing), and a permanent warning is a
					// false positive. Real ES lag is observable via the
					// /search/sync-progress watermark on the Settings page and
					// the elasticsearch_sync_* Prometheus metrics.
					if len(esCompiled.CandidateAssetIDs) == 0 {
						compiled.CandidateAssetIDs = nil
					} else {
						compiled.CandidateAssetIDs = esCompiled.CandidateAssetIDs
					}
				}
				if plan.UseESFacets {
					compiled.Facets = esCompiled.Facets
				}
			}
		}
	}
	if _, _, err := pgexec.BuildExprWhereClause(compiled.NormalizedQuery.Where, 1); err != nil {
		return nil, err
	}
	if _, _, err := filter.ResolveSortBy(compiled.SortBy, 1); err != nil {
		return nil, err
	}
	return compiled, nil
}

func (h *Handler) compileForRun(ctx context.Context, req queryir.QueryRequest) (*queryplan.Plan, *queryir.CompiledQuery, error) {
	plan, err := h.planner.Plan(req)
	if err != nil {
		return nil, nil, err
	}
	compiled, err := h.pgExecutor.Compile(plan)
	if err != nil {
		return nil, nil, err
	}
	if _, _, err := pgexec.BuildExprWhereClause(compiled.NormalizedQuery.Where, 1); err != nil {
		return nil, nil, err
	}
	if _, _, err := filter.ResolveSortBy(compiled.SortBy, 1); err != nil {
		return nil, nil, err
	}
	return plan, compiled, nil
}

func canSkipPGCount(plan *queryplan.Plan, compiled *queryir.CompiledQuery) bool {
	if compiled.NormalizedQuery.Where != nil {
		return false
	}
	if plan.UseESRecall && len(compiled.CandidateAssetIDs) > 0 {
		return false
	}
	return true
}

func (h *Handler) applyESRecall(ctx context.Context, plan *queryplan.Plan, compiled *queryir.CompiledQuery) {
	if !plan.UseESRecall || h.esExecutor == nil {
		return
	}
	body, err := h.esExecutor.Compile(plan, false)
	if err != nil {
		compiled.Warnings = append(compiled.Warnings, "elasticsearch compile unsupported; fell back to postgres-only execution")
		return
	}
	esRecallStarted := time.Now()
	esCompiled, err := h.esExecutor.Execute(ctx, body, true)
	if err != nil {
		metrics.QueryRunPhaseDurationSeconds.WithLabelValues("es_recall", "error").Observe(time.Since(esRecallStarted).Seconds())
		compiled.Warnings = append(compiled.Warnings, "elasticsearch unavailable; fell back to postgres-only execution")
		return
	}
	metrics.QueryRunPhaseDurationSeconds.WithLabelValues("es_recall", "ok").Observe(time.Since(esRecallStarted).Seconds())
	if len(esCompiled.Warnings) > 0 {
		compiled.Warnings = append(compiled.Warnings, esCompiled.Warnings...)
	}
	if len(esCompiled.CandidateAssetIDs) == 0 {
		compiled.CandidateAssetIDs = nil
	} else {
		compiled.CandidateAssetIDs = esCompiled.CandidateAssetIDs
	}
}

func (h *Handler) fetchESFacetsOrTotal(ctx context.Context, plan *queryplan.Plan, trackTotalHits bool) (*queryir.CompiledQuery, error) {
	var body map[string]any
	var err error
	if trackTotalHits && !plan.UseESFacets {
		body, err = h.esExecutor.CompileCountOnly(plan)
	} else {
		body, err = h.esExecutor.Compile(plan, trackTotalHits)
	}
	if err != nil {
		return nil, err
	}
	phaseLabel := "es_facet"
	if !plan.UseESFacets {
		phaseLabel = "es_total"
	}
	started := time.Now()
	esCompiled, err := h.esExecutor.Execute(ctx, body, false)
	if err != nil {
		metrics.QueryRunPhaseDurationSeconds.WithLabelValues(phaseLabel, "error").Observe(time.Since(started).Seconds())
		return nil, err
	}
	metrics.QueryRunPhaseDurationSeconds.WithLabelValues(phaseLabel, "ok").Observe(time.Since(started).Seconds())
	return esCompiled, nil
}

func (h *Handler) executeCompiledRun(ctx context.Context, plan *queryplan.Plan, compiled *queryir.CompiledQuery) ([]*models.Asset, int64, error) {
	h.applyESRecall(ctx, plan, compiled)

	skipPGCount := canSkipPGCount(plan, compiled)
	needESFacets := plan.UseESFacets && h.esExecutor != nil
	needESTotalOnly := skipPGCount && h.esExecutor != nil && !needESFacets

	var (
		items []*models.Asset
		total int64
		esOut *queryir.CompiledQuery
		pgErr error
		esErr error
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		pgStarted := time.Now()
		if skipPGCount {
			items, pgErr = h.pgExecutor.ExecutePage(ctx, h.assetUC, compiled)
		} else {
			items, total, pgErr = h.pgExecutor.Execute(ctx, h.assetUC, compiled)
		}
		outcome := "ok"
		if pgErr != nil {
			outcome = "error"
		}
		metrics.QueryRunPhaseDurationSeconds.WithLabelValues("pg_refine", outcome).Observe(time.Since(pgStarted).Seconds())
		return pgErr
	})

	if needESFacets || needESTotalOnly {
		g.Go(func() error {
			var err error
			esOut, err = h.fetchESFacetsOrTotal(ctx, plan, skipPGCount)
			esErr = err
			return err
		})
	}

	if err := g.Wait(); err != nil {
		if pgErr != nil {
			return nil, 0, pgErr
		}
		return nil, 0, err
	}

	if esOut != nil {
		if len(esOut.Warnings) > 0 {
			compiled.Warnings = append(compiled.Warnings, esOut.Warnings...)
		}
		if needESFacets {
			compiled.Facets = esOut.Facets
		}
	}

	if skipPGCount {
		switch {
		case esOut != nil && esOut.MatchTotal > 0:
			total = esOut.MatchTotal
		case esErr != nil || esOut == nil:
			compiled.Warnings = append(compiled.Warnings, "elasticsearch unavailable; used postgres count")
			items, total, pgErr = h.pgExecutor.Execute(ctx, h.assetUC, compiled)
			if pgErr != nil {
				return nil, 0, pgErr
			}
		default:
			compiled.Warnings = append(compiled.Warnings, "elasticsearch total unavailable; used postgres count")
			items, total, pgErr = h.pgExecutor.Execute(ctx, h.assetUC, compiled)
			if pgErr != nil {
				return nil, 0, pgErr
			}
		}
	}

	return items, total, nil
}

func buildResultColumns(fields []string) []queryir.ResultColumn {
	if len(fields) == 0 {
		return []queryir.ResultColumn{}
	}
	out := make([]queryir.ResultColumn, 0, len(fields))
	for _, f := range fields {
		name := strings.TrimSpace(f)
		if name == "" {
			continue
		}
		colType := "string"
		nullable := true
		switch name {
		case "asset_id", "mcap_file_id", "created_at", "updated_at":
			nullable = false
		case "duration_ms", "duration_sec", "version", "start_timestamp_ns", "end_timestamp_ns", "delivery_count":
			colType = "number"
		}
		if name == "created_at" || name == "updated_at" || name == "expire_at" || name == "last_delivered_at" {
			colType = "timestamp"
		}
		out = append(out, queryir.ResultColumn{Name: name, Type: colType, Nullable: nullable})
	}
	return out
}

func (h *Handler) fieldCapabilitiesFor(resource string, fallback []queryir.FieldCapabilityBrief) []queryir.FieldCapabilityBrief {
	if h.fieldRegistry == nil {
		return fallback
	}
	out := make([]queryir.FieldCapabilityBrief, 0, len(fallback))
	for _, item := range fallback {
		if engines := h.fieldRegistry.EnginesFor(resource, item.Field); len(engines) > 0 {
			out = append(out, queryir.FieldCapabilityBrief{
				Field:   item.Field,
				Engines: engines,
			})
			continue
		}
		out = append(out, item)
	}
	return out
}

func writeQueryError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown field"):
		httpresp.Unprocessable(c, httpresp.CodeUnsupportedField, msg, nil)
	case strings.Contains(msg, "unsupported operator"):
		httpresp.Unprocessable(c, httpresp.CodeUnsupportedOperator, msg, nil)
	case strings.Contains(msg, "unplannable"):
		httpresp.Unprocessable(c, httpresp.CodeUnplannableQuery, msg, nil)
	case strings.Contains(msg, "unsupported scope.resource"),
		strings.Contains(msg, "unsupported schema_version"),
		strings.Contains(msg, "sort[0].field is required"),
		strings.Contains(msg, "unsupported sort direction"),
		strings.Contains(msg, "missing field in predicate"),
		strings.Contains(msg, "missing operator in predicate"),
		strings.Contains(msg, "invalid where expression"):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, msg, nil)
	default:
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, msg, nil)
	}
}

func (h *Handler) ListSavedQueries(c *gin.Context) {
	if h.savedQueries == nil {
		httpresp.Error(c, 503, httpresp.CodeServiceUnavailable, "saved queries unavailable", nil)
		return
	}
	items, err := h.savedQueries.List(c.Request.Context())
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if items == nil {
		items = []*models.SavedQuery{}
	}
	c.JSON(200, gin.H{"items": items})
}

func (h *Handler) GetSavedQuery(c *gin.Context) {
	if h.savedQueries == nil {
		httpresp.Error(c, 503, httpresp.CodeServiceUnavailable, "saved queries unavailable", nil)
		return
	}
	item, err := h.savedQueries.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if item == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "saved query not found")
		return
	}
	c.JSON(200, item)
}

func (h *Handler) CreateSavedQuery(c *gin.Context) {
	h.upsertSavedQuery(c, true)
}

func (h *Handler) UpdateSavedQuery(c *gin.Context) {
	h.upsertSavedQuery(c, false)
}

func (h *Handler) DeleteSavedQuery(c *gin.Context) {
	if h.savedQueries == nil {
		httpresp.Error(c, 503, httpresp.CodeServiceUnavailable, "saved queries unavailable", nil)
		return
	}
	if err := h.savedQueries.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(200, gin.H{"deleted": true})
}

func (h *Handler) upsertSavedQuery(c *gin.Context, create bool) {
	if h.savedQueries == nil {
		httpresp.Error(c, 503, httpresp.CodeServiceUnavailable, "saved queries unavailable", nil)
		return
	}
	var req struct {
		Name        string               `json:"name" binding:"required"`
		Description string               `json:"description"`
		QueryIRJSON queryir.QueryRequest `json:"query_ir_json"`
		Owner       string               `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	compiled, err := h.compileAndValidate(c.Request.Context(), req.QueryIRJSON)
	if err != nil {
		writeQueryError(c, err)
		return
	}
	item := &models.SavedQuery{
		SavedQueryID:  c.Param("id"),
		Name:          req.Name,
		Description:   req.Description,
		Resource:      compiled.NormalizedQuery.Scope.Resource,
		SchemaVersion: compiled.NormalizedQuery.SchemaVersion,
		Owner:         req.Owner,
		QueryIRJSON:   map[string]interface{}{},
	}
	raw, _ := json.Marshal(compiled.NormalizedQuery)
	_ = json.Unmarshal(raw, &item.QueryIRJSON)

	if create {
		created, err := h.savedQueries.Create(c.Request.Context(), item)
		if err != nil {
			httpresp.Internal(c, err.Error())
			return
		}
		c.JSON(201, created)
		return
	}
	updated, err := h.savedQueries.Update(c.Request.Context(), item)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if updated == nil {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, "saved query not found")
		return
	}
	c.JSON(200, updated)
}
