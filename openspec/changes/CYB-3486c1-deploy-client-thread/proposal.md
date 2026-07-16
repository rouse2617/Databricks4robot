# CYB-3486 PR 4c.1 — thread the resolved argo client through Deploy's downstream ops

## Why

PR 4c (already merged as #404) flipped `submitRuntimeWorkflow` to route through the argo factory when a target has a non-default `ClusterID`. But the same `Deploy` method (`usecase/pipeline/usecase.go:3472-3496`) had four more Argo calls right after the submit that still hit `uc.wfClient` — the process-global singleton:

1. `getWorkflowWithUID` → `wfClient.GetWorkflow` — reads back the just-submitted workflow's UID for the runtime-config owner reference
2. `wfClient.DeleteWorkflow` — cleanup if UID lookup fails
3. `wfClient.DeleteWorkflow` — cleanup if the runtime config projection Create fails
4. `wfClient.GetWorkflowStatus` + `wfClient.GetWorkflow` — first-phase polling

Consequence for a delivery-clust target: submit lands on delivery-clust ✓ but the UID read hits cyber-clust (returns nothing, config-owner reference build fails); the "rollback" delete also fires against cyber-clust (no-op), leaving a zombie workflow on delivery-clust.

Spotted by `gemini-code-assist` in the [PR #404 review](https://github.com/CyberOrigin2077/cyber-databrew/pull/404#pullrequestreview-4702594048). Silent misroute — would not surface until an SRE prereq (cross-project WIF + external argo URL) is done and someone submits on delivery-clust. No live victims yet on dev; still land the fix now so day-of-SRE-completion is safe.

## What Changes

- New helper `Usecase.resolveArgoClient(ctx, target)` returns the argo client for the target's cluster — factory-first, singleton fallback, nil-safe for adapter-only test paths.
- `submitRuntimeWorkflow` now takes an explicit `client argo.WorkflowClient` parameter. It no longer resolves the factory internally — that logic moved up so the SAME client can be reused for every downstream op.
- `getWorkflowWithUID` now takes an explicit `client argo.WorkflowClient` parameter (previously read `uc.wfClient` directly).
- `Deploy` resolves `client` once via `resolveArgoClient(ctx, target)` and threads it through:
  - `submitRuntimeWorkflow(ctx, client, ...)`
  - `getWorkflowWithUID(ctx, client, ...)`
  - Both `client.DeleteWorkflow` rollback calls
  - `client.GetWorkflowStatus` and `client.GetWorkflow` first-phase polling

## What NOT in this PR

- Retry / Rerun / Resubmit / Terminate paths — still on `uc.wfClient` in their own places (`usecase.go:4118 / 4167 / 4920`). Same class of misroute; PR 4d target.
- `run_watcher` status polling — separate goroutine, PR 4d.
- Argo adapter interface — its constructor is unchanged; the shim is still exercised in the adapter-only unit-test path.

## Compat / test matrix

| Wiring | `resolveArgoClient` | `submitRuntimeWorkflow(client)` |
|--------|--------------------|--------------------------------|
| `argoFactory` set (production) | Factory-resolved client | `client.CreateWorkflow` (direct) |
| `argoFactory` nil, `wfClient` set (legacy) | Singleton | Same client (adapter branch skipped because factory is nil AND there may not be one) |
| `argoFactory` nil, adapter set, `wfClient` nil (adapter-only tests) | Returns nil | Adapter branch fires (adapter wins when factory is nil AND adapter is set) |
| Both `argoFactory` and adapter set (unlikely) | Factory-resolved client | Factory client path takes over — adapter is bypassed on the submit-path |
| Nothing wired | Returns nil | `ErrWorkflowUnavailable` |

## Verify plan

- [x] `go build ./...` clean
- [x] `go test ./...` full suite green — including `TestDeploy_RuntimeAdapterSubmitPreservesRuntimeConfigOwnerLookup` (adapter-only path preserved)
- [x] 7 tests in `submit_routing_test.go`:
  - 5 for `resolveArgoClient` (routing / empty ID / error / no-factory / no-nothing)
  - 2 for `submitRuntimeWorkflow` (uses-provided-client / nil-client-adapter-fallback)
  - 1 new for `getWorkflowWithUID` (explicit guardrail against Gemini's finding)
- [ ] Dev deploy → default-target pipeline submit still works
- [ ] Dev deploy → attempt on a delivery-clust target: fails cleanly with `resolve argo client for cluster "cluster-delivery"` (expected until SRE prereq done, but now downstream ops also error consistently instead of misrouting)
