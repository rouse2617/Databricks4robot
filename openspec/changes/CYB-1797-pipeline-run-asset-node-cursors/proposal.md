# Proposal — CYB-1797 Pipeline run asset-node cursors

## Why
Pipeline run asset-node pagination currently filters by row ID while ordering by business fields such as asset, node, cost, duration, or status. This can skip or duplicate rows across pages, and the current next cursor is derived from the overflow row, which can skip that row on the next request.

## What Changes
### Modified Capabilities
- Pipeline run asset-node listing returns stable pages that do not skip or duplicate rows.
- Existing sort modes remain available while each mode uses cursor semantics aligned with its ordering.
- Invalid or stale cursor values produce a clear client error instead of silently returning incorrect pages.

## Impact
- `backend/internal/handlers/pipeline` — accepts and forwards asset-node list cursor/order parameters.
- `backend/internal/postgres/pipeline_repo.go` — builds cursor-aware asset-node queries.
- `backend/internal/models/pipeline.go` — may need cursor-related response/request model support.
- `backend/internal/postgres/pipeline_repo*_test.go` or adjacent tests — regression coverage for all asset-node sort modes.
- `api/openapi.yaml` and `docs/review/api-guide.md` — document cursor semantics if response shape or cursor encoding changes.

## Scope
### In
- Stable cursor pagination for pipeline run asset nodes.
- Default, cost, duration, and status ordering.
- Regression tests that prove no missing or duplicate items across pages.

### Out
- New UI behavior beyond consuming the existing response contract.
- Changing pipeline run node ingestion semantics.
- Database schema changes.

## Success Criteria
- Fetching all pages for a run returns the same item set as fetching without a cursor.
- No item appears more than once across paginated responses.
- `nextCursor` represents the last returned row's sort position for the active order.
- Invalid cursor payloads return `400 INVALID_ARGUMENT`.
