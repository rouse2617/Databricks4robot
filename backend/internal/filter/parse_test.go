package filter

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// ── Generators ───────────────────────────────────────────────────────────────

// validOpNames returns all valid operator names.
func validOpNames() []string {
	return []string{"eq", "ne", "lt", "gt", "lte", "gte", "like", "ilike", "in", "nin", "contains"}
}

// randFieldName generates a random plain field name (no dots).
func randFieldName(r *rand.Rand) string {
	n := 2 + r.Intn(10)
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(r.Intn(26))
	}
	return string(b)
}

// randJsonbField generates a JSONB dot-path field name.

var allowedExactInputFields = []string{
	"asset_id",
	"mcap_file_id",
	"start_timestamp_ns",
	"status",
	"created_at",
	"updated_at",
	"version",
	"end_timestamp_ns",
	"duration_sec",
	"reviewer",
	"owner",
	"type",
	"env",
	"task",
	"delivery_count",
	"last_delivered_to",
	"last_delivered_at",
	"retention_tier",
	"lifecycle_state",
	"asset_type",
	"expire_at",
}

var allowedDynamicInputFields = []string{
	"tag.priority",
	"tags.notes",
	"algo.hand_tracking@1.2.0:status",
	"files.preview_mp4",
	"lifecycle.total_size_bytes",
}

func randAllowedFilterField(r *rand.Rand) string {
	all := append(append([]string{}, allowedExactInputFields...), allowedDynamicInputFields...)
	return all[r.Intn(len(all))]
}

// randSimpleValue generates a random simple string value (no colons, safe for parsing).
func randSimpleValue(r *rand.Rand) string {
	n := 1 + r.Intn(15)
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(r.Intn(26))
	}
	return string(b)
}

