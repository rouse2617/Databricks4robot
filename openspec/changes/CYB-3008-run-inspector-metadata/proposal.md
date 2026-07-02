# Proposal - CYB-3008 Run Inspector Metadata

## Summary

Expose Run Inputs, Run Outputs, and Runtime reference data in Run Inspector.

## Problem

The backend and API expose `/runs/{id}/inputs`, `/runs/{id}/outputs`, and `/runs/{id}/runtime`, but the Run Inspector frontend still loads only events, asset-node snapshots, and cost. Config-as-RunInput and RunOutput projections are not visible to users unless they call the API manually.

## Goals

- Load Run inputs, outputs, and runtime metadata alongside existing Run ledger data.
- Render a compact Run Inspector section for inputs, outputs, and runtime debug reference.
- Keep failures in metadata loading non-fatal to DAG/events/logs.
- Cover the hook and page behavior with focused frontend tests.

## Non-Goals

- No backend API shape change.
- No schema migration.
- No replacement of node-level logs, terminal, Pod, or metrics debug paths.

## Acceptance Criteria

- Run Inspector calls `/runs/{id}/inputs`, `/runs/{id}/outputs`, and `/runs/{id}/runtime` after resolving a Run.
- Inputs show config and asset entries using RunInput fields.
- Outputs show available output projections with reasonable empty state.
- Runtime reference shows workflow name, namespace, UID, target, and runtime status when available.
- Metadata failures show a local warning and do not blank the Run Inspector.
