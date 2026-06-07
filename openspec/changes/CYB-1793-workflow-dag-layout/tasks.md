# Tasks — CYB-1793

## Context Files
- `Frontend/src/pages/WorkflowDagView.tsx` — graph layout and edge generation.
- `Frontend/src/pages/WorkflowDagNode.tsx` — node rendering and read-only handles.
- `Frontend/src/pages/WorkflowDagView.css` — execution DAG canvas and edge styling.
- `Frontend/src/pages/WorkflowDagNode.css` — node chrome and handle styling.
- `Frontend/src/pages/WorkflowDagView.test.tsx` — layout regression tests.
- `docs/agents/deploy-verification.md` — frontend dev deploy and Chrome DevTools MCP requirements.

## Implementation
- [x] [Frontend] Add a fan-in/join layout regression test using the complex DAG shape.
- [x] [Frontend] Post-process dagre layout so multi-input join nodes are centered relative to their direct visible predecessors.
- [x] [Frontend] Reduce edge clutter for join edges without changing dependency semantics.
- [x] [Frontend] Hide or de-emphasize React Flow handles in the read-only execution DAG.
- [x] [Frontend] Preserve node search, selection, action buttons, pan/zoom, and fitView behavior.

## Verification
- [x] [Frontend] `npm run test -- --run src/pages/WorkflowDagView.test.tsx`
- [x] [Frontend] `npm run lint`
- [x] [Frontend] `npm run build`
- [x] Deploy frontend dev Worker.
- [x] Chrome DevTools MCP verification on `/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`.
- [x] Chrome DevTools MCP console/network check has no new runtime errors.

### Verification Evidence — 2026-06-07

- Frontend unit test passed: `1` file, `4` tests.
- Frontend lint passed: Biome checked `235` files, no fixes applied.
- Frontend build passed; Vite emitted only existing large chunk warnings.
- Dev Worker deployed: `cyber-databrew-dev`, version `46345da4-2b70-4747-9099-7f5ba31396a5`.
- Chrome DevTools MCP DOM assertion on the complex DAG:
  - `nodeCount = 8`
  - `qa-left-b`, `qa-right-b`, `qa-mid-a` centerY values: `348.2`, `499.1`, `650.0`
  - average predecessor centerY: `499.1`
  - `qa-join-abc` centerY: `499.1`
  - `joinDeltaY = 0`
  - fan-in styled edge count: `3`
  - hidden handle count: `16`
  - visible or clickable handle count: `0`
- Chrome DevTools MCP content checks: status `Succeeded`, `DataBrew 运行`, `8` nodes, `52` events.
- Chrome DevTools MCP network checks: workflow detail APIs returned `200`; Cloudflare RUM returned `204`.
- Chrome DevTools MCP console checks: no runtime `error` or `warn`; remaining messages are browser issues for deprecated feature usage and form fields missing `id` / `name`.

## Documentation / Tracking
- [x] Record verification evidence in this `tasks.md`.
- [x] PR body references CYB-1793 and this OpenSpec change.