// ── Property 1: Filter parse round-trip consistency ──────────────────────────
// For any valid filter string, parsing it, serializing back, and re-parsing
// should produce an equivalent Filter object.
// **Validates: Requirements 11.1**
func TestProperty1_FilterParseRoundTrip(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Pick a random operator (exclude like/ilike for simpler round-trip,
		// and in/nin/contains which have special value handling)
		simpleOps := []string{"eq", "ne", "lt", "gt", "lte", "gte"}
		opName := simpleOps[r.Intn(len(simpleOps))]

		// Generate field and value
		field := randAllowedFilterField(r)
		value := randSimpleValue(r)

		filterStr := fmt.Sprintf("%s:%s:%s", field, opName, value)

		// Parse
		f1, err := ParseFilter(filterStr)
		if err != nil {
			t.Logf("parse error for %q: %v", filterStr, err)
			return false
		}

		// Serialize back
		serialized := f1.Serialize()

		// Re-parse
		f2, err := ParseFilter(serialized)
		if err != nil {
			t.Logf("re-parse error for %q: %v", serialized, err)
			return false
		}

		// Compare
		if f1.Field != f2.Field {
			t.Logf("field mismatch: %q vs %q", f1.Field, f2.Field)
			return false
		}
		if f1.Op != f2.Op {
			t.Logf("op mismatch: %q vs %q", f1.Op, f2.Op)
			return false
		}
		if fmt.Sprintf("%v", f1.Value) != fmt.Sprintf("%v", f2.Value) {
			t.Logf("value mismatch: %v vs %v", f1.Value, f2.Value)
			return false
		}
		if f1.IsJsonb != f2.IsJsonb {
			t.Logf("isJsonb mismatch: %v vs %v", f1.IsJsonb, f2.IsJsonb)
			return false
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 1 failed: %v", err)
	}
}

// ── Property 2: Parsed operator always valid ─────────────────────────────────
// For any valid filter string with a recognized operator, the parsed Op field
// should map to a valid SQL operator in OperatorMap.
// **Validates: Requirements 11.2, 1.1**
func TestProperty2_ParsedOperatorAlwaysValid(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		ops := validOpNames()
		opName := ops[r.Intn(len(ops))]
		field := randAllowedFilterField(r)
		value := randSimpleValue(r)

		filterStr := fmt.Sprintf("%s:%s:%s", field, opName, value)
		f, err := ParseFilter(filterStr)
		if err != nil {
			t.Logf("parse error for %q: %v", filterStr, err)
			return false
		}

		// Check that Op is a valid SQL operator
		found := false
		for _, sqlOp := range OperatorMap {
			if f.Op == sqlOp {
				found = true
				break
			}
		}
		if !found {
			t.Logf("invalid SQL operator %q for input %q", f.Op, filterStr)
			return false
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 2 failed: %v", err)
	}
}

// ── Property 3: Dot fields marked as JSONB ───────────────────────────────────
// For any allowed filter string, IsJsonb should match the resolved storage mapping.
// **Validates: Requirements 1.3, 11.3**
func TestProperty3_DotFieldsMarkedAsJsonb(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		ops := validOpNames()
		opName := ops[r.Intn(len(ops))]
		value := randSimpleValue(r)
		field := randAllowedFilterField(r)

		filterStr := fmt.Sprintf("%s:%s:%s", field, opName, value)
		f, err := ParseFilter(filterStr)
		if err != nil {
			t.Logf("parse error for %q: %v", filterStr, err)
			return false
		}

		spec, err := ResolveField(field)
		if err != nil {
			t.Logf("resolve error for %q: %v", field, err)
			return false
		}

		if f.IsJsonb != spec.IsJSONB {
			t.Logf("IsJsonb=%v but resolved mapping for %q says %v", f.IsJsonb, field, spec.IsJSONB)
			return false
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 3 failed: %v", err)
	}
}

// ── Property 4: Value type inference correctness ─────────────────────────────
// For any valid filter string, the parser should correctly infer value types:
// integers → int, floats → float64, RFC3339 → time.Time, JSON arrays → []interface{}.
// **Validates: Requirements 1.4, 1.5, 1.8, 1.9**
func TestProperty4_ValueTypeInference(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		field := randAllowedFilterField(r)

		// Generate a value of a known type and verify inference
		typeChoice := r.Intn(4)
		var valueStr string
		var expectedType string

		switch typeChoice {
		case 0: // integer
			n := r.Intn(10000)
			valueStr = fmt.Sprintf("%d", n)
			expectedType = "int"
		case 1: // float
			f := r.Float64() * 1000
			valueStr = fmt.Sprintf("%.2f", f)
			expectedType = "float64"
		case 2: // RFC3339 timestamp
			ts := time.Date(2020+r.Intn(6), time.Month(1+r.Intn(12)), 1+r.Intn(28),
				r.Intn(24), r.Intn(60), r.Intn(60), 0, time.UTC)
			valueStr = ts.Format(time.RFC3339)
			expectedType = "time.Time"
		case 3: // JSON array
			arr := make([]int, 1+r.Intn(5))
			for i := range arr {
				arr[i] = r.Intn(100)
			}
			parts := make([]string, len(arr))
			for i, v := range arr {
				parts[i] = fmt.Sprintf("%d", v)
			}
			valueStr = "[" + strings.Join(parts, ",") + "]"
			expectedType = "[]interface {}"
		}

		filterStr := fmt.Sprintf("%s:eq:%s", field, valueStr)
		f, err := ParseFilter(filterStr)
		if err != nil {
			t.Logf("parse error for %q: %v", filterStr, err)
			return false
		}

		actualType := fmt.Sprintf("%T", f.Value)
		if actualType != expectedType {
			t.Logf("type mismatch for value %q: expected %s, got %s", valueStr, expectedType, actualType)
			return false
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 4 failed: %v", err)
	}
}

// ── Property 5: Like/Ilike wildcard auto-wrapping ────────────────────────────
// For any like/ilike filter where the value doesn't contain wildcards,
// the parsed value should start and end with %.
// **Validates: Requirements 1.10**
func TestProperty5_LikeIlikeWildcardAutoWrap(t *testing.T) {
	cfg := &quick.Config{MaxCount: 100}
	err := quick.Check(func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		field := randAllowedFilterField(r)
		value := randSimpleValue(r) // no wildcards
		ops := []string{"like", "ilike"}
		opName := ops[r.Intn(len(ops))]

		filterStr := fmt.Sprintf("%s:%s:%s", field, opName, value)
		f, err := ParseFilter(filterStr)
		if err != nil {
			t.Logf("parse error for %q: %v", filterStr, err)
			return false
		}

		strVal, ok := f.Value.(string)
		if !ok {
			t.Logf("expected string value, got %T", f.Value)
			return false
		}

		if !strings.HasPrefix(strVal, "%") || !strings.HasSuffix(strVal, "%") {
			t.Logf("value %q should be wrapped with %% for op %s", strVal, opName)
			return false
		}
		return true
	}, cfg)
	if err != nil {
		t.Fatalf("Property 5 failed: %v", err)
	}
}

