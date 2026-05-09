package queryplan

import (
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

type Plan struct {
	NormalizedQuery  queryir.QueryRequest
	Steps            []queryir.DebugPlanStep
	UseElasticsearch bool
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
	steps := []queryir.DebugPlanStep{{Engine: "postgres", Mode: "filter"}}
	if p.useElasticsearch {
		steps = []queryir.DebugPlanStep{
			{Engine: "elasticsearch", Mode: "recall"},
			{Engine: "postgres", Mode: "refine"},
		}
		if len(normalized.Facets) > 0 {
			steps = append(steps, queryir.DebugPlanStep{Engine: "elasticsearch", Mode: "facet"})
		}
	}
	return &Plan{
		NormalizedQuery:  normalized,
		Steps:            steps,
		UseElasticsearch: p.useElasticsearch,
	}, nil
}
