# Proposal - CYB-3009 Run Relation Event Projection

## Summary

Project non-batch Run relations from the RunEvent ledger into `/runs/{id}/children`.

## Problem

`/runs/{id}/children` currently derives only `batch_child` relations from batch job metadata. Product operations that create a new Run from an existing Run, such as resubmit, rerun, and legacy full retry, write events but are not visible as Run Tree children of the source Run.

## Goals

- Write source Run events with `childRunId` and relation payload when a new Run is created by resubmit, rerun, or legacy retry.
- Project `retry_of`, `resubmit_of`, and `rerun_of` relations from source Run events.
- Combine event-derived relations with existing batch child relations.
- Keep this additive and schema-free.

## Non-Goals

- No durable `run_relations` table in this slice.
- No removal of existing batch relation derivation.
- No change to runtime-level retry semantics on `/runs/{id}/retry`.

## Acceptance Criteria

- Source Run children include rerun-created child Runs with relation `rerun_of`.
- Source Run children include resubmit-created child Runs with relation `resubmit_of`.
- Legacy `/pipeline-runs/{id}/retry` full-run retry can be projected as `retry_of`.
- Batch child projection continues to work.
