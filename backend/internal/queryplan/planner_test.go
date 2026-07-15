package queryplan

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

type fakeGap struct{ v int64 }

func (f fakeGap) Gap() int64 { return f.v }

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
	if plan.UsePGFacets {
		t.Fatalf("both engines set for facets, expected only ES")
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

// CYB-3384 — facet engine routing.

func TestPGBridgePlanner_FacetsAutoGapZeroPrefersES(t *testing.T) {
	planner := NewPGBridgePlanner(true).WithGapProvider(fakeGap{v: 0})
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope:  queryir.QueryScope{Resource: "assets"},
		Facets: []queryir.QueryFacet{{Field: "asset_type", Size: 20}},
	})
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	if !plan.UseESFacets || plan.UsePGFacets {
		t.Fatalf("gap==0 must choose ES, got UseES=%v UsePG=%v", plan.UseESFacets, plan.UsePGFacets)
	}
}

func TestPGBridgePlanner_FacetsAutoGapPositiveFallsBackToPG(t *testing.T) {
	planner := NewPGBridgePlanner(true).WithGapProvider(fakeGap{v: 5})
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope: queryir.QueryScope{Resource: "assets"},
		Facets: []queryir.QueryFacet{
			{Field: "asset_type", Size: 20},
			{Field: "mcap.vendor_id", Size: 20}, // unsupported → dropped
		},
	})
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	if plan.UseESFacets || !plan.UsePGFacets {
		t.Fatalf("gap>0 must choose PG, got UseES=%v UsePG=%v", plan.UseESFacets, plan.UsePGFacets)
	}
	if len(plan.PGFacetDroppedFields) != 1 || plan.PGFacetDroppedFields[0] != "mcap.vendor_id" {
		t.Fatalf("expected dropped=[mcap.vendor_id], got %v", plan.PGFacetDroppedFields)
	}
	last := plan.Steps[len(plan.Steps)-1]
	if last.Engine != "postgres" || last.Mode != "facet" {
		t.Fatalf("last step should be postgres/facet, got %+v", last)
	}
}

func TestPGBridgePlanner_FacetsOverrideESForcesES(t *testing.T) {
	planner := NewPGBridgePlanner(true).
		WithGapProvider(fakeGap{v: 999}).
		WithFacetEngine(FacetEngineES)
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope:  queryir.QueryScope{Resource: "assets"},
		Facets: []queryir.QueryFacet{{Field: "asset_type", Size: 20}},
	})
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	if !plan.UseESFacets || plan.UsePGFacets {
		t.Fatalf("override=es must force ES even when gap>0, got UseES=%v UsePG=%v", plan.UseESFacets, plan.UsePGFacets)
	}
}

func TestPGBridgePlanner_FacetsOverridePGForcesPG(t *testing.T) {
	planner := NewPGBridgePlanner(true).
		WithGapProvider(fakeGap{v: 0}).
		WithFacetEngine(FacetEnginePG)
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope:  queryir.QueryScope{Resource: "assets"},
		Facets: []queryir.QueryFacet{{Field: "asset_type", Size: 20}},
	})
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	if plan.UseESFacets || !plan.UsePGFacets {
		t.Fatalf("override=pg must force PG even when gap==0, got UseES=%v UsePG=%v", plan.UseESFacets, plan.UsePGFacets)
	}
}

func TestPGBridgePlanner_FacetsWhenESDisabledFallsBackToPG(t *testing.T) {
	planner := NewPGBridgePlanner(false)
	plan, err := planner.Plan(queryir.QueryRequest{
		Scope:  queryir.QueryScope{Resource: "assets"},
		Facets: []queryir.QueryFacet{{Field: "asset_type", Size: 20}},
	})
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	if plan.UseESFacets {
		t.Fatalf("ES disabled must not choose ES facet")
	}
	if !plan.UsePGFacets {
		t.Fatalf("expected PG facet fallback when ES disabled")
	}
}

func TestParseFacetEngine(t *testing.T) {
	cases := map[string]FacetEngine{
		"":              FacetEngineAuto,
		"auto":          FacetEngineAuto,
		"nonsense":      FacetEngineAuto,
		"ES":            FacetEngineES,
		"elasticsearch": FacetEngineES,
		"pg":            FacetEnginePG,
		"POSTGRES":      FacetEnginePG,
		"  pg  ":        FacetEnginePG,
	}
	for in, want := range cases {
		if got := ParseFacetEngine(in); got != want {
			t.Fatalf("ParseFacetEngine(%q) = %v, want %v", in, got, want)
		}
	}
}
