# Design — CYB-2177

## Architecture Context
- **Constraints**: DataBrew submits Argo Workflows, but Kubernetes Scheduler owns the final placement decision. Users should not manage Kubernetes scheduling internals.
- **Goals**: Prevent infinite-running batch UI states, fail resource-unschedulable work with actionable diagnostics, and reject obvious resource-limit violations before workflow creation.
- **Non-Goals**: Do not introduce Kueue/Volcano, do not change GKE node pools, and do not expose taints/tolerations/node selectors in the normal component form.

## Affected Modules
- `backend/internal/usecase/pipeline` — run submission validation and watcher/status convergence.
- `backend/internal/usecase/backfill` — batch item aggregation after child runs become terminal.
- `backend/internal/transpiler` — resource extraction remains the source for manifest requests; validation should reuse its resource model where practical.
- `backend/internal/postgres` — persist terminal status/message updates through existing repositories.

## Architecture Decisions

### Decision 1: Runtime unschedulable convergence before cluster scheduler changes
- **Approach**: Detect `Pending` nodes whose message contains Kubernetes scheduler unschedulable signals and mark the run `Error` only after a configurable age threshold.
- **Alternative**: Configure ResourceQuota/LimitRange first and rely on Kubernetes admission.
- **Rationale**: Runtime convergence directly fixes the current infinite `running` symptom without changing shared dev cluster policy.
- **Trade-off**: Some invalid jobs are still submitted, but they stop lingering indefinitely.
- **Risk**: Over-eager detection could fail jobs that would have scheduled after autoscaler capacity appears.
- **Mitigation**: Require both explicit unschedulable scheduler wording and an elapsed threshold; make the threshold configurable.
- **Rollback**: Disable the feature with configuration or raise the threshold while preserving existing watcher behavior.

### Decision 2: Static deploy-time resource ceilings as the first validation layer
- **Approach**: Add environment-configured max CPU, memory, disk, and GPU values for the default execution target and validate component resources before workflow submission.
- **Alternative**: Dynamically query nodes and replicate Kubernetes Scheduler feasibility logic.
- **Rationale**: Static ceilings are cheap, deterministic, easy to test, and produce clear user errors. Full scheduler replication is complex and still imperfect.
- **Trade-off**: Static ceilings can be stale if the cluster changes.
- **Risk**: Valid workloads may be rejected if limits are configured too low.
- **Mitigation**: Treat unset limits as disabled, document defaults, and keep messages explicit about configured environment limits rather than claiming live cluster truth.
- **Rollback**: Unset or raise the configured ceilings.

### Decision 3: Defer ResourceQuota/LimitRange rollout
- **Approach**: Keep Kubernetes ResourceQuota/LimitRange as a follow-up platform rollout after observing validation behavior.
- **Alternative**: Apply quota immediately as part of this backend bug fix.
- **Rationale**: Quota changes affect every workload in the namespace and can break unrelated dev tasks. Backend validation is safer for the product path.
- **Risk**: Non-DataBrew clients can still submit oversized pods directly.
- **Mitigation**: This change targets DataBrew pipeline submissions; platform-level enforcement can follow with a separate rollout plan.

## Data Flow

```text
User deploys pipeline
  -> backend validates component resources against configured ceilings
  -> if valid, Argo Workflow is submitted
  -> watcher refreshes workflow/node snapshots
  -> Pending node with Unschedulable message older than threshold
  -> run status becomes Error with scheduler message
  -> backfill item and batch aggregation converge to failed/terminal state
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| False-positive failure during temporary autoscaler delay | A run may fail instead of eventually scheduling | Use explicit scheduler messages plus a conservative timeout |
| Static ceilings drift from real cluster capacity | Users see rejection even after cluster expansion | Make ceilings configuration-driven and easy to update |
| Existing tests assume `Pending` maps to `Running` forever | Test updates required | Add targeted cases for pre-threshold and post-threshold behavior |
| Quota not enforced cluster-wide | Direct Kubernetes/Argo clients can still bypass DataBrew validation | Track ResourceQuota/LimitRange as follow-up platform work |
