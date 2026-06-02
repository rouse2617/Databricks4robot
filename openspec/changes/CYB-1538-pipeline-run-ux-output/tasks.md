# Tasks — CYB-1538 Pipeline Run UX Output

## Context Files
- `Frontend/src/api/pipelineApi.ts`
- `Frontend/src/pages/PipelinePage.tsx`
- `Frontend/src/components/pipeline/DeployPanel.tsx`
- `Frontend/src/pages/ComponentManager.tsx`
- `backend/internal/postgres/pipeline_repo.go`
- `backend/internal/transpiler/transpiler.go`

## Checkpoint
- [x] Write OpenSpec proposal, decisions, tasks, and spec delta.
- [x] Stop for confirmation before runtime edits.

## Implementation
- [x] Change saved-pipeline run API helper to call `/pipeline-runs/template/:id`.
- [x] Send `asset_ids: []` for no-asset runs from pipeline run UI paths.
- [x] Normalize nil pipeline run asset IDs before Postgres save.
- [x] Cast pipeline run `asset_ids` inserts as `text[]`.
- [x] Make Argo output parameter declaration conditional on consumption or explicit output-path writes.
- [x] Support shell output directory injection for `["sh"], ["-c", script]` and `["sh", "-c"], [script]`.
- [x] Disable autocomplete on fields that showed append-like behavior during MCP/browser operation.

## Verification
- [x] `cd backend && go test ./internal/transpiler ./internal/postgres`
- [x] `cd Frontend && npm test -- pipelineApi.test.ts PipelinePage.test.tsx DeployPanel.test.tsx`
- [x] `cd Frontend && npx biome check <changed frontend files>`
- [x] `cd Frontend && npm run build`
- [ ] Browser smoke past login with deployed frontend/backend.
