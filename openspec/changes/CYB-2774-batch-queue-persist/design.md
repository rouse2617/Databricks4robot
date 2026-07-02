# Design — CYB-2774

## Architecture Context

- **Constraints**: Go 1.25+, PostgreSQL single source of truth, no new external dependencies
- **Goals**: Batch item processing survives service restart without manual recovery
- **Non-Goals**: Full-featured job queue system; Argo workflow changes; external message broker

## Affected Modules

- `backend/internal/repository/backfill_repository.go` — new `ClaimNextItems`, `ResetStaleItems`
- `backend/internal/postgres/backfill_repo.go` — SQL with `SKIP LOCKED`, lease reset
- `backend/internal/usecase/backfill/usecase.go` — `runItems` polling mode, stale reaper
- `backend/internal/usecase/pipeline/batch.go` — `processBatchJob` polling mode
- `backend/migrations/` — add `attempts` column

## Architecture Decisions

### Decision 1: DB-pulled work queue instead of in-memory channel
- **Approach**: Workers poll `backfill_items` with `SELECT ... FOR UPDATE SKIP LOCKED` instead of reading from a `chan`
- **Alternative**: Keep `chan` but persist pending items (hybrid — still loses in-flight items)
- **Rationale**: SKIP LOCKED is the standard Postgres pattern for safe concurrent work claiming, used by Radish, pqueue, go-jobqueue, and other production queue libraries. Zero new dependencies.
- **Trade-off**: Adds polling latency (~500ms-1s) vs zero-latency channel; negligible for batch throughput (5 workers × 2s/item)
- **Risk**: Low — pattern is well-established; rollback is reverting to channel-based processing

### Decision 2: Lease timeout at item level
- **Approach**: When a worker claims an item, set `started_at = NOW()`. A stale reaper reclaims items where `status = 'running' AND started_at < NOW() - lease_timeout`. Lease timeout matches the per-item deploy timeout (60s).
- **Alternative**: No lease, rely on restart-time scan only
- **Rationale**: Handles both crash-during-submit and silent-goroutine-death scenarios
- **Rollback**: Keep the lease_timeout config, set very high to effectively disable

### Decision 3: Stale reaper as background goroutine in Usecase
- **Approach**: A long-lived goroutine in `Usecase` struct polls `backfill_items` for stale running items every lease_timeout interval
- **Alternative**: Cron-based or outbox-based reaper
- **Rationale**: Simplest, co-located with the processing logic, no infra dependency
- **Risk**: Must stop gracefully on shutdown to avoid double-processing race

## Data Flow

```
Service Start
  │
  ├─► BackfillUsecase.Init()
  │     ├─► start stale reaper goroutine
  │     └─► scan backfill_jobs WHERE status='running'
  │           └─► for each: start runItems() polling workers
  │
  Worker Loop (per batch job, N goroutines)
  │
  ├─► SELECT id FROM backfill_items
  │     WHERE job_id = $1 AND status = 'pending'
  │     ORDER BY created_at LIMIT 1
  │     FOR UPDATE SKIP LOCKED
  │
  ├─► UPDATE SET status='running', started_at=NOW()
  │
  ├─► execute deploy (CreateRunByTemplateID)
  │     ├─► success → UPDATE status='completed'
  │     └─► failure → UPDATE status='failed', error_message=...
  │
  └─► loop (poll after short delay)
  │
  Stale Reaper (every lease_timeout)
  │
  ├─► SELECT bi.* FROM backfill_items bi
  │     JOIN backfill_jobs bj ON bj.id = bi.job_id
  │     WHERE bi.status = 'running'
  │       AND bi.started_at < NOW() - lease_timeout
  │       AND bj.status = 'running'
  │
  └─► UPDATE SET status='pending', started_at=NULL, attempts=attempts+1
```

## Data Model Changes

- **Table**: `backfill_items`
- **Change**: Add `attempts INT NOT NULL DEFAULT 0`
- **Migration**: `backend/migrations/061_backfill_items_attempts.sql`

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 重复提交 (duplicate run) | 相同 asset 产生多个 Argo workflow | `UpsertBatchSubtaskRun` 已去重；reaper 回收时检查 pipeline_run_id |
| Reaper 和 worker 竞争 | 刚 claimed 的 item 被 reaper 抢走 | started_at 更新在 worker 取到后立即设置；lease_timeout 远大于正常耗时 |
| 轮询压力 (polling) | 频繁查 DB | 5 个 worker + 1 个 reaper，查询带索引；可调 polling 间隔 |
