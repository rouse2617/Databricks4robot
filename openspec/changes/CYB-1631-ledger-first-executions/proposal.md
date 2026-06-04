# Proposal — CYB-1631

## Why

Execution records can hide estimated cost because the UI uses live Argo workflows as the primary list source, while Argo TTL can remove workflows before users review historical runs.

## What Changes

### New Capabilities

- Pipeline execution records remain visible from the DataBrew run ledger after the Argo workflow has been removed.
- Estimated run cost is shown from DataBrew `pipeline_runs` ledger data when available.
- Newly submitted Argo workflows keep their runtime object for 30 days after completion to support short-term DAG, pod, and log diagnostics.

### Modified Capabilities

- The Pipeline execution list uses durable DataBrew run records as the primary table source.
- Live Argo workflow data becomes optional enrichment for active runs rather than the source of truth.
- The generated Argo workflow TTL changes from 1 hour to 30 days; durable history still comes from DataBrew run tables.

## Impact

- **Affected code**: `Frontend/src/pages/WorkflowExecutionList.tsx`, `Frontend/src/api/pipelineApi.ts`, `backend/internal/transpiler/transpiler.go`, `backend/internal/usecase/pipeline/usecase.go`
- **New APIs**: none
- **Dependencies**: none

## Scope

- **In scope**: execution list data-source ordering, cost display from ledger records, empty-state wording for unavailable cost, Argo workflow TTL extension for new runs, focused frontend/backend tests.
- **Out of scope**: changing the pricing algorithm, adding new database columns, actual GCP Billing reconciliation, Argo workflow archive.

## Success Criteria

- [ ] Historical DataBrew runs appear in the execution list even when their Argo workflow is no longer returned by `/workflows`.
- [ ] Runs with `totalEstimatedCost` display a formatted estimated cost in the list.
- [ ] Active Argo workflows that do not yet have ledger records still appear as live records.
- [ ] Cost empty states distinguish pending active runs from unavailable historical cost.
- [ ] Frontend tests cover ledger-first cost display and live workflow fallback.
- [ ] New workflow manifests use `ttlStrategy.secondsAfterCompletion = 2592000` seconds.
