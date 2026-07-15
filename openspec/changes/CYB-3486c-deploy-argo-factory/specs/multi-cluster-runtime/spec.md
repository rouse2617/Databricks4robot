# Spec Delta — pipeline submit routes via argoFactory

## MODIFIED

### `POST /api/v1/deploy` — submit path

- **Adds** cluster-scoped Argo submission: when the Argo `ClientFactory` is
  wired on the pipeline usecase, `submitRuntimeWorkflow` resolves the Argo
  client via `factory.ForTarget(target)` (which picks up
  `ExecutionTarget.ClusterID` from the DB row) and calls `CreateWorkflow`
  through that client — instead of the process-global `wfClient`.
- **Empty `ClusterID`** on the target continues to work: the factory falls
  through to `cluster-default`, whose config is the env-derived Argo
  singleton — byte-identical to pre-3486.
- **Factory failure** (misconfigured cluster row, unreachable Argo, DB
  error) returns `ErrWorkflowUnavailable` wrapped with the cluster ID for
  operator triage.

## What stays UNCHANGED

- The public API contract of `/api/v1/deploy` — same request/response,
  same status codes.
- Run status polling (`GetWorkflow` in `SyncActiveRunEvents`) — still uses
  the singleton. PR 4d will route those by cluster.
- Terminate / Retry / Resubmit / Suspend / Resume — still on the singleton.
- The runtime adapter (`internal/runtimeos/adapter/argo`) — its
  constructor and interface are untouched; the shim is only exercised on
  the nil-factory fallback path.
