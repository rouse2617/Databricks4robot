# Proposal — CYB-2177

## Why
Pipeline batch executions can remain `running` indefinitely when child workflows are stuck in Kubernetes `Pending/Unschedulable` because requested component resources exceed the current execution environment.

## What Changes

### New Capabilities
- Pipeline runs converge long-running unschedulable nodes to a terminal `Error` state with the original scheduler reason preserved.
- Pipeline deployment rejects clearly over-limit component resource requests before submitting Argo workflows when static environment limits are configured.

### Modified Capabilities
- Batch execution status aggregation treats resource-unschedulable child runs as failed rather than leaving the parent batch running forever.
- Resource validation reports actionable messages that point users to lower resource requests, another compute class, or platform capacity expansion.

## Impact
- **Affected code**: `backend/internal/usecase/pipeline`, `backend/internal/usecase/backfill`, `backend/internal/transpiler`, `backend/internal/models`, `backend/internal/postgres`
- **New APIs**: None expected. Existing deploy/run/backfill responses may return `400 INVALID_ARGUMENT` earlier for resource limits and existing run records may transition to `Error` sooner.
- **Dependencies**: No new service dependencies. Kueue, Volcano, and Kubernetes quota changes are out of scope for this bug fix.

## Scope
- **In scope**:
  - Add conservative unschedulable detection in the run watcher/status refresh path.
  - Add configurable static resource ceilings for deploy-time validation.
  - Preserve Kubernetes scheduler messages in run/node/batch diagnostics.
  - Add backend tests for unschedulable timeout and resource limit rejection.
- **Out of scope**:
  - Installing Kueue, Volcano, or new GKE autoscaling/node-pool infrastructure.
  - Exposing Kubernetes concepts such as tolerations, node selectors, or affinity directly to normal component users.
  - Mutating existing component definitions to reduce user-requested resources.
  - Enforcing ResourceQuota/LimitRange in the cluster without a separate platform rollout.

## Success Criteria
- [ ] A run with a node stuck in `Pending` and scheduler message `Unschedulable` beyond the configured threshold becomes `Error`.
- [ ] The parent batch containing only failed/errored unschedulable children stops showing `running`.
- [ ] A component requesting resources above configured environment limits is rejected before workflow submission with a clear message.
- [ ] Normal short `Pending` runs are not marked failed before the threshold.
- [ ] Existing successful and failed workflow status synchronization still passes targeted backend tests.
