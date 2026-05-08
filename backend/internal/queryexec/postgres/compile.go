package postgres

import (
	"data-platform/internal/queryir"
	"data-platform/internal/queryplan"
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
