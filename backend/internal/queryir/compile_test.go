package queryir

import "testing"

func TestCompile_AndPredicates(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Scope:         QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			And: []QueryExpr{
				{Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "alice"}},
				{Pred: &QueryPredicate{Field: "delivery_count", Op: "gte", Value: 1}},
			},
		},
		Sort: []QuerySort{{Field: "created_at", Direction: "desc"}},
		Page: QueryPage{Page: 2, PageSize: 30},
	}

	q, err := Compile(req)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if q.FilterStrings != nil {
		t.Fatalf("expected FilterStrings to be nil (expr-native), got %#v", q.FilterStrings)
	}
	if q.SortBy != "-created_at" {
		t.Fatalf("unexpected sort: %s", q.SortBy)
	}
	if q.Page != 2 || q.PageSize != 30 {
		t.Fatalf("unexpected page: %+v", q)
	}
	if q.NormalizedQuery.SchemaVersion != "v1" {
		t.Fatalf("unexpected normalized schema version: %q", q.NormalizedQuery.SchemaVersion)
	}
	if got := len(q.FieldCapabilities); got != 3 {
		t.Fatalf("expected 3 field capabilities, got %d", got)
	}
}

func TestCompile_PredicateShorthand(t *testing.T) {
	req := QueryRequest{
		Scope: QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "alice"},
		},
	}

	q, err := Compile(req)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if q.FilterStrings != nil {
		t.Fatalf("expected FilterStrings to be nil (expr-native), got %#v", q.FilterStrings)
	}
	if q.Page != 1 || q.PageSize != 20 {
		t.Fatalf("expected default page/page_size, got %+v", q.NormalizedQuery.Page)
	}
}

func TestCompile_CollectsOrLeafFilters(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Scope:         QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			Or: []QueryExpr{
				{Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "alice"}},
				{Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "bob"}},
			},
		},
	}
	compiled, err := Compile(req)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if compiled.FilterStrings != nil {
		t.Fatalf("expected FilterStrings to be nil (expr-native), got %#v", compiled.FilterStrings)
	}
}

func TestCompile_CollectsNotLeafFilters(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Scope:         QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			Not: &QueryExpr{
				Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "alice"},
			},
		},
	}
	compiled, err := Compile(req)
	if err != nil {
		t.Fatalf("Compile() err = %v", err)
	}
	if compiled.FilterStrings != nil {
		t.Fatalf("expected FilterStrings to be nil (expr-native), got %#v", compiled.FilterStrings)
	}
}

func TestCompile_RejectsInvalidExprNode(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Scope:         QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "alice"},
			And:  []QueryExpr{{Pred: &QueryPredicate{Field: "reviewer", Op: "eq", Value: "bob"}}},
		},
	}
	_, err := Compile(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != `invalid where expression: exactly one of and/or/not/pred must be set` {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestCompile_RejectsMissingPredicateOperator(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Scope:         QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			Pred: &QueryPredicate{Field: "owner", Value: "alice"},
		},
	}
	_, err := Compile(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != `missing operator in predicate` {
		t.Fatalf("unexpected error: %q", got)
	}
}
