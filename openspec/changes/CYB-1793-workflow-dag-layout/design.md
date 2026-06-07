# Design — CYB-1793

## Architecture Context
- **Constraints**: The execution DAG currently uses React Flow plus `dagre`; this change should stay within the current stack unless proven insufficient.
- **Goals**: Make fan-in joins visually intentional, reduce edge clutter, and remove edit-mode cues from read-only workflow detail.
- **Non-Goals**: Do not introduce real data-flow semantics. Do not change pipeline designer `input` / `output` ports.

## External UI Notes
- Airflow separates operational run scanning from dependency visualization: Grid View is the primary status matrix, while Graph View shows task dependencies and run-specific task state.
- Airflow's Graph View framing is the relevant precedent for DataBrew's execution DAG: each node is a task, edges are dependencies, and click actions drill into metadata/logs.
- Argo's DAG model is dependency-first: tasks declare dependencies rather than visual data-flow ports.

## Affected Modules
- `Frontend/src/pages/WorkflowDagView.tsx` — layout, edge generation, and graph metadata.
- `Frontend/src/pages/WorkflowDagNode.tsx` — read-only node affordances.
- `Frontend/src/pages/WorkflowDagView.css` / `WorkflowDagNode.css` — visual treatment of edges, handles, and convergence.
- `Frontend/src/pages/WorkflowDagView.test.tsx` — deterministic fan-in regression coverage.

## Architecture Decisions

### Decision 1: Keep React Flow and dagre, add fan-in post-processing
- **Approach**: Use existing dagre ranks, then post-process visible nodes with multiple incoming edges so their `y` coordinate is centered over their visible direct predecessors.
- **Alternative**: Replace dagre with ELK.js or another layered graph engine.
- **Rationale**: The current issue is not general graph layout failure; it is a specific fan-in readability problem. A focused post-process is smaller and easier to regression-test.
- **Trade-off**: Complex nested subgraphs may still need a stronger layout engine later.
- **Rollback**: Remove the post-process and revert to raw dagre positions.

### Decision 2: Treat execution DAG as read-only status graph
- **Approach**: Hide or visually suppress React Flow handles in `WorkflowDagNode` when used by execution detail; keep node selection and action buttons.
- **Alternative**: Leave handles visible because they are harmless.
- **Rationale**: Handles imply editable ports/edge creation. In workflow detail they do not help diagnose execution state and contribute to visual noise.

### Decision 3: Prefer readable dependency edges over data-flow symbolism
- **Approach**: Tune edge type/style for the execution DAG so incoming join edges visually converge without excessive right-angle detours.
- **Alternative**: Add explicit synthetic join nodes.
- **Rationale**: Synthetic nodes can make the graph look more complex than the workflow actually is. The real join node already exists and should be visually emphasized.

## Data Flow

```text
workflow.nodes + workflow.edges
  -> filter displayable nodes
  -> build dependency edges
  -> dagre rank layout
  -> fan-in join y-centering
  -> React Flow read-only render
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Centering a join node may overlap another node in dense graphs | Visual clutter | Clamp or shift only when the rank has enough spacing; keep tests focused on known fan-in case |
| Hiding handles could make edge endpoints less obvious | Users may need direction cues | Keep arrowheads and subtle edge endpoint styling |
| Edge changes could affect screenshots users rely on | Minor UI adjustment | Verify on known simple and complex DAGs with Chrome DevTools MCP |
