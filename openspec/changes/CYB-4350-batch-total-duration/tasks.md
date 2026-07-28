# CYB-4350 tasks

## Backend

- [ ] `models/backfill.go` — add `TotalDurationMs int64 \`json:"totalDurationMs"\`` to `BackfillJob`
- [ ] `postgres/backfill_repo.go` — new method `TotalDurationByBatchIDs(ctx, batchIDs []string) (map[string]int64, error)` that runs the aggregate SQL from spec and returns `batch_id → total_ms`. Missing batches implicit-zero at the caller.
- [ ] `usecase/backfill/usecase.go`:
  - `ListJobs` — after the existing list returns, call `TotalDurationByBatchIDs` with the resolved ids; write the returned map into each `BackfillJob.TotalDurationMs`
  - `GetJob` (single) — call the same repo method with a `[]string{id}`, set `TotalDurationMs`
  - `formatBatchJobNotificationText` — accept a new `totalDurationMs int64` parameter; append line `总时长:<formatDuration>` before the `耗时` line (or wherever fits the reading order). Callers pass the fetched value.
  - Completion notification flow — before calling `formatBatchJobNotificationText`, look up the total duration for the batch. One extra query per completion event is fine.
- [ ] `notify/feishu/` — no changes; text template lives in the usecase package.
- [ ] Duration formatter — pick a spot for a shared helper (`internal/durations` or similar; grep first to see if one already exists). Format `Xh Ym`, `Ym Ns`, `Ns`, `—` for 0.

## Frontend

- [ ] `Frontend/src/api/batchJobApi.ts` — extend `BatchJob` type with `totalDurationMs?: number`
- [ ] `Frontend/src/lib/batchJobs.ts` — move `formatDurationMs(ms: number): string` here from `pages/presets/durations.tsx`. Re-export from durations preset for backward compatibility.
- [ ] `Frontend/src/pages/BatchJobList.tsx` — new column `总时长`, sortable, positioned right after 耗时; render via `formatDurationMs`
- [ ] Optional: keep the column visible when `totalDurationMs` is undefined by showing `—` (older-server compat)

## Tests

- [ ] Backend repo test: verifies SQL params + result mapping (fakeDB pattern) — empty batch list returns empty map; multiple batches return each with their sum
- [ ] Backend usecase test: `ListJobs` populates `TotalDurationMs`; `GetJob` populates it; `formatBatchJobNotificationText` renders the new line correctly with and without duration
- [ ] Frontend test: `BatchJobList.test.tsx` asserts the new column renders and sorts

## Verification

- [ ] `go test ./...`
- [ ] `go build -ldflags "-w -s" ./...`
- [ ] `npx tsc --noEmit`
- [ ] `npx vitest run <touched>`
- [ ] `npx vite build`

## Ship

- [ ] Commit + push + PR base=dev
- [ ] Merge → deploy → dev smoke: batch list shows new column with sensible numbers; force a batch to completion to see Feishu notification carry the 总时长 line
