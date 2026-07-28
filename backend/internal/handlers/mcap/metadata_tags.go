// CYB-3797: extract known mcap-file metadata fields into asset tags at ingest
// time so newly uploaded mcap arrive already tagged (task / source / city).
// See openspec/changes/CYB-3797-mcap-auto-tag/ for the full rule set + rationale.
package mcap

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

const (
	// metadataTagSourceType marks tags that came from mcap-ingest auto-extract,
	// distinct from human/algo/compliance labels. Lets us audit and (if the
	// rule set ever changes) selectively re-derive from `mcap_files.metadata`.
	metadataTagSourceType = "system"
	metadataTagSourceName = "mcap_ingest"
)

// poiSuffixesForCityFilter is the set of Chinese POI-name endings that also
// end in 市 but are NOT city names. Keeps `惠乐超市` from being tagged as a
// city while still letting `合肥市` through.
var poiSuffixesForCityFilter = []string{
	"超市", "大厦", "商店", "商场", "广场", "医院", "公司", "工厂", "学校", "大学",
}

// extractMetadataTags reads three specific fields off the mcap metadata JSONB
// and returns the corresponding tag upsert inputs. Unknown keys are ignored;
// value-shape mismatches skip that specific tag with a WARN log line and
// return the remaining tags — never returns an error, so the mcap create
// path can call this unconditionally.
//
// See openspec/changes/CYB-3797-mcap-auto-tag/specs/tags/spec.md for the
// canonical rule table.
func extractMetadataTags(assetID, tenantID, projectID string, meta map[string]any) []repository.AssetTagUpsertInput {
	if meta == nil {
		return nil
	}
	var out []repository.AssetTagUpsertInput

	// 1) vibecap_tasks[] → one `task:*` tag per entry.
	for _, task := range extractVibecapTasks(meta["vibecap_tasks"]) {
		out = append(out, repository.AssetTagUpsertInput{
			AssetID:    assetID,
			TagKey:     "task",
			TagValue:   task,
			SourceType: metadataTagSourceType,
			SourceName: metadataTagSourceName,
			TenantID:   tenantID,
			ProjectID:  projectID,
		})
	}

	// 2) source_platform → single `source` tag.
	if src := extractStringField(meta["source_platform"]); src != "" {
		out = append(out, repository.AssetTagUpsertInput{
			AssetID:    assetID,
			TagKey:     "source",
			TagValue:   src,
			SourceType: metadataTagSourceType,
			SourceName: metadataTagSourceName,
			TenantID:   tenantID,
			ProjectID:  projectID,
		})
	}

	// 3) location.address → parse Chinese city, single `city` tag.
	if city := extractChineseCity(getNestedString(meta, "location", "address")); city != "" {
		out = append(out, repository.AssetTagUpsertInput{
			AssetID:    assetID,
			TagKey:     "city",
			TagValue:   city,
			SourceType: metadataTagSourceType,
			SourceName: metadataTagSourceName,
			TenantID:   tenantID,
			ProjectID:  projectID,
		})
	}

	return out
}

// extractVibecapTasks accepts nil, []string, []any, or a JSON-encoded string
// that parses to any of the above. Anything else logs a warning and returns
// an empty slice — the mcap create is not aborted for a producer-side shape
// glitch.
func extractVibecapTasks(v any) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []string:
		return trimNonEmpty(x)
	case []any:
		out := make([]string, 0, len(x))
		for _, elem := range x {
			if s, ok := elem.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		var parsed []string
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			return trimNonEmpty(parsed)
		}
		// Not JSON — treat the raw string as a single task value (some
		// producers may send `"备餐操作"` unquoted).
		return []string{s}
	default:
		log.Printf("mcap metadata tag extract skip: vibecap_tasks type unexpected %T", v)
		return nil
	}
}

func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// extractStringField coerces a metadata value to a trimmed string, or
// returns "" for nil / non-string types.
func extractStringField(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	log.Printf("mcap metadata tag extract skip: string field type unexpected %T", v)
	return ""
}

// getNestedString walks a nested map using the given path and returns the
// terminal value as a string. Returns "" if any hop is missing or non-map.
func getNestedString(root map[string]any, path ...string) string {
	if len(path) == 0 {
		return ""
	}
	cur := any(root)
	for i, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		next, exists := m[key]
		if !exists {
			return ""
		}
		if i == len(path)-1 {
			if s, ok := next.(string); ok {
				return strings.TrimSpace(s)
			}
			return ""
		}
		cur = next
	}
	return ""
}

// extractChineseCity parses a comma-separated Chinese address and returns
// the first segment ending in 市 that (a) is at least 3 characters,
// (b) does not end in a common POI suffix (超市, 大厦, ...), scanning
// segments from the end of the address (real cities live near the tail
// of Chinese formal addresses, right before province / postcode / country).
// Returns "" when no segment qualifies — typical for single-segment
// address strings that are just a POI name.
func extractChineseCity(addr string) string {
	if addr == "" {
		return ""
	}
	segs := strings.Split(addr, ",")
	if len(segs) < 2 {
		return ""
	}
	for i := len(segs) - 1; i >= 0; i-- {
		seg := strings.TrimSpace(segs[i])
		if !strings.HasSuffix(seg, "市") {
			continue
		}
		// Count runes, not bytes — Chinese chars are multi-byte in UTF-8.
		if runeLen(seg) < 3 {
			continue
		}
		isPOI := false
		for _, suffix := range poiSuffixesForCityFilter {
			if strings.HasSuffix(seg, suffix) {
				isPOI = true
				break
			}
		}
		if isPOI {
			continue
		}
		return seg
	}
	return ""
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
