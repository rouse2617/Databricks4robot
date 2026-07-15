# Workflow logs — SSE streaming

## MODIFIED: only follow logs while the node is running

The SSE log endpoint (`GET /api/v1/workflows/:name/logs/stream`) previously
forced follow (live-tail) mode for every request (`GetWorkflowLogStream`
hardcoded `Follow=true`). Following a node whose pod has finished never receives
new lines and never EOFs through Argo's follow API, so the request hung until
the Cloud Run request timeout (~60s) → 504, and the browser client
auto-reconnected, exhausting the per-host connection pool and stalling the UI.

### Requirement: follow is gated on node liveness

The stream MUST live-tail (`Follow=true`) only when the requested node is still
running. When the node is fulfilled (finished/skipped/omitted, per Argo
`NodeStatus.Fulfilled()`), the stream MUST fetch logs once (`Follow=false`):
Argo returns the existing logs then EOF, and the SSE emits its terminal `end`
(`stream-complete`) event and closes promptly (~1–2s) instead of hanging.

Liveness MUST be evaluated at the **node** level (`workflow.Status.Nodes[nodeID]`),
not the workflow level — a running workflow may contain an already-finished node
whose logs must not be followed.

### Requirement: the shared client respects the caller's follow flag

`GetWorkflowLogStream` MUST honor the `Follow` value passed by the caller and
MUST NOT force it on. The SSE handler is the component that decides follow based
on node liveness.
