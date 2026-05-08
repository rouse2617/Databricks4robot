package elasticsearch

import (
	"data-platform/internal/elasticsearch"
	"data-platform/internal/queryplan"
)

type Executor struct {
	client *elasticsearch.Client
}

func New(client *elasticsearch.Client) *Executor {
	return &Executor{client: client}
}

func (e *Executor) Compile(plan *queryplan.Plan) (map[string]any, error) {
	return elasticsearch.BuildQueryIRSearchBody(plan.NormalizedQuery, true)
}
