package elasticsearch

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
)

func TestWrapCurrentRevisionOnlyQuery(t *testing.T) {
	req := queryir.QueryRequest{Scope: queryir.QueryScope{Resource: queryir.ResourceAssets}}
	wrapped := wrapCurrentRevisionOnlyQuery(req, map[string]any{"match_all": map[string]any{}})
	raw, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(raw)
	for _, part := range []string{"is_current", "logical_asset_id", "minimum_should_match"} {
		if !strings.Contains(s, part) {
			t.Fatalf("expected %q in query: %s", part, s)
		}
	}

	unchanged := wrapCurrentRevisionOnlyQuery(queryir.QueryRequest{
		Scope: queryir.QueryScope{Resource: queryir.ResourceAssets, IncludeHistory: true},
	}, map[string]any{"match_all": map[string]any{}})
	if _, ok := unchanged["match_all"]; !ok {
		t.Fatalf("expected match_all preserved, got %#v", unchanged)
	}
}
