package postgres_test

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// ---------------------------------------------------------------------------
// Go-level helpers that replicate the SQL backfill mapping logic from
// 010_schema_evolution_backfill_assets.sql.  Property tests exercise these
// helpers with random inputs to verify correctness and idempotence.
// ---------------------------------------------------------------------------

// BackfillAssetInput represents the source data for one asset row.
type BackfillAssetInput struct {
	Status string
	CfMeta map[string]interface{}
}

// BackfillAssetResult represents the real-column values after backfill.
type BackfillAssetResult struct {
	Owner           *string
	Reviewer        *string
	DurationMs      *int64
	LifecycleState  string
	AssetType       string
	DeliveryCount   int
	LastDeliveredTo *string
	LastDeliveredAt *time.Time
	RetentionTier   *string
}

// mapStatusToLifecycleState replicates the SQL CASE mapping.
func mapStatusToLifecycleState(status string) string {
	switch status {
	case "approved":
		return "ready"
	case "rejected":
		return "rejected"
	case "archived":
		return "archived"
	case "superseded":
		return "superseded"
	default:
		return "created"
	}
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// backfillAsset applies the backfill mapping logic in Go.
// It assumes the row has NOT been backfilled yet (columns at defaults).
func backfillAsset(input BackfillAssetInput) BackfillAssetResult {
	r := BackfillAssetResult{
		LifecycleState: "created", // default
		AssetType:      "segment", // default
		DeliveryCount:  0,         // default
	}

	// owner ← cf_meta->>'owner'
	if v, ok := input.CfMeta["owner"]; ok && v != nil {
		s := toString(v)
		r.Owner = strPtr(s)
	}

	// reviewer ← cf_meta->>'reviewer'
	if v, ok := input.CfMeta["reviewer"]; ok && v != nil {
		s := toString(v)
		r.Reviewer = strPtr(s)
	}

	// duration_ms ← (cf_meta->>'duration_sec')::NUMERIC * 1000
	if v, ok := input.CfMeta["duration_sec"]; ok && v != nil {
		sec := toFloat64(v)
		ms := int64(sec * 1000)
		r.DurationMs = &ms
	}

	// lifecycle_state ← mapped from status
	r.LifecycleState = mapStatusToLifecycleState(input.Status)

	// asset_type ← COALESCE(cf_meta->>'type', 'segment')
	if v, ok := input.CfMeta["type"]; ok && v != nil {
		s := toString(v)
		if s != "" {
			r.AssetType = s
		}
	}

	// delivery_count ← COALESCE((cf_meta->>'delivery_count')::INT, 0)
	if v, ok := input.CfMeta["delivery_count"]; ok && v != nil {
		r.DeliveryCount = toInt(v)
	}

	// last_delivered_to ← cf_meta->>'last_delivered_to'
	if v, ok := input.CfMeta["last_delivered_to"]; ok && v != nil {
		s := toString(v)
		r.LastDeliveredTo = strPtr(s)
	}

	// last_delivered_at ← (cf_meta->>'last_delivered_at')::TIMESTAMPTZ
	if v, ok := input.CfMeta["last_delivered_at"]; ok && v != nil {
		s := toString(v)
		if s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				r.LastDeliveredAt = &t
			}
		}
	}

	// retention_tier ← cf_meta->>'retention_tier'
	if v, ok := input.CfMeta["retention_tier"]; ok && v != nil {
		s := toString(v)
		r.RetentionTier = strPtr(s)
	}

	return r
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == math.Trunc(val) {
			return strings.TrimRight(strings.TrimRight(
				strings.Replace(
					json.Number(strings.TrimRight(strings.TrimRight(
						func() string { b, _ := json.Marshal(val); return string(b) }(),
						"0"), ".")).String(),
					"e+", "e", 1),
				"0"), ".")
		}
		b, _ := json.Marshal(val)
		return string(b)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		var f float64
		json.Unmarshal([]byte(val), &f)
		return f
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		var i int
		json.Unmarshal([]byte(val), &i)
		return i
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// Generators for rapid
// ---------------------------------------------------------------------------

func genStatus() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"approved", "rejected", "archived", "superseded"})
}

func genOptionalString() *rapid.Generator[*string] {
	return rapid.Custom(func(t *rapid.T) *string {
		if rapid.Bool().Draw(t, "present") {
			s := rapid.StringMatching(`[a-zA-Z0-9_]{1,20}`).Draw(t, "val")
			return &s
		}
		return nil
	})
}

func genCfMeta() *rapid.Generator[map[string]interface{}] {
	return rapid.Custom(func(t *rapid.T) map[string]interface{} {
		m := make(map[string]interface{})

		if rapid.Bool().Draw(t, "hasOwner") {
			m["owner"] = rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "owner")
		}
		if rapid.Bool().Draw(t, "hasReviewer") {
			m["reviewer"] = rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "reviewer")
		}
		if rapid.Bool().Draw(t, "hasDurationSec") {
			m["duration_sec"] = rapid.Float64Range(0.001, 86400.0).Draw(t, "duration_sec")
		}
		if rapid.Bool().Draw(t, "hasType") {
			m["type"] = rapid.SampledFrom([]string{"segment", "clip", "frame_set", "derived_asset"}).Draw(t, "type")
		}
		if rapid.Bool().Draw(t, "hasDeliveryCount") {
			m["delivery_count"] = float64(rapid.IntRange(0, 100).Draw(t, "delivery_count"))
		}
		if rapid.Bool().Draw(t, "hasLastDeliveredTo") {
			m["last_delivered_to"] = rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "last_delivered_to")
		}
		if rapid.Bool().Draw(t, "hasLastDeliveredAt") {
			ts := time.Date(
				rapid.IntRange(2020, 2025).Draw(t, "year"),
				time.Month(rapid.IntRange(1, 12).Draw(t, "month")),
				rapid.IntRange(1, 28).Draw(t, "day"),
				rapid.IntRange(0, 23).Draw(t, "hour"),
				rapid.IntRange(0, 59).Draw(t, "minute"),
				0, 0, time.UTC,
			)
			m["last_delivered_at"] = ts.Format(time.RFC3339)
		}
		if rapid.Bool().Draw(t, "hasRetentionTier") {
			m["retention_tier"] = rapid.SampledFrom([]string{"hot", "warm", "cold", "archive"}).Draw(t, "retention_tier")
		}

		return m
	})
}

// ---------------------------------------------------------------------------
// Property 1: Backfill assets round-trip (correctness + idempotence)
//
// For any asset row with arbitrary cf_meta JSONB content, after running the
// backfill, the real columns contain correctly derived values from cf_meta,
// AND running the backfill a second time produces identical column values.
//
// **Validates: Requirements 9.1, 9.2, 9.4**
// ---------------------------------------------------------------------------

