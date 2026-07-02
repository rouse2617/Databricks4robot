# Tasks — CYB-1531

## Context files
- `backend/internal/argo/client.go` — Argo workflow/log client behavior.
- `backend/internal/handlers/workflow/handler.go` — workflow detail response.
- `backend/internal/handlers/workflow/logs_sse.go` — log stream endpoint.
- `Frontend/src/api/workflowApi.ts` — workflow/log API client types.
- `Frontend/src/pages/useWorkflowDetail.ts` — workflow detail and log state.
- `Frontend/src/pages/PipelinePage.tsx` — designer drag/drop integration.
- `Frontend/src/components/pipeline/*` — canvas and component palette behavior.
- `docs/agents/deploy-before-commit.md` — deploy requirements.
- `docs/agents/deploy-verification.md` — dev verification requirements.

## Implementation
- [x] [backend] Add or expose pod-name resolution for Argo Pod nodes using workflow node metadata and/or pod annotations.
- [x] [backend] Update workflow detail JSON so Pod nodes include `podName` when available.
- [x] [backend] Update workflow log endpoints to use resolved pod names while still accepting node ids from the frontend.
- [x] [backend] Remove `grep=podName` from Argo log requests so logs are not filtered empty.
- [x] [Frontend] Update workflow node types and detail UI to show real pod name.
- [x] [Frontend] Ensure log drawer requests logs for the selected node and renders non-empty fallback logs.
- [x] [Frontend] Fix component drag/drop identity so the dropped component matches the dragged palette item.
- [x] [Frontend] Make the workflow detail layout less sparse where feasible without a broad redesign.

## Verification
- [x] [backend] Run targeted Go tests for Argo/workflow handlers.
- [x] [backend] Run full `go test ./...`.
- [x] [Frontend] Run lint/tests for touched frontend code.
- [x] [Frontend] Build frontend.
- [x] [dev] Deploy backend dev if backend log behavior changes.
- [x] [dev] Deploy frontend dev if UI behavior changes.
- [x] [dev] Verify with Chrome DevTools MCP:
  - [x] Open `/pipeline?tab=design`, drag named components, confirm dropped node names match.
  - [x] Open a succeeded workflow detail and confirm logs render for a Pod node.
  - [x] Confirm node details show real pod name.

## API Contract Sync
- [x] Confirm whether `api/openapi.yaml` documents workflow node detail fields; update if response shape adds `podName`.
- [x] Confirm `docs/review/api-guide.md` workflow log examples remain accurate; update if needed.

## Deploy Record
- Backend dev revision: `cyber-databrew-backend-dev-00410-dkh`
- Frontend dev revision: `cyber-databrew-frontend-dev-00296-qjr`
- Backend image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:156a2ef-cyb1531`
- Frontend image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend:156a2ef-cyb1531-cloudrun`
- Browser evidence:
  - `deploy-verify-drag.png`
  - `deploy-verify-logs.png`
