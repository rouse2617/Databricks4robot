# Tasks — CYB-1099

## Context files
- `backend/internal/handlers/asset/handler.go` — existing stream handler skeleton.
- `backend/internal/handlers/asset/handler_test.go` — handler test patterns and fake SQL rows.
- `backend/routes/routes.go` — route registration.
- `api/openapi.yaml` — API contract.
- `docs/review/api-guide.md` — user-facing API guide.
- `scripts/api-guide-smoke.sh` — dev smoke coverage.

## Implementation
- [x] [backend] Validate malformed `Last-Event-ID` with `400 INVALID_ARGUMENT`.
- [x] [backend] Return `404 ASSET_NOT_FOUND` when the asset does not exist before opening the SSE stream.
- [x] [backend] Keep stream query batches bounded and ordered by `event_seq ASC`.
- [x] [backend] Add focused tests for happy path, resume cursor, malformed header, and missing asset.

## API contract sync
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — add `/api/v1/assets/{id}/events/stream`, `Last-Event-ID`, `text/event-stream`, and errors.
- [x] `docs/review/api-guide.md` — add curl examples, SSE frame shape, resume semantics, and errors.
- [x] `sdk/src/asset_sdk/` + `client.py` — out of scope; record if deferred.
- [x] `scripts/api-guide-smoke.sh` — add a bounded stream smoke or documented skip when no asset is available.
- [x] `openspec/changes/CYB-1099-events-stream/specs/asset-management/spec.md` — behavior delta.
- [x] [Frontend] `src/api/` or hooks — out of scope; UI does not consume the stream in this issue.

## Local verification
- [x] `cd backend && go test ./internal/handlers/asset/...`
- [x] `cd backend && make fmt && make vet` (scoped as `gofmt -l` on touched files + `go vet ./internal/handlers/asset/...`)

## Deploy verification
- [x] Backend dev deploy required before commit.
- [x] Run targeted stream smoke against dev with an existing asset.
- [x] Record image tag, revision, and dev URL.

### Combined deploy verification — 2026-05-24

Validated together with CYB-1109, CYB-1110, and CYB-1111 from combined worktree `/Users/hrp/cyber/cyber-databrew-CYB-combined-test`.

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:8a27089-cyb1099-1109-1110-1111-combined` | `cyber-databrew-backend-dev-00170-6cl` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

Targeted smoke:
- `GET /api/v1/assets/{id}/events/stream` returned `200` for an existing asset.
- `GET /api/v1/assets/{id}/events/stream` with `Last-Event-ID: not-a-number` returned `400 INVALID_ARGUMENT`.
- `scripts/api-guide-smoke.sh` covered the SSE checks; result was 27 passed / 1 failed, with the only failure being pre-existing `/healthz` 404 on the Cloud Run root host.

## PR
- [ ] PR template filled; Linear `CYB-1099` and OpenSpec change ID linked.
