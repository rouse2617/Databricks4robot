// CYB-3384: FacetExecutor runs facet aggregations against Postgres when the
// planner routes the facet phase to PG (either because DBK_FACET_ENGINE=pg or
// because SyncHealthCache saw a non-zero PG↔ES gap). Behaviour mirrors the
// existing ES facet path — same WHERE clause, same output shape — so the
// handler dispatch can swap engines without touching serialization.
package postgres

import (
	"context"
	"strings"

	pgrepo "github.com/CyberOrigin2077/cyber-databrew/internal/postgres"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryir"
	"github.com/CyberOrigin2077/cyber-databrew/internal/queryplan"
)

// FacetSource is the minimal interface FacetExecutor needs to compute
// aggregation buckets on assets. Implemented by *postgres.AssetRepo. Kept as
// an interface so tests and alternative back-ends can supply fakes.
type FacetSource interface {
	FacetCounts(ctx context.Context, field, whereSQL string, whereArgs []interface{}, size int) ([]pgrepo.AssetFacetBucket, error)
}

// FacetExecutor runs the aggregation stage for a compiled query when the
// planner has selected PG for facets.
type FacetExecutor struct {
	source FacetSource
}

// NewFacetExecutor constructs an executor. A nil source is treated as
// disabled — Execute returns nil so callers can wire the executor
// unconditionally and only exercise it when the planner asks for PG facets.
func NewFacetExecutor(source FacetSource) *FacetExecutor {
	return &FacetExecutor{source: source}
}

// Execute runs one COUNT(*) GROUP BY per requested facet field, restricted to
// the same WHERE clause the list phase uses. Fields outside
// queryplan.PGSupportedFacetFields are dropped and reported via the returned
// warnings slice. Errors on individual fields are surfaced as warnings too —
// one bad bucket must not fail the whole request.
func (e *FacetExecutor) Execute(ctx context.Context, compiled *queryir.CompiledQuery) (map[string][]queryir.FacetBucket, []string, error) {
	if e == nil || e.source == nil {
		return nil, nil, nil
	}
	if compiled == nil || len(compiled.NormalizedQuery.Facets) == 0 {
		return nil, nil, nil
	}
	exec := &Executor{}
	whereSQL, args, _, empty, err := exec.buildListParams(compiled)
	if err != nil {
		return nil, nil, err
	}
	if empty {
		// Candidate list was empty (ES recall returned nothing) — no rows to
		// aggregate, return empty buckets for every requested field so the UI
		// sees an authoritative "0" instead of stale state.
		return map[string][]queryir.FacetBucket{}, nil, nil
	}

	out := make(map[string][]queryir.FacetBucket, len(compiled.NormalizedQuery.Facets))
	var warnings []string
	for _, facet := range compiled.NormalizedQuery.Facets {
		field := strings.TrimSpace(facet.Field)
		if !queryplan.PGSupportedFacetFields[field] {
			warnings = append(warnings, "pg facet fallback does not support field: "+facet.Field)
			continue
		}
		size := facet.Size
		if size <= 0 {
			size = 20
		}
		rawBuckets, ferr := e.source.FacetCounts(ctx, field, whereSQL, args, size)
		if ferr != nil {
			warnings = append(warnings, "pg facet fallback error for "+facet.Field+": "+ferr.Error())
			continue
		}
		buckets := make([]queryir.FacetBucket, 0, len(rawBuckets))
		for _, b := range rawBuckets {
			buckets = append(buckets, queryir.FacetBucket{Value: b.Value, Count: b.Count})
		}
		out[field] = buckets
	}
	return out, warnings, nil
}
