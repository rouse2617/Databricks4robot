# CYB-4350: batch total-duration column + Feishu notification line

## Context

The batch job list gives progress + time-elapsed but not "how much video was actually processed" — a signal most users care about when planning capacity or reporting. A new column `总时长` = sum of `assets.duration_ms` across every asset touched by the batch's child runs surfaces that directly.

Same signal in the CYB-3071 Feishu completion notification: right now `【批量任务完成】name 状态:… 总数:… 成功:… 失败:… 耗时:…` — adding a `总时长` line lets non-tech recipients see corpus size without opening the app.

## Change

### Backend

`BackfillJob` model gains `TotalDurationMs int64 \`json:"totalDurationMs"\``.

**List path (`GET /api/v1/backfill`)**: existing repo `ListJobs` runs unchanged. After the main list resolves, run a single supplementary aggregate query keyed by the returned batch ids:

```sql
SELECT pr.batch_job_id AS batch_id,
       COALESCE(SUM(a.duration_ms), 0) AS total_ms
FROM pipeline_runs pr
CROSS JOIN LATERAL unnest(pr.asset_ids) AS aid
JOIN assets a ON a.asset_id = aid AND a.is_deleted = FALSE
WHERE pr.batch_job_id = ANY($1)
GROUP BY pr.batch_job_id;
```

Merge into the response items before serialization. Batches not appearing in the result get `total_ms = 0` (no child runs / all children unassigned).

**Single path (`GET /api/v1/backfill/:id`)**: same query with a single-element id array.

**Feishu template (`formatBatchJobNotificationText` at usecase.go:1607)**: append a line `总时长:XhYm` between the counters and the create-time block. Data pulled by an additional call in the completion notification flow (single-batch total-duration query) — one query per completion event is fine.

### Frontend

`BatchJobList.tsx`:
- New Table column `总时长`, value from `job.totalDurationMs`, rendered via a shared `formatDurationMs(ms)` helper.
- Position: right after existing 耗时 column (matches user's screenshot arrow location).
- Sortable via `total_duration_ms desc`; default sort unchanged.

`Frontend/src/api/batchJobApi.ts` — `BatchJob` type gains `totalDurationMs?: number` (optional so old server responses don't break the type).

`Frontend/src/lib/batchJobs.ts` — pull `formatDurationMs` from `pages/presets/durations.tsx` into this shared lib so both the batch list and the durations preset use the same formatter. Presets re-export via `presets/durations`.

## Semantics (documented per user decision)

- **All child assets counted** — success + failure alike (total-corpus scope, not just delivered)
- **No dedup across runs** — if the same asset appears in multiple child runs' asset_ids, it counts once per occurrence (each subtask contributes its own asset time)
- **Soft-deleted assets excluded** — `is_deleted = FALSE` in the join predicate
- **Missing duration** — treated as 0 (COALESCE), same as CYB-4294 `/assets/durations`

## Verification

- Backend: `go test ./...` + `go build`
- Frontend: `tsc / vitest / vite build`
- Dev smoke: batch list column renders sensible values; force a batch to completion to see Feishu notification carry the new line (manual — verify webhook receives it)

## Non-goals

- No denormalized `batch_jobs.total_duration_ms` column — CYB-4350 does on-demand aggregation. If the aggregate query becomes hot, split into a follow-up.
- No per-node breakdown — that lives on the batch detail page's node summary.
- No historical backfill / migration — new field is computed live from the same `assets` table.
