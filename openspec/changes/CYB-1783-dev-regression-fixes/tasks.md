# Tasks — CYB-1783

## Context Files
- `backend/internal/usecase/pipeline/usecase.go` — template detail and version lookup behavior.
- `backend/internal/handlers/pipeline/handler.go` — template detail HTTP behavior and tests.
- `backend/routes/routes.go` — action route registration.
- `backend/internal/handlers/action/handler.go` — action list response envelope.
- `Frontend/src/components/pipeline/AssetPicker.tsx` — run modal asset search and selection.
- `Frontend/src/components/pipeline/AssetPicker.test.tsx` — component regression coverage.
- `api/openapi.yaml` — existing action annotation contract.
- `docs/review/api-guide.md` — API guide sync if route documentation needs adjustment.

## Implementation
- [x] [backend] Add a failing regression test for unique short template id lookup.
- [x] [backend] Resolve short template ids only when the prefix match is unique; keep missing or ambiguous prefixes as not found.
- [x] [backend] Add action annotation route aliases that share the existing action handler behavior.
- [x] [Frontend] Add direct asset lookup fallback in `AssetPicker` after empty search results for exact asset id queries.
- [x] [Frontend] Preserve ordinary search result behavior and empty-state behavior when direct lookup also fails.

## API Contract Sync
- [x] Verify `api/openapi.yaml` already includes `/api/v1/assets/{id}/action-annotations`.
- [x] Update `docs/review/api-guide.md` if action route documentation does not mention the served alias.
- [x] Add or update an API smoke check for the action annotation alias.
- [x] Confirm no SDK changes are required because this restores an already-documented route and does not introduce a new public resource.
- [x] Keep OpenSpec behavior delta aligned with implementation.

## Verification
- [x] [backend] `go test ./internal/handlers/pipeline/...`
- [x] [backend] Route/action alias targeted test or equivalent package test.
- [x] [backend] `go test ./...`
- [x] [Frontend] `npm run test -- --run src/components/pipeline/AssetPicker.test.tsx`
- [x] [Frontend] `npm run lint`
- [x] [Frontend] `npm run build`
- [x] Deploy backend dev if backend runtime changes are included.
- [x] Deploy frontend dev Worker if frontend runtime changes are included.
- [x] Chrome DevTools MCP verification on `/pipeline?templateId=c00f1136`.
- [x] Chrome DevTools MCP verification that pipeline run modal can find/select `CYB10A01`.
- [x] Chrome DevTools MCP verification that `/assets/CYB10A01` Action timeline no longer shows the 404 error state.

## Documentation / Tracking
- [x] Append fix and verification evidence to `docs/review/pipeline-dev-regression-log-2026-06-07.md`.
- [x] Record deploy revision/version ids in this `tasks.md` after deployment.
- [ ] PR body references CYB-1783 and this OpenSpec change.

## Deploy Record
- **Backend image tag**: `cyber-databrew-backend:6d4cac9-cyb1783b`
- **Backend image digest**: `sha256:8741db84869ddee09b6e2be5cb79c5282c04177a132798119b6e620297fccb49`
- **Backend Cloud Run revision**: `cyber-databrew-backend-dev-cyb1783b2`
- **Backend dev URL**: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- **Frontend Worker**: `cyber-databrew-dev`
- **Worker Version ID**: `e5206c26-ce82-4cb4-8914-800c34fad18c`
- **Frontend dev URL**: `https://cyber-databrew-dev.cyberorigin.ai/`

## Dev Verification Evidence
- `GET /api/v1/pipelines/c00f1136` -> `200`, template `test`, full id `c00f1136-1f86-4a32-a25d-91bf81557fc8`.
- `GET /api/v1/assets/CYB10A01/action-annotations?limit=200` -> `200`, empty action envelope.
- `GET /api/v1/assets/CYB10A01/actions?limit=200` -> `200`, backward-compatible empty action envelope.
- Chrome DevTools MCP: `/pipeline?templateId=c00f1136` loads `test` with two `Count Lines` nodes.
- Chrome DevTools MCP: deploy modal search `CYB10A01` shows selectable row with type `segment`, state `ready`, storage `gs://cyb1100/CYB10A01`.
- Chrome DevTools MCP: `/assets/CYB10A01` Action timeline shows empty state `该 seg 暂无 action`, not a 404 error.
- Chrome DevTools MCP console: no runtime `error` / `warn`; only existing browser issues for deprecated feature usage and form fields missing `id` / `name`.
