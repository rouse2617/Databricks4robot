# CYB-1642 — Workflow Detail Full-Width Workbench Layout

## Problem

Workflow execution detail currently renders inside a centered `1400px` container. On wide desktop screens, the DAG, asset-node table, and event timeline appear compressed in the middle of the page with large unused whitespace on both sides.

This is caused by `WorkflowDetailPage.tsx` setting `maxWidth: 1400` and `margin: "0 auto"` on the page root.

## Goals

- Make workflow execution detail use the available desktop viewport width.
- Preserve the existing dense workbench structure: header, summary cards, DAG, node detail table, and event timeline.
- Keep compact internal padding so content remains scannable and does not touch the sidebar or viewport edge.
- Avoid changing backend APIs, data models, execution behavior, or routing.

## Non-Goals

- No redesign of the pipeline module.
- No changes to run events, cost, logs, Pod diagnostics, terminal, or Argo data fetching.
- No mobile-specific redesign.

## Scope

Frontend-only UI bugfix for the workflow execution detail page layout.

Primary file:

- `Frontend/src/pages/WorkflowDetailPage.tsx`

Optional supporting file if needed:

- `Frontend/src/styles/pipeline.css`

## Acceptance Criteria

- On wide desktop viewports, workflow detail content spans the available work area instead of being capped at 1400px.
- The DAG area, node table, and event timeline use the extra horizontal space.
- Existing summary cards and action buttons remain aligned and readable.
- Narrower desktop/tablet layouts do not regress.
- Chrome DevTools MCP verification on dev confirms no new console errors and no broken workflow detail layout.
