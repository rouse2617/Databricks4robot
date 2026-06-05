# Design — CYB-1684

## Current State

`pipeline_runs` persists:

- `template_id`
- `pipeline_name`
- `template_version`
- `workflow_name`
- `batch_run_id`

But the UI still does not present execution origin as a first-class concept.

Current gaps:

1. The execution list uses workflow-centric rows and only surfaces `模板 vX`.
2. The detail page does not treat pipeline/version/snapshot/trigger source as a stable header-level summary.
3. Run history can drift semantically because some context is inferred from current template state rather than clearly preserved in the run snapshot.
4. Batch/manual/API-trigger distinctions are not expressed in the run model or UI.

## Proposed Model

Execution traceability should be treated as part of the durable run ledger.

### Run Snapshot Fields

The run ledger should retain enough fields to explain historical origin without depending on current template lookup:

- `pipeline_template_id`
- `pipeline_template_name`
- `pipeline_version_number`
- `pipeline_version_id` or `snapshot_id`
- `trigger_source`

Where possible, existing fields should be reused or renamed logically in API projection instead of duplicating storage without purpose.

### Trigger Source

Define a small stable enum for UI and API:

- `manual`
- `asset_run`
- `batch`
- `api`

The backend determines this at run creation time and persists it with the run.

## API Shape

Existing run list/detail endpoints should return traceability explicitly, not force the frontend to infer it from workflow name plus deployment lookups.

List/detail rows should expose a compact structure equivalent to:

- pipeline name
- version number
- template id
- snapshot/version id
- trigger source
- optional failure summary

This keeps list/detail aligned and reduces frontend ad hoc joining logic.

## Frontend Information Hierarchy

### Execution List

Each row should read like:

- primary: run name
- secondary: `流水线 <pipeline name> · v<version>`
- tertiary: short run id / snapshot id / trigger source as density allows

The list should also improve adjacent usability:

- hide the empty label filter row when there are no label options
- add top-level execution summary counts
- failed rows should expose failure context faster than the current "click through to detail" flow
- bulk delete disabled state should explain why it is disabled

### Execution Detail

Header summary should display:

- pipeline name
- version
- snapshot/template identity
- trigger source
- asset mode when relevant (`no-asset`, single asset, batch)

This information belongs beside status and duration, not deep in the page.

## Risks

- Run model changes can require DB migration if current persisted fields are insufficient.
- API additions must stay backward-compatible for existing frontend paths.
- Trigger source can become ambiguous if creation flows do not define it consistently.
- Execution list density can regress if too much metadata is surfaced without a compact hierarchy.
