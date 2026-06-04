# CYB-1644 Tasks

## Context files

- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx` — runtime tab monitoring, billing, and terminal guidance rendering
- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.test.tsx` — node detail runtime UI assertions
- `Frontend/src/pages/WorkflowDetailPage.tsx` — log viewer empty-state and asset-node cost row copy
- `Frontend/src/pages/WorkflowDetailPage.test.tsx` — workflow detail log/cost UI assertions

## 1. Implementation

- [x] [Frontend] Classify runtime monitoring empty states by node phase and Pod diagnostic readiness before showing warning copy.
- [x] [Frontend] Classify runtime billing empty states by node phase and cost snapshot availability before showing warning copy.
- [x] [Frontend] Replace technical live-log pagination / no-log copy with user-facing language for pending or unschedulable nodes.
- [x] [Frontend] Condense redundant terminal guidance in the runtime tab without hiding the terminal entry point on the node card.
- [x] [Frontend] Add or update focused tests for pending/running/completed node runtime empty states and log viewer copy.

## 2. Verification

- [x] Run targeted frontend tests for workflow detail and node detail panels.
- [x] Run frontend build or equivalent Tier L check before PR.
- [x] Verify with Chrome DevTools MCP on a workflow detail page after frontend dev deployment.
- [ ] Open PR to `dev` with Linear and OpenSpec links.

## Deploy record — CYB-1644

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| frontend-dev | `cyber-databrew-frontend:cyb1644-20260604154020` | `cyber-databrew-frontend-dev-00319-kqp` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |
