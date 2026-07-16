# Tasks — CYB-3486 PR 4c.1: thread argo client through Deploy downstream ops

## Landing PR (this iteration)

- [x] `Usecase.resolveArgoClient(ctx, target) (argo.WorkflowClient, error)` — factory-first / singleton fallback / nil-safe
- [x] `submitRuntimeWorkflow(ctx, client, ...)` — takes explicit client; internal factory lookup removed
- [x] `getWorkflowWithUID(ctx, client, name, namespace)` — takes explicit client
- [x] `Deploy` resolves `client` once, threads through submit + getWorkflowWithUID + 2× DeleteWorkflow + GetWorkflowStatus + GetWorkflow
- [x] Adapter-only unit-test path preserved via `argoFactory == nil && runtimeAdapter != nil` branch in submitRuntimeWorkflow
- [x] `TestGetWorkflowWithUID_UsesProvidedClient` guardrail against future singleton regression
- [x] `submit_routing_test.go` rewritten around `resolveArgoClient` (5 tests) + `submitRuntimeWorkflow` (2 tests) + getWorkflowWithUID (1 test) = 8 tests total
- [x] `go test ./...` full suite green

## Verify plan (dev)

- [ ] Merge to dev → GHA deploy-dev green
- [ ] `POST /api/v1/deploy` on the default target → default cluster (byte-identical)
- [ ] `POST /api/v1/deploy` on a delivery-clust target → fails at `resolveArgoClient` with the cluster-tagged error; no partial state left on either cluster

## Follow-ups

- **PR 4d** — Retry / Rerun / Resubmit / Terminate / run_watcher all still use `uc.wfClient`. Same class of fix, larger scope. Coming next.
