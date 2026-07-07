# CYB-3108 — Batch list: fix missing 完成时间 + real run duration

## Problem

In the batch task list, most batches show 完成时间 / 耗时 as "—", and even when
present the 耗时 is misleading:

1. **完成时间 not stamped.** `backfill_jobs.finished_at` is only stamped by
   `UpdateJobStatus` (used for synchronous submission failures). The normal async
   finalization writes the terminal status via `UpdateJobProgress`
   (`backfill_repo.go`) / pilot's `UpdateJobPilotPhase`, neither of which stamped
   `finished_at`. So normally-completed/failed batches — including ones created
   today — had `finished_at = NULL` → "—". (Verified via code paths: the two rows
   that *did* have it in the report both fast-failed at submission.)

2. **耗时 semantics.** The list computed 耗时 = `finished_at − created_at`, i.e.
   wall-clock from submit to finish, which includes subtask queue/pending time,
   scheduling lag, and paused time — not real run time.

## What

- **A**: stamp `finished_at` on the async terminal paths (`UpdateJobProgress`,
  `UpdateJobPilotPhase`) using
  `CASE WHEN status IN ('completed','failed') THEN COALESCE(finished_at, NOW()) ELSE finished_at END`,
  matching `UpdateJobStatus`.
- **B**: `FindAllJobs` now LEFT JOINs a per-job subtask aggregate
  (`MIN(started_at)`, `MAX(finished_at)`) exposed on `BackfillJob` as
  `runStartedAt` / `runFinishedAt` (no shared-scan disruption, no new interface
  method → no mock churn).
- **Frontend** (`BatchJobList`): 完成时间 = `finishedAt ?? runFinishedAt` (terminal
  only); 耗时 = `runFinishedAt − runStartedAt` (first subtask start → last finish,
  excludes submit/queue/pause), with a tooltip. Logic extracted to pure helpers in
  `lib/batchJobs.ts` and unit-tested.

## Scope

- Backend (model + `FindAllJobs` query + two stamping SQLs) + Frontend
  (`BatchJobList`, `lib/batchJobs`, `batchJobApi` type) + OpenAPI (`BackfillJob`
  gains `runStartedAt`/`runFinishedAt`).
- No DB migration: run span is derived from subtasks at read time, so existing rows
  render correctly too. `finished_at` stamping is forward-fixing for the stored
  column (sorting/notification/canonical).

## Out of scope

- SDK: the batch-job list is not a public SDK surface (the SDK only filters *runs*
  by `batchJobId`), so no SDK change (contract-sync rows 1/7 covered: OpenAPI + spec).
- Excluding inter-subtask idle gaps from 耗时 (would need interval-union); the
  first-start→last-finish span is the pragmatic real-run metric.
