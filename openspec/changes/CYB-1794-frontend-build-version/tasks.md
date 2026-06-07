# Tasks — CYB-1794

## Context Files

- `Frontend/vite.config.ts` — build metadata injection.
- `Frontend/src/lib/appVersion.ts` — version label helper.
- `Frontend/src/components/AppLayout.tsx` — global app shell.
- `Frontend/src/styles/design-tokens.css` — app layout styling.
- `docs/agents/deploy-verification.md` — frontend dev deploy and Chrome DevTools MCP requirements.

## Implementation

- [x] [Frontend] Extend app version helper to expose package version, short build ref, full build ref, and environment label.
- [x] [Frontend] Add a global lower-corner build marker in `AppLayout`.
- [x] [Frontend] Style the marker so it is visible, secondary, responsive, and does not block normal interactions.
- [x] [Frontend] Ensure deployed builds receive a useful commit/build reference through existing Vite metadata.
- [x] [Frontend] Keep existing Dashboard version display behavior compatible.

## Verification

- [x] [Frontend] `npm run lint`
- [x] [Frontend] `npm run test -- --run` for related helper/layout tests if present or added.
- [x] [Frontend] `npm run build`
- [x] Deploy frontend dev Worker with commit/build metadata.
- [x] Chrome DevTools MCP verifies the marker on dev.
- [x] Chrome DevTools MCP console/network check has no new runtime errors.

### Verification Evidence — 2026-06-07

- `cd Frontend && npm run test -- --run src/lib/appVersion.test.ts` -> pass, `4` tests.
- `cd Frontend && npm run lint` -> pass, Biome checked `237` files.
- `cd Frontend && VITE_APP_ENV=dev npm run build` -> pass; Vite emitted only existing large chunk warnings.
- Dev Worker deployed: `cyber-databrew-dev`, version `8e4a9fc9-54b9-45d5-8cb5-535a7fcb75ba`.
- Chrome DevTools MCP `/dashboard` checks:
  - global marker visible with text `v0.1.1 · a714077-dirty · dev`.
  - marker accessibility label: `前端版本 v0.1.1 · a714077-dirty · dev`.
  - popover details show version `0.1.1`, build `a714077-dirty`, environment `dev`.
  - desktop marker is rendered inside the sidebar bottom area after UX feedback.
- Chrome DevTools MCP workflow DAG checks:
  - marker visible on `/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`.
  - marker rect `12,813,196x24`, sidebar rect `0,0,220x851`, `insideSidebar=true`.
  - marker does not overlap React Flow controls: controls rect `1846,627,28x80`; overlap `false`.
  - mobile viewport check: marker rect `8,812,169x24`, `fitsViewport=true`.
  - key dashboard/workflow APIs returned `200`; Cloudflare RUM returned `204`.
  - console has no runtime `error` / `warn`; remaining message is an existing browser issue for a form field missing `id` / `name`.

## Documentation / Tracking

- [x] Record deploy and MCP evidence in this `tasks.md`.
- [ ] PR body references CYB-1794 and this OpenSpec change.
