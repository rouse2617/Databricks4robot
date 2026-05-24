# Tasks — CYB-1111

## Context files
- `deploy/k8s/jobs/biglake-silver-gold-build-once.yaml` — existing Silver/Gold job.
- `deploy/cloudrun/bronze-incremental/main.py` — Bronze ingest and Silver quality context.
- `backend/internal/handlers/lakehouse/handler.go` — table listing and lakehouse query behavior.
- `backend/internal/handlers/lakehouse/handler_test.go` — lakehouse handler tests.
- `docs/review/lakehouse-incremental-ingestion.md` — Bronze duplicate and cursor behavior.
- `docs/review/api-guide.md` — lakehouse user docs.

## Implementation
- [x] [infra] Ensure the Silver asset-events export path is explicit and repeatable.
- [x] [backend] Update `/lakehouse/tables` to report `silver_asset_events_current` when available.
- [x] [backend] Add or update handler tests for multi-table row count response shaping.
- [x] [docs] Update api-guide/lakehouse docs with Silver table semantics and disabled-environment behavior.
- [x] [scripts] Update smoke coverage to tolerate missing Silver while verifying table listing shape.

## API contract sync
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — describe table listing can include Silver asset-events table.
- [x] `docs/review/api-guide.md` — document Silver table semantics and examples.
- [x] `scripts/api-guide-smoke.sh` — update lakehouse table check.
- [x] `openspec/changes/CYB-1111-silver-asset-events-export/specs/lakehouse/spec.md` — behavior delta.
- [x] SDK/Frontend — out of scope; response shape remains compatible.

## Local verification
- [x] `cd backend && go test ./internal/handlers/lakehouse/...`
- [x] `cd backend && make fmt && make vet` (scoped as `gofmt -l` on touched files + `go vet ./internal/handlers/lakehouse/...`)

## Deploy verification
- [x] Backend dev deploy required if handler behavior changes.
- [x] Run lakehouse table smoke; document if BigQuery/lakehouse is disabled.

### Combined deploy verification — 2026-05-24

Validated together with CYB-1099, CYB-1109, and CYB-1110 from combined worktree `/Users/hrp/cyber/cyber-databrew-CYB-combined-test`.

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:8a27089-cyb1099-1109-1110-1111-combined` | `cyber-databrew-backend-dev-00170-6cl` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

Lakehouse smoke:
- `GET /api/v1/lakehouse/tables` returned `200`.
- Response shape contained `items[].table_name` and `items[].row_count`.
- `bronze_asset_events` was present with row count `1570418`.
- `silver_asset_events_current` was absent, which is tolerated until the Silver export job materializes the table.
- `scripts/api-guide-smoke.sh` covered the lakehouse shape check; result was 27 passed / 1 failed, with the only failure being pre-existing `/healthz` 404 on the Cloud Run root host.

## PR
- [ ] PR template filled; Linear `CYB-1111` and OpenSpec change ID linked.
