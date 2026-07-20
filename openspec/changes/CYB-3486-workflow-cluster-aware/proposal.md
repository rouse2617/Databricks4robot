# CYB-3486 — Workflow handler cluster-aware routing

## Why

The `/api/v1/workflows/*` handler (`backend/internal/handlers/workflow`) was
wired with a single, startup-injected Argo `WorkflowClient` (`New(wfClient,
namespace)`). That singleton always targets the **default** cluster
(cyber-clust argo-server). Any `/workflows/*` request for a workflow that lives
on a non-default cluster (e.g. delivery-clust) therefore fails with:

```
argo resource not found: {"code":5,"message":"workflows.argoproj.io \"<name>\" not found"}
```

because the HTTP call hits cyber-clust's argo-server, which does not know about
delivery-clust's workflow. The user-visible symptom is that **delivery-clust
runs show no logs** in the run detail Log tab — a blocker for on-call triage.

The sibling `/api/v1/runs/*` (PipelineRun) path is already cluster-aware
(`usecase/pipeline` `resolveArgoClientForRun` / `resolveRunClusterID`, PR #437).
The only remaining gap is `/workflows/*`, which the run detail page uses for
GetWorkflow / logs / log-stream / node-pod / terminal-session.

## What Changes

Make the workflow handler resolve, **per request**, which cluster a workflow
lives on (by its name → the owning `pipeline_run` → the run's execution target
→ `cluster_id`) and use the matching Argo client from `argo.ClientFactory`,
falling back to the injected singleton when the factory is not wired or the
cluster cannot be resolved.

- Inject `argo.ClientFactory` and `repository.ExecutionTargetRepository` into
  the handler (new setters `SetArgoFactory`, `SetExecutionTargetRepo`).
- Resolve the owning run's `cluster_id` with the same precedence as the proven
  `/runs/*` path: `run.ExecutionTarget.ClusterID`, else lookup by
  `ExecutionTargetID` (memoized), else `cluster-default`.
- Route every Argo call (`GetWorkflow`, `GetWorkflowLogs`,
  `GetWorkflowLogStream`, `ListWorkflows`, retry/resubmit/suspend/stop/resume/
  terminate/delete, node-pod GetWorkflow, terminal-session GetWorkflow) through
  the resolved client instead of `h.wfClient`.
- Resolve the namespace per run (already the case; refactored to share one run
  lookup per request).
- Node pod diagnostics: resolve the per-cluster K8s client from the existing
  `k8s.ClientFactory` so `/nodes/:nodeId/pod` reads the right cluster.

**Out of scope (follow-up):** the interactive Pod terminal *exec* attach still
uses the default-cluster `ExecClient`. Making it cluster-aware requires the
`k8s.ClientFactory` to expose a per-cluster `*rest.Config` (SPDY), which is new
infrastructure; deferred. The terminal-session *creation* (workflow/pod
validation) is routed correctly by this change.

## Impact

- Affected specs: `pipeline` (workflow monitoring behavior).
- Affected code: `backend/internal/handlers/workflow/*`, `backend/cmd/server/core.go`.
- Backward compatible: factory-nil (no-PG test infra) and default-cluster
  workflows behave exactly as before (singleton fallback).
- No API contract change: routes, request/response shapes, and status codes are
  unchanged — this is an internal routing fix, not a new/changed HTTP surface.
