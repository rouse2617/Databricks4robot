# CYB-1682 — Show component names on workflow DAG nodes

## Motivation
Workflow detail DAG nodes currently show Argo task/template identifiers such as `step-step-1` as the primary node title. Those identifiers are useful for backend routing but not meaningful to users reading the pipeline graph. Users expect the node title to describe the actual component or pipeline step they authored.

## Scope
- Resolve a user-facing display name for workflow nodes from the DataBrew pipeline template/run metadata when available.
- Render that display name as the primary DAG node title and node-detail heading.
- Keep Argo node/task/template identifiers available as secondary metadata and continue using them for logs, Pod diagnostics, resources, and debug APIs.
- Add focused frontend tests for the mapping so future changes do not regress to `step-step-*` titles.

## Out of Scope
- Renaming Argo workflow tasks/templates in submitted manifests.
- Changing log, Pod, resource, or terminal API path identifiers.
- Adding a new backend endpoint.
