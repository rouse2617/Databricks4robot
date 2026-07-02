# Proposal — CYB-1556 Node Detail UI

## Problem

Workflow DAG nodes expose important diagnostics only after opening a detail drawer and switching across many tabs. The run dialog also uses implementation-heavy wording like "节点", which is abstract for users who think in pipeline steps.

## Approach

Refine the frontend only:

- Use user-facing wording such as "步骤" in pipeline version/run UI.
- Add compact diagnostic affordances directly on workflow DAG node cards.
- Highlight failure messages on failed/error node cards.
- Collapse the node detail drawer into fewer top-level sections: overview, logs, runtime, and input/output.
- Let node-card action icons open the drawer directly on the relevant section.

## Non-Goals

- No backend API changes.
- No new log streaming or pod exec backend work.
- No data model changes.
