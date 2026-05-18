package queryplan

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

func TestPGBridgePlanner_Plan(t *testing.T) {
	planner := NewPGBridgePlanner(false)
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope: queryir.QueryScope{Resource: "assets"},
		Where: &queryir.QueryExpr{
			Pred: &queryir.QueryPredicate{Field: "owner", Op: "eq", Value: "alice"},
		},
	})
	if err != nil {
		t.Fatalf("Plan() err = %v", err)
	}
	if plan.NormalizedQuery.SchemaVersion != "v1" {
		t.Fatalf("unexpected normalized schema version: %q", plan.NormalizedQuery.SchemaVersion)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Engine != "postgres" || plan.Steps[0].Mode != "filter" {
		t.Fatalf("unexpected steps: %+v", plan.Steps)
	}
}

func TestPGBridgePlanner_RejectsUnsupportedScope(t *testing.T) {
	planner := NewPGBridgePlanner(false)
	_, err := planner.Plan(queryir.QueryRequest{
		SchemaVersion: "v1",
		Scope:         queryir.QueryScope{Resource: "deliveries"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPGBridgePlanner_PlanWithElasticsearch(t *testing.T) {
	planner := NewPGBridgePlanner(true)
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope: queryir.QueryScope{Resource: "assets"},
		Facets: []queryir.QueryFacet{
			{Field: "owner", Size: 10},
		},
	})
	if err != nil {
		t.Fatalf("Plan() err = %v", err)
	}
	if plan.UseESRecall {
		t.Fatalf("expected structured query to skip es recall")
	}
	if !plan.UseESFacets {
		t.Fatalf("expected facets to use es")
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("unexpected steps: %+v", plan.Steps)
	}
}

func TestPGBridgePlanner_UsesRecallForFulltext(t *testing.T) {
	planner := NewPGBridgePlanner(true)
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope: queryir.QueryScope{Resource: "assets"},
		Where: &queryir.QueryExpr{
			Pred: &queryir.QueryPredicate{Field: "_fulltext", Op: "ilike", Value: "night run"},
		},
	})
	if err != nil {
		t.Fatalf("Plan() err = %v", err)
	}
	if !plan.UseESRecall {
		t.Fatalf("expected es recall for fulltext")
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("unexpected steps: %+v", plan.Steps)
	}
}

func TestPGBridgePlanner_UsesRecallForSemanticMode(t *testing.T) {
	planner := NewPGBridgePlanner(true)
	plan, err := planner.Plan(queryir.QueryRequest{
		Mode:  "semantic",
		Scope: queryir.QueryScope{Resource: "assets"},
	})
	if err != nil {
		t.Fatalf("Plan() err = %v", err)
	}
	if !plan.UseESRecall {
		t.Fatalf("expected es recall for semantic mode")
	}
}
