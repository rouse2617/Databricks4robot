package mcap

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Pin the 3-rule extraction on a pangzi-shape mcap payload. This is the
// contract for CYB-3797: the same 3 tag rows must land, in some order,
// every time.
func TestExtractMetadataTags_PangziShape(t *testing.T) {
	meta := map[string]any{
		"vibecap_tasks":   []any{"备餐操作", "台面清洁"},
		"source_platform": "vibecap",
		"location": map[string]any{
			"address": "合肥新民医院, 合瓦路, 上城, 上城社区, 杏林街道, 庐阳区, 合肥市, 安徽省, 230061, 中国",
		},
		// Unknown fields — must be ignored, not create extra tags.
		"collector_height":      1.6,
		"collection_session_id": "3b2f52a4-a5d6-4c1c-bb7d-82c8a502efcc",
		"weather":               "sunny",
	}
	got := extractMetadataTags("mcap123", "t1", "p1", meta)
	// 2 task + 1 source + 1 city = 4 rows.
	if len(got) != 4 {
		t.Fatalf("expected 4 tags, got %d: %+v", len(got), got)
	}
	assertContainsTag(t, got, "task", "备餐操作")
	assertContainsTag(t, got, "task", "台面清洁")
	assertContainsTag(t, got, "source", "vibecap")
	assertContainsTag(t, got, "city", "合肥市")
	// Every tag must carry the auto-extract source-type + source-name so we
	// can audit + selectively re-derive later.
	for _, tag := range got {
		if tag.SourceType != metadataTagSourceType {
			t.Errorf("tag %+v: source_type = %q; want %q", tag, tag.SourceType, metadataTagSourceType)
		}
		if tag.SourceName != metadataTagSourceName {
			t.Errorf("tag %+v: source_name = %q; want %q", tag, tag.SourceName, metadataTagSourceName)
		}
		if tag.AssetID != "mcap123" {
			t.Errorf("tag %+v: asset_id = %q; want mcap123", tag, tag.AssetID)
		}
	}
}

// vibecap_tasks can arrive as native array or JSON-encoded string
// (pangzi 2026-07-21 sent the latter — a subtle real-world quirk).
func TestExtractVibecapTasks_ShapeVariants(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"nil", nil, nil},
		{"empty native array", []any{}, []string{}},
		{"native []string", []string{"A", "B"}, []string{"A", "B"}},
		{"native []any of strings", []any{"备餐操作", "台面清洁"}, []string{"备餐操作", "台面清洁"}},
		{"json-encoded string array", `["备餐操作","台面清洁"]`, []string{"备餐操作", "台面清洁"}},
		{"plain unquoted string treated as one value", "备餐操作", []string{"备餐操作"}},
		{"whitespace-only native entries filtered", []any{"", "  ", "A"}, []string{"A"}},
		{"non-string entries in []any skipped", []any{1, "A", true, "B"}, []string{"A", "B"}},
		{"unexpected type returns empty (no error)", 42, nil},
		{"map treated as unexpected type", map[string]any{"k": "v"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractVibecapTasks(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %v, want %v", got, tc.want)
			}
			for i, v := range got {
				if v != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, v, tc.want[i])
				}
			}
		})
	}
}

// City parsing is a documented best-effort: `X市` where X≥2 chars AND not a
// POI-name suffix (超市, 大厦, ...). Reverse-iterate so real cities in the
// tail of a formal Chinese address are picked before POI names near the
// head.
func TestExtractChineseCity_Cases(t *testing.T) {
	cases := []struct {
		name string
		addr string
		want string
	}{
		{"pangzi full address", "合肥新民医院, 合瓦路, 上城, 上城社区, 杏林街道, 庐阳区, 合肥市, 安徽省, 230061, 中国", "合肥市"},
		{"single-segment POI name skipped", "惠乐超市", ""},
		{"multi-segment with POI first, real city later", "惠乐超市, 合肥市, 安徽省", "合肥市"},
		{"POI in the tail is still filtered by suffix list", "合肥市, 某某医院", "合肥市"},
		{"no 市 segment", "合瓦路, 杏林街道, 庐阳区", ""},
		{"empty string", "", ""},
		{"only 市 char, too short", "市", ""},
		{"two-char city too short (哈市 is a colloquial abbrev, ignored)", "南京路, 哈市, 中国", ""},
		{"whitespace-noisy address", "  合瓦路  ,   合肥市   , 安徽省 ", "合肥市"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractChineseCity(tc.addr); got != tc.want {
				t.Errorf("extractChineseCity(%q) = %q; want %q", tc.addr, got, tc.want)
			}
		})
	}
}

// If metadata is nil or missing all 3 known keys, we return empty — no
// spurious tags, no error.
func TestExtractMetadataTags_NoKnownKeys(t *testing.T) {
	cases := []struct {
		name string
		meta map[string]any
	}{
		{"nil metadata", nil},
		{"empty map", map[string]any{}},
		{"only unknown keys", map[string]any{"foo": "bar", "count": 42}},
		{"known keys with unusable values", map[string]any{
			"vibecap_tasks":   nil,
			"source_platform": 123,
			"location":        "just a string not a map",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractMetadataTags("a1", "t1", "p1", tc.meta)
			if len(got) != 0 {
				t.Errorf("expected 0 tags, got %d: %+v", len(got), got)
			}
		})
	}
}

// A single source_platform tag with a valid string still produces one row
// even when the other 2 rules yield nothing.
func TestExtractMetadataTags_PartialMetadata(t *testing.T) {
	meta := map[string]any{
		"source_platform": "grace",
		// no vibecap_tasks, no location
	}
	got := extractMetadataTags("a1", "", "", meta)
	if len(got) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(got))
	}
	if got[0].TagKey != "source" || got[0].TagValue != "grace" {
		t.Errorf("got %+v; want source=grace", got[0])
	}
}

// assertContainsTag scans `tags` for a match on (key, value). Fails the
// test if not found. Handy for assertions that don't care about the
// specific order of the multi-value task rows.
func assertContainsTag(t *testing.T, tags []repository.AssetTagUpsertInput, key, value string) {
	t.Helper()
	for _, tag := range tags {
		if tag.TagKey == key && tag.TagValue == value {
			return
		}
	}
	t.Errorf("expected tag (%s=%s) in %+v", key, value, tags)
}
