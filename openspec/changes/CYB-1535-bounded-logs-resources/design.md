# Design — CYB-1535 Bounded Logs And Resources

## Log Reads
Keep `GET /api/v1/workflows/{name}/logs?nodeId=...` but add bounded query parameters:
- `container`
- `tailLines`
- `limitBytes`
- `sinceSeconds`
- `sinceTime`
- `previous`
- `timestamps`

Defaults:
- `tailLines=200`
- `limitBytes=262144`
- `container=main`

Maximums:
- `tailLines <= 2000`
- `limitBytes <= 2097152`

Response shape:
```json
{
  "workflowName": "wf",
  "nodeId": "node",
  "podName": "pod",
  "container": "main",
  "source": "argo-live",
  "logs": "...",
  "lineCount": 200,
  "truncated": true,
  "nextCursor": null
}
```

Do not implement offset pagination over live Argo logs. Kubernetes and Argo log APIs support tail, since, and byte limits; stable cursors require persisted log chunks.

## Streaming
Register canonical:
- `GET /api/v1/workflows/{name}/logs/stream?nodeId=...`

Keep `/log/stream` only as a temporary compatibility alias if the frontend already calls it.

SSE events use JSON payloads:
```text
event: log
data: {"podName":"pod","container":"main","line":"..."}

event: heartbeat
data: {}

event: end
data: {"reason":"workflow-complete"}
```

The handler must set streaming headers, respect request cancellation, and close the upstream body. It must not fall back to full `ReadAll`.

## Resource Usage
Keep `GET /api/v1/deployments/{id}/resources` for compatibility but clarify the source of each value.

Add workflow-level endpoints:
- `GET /api/v1/workflows/{name}/resources`
- `GET /api/v1/workflows/{name}/nodes/{nodeId}/resources`

Response source metadata:
```json
{
  "source": {
    "workflow": "argo-live",
    "metrics": "unavailable",
    "spec": "stored-manifest"
  }
}
```

Running workflows read Argo live status. Completed workflows should use stored snapshots once `pipeline_run_nodes` exists. Live Kubernetes Metrics API is optional and must be explicitly marked as unavailable when absent.

## Risks
- Large logs can still overload proxies if SSE streams are left open too long.
- Argo/pod TTL can remove logs and resources before users inspect them.
- Promising pagination without persisted logs would be misleading.
- Current OpenAPI language overstates live CPU/memory usage.
