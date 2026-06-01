# Proposal — CYB-1532

## Why
Pipeline can already submit Argo workflows, but users cannot reliably run a saved pipeline against a selected asset set and then inspect status, logs, and resources by asset and step. Cluster and namespace are also hidden backend configuration, which blocks the future multi-cluster operating model.

## What Changes

### New Capabilities
- Pipeline runs become asset-driven: users choose a saved pipeline, a batch of assets, and an execution target before submitting.
- DataBrew exposes execution targets as first-class runtime destinations, starting with the current dev Argo namespace and leaving room for multiple clusters and namespaces.
- Execution records show the selected assets, target, workflow, node status, logs, and resource information in one product flow.
- Pipeline components include enough runtime contract metadata to tell users what a valid step must produce before it is run.

### Modified Capabilities
- Saved pipeline management is separated from execution history so template CRUD and run monitoring are not mixed in the same work surface.
- Existing deployment records are treated as workflow-level runs for compatibility, but the user-facing model moves toward pipeline runs and per asset x node tracking.
- Run actions that bind assets require an explicit asset selection path instead of hiding asset selection behind a dropdown.

## Impact
- **Affected code**: `backend/internal/usecase/pipeline`, `backend/internal/handlers/pipeline`, `backend/internal/repository`, `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/components/pipeline`, `Frontend/src/api/pipelineApi.ts`
- **New APIs**: likely `GET /api/v1/execution-targets`, `POST /api/v1/pipeline-runs`, `GET /api/v1/pipeline-runs`, `GET /api/v1/pipeline-runs/:id`
- **Dependencies**: no new third-party dependency planned for the MVP

## Scope
- **In scope**: product model cleanup, dev execution target, asset-driven run submission, run list/detail UI, node log/resource surfacing, and component runtime contract validation/hints.
- **Out of scope**: replacing Argo, implementing a full multi-cluster controller, cross-cluster credentials rotation, autoscaling policy engine, historical metrics warehouse, and production deployment.

## Success Criteria
- [ ] A user can create or edit a reusable component and understand the step output contract before running it.
- [ ] A user can drag components into a pipeline, save it, reopen it, and run it with selected assets.
- [ ] A run record displays pipeline name, selected asset count, execution target, workflow name, status, and timestamps.
- [ ] A run detail view lets the user inspect node status, pod name, host, logs, and resource duration.
- [ ] The same run model can represent at least one asset x one node, with a clear path to multiple assets x multiple nodes.
- [ ] Direct runs without selected assets remain possible only as an explicit no-asset run, not an accidental default.

## Goals (SLO)
- **Latency**: run submission p95 under 2 seconds excluding Argo scheduling time.
- **Concurrency**: list views remain usable with 100 recent runs and at least 50 active status refreshes.
- **Quality**: targeted backend and frontend tests cover asset selection, target selection, run creation, and run list/detail rendering.
