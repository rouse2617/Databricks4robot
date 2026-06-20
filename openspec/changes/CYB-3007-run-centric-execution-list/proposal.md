# Proposal - CYB-3007 Run-Centric Execution List

## Summary

Make Execution Hub use DataBrew Runs as the only product list data source.

## Problem

The Execution Hub list had already moved its primary data path to `/api/v1/runs`, but the label-filter path still queried Argo `/workflows` and merged runtime workflow rows with Run ledger rows. That keeps `workflowName` and Argo workflow listing in the product execution list path, which conflicts with Runtime OS:

- Run is the product execution entity.
- Workflow is a runtime debug reference.
- Product execution rows must not be invented from live Argo workflows.

## Goals

- Remove Argo workflow listing from `WorkflowExecutionList`.
- Derive execution rows only from `/api/v1/runs`.
- Preserve runtime workflow debug usage in detail views, logs, terminal, Pod diagnostics, and operations fallback.
- Keep existing filters working against available Run ledger data.
- Add regression coverage that label-filtered list loads do not query Argo workflow listing.

## Non-Goals

- No backend schema migration.
- No removal of legacy `/workflows` APIs.
- No change to Run Inspector runtime debug calls.
- No server-side label filtering in this slice.

## Acceptance Criteria

- Execution Hub list refresh calls `listRuns` and does not call `listWorkflows`.
- Label-filtered Execution Hub loads still do not call `listWorkflows`.
- Live-only Argo workflows never appear as product execution rows.
- Existing Run list status, name, date, version, selection, and batch-scoped behavior remain functional.