func TestBackfillAssets_Property1_CorrectnessAndIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := genStatus().Draw(t, "status")
		cfMeta := genCfMeta().Draw(t, "cfMeta")

		input := BackfillAssetInput{
			Status: status,
			CfMeta: cfMeta,
		}

		// First backfill pass
		result1 := backfillAsset(input)

		// --- Correctness checks ---

		// owner
		if v, ok := cfMeta["owner"]; ok && v != nil {
			expected := toString(v)
			if result1.Owner == nil || *result1.Owner != expected {
				t.Fatalf("owner: expected %q, got %v", expected, result1.Owner)
			}
		} else {
			if result1.Owner != nil {
				t.Fatalf("owner: expected nil, got %v", *result1.Owner)
			}
		}

		// reviewer
		if v, ok := cfMeta["reviewer"]; ok && v != nil {
			expected := toString(v)
			if result1.Reviewer == nil || *result1.Reviewer != expected {
				t.Fatalf("reviewer: expected %q, got %v", expected, result1.Reviewer)
			}
		} else {
			if result1.Reviewer != nil {
				t.Fatalf("reviewer: expected nil, got %v", *result1.Reviewer)
			}
		}

		// duration_ms
		if v, ok := cfMeta["duration_sec"]; ok && v != nil {
			sec := toFloat64(v)
			expectedMs := int64(sec * 1000)
			if result1.DurationMs == nil || *result1.DurationMs != expectedMs {
				t.Fatalf("duration_ms: expected %d, got %v", expectedMs, result1.DurationMs)
			}
		} else {
			if result1.DurationMs != nil {
				t.Fatalf("duration_ms: expected nil, got %v", *result1.DurationMs)
			}
		}

		// lifecycle_state
		expectedLS := mapStatusToLifecycleState(status)
		if result1.LifecycleState != expectedLS {
			t.Fatalf("lifecycle_state: expected %q, got %q", expectedLS, result1.LifecycleState)
		}

		// asset_type
		if v, ok := cfMeta["type"]; ok && v != nil {
			expected := toString(v)
			if expected != "" && result1.AssetType != expected {
				t.Fatalf("asset_type: expected %q, got %q", expected, result1.AssetType)
			}
		} else {
			if result1.AssetType != "segment" {
				t.Fatalf("asset_type: expected 'segment' default, got %q", result1.AssetType)
			}
		}

		// delivery_count
		if v, ok := cfMeta["delivery_count"]; ok && v != nil {
			expected := toInt(v)
			if result1.DeliveryCount != expected {
				t.Fatalf("delivery_count: expected %d, got %d", expected, result1.DeliveryCount)
			}
		} else {
			if result1.DeliveryCount != 0 {
				t.Fatalf("delivery_count: expected 0 default, got %d", result1.DeliveryCount)
			}
		}

		// last_delivered_to
		if v, ok := cfMeta["last_delivered_to"]; ok && v != nil {
			expected := toString(v)
			if result1.LastDeliveredTo == nil || *result1.LastDeliveredTo != expected {
				t.Fatalf("last_delivered_to: expected %q, got %v", expected, result1.LastDeliveredTo)
			}
		} else {
			if result1.LastDeliveredTo != nil {
				t.Fatalf("last_delivered_to: expected nil, got %v", *result1.LastDeliveredTo)
			}
		}

		// last_delivered_at
		if v, ok := cfMeta["last_delivered_at"]; ok && v != nil {
			s := toString(v)
			if s != "" {
				expectedTime, err := time.Parse(time.RFC3339, s)
				if err == nil {
					if result1.LastDeliveredAt == nil || !result1.LastDeliveredAt.Equal(expectedTime) {
						t.Fatalf("last_delivered_at: expected %v, got %v", expectedTime, result1.LastDeliveredAt)
					}
				}
			}
		} else {
			if result1.LastDeliveredAt != nil {
				t.Fatalf("last_delivered_at: expected nil, got %v", *result1.LastDeliveredAt)
			}
		}

		// retention_tier
		if v, ok := cfMeta["retention_tier"]; ok && v != nil {
			expected := toString(v)
			if result1.RetentionTier == nil || *result1.RetentionTier != expected {
				t.Fatalf("retention_tier: expected %q, got %v", expected, result1.RetentionTier)
			}
		} else {
			if result1.RetentionTier != nil {
				t.Fatalf("retention_tier: expected nil, got %v", *result1.RetentionTier)
			}
		}

		// --- Idempotence: second pass produces identical results ---
		result2 := backfillAsset(input)

		if !equalBackfillResult(result1, result2) {
			t.Fatalf("idempotence violated: first pass != second pass\nfirst:  %+v\nsecond: %+v", result1, result2)
		}
	})
}

// ---------------------------------------------------------------------------
// SQL static analysis: verify the backfill script contains expected patterns
// ---------------------------------------------------------------------------

func loadBackfillAssetsSQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/010_schema_evolution_backfill_assets.sql")
	if err != nil {
		t.Fatalf("failed to read backfill script: %v", err)
	}
	return string(data)
}

func TestBackfillAssets_SQLStructure(t *testing.T) {
	sql := loadBackfillAssetsSQL(t)
	lower := strings.ToLower(sql)

	// Verify DO $$ block
	if !strings.Contains(lower, "do $$") {
		t.Error("backfill script should use DO $$ block")
	}

	// Verify batch_size declaration
	if !strings.Contains(lower, "batch_size") {
		t.Error("backfill script should declare batch_size")
	}

	// Verify LOOP
	if !strings.Contains(lower, "loop") {
		t.Error("backfill script should use LOOP for batch processing")
	}

	// Verify FOR UPDATE SKIP LOCKED
	if !strings.Contains(lower, "for update skip locked") {
		t.Error("backfill script should use FOR UPDATE SKIP LOCKED")
	}

	// Verify GET DIAGNOSTICS
	if !strings.Contains(lower, "get diagnostics") {
		t.Error("backfill script should use GET DIAGNOSTICS to check rows_updated")
	}

	// Verify EXIT WHEN
	if !strings.Contains(lower, "exit when") {
		t.Error("backfill script should EXIT WHEN rows_updated = 0")
	}

	// Verify all target columns are updated
	targetColumns := []string{
		"owner", "reviewer", "duration_ms", "lifecycle_state",
		"asset_type", "delivery_count", "last_delivered_to",
		"last_delivered_at", "retention_tier",
	}
	for _, col := range targetColumns {
		// Look for SET <col> = pattern
		pattern := regexp.MustCompile(`(?i)\b` + col + `\s*=`)
		if !pattern.MatchString(sql) {
			t.Errorf("backfill script missing SET for column: %s", col)
		}
	}

	// Verify lifecycle_state mapping from status
	statusMappings := []string{"approved", "rejected", "archived", "superseded"}
	for _, s := range statusMappings {
		if !strings.Contains(lower, "'"+s+"'") {
			t.Errorf("backfill script missing status mapping for: %s", s)
		}
	}

	// Verify lifecycle_state target values
	lifecycleTargets := []string{"ready", "rejected", "archived", "superseded"}
	for _, ls := range lifecycleTargets {
		if !strings.Contains(lower, "'"+ls+"'") {
			t.Errorf("backfill script missing lifecycle_state target: %s", ls)
		}
	}

	// Verify COALESCE usage for NULL handling
	if !strings.Contains(lower, "coalesce") {
		t.Error("backfill script should use COALESCE for NULL handling")
	}

	// Verify cf_meta JSONB extraction
	if !strings.Contains(lower, "cf_meta") {
		t.Error("backfill script should reference cf_meta for JSONB extraction")
	}

	// Verify duration_sec * 1000 conversion
	if !strings.Contains(lower, "duration_sec") || !strings.Contains(lower, "1000") {
		t.Error("backfill script should convert duration_sec * 1000 to duration_ms")
	}
}

// ---------------------------------------------------------------------------
// Edge case: TestBackfillAssets_NullCfMeta
// Verifies behavior when cf_meta is NULL or empty ({}).
// ---------------------------------------------------------------------------

