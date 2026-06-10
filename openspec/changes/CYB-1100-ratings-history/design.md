# Design — CYB-1100

## Architecture Context
- **Version source**: logical asset revisions live in `assets.logical_asset_id`, `assets.revision`, and `assets.is_current`, with `logical_assets` coordinating the current revision.
- **Rating source for this endpoint**: `asset_metrics` rows whose `metric_key` starts with `rating.`. Product docs define these as per-asset quality/rating signals that follow a concrete revision `asset_id`.
- **Adjacent algorithm state model**: `asset_algo_latest` stores per-asset algorithm status/result projection. It may explain how a revision was produced, but it is not the ratings source for CYB-1100.
- **Constraints**: no schema migration, no auth middleware changes, no outbox changes, and no runtime implementation before OpenSpec approval.

## Affected Modules
- `backend/internal/handlers/asset/handler.go` — validate logical asset id, fetch revision rows and `rating.*` metrics, shape response.
- `backend/routes/routes.go` — existing `/api/v1/logical-assets/:id/ratings-history` route registration.
- `api/openapi.yaml` — add path and schemas for `LogicalAssetRatingsHistoryResponse`.
- `docs/review/api-guide.md` — add curl example, response example, and error notes near logical asset provenance/current docs.
- `scripts/api-guide-smoke.sh` — add one success smoke and one validation/not-found smoke for dev.

## Response Contract

Return one item per non-deleted revision. Each item contains revision metadata and a `ratings` array. A revision with no `asset_metrics` rows matching `metric_key LIKE 'rating.%'` returns `ratings: []`.

```json
{
  "logical_asset_id": "aaaaaaaa",
  "items": [
    {
      "asset_id": "aaaaaaaa",
      "revision": 1,
      "is_current": false,
      "lifecycle_state": "ready",
      "created_at": "2026-05-20T10:00:00Z",
      "ratings": [
        {
          "metric_key": "rating.quality_score",
          "metric_type": "float",
          "metric_value": 4.5,
          "metric_unit": "stars",
          "source_type": "human",
          "source_name": "alg_zhang",
          "eval_name": "manual_rating",
          "eval_version": "v1",
          "target_type": "asset",
          "target_id": "",
          "run_id": "R001abcDEF234ghi",
          "recorded_at": "2026-05-20T10:15:00Z"
        }
      ]
    }
  ],
  "count": 1
}
```

## Architecture Decisions

### Decision 1: Use logical asset family as the boundary
- **Approach**: Query non-deleted `assets` rows for the requested `logical_asset_id` and attach rating rows per revision.
- **Alternative**: Accept any revision `asset_id` and infer its logical family.
- **Rationale**: The Linear endpoint is explicitly `/logical-assets/{id}/ratings-history`, and `GET /logical-assets/{id}/current` already uses logical id semantics.
- **Trade-off**: Clients that only have a revision id must call provenance/current first or use the asset detail response to get `logical_asset_id`.

### Decision 2: Preserve revisions with no rating rows
- **Approach**: Build revision items first, then attach zero or more rating records.
- **Alternative**: Inner join rating rows and omit unrated revisions.
- **Rationale**: Missing rows are meaningful trend data: a newly promoted revision may not have algorithm results yet.
- **Trade-off**: Clients need to render empty `ratings` arrays.

### Decision 3: Treat missing logical asset separately from empty ratings
- **Approach**: If no non-deleted revision rows exist for the logical id, return `404 ASSET_NOT_FOUND`; if revision rows exist but no rating rows, return `200` with empty arrays.
- **Alternative**: Always return `200` with `items: []`.
- **Rationale**: `items: []` cannot distinguish typo/deleted logical asset from a valid family that has not been rated.

### Decision 4: Source ratings from `asset_metrics` `rating.*`
- **Approach**: Ratings in this checkpoint mean queryable metric rows where `metric_key` is in the `rating.*` namespace.
- **Alternative**: Use `asset_algo_latest` result fields because the current skeleton does that.
- **Rationale**: Product docs define `asset_metrics` `rating.*` as the manual/quality rating model and explicitly point cross-version comparisons to this endpoint. `asset_algo_latest` is algorithm state, not rating history.
- **Trade-off**: Algorithm result tags/scores remain outside the response unless a later contract adds them as a separate source type.

## Data Flow

```text
Client -> GET /api/v1/logical-assets/{id}/ratings-history
  -> Static token auth
  -> asset handler validates logical_asset_id
  -> PostgreSQL fetches non-deleted revisions ordered by revision ASC
  -> PostgreSQL fetches asset_metrics rows with metric_key LIKE 'rating.%' for those revision asset_ids
  -> JSON response groups rating rows under each revision
```

## Data Model Changes
- None.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Existing route skeleton returns flat `asset_algo_latest` rows | Contract drift between handler and OpenAPI | Replace skeleton behavior and tests should assert grouped `items[].ratings[]` from `asset_metrics` |
| `asset_metrics` primary key may keep only the latest value for a metric identity | Endpoint shows current rating rows per revision, not every overwritten historical value | Document source semantics and keep raw rerun/review history in eval/events APIs |
| Older assets may have blank or missing `logical_asset_id` | Valid legacy rows may not be addressable by logical endpoint | Return `404`; use asset provenance/detail to identify versioned families |
| Algorithm result fields also look rating-like | Product may expect them in the same response | Keep CYB-1100 to documented `rating.*` metric rows; add algorithm state only through a later contract |
