# Proposal - CYB-3005 Config Run Inputs

## Summary

Materialize deploy-level and node-level pipeline configs into the product Run input projection returned by `/api/v1/runs/{id}/inputs`.

## Problem

Runtime OS defines Config as a Run Input, but the current implementation only derives node-level `runtimeConfig` fields from `PipelineJSON`. Deploy-level `configSelection` is resolved during submission but is not preserved as a product input, and config inputs do not expose a content hash for immutable audit.

## Goals

- Persist config input metadata in the existing Run `PipelineJSON` compatibility payload at creation time.
- Include deploy-level config selections and node-level config bindings in `/runs/{id}/inputs`.
- Expose `contentHash` for config inputs without exposing raw config content.
- Preserve existing runtime ConfigMap projection and validation behavior.
- Avoid database migrations in this slice.

## Non-Goals

- No dedicated `run_inputs` table yet.
- No change to runtime config materialization behavior.
- No change to config authoring APIs.
- No frontend UI redesign in this slice.

## Acceptance Criteria

- A Run created with deploy-level `configSelection` returns a `config` RunInput with source, file name, mount path, target filename, ref/version where applicable, and content hash.
- A Run created with node-level `runtimeConfig` returns one `config` RunInput per bound node with the same metadata.
- Raw config content is not returned in RunInput snapshots.
- Existing node-level fallback still works for historical Runs that lack materialized config input metadata.
- `go test ./internal/usecase/pipeline ./internal/handlers/pipeline` and `go test ./...` pass.
