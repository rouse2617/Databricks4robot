# Design — CYB-1344

## Architecture Context
- **Constraints**: Workflow detail data is served through the existing `GET /api/v1/workflows/{name}` endpoint; frontend DAG rendering uses React Flow.
- **Goals**: Make workflow graph topology explicit in the API, preserve omitted-step dependencies, and keep React Flow as a thin renderer.
- **Non-Goals**: Porting Argo UI's full GraphPanel, adding artifact graph rendering, or changing Argo workflow execution behavior.

## Affected Modules
- `backend/internal/handlers/workflow/handler.go` — add normalized workflow DAG edge response data.
- `api/openapi.yaml` — document the new `WorkflowDagEdge` schema and `WorkflowDetail.edges`.
- `docs/review/api-guide.md` — document the workflow detail edge contract.
- `Frontend/src/api/workflowApi.ts` — add typed edge contract.
- `Frontend/src/pages/WorkflowDagView.tsx` — render backend-provided edges before local fallback inference.

## Architecture Decisions

### Decision 1: Backend owns workflow DAG edge normalization
- **Approach**: Add `edges` to the workflow detail response. Each edge has `id`, `source`, `target`, and `kind` (`runtime`, `dag`, or `fallback`).
- **Alternative**: Return only raw Argo `children`, `boundaryID`, and `outboundNodes` and let the frontend reconstruct topology.
- **Rationale**: The backend already adapts Argo's response into the product API. Keeping topology normalization there prevents every UI consumer from reimplementing Argo-specific graph semantics.
- **Trade-off**: The workflow API contract grows, and backend tests must cover graph edge cases.
- **Rollback**: Frontend retains local edge inference as fallback; removing `edges` from the response reverts to current behavior.

### Decision 2: Borrow Argo UI concepts, not its graph renderer
- **Approach**: Use Argo UI's `children`, `boundaryID`, and `outboundNodes` concepts as reference behavior while continuing to render with React Flow.
- **Alternative**: Copy Argo UI `GraphPanel` and its graph layout/filter/collapse stack.
- **Rationale**: Our UI already uses React Flow and has a simpler workflow detail surface. Copying Argo's renderer would bring unrelated collapse, artifact, icon, and layout complexity.
- **Trade-off**: We implement a smaller normalization layer instead of inheriting all upstream UI behavior.

## Data Flow

```text
Argo Workflow NodeStatus
  -> backend workflow handler normalizes display nodes + DAG edges
  -> GET /api/v1/workflows/{name}
  -> frontend typed WorkflowDetail
  -> React Flow nodes + backend edges
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Edge normalization misses an Argo node shape | Some workflows still lack edges | Keep existing frontend fallback and add focused unit tests for failed/omitted and normal chains |
| Hidden root/group nodes produce dangling edges | React Flow may render off-node edge fragments | Compress edges to visible source/target nodes and discard unresolved endpoints |
| API shape drift between backend, OpenAPI, and frontend | Runtime UI mismatch | Update OpenAPI, frontend types, API guide, and tests in the same change |