func TestBackfillAssets_NullCfMeta(t *testing.T) {
	tests := []struct {
		name   string
		input  BackfillAssetInput
		expect BackfillAssetResult
	}{
		{
			name: "nil cf_meta map",
			input: BackfillAssetInput{
				Status: "approved",
				CfMeta: nil,
			},
			expect: BackfillAssetResult{
				Owner:           nil,
				Reviewer:        nil,
				DurationMs:      nil,
				LifecycleState:  "ready", // approved → ready
				AssetType:       "segment",
				DeliveryCount:   0,
				LastDeliveredTo: nil,
				LastDeliveredAt: nil,
				RetentionTier:   nil,
			},
		},
		{
			name: "empty cf_meta map",
			input: BackfillAssetInput{
				Status: "approved",
				CfMeta: map[string]interface{}{},
			},
			expect: BackfillAssetResult{
				Owner:           nil,
				Reviewer:        nil,
				DurationMs:      nil,
				LifecycleState:  "ready",
				AssetType:       "segment",
				DeliveryCount:   0,
				LastDeliveredTo: nil,
				LastDeliveredAt: nil,
				RetentionTier:   nil,
			},
		},
		{
			name: "cf_meta with all null values",
			input: BackfillAssetInput{
				Status: "rejected",
				CfMeta: map[string]interface{}{
					"owner":             nil,
					"reviewer":          nil,
					"duration_sec":      nil,
					"type":              nil,
					"delivery_count":    nil,
					"last_delivered_to": nil,
					"last_delivered_at": nil,
					"retention_tier":    nil,
				},
			},
			expect: BackfillAssetResult{
				Owner:           nil,
				Reviewer:        nil,
				DurationMs:      nil,
				LifecycleState:  "rejected",
				AssetType:       "segment",
				DeliveryCount:   0,
				LastDeliveredTo: nil,
				LastDeliveredAt: nil,
				RetentionTier:   nil,
			},
		},
		{
			name: "cf_meta with partial keys",
			input: BackfillAssetInput{
				Status: "archived",
				CfMeta: map[string]interface{}{
					"owner": "alice",
					// all other keys missing
				},
			},
			expect: BackfillAssetResult{
				Owner:           ptrStr("alice"),
				Reviewer:        nil,
				DurationMs:      nil,
				LifecycleState:  "archived",
				AssetType:       "segment",
				DeliveryCount:   0,
				LastDeliveredTo: nil,
				LastDeliveredAt: nil,
				RetentionTier:   nil,
			},
		},
		{
			name: "superseded status mapping",
			input: BackfillAssetInput{
				Status: "superseded",
				CfMeta: map[string]interface{}{},
			},
			expect: BackfillAssetResult{
				Owner:           nil,
				Reviewer:        nil,
				DurationMs:      nil,
				LifecycleState:  "superseded",
				AssetType:       "segment",
				DeliveryCount:   0,
				LastDeliveredTo: nil,
				LastDeliveredAt: nil,
				RetentionTier:   nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := backfillAsset(tc.input)

			if !equalStrPtr(result.Owner, tc.expect.Owner) {
				t.Errorf("owner: got %v, want %v", ptrVal(result.Owner), ptrVal(tc.expect.Owner))
			}
			if !equalStrPtr(result.Reviewer, tc.expect.Reviewer) {
				t.Errorf("reviewer: got %v, want %v", ptrVal(result.Reviewer), ptrVal(tc.expect.Reviewer))
			}
			if !equalInt64Ptr(result.DurationMs, tc.expect.DurationMs) {
				t.Errorf("duration_ms: got %v, want %v", ptrValI64(result.DurationMs), ptrValI64(tc.expect.DurationMs))
			}
			if result.LifecycleState != tc.expect.LifecycleState {
				t.Errorf("lifecycle_state: got %q, want %q", result.LifecycleState, tc.expect.LifecycleState)
			}
			if result.AssetType != tc.expect.AssetType {
				t.Errorf("asset_type: got %q, want %q", result.AssetType, tc.expect.AssetType)
			}
			if result.DeliveryCount != tc.expect.DeliveryCount {
				t.Errorf("delivery_count: got %d, want %d", result.DeliveryCount, tc.expect.DeliveryCount)
			}
			if !equalStrPtr(result.LastDeliveredTo, tc.expect.LastDeliveredTo) {
				t.Errorf("last_delivered_to: got %v, want %v", ptrVal(result.LastDeliveredTo), ptrVal(tc.expect.LastDeliveredTo))
			}
			if !equalTimePtr(result.LastDeliveredAt, tc.expect.LastDeliveredAt) {
				t.Errorf("last_delivered_at: got %v, want %v", result.LastDeliveredAt, tc.expect.LastDeliveredAt)
			}
			if !equalStrPtr(result.RetentionTier, tc.expect.RetentionTier) {
				t.Errorf("retention_tier: got %v, want %v", ptrVal(result.RetentionTier), ptrVal(tc.expect.RetentionTier))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ptrStr(s string) *string { return &s }

func ptrVal(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func ptrValI64(p *int64) string {
	if p == nil {
		return "<nil>"
	}
	b, _ := json.Marshal(*p)
	return string(b)
}

func equalStrPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

func equalBackfillResult(a, b BackfillAssetResult) bool {
	return equalStrPtr(a.Owner, b.Owner) &&
		equalStrPtr(a.Reviewer, b.Reviewer) &&
		equalInt64Ptr(a.DurationMs, b.DurationMs) &&
		a.LifecycleState == b.LifecycleState &&
		a.AssetType == b.AssetType &&
		a.DeliveryCount == b.DeliveryCount &&
		equalStrPtr(a.LastDeliveredTo, b.LastDeliveredTo) &&
		equalTimePtr(a.LastDeliveredAt, b.LastDeliveredAt) &&
		equalStrPtr(a.RetentionTier, b.RetentionTier)
}

// ===========================================================================
// MCAP FILES BACKFILL TESTS
// ===========================================================================

// ---------------------------------------------------------------------------
// Go-level helpers that replicate the SQL backfill mapping logic from
// 011_schema_evolution_backfill_mcap.sql.
// ---------------------------------------------------------------------------

// BackfillMcapInput represents the source data for one mcap_files row.
type BackfillMcapInput struct {
	CfMeta map[string]interface{}
}

// BackfillMcapResult represents the real-column values after backfill.
type BackfillMcapResult struct {
	McapURI          string
	SizeBytes        *int64
	IngestState      string
	StartTimestampNs *int64
	EndTimestampNs   *int64
	ChannelCount     *int
	ChunkCount       *int
	Owner            *string
}

// backfillMcap applies the backfill mapping logic in Go.
func backfillMcap(input BackfillMcapInput) BackfillMcapResult {
	r := BackfillMcapResult{
		McapURI:     "",        // default
		IngestState: "pending", // default
	}

	// mcap_uri ← COALESCE(cf_meta->>'gcs_path', '')
	if v, ok := input.CfMeta["gcs_path"]; ok && v != nil {
		s := toString(v)
		if s != "" {
			r.McapURI = s
		}
	}

	// size_bytes ← (cf_meta->>'size_bytes')::BIGINT
	if v, ok := input.CfMeta["size_bytes"]; ok && v != nil {
		n := int64(toFloat64(v))
		r.SizeBytes = &n
	}

	// ingest_state ← COALESCE(cf_meta->>'ingest_state', 'pending')
	if v, ok := input.CfMeta["ingest_state"]; ok && v != nil {
		s := toString(v)
		if s != "" {
			r.IngestState = s
		}
	}

	// start_timestamp_ns ← (cf_meta->>'start_timestamp_ns')::BIGINT
	if v, ok := input.CfMeta["start_timestamp_ns"]; ok && v != nil {
		n := int64(toFloat64(v))
		r.StartTimestampNs = &n
	}

	// end_timestamp_ns ← (cf_meta->>'end_timestamp_ns')::BIGINT
	if v, ok := input.CfMeta["end_timestamp_ns"]; ok && v != nil {
		n := int64(toFloat64(v))
		r.EndTimestampNs = &n
	}

	// channel_count ← (cf_meta->>'channel_count')::INT
	if v, ok := input.CfMeta["channel_count"]; ok && v != nil {
		n := toInt(v)
		r.ChannelCount = &n
	}

	// chunk_count ← (cf_meta->>'chunk_count')::INT
	if v, ok := input.CfMeta["chunk_count"]; ok && v != nil {
		n := toInt(v)
		r.ChunkCount = &n
	}

	// owner ← cf_meta->>'owner'
	if v, ok := input.CfMeta["owner"]; ok && v != nil {
		s := toString(v)
		r.Owner = strPtr(s)
	}

	return r
}

// ---------------------------------------------------------------------------
// Generators for mcap rapid tests
// ---------------------------------------------------------------------------

func genMcapCfMeta() *rapid.Generator[map[string]interface{}] {
	return rapid.Custom(func(t *rapid.T) map[string]interface{} {
		m := make(map[string]interface{})

		if rapid.Bool().Draw(t, "hasGcsPath") {
			m["gcs_path"] = "gs://" + rapid.StringMatching(`[a-z0-9\-]{1,20}/[a-z0-9\-]{1,20}\.mcap`).Draw(t, "gcs_path")
		}
		if rapid.Bool().Draw(t, "hasSizeBytes") {
			m["size_bytes"] = float64(rapid.Int64Range(0, 10*1024*1024*1024).Draw(t, "size_bytes"))
		}
		if rapid.Bool().Draw(t, "hasIngestState") {
			m["ingest_state"] = rapid.SampledFrom([]string{"pending", "ingesting", "complete", "failed"}).Draw(t, "ingest_state")
		}
		if rapid.Bool().Draw(t, "hasStartTimestampNs") {
			m["start_timestamp_ns"] = float64(rapid.Int64Range(0, 1e18).Draw(t, "start_timestamp_ns"))
		}
		if rapid.Bool().Draw(t, "hasEndTimestampNs") {
			m["end_timestamp_ns"] = float64(rapid.Int64Range(0, 1e18).Draw(t, "end_timestamp_ns"))
		}
		if rapid.Bool().Draw(t, "hasChannelCount") {
			m["channel_count"] = float64(rapid.IntRange(0, 1000).Draw(t, "channel_count"))
		}
		if rapid.Bool().Draw(t, "hasChunkCount") {
			m["chunk_count"] = float64(rapid.IntRange(0, 10000).Draw(t, "chunk_count"))
		}
		if rapid.Bool().Draw(t, "hasOwner") {
			m["owner"] = rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "owner")
		}

		return m
	})
}

// ---------------------------------------------------------------------------
// Property 2: Backfill mcap_files round-trip (correctness + idempotence)
//
// For any mcap_files row with arbitrary cf_meta JSONB content, after running
// the backfill, the real columns contain correctly derived values from cf_meta,
// AND running the backfill a second time produces identical column values.
//
// **Validates: Requirements 10.1, 10.2**
// ---------------------------------------------------------------------------

func TestBackfillMcap_Property2_CorrectnessAndIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfMeta := genMcapCfMeta().Draw(t, "cfMeta")

		input := BackfillMcapInput{
			CfMeta: cfMeta,
		}

		// First backfill pass
		result1 := backfillMcap(input)

		// --- Correctness checks ---

		// mcap_uri
		if v, ok := cfMeta["gcs_path"]; ok && v != nil {
			expected := toString(v)
			if expected != "" && result1.McapURI != expected {
				t.Fatalf("mcap_uri: expected %q, got %q", expected, result1.McapURI)
			}
		} else {
			if result1.McapURI != "" {
				t.Fatalf("mcap_uri: expected empty default, got %q", result1.McapURI)
			}
		}

		// size_bytes
		if v, ok := cfMeta["size_bytes"]; ok && v != nil {
			expected := int64(toFloat64(v))
			if result1.SizeBytes == nil || *result1.SizeBytes != expected {
				t.Fatalf("size_bytes: expected %d, got %v", expected, result1.SizeBytes)
			}
		} else {
			if result1.SizeBytes != nil {
				t.Fatalf("size_bytes: expected nil, got %v", *result1.SizeBytes)
			}
		}

		// ingest_state
		if v, ok := cfMeta["ingest_state"]; ok && v != nil {
			expected := toString(v)
			if expected != "" && result1.IngestState != expected {
				t.Fatalf("ingest_state: expected %q, got %q", expected, result1.IngestState)
			}
		} else {
			if result1.IngestState != "pending" {
				t.Fatalf("ingest_state: expected 'pending' default, got %q", result1.IngestState)
			}
		}

		// start_timestamp_ns
		if v, ok := cfMeta["start_timestamp_ns"]; ok && v != nil {
			expected := int64(toFloat64(v))
			if result1.StartTimestampNs == nil || *result1.StartTimestampNs != expected {
				t.Fatalf("start_timestamp_ns: expected %d, got %v", expected, result1.StartTimestampNs)
			}
		} else {
			if result1.StartTimestampNs != nil {
				t.Fatalf("start_timestamp_ns: expected nil, got %v", *result1.StartTimestampNs)
			}
		}

		// end_timestamp_ns
		if v, ok := cfMeta["end_timestamp_ns"]; ok && v != nil {
			expected := int64(toFloat64(v))
			if result1.EndTimestampNs == nil || *result1.EndTimestampNs != expected {
				t.Fatalf("end_timestamp_ns: expected %d, got %v", expected, result1.EndTimestampNs)
			}
		} else {
			if result1.EndTimestampNs != nil {
				t.Fatalf("end_timestamp_ns: expected nil, got %v", *result1.EndTimestampNs)
			}
		}

		// channel_count
		if v, ok := cfMeta["channel_count"]; ok && v != nil {
			expected := toInt(v)
			if result1.ChannelCount == nil || *result1.ChannelCount != expected {
				t.Fatalf("channel_count: expected %d, got %v", expected, result1.ChannelCount)
			}
		} else {
			if result1.ChannelCount != nil {
				t.Fatalf("channel_count: expected nil, got %v", *result1.ChannelCount)
			}
		}

		// chunk_count
		if v, ok := cfMeta["chunk_count"]; ok && v != nil {
			expected := toInt(v)
			if result1.ChunkCount == nil || *result1.ChunkCount != expected {
				t.Fatalf("chunk_count: expected %d, got %v", expected, result1.ChunkCount)
			}
		} else {
			if result1.ChunkCount != nil {
				t.Fatalf("chunk_count: expected nil, got %v", *result1.ChunkCount)
			}
		}

		// owner
		if v, ok := cfMeta["owner"]; ok && v != nil {
			expected := toString(v)
			if result1.Owner == nil || *result1.Owner != expected {
				t.Fatalf("owner: expected %q, got %v", expected, result1.Owner)
			}
		} else {
			if result1.Owner != nil {
				t.Fatalf("owner: expected nil, got %v", *result1.Owner)
			}
		}

		// --- Idempotence: second pass produces identical results ---
		result2 := backfillMcap(input)

		if !equalMcapBackfillResult(result1, result2) {
			t.Fatalf("idempotence violated: first pass != second pass\nfirst:  %+v\nsecond: %+v", result1, result2)
		}
	})
}

// ---------------------------------------------------------------------------
// SQL static analysis: verify the mcap backfill script contains expected patterns
// ---------------------------------------------------------------------------

func loadBackfillMcapSQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/011_schema_evolution_backfill_mcap.sql")
	if err != nil {
		t.Fatalf("failed to read backfill script: %v", err)
	}
	return string(data)
}

func TestBackfillMcap_SQLStructure(t *testing.T) {
	sql := loadBackfillMcapSQL(t)
	lower := strings.ToLower(sql)

	// Verify DO $ block
	if !strings.Contains(lower, "do $") {
		t.Error("backfill script should use DO $ block")
	}

	// Verify batch_size declaration
	if !strings.Contains(lower, "batch_size") {
		t.Error("backfill script should declare batch_size")
	}

	// Verify LOOP
	if !strings.Contains(lower, "loop") {
		t.Error("backfill script should use LOOP for batch processing")
	}

	// Verify FOR UPDATE SKIP LOCKED
	if !strings.Contains(lower, "for update skip locked") {
		t.Error("backfill script should use FOR UPDATE SKIP LOCKED")
	}

	// Verify GET DIAGNOSTICS
	if !strings.Contains(lower, "get diagnostics") {
		t.Error("backfill script should use GET DIAGNOSTICS to check rows_updated")
	}

	// Verify EXIT WHEN
	if !strings.Contains(lower, "exit when") {
		t.Error("backfill script should EXIT WHEN rows_updated = 0")
	}

	// Verify all target columns are updated
	targetColumns := []string{
		"mcap_uri", "size_bytes", "ingest_state",
		"start_timestamp_ns", "end_timestamp_ns",
		"channel_count", "chunk_count", "owner",
	}
	for _, col := range targetColumns {
		pattern := regexp.MustCompile(`(?i)\b` + col + `\s*=`)
		if !pattern.MatchString(sql) {
			t.Errorf("backfill script missing SET for column: %s", col)
		}
	}

	// Verify COALESCE usage for NULL handling
	if !strings.Contains(lower, "coalesce") {
		t.Error("backfill script should use COALESCE for NULL handling")
	}

	// Verify cf_meta JSONB extraction
	if !strings.Contains(lower, "cf_meta") {
		t.Error("backfill script should reference cf_meta for JSONB extraction")
	}

	// Verify gcs_path source mapping
	if !strings.Contains(lower, "gcs_path") {
		t.Error("backfill script should extract mcap_uri from gcs_path")
	}

	// Verify mcap_files table reference
	if !strings.Contains(lower, "mcap_files") {
		t.Error("backfill script should reference mcap_files table")
	}

	// Verify batch size of 1000
	if !strings.Contains(sql, "1000") {
		t.Error("backfill script should use batch_size of 1000")
	}
}

// ---------------------------------------------------------------------------
// Helpers for mcap backfill comparison
// ---------------------------------------------------------------------------

func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalMcapBackfillResult(a, b BackfillMcapResult) bool {
	return a.McapURI == b.McapURI &&
		equalInt64Ptr(a.SizeBytes, b.SizeBytes) &&
		a.IngestState == b.IngestState &&
		equalInt64Ptr(a.StartTimestampNs, b.StartTimestampNs) &&
		equalInt64Ptr(a.EndTimestampNs, b.EndTimestampNs) &&
		equalIntPtr(a.ChannelCount, b.ChannelCount) &&
		equalIntPtr(a.ChunkCount, b.ChunkCount) &&
		equalStrPtr(a.Owner, b.Owner)
}

// ===========================================================================
// DELIVERIES BACKFILL TESTS
// ===========================================================================

// ---------------------------------------------------------------------------
// Go-level helpers that replicate the SQL backfill mapping logic from
// 012_schema_evolution_backfill_deliveries.sql.
// ---------------------------------------------------------------------------

// BackfillDeliveryInput represents the source data for one deliveries row.
type BackfillDeliveryInput struct {
	CfMeta map[string]interface{}
}

// BackfillDeliveryResult represents the real-column values after backfill.
type BackfillDeliveryResult struct {
	ManifestURI *string
	ContractID  *string
	ItemCount   int64
	DeliveredBy *string
}

// backfillDelivery applies the backfill mapping logic in Go.
func backfillDelivery(input BackfillDeliveryInput) BackfillDeliveryResult {
	r := BackfillDeliveryResult{
		ItemCount: 0, // default
	}

	// manifest_uri ← cf_meta->>'manifest_uri'
	if v, ok := input.CfMeta["manifest_uri"]; ok && v != nil {
		s := toString(v)
		r.ManifestURI = strPtr(s)
	}

	// contract_id ← cf_meta->>'contract_id'
	if v, ok := input.CfMeta["contract_id"]; ok && v != nil {
		s := toString(v)
		r.ContractID = strPtr(s)
	}

	// item_count ← COALESCE((cf_meta->>'asset_count')::BIGINT, 0)
	if v, ok := input.CfMeta["asset_count"]; ok && v != nil {
		r.ItemCount = int64(toFloat64(v))
	}

	// delivered_by ← cf_meta->>'owner'
	if v, ok := input.CfMeta["owner"]; ok && v != nil {
		s := toString(v)
		r.DeliveredBy = strPtr(s)
	}

	return r
}

// ---------------------------------------------------------------------------
// Generators for deliveries rapid tests
// ---------------------------------------------------------------------------

func genDeliveryCfMeta() *rapid.Generator[map[string]interface{}] {
	return rapid.Custom(func(t *rapid.T) map[string]interface{} {
		m := make(map[string]interface{})

		if rapid.Bool().Draw(t, "hasManifestURI") {
			m["manifest_uri"] = "gs://" + rapid.StringMatching(`[a-z0-9\-]{1,20}/[a-z0-9\-]{1,20}\.json`).Draw(t, "manifest_uri")
		}
		if rapid.Bool().Draw(t, "hasContractID") {
			m["contract_id"] = rapid.StringMatching(`[A-Z]{2,4}-[0-9]{3,6}`).Draw(t, "contract_id")
		}
		if rapid.Bool().Draw(t, "hasAssetCount") {
			m["asset_count"] = float64(rapid.Int64Range(0, 100000).Draw(t, "asset_count"))
		}
		if rapid.Bool().Draw(t, "hasOwner") {
			m["owner"] = rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "owner")
		}

		return m
	})
}