func TestParseFilter_ResolvesAliases(t *testing.T) {
	f, err := ParseFilter("tags.notes:eq:night-run")
	if err != nil {
		t.Fatalf("ParseFilter tags alias: %v", err)
	}
	if f.Field != "tag.notes" {
		t.Fatalf("expected canonical tag field, got %q", f.Field)
	}
	if f.StorageField != "asset_tags.notes" {
		t.Fatalf("expected asset_tags storage field, got %q", f.StorageField)
	}

	f, err = ParseFilter("algo.hand_tracking@1.2.0:status:eq:failed")
	if err != nil {
		t.Fatalf("ParseFilter algo alias: %v", err)
	}
	if f.Field != "algo.hand_tracking@1.2.0:status" {
		t.Fatalf("expected canonical algo field, got %q", f.Field)
	}
	if f.StorageField != "asset_algo_latest.hand_tracking@1.2.0:status" {
		t.Fatalf("expected asset_algo_latest storage field, got %q", f.StorageField)
	}
}

func TestParseFilter_RejectsUnknownField(t *testing.T) {
	if _, err := ParseFilter("drop_table:eq:oops"); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

// ── ResolveFieldAlias tests (cf_* legacy layer removed in P2-9) ──────────────

func TestResolveFieldAlias_TagPrefix(t *testing.T) {
	// After cf_* removal, tag. prefix is no longer aliased.
	got := ResolveFieldAlias("tag.notes")
	if got != "tag.notes" {
		t.Fatalf("expected tag.notes (no alias), got %q", got)
	}
}

func TestResolveFieldAlias_FilePrefix(t *testing.T) {
	got := ResolveFieldAlias("file.raw")
	if got != "file.raw" {
		t.Fatalf("expected file.raw (no alias), got %q", got)
	}
}

func TestResolveFieldAlias_AlgoPrefix(t *testing.T) {
	got := ResolveFieldAlias("algo.hand_tracking")
	if got != "algo.hand_tracking" {
		t.Fatalf("expected algo.hand_tracking (no alias), got %q", got)
	}
}

func TestResolveFieldAlias_NoAlias(t *testing.T) {
	got := ResolveFieldAlias("status")
	if got != "status" {
		t.Fatalf("expected status unchanged, got %q", got)
	}
}

func TestResolveFieldAlias_UnknownPrefix(t *testing.T) {
	got := ResolveFieldAlias("custom.field")
	if got != "custom.field" {
		t.Fatalf("expected custom.field unchanged, got %q", got)
	}
}

// ── ValidateFieldWhitelist tests ─────────────────────────────────────────────

func TestValidateFieldWhitelist_KnownFields(t *testing.T) {
	knownFields := []string{
		"asset_id", "mcap_file_id", "owner", "reviewer", "status", "env",
		"scene", "task", "batch", "duration_sec", "created_at", "updated_at",
		"delivery_count", "tag.priority", "tag.quality", "tag.notes",
		"tag.scene", "tag.task", "tag.batch", "algo_status", "has:delivery",
	}
	for _, field := range knownFields {
		if err := ValidateFieldWhitelist(field); err != nil {
			t.Errorf("expected field %q to pass whitelist, got error: %v", field, err)
		}
	}
}

func TestValidateFieldWhitelist_UnknownField(t *testing.T) {
	err := ValidateFieldWhitelist("unknown_field")
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected 'unknown field' in error, got: %v", err)
	}
}

func TestValidateFieldWhitelist_UnknownDotField(t *testing.T) {
	err := ValidateFieldWhitelist("tag.nonexistent")
	if err == nil {
		t.Fatal("expected error for tag.nonexistent (not in whitelist)")
	}
}

// ── ParseFilter with alias + whitelist integration ───────────────────────────

func TestParseFilter_AliasedTagField(t *testing.T) {
	f, err := ParseFilter("tag.notes:ilike:%test%")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.StorageField != "asset_tags.notes" {
		t.Fatalf("expected StorageField asset_tags.notes, got %q", f.StorageField)
	}
	if f.Field != "tag.notes" {
		t.Fatalf("expected canonical Field tag.notes, got %q", f.Field)
	}
	if f.IsJsonb != true {
		t.Fatal("expected IsJsonb=true for tag field")
	}
}

