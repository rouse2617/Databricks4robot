# CYB-3715c: project 7 flatten fields into ES doc + facet whitelist

## Context

CYB-3715 (PR #526 + #527) landed the top-level filter/facet plumbing on the PG side. Dev smoke proves `mode=structured` filter works (`camera_model=CyberCap2` → 1008 hits). But `mode=keyword` (ES recall + PG refine) returns **0 hits** for the same filter because the ES doc has these fields only under the nested `mcap.<col>` object (CYB-3297 Phase C), not at the top level where the filter/facet compiler addresses them.

Dev evidence:
```
mode=structured  camera_model=CyberCap2 → total=1008  plan=[postgres.filter]
mode=keyword q=CyberCap2 + camera_model=CyberCap2 → total=0  plan=[elasticsearch.recall, postgres.refine]
```

## Change

1. `internal/searchindex/builder.go` — emit 7 flatten fields as top-level string properties on the ES doc when the Asset struct field is non-empty. Skip when empty so ES dynamic mapping does not create empty-string buckets for every asset (matches the existing `mcap.<col>` pattern).
2. `internal/elasticsearch/query_ir.go` — extend the ES facet whitelist to accept the 4 facet-able flatten fields (`camera_model`, `data_source`, `collection_method`, `source_platform`). UUID fields stay filter-only (matches PG `PGSupportedFacetFields`).
3. `internal/searchindex/builder_test.go` — two tests: emits-when-populated and omits-when-empty.

## Deploy

- Backend rollout picks up the new projection immediately for **new writes** (persistence path).
- For existing dev docs (~800 pangzi raw_mcap assets), post-deploy trigger `POST /api/v1/admin/search/reindex` to rebuild the index. Non-blocking: PG-only path already covers filter/facet via CYB-3384 planner fallback until reindex finishes.

## Non-goals

- ES mapping template — deploy uses dynamic mapping so keyword sub-fields materialize on first indexed value. No explicit mapping change needed for the initial cut.
- Frontend chips (CYB-3715d) — separate PR.
- Ticket for C (metadata fulltext) — separate work, still blocked on user clarification.
