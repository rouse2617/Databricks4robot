package queryir

import "testing"

func TestApplyCurrentOnlyFilter(t *testing.T) {
	if !ApplyCurrentOnlyFilter(QueryRequest{Scope: QueryScope{Resource: ResourceAssets}}) {
		t.Fatal("expected default current-only for assets")
	}
	if ApplyCurrentOnlyFilter(QueryRequest{
		Scope: QueryScope{Resource: ResourceAssets, IncludeHistory: true},
	}) {
		t.Fatal("expected no filter when include_history")
	}
	if ApplyCurrentOnlyFilter(QueryRequest{Scope: QueryScope{Resource: "other"}}) {
		t.Fatal("expected no filter for non-assets resource")
	}
}
