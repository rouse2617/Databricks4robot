package elasticsearch

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
)

func TestCompile_IncludesFacetsAndTreeQuery(t *testing.T) {
	exec := New(nil)
	plan := &queryplan.Plan{
		NormalizedQuery: queryir.QueryRequest{
			SchemaVersion: "v1",
			Scope:         queryir.QueryScope{Resource: "assets"},
			Where: &queryir.QueryExpr{
				And: []queryir.QueryExpr{
					{Pred: &queryir.QueryPredicate{Field: "owner", Op: "eq", Value: "alice"}},
					{Pred: &queryir.QueryPredicate{Field: "_fulltext", Op: "ilike", Value: "forklift"}},
				},
			},
			Facets: []queryir.QueryFacet{
				{Field: "owner", Size: 10},
			},
		},
		UseESRecall: true,
		UseESFacets: true,
	}

	body, err := exec.Compile(plan, false)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if _, ok := body["aggs"]; !ok {
		t.Fatalf("expected aggs in body")
	}
	query, ok := body["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query map")
	}
	boolQuery, ok := query["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool query")
	}
	if _, ok := boolQuery["must"]; !ok {
		t.Fatalf("expected AND to compile to must")
	}
}

func TestCompile_FacetOnlyBodySkipsHits(t *testing.T) {
	exec := New(nil)
	plan := &queryplan.Plan{
		NormalizedQuery: queryir.QueryRequest{
			SchemaVersion: "v1",
			Scope:         queryir.QueryScope{Resource: "assets"},
			Where: &queryir.QueryExpr{
				Pred: &queryir.QueryPredicate{Field: "owner", Op: "eq", Value: "alice"},
			},
			Facets: []queryir.QueryFacet{
				{Field: "owner", Size: 10},
			},
		},
		UseESRecall: false,
		UseESFacets: true,
	}

	body, err := exec.Compile(plan, false)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if got, ok := body["size"].(int); !ok || got != 0 {
		t.Fatalf("expected size=0 for facet-only query, got=%#v", body["size"])
	}
	if got, ok := body["track_total_hits"].(bool); !ok || got {
		t.Fatalf("expected track_total_hits=false, got=%#v", body["track_total_hits"])
	}
	if _, ok := body["sort"]; ok {
		t.Fatalf("did not expect sort in facet-only query")
	}
	if _, ok := body["aggs"]; !ok {
		t.Fatalf("expected aggs in facet-only query")
	}
}
