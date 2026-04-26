package filter

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// ── Property 6: Query builder parameterization safety ────────────────────────
// For any filter list, BuildWhereClause should produce SQL with only positional
// placeholders ($1, $2, ...) and no raw user values in the SQL string.
// **Validates: Requirements 2.6**
func TestProperty6_ParameterizationSafety(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate 1-5 filters with distinctive values that won't collide with field names.
		// Use a "VAL_" prefix + digits to ensure uniqueness.
		numFilters := 1 + r.Intn(5)
		filters := make([]Filter, numFilters)
		for i := range filters {
			value := fmt.Sprintf("VAL_%d_%d", i, r.Intn(100000))
			filters[i] = Filter{
				Field:   randFieldName(r),
				Op:      "=",
				Value:   value,
				IsJsonb: false,
			}
		}

		wc, err := BuildWhereClause(filters, 1)
		if err != nil {
			t.Logf("build error: %v", err)
			return false
		}

		// SQL should not contain any of the raw user values
		for _, f := range filters {
			if strVal, ok := f.Value.(string); ok && strVal != "" {
				if strings.Contains(wc.SQL, strVal) {
					t.Logf("SQL contains raw value %q: %s", strVal, wc.SQL)
					return false
				}
			}
		}

		// SQL should contain positional placeholders
		paramPattern := regexp.MustCompile(`\$\d+`)
		matches := paramPattern.FindAllString(wc.SQL, -1)
		if len(matches) == 0 {
			t.Logf("SQL has no positional placeholders: %s", wc.SQL)
			return false
		}

		// All values should be in Args
		if len(wc.Args) != numFilters {
			t.Logf("args count mismatch: expected %d, got %d", numFilters, len(wc.Args))
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 6 failed: %v", err)
	}
}

// ── Property 7: Multiple filters AND combination ─────────────────────────────
// For any N filters (N >= 2), the SQL should contain exactly N conditions
// joined by AND.
// **Validates: Requirements 2.7**
func TestProperty7_MultipleFiltersANDCombination(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		numFilters := 2 + r.Intn(5) // 2-6 filters
		filters := make([]Filter, numFilters)
		for i := range filters {
			filters[i] = Filter{
				Field:   randFieldName(r),
				Op:      "=",
				Value:   randSimpleValue(r),
				IsJsonb: false,
			}
		}

		wc, err := BuildWhereClause(filters, 1)
		if err != nil {
			t.Logf("build error: %v", err)
			return false
		}

		// Count conditions by splitting on " AND "
		parts := strings.Split(wc.SQL, " AND ")
		if len(parts) != numFilters {
			t.Logf("expected %d conditions, got %d in SQL: %s", numFilters, len(parts), wc.SQL)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 7 failed: %v", err)
	}
}

// ── Property 8: JSONB path generates extraction expression ───────────────────
// For any IsJsonb=true filter with a dot field, the SQL should contain
// the JSONB extraction operator (#>>).
// **Validates: Requirements 2.2**
func TestProperty8_JsonbPathExtraction(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		col := randFieldName(r)
		key := randFieldName(r)
		field := col + "." + key

		f := Filter{
			Field:   field,
			Op:      "=",
			Value:   randSimpleValue(r),
			IsJsonb: true,
		}

		wc, err := BuildWhereClause([]Filter{f}, 1)
		if err != nil {
			t.Logf("build error: %v", err)
			return false
		}

		// SQL should contain #>> extraction operator
		if !strings.Contains(wc.SQL, "#>>") {
			t.Logf("SQL missing #>> for JSONB field %q: %s", field, wc.SQL)
			return false
		}

		// SQL should contain the JSONB path format '{key}'
		expectedPath := fmt.Sprintf("'{%s}'", key)
		if !strings.Contains(wc.SQL, expectedPath) {
			t.Logf("SQL missing path %s for field %q: %s", expectedPath, field, wc.SQL)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 8 failed: %v", err)
	}
}

// ── Property 9: JSONB numeric comparison type casting ────────────────────────
// For any IsJsonb=true filter with a numeric comparison operator (<, >, <=, >=),
// the SQL should contain explicit type casting (::NUMERIC or ::TIMESTAMPTZ).
// **Validates: Requirements 2.5**
func TestProperty9_JsonbNumericTypeCasting(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		col := randFieldName(r)
		key := randFieldName(r)
		field := col + "." + key

		numericOps := []string{"<", ">", "<=", ">="}
		op := numericOps[r.Intn(len(numericOps))]

		// Randomly choose between numeric and timestamp values
		var value interface{}
		var expectedCast string
		if r.Intn(2) == 0 {
			value = r.Intn(1000)
			expectedCast = "::NUMERIC"
		} else {
			value = time.Date(2024, 1, 1+r.Intn(28), 0, 0, 0, 0, time.UTC)
			expectedCast = "::TIMESTAMPTZ"
		}

		f := Filter{
			Field:   field,
			Op:      op,
			Value:   value,
			IsJsonb: true,
		}

		wc, err := BuildWhereClause([]Filter{f}, 1)
		if err != nil {
			t.Logf("build error: %v", err)
			return false
		}

		if !strings.Contains(wc.SQL, expectedCast) {
			t.Logf("SQL missing %s cast for JSONB numeric comparison: %s", expectedCast, wc.SQL)
			return false
		}

		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 9 failed: %v", err)
	}
}

