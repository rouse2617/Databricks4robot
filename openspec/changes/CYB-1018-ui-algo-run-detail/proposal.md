# Proposal — CYB-1018 UI

## Why

Backend `algo_runs` is on `dev` but asset detail still hides `run_id` on the algo tab. Operators cannot connect per-asset algorithm state to a registered run without reading raw JSON or API docs.

## What Changes

- New `RunIdLink` component: monospace id, popover summary from `GET /algo-runs/{id}`
- Algo tab: **来自 run** column when `run_id` is a registered 16-char id
- Version provenance: **触发 run** uses `RunIdLink` for registered ids; legacy ids unchanged
- Algo matrix popover: reuse `RunIdLink` for Run ID row
- Design spec: [`docs/review/unified-asset-catalog/design/algo-runs-ui.md`](../../../docs/review/unified-asset-catalog/design/algo-runs-ui.md) (ui-ux-pro-max)

## Scope

- **In**: Frontend only; no backend changes
- **Out**: Dedicated algo-runs list/monitor pages (P1.5)

## Success Criteria

- [ ] Asset with `hand_track@x:run_id` shows link; popover loads run row
- [ ] Invalid/legacy short `run_id` renders as plain text (no broken API call)
- [ ] Dev UI verified; console clean on sample assets
