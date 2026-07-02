# Decisions — CYB-1537 PR #77 Review Follow-up

## 2026-06-02 — Bundle three review findings into one issue
- **Context**: PR #77 review flagged three residual issues (silent JSON bind error, full `deployments` table scan, `[]byte` round-trips in SSE). All are bug/perf fixes in the same PR, no runtime behavior change visible to callers, and touching three different files.
- **Decision**: Bundle into a single Linear issue (CYB-1537) and a single OpenSpec change directory; one `fix/CYB-1537-pr77-followups` branch and one PR.
- **Alternatives**: Three separate issues + change dirs + branches (more process overhead for three closely related fixes in the same PR).
- **Rationale**: AI-RULES §"Automatic behavior" treats related follow-up fixes from one review as a single change set. Bundling keeps the OpenSpec ↔ code ↔ PR ↔ deploy-verify cycle in lock-step. Out-of-scope items are still listed in the Linear description so they are not lost.

## 2026-06-02 — `pipelineRepo` is the primary lookup; `deploymentRepo` stays as fallback
- **Context**: `getWorkflowResourceUsage` currently scans all `pipeline_deployments` to find one row. The new `pipeline_runs` table has a `workflow_name UNIQUE` constraint and an `execution_target_id`/`target_snapshot` already captured, so a targeted lookup is cheaper and matches the design intent of CYB-1534.
- **Decision**: Call `pipelineRepo.FindByWorkflowName` first; if the row is missing, fall back to the legacy `deploymentRepo.FindAll` scan. This preserves behavior for runs that pre-date the migration while making the new run repo the primary read path.
- **Alternatives**: Remove the deployment scan entirely (breaks compat for any pre-migration run still in `pipeline_deployments` only). Replace the deployment scan with a targeted `FindByWorkflowName` on the deployment repo (requires adding a new repo method that PR #77 did not add).
- **Rationale**: Minimum-diff fix that satisfies the "first-class run repo is the primary path" intent without expanding scope into the deployment repo interface. The fallback can be removed in a later cleanup once all dev/staging runs have been backfilled into `pipeline_runs`.

## 2026-06-02 — Empty body stays valid on the template endpoint
- **Context**: The original `_ = c.ShouldBindJSON(&req)` pattern was intentional: `CreateRunByTemplate` allows callers to POST with no body, falling back to the template's stored asset IDs. Fixing the silent error must not break the empty-body path.
- **Decision**: Only return 400 when the body is non-empty and fails to bind. An empty body keeps the existing behavior (usecase invoked with zero-value body).
- **Alternatives**: Always require a JSON body (breaks existing clients). Always ignore the bind error (current bug).
- **Rationale**: Preserves the documented contract for the endpoint while surfacing real client mistakes. Matches the "optional body, but if you sent something, it must be valid" pattern used elsewhere in the codebase.
