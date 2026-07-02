# CYB-1561 Workflow Detail Interaction Polish

## Problem

The workflow execution detail page already opens to the DAG view, but several interactions add friction:

- The DAG node "view logs" action opens the node drawer first, then requires another click to open the log viewer.
- The large-log modal lacks quick metadata and a copy affordance for the currently rendered log tail.
- Placeholder sections such as asset-node details consume screen space before backend data is available.
- Some search inputs are missing stable `id` / `name` attributes, producing browser accessibility issues.

These issues make the core debugging workflow slower even though the underlying logs and node details are available.

## Goals

- Make viewing node logs a one-click action from the DAG node.
- Keep the node drawer available for summary, runtime, and input/output details.
- Improve the log viewer for large logs without changing backend log APIs.
- Reduce visual weight of backend-placeholder panels on the execution detail page.
- Fix minor form accessibility issues on touched search fields.

## Non-Goals

- No backend API changes.
- No Argo log pagination, tail, or streaming implementation in this change.
- No asset-node ledger backend work.
- No redesign of the whole pipeline module.

## Scope

Frontend-only changes under workflow execution detail pages and directly related DAG node interactions.
