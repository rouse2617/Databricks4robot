# Proposal — CYB-1538 Pipeline Run UX Output

## Problem
The saved pipeline run action still calls the compatibility deploy endpoint and omits `asset_ids` when no assets are selected. With first-class `pipeline_runs`, that can persist a null asset list and violate the database contract.

Components also get a default output port from the UI, but Argo treats declared output parameters as required files. A simple component that prints success can still fail when it does not write `/tmp/outputs/output`.

## Goals
- Route saved-pipeline run actions through the first-class pipeline-run API.
- Send an explicit empty `asset_ids` array for no-asset runs.
- Persist no-asset pipeline runs without storing null `asset_ids`.
- Avoid requiring `/tmp/outputs/<name>` for unconsumed default output ports.
- Keep consumed output ports and explicit output-file components compatible with Argo.
- Reduce browser autofill/append behavior in pipeline/component form fields.

## Non-Goals
- Redesigning the component output schema.
- Removing compatibility deploy endpoints.
- Implementing asset selection or execution target redesign.

## Acceptance Criteria
- Running a saved pipeline with no selected assets creates a run request with `asset_ids: []`.
- `pipeline_runs.asset_ids` is saved as an empty Postgres text array, not null.
- A single-node echo component with an unused default output port does not fail because `/tmp/outputs/output` is absent.
- A component that writes `/tmp/outputs/output`, or whose output is consumed by another node, still declares the Argo output parameter.
