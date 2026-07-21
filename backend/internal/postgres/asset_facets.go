// CYB-3384: AssetRepo.FacetCounts computes strong-consistency facet buckets
// directly from PostgreSQL so the planner can fall back off Elasticsearch when
// pg_es_gap > 0. The set of supported fields is intentionally narrow — the
// first iteration covers the direct columns and jsonb keys that dominate the
// Assets page facet sidebar; nested-join fields (mcap.*, tag.*) are rejected
// and the caller drops them from the response.
package postgres

import (
	"context"
	"fmt"
	"strings"
)

// AssetFacetBucket is one row of a terms-style aggregation.
type AssetFacetBucket struct {
	Value string
	Count int64
}

// facetSelectExpr maps a facet field name (as accepted by the query IR) to the
// SQL expression to GROUP BY. We keep this explicit rather than deriving from
// column metadata so an unrecognized field is rejected upstream (before it
// reaches Postgres) and never becomes an injection vector.
var facetSelectExpr = map[string]string{
	"asset_type":        `asset_type`,
	"lifecycle_state":   `lifecycle_state`,
	"owner":             `owner`,
	"env":               `metadata->>'env'`,
	// CYB-3715 mirror columns — direct column names, no JSONB path. Uuid
	// fields are omitted (see planner.PGSupportedFacetFields for the
	// filter-only rationale).
	"camera_model":      `camera_model`,
	"data_source":       `data_source`,
	"collection_method": `collection_method`,
	"source_platform":   `source_platform`,
}

// FacetCounts returns terms-aggregation buckets on the given field, applied
// to assets matching the caller's whereSQL/args. The base filter
// (`is_deleted = FALSE`) is applied automatically to match the semantics of
// ListWithFilters. size caps the number of buckets returned; when 0 or
// negative it defaults to 20 (aligned with the ES `terms.size` default).
//
// NULL values on the target field are grouped under the empty string "" and
// dropped so the response reflects only observed values, matching ES `terms`
// aggregation behavior on missing fields.
func (r *AssetRepo) FacetCounts(ctx context.Context, field, whereSQL string, whereArgs []interface{}, size int) ([]AssetFacetBucket, error) {
	expr, ok := facetSelectExpr[strings.TrimSpace(field)]
	if !ok {
		return nil, fmt.Errorf("postgres AssetRepo.FacetCounts: unsupported field %q", field)
	}
	if size <= 0 {
		size = 20
	}
	baseWhere := assetListBaseWhere(whereSQL)
	limitParam := len(whereArgs) + 1
	sqlText := fmt.Sprintf(
		`SELECT COALESCE(%s, '') AS bucket, COUNT(*)::bigint AS c
		 FROM assets
		 WHERE %s
		 GROUP BY bucket
		 ORDER BY c DESC, bucket ASC
		 LIMIT $%d`,
		expr, baseWhere, limitParam,
	)
	args := append(append([]interface{}{}, whereArgs...), size)
	rows, err := r.c.db.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres AssetRepo.FacetCounts %q: %w", field, err)
	}
	defer rows.Close()

	var out []AssetFacetBucket
	for rows.Next() {
		var b AssetFacetBucket
		if err := rows.Scan(&b.Value, &b.Count); err != nil {
			return nil, fmt.Errorf("postgres AssetRepo.FacetCounts %q scan: %w", field, err)
		}
		if b.Value == "" {
			continue
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres AssetRepo.FacetCounts %q rows: %w", field, err)
	}
	return out, nil
}
