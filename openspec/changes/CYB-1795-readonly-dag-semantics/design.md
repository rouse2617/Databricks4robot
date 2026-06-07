# Design — CYB-1795

## Architecture Context
- **Constraints**: Workflow execution detail uses `@xyflow/react` for the DAG and custom `WorkflowDagNode` cards for node actions.
- **Goals**: Make the execution DAG accurately announce as a read-only inspection graph while preserving current diagnostics interactions.
- **Non-Goals**: Do not alter graph topology, layout, execution data, pipeline designer behavior, or backend APIs.

## Affected Modules
- `Frontend/src/pages/WorkflowDagView.tsx` — React Flow graph configuration and generated node/edge metadata.
- `Frontend/src/pages/WorkflowDagView.test.tsx` — focused regression coverage for read-only node/edge semantics.

## Architecture Decisions

### Decision 1: Configure React Flow as a read-only inspection graph
- **Approach**: Use React Flow read-only controls such as disabled node/edge focusability, disabled keyboard edit affordances, and custom accessibility labels where needed.
- **Alternative**: Leave React Flow defaults because visual editing is already disabled.
- **Rationale**: The visual graph is already read-only, but keyboard/a11y metadata still communicates edit-mode behavior. Semantics need to match product behavior.

### Decision 2: Preserve selection and embedded action controls
- **Approach**: Keep node click selection and embedded node action buttons, while preventing the graph container from advertising node movement or deletion.
- **Alternative**: Disable all selection and focus within React Flow.
- **Rationale**: Operators need to inspect logs, runtime, IO, terminal, and details from node cards. The fix should remove misleading edit semantics, not reduce diagnostics.

### Decision 3: Verify against the existing complex DAG run
- **Approach**: Reuse the dev regression workflow `qa-complex-dag-20260607041627-e3d3c7` because it exercises fanout/fanin topology and all node action affordances.
- **Alternative**: Create a new workflow run.
- **Rationale**: The issue is frontend semantics, not execution correctness. A known successful run gives stable visual and network expectations.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Disabling graph focusability also hides useful node action focus paths | Keyboard users lose diagnostics access | Verify action buttons remain in the MCP accessibility snapshot |
| React Flow defaults may still inject hidden descriptions | Snapshot still mentions move/delete | Add explicit aria label overrides or per-node/edge metadata until descriptions match read-only behavior |
| Over-disabling keyboard behavior may break selection | Node detail no longer opens | Keep click selection and verify with MCP on deployed dev |
