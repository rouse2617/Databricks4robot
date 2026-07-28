# Batch total-duration spec — CYB-4350

## Response contract change

`BackfillJob` gains a single new field, always present:

```json
"totalDurationMs": 12345678
```

- Type: int64 (no decimals; ms precision)
- Semantics: **sum of `assets.duration_ms`** across the union (with multiplicity) of every `asset_id` in every `pipeline_runs.asset_ids` array where `batch_job_id == this batch`, filtered by `assets.is_deleted = FALSE`
- Absent-run case (batch with no child runs yet): `totalDurationMs = 0`
- Missing-duration case (asset row exists but `duration_ms` is NULL): treated as 0 via `COALESCE`
- Soft-deleted-asset case: excluded (join predicate on `is_deleted = FALSE`)

## Aggregate SQL

```sql
SELECT pr.batch_job_id AS batch_id,
       COALESCE(SUM(a.duration_ms), 0) AS total_ms
FROM pipeline_runs pr
CROSS JOIN LATERAL unnest(pr.asset_ids) AS aid
JOIN assets a ON a.asset_id = aid AND a.is_deleted = FALSE
WHERE pr.batch_job_id = ANY($1)
GROUP BY pr.batch_job_id;
```

Repo method `TotalDurationByBatchIDs(ctx, batchIDs []string) (map[string]int64, error)`.

Batches present in `batchIDs` but absent from the query result map get an implicit `0` at the caller. Empty `batchIDs` short-circuits to an empty map without an SQL round trip.

## Feishu notification template

`formatBatchJobNotificationText` gains a parameter `totalDurationMs int64`. Text output when non-zero:

```
【批量任务完成】<name>
状态:<status>
总数:N  成功:M  失败:K
总时长:<formatted>
[创建人:...]
[耗时:...]
[链接:...]
```

When `totalDurationMs == 0`: omit the `总时长` line entirely (don't show `总时长:—` — cleaner).

Formatting:
- ≥1h → `${h}h ${m}m` (drop the seconds)
- 1min–1h → `${m}m ${s}s`
- <1min → `${s}s`
- 0 → omit

## Frontend column

Column key `totalDurationMs`, header `总时长`, position immediately after existing `耗时` column.

- Cell value: `formatDurationMs(job.totalDurationMs ?? 0)` — `—` when 0
- Sortable: yes, by `totalDurationMs` (numeric)
- Width: `100px` (match sibling numeric columns like 耗时)

## Shared formatter

`Frontend/src/lib/batchJobs.ts` exports `formatDurationMs(ms: number): string`. Signature and behavior identical to today's implementation in `pages/presets/durations.tsx`. That file re-exports it for backward compatibility so preset call sites don't move.

## Non-changes

- List response envelope stays `{ items: [...] }` (same as today)
- No pagination changes
- No new endpoint — piggybacks on `/backfill` and `/backfill/:id`
- No websocket / SSE / update push — total duration is a snapshot at fetch time; user re-fetches to see updates
