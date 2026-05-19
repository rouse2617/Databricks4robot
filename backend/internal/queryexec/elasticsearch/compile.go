package elasticsearch

import (
	"github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
)

type Executor struct {
	client *elasticsearch.Client
}

func New(client *elasticsearch.Client) *Executor {
	return &Executor{client: client}
}

func (e *Executor) Compile(plan *queryplan.Plan, trackTotalHits bool) (map[string]any, error) {
	return elasticsearch.BuildQueryIRSearchBody(
		plan.NormalizedQuery,
		plan.UseESRecall,
		plan.UseESFacets,
		trackTotalHits,
	)
}

// CompileCountOnly builds a size-0 ES query that returns track_total_hits only.
func (e *Executor) CompileCountOnly(plan *queryplan.Plan) (map[string]any, error) {
	return elasticsearch.BuildQueryIRSearchBody(plan.NormalizedQuery, false, false, true)
}
