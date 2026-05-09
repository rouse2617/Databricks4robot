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
	}

	body, err := exec.Compile(plan)
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
