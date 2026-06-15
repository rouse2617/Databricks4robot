# Proposal — CYB-2097 Backfill results migration (M1)

## Why

Batch/backfill workflows can finish without a registered algorithm result, leaving operators blind to "awaiting upload" items and blocking rollup. M1 closes the visibility chasm with an authenticated upload API, `algo_run_results` staging, and UI states — without migrating the full Databricks scheduler.

## What Changes

### Modified capabilities
- `POST /api/v1/backfill/results` — validate manifest + payload; upsert staging row.
- Batch/backfill detail UI — show awaiting_result / registered / validation_failed.
- Remove production `Implement me` stubs on completion paths that gate on result presence.

### Unchanged (deferred)
- GCS object-finalize webhook consumer.
- Databricks `backfill_scheduler.py` production wiring.
- Bulk historical bucket import.

## Impact

- **Affected code**: `backend/internal/handlers/backfill`, `backend/routes`, `Frontend/src` backfill surfaces, `api/openapi.yaml`, `docs/review/api-guide.md`, SDK client.
- **New APIs**: `POST /api/v1/backfill/results`
- **Dependencies**: Existing backfill auth model; `algo_run_results` table.

## Scope

- **In scope**: HTTP upload → staging → UI visibility (M1).
- **Out of scope**: CYB-258 rollup/rerun semantics, CYB-259 node drilldown (referenced, not duplicated).

## Success Criteria

- [ ] OpenSpec checkpoint approved before runtime edits.
- [ ] Smoke `scripts/smoke-backfill-results-api-dev.sh` passes on dev after deploy.
- [ ] Batch item shows actionable state when workflow done but result missing.
