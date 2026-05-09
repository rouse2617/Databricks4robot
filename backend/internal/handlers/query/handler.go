package query

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	corees "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
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

// Validate validates and compiles query IR without executing it.
func (h *Handler) Validate(c *gin.Context) {
	var req queryir.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
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
	var req queryir.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	compiled, err := h.compileAndValidate(c.Request.Context(), req)
	if err != nil {
		writeQueryError(c, err)
		return
	}
	items, total, err := h.pgExecutor.Execute(c.Request.Context(), h.assetUC, compiled)
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
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
	if plan.UseElasticsearch && h.esExecutor != nil {
		body, err := h.esExecutor.Compile(plan)
		if err != nil {
			compiled.Warnings = append(compiled.Warnings, "elasticsearch compile unsupported; fell back to postgres-only execution")
		} else {
			esCompiled, err := h.esExecutor.Execute(ctx, body)
			if err != nil {
				compiled.Warnings = append(compiled.Warnings, "elasticsearch unavailable; fell back to postgres-only execution")
			} else {
				// Avoid false negatives when ES is behind CDC and returns no candidates.
				if len(esCompiled.CandidateAssetIDs) == 0 {
					compiled.Warnings = append(compiled.Warnings, "elasticsearch recall returned 0 candidates; fell back to postgres scan (possible CDC lag)")
					compiled.CandidateAssetIDs = nil
				} else {
					compiled.CandidateAssetIDs = esCompiled.CandidateAssetIDs
				}
				compiled.Facets = esCompiled.Facets
			}
		}
	}
	if _, _, err := pgexec.BuildExprWhereClause(compiled.NormalizedQuery.Where, 1); err != nil {
		return nil, err
	}
	if _, err := filter.ResolveSortBy(compiled.SortBy); err != nil {
		return nil, err
	}
	return compiled, nil
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
