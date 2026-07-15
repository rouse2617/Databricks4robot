# CYB-3486 PR 4c — pipeline Deploy submit via argoFactory.ForTarget

## Why

PR 4b flipped `elastic_quota` + `resource_quota` to the K8s factory. But the primary reason multi-cluster exists — **letting a pipeline actually submit to `delivery-clust`** — was still doing the wrong thing:

- `submitRuntimeWorkflow` in `usecase/pipeline/usecase.go:3555` called `uc.runtimeAdapter.Submit(...)` or `uc.wfClient.CreateWorkflow(...)`. Both use the process-wide Argo singleton bound at startup.
- `ExecutionTarget.ClusterID` was persisted (PR 1) but **not read anywhere in the submit path** (Explore confirmed).
- A pipeline whose target has `ClusterID=cluster-delivery` would still submit to `cyber-clust`. Silent misroute.

## What Changes

- `usecase/pipeline.Usecase` gets `argoFactory argo.ClientFactory` field + `SetArgoFactory` setter.
- `submitRuntimeWorkflow(ctx, target, runID, runName, wf, namespace)` — target is now an explicit parameter, no longer implicit via singleton.
- New factory branch at the top of `submitRuntimeWorkflow`:
  - When factory is wired, resolve the argo client via `factory.ForTarget(target)` and call `client.CreateWorkflow` directly.
  - Adapter path preserved for the nil-factory fallback (unit tests without a wired factory + any code path that hasn't yet been migrated).
- `cmd/server/core.go` wires `puc.SetArgoFactory(inf.argoFactory)`.

## What NOT in this PR

- `run_watcher` status polling still hits the singleton (`uc.wfClient.GetWorkflow` at lines 2327/2567/2641/2701). That's PR 4d — one goroutine per active cluster, not one goroutine total.
- Retry / Rerun / Resubmit / Terminate — all still route through `uc.wfClient` methods. They enter via `CreateRun` which does NOT call submit, so they need their own PR (call it PR 4d.2) to be per-cluster aware. For now they'll only work on the default cluster.
- The runtime adapter (`internal/runtimeos/adapter/argo`) doesn't grow a factory yet — the shape-translation shim keeps its old constructor so nothing else has to move.

## Deploy prerequisites (SRE) — NOT part of this PR

Any pipeline submitting with a non-default `ClusterID` will hit these when this PR lands:

1. **Cross-project WIF binding**: dev backend Cloud Run SA (green-valley-442103) needs `iam.workloadIdentityUser` on the target-cluster KSA (e.g. `cyber-delivery-prod/workflow-runner`).
2. **Argo server externally reachable**: `argoServerUrl` in the `clusters` row currently points at `argo-workflows-server.<ns>.svc.cluster.local:2746` — cluster-internal DNS. Backend Cloud Run cannot resolve it. delivery-clust needs a LoadBalancer / ingress on its Argo server + the clusters row updated to that public URL.

Until both land, submitting to delivery-clust will fail at first `CreateWorkflow` call with a network error. That's OK — this PR ships the code seam so the day SRE finishes the binding, we can hit deploy immediately.

## Compat matrix

| State | Behavior |
|-------|----------|
| `argoFactory == nil` (no PG, tests) | Legacy adapter/singleton path — byte-identical to pre-3486 |
| Factory wired, `target.ClusterID == ""` | Route via factory to `cluster-default` (env fallback) — functionally identical to legacy |
| Factory wired, `target.ClusterID == cluster-default` | Same as above |
| Factory wired, `target.ClusterID == cluster-delivery` | Route to delivery-clust argo server (fails until SRE prereq done) |

## Verify plan

- [x] Unit tests:
  - `TestSubmitRuntimeWorkflow_FactoryRoutesByTarget` — non-default ClusterID hits the right client, singleton NOT called
  - `TestSubmitRuntimeWorkflow_FactoryEmptyClusterIDFallsToDefault` — empty ClusterID → cluster-default
  - `TestSubmitRuntimeWorkflow_FactoryErrorWrapped` — factory error → `ErrWorkflowUnavailable`
  - `TestSubmitRuntimeWorkflow_NoFactoryFallsToLegacy` — nil factory still runs adapter/singleton
- [x] `go build ./... && go test ./...` clean
- [ ] Dev deploy → submit a pipeline on the default target → runs on cyber-clust as before
- [ ] Dev deploy → attempt submit on a delivery-clust target → fails with a clear "resolve argo client" error (expected, not a regression)
