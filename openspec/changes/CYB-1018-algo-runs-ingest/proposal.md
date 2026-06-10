# Proposal — CYB-1018

## Why

Workers already stamp `run_id` on `asset_algo_latest`, `actions`, and eval rows, but there is no `algo_runs` anchor or ingest API. Operators cannot answer “what ran, with what params, on which assets?” without scraping JSONB.

## What Changes

- New `algo_runs` table (16-char `run_id`, status machine, reproducibility fields)
- `POST /api/v1/algo-runs`, `/:run_id/start`, `/:run_id/finish`, `GET /:run_id`
- FK from `asset_algo_latest`, `actions`, `asset_eval_results`, `asset_tags` → `algo_runs` (orphan `run_id` nulled on migrate)
- `asset_events` type `algo_run_applied` when per-asset algo finish carries a registered `run_id`
- Per-asset `start`/`finish` rejects unknown `run_id` when non-empty

## Scope

- **In scope**: DDL 031, CRUD lifecycle API, FK wiring, `algo_run_applied`, OpenAPI/api-guide/smoke, unit tests
- **Out of scope**: `GET /algo-runs` list, `affected-assets`, cancel, SDK, UI “来自 run R001” (follow-up)

## Success Criteria

- [ ] `POST /algo-runs` creates `pending` row; start → `running`; finish → terminal state with stats
- [ ] `GET /algo-runs/{id}` returns full row
- [ ] `asset_algo_latest.run_id` FK enforced for new writes
- [ ] `FinishAlgo` with valid `run_id` appends `algo_run_applied` on the asset

## Design

See [`design.md`](design.md) and [`docs/review/unified-asset-catalog/design/algo-runs.md`](../../../docs/review/unified-asset-catalog/design/algo-runs.md).
