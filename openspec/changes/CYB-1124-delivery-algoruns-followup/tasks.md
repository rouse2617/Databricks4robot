# Tasks — CYB-1124

## Context files
- `backend/internal/handlers/algorun/handler.go` — response envelope and query parsing for algo-runs list
- `backend/internal/usecase/algorun/usecase.go` — pagination normalization behavior
- `backend/internal/postgres/algo_runs.go` — repo pagination and total count semantics
- `backend/internal/handlers/delivery/handler.go` — delivery commit, C2 add-items, cancel, retry, ack flows
- `backend/internal/postgres/repos.go` — delivery item writes, asset delivery index refresh, idempotency persistence
- `backend/internal/repository/common.go` — repository interfaces for delivery and idempotency behavior
- `backend/internal/handlers/delivery/handler_test.go` — handler-level regression tests
- `backend/internal/postgres/repos_test.go` — repo-level SQL behavior tests
- `scripts/api-guide-smoke.sh` — dev smoke paths for algo-runs and delivery C2

## Root cause notes
- `GET /algo-runs` normalizes page values inside usecase/repo but returns handler-local raw query values.
- `POST /deliveries/{id}/items` ignores duplicate insert results yet increments count using raw request length.
- `POST /deliveries/{id}/cancel` performs delivery update before asset index refresh without a transaction wrapping both effects.
- `POST /deliveries` reads idempotency before side effects and saves with upsert after side effects, leaving a same-key concurrent race.

## Implementation
- [x] [backend] Add focused failing tests for normalized algo-runs list response metadata.
- [x] [backend] Return normalized `page/page_size` values from algo-runs list.
- [x] [backend] Add focused failing tests for duplicate C2 add-items count behavior.
- [x] [backend] Make C2 add-items count reflect actual unique delivery item rows after insertion.
- [x] [backend] Add focused failing tests for cancel refresh failure rollback semantics.
- [x] [backend] Make delivery cancel update and asset index refresh atomic.
- [x] [backend] Add focused failing tests for same-key idempotency conflict/concurrency behavior where feasible.
- [x] [backend] Harden delivery commit idempotency so same-key mismatched payloads cannot overwrite records after side effects.
- [x] [backend] Keep changes scoped; do not touch migrations, auth middleware, or outbox internals.

## API contract sync
Existing HTTP APIs are behaviorally corrected. No new endpoints are introduced.

- [x] `api/openapi.yaml` — update only if documented response/error semantics need clarification.
- [x] `docs/review/api-guide.md` — clarify corrected pagination/idempotency/delivery item semantics if needed.
- [x] `scripts/api-guide-smoke.sh` — adjust delivery C2 smoke to exercise unique item count and idempotency/error paths.
- [x] `openspec/changes/CYB-1124-*/specs/*/spec.md` — behavior delta.
- [x] SDK — no public SDK method change expected; record in decisions if this remains out of scope.

## Local verification (Tier L per AI-RULES)
- [x] `cd backend && make fmt && make vet && go test ./...`
- [x] `cd Frontend && npm run lint && npm run build` if Frontend files are touched — N/A, no Frontend files touched
- [x] `cd sdk && env UV_CACHE_DIR=/private/tmp/uv-cache uv run pytest tests/unit/` if SDK files are touched — N/A, no SDK files touched

## Deploy verification (before commit — runtime only)
- [x] Build backend image with git SHA tag and `cloudrun-dev-latest`.
- [x] Push backend image tags.
- [x] Deploy backend dev using SHA-tagged image and record image tag, Cloud Run revision, and URL.
- [x] Source `scripts/dev-backend-env.sh` and run targeted backend smoke for:
  - normalized algo-runs pagination metadata
  - duplicate delivery C2 add-items count behavior
  - delivery cancel/index consistency
  - delivery idempotency conflict behavior
- [x] Run `scripts/api-guide-smoke.sh` with the relevant write mode if safe for dev.

### Deploy record — CYB-1124
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:a15de1f-cyb1124` | `cyber-databrew-backend-dev-00164-vhr` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

### Dev verification evidence — 2026-05-23
- `GET /api/v1/algo-runs?page=-1&page_size=999` on Cloud Run dev returned `{"page":1,"page_size":50,"total":4,"items_len":4}`.
- `RUN_WRITES=1 BASE="$BASE" TOKEN="$GRACE_TOKEN" IAP_TOKEN="$CLOUDRUN_ID_TOKEN" bash scripts/api-guide-smoke.sh` returned `37 passed, 1 failed`; the only failure was Cloud Run root `/healthz` returning Google 404. Business endpoints passed, including duplicate delivery C2 add-items count (`asset_count=1`) and same idempotency key with different payload returning `409`.
- Targeted delivery cancel/index smoke on Cloud Run dev created a draft, committed it, verified asset `delivery_count=1`, cancelled it, then verified asset `delivery_count=0` for delivery `d35357df-ded9-43a3-9b1c-9f644fb4a596`.

## PR
- [ ] PR template filled with Linear `CYB-1124`, OpenSpec change id, tests, and deploy evidence.
- [ ] Linear updated with PR/deploy summary after merge.
