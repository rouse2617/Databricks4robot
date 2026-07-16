# Spec Delta — Deploy threads the resolved argo client through every workflow op

## MODIFIED

### `POST /api/v1/deploy` — post-submit workflow ops

Fixes a routing gap in PR 4c: after `submitRuntimeWorkflow` routed to the target's cluster, four subsequent Argo calls in the same `Deploy` method (UID lookup, two rollback deletes, and first-phase status poll) still went to the process-global singleton — silent cross-cluster misroute on non-default targets.

Now Deploy:

1. Resolves the argo client ONCE at the start of the submit block via `resolveArgoClient(ctx, target)` — factory-first, singleton fallback.
2. Passes that same `client` into `submitRuntimeWorkflow(ctx, client, ...)`.
3. Uses `client.GetWorkflow` (via `getWorkflowWithUID`) for the runtime-config owner-reference lookup.
4. Uses `client.DeleteWorkflow` for both rollback paths (UID lookup failure, runtime-config Create failure).
5. Uses `client.GetWorkflowStatus` + `client.GetWorkflow` for first-phase status polling.

### Error mapping

- `resolveArgoClient` failure → `ErrWorkflowUnavailable: resolve argo client for cluster "<clusterID>": <cause>` returned by Deploy before any Argo call is made.
- `runtimeConfigProjection` requested but no client resolvable → `ErrWorkflowUnavailable: runtime config projection requires an argo client`.

## What stays UNCHANGED

- The public `POST /api/v1/deploy` request/response shape.
- Runtime adapter fallback for the adapter-only unit-test scenario (`argoFactory` nil AND `runtimeAdapter` set AND `wfClient` may or may not be set): adapter still wins on submit.
- Retry / Rerun / Resubmit / Terminate / Suspend / Resume — still on `uc.wfClient`; PR 4d target.
- `run_watcher` status polling — still on the singleton; PR 4d target.