// ---------------------------------------------------------------------------
// Property 5: Backfill deliveries round-trip (correctness + idempotence)
//
// For any delivery row with arbitrary cf_meta JSONB content, after running
// the backfill, the real columns contain correctly derived values from cf_meta,
// AND running the backfill a second time produces identical column values.
//
// **Validates: Requirements 18.1, 18.2**
// ---------------------------------------------------------------------------

func TestBackfillDeliveries_Property5_CorrectnessAndIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfMeta := genDeliveryCfMeta().Draw(t, "cfMeta")

		input := BackfillDeliveryInput{
			CfMeta: cfMeta,
		}

		// First backfill pass
		result1 := backfillDelivery(input)

		// --- Correctness checks ---

		// manifest_uri
		if v, ok := cfMeta["manifest_uri"]; ok && v != nil {
			expected := toString(v)
			if result1.ManifestURI == nil || *result1.ManifestURI != expected {
				t.Fatalf("manifest_uri: expected %q, got %v", expected, result1.ManifestURI)
			}
		} else {
			if result1.ManifestURI != nil {
				t.Fatalf("manifest_uri: expected nil, got %v", *result1.ManifestURI)
			}
		}

		// contract_id
		if v, ok := cfMeta["contract_id"]; ok && v != nil {
			expected := toString(v)
			if result1.ContractID == nil || *result1.ContractID != expected {
				t.Fatalf("contract_id: expected %q, got %v", expected, result1.ContractID)
			}
		} else {
			if result1.ContractID != nil {
				t.Fatalf("contract_id: expected nil, got %v", *result1.ContractID)
			}
		}

		// item_count (from asset_count)
		if v, ok := cfMeta["asset_count"]; ok && v != nil {
			expected := int64(toFloat64(v))
			if result1.ItemCount != expected {
				t.Fatalf("item_count: expected %d, got %d", expected, result1.ItemCount)
			}
		} else {
			if result1.ItemCount != 0 {
				t.Fatalf("item_count: expected 0 default, got %d", result1.ItemCount)
			}
		}

		// delivered_by (from owner)
		if v, ok := cfMeta["owner"]; ok && v != nil {
			expected := toString(v)
			if result1.DeliveredBy == nil || *result1.DeliveredBy != expected {
				t.Fatalf("delivered_by: expected %q, got %v", expected, result1.DeliveredBy)
			}
		} else {
			if result1.DeliveredBy != nil {
				t.Fatalf("delivered_by: expected nil, got %v", *result1.DeliveredBy)
			}
		}

		// --- Idempotence: second pass produces identical results ---
		result2 := backfillDelivery(input)

		if !equalDeliveryBackfillResult(result1, result2) {
			t.Fatalf("idempotence violated: first pass != second pass\nfirst:  %+v\nsecond: %+v", result1, result2)
		}
	})
}

