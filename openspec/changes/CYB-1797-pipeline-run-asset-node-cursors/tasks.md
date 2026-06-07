# Tasks — CYB-1797

## Context Files
- `backend/internal/handlers/pipeline/handler.go` — asset-node HTTP parameter handling and error mapping.
- `backend/internal/postgres/pipeline_repo.go` — asset-node query, ordering, cursor generation.
- `backend/internal/models/pipeline.go` — asset-node list options and response models.
- `backend/internal/handlers/pipeline/handler_test.go` — HTTP regression tests.
- `backend/internal/postgres/*pipeline*_test.go` — repository cursor regression coverage.
- `api/openapi.yaml` — cursor contract if response semantics are documented.
- `docs/review/api-guide.md` — API usage notes if cursor encoding changes.

## Implementation
- [x] [backend] Add opaque cursor encode/decode helpers for pipeline run asset-node listing.
- [x] [backend] Add order-specific keyset predicates for default, cost, duration, and status ordering.
- [x] [backend] Ensure every order mode has a deterministic `id ASC` tie-breaker.
- [x] [backend] Generate `nextCursor` from the last returned row, not the overflow row.
- [x] [backend] Map malformed or mismatched cursors to `400 INVALID_ARGUMENT`.
- [x] [backend] Keep no-cursor requests behavior-compatible for existing clients.
- [x] [backend] Update API contract docs if the cursor format or error behavior is externally documented.

## Verification
- [x] [backend] Add repository tests that page through default ordering without skips or duplicates.
- [x] [backend] Add repository tests for `orderBy=cost`, `orderBy=duration`, and `orderBy=status`.
- [x] [backend] Add handler/usecase test for malformed cursor returning `400 INVALID_ARGUMENT` where practical.
- [x] [backend] `go test ./internal/postgres ./internal/handlers/pipeline ./internal/usecase/pipeline`
- [x] [backend] `go test ./...`
- [ ] Deploy backend dev.
- [ ] Run targeted smoke against `/api/v1/pipeline-runs/:id/asset-nodes` for multi-page default and non-default sort modes.

## Documentation / Tracking
- [ ] PR body references `CYB-1797`.
- [ ] PR body references `openspec/changes/CYB-1797-pipeline-run-asset-node-cursors`.
- [ ] Record deploy verification evidence in this `tasks.md` or PR body.

## Verification Evidence
- 2026-06-07: `cd backend && go test ./internal/postgres ./internal/handlers/pipeline ./internal/usecase/pipeline` passed.
- 2026-06-07: `cd backend && go test ./...` passed.
