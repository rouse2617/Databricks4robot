# Tasks — CYB-1795

## Context Files
- `Frontend/src/pages/WorkflowDagView.tsx` — React Flow DAG configuration and generated node/edge metadata.
- `Frontend/src/pages/WorkflowDagView.test.tsx` — DAG regression tests.
- `docs/agents/deploy-verification.md` — frontend dev deploy and Chrome DevTools MCP requirements.

## Implementation
- [x] [Frontend] Configure execution DAG nodes and edges so they do not advertise move/delete behavior.
- [x] [Frontend] Preserve node selection, node search, node action buttons, pan/zoom, and fit view.
- [x] [Frontend] Add focused tests for read-only node/edge metadata where practical.

## Verification
- [x] [Frontend] `npm run test -- --run src/pages/WorkflowDagView.test.tsx`
- [x] [Frontend] `npm run lint`
- [x] [Frontend] `npm run build`
- [x] Deploy frontend dev Worker.
- [x] Chrome DevTools MCP verification on `/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`.
- [x] Chrome DevTools MCP confirms no node/edge move/delete accessibility text.
- [x] Chrome DevTools MCP console/network check has no new runtime errors.

### Verification Evidence — 2026-06-07

- Frontend unit test passed: `1` file, `5` tests.
- Frontend lint passed: Biome checked `237` files, no fixes applied.
- Frontend build passed; Vite emitted only existing large chunk warnings.
- Dev Worker deployed: `cyber-databrew-dev`, version `f6fcbda4-c0b6-4128-8260-7e731995a2bb`.
- Chrome DevTools MCP URL: `https://cyber-databrew-dev.cyberorigin.ai/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`.
- Chrome DevTools MCP DOM assertion:
  - `hasMoveText = false`
  - `hasDeleteText = false`
  - `nodeCount = 8`
  - `edgeCount = 9`
  - `focusableNodes = 0`
  - `focusableEdges = 0`
  - node actions present: logs, runtime, IO, terminal, details
  - status/content present: `Succeeded`, `DataBrew 运行`, `8 个步骤`, `52`
- Chrome DevTools MCP interaction checks:
  - Node search for `qa-mid-a` filters the DAG card and keeps node action buttons available.
  - `qa-mid-a` detail action opens the node detail dialog.
- Chrome DevTools MCP network checks: workflow detail APIs returned `200`, including run detail, workflow detail, events, asset nodes, cost summary, and node logs.
- Chrome DevTools MCP console checks: no runtime `error` or `warn`; remaining messages are existing browser issues for deprecated feature usage and one form field missing `id` / `name`.
- Screenshot evidence saved outside the repo: `/tmp/cyb-1795-readonly-dag-dev.png`.

## Documentation / Tracking
- [x] Record deploy verification evidence in this `tasks.md`.
- [ ] PR body references CYB-1795 and this OpenSpec change.
