# Tasks — CYB-1541 Pipeline Run Smoke Fixes

## Context Files
- `backend/internal/models/pipeline.go`
- `backend/internal/postgres/pipeline_repo.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `Frontend/src/components/pipeline/PipelinePage.tsx`
- `Frontend/src/api/pipelineApi.ts`

## Checkpoint
- [x] Create Linear issue `CYB-1541`.
- [x] Write OpenSpec proposal, design, tasks, and spec delta.
- [x] Stop for OpenSpec confirmation before runtime edits.

## Implementation
- [x] Reproduce input replacement failure against local frontend with MCP.
- [x] Normalize empty `asset_ids` before saving `pipeline_runs`.
- [x] Add or update backend regression coverage for no-asset run persistence.
- [x] Fix pipeline name and workflow name input replacement behavior.
- [x] Reset run dialog asset search state on close/open and prevent stale search text from appending.
- [x] Add click-to-add fallback for pipeline components when drag/drop is unreliable.
- [x] Show stable asset-style IDs for pipeline templates, components, and execution records in management views.

## Verification
- [x] Backend targeted tests for pipeline repository/usecase.
- [x] Frontend changed-file Biome check.
- [x] Frontend focused tests for AssetPicker, PipelinePage, DeployPanel.
- [x] Frontend production build.
- [ ] MCP local smoke at `http://localhost:5176` against `http://localhost:8080`:
  - [ ] Create or select component.
  - [ ] Drag into pipeline canvas.
  - [ ] Save pipeline.
  - [ ] Run without asset.
  - [ ] Run with asset.
  - [ ] Open execution detail and logs.
  - [ ] Confirm pipeline/component/execution IDs are visible.
- [ ] MCP smoke after user deploys this branch into local full-stack environment.

## Follow-up Observations
- [ ] If Argo still fails on `/tmp/outputs/output`, record separately because it is component output policy, not this regression.
