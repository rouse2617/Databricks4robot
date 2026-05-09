package postgres

import (
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
)

type Executor struct{}

func New() *Executor {
	return &Executor{}
}

func (e *Executor) Compile(plan *queryplan.Plan) (*queryir.CompiledQuery, error) {
	compiled, err := queryir.Compile(plan.NormalizedQuery)
	if err != nil {
		return nil, err
	}
	compiled.DebugPlan = queryir.DebugPlan{Steps: plan.Steps}
	return compiled, nil
}
