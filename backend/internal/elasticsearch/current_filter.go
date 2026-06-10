package elasticsearch

import "github.com/CyberOrigin2077/cyber-databrew/internal/queryir"

// wrapCurrentRevisionOnlyQuery hides non-current revisions unless include_history is set.
// Legacy docs without logical_asset_id remain visible (pre-CYB-1013 rows).
func wrapCurrentRevisionOnlyQuery(req queryir.QueryRequest, query map[string]any) map[string]any {
	if !queryir.ApplyCurrentOnlyFilter(req) {
		return query
	}
	currentClause := map[string]any{
		"bool": map[string]any{
			"should": []map[string]any{
				{"term": map[string]any{"is_current": true}},
				{"bool": map[string]any{
					"must_not": []map[string]any{
						{"exists": map[string]any{"field": "logical_asset_id"}},
					},
				}},
			},
			"minimum_should_match": 1,
		},
	}
	return map[string]any{
		"bool": map[string]any{
			"must": []map[string]any{query, currentClause},
		},
	}
}