func TestBuildWhereClause_AlgoKeyWithDotsUsesSingleJsonKey(t *testing.T) {
	wc, err := BuildWhereClause([]Filter{{
		Field:        "algo.hand_tracking@1.2.0:status",
		StorageField: "cf_algo.hand_tracking@1.2.0:status",
		Op:           "=",
		Value:        "failed",
		IsJsonb:      true,
	}}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause: %v", err)
	}

	expected := "cf_algo#>>'{hand_tracking@1.2.0:status}' = $1"
	if wc.SQL != expected {
		t.Fatalf("expected %q, got %q", expected, wc.SQL)
	}
}

func TestResolveSortBy_WhitelistAndAliases(t *testing.T) {
	got, err := ResolveSortBy("-tags.notes")
	if err != nil {
		t.Fatalf("ResolveSortBy alias: %v", err)
	}
	if got != "cf_tag#>>'{notes}' DESC" {
		t.Fatalf("unexpected orderBy: %q", got)
	}

	got, err = ResolveSortBy("owner")
	if err != nil {
		t.Fatalf("ResolveSortBy owner: %v", err)
	}
	if got != "cf_meta#>>'{owner}' ASC" {
		t.Fatalf("unexpected owner orderBy: %q", got)
	}

	if _, err := ResolveSortBy("-drop_table"); err == nil {
		t.Fatal("expected invalid sort field to be rejected")
	}
}

// ── Virtual field SQL generation tests ───────────────────────────────────────

func TestBuildWhereClause_AlgoStatusVirtual(t *testing.T) {
	wc, err := BuildWhereClause([]Filter{{
		Field:     "algo_status",
		Op:        "=",
		Value:     "failed",
		IsVirtual: true,
	}}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause algo_status: %v", err)
	}

	expected := "EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo) WHERE key LIKE '%:status' AND value = $1)"
	if wc.SQL != expected {
		t.Fatalf("expected %q, got %q", expected, wc.SQL)
	}
	if len(wc.Args) != 1 || wc.Args[0] != "failed" {
		t.Fatalf("expected args [failed], got %v", wc.Args)
	}
}

func TestBuildWhereClause_AlgoStatusWithOtherFilters(t *testing.T) {
	wc, err := BuildWhereClause([]Filter{
		{Field: "status", Op: "=", Value: "approved", IsJsonb: false},
		{Field: "algo_status", Op: "=", Value: "ok", IsVirtual: true},
	}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause combined: %v", err)
	}

	if !strings.Contains(wc.SQL, "status = $1") {
		t.Fatalf("expected status = $1 in SQL, got %q", wc.SQL)
	}
	if !strings.Contains(wc.SQL, "EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo) WHERE key LIKE '%:status' AND value = $2)") {
		t.Fatalf("expected algo_status EXISTS in SQL, got %q", wc.SQL)
	}
	if len(wc.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(wc.Args))
	}
}

func TestBuildWhereClause_HasDeliveryTrue(t *testing.T) {
	wc, err := BuildWhereClause([]Filter{{
		Field:     "has:delivery",
		Op:        "=",
		Value:     true,
		IsVirtual: true,
	}}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause has:delivery true: %v", err)
	}

	if wc.SQL != "delivery_count > 0" {
		t.Fatalf("expected 'delivery_count > 0', got %q", wc.SQL)
	}
	if len(wc.Args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(wc.Args))
	}
}

func TestBuildWhereClause_HasDeliveryFalse(t *testing.T) {
	wc, err := BuildWhereClause([]Filter{{
		Field:     "has:delivery",
		Op:        "=",
		Value:     false,
		IsVirtual: true,
	}}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause has:delivery false: %v", err)
	}

	if wc.SQL != "delivery_count = 0" {
		t.Fatalf("expected 'delivery_count = 0', got %q", wc.SQL)
	}
}

func TestBuildWhereClause_HasDeliveryStringTrue(t *testing.T) {
	wc, err := BuildWhereClause([]Filter{{
		Field:     "has:delivery",
		Op:        "=",
		Value:     "true",
		IsVirtual: true,
	}}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause has:delivery string true: %v", err)
	}

	if wc.SQL != "delivery_count > 0" {
		t.Fatalf("expected 'delivery_count > 0', got %q", wc.SQL)
	}
}
