# Proposal — CYB-1660

## Why

Pipeline execution detail navigation can open the wrong identifier after Argo workflow cleanup or when DataBrew run IDs differ from workflow names. Failed runs also do not always surface the failure cause or log entry clearly enough for debugging.

## What Changes

- Resolve execution details from durable DataBrew pipeline run IDs before falling back to live workflow names.
- Preserve `runId` in execution-detail links from the execution list, run dialog, and asset lineage.
- Make failed nodes and status drift visible in the detail header.
- Keep log entry points available for failed asset-node rows even when `logRef` is missing.
- Make execution-list operation controls explicit when no operation is available.

## Impact

- **Affected code**: `Frontend/src/pages/useWorkflowDetail.ts`, `Frontend/src/pages/WorkflowDetailPage.tsx`, `Frontend/src/pages/WorkflowExecutionList.tsx`, `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/pages/WorkflowDagView.tsx`, `Frontend/src/components/asset-detail/LineageTab.tsx`, `Frontend/src/api/pipelineApi.ts`
- **New APIs**: none; frontend now consumes the existing `GET /api/v1/pipeline-runs/{runId}` endpoint.
- **Dependencies**: none.

## Scope

- **In scope**: execution-detail lookup stability, failure visibility, node log entry consistency, focused frontend tests.
- **Out of scope**: backend API changes, Argo controller behavior changes, deployment automation changes, new logging or terminal backends.

## Success Criteria

- [ ] Execution detail pages can resolve by stable DataBrew `runId` when workflow names are stale or ambiguous.
- [ ] TTL-cleaned or ledger-first runs keep a useful run ledger view instead of a confusing 404-first experience.
- [ ] Failed nodes show a prominent summary and keep a log entry available.
- [ ] Status mismatch between DataBrew ledger and Argo workflow is explicitly warned.
- [ ] Related frontend tests and production build pass.
