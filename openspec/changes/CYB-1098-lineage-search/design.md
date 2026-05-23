# Design — CYB-1098

## Architecture Context
- **Constraints**: `asset_relations` is the existing PostgreSQL source of complex lineage. `backend/migrations/`, `backend/internal/middleware/auth*`, and `backend/internal/outbox/` are off-limits unless explicitly approved.
- **Goals**: Expose a read-only compliance lineage search API with bounded recursive traversal, predictable validation, and contract documentation.
- **Non-Goals**: Add new tables, change relation write semantics, implement frontend UI, or add SDK parity in this issue.

## Affected Modules
- `backend/internal/handlers/audit/handler.go` — validate lineage query filters, run recursive CTE traversal, and shape the response.
- `backend/routes/routes.go` — ensure audit lineage route registration remains guarded by handler availability.
- `api/openapi.yaml` — publish endpoint contract, query parameters, response schema, and errors.
- `docs/review/api-guide.md` — add curl examples for success, empty results, and validation failures.
- `scripts/api-guide-smoke.sh` — add dev smoke checks for success and validation error.

## Architecture Decisions

### Decision 1: Source lineage search from `asset_relations`
- **Approach**: Traverse `asset_relations` directly using `parent_asset_id` and `child_asset_id`.
- **Alternative**: Reconstruct lineage from `assets.parent_asset_id` or historical `asset_events`.
- **Rationale**: CYB-1098 asks for dependency relationships; `asset_relations` is the schema dedicated to multi-parent, merge, split, sample, and derived lineage.
- **Trade-off**: The endpoint returns structural lineage currently recorded in PostgreSQL, not an event-sourced reconstruction of past relation states.
- **Rollback**: Remove the `/api/v1/audit/lineage-search` route and OpenAPI/api-guide entries; no data rollback is needed.

### Decision 2: Use recursive CTEs with explicit cycle protection
- **Approach**: Use PostgreSQL `WITH RECURSIVE` traversal with a visited path and `depth < max_depth` guard.
- **Alternative**: Load relations into Go and traverse in memory.
- **Rationale**: Recursive CTE keeps traversal close to the indexed relation table and avoids moving a large relation graph into the handler.
- **Trade-off**: SQL is more complex than an in-memory walk, so focused tests should cover cycles, direction, and depth behavior.

### Decision 3: Default to dependency/structural relation types
- **Approach**: Traverse dependency/structural `asset_relations.relation_type` values by default, starting with `split_from`, `contains`, `derived_from`, `merged_from`, and `sampled_from`; allow a validated `relation_types` query parameter for narrower searches.
- **Alternative**: Traverse every relation type, including versioning-only edges such as `revision_of`.
- **Rationale**: Compliance lineage should answer asset dependency and flow questions. Including every future edge type by default risks mixing version history or display-only relationships into dependency results.
- **Trade-off**: Clients that need version lineage must request it explicitly or use a dedicated version/provenance endpoint.

### Decision 4: Keep the API bounded and read-only
- **Approach**: Require `asset_id`, default direction to `both`, default depth to a conservative value, and cap depth at a documented maximum.
- **Alternative**: Allow unbounded traversal or silently ignore invalid filters.
- **Rationale**: Compliance searches can start from high-degree assets; hard bounds prevent expensive recursive scans and make client behavior predictable.

## Data Flow

```text
Client -> GET /api/v1/audit/lineage-search?asset_id=...&direction=...&depth=...
  -> Static token auth
  -> audit handler validates query filters
  -> PostgreSQL recursive CTE over asset_relations
  -> JSON response with asset_id, direction, depth, nodes, and count
```

## Data Model Changes
- None.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cyclic relation data can recurse repeatedly | Slow or duplicate results | Track visited asset ids inside the recursive CTE and cap depth |
| High-degree assets can return many related nodes | Large responses or slow queries | Keep depth capped, order deterministically, and smoke-test representative dev data |
| Relation types may be heterogeneous | Clients may misread lineage semantics | Default to dependency relation types, allow explicit filtering, and return relation type metadata |
