# CYB-3713 — Wire `mode=keyword` + `q` to existing fulltext dispatch

## Why

`POST /api/v1/queries/run` with `mode: "keyword"` accepts a `q` parameter but **silently ignores it**. Every keyword variant returns the same total as a no-filter structured query (verified on dev 2026-07-21):

| q | total |
|---|---:|
| `备餐操作` | 242 |
| `合肥` | 242 |
| `pangzi` | 242 |
| `xxxxxxxxxxxx_nonexistent` | 242 |
| `""` (empty) | 242 |
| structured no-filter (control) | 242 |

Any frontend that hits this endpoint with a keyword returns the entire asset list. This is a real user-facing correctness bug blocking discovery of the pangzi VibeCap ingest and any future full-text search.

## Root cause (dev-verified)

Two independent plumbing gaps stacked:

1. **`QueryRequest.Q` field does not exist.** `backend/internal/queryir/types.go` `QueryRequest` has `SchemaVersion / Mode / Scope / Select / Where / Sort / Page / Facets / Debug` but no `Q string`. The JSON `"q"` on the wire is silently dropped by `json.Unmarshal` (unknown field).
2. **`shouldUseESRecall` does not branch on `keyword`.** `backend/internal/queryplan/planner.go:172` only forces ES recall when `req.Mode == "semantic" || req.Mode == "similar"` or when `hasFulltextPredicate(req.Where) == true`. With no `_fulltext` predicate in the where tree, keyword requests fall through to PG filter with empty WHERE → return everything.

Downstream is **already wired**:
- ES side: `backend/internal/elasticsearch/client.go:363` `buildSearchModeQuery(mode, query)` has a default branch (matched by `"keyword"` and anything else) that emits a non-fuzzy multi-field bool query (`asset_id term`, `asset_type term`, `notes match`, `owner.text match`, `reviewer.text match`).
- PG side: `backend/internal/queryexec/postgres/expr.go:120` `buildFulltextClause` emits `ILIKE %pattern%` across the same 6 fields plus `asset_tags.notes`. Ops accepted: `ilike`, `like`, `eq`.

So the fix is to close the plumbing gap, not to redesign anything.

## What Changes

**1. `queryir/types.go`** — Add `Q` field to `QueryRequest`:
```go
Q string `json:"q,omitempty"`
```
Non-breaking (new optional field).

**2. `queryir/compile.go` — `Normalize()`** — After trimming Mode + Where, when `mode ∈ {"keyword","semantic","similar"}` and `Q != ""`, synthesize a fulltext predicate and merge into `Where`:
```go
if isFulltextMode(normalized.Mode) && normalized.Q != "" {
    ft := &QueryExpr{Pred: &QueryPredicate{Field: "_fulltext", Op: "ilike", Value: normalized.Q}}
    if normalized.Where == nil {
        normalized.Where = ft
    } else {
        normalized.Where = &QueryExpr{And: []QueryExpr{*normalized.Where, *ft}}
    }
}
```

**Why include `semantic` and `similar`**: they had the same silent-drop bug for `q` (the switch in `shouldUseESRecall` triggered ES recall, but no `q` field existed so ES only saw the `where` tree — for a request with `mode=semantic, q="X"` and no where, ES got `match_all`). Fixing all three in one place is symmetric and cheap.

**3. `queryplan/planner.go`** — No change. `hasFulltextPredicate(req.Where)` already returns `true` once step 2 injects the predicate, so `shouldUseESRecall` fires ES recall automatically. Adding an explicit `case "keyword"` would be redundant defense; skip to minimize surface.

**4. Warning surface** — When ES is unhealthy or `useElasticsearch == false`, the existing PG fallback picks up `_fulltext ilike` (already supported by `buildFulltextClause`). Emit a `debug_plan.warnings` entry `"keyword search degraded to postgres ilike (elasticsearch unavailable)"` in the handler when this fallback path fires, so the client knows results may be less relevant than an ES full-text match.

**5. Tests**
- `queryir/compile_test.go`: Normalize with `mode=keyword` + `q="foo"` injects `_fulltext ilike foo`; with `where` set, wraps in AND.
- `queryir/compile_test.go`: Normalize with `mode=structured` + `q="foo"` does **not** inject (only for fulltext modes).
- `queryplan/planner_test.go`: Plan with `q="foo" mode="keyword"` sets `plan.UseESRecall = true`.
- `handlers/query/handler_test.go`: End-to-end handler flow — request with `mode=keyword` + `q="备餐操作"` produces ES body containing `buildSearchModeQuery` multi-field bool.

## Impact

**Runtime**
- `POST /queries/run` with `mode=keyword` + `q` now returns filtered results instead of the full list. Users who previously saw an unfiltered list will see a smaller, correctly filtered set — this is a behavior change but is exactly what the API contract promised.
- No new endpoint, no schema change, no migration.
- Field coverage: fulltext search hits `asset_id`, `asset_type`, `owner`, `reviewer`, `notes` tag, and `mcap_file_id` (PG side has one more via `asset_tags` sub-query). Long-tail `mcap_files.metadata` JSONB is **not covered by this ticket** — that requires either metadata-flatten (CYB-3715) or a separate ES mapping expansion.

**API contract sync**
- `api/openapi.yaml` — add `q` optional field to `QueryRequest` schema.
- `docs/review/api-guide.md` — add one paragraph on keyword mode.
- `scripts/api-guide-smoke.sh` — add assertion that `q=nonexistent-xxx` returns 0 hits.

**Off-limits**
- None. This PR does not touch `middleware/auth*`, `outbox/`, `migrations/`, `.env*`, or `schemas/pg-phase0.sql`.

**Compat**
- Fully backward-compatible. `q` is optional; requests without it behave exactly as today. `mode=structured` with `q` is a no-op (guarded by `isFulltextMode`).

## Rollback

`git revert` the merge commit. No data changes. The pre-fix (silently ignored `q`) behavior returns.