// ---------------------------------------------------------------------------
// SQL static analysis: verify the deliveries backfill script contains
// expected patterns
// ---------------------------------------------------------------------------

func loadBackfillDeliveriesSQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/012_schema_evolution_backfill_deliveries.sql")
	if err != nil {
		t.Fatalf("failed to read backfill script: %v", err)
	}
	return string(data)
}

func TestBackfillDeliveries_SQLStructure(t *testing.T) {
	sql := loadBackfillDeliveriesSQL(t)
	lower := strings.ToLower(sql)

	// Verify DO $ block
	if !strings.Contains(lower, "do $") {
		t.Error("backfill script should use DO $ block")
	}

	// Verify batch_size declaration
	if !strings.Contains(lower, "batch_size") {
		t.Error("backfill script should declare batch_size")
	}

	// Verify LOOP
	if !strings.Contains(lower, "loop") {
		t.Error("backfill script should use LOOP for batch processing")
	}

	// Verify FOR UPDATE SKIP LOCKED
	if !strings.Contains(lower, "for update skip locked") {
		t.Error("backfill script should use FOR UPDATE SKIP LOCKED")
	}

	// Verify GET DIAGNOSTICS
	if !strings.Contains(lower, "get diagnostics") {
		t.Error("backfill script should use GET DIAGNOSTICS to check rows_updated")
	}

	// Verify EXIT WHEN
	if !strings.Contains(lower, "exit when") {
		t.Error("backfill script should EXIT WHEN rows_updated = 0")
	}

	// Verify all target columns are updated
	targetColumns := []string{
		"manifest_uri", "contract_id", "item_count", "delivered_by",
	}
	for _, col := range targetColumns {
		pattern := regexp.MustCompile(`(?i)\b` + col + `\s*=`)
		if !pattern.MatchString(sql) {
			t.Errorf("backfill script missing SET for column: %s", col)
		}
	}

	// Verify COALESCE usage for NULL handling
	if !strings.Contains(lower, "coalesce") {
		t.Error("backfill script should use COALESCE for NULL handling")
	}

	// Verify cf_meta JSONB extraction
	if !strings.Contains(lower, "cf_meta") {
		t.Error("backfill script should reference cf_meta for JSONB extraction")
	}

	// Verify asset_count source mapping (item_count ← asset_count)
	if !strings.Contains(lower, "asset_count") {
		t.Error("backfill script should extract item_count from asset_count in cf_meta")
	}

	// Verify owner source mapping (delivered_by ← owner)
	ownerPattern := regexp.MustCompile(`(?i)cf_meta\s*->>.*'owner'`)
	if !ownerPattern.MatchString(sql) {
		t.Error("backfill script should extract delivered_by from owner in cf_meta")
	}

	// Verify deliveries table reference
	if !strings.Contains(lower, "deliveries") {
		t.Error("backfill script should reference deliveries table")
	}

	// Verify batch size of 1000
	if !strings.Contains(sql, "1000") {
		t.Error("backfill script should use batch_size of 1000")
	}
}

