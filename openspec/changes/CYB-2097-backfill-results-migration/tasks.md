# Tasks — CYB-2097 backfill results migration

## 1. OpenSpec (this change)

- [x] proposal.md
- [x] design.md
- [x] tasks.md
- [x] specs/backfill/spec.md
- [x] context-files.md
- [x] decisions.md
- [ ] User checkpoint: 「OpenSpec OK，继续」before any `backend/` / `Frontend/` edit

## 2. Backend — upload API (after approval)

- [ ] Implement `POST /api/v1/backfill/results` handler (validate manifest, asset_id, report_id, version, payload)
- [ ] Upsert `algo_run_results`; idempotent on asset + algo_key + version
- [ ] Wire routes; remove/replace `Implement me` stubs on affected completion path in `services.go`
- [ ] Unit tests for validation + upsert; integration test if harness exists

## 3. API contract sync (same PR)

- [ ] `api/openapi.yaml`
- [ ] `docs/review/api-guide.md` (align with existing backfill results section)
- [ ] `scripts/smoke-backfill-results-api-dev.sh` — happy + error path
- [ ] SDK client if endpoint is public REST surface

## 4. Frontend (after approval)

- [ ] `BackfillResultsUpload.tsx` — call API, show errors, disable when not awaiting
- [ ] `BatchJobDetailPage` / backfill detail — item state for awaiting_result / registered
- [ ] Types in `backfillApi.ts`

## 5. Ops alignment

- [ ] Document env vars for `upload-deface-backfill-10.sh` vs dev smoke
- [ ] Cross-check `AlgoProject/backfill_upload_results.py` field names with handler

## 6. Verify (Tier L before PR)

- [ ] `make fmt && make vet`; `go test` touched packages
- [ ] `cd Frontend && npm run lint && npm run test -- --run` (touched)
- [ ] Smoke on dev after deploy (`deploy-before-commit.md`)
- [ ] Optional: pilot upload on M1 asset set (~10) per ops script

## 7. Out of scope (record in decisions)

- [ ] GCS object finalize webhook consumer
- [ ] Databricks `backfill_scheduler.py` production wiring
- [ ] Bulk historical bucket import
