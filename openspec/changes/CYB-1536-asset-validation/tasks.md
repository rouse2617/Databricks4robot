# Tasks — CYB-1536 Asset Validation

## Context Files
- `backend/internal/repository/asset_repository.go`
- `backend/internal/postgres/repos.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/algorun/usecase.go`
- `backend/internal/handlers/pipeline/handler.go`
- `backend/internal/handlers/algorun/handler.go`
- `backend/cmd/server/core.go`
- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `scripts/api-guide-smoke.sh`

## Checkpoint
- [x] Write OpenSpec proposal, design, tasks, and spec delta.
- [x] Stop for OpenSpec confirmation before runtime edits.

## Implementation
- [x] Add batch existence method to `AssetRepository`.
- [x] Implement Postgres batch existence query excluding deleted assets.
- [x] Add shared asset ID validation helper.
- [x] Replace pipeline deploy per-ID validation loop with shared helper.
- [x] Wire asset validation dependency into algo-run usecase.
- [x] Validate `algo-runs.input_asset_ids` before insert.
- [x] Return structured missing/invalid asset details from handlers.
- [x] Preserve empty asset list and filter-only behavior.

## API Contract Sync
- [x] Update `api/openapi.yaml` docs for `asset_ids` and `input_asset_ids`.
- [x] Add documented `400 INVALID_ARGUMENT` missing asset examples.
- [x] Update `docs/review/api-guide.md`.
- [x] Update `scripts/api-guide-smoke.sh` with one missing asset error case.
- [x] Update SDK tests if SDK validates or documents algo-run payloads. N/A: SDK does not validate these payloads in this change.

## Verification
- [x] `cd backend && go test ./internal/usecase/pipeline/...`
- [x] `cd backend && go test ./internal/usecase/algorun/...`
- [x] `cd backend && go test ./internal/handlers/pipeline/...`
- [x] `cd backend && go test ./internal/handlers/algorun/...`
- [x] `cd backend && go test ./internal/postgres/...`
- [x] `cd backend && go test ./...`
- [ ] Dev smoke: pipeline deploy missing asset returns `400`.
- [ ] Dev smoke: algo-run create missing asset returns `400`.