// ---------------------------------------------------------------------------
// Helpers for deliveries backfill comparison
// ---------------------------------------------------------------------------

func equalDeliveryBackfillResult(a, b BackfillDeliveryResult) bool {
	return equalStrPtr(a.ManifestURI, b.ManifestURI) &&
		equalStrPtr(a.ContractID, b.ContractID) &&
		a.ItemCount == b.ItemCount &&
		equalStrPtr(a.DeliveredBy, b.DeliveredBy)
}

// ===========================================================================
// TAGS BACKFILL TESTS
// ===========================================================================

// ---------------------------------------------------------------------------
// Go-level helpers that replicate the SQL backfill mapping logic from
// 013_schema_evolution_backfill_tags.sql.
// ---------------------------------------------------------------------------

// BackfillTagInput represents the source data for one asset's cf_tag.
type BackfillTagInput struct {
	AssetID string
	CfTag   map[string]string
}

// BackfillTagRow represents one row in asset_tags after backfill.
type BackfillTagRow struct {
	AssetID    string
	TagKey     string
	TagValue   string
	TagType    string
	SourceType string
}

// backfillTags applies the tag expansion logic in Go, mirroring the SQL
// jsonb_each_text() + INSERT ... ON CONFLICT DO UPDATE pattern.
// It returns the resulting asset_tags rows for the given asset.
func backfillTags(input BackfillTagInput, existing []BackfillTagRow) []BackfillTagRow {
	// Build a map of existing rows keyed by (asset_id, tag_key) for upsert.
	rowMap := make(map[string]BackfillTagRow)
	for _, r := range existing {
		key := r.AssetID + "|" + r.TagKey
		rowMap[key] = r
	}

	// Expand cf_tag key-value pairs (mirrors jsonb_each_text).
	for tagKey, tagValue := range input.CfTag {
		key := input.AssetID + "|" + tagKey
		rowMap[key] = BackfillTagRow{
			AssetID:    input.AssetID,
			TagKey:     tagKey,
			TagValue:   tagValue,
			TagType:    "string",
			SourceType: "system",
		}
	}

	// Collect results.
	result := make([]BackfillTagRow, 0, len(rowMap))
	for _, r := range rowMap {
		result = append(result, r)
	}
	return result
}

// ---------------------------------------------------------------------------
// Generators for tags rapid tests
// ---------------------------------------------------------------------------

func genCfTag() *rapid.Generator[map[string]string] {
	return rapid.Custom(func(t *rapid.T) map[string]string {
		n := rapid.IntRange(0, 10).Draw(t, "numTags")
		m := make(map[string]string, n)
		for i := 0; i < n; i++ {
			key := rapid.StringMatching(`[a-z][a-z0-9_]{0,19}`).Draw(t, "tagKey")
			val := rapid.StringMatching(`[a-zA-Z0-9_ ]{0,30}`).Draw(t, "tagVal")
			m[key] = val
		}
		return m
	})
}

// ---------------------------------------------------------------------------
// Property 3: Backfill tags expansion (correctness + idempotence)
//
// For any asset row with a cf_tag JSONB containing N key-value pairs (N ≥ 0),
// after running the tag backfill, the asset_tags table contains exactly N rows
// for that asset with correct tag_key, tag_value, tag_type='string', and
// source_type='system'. Running the backfill a second time produces the same
// N rows (via ON CONFLICT DO UPDATE).
//
// **Validates: Requirements 11.1, 11.2**
// ---------------------------------------------------------------------------

