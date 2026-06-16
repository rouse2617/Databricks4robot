# Batch attempt history and observability

## Problem

Batch operators cannot see per-asset rerun history, node-level drill-down for Running/Pending assets, or node progress on the subtask list without opening each workflow detail.

## Scope

- Rerun creates a new `pipeline_run` per attempt; historical runs and node snapshots are preserved.
- `GET /backfill/:id/items/:itemId/attempts` lists all attempts for a logical subtask.
- Batch pipeline-run list includes `nodeProgress` for current attempts.
- Node overview supports drill-down for Running and Pending counts.
- `POST /backfill/validate-assets` previews registered vs unknown asset IDs.

## Out of scope

- Full asset×node matrix UI
- Cross-batch asset history on asset detail page
- DB migration for attempt_no column (derive from run created_at)
