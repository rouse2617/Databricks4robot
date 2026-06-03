# Tasks — CYB-1613

## Context files
- See `context-files.md`.

## Checkpoint
- [x] Write OpenSpec proposal, tasks, context files, and pipeline spec delta.
- [x] Stop for OpenSpec confirmation before runtime edits. User confirmed continuation in chat.

## Implementation
- [x] [backend] Add a helper to convert runtime `wfv1.NodeStatus` to workflow detail node response.
- [x] [backend] Add static DAG task placeholders for tasks present in `spec.templates[].dag.tasks` but absent from runtime status.
- [x] [backend] Preserve runtime node metadata when both runtime status and static task definition exist.
- [x] [backend] Extend DAG edge construction so static dependencies can point to pending task placeholders.
- [x] [backend] Keep helper/container/exit templates from appearing as standalone pending nodes.
- [x] [backend] Return workflow timestamps as UTC RFC3339 instead of formatting local time with a literal `Z`.

## API contract sync
See [`docs/agents/AI-RULES.md` § API contract sync](../../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — updated workflow detail wording for runtime + static pending DAG task nodes.
- [x] `docs/review/api-guide.md` — updated workflow detail wording for full DAG rendering.
- [x] `scripts/api-guide-smoke.sh` or targeted workflow smoke — covered by handler regression test; no new endpoint or smoke path required.
- [x] `openspec/changes/CYB-1613-workflow-full-dag/specs/pipeline/spec.md` — keep behavior delta aligned with implementation.
- [x] [SDK] Confirm no SDK code change needed because endpoint shape/fields are unchanged.
- [x] [Frontend] Confirm no frontend API type change needed because `phase` and `type` are string fields.

## Local verification
- [x] [backend] `cd backend && go test ./internal/handlers/workflow/...`
- [x] [backend] `cd backend && go test ./...` if API docs/smoke changes or shared helpers are touched.

## Deploy verification
- [x] Backend dev deploy or local full-stack smoke. Verified locally with frontend `http://localhost:5176` and backend `http://localhost:8080`.
- [x] Open a running multi-node workflow detail and confirm all static DAG tasks are visible immediately. Verified `cyb1613-full-dag-smoke-92c88`: Argo status had root + first pod only, backend/UI showed all 4 DAG steps with 3 pending downstream nodes.

## PR
- [x] PR template filled with Linear `CYB-1613`, OpenSpec change ID, test evidence, and deploy evidence.
