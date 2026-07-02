# Design — CYB-1609

## Architecture Context
- **Constraints**: Current live logs come from Argo/Kubernetes log APIs. They support tail, since, timestamps, and byte limits, but they do not provide stable historical offset pagination for completed pod logs.
- **Goals**: Prevent unbounded log reads, make UI state obvious, and avoid implying a cursor exists when the backend cannot produce one.
- **Non-Goals**: Persisting log chunks, indexing log lines, integrating Cloud Logging, or building cross-cluster log aggregation.

## Affected Modules
- `backend/internal/argo` — preserve bounded log options and expose any supported cursor/window metadata from log reads.
- `backend/internal/handlers/workflow` — validate log query parameters, return pagination/truncation metadata, and keep SSE cancellation-safe.
- `api/openapi.yaml` — document extended query/response metadata and error cases.
- `docs/review/api-guide.md` — add curl examples for bounded windows, follow, download, and unavailable pagination.
- `Frontend/src/api/workflowApi.ts` — type the log response metadata and URL builder parameters.
- `Frontend/src/pages/useWorkflowDetail.ts` — manage selected node, loaded windows, follow lifecycle, download, and error state.
- `Frontend/src/pages/WorkflowDetailPage.tsx` — render a compact log workbench with clear controls and bounded rendering.
- `Frontend/src/pages/WorkflowDagNode.tsx` — expose failed-node one-click log action.

## Architecture Decisions

### Decision 1: Cursor metadata must be truthful
- **Approach**: Return cursor/window metadata only when the backend has a meaningful next request. For live Argo/Kubernetes logs, historical pagination can remain unavailable and the response must say so through metadata and UI copy.
- **Alternative**: Invent an offset cursor by counting lines in the returned payload.
- **Rationale**: Offset cursors over live pod logs are unstable after rotation, truncation, pod deletion, or timestamp changes; fake cursors create data loss and duplicate lines.
- **Trade-off**: The first version may support "load newer/follow" better than "page all historical logs" until persisted log chunks exist.
- **Rollback**: Keep existing `logs` string response fields compatible so older frontend code can still render bounded tail logs.

### Decision 2: UI renders bounded chunks, not a giant textarea
- **Approach**: Keep loaded chunks in a bounded client buffer, show truncation metadata, and download from loaded content or a bounded backend response.
- **Alternative**: Append every SSE line forever into one string.
- **Rationale**: A single growing string is the failure mode that makes 100w-line logs freeze the browser.
- **Risk**: Users may expect full historical logs from the browser.
- **Mitigation**: Copy must explain whether they are viewing tail, streamed lines, or a limited window; future archived logs can add full download.

### Decision 3: Follow state is explicit
- **Approach**: Model follow as `idle | connecting | connected | ended | error`, render visible state, and allow manual stop/reconnect.
- **Alternative**: Boolean `following` only.
- **Rationale**: Current interaction can fire an SSE request without giving the user visible feedback when the stream closes quickly.

## Data Flow

```mermaid
flowchart LR
  User[User selects node] --> UI[Workflow log viewer]
  UI --> API[GET /workflows/{name}/logs]
  API --> Argo[Argo/Kubernetes log API]
  Argo --> API
  API --> UI
  UI --> SSE[GET /workflows/{name}/logs/stream]
  SSE --> UI
```

## Data Model Changes
- No Postgres schema change planned.

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Argo cannot provide stable historical cursors | Load-more may be limited | Return honest metadata and defer persisted log chunks |
| Large SSE stream grows client memory | Browser freezes | Cap displayed buffer and show dropped-line/truncated hints |
| Completed pods are cleaned by TTL | Logs unavailable | Surface unavailable state and keep DataBrew events as fallback context |
| Download can be mistaken for full archive | User misses older lines | Label downloads as current loaded/bounded window unless archive exists |
