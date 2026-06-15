# Context files — CYB-2097

## Backend

- `backend/routes/routes.go` — backfill route group
- `backend/internal/usecase/backfill/usecase.go`
- `backend/internal/postgres/backfill_repo.go`
- `backend/internal/postgres/repos.go` — `algo_run_results` if present
- `backend/internal/handlers/backfill/` — upload handler target

## Frontend

- `Frontend/src/pages/BatchJobDetailPage.tsx`
- `Frontend/src/api/batchJobApi.ts`
- `Frontend/src/components/backfill/` — `BackfillResultsUpload.tsx` (new)

## Contract & ops

- `api/openapi.yaml`
- `docs/review/api-guide.md` — backfill results section
- `scripts/smoke-backfill-results-api-dev.sh` (new)
- `scripts/dev-backend-env.sh`

## Reference (non-SSOT)

- `AlgoProject/backfill_upload_results.py`
- `scripts/upload-deface-backfill-10.sh` (if present on branch)

## Related OpenSpec

- CYB-258 — rollup/rerun
- CYB-259 — node drilldown
- CYB-2041 — legacy manifest repro harness (ops only)
