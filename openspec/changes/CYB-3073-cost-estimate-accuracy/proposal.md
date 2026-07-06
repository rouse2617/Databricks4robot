# Proposal — CYB-3073

## Why

The batch "总成本" (estimated cost) over-estimates: (1) `ComputeRunCost` sums the DAG rollup node on top of its child pods (double-count); (2) every node is priced at the GPU pool rate ($1.20/h) because the real node instance type is never captured, so CPU-only steps are ~13× too high. Found on a real run: shown $0.39 vs true ~$0.23 (Pod-only), and CPU steps mispriced.

## What Changes

### Modified Capabilities
- **runtime-os**: A pipeline run's estimated cost sums only leaf (Pod) nodes, not aggregate DAG/Steps nodes — so the total matches the per-step breakdown.
- **runtime-os**: Per-node cost is priced by the node's real machine type / accelerator / provisioning, not a fixed GPU-pool default — so CPU work is priced at CPU rates.

## Impact
- **Affected code**: `backend/internal/usecase/pipeline/cost.go` (ComputeRunCost filter), `backend/internal/usecase/pipeline/usecase.go` (node cost derivation ~1867 + a node instance-profile resolver), `backend/internal/k8s` (node label lookup), `backend/cmd/server` (wire resolver).
- **New APIs**: none.
- **Dependencies**: none new. **RBAC**: backend SA needs `get/list nodes` (cluster-scoped) for Bug 2.

## Scope
- **In scope**: (Bug 1) ComputeRunCost + any total aggregation sum Pod leaves only; (Bug 2) resolve real node instance profile from `HostNodeName` → K8s node labels (`node.kubernetes.io/instance-type`, `cloud.google.com/gke-accelerator`, `cloud.google.com/gke-provisioning`), cached, fed into `resourcesDurationToCost`.
- **Out of scope**: CUD/SUD, network/storage costs, prod rollout, changing the pricing table values.

## Success Criteria
- [ ] A run's total cost equals the sum of its per-step (Pod) costs shown in the UI.
- [ ] A CPU-only step is priced at its CPU node rate, not the GPU rate.
- [ ] Unknown/looked-up-miss nodes degrade gracefully (fall back to prior default, never error).
- [ ] Estimate stays best-effort; no crash if K8s node lookup fails.

## Rollout note
Historical run totals will drop (both fixes reduce the estimate). Bug 1 (no RBAC) can ship first; Bug 2 gated on the `get nodes` RBAC grant.
