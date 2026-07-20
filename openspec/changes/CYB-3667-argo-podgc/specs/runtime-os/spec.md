# Runtime OS Spec Delta — CYB-3667

## MODIFIED Requirements

### Requirement: Transpiled workflows reclaim step pods on success
- **Before**: The system SHALL set only a `TTLStrategy` on each generated Argo
  Workflow, so a completed workflow's step pods persist until the Workflow
  object's TTL (30 days) cascades their deletion.
- **After**: The system SHALL additionally set `PodGC.Strategy =
  OnWorkflowSuccess` on each generated Argo Workflow, so the step pods of a
  fully-successful workflow are deleted promptly after completion, while the
  pods of a failed workflow are retained for operator diagnostics. The Workflow
  object itself SHALL continue to be reclaimed by the existing `TTLStrategy`.
- **Reason**: Terminated step pods from successful runs accumulate in etcd
  (no automatic GC below kube-controller-manager's terminated-pod threshold) and
  add API-server pressure with no operational value, since DataBrew — not the
  pod objects — is the durable run ledger.

#### Scenario: Successful workflow's pods are reclaimed
- **Given** a pipeline run whose workflow reaches phase `Succeeded`
- **When** the workflow completes
- **Then** its step pods are deleted shortly after (not resident for 30 days),
  and `spec.podGC.strategy` on the workflow is `OnWorkflowSuccess`

#### Scenario: Failed workflow's pods are retained
- **Given** a pipeline run whose workflow reaches a terminal `Failed` phase
- **When** the workflow completes
- **Then** its step pods remain available for diagnostics (`kubectl logs` /
  exit codes), and are cleaned up only later via the workflow object's TTL
