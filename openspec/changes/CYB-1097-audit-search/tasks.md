# Tasks — CYB-1097

## Context files
- `backend/internal/handlers/audit/handler.go` — existing audit search handler skeleton
- `backend/routes/routes.go` — route registration and nil-handler safety
- `schemas/pg-phase0.sql` — `asset_events` columns/indexes
- `api/openapi.yaml` — endpoint contract
- `docs/review/api-guide.md` — curl documentation
- `scripts/api-guide-smoke.sh` — dev smoke coverage

## Implementation
- [x] [backend] Add focused tests for audit search query validation (`limit`, `cursor`, RFC3339 times).
- [x] [backend] Add focused tests for successful filtered audit search response shape and `next_cursor`.
- [x] [backend] Harden `/api/v1/audit/search` handler behavior against nil storage and scan/query errors.
- [x] [backend] Ensure filters cover `actor`, `event_type`, `run_id`, `time_from`, and `time_to`.
- [x] [backend] Ensure responses are newest-first and include `items`, `limit`, and optional `next_cursor`.
- [x] [backend] Keep implementation read-only; do not touch migrations, auth middleware, or outbox internals.

## API contract sync
- [x] `api/openapi.yaml` — add `/api/v1/audit/search` path, query params, response schema, and errors.
- [x] `docs/review/api-guide.md` — add audit search curl examples, response example, and validation errors.
- [x] `scripts/api-guide-smoke.sh` — add success and validation-error checks.
- [x] `openspec/changes/CYB-1097-audit-search/specs/search/spec.md` — behavior delta.
- [x] SDK — out of scope unless user explicitly asks for SDK parity in this issue.
- [x] Frontend — out of scope; no UI consumes the endpoint in this issue.

## Local verification (Tier L per AI-RULES because this adds public HTTP API contract)
- [x] `gofmt` touched Go files.
- [x] `cd backend && go vet ./... && go test ./...`.
- [x] `bash -n scripts/api-guide-smoke.sh`.
- [x] `git diff --check`.
- [x] `scripts/agent-harness/after-edit.sh`.

## Deploy verification (before commit — runtime only)
- [x] Build backend image with git SHA tag and `cloudrun-dev-latest`.
- [x] Push backend image tags.
- [x] Deploy backend dev using SHA-tagged image and record image tag, Cloud Run revision, and URL.
- [x] Source `scripts/dev-backend-env.sh` and run targeted smoke for:
  - successful `/api/v1/audit/search` query
  - invalid time or cursor validation error
- [x] Run relevant `scripts/api-guide-smoke.sh` audit-search checks on dev.

### Deploy record — CYB-1097
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:b29c594-cyb1097` | `cyber-databrew-backend-dev-00165-hk8` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

### Dev verification evidence — 2026-05-23
- `GET /api/v1/audit/search?limit=5` on Cloud Run dev returned `200`, `limit=5`, `items_len=5`, and `next_cursor`.
- `GET /api/v1/audit/search?time_from=not-rfc3339` returned `400 INVALID_ARGUMENT`.
- `GET /api/v1/audit/search?cursor=not-int` returned `400 INVALID_ARGUMENT`.
- `BASE="$BASE" TOKEN="$GRACE_TOKEN" IAP_TOKEN="$CLOUDRUN_ID_TOKEN" bash scripts/api-guide-smoke.sh` returned `19 passed, 1 failed`; the only failure was Cloud Run root `/healthz` returning Google 404. The CYB-1097 audit search checks passed.

## PR
- [ ] PR template filled with Linear `CYB-1097`, OpenSpec change id, tests, and deploy evidence.
- [ ] Linear updated with commit/deploy summary after merge.
