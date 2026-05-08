package filter

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// ── Property 12: Filter promoted field resolution ────────────────────────────
// For any promoted field name in {owner, reviewer, lifecycle_state, asset_type,
// delivery_count, last_delivered_at, last_delivered_to, retention_tier, expire_at},
// ResolveField() SHALL return a fieldSpec with IsJSONB = false and StorageField
// equal to the column name.
// **Validates: Requirements 15.1**
func TestProperty12_PromotedFieldResolution(t *testing.T) {
	promotedFields := []string{
		"owner",
		"reviewer",
		"lifecycle_state",
		"asset_type",
		"delivery_count",
		"last_delivered_at",
		"last_delivered_to",
		"retention_tier",
		"expire_at",
	}

	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(promotedFields)-1).Draw(t, "fieldIndex")
		field := promotedFields[idx]

		spec, err := ResolveField(field)
		if err != nil {
			t.Fatalf("ResolveField(%q) returned error: %v", field, err)
		}

		if spec.IsJSONB {
			t.Fatalf("ResolveField(%q): expected IsJSONB=false, got true", field)
		}

		if spec.StorageField != field {
			t.Fatalf("ResolveField(%q): expected StorageField=%q, got %q", field, field, spec.StorageField)
		}
	})
}

// ── Property 13: Filter JSONB prefix support preserved ───────────────────────
// For any valid dynamic field key with prefix tag.*, algo.*, or files.*,
// ResolveField() SHALL continue to return a fieldSpec with IsJSONB = true and
// StorageField pointing to the corresponding projection/JSON path.
// **Validates: Requirements 15.4**
func TestProperty13_JSONBPrefixSupportPreserved(t *testing.T) {
	type prefixCase struct {
		inputPrefix   string
		storagePrefix string
	}
	prefixes := []prefixCase{
		{"tag.", "asset_tags."},
		{"tags.", "asset_tags."},
		{"algo.", "asset_algo_latest."},
		{"algo_results.", "asset_algo_latest."},
		{"files.", "files."},
	}

	// Generate random valid key suffixes
	rapid.Check(t, func(t *rapid.T) {
		prefixIdx := rapid.IntRange(0, len(prefixes)-1).Draw(t, "prefixIndex")
		pc := prefixes[prefixIdx]

		// Generate a valid dynamic key: alphanumeric + allowed special chars
		key := rapid.StringMatching(`[a-z][a-z0-9_]{0,9}`).Draw(t, "key")
		if key == "" {
			key = "x"
		}

		field := pc.inputPrefix + key

		spec, err := ResolveField(field)
		if err != nil {
			t.Fatalf("ResolveField(%q) returned error: %v", field, err)
		}

		if !spec.IsJSONB {
			t.Fatalf("ResolveField(%q): expected IsJSONB=true, got false", field)
		}

		expectedStorage := pc.storagePrefix + key
		if spec.StorageField != expectedStorage {
			t.Fatalf("ResolveField(%q): expected StorageField=%q, got %q", field, expectedStorage, spec.StorageField)
		}
	})
}

// ── TestFilterResolve_DurationSecConversion ──────────────────────────────────
// duration_sec should map to the duration_ms real column (IsJSONB=false).
// The canonical name stays "duration_sec" but storage points to "duration_ms".
func TestFilterResolve_DurationSecConversion(t *testing.T) {
	spec, err := ResolveField("duration_sec")
	if err != nil {
		t.Fatalf("ResolveField(duration_sec) error: %v", err)
	}

	if spec.Canonical != "duration_sec" {
		t.Errorf("expected Canonical=duration_sec, got %q", spec.Canonical)
	}
	if spec.StorageField != "duration_ms" {
		t.Errorf("expected StorageField=duration_ms, got %q", spec.StorageField)
	}
	if spec.IsJSONB {
		t.Error("expected IsJSONB=false for duration_sec (now mapped to duration_ms column)")
	}

	// Verify sort also resolves to the real column
	sortExpr, err := ResolveSortBy("duration_sec")
	if err != nil {
		t.Fatalf("ResolveSortBy(duration_sec) error: %v", err)
	}
	if !strings.Contains(sortExpr, "duration_ms") {
		t.Errorf("expected sort expression to reference duration_ms, got %q", sortExpr)
	}
	if strings.Contains(sortExpr, "#>>") {
		t.Errorf("expected no JSONB extraction in sort expression, got %q", sortExpr)
	}
}

// ── Additional unit tests for promoted fields ────────────────────────────────

func TestResolveField_NewPromotedFields(t *testing.T) {
	cases := []struct {
		field       string
		wantStorage string
		wantJSONB   bool
	}{
		{"lifecycle_state", "lifecycle_state", false},
		{"asset_type", "asset_type", false},
		{"expire_at", "expire_at", false},
		{"owner", "owner", false},
		{"reviewer", "reviewer", false},
		{"delivery_count", "delivery_count", false},
		{"last_delivered_to", "last_delivered_to", false},
		{"last_delivered_at", "last_delivered_at", false},
		{"retention_tier", "retention_tier", false},
		{"tags.key", "asset_tags.__key", false},
		{"tags.value", "asset_tags.__value", false},
		{"tags.source_type", "asset_tags.__source_type", false},
		{"tags.confidence", "asset_tags.__confidence", false},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			spec, err := ResolveField(tc.field)
			if err != nil {
				t.Fatalf("ResolveField(%q) error: %v", tc.field, err)
			}
			if spec.StorageField != tc.wantStorage {
				t.Errorf("StorageField: got %q, want %q", spec.StorageField, tc.wantStorage)
			}
			if spec.IsJSONB != tc.wantJSONB {
				t.Errorf("IsJSONB: got %v, want %v", spec.IsJSONB, tc.wantJSONB)
			}
		})
	}
}

func TestResolveField_JSONBPrefixesStillWork(t *testing.T) {
	cases := []struct {
		field       string
		wantStorage string
	}{
		{"tag.priority", "asset_tags.priority"},
		{"algo.hand_tracking@1.2.0:status", "asset_algo_latest.hand_tracking@1.2.0:status"},
		{"files.preview_mp4", "files.preview_mp4"},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			spec, err := ResolveField(tc.field)
			if err != nil {
				t.Fatalf("ResolveField(%q) error: %v", tc.field, err)
			}
			if spec.StorageField != tc.wantStorage {
				t.Errorf("StorageField: got %q, want %q", spec.StorageField, tc.wantStorage)
			}
			if !spec.IsJSONB {
				t.Error("expected IsJSONB=true for prefixed field")
			}
		})
	}
}
