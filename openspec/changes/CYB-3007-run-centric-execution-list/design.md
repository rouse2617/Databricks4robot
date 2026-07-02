# Design - CYB-3007 Run-Centric Execution List

## Approach

Keep the current `WorkflowExecutionList` component file for compatibility, but change its list data path:

- Use `listRuns({view: "summary", excludeBatch: true, ...})` as the only product list fetch.
- Project `PipelineRun` rows into the existing table record shape.
- Keep runtime workflow APIs available only through detail/debug flows.
- Keep legacy operation fallback for rows without a Run id, but normal product rows should have `runId`.

## Filtering

The frontend continues applying some filters locally against the current Run page:

- Status is passed to `/runs` and also locally checked.
- Name checks Run id, pipeline name, and runtime workflow debug name.
- Date checks Run `createdAt` and `finishedAt`.
- Label checks labels derived from Run ledger metadata currently exposed to the page, such as `asset_id`.

Server-side name/date/label filtering is not added in this slice. That can be added later as a backend Run list contract extension.

## Sorting

The execution table uses Ant Design local column sorters for duration, estimated total cost, created time, and finished time. Sorting is intentionally scoped to the currently loaded page because `/api/v1/runs` does not yet expose sort query parameters. Adding server-side sort can be handled later as an API contract extension.

## List Cost And Template Search

`/api/v1/runs?view=summary` returns lightweight Run rows, so the list must not trigger per-row runtime refreshes or full node hydration. Summary rows should use already persisted node cost snapshots to populate `totalEstimatedCost` when available. Missing snapshots remain `null` and the UI keeps the existing placeholder.

Run summary search adds a `q` query parameter that matches Run ID, pipeline name, workflow name, and linked pipeline template name. The response also includes `templateName` when the Run has a linked template, so the frontend can show which template/version produced a row.

## Compatibility

`workflowName` remains present in the projected row because:

- Existing rows and routes still display runtime debug names.
- Detail pages use it to fetch workflow debug, logs, Pod diagnostics, terminal, and metrics.

It is no longer a product-list source of truth.

## Routing

`/runs` is the stable product route for execution records. The legacy
`/pipeline?tab=executions` and `/pipeline/executions/:workflowName` routes remain
compatibility paths, but product navigation should prefer:

- `/runs` for the execution list.
- `/runs?executionView=batch` for the batch execution list.
- `/runs/:runId` for Run detail when a Run id is available.
