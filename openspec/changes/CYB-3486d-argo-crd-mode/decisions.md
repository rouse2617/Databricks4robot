# Decisions — CYB-3486d argo CRD mode

## D8 · CRD client must derive pod names (POD_NAMES=v2), not use node.ID

**Context.** Code review of the merged 4d series found `crdWorkflowClient`
using `node.ID` as the pod name in two places:

- `RetryWorkflow` — deletes stale failed pods by `node.ID`.
- `podsForLogs` (aggregate `GetWorkflowLogs`, empty podName) — reads pod logs
  by `node.ID`.

Argo v3.4+ defaults to `POD_NAMES=v2`, where a pod is named
`<workflow>-<template>-<suffix>` and is **not** equal to `node.ID` (which has no
template segment). `NodeStatus` carries no pod-name field, so the name must be
reconstructed. The rest of the backend already does this via
`handlers/workflow.resolveWorkflowPodName`; only the new CRD client diverged.

**Impact (CRD-mode clusters, e.g. `delivery-clust`).**

- Retry deleted a non-existent pod (NotFound, swallowed), leaving the real
  Failed pod in place → workflow-controller skips the retry → retry silently
  no-ops from the user's point of view.
- Aggregate log fetch resolved to non-existent pods → empty logs.

Cluster-default (cyber-clust) is unaffected — it runs through the argo-server
HTTP client, not the CRD client.

The original CRD unit tests hid the defect: they seeded pods named after the
raw `node.ID` and used node IDs without a template segment, so
`node.ID == podName` held in the fixture and never in production.

**Decision.**

1. Extract the v2 pod-naming algorithm into `argo.PodNameForNode` — one source
   of truth, in the lowest package (`argo`), reachable by both the CRD client
   and the handler (`handlers/workflow` already imports `argo`).
2. `handlers/workflow.resolveWorkflowPodName` now delegates to it (its
   TTL-cache wrapper is unchanged); the duplicated implementation is removed.
3. `RetryWorkflow` / `podsForLogs` resolve pod names through it.
4. Tests use realistic node IDs (`wf-1-2`) + `templateName`, so `podName`
   genuinely differs from `node.ID`; the retry test seeds a decoy pod named
   after the node ID and asserts it survives (locking in "delete by pod name,
   not node ID"). Verified the retry/logs tests fail against the old node.ID
   logic before the fix.

**Alternatives rejected.**

- *Set `POD_NAMES=v1` on CRD clusters* so `podName == node.ID`: fragile
  (per-cluster controller flag), and inconsistent with the rest of the backend
  which assumes v2.
- *Duplicate the algorithm inside `argo`*: leaves two copies to drift; the
  handler copy already had no unit test.
