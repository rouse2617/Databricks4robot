# Design - CYB-3009 Run Relation Event Projection

## Approach

Use existing `pipeline_run_events` as an interim relation ledger:

- When an operation creates a child Run, append an event to the source Run with payload:
  - `childRunId`
  - `sourceRunId`
  - `relation`
- Continue appending the existing event on the child Run for local audit.
- In `ListRunChildren`, read source Run events and build `RunRelation` rows for supported relation payloads.

## Supported Relations

- `retry_of` for legacy full-run retry.
- `resubmit_of` for product resubmit.
- `rerun_of` for product rerun.

## Direction

`RunRelation.ParentRunID` is the source Run id. `RunRelation.ChildRunID` is the newly created Run id. The `RelationType` describes how the child relates to the parent.

## Compatibility

Batch child relations continue to be derived from `pipeline_runs.batch_job_id`. Event-derived relations are appended and deduplicated by child id + relation type.
