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

func (e *Executor) Compile(plan *queryplan.Plan) (map[string]any, error) {
	return elasticsearch.BuildQueryIRSearchBody(plan.NormalizedQuery, true)
}
