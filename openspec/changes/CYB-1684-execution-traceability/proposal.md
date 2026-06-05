# Proposal — CYB-1684

## Why

The pipeline execution list and detail pages do not provide stable traceability for "which pipeline produced this run".

Today the execution list mostly shows:

- run name
- status
- node count
- timestamps
- a `模板 vX` tag

That is not enough for historical operations. Users still cannot reliably answer:

- which pipeline template the run came from
- which version or snapshot was executed
- whether the run was based on a draft or a saved template version
- whether the run was triggered manually, from asset-driven run flow, from batch run, or from API

This becomes worse when templates are renamed, deleted, or continue evolving after the run has finished.

## What Changes

### New Capabilities

- Pipeline runs persist stable execution-origin metadata in the run ledger.
- The execution list shows pipeline name and version directly in the row summary.
- The execution detail header shows pipeline, version, snapshot identity, and trigger source.

### Modified Capabilities

- Execution history becomes ledger-first for traceability fields instead of reconstructing context from current template state.
- The execution list information hierarchy emphasizes "what pipeline ran" before secondary metadata.
- Failed execution rows expose a short failure summary or a direct failure-entry affordance.

## Impact

- **Affected code**: `backend/internal/models/pipeline.go`, `backend/internal/usecase/pipeline/usecase.go`, `backend/internal/postgres/pipeline_repo.go`, `backend/internal/handlers/pipeline/handler.go`, `api/openapi.yaml`, `docs/review/api-guide.md`, `Frontend/src/api/pipelineApi.ts`, `Frontend/src/pages/WorkflowExecutionList.tsx`, `Frontend/src/pages/WorkflowDetailPage.tsx`
- **New APIs**: likely response-field additions on existing `/api/v1/pipeline-runs` and `/api/v1/pipeline-runs/:id`
- **Dependencies**: none

## Scope

- **In scope**:
  - persist stable pipeline traceability fields on pipeline runs
  - return those fields from run list/detail APIs
  - redesign execution list row metadata to show pipeline name/version clearly
  - show pipeline/version/snapshot/trigger source in execution detail summary
  - tighten adjacent execution-list UX gaps tied to this information architecture
- **Out of scope**:
  - reworking pipeline versioning itself
  - adding new execution filtering backends unrelated to traceability
  - broad workflow-detail redesign beyond traceability summary
  - actual run lineage graph across downstream assets

## Success Criteria

- [ ] Every execution row makes it clear which pipeline produced the run.
- [ ] Version is shown as pipeline context, not only as an isolated `模板 vX` tag.
- [ ] Execution detail shows pipeline name, version, snapshot identity, and trigger source at the top.
- [ ] Historical runs keep their original pipeline traceability even if the current template later changes name or version state.
- [ ] Trigger source is defined for at least manual run, asset run, batch run, and API run.
- [ ] Failed runs expose a visible short failure summary or direct failure-entry action from the list.
