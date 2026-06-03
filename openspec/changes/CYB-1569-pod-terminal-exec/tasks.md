# Tasks — CYB-1569

## OpenSpec
- [x] Create proposal, design, tasks, and spec delta.
- [x] Record OpenSpec checkpoint approval.

## Backend
- [x] [backend] Add terminal policy to execution target snapshot with default disabled.
- [ ] [backend] Add terminal session persistence model and migration. Deferred pending explicit migration approval.
- [x] [backend] Add in-memory terminal session store with create/find/update lifecycle methods.
- [x] [backend] Add usecase/handler logic to resolve workflow node Pod and validate target namespace/policy.
- [x] [backend] Add Kubernetes exec proxy abstraction using backend-owned credentials.
- [x] [backend] Add REST create/status/terminate handlers.
- [x] [backend] Add WebSocket attach handler with one-time/short-lived attach token.
- [x] [backend] Append run events for created/ended/failed sessions when run exists.
- [ ] [backend] Write durable audit events for session lifecycle. Deferred with persistent session table.
- [x] [backend] Add unit/handler tests for allow, deny, unavailable, and terminate behavior.

## API Contract Sync
- [x] `api/openapi.yaml` — terminal session paths, schemas, error responses.
- [x] `docs/review/api-guide.md` — examples for create/status/terminate and WebSocket attach semantics.
- [x] `sdk/src/cyber_databrew_sdk/` — terminal session helpers if public API is exposed.
- [x] `sdk/tests/unit/` — SDK tests if SDK helpers are added.
- [x] `scripts/smoke-pod-terminal-dev.sh` — dev smoke for disabled policy and allowed preflight when enabled.
- [x] `openspec/changes/CYB-1569-pod-terminal-exec/specs/pipeline/spec.md` — behavior delta.
- [ ] `docs/review/sql.md` — session table if migration is added. Not applicable in this slice.

## Frontend
- [x] [Frontend] Add typed API client for terminal session create/status/terminate.
- [x] [Frontend] Add WebSocket terminal client with frame handling and cleanup.
- [x] [Frontend] Integrate terminal policy state into `WorkflowNodeDetailPanel` runtime tab.
- [x] [Frontend] Add terminal panel shell using the existing monospace panel; interactive stdin/xterm.js deferred until the next terminal slice.
- [x] [Frontend] Show disabled/forbidden/unavailable/expired states with business copy.
- [x] [Frontend] Add terminate/disconnect controls.
- [x] [Frontend] Add tests for runtime tab terminal states.

## Verification
- [x] `cd backend && go test ./internal/handlers/workflow ./routes`
- [x] `cd backend && go test ./...`
- [x] `cd Frontend && npm run lint`
- [x] `cd Frontend && npm run test -- --run WorkflowNodeDetailPanel workflowApi`
- [x] `cd Frontend && npm run build`
- [x] `cd sdk && uv run pytest tests/unit/` if SDK changes.
- [x] Apply migration to dev before backend deploy if session table is added. No migration in this slice.
- [ ] Run `scripts/smoke-pod-terminal-dev.sh` against dev.
- [ ] Chrome DevTools MCP: verify disabled state, allowed terminal session, terminate, and console.

## Deploy / PR
- [ ] Deploy backend/frontend dev before commit.
- [ ] Record Cloud Run revisions and image tags.
- [ ] Update Linear CYB-1569 with verification evidence.
- [ ] Fill PR template and link CYB-1569 + OpenSpec change id.
