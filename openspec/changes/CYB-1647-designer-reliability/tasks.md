# Tasks — CYB-1647

## Context Files
- `/tmp/cyber-databrew-pipeline-ux-deep-test-20260607.md` — source UX report and reproduction evidence.
- `Frontend/src/pages/PipelinePage.tsx` — designer name input, add component handler, deploy controls, and canvas.
- `Frontend/src/pages/PipelinePage.test.tsx` — designer regression tests.
- `Frontend/src/components/pipeline/ComponentPalette.tsx` — component palette item rendering and actions.
- `Frontend/src/styles/pipeline.css` — palette item layout and visible add affordance.
- `docs/agents/deploy-verification.md` — frontend dev deploy and Chrome DevTools MCP verification requirements.

## Implementation
- [x] [Frontend] Track whether the current pipeline name is still the generated default and only auto-replace in that state.
- [x] [Frontend] Remove or reduce heuristic append-repair behavior once generated-name replacement has an explicit state model.
- [x] [Frontend] Add a visible add action to component palette items while keeping drag/drop support.
- [x] [Frontend] Ensure the visible add action creates the exact selected component and keeps the new node in a predictable visible position.
- [x] [Frontend] Update empty-canvas or palette copy if needed so users understand they can click-add or drag.
- [x] [Frontend] Add focused tests for generated-name replacement, preserving user-edited names, and click-add exact component insertion.

## API Contract Sync
No HTTP API is added or changed in this change. Existing pipeline component,
template, and run APIs remain unchanged.

## Local Verification
- [x] [Frontend] Run focused tests for `PipelinePage`.
- [x] [Frontend] `npm run lint`
- [x] [Frontend] `npm run build`

## Deploy Verification
- [x] [Frontend] Deploy frontend dev Worker with `wrangler deploy --env dev`.
- [x] [Frontend] Chrome DevTools MCP on dev: designer generated-name replacement.
- [x] [Frontend] Chrome DevTools MCP on dev: visible component add action creates the selected component.
- [x] [Frontend] Chrome DevTools MCP console/network check has no new runtime errors.
- [x] [Frontend] Record deployed Worker version and tested URLs in this file before commit.

### Deploy Evidence — 2026-06-07
- Worker: `cyber-databrew-dev`
- Version ID: `95e6e9eb-8f5f-430c-887a-7644ffae3d98`
- Source SHA: `f04d3d9`
- URL: `https://cyber-databrew-dev.cyberorigin.ai/pipeline`
- Chrome DevTools MCP:
  - Version badge: `v0.1.1 · f04d3d9-dirty · dev`
  - Generated name edit: `pipeline-*` replaced with `qa-cyb1647-mcp-name`
  - Click add: `添加组件 Echo Message` created selected node `Echo Message`
  - Network: `/pipeline`, `/api/v1/auth/me`, `/api/v1/pipeline-components`, `/api/v1/execution-targets` all `200`
  - Console: no error/warn; one existing DevTools form-field issue
  - Screenshot: `/tmp/cyb-1647-dev-verify-pipeline-designer.png` (not committed)

## PR
- [ ] [Frontend] PR template filled; Linear CYB-1647 and OpenSpec change ID linked.
