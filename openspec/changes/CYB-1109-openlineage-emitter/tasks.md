# Tasks — CYB-1109

## Context files
- `docs/review/unified-asset-catalog/design/openlineage-integration.md` — intended mapping and rollout notes.
- `backend/internal/models/schema_evolution.go` — `AssetEvent` shape.
- `backend/internal/postgres/repos.go` — asset event and relation query patterns.
- `backend/internal/config/config.go` — runtime config.
- `backend/cmd/server/optional.go` — optional background service wiring.

## Implementation
- [x] [backend] Add `internal/openlineage` payload structs and builder.
- [x] [backend] Add an HTTP emitter with timeout and deterministic error handling.
- [x] [backend] Add disabled-by-default config flags and README/api-guide operational notes.
- [x] [backend] Wire the emitter as an optional background consumer without changing default runtime behavior.
- [x] [backend] Add unit tests for payload mapping, skip behavior, and HTTP success/failure.

## API contract sync
- [x] Not applicable: no HTTP API is added or changed.

## Local verification
- [x] `cd backend && go test ./internal/openlineage/...`
- [x] `cd backend && go test ./cmd/server/... ./internal/config/...`
- [x] `cd backend && make fmt && make vet` (scoped as `gofmt -l` on touched files + `go vet ./internal/openlineage/... ./internal/config/... ./cmd/server/...`)

## Deploy verification
- [x] Backend dev deploy required before commit if runtime wiring changes.
- [x] Verify default disabled config keeps backend healthy.
- [x] If a test endpoint is available, verify one emitter POST; otherwise document disabled-mode verification.

### Combined deploy verification — 2026-05-24

Validated together with CYB-1099, CYB-1110, and CYB-1111 from combined worktree `/Users/hrp/cyber/cyber-databrew-CYB-combined-test`.

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:8a27089-cyb1099-1109-1110-1111-combined` | `cyber-databrew-backend-dev-00170-6cl` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

Verification:
- Cloud Run env did not set `OPENLINEAGE_EMITTER_ENABLED`, so config default keeps the emitter disabled.
- `GET /api/v1/search/sync-status` returned `200`, confirming backend health with existing outbox settings.
- No Marquez/OpenLineage test endpoint was configured, so enabled-mode POST was not exercised on dev.

## PR
- [ ] PR template filled; Linear `CYB-1109` and OpenSpec change ID linked.
