# Proposal - CYB-3006 Run Rerun

## Summary

Add product Run `rerun` semantics distinct from runtime retry and resubmit.

## Problem

Runtime OS distinguishes:

- `retry`: retry failed runtime nodes in the same Run when supported by the runtime.
- `resubmit`: create a new Run as a resubmission of the previous runtime spec.
- `rerun`: create a new Run from the original Run spec as a product-level full rerun.

The API currently exposes `retry` and `resubmit` on `/runs/{id}` but does not expose product `rerun`, leaving the old `/pipeline-runs/{id}/retry` full-rerun semantics as legacy-only.

## Goals

- Add `POST /api/v1/runs/{id}/rerun`.
- Preserve `/api/v1/runs/{id}/retry` as runtime retry.
- Preserve `/api/v1/runs/{id}/resubmit` as resubmit.
- Write RunEvent ledger entries for rerun requested, failed, and created.
- Update OpenAPI, API guide, frontend Run API, SDK manager, and safe smoke checks.

## Non-Goals

- No schema migration or durable run relation table in this slice.
- No removal of legacy `/pipeline-runs/{id}/retry`.

## Acceptance Criteria

- `POST /runs/{id}/rerun` creates a new Run from the source Run spec and returns `201`.
- Failed rerun creation writes a failed RunEvent on the source Run.
- The created Run receives a ledger event with payload `{sourceRunId, relation: "rerun_of"}`.
- Unknown source Run returns the existing not-found semantics.
- SDK and frontend API wrappers include `rerun`.
- Run Inspector exposes product-level rerun separately from runtime retry and resubmit.
