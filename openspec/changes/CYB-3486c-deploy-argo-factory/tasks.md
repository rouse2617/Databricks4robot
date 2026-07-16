# Tasks — CYB-3486 PR 4c: pipeline Deploy submit via argoFactory.ForTarget

## Landing PR (this iteration)

- [x] `usecase/pipeline.Usecase` gains `argoFactory argo.ClientFactory` field + `SetArgoFactory` setter
- [x] `submitRuntimeWorkflow` signature adds explicit `target *models.ExecutionTarget`
- [x] Factory branch at top of `submitRuntimeWorkflow` — resolves client via `factory.ForTarget(target)` and calls `CreateWorkflow` directly
- [x] Legacy adapter/singleton path preserved as fallback for nil-factory
- [x] `cmd/server/core.go` wires `puc.SetArgoFactory(inf.argoFactory)`
- [x] Sole caller (line 3444) updated to pass `target`
- [x] Helper `targetClusterID(*models.ExecutionTarget) string` for logging
- [x] 4 unit tests covering routing / empty-ID fallback / factory error wrap / no-factory legacy compat
- [x] `go test ./...` full suite green

## Verify plan (dev)

- [ ] Merge to dev → GHA deploy-dev green
- [ ] Submit a pipeline on default target (`cluster-default`) → workflow lands on cyber-clust; behavior byte-identical to pre-4c
- [ ] Attempt submit on a delivery-clust target → fails with clear `ErrWorkflowUnavailable: resolve argo client for cluster "cluster-delivery"` (expected until SRE prereqs done)

## Follow-ups

- **PR 4d** — `run_watcher` per-cluster goroutine (`GetWorkflow` polls also route by cluster)
- **PR 4e (opt)** — Retry/Rerun/Resubmit/Terminate on `uc.wfClient.*` also route by cluster
- **SRE prereqs**:
  - Cross-project WIF binding: dev backend Cloud Run SA → delivery-clust KSA
  - Expose delivery-clust Argo server externally; update clusters row `argoServerUrl`
- (opt) Frontend: cluster picker in Deploy modal makes intent explicit
