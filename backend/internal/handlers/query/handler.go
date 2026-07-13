package query

import (
	"context"
	"encoding/json"
	"errors"
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

// applyESRecall runs ES recall in place on compiled and reports recallErrored:
// true when ES was supposed to run but Compile/Execute failed, so the caller
// must fall through to the PG fallback rather than trust MatchTotal==0. Returns
// false when ES is disabled or ran successfully (including a real 0-hit result).
func (h *Handler) applyESRecall(ctx context.Context, plan *queryplan.Plan, compiled *queryir.CompiledQuery) bool {
	if !plan.UseESRecall || h.esExecutor == nil {
		return false
	}
	body, err := h.esExecutor.Compile(plan, false)
	if err != nil {
		compiled.Warnings = append(compiled.Warnings, "elasticsearch compile unsupported; fell back to postgres-only execution")
		return true
	}
	esRecallStarted := time.Now()
	esCompiled, err := h.esExecutor.Execute(ctx, body, true)
	if err != nil {
		metrics.QueryRunPhaseDurationSeconds.WithLabelValues("es_recall", "error").Observe(time.Since(esRecallStarted).Seconds())
		compiled.Warnings = append(compiled.Warnings, "elasticsearch unavailable; fell back to postgres-only execution")
		return true
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
	if len(esCompiled.ESResults) > 0 {
		normalized := make([]map[string]any, 0, len(esCompiled.ESResults))
		for _, esDoc := range esCompiled.ESResults {
			normalized = append(normalized, normalizeESAsset(esDoc))
		}
		compiled.ESResults = normalized
	}
	compiled.MatchTotal = esCompiled.MatchTotal
	return false
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

func (h *Handler) executeCompiledRun(ctx context.Context, plan *queryplan.Plan, compiled *queryir.CompiledQuery) (any, int64, error) {
	recallErrored := h.applyESRecall(ctx, plan, compiled)
	if len(compiled.ESResults) > 0 {
		return compiled.ESResults, compiled.MatchTotal, nil
	}
	// ES recall returned 0 results — short circuit, no PG fallback needed.
	// ES is the authoritative search oracle for fulltext; if it says 0, trust it.
	// Only when ES actually ran, though: on Compile/Execute error (recallErrored)
	// fall through to the PG _fulltext fallback instead of reporting a fake empty.
	if plan.UseESRecall && !recallErrored && compiled.MatchTotal == 0 {
		return []*models.Asset{}, 0, nil
	}

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

	// Save original request context before errgroup — errgroup.WithContext
	// derives a new context that is cancelled by g.Wait() after all goroutines
	// complete (Go stdlib errgroup v0.5+ behaviour). Any fallback PG queries
	// after g.Wait() that use the errgroup context would immediately fail with
	// "context canceled". See internal issue CYB-xxx.
	reqCtx := ctx
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
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
		eg.Go(func() error {
			var err error
			esOut, err = h.fetchESFacetsOrTotal(ctx, plan, skipPGCount)
			esErr = err
			return nil // ES failure is non-fatal; handled in fallback below
		})
	}

	if err := eg.Wait(); err != nil {
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

	// Use reqCtx (not ctx) here — ctx is cancelled by eg.Wait() above.
	if skipPGCount {
		switch {
		case esOut != nil && esOut.MatchTotal > 0:
			total = esOut.MatchTotal
		case esErr != nil || esOut == nil:
			compiled.Warnings = append(compiled.Warnings, "elasticsearch unavailable; used postgres count")
			items, total, pgErr = h.pgExecutor.Execute(reqCtx, h.assetUC, compiled)
			if pgErr != nil {
				return nil, 0, pgErr
			}
		default:
			compiled.Warnings = append(compiled.Warnings, "elasticsearch total unavailable; used postgres count")
			items, total, pgErr = h.pgExecutor.Execute(reqCtx, h.assetUC, compiled)
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
	switch {
	case errors.Is(err, filter.ErrUnknownField):
		httpresp.Unprocessable(c, httpresp.CodeUnsupportedField, err.Error(), nil)
	case errors.Is(err, queryir.ErrUnsupportedOperator):
		httpresp.Unprocessable(c, httpresp.CodeUnsupportedOperator, err.Error(), nil)
	default:
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
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

// normalizeESAsset converts an ES _source document map to match the PG Asset JSON
// format that the frontend expects. ES stores nested tags as {key, value, ...} and
// algos as {name, version, status}, while PG returns tags_flat + tags_detailed and
// algo_results as a flat map with composite keys.
func normalizeESAsset(esDoc map[string]any) map[string]any {
	out := make(map[string]any, len(esDoc))

	// Copy over all root-level fields that share the same name in ES and PG.
	for _, key := range []string{
		"asset_id", "mcap_file_id", "segment_locator", "asset_type",
		"lifecycle_state", "status", "owner", "reviewer", "notes",
		"start_timestamp_ns", "end_timestamp_ns", "duration_ms",
		"delivery_count", "asset_level", "version", "is_deleted",
		"retention_tier", "storage_uri", "thumb_uri",
		"parent_asset_id", "root_asset_id", "tenant_id", "project_id",
		"last_delivered_to", "last_delivered_at", "expire_at",
		"logical_asset_id", "revision", "is_current",
		"segment_index", "parent_start_offset_ms", "parent_end_offset_ms",
		"split_method", "split_algo_name", "split_algo_version",
		"split_run_id", "split_reason",
		"created_at", "updated_at",
		"metadata", "files_json",
		"algo_inputs_uris", "annot_inputs_uris",
		"actions",
	} {
		if v, ok := esDoc[key]; ok {
			out[key] = v
		}
	}

	// Map ES tags_flat → PG tags (both are flat key→value maps).
	if v, ok := esDoc["tags_flat"]; ok {
		out["tags"] = v
	}

	// Map ES tags[] → PG tags_detailed[] with field renames.
	if raw, ok := esDoc["tags"]; ok {
		if arr, ok := raw.([]any); ok {
			detailed := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					entry := map[string]any{
						"tag_key":     m["key"],
						"tag_value":   m["value"],
						"source_type": m["source_type"],
						"source_name": m["source_name"],
					}
					if taggedAt, ok := m["tagged_at"]; ok {
						entry["tagged_at"] = taggedAt
					}
					detailed = append(detailed, entry)
				}
			}
			if len(detailed) > 0 {
				out["tags_detailed"] = detailed
			}
		}
	}

	// Flatten ES algos[] → PG algo_results with composite keys.
	if raw, ok := esDoc["algos"]; ok {
		if arr, ok := raw.([]any); ok {
			algoResults := make(map[string]string, len(arr)*3)
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					name, _ := m["name"].(string)
					version, _ := m["version"].(string)
					if name == "" || version == "" {
						continue
					}
					prefix := name + "@" + version + ":"
					if status, ok := m["status"].(string); ok {
						algoResults[prefix+"status"] = status
					}
					if runID, ok := m["run_id"].(string); ok {
						algoResults[prefix+"run_id"] = runID
					}
					if finishedAt, ok := m["finished_at"].(string); ok {
						algoResults[prefix+"finished_at"] = finishedAt
					}
				}
			}
			if len(algoResults) > 0 {
				out["algo_results"] = algoResults
			}
		}
	}

	// mcap → extract mcap_file_id only if available (ES already has it at root level).
	// Skip: recorded_at, mcap (nested), lineage_*, dataset, annotation_result, ml_model, evaluation_report.

	return out
}
