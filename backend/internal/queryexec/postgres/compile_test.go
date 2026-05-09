package postgres

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
)

func TestExecutor_Compile(t *testing.T) {
	exec := New()
	plan := &queryplan.Plan{
		NormalizedQuery: queryir.QueryRequest{
			SchemaVersion: "v1",
			Scope:         queryir.QueryScope{Resource: "assets"},
			Where: &queryir.QueryExpr{
				Pred: &queryir.QueryPredicate{Field: "owner", Op: "eq", Value: "alice"},
			},
		},
		Steps: []queryir.DebugPlanStep{{Engine: "postgres", Mode: "filter"}},
	}

	compiled, err := exec.Compile(plan)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if compiled.FilterStrings != nil {
		t.Fatalf("expected FilterStrings to be nil (expr-native), got %#v", compiled.FilterStrings)
	}
	if len(compiled.DebugPlan.Steps) != 1 || compiled.DebugPlan.Steps[0].Engine != "postgres" {
		t.Fatalf("unexpected debug plan: %+v", compiled.DebugPlan)
	}
}
