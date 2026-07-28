# Spec delta — query

## ADDED

### Requirement: `mode=keyword` with a non-empty `q` performs a full-text search

`POST /api/v1/queries/run` accepts an optional top-level `q` string field. When `mode ∈ {"keyword","semantic","similar"}` and `q` is non-empty, the server MUST perform a full-text search across a curated field set (`asset_id`, `asset_type`, `owner`, `reviewer`, `mcap_file_id`, and `notes` tags on the asset) — implemented by synthesizing a `_fulltext ilike q` predicate and reusing the existing dispatch (ES `buildSearchModeQuery` when Elasticsearch is healthy; PG `buildFulltextClause` on fallback).

The `q` field is IGNORED for `mode=structured` (or any mode not in the set above).

#### Scenario: keyword mode with matching q returns filtered results

- **Given** at least one asset whose `owner="pangzi-consumer"` and whose `mcap_files.metadata.vibecap_tasks` includes `"备餐操作"`
- **When** the client sends `POST /queries/run` with body `{schema_version:"v1", mode:"keyword", q:"pangzi", scope:{resource:"assets"}, page:{page:1,page_size:5}}`
- **Then** the response includes `items` matching the term and `total` reflects the filtered count (strictly smaller than the unfiltered total), and `debug_plan.steps[0].engine == "elasticsearch"` when ES is healthy.

#### Scenario: keyword mode with non-matching q returns zero results

- **Given** no asset matches the term `xxxxxxxxxxxx_nonexistent` across the fulltext field set
- **When** the client sends `POST /queries/run` with `mode:"keyword", q:"xxxxxxxxxxxx_nonexistent"`
- **Then** the response has `total == 0` and `items == []`. This is the invariant the pre-fix behavior violated (returned full list).

#### Scenario: keyword mode with empty q keeps no-filter behavior

- **Given** the request has `mode:"keyword"` and `q:""` (or `q` omitted)
- **When** processed
- **Then** no fulltext predicate is injected; the query behaves as `mode:"structured"` with the request's existing `where` (or no filter if `where` is nil). `total` equals the unfiltered set. This preserves backward compatibility for callers who pass `mode=keyword` today without a `q`.

#### Scenario: keyword `q` combines with existing where via AND

- **Given** the request has `mode:"keyword", q:"pangzi", where:{pred:{field:"owner", op:"eq", value:"pangzi-consumer"}}`
- **When** processed
- **Then** the effective `where` is the AND of the caller-provided predicate and the synthesized `_fulltext ilike pangzi` predicate; results must satisfy both.

#### Scenario: structured mode ignores q

- **Given** the request has `mode:"structured", q:"pangzi"` (no where)
- **When** processed
- **Then** the `q` field is dropped; the request behaves as an unfiltered structured query (`total` = full asset count). This scopes the behavior change to fulltext modes only.

## MODIFIED

### Requirement: Elasticsearch degradation surfaces a warning

Previously, when the planner routed a keyword-like request to Postgres because ES was unavailable, the degradation was silent. When a synthesized `_fulltext` predicate is present and the executed path is Postgres (either forced by `DBK_QUERY_ENGINE=pg`, the `useElasticsearch=false` build flag, or an ES runtime error), the response MUST include `warnings: ["keyword search degraded to postgres ilike (elasticsearch unavailable)"]` so the client can inform the user that results may be less relevant than an ES match.

#### Scenario: ES unavailable, PG fallback fires

- **Given** ES is unavailable AND the request has `mode:"keyword", q:"foo"`
- **When** processed
- **Then** `debug_plan.steps[0].engine == "postgres"`, results come from `buildFulltextClause` (`ILIKE %foo%` across the 6 PG fields + notes tag), AND `warnings` contains the degradation string above.
