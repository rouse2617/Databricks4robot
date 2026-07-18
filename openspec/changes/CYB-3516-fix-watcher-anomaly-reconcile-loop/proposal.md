# CYB-3516 — Fix watcher anomaly reconcile self-perpetuating loop

## Problem

Old pipeline runs (created months ago) whose Argo workflows have been TTL-cleaned
keep re-entering the watcher anomaly reconcile set every polling cycle (~3s).
This produces ~80 WARN-level log lines per 40 minutes and pointless Argo gRPC
calls (code 5 NOT_FOUND) that never succeed.

## Root cause

`runObservedRecently()` falls back to `UpdatedAt` when `FinishedAt` is nil.
`PipelineRunRepo.Save()` unconditionally sets `UpdatedAt = now` on every upsert.
Every reconcile pass calls `persistRunObservation` → Save → bumps `UpdatedAt` →
run stays inside the 7-day observation window → loop forever.

## Fix

Remove `UpdatedAt` from the `runObservedRecently` timestamp fallback chain.
The function now uses only lifecycle-significant timestamps:
`FinishedAt → StartedAt → CreatedAt`.

This is the minimal change: one line removed from the ref chain. No new helpers,
no schema changes, no behavioral change for genuinely recent runs.

## Scope

- `backend/internal/usecase/pipeline/usecase.go` — `runObservedRecently()`
- `backend/internal/usecase/pipeline/usecase_test.go` — new table-driven test

No API, schema, SDK, or frontend changes.
