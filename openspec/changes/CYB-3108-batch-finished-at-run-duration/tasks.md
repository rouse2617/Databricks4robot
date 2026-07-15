# Tasks — CYB-3108

## Backend
- [x] `UpdateJobProgress` + `UpdateJobPilotPhase`: stamp `finished_at` on terminal (COALESCE).
- [x] `FindAllJobs`: LEFT JOIN subtask aggregate → `run_started_at` / `run_finished_at`; dedicated `scanBackfillJobWithRunSpan`.
- [x] `models.BackfillJob`: add `RunStartedAt` / `RunFinishedAt`.
- [x] `go build` / `go vet` / `gofmt` clean; backfill usecase + handler tests pass.

## Frontend
- [x] `batchJobApi.ts`: `runStartedAt?` / `runFinishedAt?` on `BatchJob`.
- [x] `lib/batchJobs.ts`: pure helpers `batchJobCompletionAt`, `batchJobRunDurationSeconds`, `formatDurationSeconds`.
- [x] `BatchJobList.tsx`: 完成时间 fallback + real 耗时 + tooltip.
- [x] `BatchJobList.test.tsx`: 6/6 (completion fallback, running→undefined, real span excl. queue).
- [x] biome clean, `npm run build` clean.

## Contract
- [x] OpenAPI `BackfillJob` += `runStartedAt` / `runFinishedAt`.
- [x] SDK: out of scope (batch-job list not in SDK) — documented in proposal.

## Verify
- [ ] Deploy backend (via CI) + frontend; on dev batch list confirm normally-completed batches show 完成时间 and a real 耗时 (< wall-clock).
- [ ] PR to dev.