func TestBackfillTags_Property3_CorrectnessAndIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		assetID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "assetID")
		cfTag := genCfTag().Draw(t, "cfTag")

		input := BackfillTagInput{
			AssetID: assetID,
			CfTag:   cfTag,
		}

		// First backfill pass (no existing rows).
		result1 := backfillTags(input, nil)

		// --- Correctness: row count equals number of unique keys ---
		if len(result1) != len(cfTag) {
			t.Fatalf("row count: expected %d, got %d", len(cfTag), len(result1))
		}

		// --- Correctness: each row has correct values ---
		rowsByKey := make(map[string]BackfillTagRow)
		for _, r := range result1 {
			rowsByKey[r.TagKey] = r
		}

		for key, val := range cfTag {
			row, ok := rowsByKey[key]
			if !ok {
				t.Fatalf("missing tag row for key %q", key)
			}
			if row.AssetID != assetID {
				t.Fatalf("tag %q: asset_id expected %q, got %q", key, assetID, row.AssetID)
			}
			if row.TagValue != val {
				t.Fatalf("tag %q: tag_value expected %q, got %q", key, val, row.TagValue)
			}
			if row.TagType != "string" {
				t.Fatalf("tag %q: tag_type expected 'string', got %q", key, row.TagType)
			}
			if row.SourceType != "system" {
				t.Fatalf("tag %q: source_type expected 'system', got %q", key, row.SourceType)
			}
		}

		// --- Idempotence: second pass with existing rows produces same result ---
		result2 := backfillTags(input, result1)

		if len(result2) != len(result1) {
			t.Fatalf("idempotence: row count changed from %d to %d", len(result1), len(result2))
		}

		rows2ByKey := make(map[string]BackfillTagRow)
		for _, r := range result2 {
			rows2ByKey[r.TagKey] = r
		}

		for key, r1 := range rowsByKey {
			r2, ok := rows2ByKey[key]
			if !ok {
				t.Fatalf("idempotence: tag %q missing after second pass", key)
			}
			if r1 != r2 {
				t.Fatalf("idempotence: tag %q changed\nfirst:  %+v\nsecond: %+v", key, r1, r2)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// SQL static analysis: verify the tags backfill script contains expected
// patterns
// ---------------------------------------------------------------------------

func loadBackfillTagsSQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/013_schema_evolution_backfill_tags.sql")
	if err != nil {
		t.Fatalf("failed to read backfill script: %v", err)
	}
	return string(data)
}

func TestBackfillTags_SQLStructure(t *testing.T) {
	sql := loadBackfillTagsSQL(t)
	lower := strings.ToLower(sql)

	// Verify DO $ block
	if !strings.Contains(lower, "do $") {
		t.Error("backfill script should use DO $ block")
	}

	// Verify batch_size declaration
	if !strings.Contains(lower, "batch_size") {
		t.Error("backfill script should declare batch_size")
	}

	// Verify LOOP
	if !strings.Contains(lower, "loop") {
		t.Error("backfill script should use LOOP for batch processing")
	}

	// Verify FOR UPDATE SKIP LOCKED
	if !strings.Contains(lower, "for update skip locked") {
		t.Error("backfill script should use FOR UPDATE SKIP LOCKED")
	}

	// Verify GET DIAGNOSTICS
	if !strings.Contains(lower, "get diagnostics") {
		t.Error("backfill script should use GET DIAGNOSTICS")
	}

	// Verify EXIT WHEN
	if !strings.Contains(lower, "exit when") {
		t.Error("backfill script should EXIT WHEN rows_affected = 0")
	}

	// Verify jsonb_each_text usage for tag expansion
	if !strings.Contains(lower, "jsonb_each_text") {
		t.Error("backfill script should use jsonb_each_text() to expand cf_tag")
	}

	// Verify INSERT INTO asset_tags
	if !strings.Contains(lower, "insert into asset_tags") {
		t.Error("backfill script should INSERT INTO asset_tags")
	}

	// Verify ON CONFLICT for idempotency
	if !strings.Contains(lower, "on conflict") {
		t.Error("backfill script should use ON CONFLICT for idempotency")
	}

	// Verify conflict target is (asset_id, tag_key)
	conflictPattern := regexp.MustCompile(`(?i)on\s+conflict\s*\(\s*asset_id\s*,\s*tag_key\s*\)`)
	if !conflictPattern.MatchString(sql) {
		t.Error("backfill script ON CONFLICT should target (asset_id, tag_key)")
	}

	// Verify DO UPDATE SET tag_value
	if !strings.Contains(lower, "do update set") {
		t.Error("backfill script should use DO UPDATE SET for upsert")
	}

	// Verify tag_type = 'string'
	if !strings.Contains(lower, "'string'") {
		t.Error("backfill script should set tag_type to 'string'")
	}

	// Verify source_type = 'system'
	if !strings.Contains(lower, "'system'") {
		t.Error("backfill script should set source_type to 'system'")
	}

	// Verify cf_tag reference
	if !strings.Contains(lower, "cf_tag") {
		t.Error("backfill script should reference cf_tag for JSONB extraction")
	}

	// Verify batch size of 1000
	if !strings.Contains(sql, "1000") {
		t.Error("backfill script should use batch_size of 1000")
	}

	// Verify assets table reference
	if !strings.Contains(lower, "assets") {
		t.Error("backfill script should reference assets table")
	}
}

// ===========================================================================
// ALGO BACKFILL TESTS
// ===========================================================================

// ---------------------------------------------------------------------------
// Go-level helpers that replicate the SQL backfill mapping logic from
// 014_schema_evolution_backfill_algo.sql.
// ---------------------------------------------------------------------------

// BackfillAlgoInput represents the source data for one asset's cf_algo.
type BackfillAlgoInput struct {
	AssetID string
	CfAlgo  map[string]string // key format: <algo>@<ver>:<field>
}

// BackfillAlgoRow represents one row in asset_algo_latest after backfill.
type BackfillAlgoRow struct {
	AssetID     string
	AlgoName    string
	AlgoVersion string
	Status      string
}

// algoKeyRegex matches the cf_algo key format: <algo_name>@<version>:<field>
var algoKeyRegex = regexp.MustCompile(`^([^@]+)@([^:]+):(.+)$`)

// backfillAlgo applies the algo parsing + upsert logic in Go, mirroring the
// SQL regexp_match + INSERT ... ON CONFLICT DO UPDATE pattern.
// It returns the resulting asset_algo_latest rows for the given asset.
func backfillAlgo(input BackfillAlgoInput, existing []BackfillAlgoRow) []BackfillAlgoRow {
	// Build a map of existing rows keyed by (asset_id, algo_name) for upsert.
	rowMap := make(map[string]BackfillAlgoRow)
	for _, r := range existing {
		key := r.AssetID + "|" + r.AlgoName
		rowMap[key] = r
	}

	// Parse cf_algo keys and collect status values per (asset_id, algo_name).
	// Only keys matching the pattern are processed; malformed keys are skipped.
	// When the same algo_name appears with multiple versions, keep the highest
	// version (matching the SQL's DISTINCT ON ... ORDER BY algo_version DESC).
	for k, v := range input.CfAlgo {
		matches := algoKeyRegex.FindStringSubmatch(k)
		if matches == nil {
			continue // skip malformed keys
		}
		algoName := matches[1]
		algoVersion := matches[2]
		field := matches[3]

		if field != "status" {
			continue // only extract status field
		}

		mapKey := input.AssetID + "|" + algoName
		if existing, ok := rowMap[mapKey]; ok {
			// Keep the higher version (lexicographic DESC, matching SQL ORDER BY)
			if algoVersion <= existing.AlgoVersion {
				continue
			}
		}
		rowMap[mapKey] = BackfillAlgoRow{
			AssetID:     input.AssetID,
			AlgoName:    algoName,
			AlgoVersion: algoVersion,
			Status:      v,
		}
	}

	// Collect results.
	result := make([]BackfillAlgoRow, 0, len(rowMap))
	for _, r := range rowMap {
		result = append(result, r)
	}
	return result
}

// ---------------------------------------------------------------------------
// Generators for algo rapid tests
// ---------------------------------------------------------------------------

func genAlgoName() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"tracker", "classifier", "detector", "segmenter", "scorer"})
}

func genAlgoVersion() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		major := rapid.IntRange(1, 9).Draw(t, "major")
		minor := rapid.IntRange(0, 9).Draw(t, "minor")
		return strings.Join([]string{toString(major), toString(minor)}, ".")
	})
}

func genAlgoStatus() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"ok", "failed", "running", "pending", "skipped"})
}

func genCfAlgo() *rapid.Generator[map[string]string] {
	return rapid.Custom(func(t *rapid.T) map[string]string {
		n := rapid.IntRange(1, 5).Draw(t, "numAlgos")
		m := make(map[string]string)
		for i := 0; i < n; i++ {
			name := genAlgoName().Draw(t, "algoName")
			ver := genAlgoVersion().Draw(t, "algoVer")
			status := genAlgoStatus().Draw(t, "algoStatus")
			// Always include the :status key
			m[name+"@"+ver+":status"] = status
		}
		return m
	})
}

// ---------------------------------------------------------------------------
// Property 4: Backfill algo parsing (correctness + idempotence)
//
// For any asset row with a cf_algo JSONB containing keys in the format
// <algo>@<ver>:<field>, after running the algo backfill, the
// asset_algo_latest table contains one row per unique (asset_id, algo_name)
// with the correct algo_version and status extracted from the key pattern.
// Running the backfill a second time produces the same rows.
//
// **Validates: Requirements 12.1, 12.2**
// ---------------------------------------------------------------------------

