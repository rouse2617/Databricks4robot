# Proposal — CYB-1534 Run Target Model

## Problem
Pipeline execution is still backed by the compatibility `pipeline_deployments` row. `ExecutionTarget` is synthesized from process config, `pipeline_runs` and `pipeline_run_nodes` do not exist, and node-level state is inferred from Argo on demand.

This blocks durable asset x node tracking, multi-cluster / multi-namespace governance, stable run history after Argo TTL cleanup, and run-level cost/resource accounting.

## Goals
- Persist execution targets as first-class runtime destinations.
- Persist pipeline runs independently from legacy deployment records.
- Persist pipeline run nodes with Argo node / pod identity and resource/log metadata pointers.
- Keep existing `/deploy`, `/deploy/template`, and `/deployments` endpoints compatible while introducing first-class `/pipeline-runs` APIs.
- Seed or synthesize a default target from existing Argo environment config so dev deployments keep working during migration.

## Non-Goals
- Full frontend redesign for the new run model.
- Persistent log chunk storage; that is handled by `CYB-1535-bounded-logs-resources`.
- Strict algo-run asset validation; that is handled by `CYB-1536-asset-validation`.
- Storing raw Argo tokens in the database or API responses.

## Acceptance Criteria
- A migration creates `execution_targets`, `pipeline_runs`, and `pipeline_run_nodes`.
- `GET /api/v1/execution-targets` becomes repository-backed and still returns the existing default target shape.
- `POST /api/v1/pipeline-runs` creates a durable run from either a template ID or inline pipeline definition.
- `GET /api/v1/pipeline-runs` and `GET /api/v1/pipeline-runs/{id}` return run data and node data without depending solely on legacy deployment rows.
- Existing deployment endpoints keep their current response shape by projecting from the new model or by maintaining compatibility writes.
- Active runs refresh status from Argo with bounded polling and mark runs `Expired` when the Workflow CR has been TTL-cleaned.
- OpenAPI, api-guide, SDK, and smoke coverage are updated in the same PR.
