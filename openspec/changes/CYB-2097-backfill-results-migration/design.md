# Design — CYB-2097 Backfill results migration (M1)

## ADR-1: Staging store

**Decision:** Use `algo_run_results` as M1 staging keyed by asset + derivative algo key + version.

**Rationale:** Table already exists; avoids new migration for pilot. Heavy blobs deferred to GCS pointers in a later change.

**Consequences:** Handler must enforce idempotent upsert and payload size limits; UI reads staging via existing backfill detail APIs or thin list extension.

## ADR-2: API contract SSOT

**Decision:** OpenAPI + `docs/review/api-guide.md` are authoritative; `scripts/upload-deface-backfill-10.sh` and `AlgoProject/backfill_upload_results.py` are reference only.

## ADR-3: Item visibility model

| State | Condition |
|-------|-----------|
| `awaiting_result` | Workflow terminal, expected derivative/report key absent |
| `registered` | Valid upload accepted for asset + report key |
| `validation_failed` | Upload rejected or manifest mismatch |

Counters reconcile on next batch completion pass (no silent stub completion).

## Implementation sketch

1. Handler: validate `asset_id`, `report_id`, `version`, JSON payload; authorize like other backfill routes.
2. Repo: upsert `algo_run_results` with conflict target on business key.
3. Frontend: `BackfillResultsUpload.tsx` wired from batch detail when item is awaiting.
4. Routes: register handler; grep and replace `Implement me` on affected `services.go` paths.

## Verification

- Tier L: OpenAPI sync, api-guide smoke, optional SDK unit tests.
- Dev deploy per `deploy-before-commit.md` before claiming verified.
