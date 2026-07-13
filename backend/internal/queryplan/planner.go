package queryplan

import (
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

// FacetEngine picks which engine executes the facet aggregation phase.
//
// FacetEngineAuto is the default: ES when the PG↔ES gap is 0, PG otherwise.
// FacetEngineES / FacetEnginePG are diagnostic overrides via DBK_FACET_ENGINE.
type FacetEngine int

const (
	FacetEngineAuto FacetEngine = iota
	FacetEngineES
	FacetEnginePG
)

// ParseFacetEngine maps env-var values to FacetEngine. Unrecognized / empty
// input returns FacetEngineAuto.
func ParseFacetEngine(s string) FacetEngine {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "es", "elasticsearch":
		return FacetEngineES
	case "pg", "postgres":
		return FacetEnginePG
	default:
		return FacetEngineAuto
	}
}

// PGSupportedFacetFields lists the fields for which the PG facet executor
// (CYB-3384 first pass) can compute counts. Fields outside this set are
// dropped when the planner routes facets to PG so the response stays partial
// but honest instead of quietly wrong.
var PGSupportedFacetFields = map[string]bool{
	"asset_type":      true,
	"lifecycle_state": true,
	"owner":           true,
	"env":             true,
}

// GapProvider is the minimal interface the planner uses to read the last
// observed PG↔ES gap. Implemented by *SyncHealthCache.
type GapProvider interface {
	Gap() int64
}

type Plan struct {
	NormalizedQuery queryir.QueryRequest
	Steps           []queryir.DebugPlanStep
	UseESRecall     bool
	UseESFacets     bool
	// UsePGFacets is set when facet aggregation should be computed in
	// PostgreSQL instead of Elasticsearch (either forced by config or auto-
	// triggered by a non-zero PG↔ES gap). At most one of UseESFacets and
	// UsePGFacets is true.
	UsePGFacets bool
	// PGFacetDroppedFields lists facet fields the request asked for that PG
	// cannot cover in this iteration (mcap.*, tag.*, …). Handlers report
	// these back via a `warnings` entry so the UI can hint at partial data.
	PGFacetDroppedFields []string
}

type PGBridgePlanner struct {
	useElasticsearch bool
	facetEngine      FacetEngine
	gap              GapProvider
}

func NewPGBridgePlanner(useElasticsearch bool) *PGBridgePlanner {
	return &PGBridgePlanner{useElasticsearch: useElasticsearch, facetEngine: FacetEngineAuto}
}

// WithFacetEngine sets the DBK_FACET_ENGINE override. Returns p for chaining.
func (p *PGBridgePlanner) WithFacetEngine(engine FacetEngine) *PGBridgePlanner {
	p.facetEngine = engine
	return p
}

// WithGapProvider wires the sync-health cache. Returns p for chaining.
func (p *PGBridgePlanner) WithGapProvider(gp GapProvider) *PGBridgePlanner {
	p.gap = gp
	return p
}

func (p *PGBridgePlanner) Plan(req queryir.QueryRequest) (*Plan, error) {
	normalized := queryir.Normalize(req)
	if normalized.SchemaVersion != queryir.SchemaVersionV1 {
		return nil, fmt.Errorf("unsupported schema_version %q", normalized.SchemaVersion)
	}
	if normalized.Scope.Resource != queryir.ResourceAssets {
		return nil, fmt.Errorf("unsupported scope.resource %q", normalized.Scope.Resource)
	}
	useRecall := p.useElasticsearch && shouldUseESRecall(normalized)
	requestedFacets := len(normalized.Facets) > 0

	useESFacets := false
	usePGFacets := false
	var dropped []string
	if requestedFacets {
		engine := p.resolveFacetEngine()
		switch engine {
		case FacetEngineES:
			useESFacets = p.useElasticsearch
			usePGFacets = !useESFacets
		case FacetEnginePG:
			usePGFacets = true
		}
		if usePGFacets {
			dropped = collectUnsupportedPGFacetFields(normalized.Facets)
		}
	}

	steps := []queryir.DebugPlanStep{{Engine: "postgres", Mode: "filter"}}
	if useRecall {
		steps = []queryir.DebugPlanStep{
			{Engine: "elasticsearch", Mode: "recall"},
			{Engine: "postgres", Mode: "refine"},
		}
	}
	switch {
	case useESFacets:
		steps = append(steps, queryir.DebugPlanStep{Engine: "elasticsearch", Mode: "facet"})
	case usePGFacets:
		steps = append(steps, queryir.DebugPlanStep{Engine: "postgres", Mode: "facet"})
	}
	return &Plan{
		NormalizedQuery:      normalized,
		Steps:                steps,
		UseESRecall:          useRecall,
		UseESFacets:          useESFacets,
		UsePGFacets:          usePGFacets,
		PGFacetDroppedFields: dropped,
	}, nil
}

// resolveFacetEngine turns the (override, useES, gap) triple into the engine
// that actually runs. Priority: explicit override > auto (gap-based).
func (p *PGBridgePlanner) resolveFacetEngine() FacetEngine {
	switch p.facetEngine {
	case FacetEngineES:
		return FacetEngineES
	case FacetEnginePG:
		return FacetEnginePG
	}
	// Auto mode: ES when we know PG and ES agree, PG when they don't (or
	// when ES is disabled entirely so the request would otherwise return
	// empty facets).
	if !p.useElasticsearch {
		return FacetEnginePG
	}
	if p.gap != nil && p.gap.Gap() > 0 {
		return FacetEnginePG
	}
	return FacetEngineES
}

func collectUnsupportedPGFacetFields(facets []queryir.QueryFacet) []string {
	var out []string
	for _, f := range facets {
		if !PGSupportedFacetFields[strings.TrimSpace(f.Field)] {
			out = append(out, f.Field)
		}
	}
	return out
}

func shouldUseESRecall(req queryir.QueryRequest) bool {
	switch req.Mode {
	case "semantic", "similar":
		return true
	}
	return hasFulltextPredicate(req.Where)
}

func hasFulltextPredicate(expr *queryir.QueryExpr) bool {
	if expr == nil {
		return false
	}
	if expr.Pred != nil && expr.Pred.Field == "_fulltext" {
		return true
	}
	if expr.Not != nil && hasFulltextPredicate(expr.Not) {
		return true
	}
	for i := range expr.And {
		if hasFulltextPredicate(&expr.And[i]) {
			return true
		}
	}
	for i := range expr.Or {
		if hasFulltextPredicate(&expr.Or[i]) {
			return true
		}
	}
	return false
}