func TestParseFilter_LikeIlikeTreatsNumericLiteralAsString(t *testing.T) {
	f, err := ParseFilter("mcap.scene_id:ilike:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Op != "ILIKE" {
		t.Fatalf("expected Op ILIKE, got %q", f.Op)
	}
	if f.Value != "%1%" {
		t.Fatalf("expected Value %%1%%, got %v (%T)", f.Value, f.Value)
	}

	w, err := BuildWhereClause([]Filter{*f}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause: %v", err)
	}
	if len(w.Args) != 1 || w.Args[0] != "%1%" {
		t.Fatalf("expected args[0]=%%1%%, got %#v", w.Args)
	}
}

func TestParseFilter_AliasedTagPriority(t *testing.T) {
	f, err := ParseFilter("tag.priority:eq:high")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.StorageField != "asset_tags.priority" {
		t.Fatalf("expected StorageField asset_tags.priority, got %q", f.StorageField)
	}
	if f.Field != "tag.priority" {
		t.Fatalf("expected canonical Field tag.priority, got %q", f.Field)
	}
}

func TestParseFilter_WhitelistedPlainField(t *testing.T) {
	f, err := ParseFilter("status:eq:approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Field != "status" {
		t.Fatalf("expected Field status, got %q", f.Field)
	}
	if f.StorageField != "lifecycle_state" {
		t.Fatalf("expected StorageField lifecycle_state, got %q", f.StorageField)
	}
}

func TestParseFilter_UnknownFieldReturnsError(t *testing.T) {
	_, err := ParseFilter("unknown_field:eq:value")
	if err == nil {
		t.Fatal("expected error for unknown_field")
	}
	if !strings.Contains(err.Error(), "unknown field") && !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected 'unknown field' or 'not allowed' in error, got: %v", err)
	}
}

// CYB-3715: flatten columns should be filterable as top-level asset fields
// so /queries/run stops returning 422 UNSUPPORTED_FIELD on them.
func TestParseFilter_CYB3715FlattenFields(t *testing.T) {
	cases := []struct {
		field   string
		storage string
	}{
		{"camera_model", "camera_model"},
		{"device_id", "device_id"},
		{"collector_id", "collector_id"},
		{"scene_id", "scene_id"},
		{"data_source", "data_source"},
		{"collection_method", "collection_method"},
		{"source_platform", "source_platform"},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			f, err := ParseFilter(tc.field + ":eq:x")
			if err != nil {
				t.Fatalf("ParseFilter %q: %v", tc.field, err)
			}
			if f.Field != tc.field {
				t.Fatalf("Field: expected %q, got %q", tc.field, f.Field)
			}
			if f.StorageField != tc.storage {
				t.Fatalf("StorageField: expected %q, got %q", tc.storage, f.StorageField)
			}
		})
	}
}

func TestParseFilter_VirtualFieldAlgoStatus(t *testing.T) {
	// algo_status is in the whitelist as virtual; it should pass validation
	// and ParseFilter should return a Filter with IsVirtual=true.
	f, err := ParseFilter("algo_status:eq:failed")
	if err != nil {
		t.Fatalf("ParseFilter algo_status: %v", err)
	}
	if !f.IsVirtual {
		t.Fatal("expected IsVirtual=true for algo_status")
	}
	if f.Field != "algo_status" {
		t.Fatalf("expected Field algo_status, got %q", f.Field)
	}
	if f.Op != "=" {
		t.Fatalf("expected Op =, got %q", f.Op)
	}
	if f.Value != "failed" {
		t.Fatalf("expected Value failed, got %v", f.Value)
	}
}

func TestParseFilter_VirtualFieldHasDelivery(t *testing.T) {
	f, err := ParseFilter("has:delivery:eq:true")
	if err != nil {
		t.Fatalf("ParseFilter has:delivery: %v", err)
	}
	if !f.IsVirtual {
		t.Fatal("expected IsVirtual=true for has:delivery")
	}
	if f.Field != "has:delivery" {
		t.Fatalf("expected Field has:delivery, got %q", f.Field)
	}
	if f.Value != true {
		t.Fatalf("expected Value true, got %v (%T)", f.Value, f.Value)
	}
}
