# Proposal — CYB-3061

## Why
Pipeline pods carry zero business-domain labels (only Argo/GKE's own), so real GCP billing data can never be sliced by batch/task-type/owner without a manual join — the existing self-computed cost estimator was found this session to over-count ~99.5% of its total due to a `resourcesDuration` bug, motivating a move toward GKE Cost Allocation's real billing data as ground truth.

## What Changes

### New Capabilities
- pipeline: every Argo Workflow pod is stamped with `cyber-databrew/batch-job-id`, `cyber-databrew/template-id`, `cyber-databrew/owner` labels at submission time, sanitized to satisfy Kubernetes label-value constraints

### Modified Capabilities
- (none — additive only; `Transpile()` behavior is unchanged when the new option is unset)

## Impact
- **Affected code**: `backend/internal/transpiler/transpiler.go`, `backend/internal/usecase/pipeline/usecase.go`
- **New APIs**: none (internal-only; no HTTP request/response shape changes)
- **Dependencies**: none new

## Scope
- **In scope**: labeling pods created by the main pipeline transpiler (`Transpile()`, used by `Deploy`/`DeployByTemplateID`/backfill execution)
- **Out of scope**:
  - `build_workflows.go` (component_build / rag_build workflow path) — separate builder, not part of this change
  - Fixing the existing cost-estimation formula bug in `cost.go`
  - Enabling/configuring BigQuery billing export (blocked on billing-account permissions, tracked separately)
  - Backfilling labels onto already-running/completed workflows (labels apply to newly submitted workflows only, same as GKE Cost Allocation's own enablement-forward semantics)
  - Adding a date or namespace label (both already exist as native, non-duplicative dimensions — K8s timestamp / BigQuery time partitioning, and GKE Cost Allocation's own `k8s-namespace`)

## Success Criteria
- [ ] A newly submitted pipeline workflow's pods carry `cyber-databrew/batch-job-id`, `cyber-databrew/template-id`, `cyber-databrew/owner` labels whenever the corresponding `DeployOptions` field is non-empty
- [ ] An `Owner` value containing `@` (email) produces a valid Kubernetes label (no pod creation failure)
- [ ] Omitting all three `DeployOptions` fields produces no `PodMetadata` change (backward compatible — existing callers unaffected)
- [ ] Total label count per pod stays well under GKE Cost Allocation's 50-label-per-pod cap

## Goals (SLO)
- **Quality**: new sanitization logic covered by unit tests (email, empty, >63 chars, illegal characters)
- **Compatibility**: zero behavior change for existing callers that don't set the new option (e.g. `build_workflows.go` path)
