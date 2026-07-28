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

// CYB-3713: keyword mode used to silently drop `q`. Normalize now
// synthesizes a `_fulltext ilike Q` predicate so the existing dispatch
// picks it up. These tests pin that plumbing.

func TestNormalize_KeywordInjectsFulltextPredicate(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Mode:          "keyword",
		Q:             "备餐操作",
		Scope:         QueryScope{Resource: "assets"},
		Page:          QueryPage{Page: 1, PageSize: 5},
	}
	got := Normalize(req)
	if got.Where == nil || got.Where.Pred == nil {
		t.Fatalf("expected injected fulltext predicate, got where=%+v", got.Where)
	}
	if got.Where.Pred.Field != "_fulltext" || got.Where.Pred.Op != "ilike" || got.Where.Pred.Value != "备餐操作" {
		t.Fatalf("unexpected injected predicate: %+v", got.Where.Pred)
	}
}

func TestNormalize_KeywordAndWrapsExistingWhere(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Mode:          "keyword",
		Q:             "pangzi",
		Scope:         QueryScope{Resource: "assets"},
		Where: &QueryExpr{
			Pred: &QueryPredicate{Field: "owner", Op: "eq", Value: "pangzi-consumer"},
		},
		Page: QueryPage{Page: 1, PageSize: 5},
	}
	got := Normalize(req)
	if got.Where == nil || len(got.Where.And) != 2 {
		t.Fatalf("expected AND-wrapped where with 2 leaves, got %+v", got.Where)
	}
	// First leaf preserves original owner predicate.
	if got.Where.And[0].Pred == nil || got.Where.And[0].Pred.Field != "owner" {
		t.Fatalf("first leaf lost original owner predicate: %+v", got.Where.And[0])
	}
	// Second leaf is the injected _fulltext.
	if got.Where.And[1].Pred == nil || got.Where.And[1].Pred.Field != "_fulltext" || got.Where.And[1].Pred.Value != "pangzi" {
		t.Fatalf("second leaf missing fulltext injection: %+v", got.Where.And[1])
	}
}

func TestNormalize_StructuredModeIgnoresQ(t *testing.T) {
	req := QueryRequest{
		SchemaVersion: "v1",
		Mode:          "structured",
		Q:             "备餐操作", // must be dropped for structured mode
		Scope:         QueryScope{Resource: "assets"},
		Page:          QueryPage{Page: 1, PageSize: 5},
	}
	got := Normalize(req)
	if got.Where != nil {
		t.Fatalf("structured mode with q must not inject a where, got %+v", got.Where)
	}
}

func TestNormalize_EmptyQKeepsNoFilter(t *testing.T) {
	// Even for keyword mode, an empty q must not inject — preserves
	// backward compat for callers who set mode=keyword without a query.
	req := QueryRequest{
		SchemaVersion: "v1",
		Mode:          "keyword",
		Q:             "   ", // whitespace-only, trimmed to empty
		Scope:         QueryScope{Resource: "assets"},
		Page:          QueryPage{Page: 1, PageSize: 5},
	}
	got := Normalize(req)
	if got.Where != nil {
		t.Fatalf("empty q must not inject a where, got %+v", got.Where)
	}
	if got.Q != "" {
		t.Fatalf("empty q should be trimmed to empty string, got %q", got.Q)
	}
}

func TestNormalize_SemanticAndSimilarAlsoInject(t *testing.T) {
	// The three fulltext modes share the same injection path — verifies
	// the fix is symmetric across modes so callers using semantic/similar
	// with q also get their q honored (was silently dropped before).
	for _, mode := range []string{"semantic", "similar"} {
		req := QueryRequest{
			SchemaVersion: "v1",
			Mode:          mode,
			Q:             "target-asset",
			Scope:         QueryScope{Resource: "assets"},
			Page:          QueryPage{Page: 1, PageSize: 5},
		}
		got := Normalize(req)
		if got.Where == nil || got.Where.Pred == nil || got.Where.Pred.Field != "_fulltext" {
			t.Fatalf("mode=%q with q should inject _fulltext, got %+v", mode, got.Where)
		}
	}
}

func TestIsFulltextMode(t *testing.T) {
	// Pin the mode set so a future contributor doesn't accidentally add a
	// mode name that would silently inject `_fulltext` without a spec.
	yes := []string{"keyword", "semantic", "similar"}
	no := []string{"structured", "", "STRUCTURED", "Keyword", "any-other"}
	for _, m := range yes {
		if !IsFulltextMode(m) {
			t.Errorf("IsFulltextMode(%q) = false; want true", m)
		}
	}
	for _, m := range no {
		if IsFulltextMode(m) {
			t.Errorf("IsFulltextMode(%q) = true; want false", m)
		}
	}
}
