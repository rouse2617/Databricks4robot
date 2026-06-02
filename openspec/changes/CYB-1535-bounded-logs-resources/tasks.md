# Tasks — CYB-1535 Bounded Logs And Resources

## Context Files
- `backend/internal/argo/client.go`
- `backend/internal/handlers/workflow/handler.go`
- `backend/internal/handlers/workflow/logs_sse.go`
- `backend/internal/usecase/pipeline/resource_usage.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/routes/routes.go`
- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `Frontend/src/api/workflowApi.ts`

## Checkpoint
- [x] Write OpenSpec proposal, design, tasks, and spec delta.
- [x] Stop for OpenSpec confirmation before runtime edits.

## Implementation
- [x] Add bounded log query options to the Argo client.
- [x] Make workflow log reads default to bounded tail responses.
- [x] Enforce `tailLines` and `limitBytes` server-side maximums.
- [x] Register `/api/v1/workflows/:name/logs/stream`.
- [x] Keep or redirect `/api/v1/workflows/:name/log/stream` compatibility if frontend currently calls it.
- [x] Remove full-log fallback from SSE code paths.
- [x] Emit structured JSON SSE events and heartbeat events.
- [x] Add workflow-level resource endpoint.
- [x] Add node-level resource endpoint.
- [x] Correct resource response source metadata and live metrics semantics.

## API Contract Sync
- [x] Update `api/openapi.yaml` log schemas, stream endpoint, and resource schemas.
- [x] Update `docs/review/api-guide.md` with bounded log examples and resource source notes.
- [x] Update `scripts/api-guide-smoke.sh` for missing node ID, bounded logs, and stream route if practical.
- [x] Keep frontend API compatibility: JSON log response still includes `logs`, and `/log/stream` remains an alias.

## Verification
- [x] `cd backend && go test ./internal/argo/...`
- [x] `cd backend && go test ./internal/handlers/workflow/...`
- [x] `cd backend && go test ./internal/usecase/pipeline/...`
- [x] `cd backend && go test ./...`
- [ ] Dev smoke: bounded log request with default params.
- [ ] Dev smoke: missing `nodeId` returns `400`.
- [ ] Dev smoke: stream route returns `text/event-stream`.
