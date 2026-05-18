package queryplan

import (
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

type Plan struct {
	NormalizedQuery queryir.QueryRequest
	Steps           []queryir.DebugPlanStep
	UseESRecall     bool
	UseESFacets     bool
}

type PGBridgePlanner struct {
	useElasticsearch bool
}

func NewPGBridgePlanner(useElasticsearch bool) *PGBridgePlanner {
	return &PGBridgePlanner{useElasticsearch: useElasticsearch}
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
	useFacets := p.useElasticsearch && len(normalized.Facets) > 0

	steps := []queryir.DebugPlanStep{{Engine: "postgres", Mode: "filter"}}
	if useRecall {
		steps = []queryir.DebugPlanStep{
			{Engine: "elasticsearch", Mode: "recall"},
			{Engine: "postgres", Mode: "refine"},
		}
	}
	if useFacets {
		steps = append(steps, queryir.DebugPlanStep{Engine: "elasticsearch", Mode: "facet"})
	}
	return &Plan{
		NormalizedQuery: normalized,
		Steps:           steps,
		UseESRecall:     useRecall,
		UseESFacets:     useFacets,
	}, nil
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