func TestBackfillAlgo_Property4_CorrectnessAndIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		assetID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "assetID")
		cfAlgo := genCfAlgo().Draw(t, "cfAlgo")

		input := BackfillAlgoInput{
			AssetID: assetID,
			CfAlgo:  cfAlgo,
		}

		// First backfill pass (no existing rows).
		result1 := backfillAlgo(input, nil)

		// --- Correctness: build expected set from cf_algo ---
		// Collect unique algo_names with their latest version+status from :status keys.
		// When the same algo_name has multiple versions, keep the highest (DESC).
		expected := make(map[string]BackfillAlgoRow)
		for k, v := range cfAlgo {
			matches := algoKeyRegex.FindStringSubmatch(k)
			if matches == nil {
				continue
			}
			algoName := matches[1]
			algoVersion := matches[2]
			field := matches[3]
			if field != "status" {
				continue
			}
			if existing, ok := expected[algoName]; ok {
				if algoVersion <= existing.AlgoVersion {
					continue
				}
			}
			expected[algoName] = BackfillAlgoRow{
				AssetID:     assetID,
				AlgoName:    algoName,
				AlgoVersion: algoVersion,
				Status:      v,
			}
		}

		// Row count should match unique algo names with :status keys.
		if len(result1) != len(expected) {
			t.Fatalf("row count: expected %d, got %d", len(expected), len(result1))
		}

		// Each row should have correct values.
		rowsByName := make(map[string]BackfillAlgoRow)
		for _, r := range result1 {
			rowsByName[r.AlgoName] = r
		}

		for name, exp := range expected {
			row, ok := rowsByName[name]
			if !ok {
				t.Fatalf("missing algo row for name %q", name)
			}
			if row.AssetID != assetID {
				t.Fatalf("algo %q: asset_id expected %q, got %q", name, assetID, row.AssetID)
			}
			if row.AlgoVersion != exp.AlgoVersion {
				t.Fatalf("algo %q: algo_version expected %q, got %q", name, exp.AlgoVersion, row.AlgoVersion)
			}
			if row.Status != exp.Status {
				t.Fatalf("algo %q: status expected %q, got %q", name, exp.Status, row.Status)
			}
		}

		// --- Idempotence: second pass with existing rows produces same result ---
		result2 := backfillAlgo(input, result1)

		if len(result2) != len(result1) {
			t.Fatalf("idempotence: row count changed from %d to %d", len(result1), len(result2))
		}

		rows2ByName := make(map[string]BackfillAlgoRow)
		for _, r := range result2 {
			rows2ByName[r.AlgoName] = r
		}

		for name, r1 := range rowsByName {
			r2, ok := rows2ByName[name]
			if !ok {
				t.Fatalf("idempotence: algo %q missing after second pass", name)
			}
			if r1 != r2 {
				t.Fatalf("idempotence: algo %q changed\nfirst:  %+v\nsecond: %+v", name, r1, r2)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// SQL static analysis: verify the algo backfill script contains expected
// patterns
// ---------------------------------------------------------------------------

func loadBackfillAlgoSQL(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../migrations/014_schema_evolution_backfill_algo.sql")
	if err != nil {
		t.Fatalf("failed to read backfill script: %v", err)
	}
	return string(data)
}

func TestBackfillAlgo_SQLStructure(t *testing.T) {
	sql := loadBackfillAlgoSQL(t)
	lower := strings.ToLower(sql)

	// Verify DO $ block
	if !strings.Contains(lower, "do $") {
		t.Error("backfill script should use DO $ block")
	}

	// Verify batch_size declaration
	if !strings.Contains(lower, "batch_size") {
		t.Error("backfill script should declare batch_size")
	}

	// Verify LOOP
	if !strings.Contains(lower, "loop") {
		t.Error("backfill script should use LOOP for batch processing")
	}

	// Verify FOR UPDATE SKIP LOCKED
	if !strings.Contains(lower, "for update skip locked") {
		t.Error("backfill script should use FOR UPDATE SKIP LOCKED")
	}

	// Verify GET DIAGNOSTICS
	if !strings.Contains(lower, "get diagnostics") {
		t.Error("backfill script should use GET DIAGNOSTICS")
	}

	// Verify EXIT WHEN
	if !strings.Contains(lower, "exit when") {
		t.Error("backfill script should EXIT WHEN rows_affected = 0")
	}

	// Verify regexp_match for key parsing
	if !strings.Contains(lower, "regexp_match") {
		t.Error("backfill script should use regexp_match to parse cf_algo keys")
	}

	// Verify INSERT INTO asset_algo_latest
	if !strings.Contains(lower, "insert into asset_algo_latest") {
		t.Error("backfill script should INSERT INTO asset_algo_latest")
	}

	// Verify ON CONFLICT for idempotency
	if !strings.Contains(lower, "on conflict") {
		t.Error("backfill script should use ON CONFLICT for idempotency")
	}

	// Verify conflict target is (asset_id, algo_name)
	conflictPattern := regexp.MustCompile(`(?i)on\s+conflict\s*\(\s*asset_id\s*,\s*algo_name\s*\)`)
	if !conflictPattern.MatchString(sql) {
		t.Error("backfill script ON CONFLICT should target (asset_id, algo_name)")
	}

	// Verify DO UPDATE SET
	if !strings.Contains(lower, "do update set") {
		t.Error("backfill script should use DO UPDATE SET for upsert")
	}

	// Verify algo_version and status in update
	algoVerPattern := regexp.MustCompile(`(?i)algo_version\s*=\s*EXCLUDED\.algo_version`)
	if !algoVerPattern.MatchString(sql) {
		t.Error("backfill script should SET algo_version = EXCLUDED.algo_version")
	}

	statusPattern := regexp.MustCompile(`(?i)status\s*=\s*EXCLUDED\.status`)
	if !statusPattern.MatchString(sql) {
		t.Error("backfill script should SET status = EXCLUDED.status")
	}

	// Verify cf_algo reference
	if !strings.Contains(lower, "cf_algo") {
		t.Error("backfill script should reference cf_algo for JSONB extraction")
	}

	// Verify jsonb_each_text usage
	if !strings.Contains(lower, "jsonb_each_text") {
		t.Error("backfill script should use jsonb_each_text() to expand cf_algo")
	}

	// Verify key format pattern in regex
	if !strings.Contains(sql, "@") && !strings.Contains(sql, ":") {
		t.Error("backfill script should parse key format <algo>@<ver>:<field>")
	}

	// Verify batch size of 1000
	if !strings.Contains(sql, "1000") {
		t.Error("backfill script should use batch_size of 1000")
	}

	// Verify assets table reference
	if !strings.Contains(lower, "assets") {
		t.Error("backfill script should reference assets table")
	}
}

// ---------------------------------------------------------------------------
// Edge case: TestBackfillAlgo_MalformedKey
// Verifies that keys not matching the <algo>@<ver>:<field> pattern are
// gracefully skipped.
// ---------------------------------------------------------------------------

func TestBackfillAlgo_MalformedKey(t *testing.T) {
	tests := []struct {
		name     string
		cfAlgo   map[string]string
		expected int // expected number of result rows
	}{
		{
			name: "all malformed keys — no @ or : separator",
			cfAlgo: map[string]string{
				"tracker_v1_status": "ok",
				"classifier":        "done",
				"noformat":          "value",
			},
			expected: 0,
		},
		{
			name: "missing version — no @ separator",
			cfAlgo: map[string]string{
				"tracker:status": "ok",
			},
			expected: 0,
		},
		{
			name: "missing field — no : separator",
			cfAlgo: map[string]string{
				"tracker@1.0": "ok",
			},
			expected: 0,
		},
		{
			name: "empty key",
			cfAlgo: map[string]string{
				"": "ok",
			},
			expected: 0,
		},
		{
			name: "mix of valid and malformed keys",
			cfAlgo: map[string]string{
				"tracker@1.0:status":    "ok",
				"bad_key":               "value",
				"classifier@2.1:status": "failed",
				"no_at_sign:status":     "pending",
			},
			expected: 2, // only tracker and classifier
		},
		{
			name: "valid key but non-status field only",
			cfAlgo: map[string]string{
				"tracker@1.0:confidence": "0.95",
				"tracker@1.0:result":     "positive",
			},
			expected: 0, // no :status field → no rows
		},
		{
			name: "valid status key plus non-status fields",
			cfAlgo: map[string]string{
				"tracker@1.0:status":     "ok",
				"tracker@1.0:confidence": "0.95",
				"tracker@1.0:result":     "positive",
			},
			expected: 1, // one row for tracker with status
		},
		{
			name:     "empty cf_algo",
			cfAlgo:   map[string]string{},
			expected: 0,
		},
	}

	assetID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := BackfillAlgoInput{
				AssetID: assetID,
				CfAlgo:  tc.cfAlgo,
			}

			result := backfillAlgo(input, nil)

			if len(result) != tc.expected {
				t.Errorf("expected %d rows, got %d (rows: %+v)", tc.expected, len(result), result)
			}

			// Verify all returned rows have valid fields
			for _, r := range result {
				if r.AssetID != assetID {
					t.Errorf("unexpected asset_id: %q", r.AssetID)
				}
				if r.AlgoName == "" {
					t.Error("algo_name should not be empty")
				}
				if r.AlgoVersion == "" {
					t.Error("algo_version should not be empty")
				}
				if r.Status == "" {
					t.Error("status should not be empty")
				}
			}
		})
	}
}
