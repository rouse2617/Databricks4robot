# Proposal — CYB-1541 Pipeline Run Smoke Fixes

## Why
Local full-stack smoke cannot complete the core pipeline flow because no-asset runs fail during persistence and key form inputs append stale text instead of replacing user edits.

## What Changes

### Modified Capabilities
- Pipeline runs preserve an explicitly empty asset list as `[]` instead of persisting it as `NULL`.
- Pipeline run dialogs and design inputs replace existing values reliably and reset transient search state when reopened.
- Pipeline, component, and execution management surfaces expose asset-style 8-character stable ID labels alongside names, while retaining the full underlying ID for debugging.
- Local smoke can cover create/select component, drag/save pipeline, no-asset run, asset run, and execution-detail inspection without generating invalid workflow names from stale input text.

## Impact
- **Affected code**: `backend/internal/postgres`, `backend/internal/usecase/pipeline`, `Frontend/src/components/pipeline`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: Fix persistence of empty `asset_ids`, add regression coverage where practical, fix frontend controlled input replacement/reset behavior, and verify through local full-stack UI smoke.
- **In scope**: Show stable IDs for pipeline templates, components, and execution records where those objects are listed or inspected.
- **Out of scope**: Redesigning asset search semantics, changing Argo output declaration policy, or implementing run_events / asset x node backend data.

## Success Criteria
- [ ] `POST /api/v1/pipeline-runs/template/{id}` with `asset_ids: []` creates a run or reaches Argo validation without a Postgres `asset_ids` not-null error.
- [ ] Pipeline name and asset-search inputs can be replaced without stale text being appended.
- [ ] Closing and reopening the run dialog clears transient asset search text/results.
- [ ] Users can see IDs for saved pipelines, components, and completed/running execution records.
- [ ] A local browser smoke validates create/select component, drag/save pipeline, no-asset run, asset run, and execution detail/log access.
