package deliveryrules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// AssetSnapshot is the in-memory view used for rule matching.
type AssetSnapshot struct {
	AssetID        string
	AssetType      string
	LifecycleState string
	TagsByKey      map[string][]string // tag_key -> values from all sources
}

func BuildSnapshot(a *models.Asset, tags []*models.AssetTag) AssetSnapshot {
	snap := AssetSnapshot{
		AssetID:        a.AssetID,
		AssetType:      a.AssetType,
		LifecycleState: a.LifecycleState,
		TagsByKey:      map[string][]string{},
	}
	for _, t := range tags {
		snap.TagsByKey[t.TagKey] = append(snap.TagsByKey[t.TagKey], t.TagValue)
	}
	return snap
}

// Matches returns true when the asset satisfies all predicates (rule hit = should block).
func Matches(snap AssetSnapshot, dsl *QueryDSL) (bool, error) {
	if dsl == nil {
		return false, fmt.Errorf("nil query_dsl")
	}
	if len(dsl.AssetTypes) > 0 {
		ok := false
		for _, t := range dsl.AssetTypes {
			if strings.EqualFold(t, snap.AssetType) {
				ok = true
				break
			}
		}
		if !ok {
			return false, nil
		}
	}
	for _, p := range dsl.Where {
		ok, err := evalPredicate(snap, p)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func evalPredicate(snap AssetSnapshot, p Predicate) (bool, error) {
	field := strings.TrimSpace(p.Field)
	op := strings.ToLower(strings.TrimSpace(p.Op))
	switch {
	case strings.HasPrefix(field, "tag."):
		key := strings.TrimPrefix(field, "tag.")
		return evalTag(snap.TagsByKey[key], op, p.Value)
	case field == "asset_type":
		return compareScalar(snap.AssetType, op, p.Value)
	case field == "lifecycle_state":
		return compareScalar(snap.LifecycleState, op, p.Value)
	default:
		return false, fmt.Errorf("unsupported field %q", field)
	}
}

func evalTag(values []string, op string, want any) (bool, error) {
	switch op {
	case "exists":
		return len(values) > 0, nil
	case "not_exists":
		return len(values) == 0, nil
	case "eq", "ne", "in", "contains":
		return compareTagValues(values, op, want)
	default:
		return false, fmt.Errorf("unsupported tag op %q", op)
	}
}

func compareTagValues(values []string, op string, want any) (bool, error) {
	switch op {
	case "eq":
		wantStr := scalarString(want)
		for _, v := range values {
			if v == wantStr {
				return true, nil
			}
		}
		return false, nil
	case "ne":
		wantStr := scalarString(want)
		for _, v := range values {
			if v == wantStr {
				return false, nil
			}
		}
		return len(values) > 0, nil
	case "contains":
		sub := scalarString(want)
		for _, v := range values {
			if strings.Contains(v, sub) {
				return true, nil
			}
		}
		return false, nil
	case "in":
		set, err := stringSet(want)
		if err != nil {
			return false, err
		}
		for _, v := range values {
			if set[v] {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported tag op %q", op)
	}
}

func compareScalar(got, op string, want any) (bool, error) {
	switch op {
	case "eq":
		return got == scalarString(want), nil
	case "ne":
		return got != scalarString(want), nil
	case "in":
		set, err := stringSet(want)
		if err != nil {
			return false, err
		}
		return set[got], nil
	case "exists":
		return got != "", nil
	case "not_exists":
		return got == "", nil
	default:
		return false, fmt.Errorf("unsupported op %q", op)
	}
}

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	default:
		return fmt.Sprint(v)
	}
}

func stringSet(v any) (map[string]bool, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array value for in operator")
	}
	out := make(map[string]bool, len(arr))
	for _, item := range arr {
		out[scalarString(item)] = true
	}
	return out, nil
}
