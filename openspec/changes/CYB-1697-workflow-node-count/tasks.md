# Tasks — CYB-1697

## Implementation
- [x] Update workflow summary node-count calculation to exclude Argo DAG/root/controller nodes.
- [x] Keep workflow detail `nodes` payload unchanged so DAG rendering and node diagnostics still work.
- [x] Add/adjust backend workflow handler tests for root DAG plus two Pod steps.
- [x] Add/adjust frontend execution-list regression only if the UI merge logic needs a guard.

## Verification
- [x] `go test ./internal/handlers/workflow`
- [x] Relevant frontend targeted tests if frontend code changes.
- [x] Dev API smoke for `/api/v1/workflows` and `/api/v1/pipeline-runs/{id}`.
- [x] Chrome DevTools MCP verification on `/pipeline?tab=executions` and `/pipeline/executions/test-77705a?runId=4ec0af22-d1eb-4880-9a62-088a8a5951b0`.

## Documentation / Tracking
- [ ] Record PR link and verification evidence on Linear CYB-1697.
- [ ] PR body references CYB-1697 and this OpenSpec change.

## Deploy Record
- **Backend image**: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:5271de3`
- **Cloud Run revision**: `cyber-databrew-backend-dev-00630-hmj`
- **Dev URL**: `https://cyber-databrew-dev.cyberorigin.ai/`
