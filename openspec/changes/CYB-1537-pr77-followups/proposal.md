# Proposal — CYB-1537 PR #77 Review Follow-up

## Problem

Code review of merged PR #77 (pipeline run backend model) flagged three residual runtime issues that the user approved merging over:

1. `CreateRunByTemplate` silently discards `ShouldBindJSON` errors. The request body is optional, so empty body must still work, but malformed JSON (truncated body, wrong `Content-Type`) currently passes through and surfaces as a confusing downstream error.
2. `getWorkflowResourceUsage` reads the entire `pipeline_deployments` table on every call to locate a single matching row. The PR's whole point was to make runs first-class with `workflow_name UNIQUE` on `pipeline_runs`; the new repo should be the primary read path.
3. `logs_sse.go` allocates `[]byte(line)` twice per emitted line to compute length and to slice. Go strings are byte sequences, so both conversions are unnecessary and add real GC pressure under high-volume log streams.

## Goals

- Surface JSON binding failures for `POST /api/v1/pipeline-runs/template/{id}` while preserving the empty-body contract.
- Make the new `pipelineRepo` the primary lookup in `getWorkflowResourceUsage`; keep `deploymentRepo` as a fallback for legacy rows that have no `pipeline_runs` entry yet.
- Drop the `[]byte(line)` round-trips in the SSE log streamer without changing observable behavior.
- Cover each fix with a focused unit test that fails on the old code and passes on the new code.

## Non-Goals

- Frontend changes (no API contract change).
- Pipeline run workflow retry semantics and the `pipeline_runs.workflow_name UNIQUE` interaction (deferred — needs product decision).
- `GetResourceUsage` `ObservedAt` documentation.
- SDK parametrized tests for `tail_lines` / `limit_bytes` / `follow` / `container`.
- `ErrNodeNotFound` vs `ErrDeploymentNotFound` naming cleanup.
- Replacing the legacy `deploymentRepo` lookup entirely.

## Acceptance Criteria

- `POST /api/v1/pipeline-runs/template/{id}` with a malformed JSON body returns 400 with a clear error and does not invoke the usecase.
- `POST /api/v1/pipeline-runs/template/{id}` with no body or an empty body behaves exactly as before (usecase invoked with zero-value body).
- `getWorkflowResourceUsage` calls `pipelineRepo.FindByWorkflowName` first and only falls back to `deploymentRepo.FindAll` when the run repo has no row.
- `logs_sse.go` no longer contains `[]byte(line)` outside of the line scan buffer.
- All existing tests pass; three new tests (one per fix) pass.
