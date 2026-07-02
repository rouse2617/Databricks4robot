# Tasks — CYB-1609

## Context files
- See `context-files.md`.

## Checkpoint
- [x] Write OpenSpec proposal, design, tasks, context files, and pipeline spec delta.
- [x] Stop for OpenSpec confirmation before runtime edits.

## Implementation
- [x] [backend] Extend workflow log response metadata with applied bounds, truncation, and pagination availability while preserving existing `logs` compatibility.
- [x] [backend] Add or tighten validation for cursor/window query parameters without pretending Argo supports stable historical offset pagination.
- [x] [backend] Keep SSE follow cancellation-safe and ensure end/error events are documented and test-covered.
- [x] [Frontend] Update workflow log API types and URL builders for bounded window/follow parameters.
- [x] [Frontend] Replace boolean-only follow state with lifecycle state: idle, connecting, connected, ended, error.
- [x] [Frontend] Render log viewer controls for bounded tail, load more when available, stop/reconnect follow, download, and clear unavailable states.
- [x] [Frontend] Cap the in-memory/displayed log buffer and show when older displayed lines were dropped or the response was truncated.
- [x] [Frontend] Ensure failed DAG node log action opens the log viewer with the node selected and fetches a bounded tail.

## API contract sync
See [`docs/agents/AI-RULES.md` § API contract sync](../../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — update workflow log response schema, query parameters, SSE lifecycle examples, and error cases.
- [x] `docs/review/api-guide.md` — add curl examples for bounded window, follow, unavailable pagination, and download scope.
- [x] `scripts/api-guide-smoke.sh` or a workflow log smoke script — verify missing node ID, bounded defaults, and stream route if practical.
- [x] `openspec/changes/CYB-1609-pipeline-log-cursor/specs/pipeline/spec.md` — keep behavior delta aligned with implementation.
- [x] [Frontend] `Frontend/src/api/workflowApi.ts` and log hook types — align with OpenAPI.
- [x] [SDK] Update workflow logs SDK params and unit tests.

## Local verification
- [x] [backend] `cd backend && go test ./internal/argo/...`
- [x] [backend] `cd backend && go test ./internal/handlers/workflow/...`
- [x] [backend] `cd backend && go test ./...`
- [x] [Frontend] `cd Frontend && npm run lint`
- [x] [Frontend] `cd Frontend && npm run test -- --run useWorkflowDetail`
- [x] [Frontend] `cd Frontend && npm run build`
- [x] [SDK] `cd sdk && uv run ruff check src/cyber_databrew_sdk/managers/workflows.py tests/unit/test_managers.py`
- [x] [SDK] `cd sdk && uv run pytest tests/unit/test_managers.py -q`

## Deploy verification
- [ ] Apply migrations if any are added. Expected: none.
- [ ] Build, push, and deploy backend/frontend dev with SHA image tags if runtime code changes.
- [ ] Record Cloud Run image tags, revisions, and URLs.

### Frontend MCP verification
- [ ] Open `/pipeline/executions/<workflow>` on dev.
- [ ] Open a successful node log viewer and verify bounded tail metadata.
- [ ] Open a failed node via one-click log action and verify the selected node is correct.
- [ ] Start follow, observe connecting/live/end or error state, and stop/reconnect.
- [ ] Download loaded logs and verify file scope metadata.
- [ ] Confirm browser console has no new errors.

## PR
- [ ] PR template filled with Linear `CYB-1609`, OpenSpec change ID, API sync rows, test evidence, and deploy evidence.
