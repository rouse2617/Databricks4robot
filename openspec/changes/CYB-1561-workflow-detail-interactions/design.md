# Design

## Current Interaction

On `/pipeline/executions/:name`, users can click a DAG node. The page opens a node detail drawer. For logs, the DAG node action currently selects the "logs" tab in the drawer, then users click "open log viewer" to see the actual log content.

This creates a nested interaction:

1. DAG node action
2. Drawer log tab
3. Log modal

## Proposed Interaction

Use action-specific behavior:

- Clicking the node body opens the summary drawer.
- Clicking the log icon opens the log viewer directly for that node.
- Clicking runtime / input-output / summary icons opens the drawer on the matching tab.

The selected node remains highlighted while either the drawer or log modal is open.

## Large Log Viewer

The existing frontend-side render cap remains unchanged. The log viewer adds:

- Search input with stable `id` and `name`.
- Visible rendered character and total line count.
- "Copy visible logs" action for the currently rendered log tail.
- Taller modal body so the log area is the main focus.

This is intentionally not a replacement for backend tail / pagination / follow mode.

## Placeholder Panels

The asset-node details area is backend-dependent. Until data exists, show it as a compact placeholder strip instead of a full empty table. This preserves the future concept but gives the DAG more vertical room.

## Risks

- Decoupling node selection from drawer visibility can regress drawer open/close behavior.
- The direct log action must still load logs from the selected node and keep the node highlighted.
- Copy-to-clipboard can fail in restricted browser contexts; the UI should fail gracefully if this appears during verification.
