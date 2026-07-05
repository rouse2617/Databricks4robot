# Phase 4 — Persistent Dispatcher (Outbox Engine) — Design

> **Status:** design-only. No code yet. Awaiting user sign-off before implementation.
> **Plan reference:** `~/.claude/plans/sprightly-meandering-pascal.md`
> **Depends on:** Phase 0 (tx client), Phase 1 (idempotency hinge) — both committed in `d8fbf090`.

---

## 0. Goal (recap)

Replace the in-memory goroutine dispatch (`runItems` → `executeItem`) with a **persistent outbox-style dispatcher** that survives Cloud Run instance recycling, multi-instance concurrency, and process crashes without losing or duplicating sub-task submissions.

The end state (after Phase 4 + the patch-cleanup that comes with it):
- Four batch entry points (`CreateBackfill`, `ResumeJob`, `Rerun`, `ContinueFull`) only enqueue — they write `dispatch_state='pending'` (or bumped `dispatch_generation`) inside the same transaction as the business write. **Never spawn a worker goroutine anymore.**
- One long-running `Dispatcher.Run(ctx)` ticker (2s tick) repeatedly claims ready items via `FOR UPDATE SKIP LOCKED` + lease, and submits via the existing `pipelineUC.DeployByTemplateID` (which already has Phase 1's deterministic workflow-name + AlreadyExists adoption + Terminating rejection wired in).
- Multi-instance safe: every replica runs the same dispatcher loop; row-level lease ensures no double submit, no lost lease.
- The whole fragile machinery — `runItems`, `executeItem`, `runAlreadySubmitted`, the soft dedup guard, and the in-memory "claim → 60s timeout → 3 attempts → failed" dance — **is deleted**, not patched. The DB state machine is the single source of truth.

---

## 1. Data model — extend `backfill_items`

`backfill_items` already behaves 80% like a queue (SKIP-LOCKED claim, `attempts`, reaper). We push it the last 20%: add dispatch columns instead of carving out a parallel `dispatch_outbox` table. Reasons:
- One queue, one source of truth, one claim query
- No cross-table consistency problem to debug
- `FOR UPDATE SKIP LOCKED` already battle-tested on `backfill_items.status` in this codebase

### 1.1 Migration `064_backfill_dispatch_outbox.sql`

```sql
-- Add dispatch state machine columns.
ALTER TABLE backfill_items
  ADD COLUMN dispatch_state           VARCHAR(16)  NOT NULL DEFAULT 'pending',
  -- States: pending | claimed | submitting | submitted | failed | dead
  -- 'pending' / 'failed' / 'dead' live across restart.
  -- 'claimed' / 'submitting' are lease-bounded.
  ADD COLUMN dispatch_generation      BIGINT       NOT NULL DEFAULT 1,
  -- Bumped by PrepareItemsForRerun. Part of the deterministic identity
  -- in workflow_name_planned — a rerun must legitimately produce a NEW
  -- workflow, never silently re-adopt an old one via 409 + AlreadyExists.
  ADD COLUMN workflow_name_planned    VARCHAR(253),
  -- The deterministic name (Phase 1) computed at claim time and used
  -- verbatim in DeployOptions.PreallocatedWorkflowName. NULL when not
  -- yet claimed (item is in 'pending' waiting for dispatcher).
  ADD COLUMN dispatch_lease_expires_at TIMESTAMPTZ,
  -- Active lease wall-clock. NULL when not held. Stale entries (the
  -- owning instance died) are re-claimable after NOW() > this.
  ADD COLUMN dispatch_last_error      TEXT,
  -- Failure detail for observability + DLQ reasoning.
  ADD COLUMN dispatch_seq             BIGSERIAL    NOT NULL,
  -- STRICTLY for cold-archive water-mark (see 1.3 below). MUST NOT be
  -- used as the cursor for active claiming — see "游标铁律" in the
  -- plan. Why bigserial is safe here: any "Transaction A got seq=10,
  -- B got seq=11, B committed first" race would break a high-water-
  -- mark cursor forever. We don't store a cursor that consumes seq.

-- Partial index optimised for the hot dispatcher claim path.
-- Includes claimed-expired rows (lease-bounded) so the same index
-- serves both first-claim and re-claim queries.
CREATE INDEX idx_backfill_dispatch_hot
  ON backfill_items (job_id, dispatch_state, dispatch_lease_expires_at)
  WHERE dispatch_state IN ('pending', 'failed', 'claimed', 'submitting');

-- REPLICA IDENTITY DEFAULT: serialise only the PK column in CDC WAL,
-- not the full row. We never use FULL because the row carries log
-- blob content that we don't want broadcast over PG logical
-- replication streams.
-- (Set per-table; this is a table-level option, not enforced via
--  sql DDL — owner of backfill_items needs to apply it from psql:
--    ALTER TABLE backfill_items SET (autovacuum_vacuum_scale_factor = 0.02);
--  But we COULD include this in the migration to keep rollout atomic.)
ALTER TABLE backfill_items SET (
  fillfactor                         = 80,
  autovacuum_vacuum_scale_factor     = 0.02,
  autovacuum_vacuum_threshold        = 200,
  autovacuum_vacuum_cost_limit       = 3000,
  autovacuum_vacuum_cost_delay       = 1
);

-- backfill_items.id is TEXT (uuid string). No index change needed
-- for the ordering pass — we ORDER BY created_at (or dispatch_seq for
-- archive only), never by id.
```

### 1.2 Why extend instead of new table — explicit

- A parallel `dispatch_outbox` table would need its own SKIP-LOCKED-on-`status` index, its own claim → business-row lookup join (item ↔ outbox row ↔ item), and a transactional guarantee that "outbox row written" ≡ "business row created." That consistency contract **cannot be inside a WithTx wrapper unless we duplicate the row** — defeats the point.
- `backfill_items` already has the lifecycle we want: pipeline_run_id (claim), attempts (DLQ counter), workflow_name (legacy placeholder write), `FindIncompleteJobs` (recovery).
- Adding 6 columns is a *forward-compatible* migration: existing legacy reads (claimed-by-`status` paths in `runItems`-era code that we'll delete anyway) keep working because the defaults match legacy semantics.

### 1.3 The dispatcher state machine

```
                          ┌─────────────────┐
                          │     pending      │  ← initial, also after PrepareItemsForRerun
                          └────────┬────────┘
                                   │ ClaimNextDispatch (lease 30s, SKIP LOCKED)
                                   │ atomic: state=claimed, lease_expires_at=now()+30s,
                                   │ workflow_name_planned = deterministic_name(...)
                                   ▼
                          ┌─────────────────┐
                          │     claimed      │
                          └────────┬────────┘
                                   │ dispatcher calls Submit one
                                   ▼
                          ┌─────────────────┐
                          │   submitting     │  (lease still held; refreshed if needed)
                          └────┬───────┬────┘
                success      │      │ other error / timeout / crash
            (Argo OK or      │      │
             AlreadyExists) │      │ reaper expires lease → back to 'pending' (if attempts < N)
                              │      │
                              ▼      ▼
                   ┌──────────────┐   ┌────────────┐
                   │  submitted   │   │   failed    │   (failed = retryable; back to pending)
                   │ (absorbing)  │   │ (retryable) │
                   └──────────────┘   └──────┬─────┘
                                                │ attempts >= maxItemAttempts (3)
                                                ▼
                                          ┌────────────┐
                                          │    dead     │   (absorbing)
                                          │ (absorbing) │
                                          └────────────┘
```

**Absorbing states: `submitted`, `dead`.** These two are *terminal* — no path leads back to `pending`. They are the DB-side guarantee against TTL ghost re-creation (Plan §"核心洞察·铁律"). `ClaimNextDispatch`'s WHERE clause must not include them under any circumstance.

**Reaper**: A simple `ResetStaleDispatchedItems` SQL applied every 60s by the dispatcher loop itself (or as a separate skill — but inlining it into the dispatcher keeps it single-component):
```sql
UPDATE backfill_items
SET dispatch_state='pending', dispatch_lease_expires_at=NULL,
    dispatch_last_error='lease expired'
WHERE dispatch_state IN ('claimed','submitting')
  AND dispatch_lease_expires_at < NOW()
  AND attempts < $maxItemAttempts;
```
**Critical:** never move `'submitted'` or `'dead'` back. The WHERE clause is the safety.

### 1.4 Indices — what goes where

| Index | Columns | Purpose |
| --- | --- | --- |
| (existing `backfill_items.id` PK) | id | unchanged |
| (existing `id_job_id_idx`) | job_id | unchanged |
| **`idx_backfill_dispatch_hot` (NEW)** | (job_id, dispatch_state, dispatch_lease_expires_at) partial WHERE `dispatch_state IN ('pending','failed','claimed','submitting')` | Hot dispatcher claim path |
| `idx_backfill_dispatch_archive` (NEW, deferred) | (dispatch_seq) partial WHERE `dispatch_state IN ('submitted','dead')` | Cold archive water-mark only — created in Phase 4 only if cold-archival kicks off |

---

## 2. Repository surface

### 2.1 New repo methods (batch)

```go
type BackfillRepository interface {
    // Existing
    ClaimNextItem(ctx, jobID string) (*BackfillItem, error)              // legacy; delete Phase 4+
    ResetStaleItems(...) (...)                                       // legacy; delete Phase 4+
    PrepareItemsForRerun(... []string) error                          // rewrite to bump generation + reset dispatch_state

    // New (added in Phase 4 migration 064)
    ClaimNextDispatch(ctx, instanceID string, leaseSec int) (*BackfillItem, error)
    //  ← SKIP-LOCKED, returns one row, atomic state=claimed + lease set + workflow_name_planned computed

    MarkDispatchSubmitting(ctx, itemID string) error
    //  ← refresh lease; called by dispatcher right before Submit to extend heartbeat

    MarkDispatched(ctx, itemID string, wfName string, argoUID string) error
    //  ← state=submitted, workflow_name_planned=wfName, dispatch_lease_expires_at=NULL
    //  ← also write back to backfill_items.pipeline_run_id / workflow_name

    MarkDispatchFailedRetryable(ctx, itemID string, attempts int, err string, retryIn time.Duration) error
    //  ← state=failed, attempts++, dispatch_last_error=err,
    //  ← lease extended by retryIn (exponential backoff)
    //  ← if attempts >= maxItemAttempts -> state=dead instead

    ResetStaleDispatchedItems(ctx, leaseSec int, maxAttempts int) (int, error)
    //  ← the new reaper; replaced existing ResetStaleItems
}
```

The legacy `ClaimNextItem` and `ResetStaleItems` keep existing signatures so the dispatcher + backfill package both compile during the migration. Both get deleted in Phase 4 step 5 after the entry points are switched.

### 2.2 `ClaimNextDispatch` SQL — full text

```sql
WITH picked AS (
    SELECT id
    FROM backfill_items
    WHERE job_id IS NOT NULL
      AND dispatch_state IN ('pending', 'failed')
      AND (dispatch_lease_expires_at IS NULL
           OR dispatch_lease_expires_at < NOW())
    ORDER BY created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE backfill_items bi
SET dispatch_state           = 'claimed',
    dispatch_lease_expires_at = NOW() + ($2 || ' seconds')::interval,
    workflow_name_planned    = COALESCE(
        NULLIF(bi.workflow_name_planned, ''),
        compute_deterministic_wf_name(bi.id, bi.workflow_name, bi.dispatch_generation)
    ),
    attempts                  = attempts + 1
FROM picked
WHERE bi.id = picked.id
RETURNING bi.id, bi.job_id, bi.asset_id, bi.pipeline_run_id,
          bi.workflow_name, bi.workflow_name_planned, bi.dispatch_state,
          bi.dispatch_generation, bi.dispatch_lease_expires_at,
          bi.attempts, bi.error_message;
```

Two non-obvious design decisions:

1. **`ORDER BY created_at ASC` (NOT by id)**: `id` is uuid-string (TEXT column); ordering by it is RANDOM, not FIFO, would break `LIFO/FIFO` fairness across backfills. This is the exact "游标坑" — see design doc §"游标铁律".
2. **`compute_deterministic_wf_name` UDF call**: I'm using UDF here for clarity + to share the hashing with batch path. Alternative: do this in Go. Decision: **do this in Go** (`pipelineUSecase.computeWfNameForItem(item)`). Keeps the migration pure SQL with no surprise behaviour, lets Phase 1 logic stay one place. Remove the `compute_deterministic_wf_name(...)` call from SQL above; instead Go computes it AFTER the UPDATE returns, then it goes into the `workflow_name_planned` set on a small follow-up UPDATE. Two statements is fine — SKIP-LOCKED atomicity still holds on the row-level claim.

### 2.3 `ResetStaleDispatchedItems` SQL

```sql
WITH reclaimed AS (
    UPDATE backfill_items
    SET dispatch_state         = 'pending',
        dispatch_lease_expires_at = NULL,
        dispatch_last_error    = 'lease expired'
    WHERE dispatch_state IN ('claimed', 'submitting')
      AND dispatch_lease_expires_at < NOW() - ($1 || ' seconds')::interval
      AND attempts < $2
    RETURNING id
)
SELECT COUNT(*) FROM reclaimed;
```

Where $1 = leaseSec (e.g. 30) and $2 = maxAttempts (3). The clause "attempts < $2" is what steers the item toward `dead` instead of endless `pending`. Note we still need a separate `MarkDispatchFailedRetryable` path that handles the case where lease DIDN'T expire (e.g. argo returned a **non-network** error after `MarkDispatchSubmitting`).

---

## 3. Dispatcher — `backend/internal/usecase/backfill/dispatcher.go`

### 3.1 Architecture, end to end

```
core.go setupCore() ─── creates Dispatcher{repo, uc, cfg} ───┐
                                                            │
                              ┌──── wires goroutine ────────┴─────────────┐
                              │                                                   │
   cron-like ticker (2s) ─►   dispatchOneCycle()                                │
                              │  ┌───────────────────────────────────────┐      │
                              │  │ ClaimNextDispatch (one row, lease)   │      │
                              │  │ ...or returns nil → sleep tick       │      │
                              │  └───────────────────────────────────────┘      │
                              │  ┌───────────────────────────────────────┐      │
                              │  │ submitOne(item):                     │      │
                              │  │  1. compute wfName_plan in Go         │      │
                              │  │  2. MarkDispatchSubmitting            │      │
                              │  │  3. pipelineUC.DeployByTemplateID     │      │
                              │  │     (Phase 1 idempotency hinge:        │      │
                              │  │      ErrAlreadyExists → adopt via      │      │
                              │  │      GetWorkflow; ErrWorkflowBeing-   │      │
                              │  │      Deleted → drain to dead)          │      │
                              │  │  4. on success:                       │      │
                              │  │     - update backfill_items.pipeline_ │      │
                              │  │       run_id + wfName              │      │
                              │  │     - MarkDispatched                  │      │
                              │  │  5. on retryable failure:             │      │
                              │  │     - MarkDispatchFailedRetryable     │      │
                              │  │     - if attempts >= N → dead          │      │
                              │  └───────────────────────────────────────┘      │
                              │  ┌───────────────────────────────────────┐      │
                              │  │ resetStale() per cycle (cheap):       │      │
                              │  │   ResetStaleDispatchedItems           │      │
                              │  └───────────────────────────────────────┘      │
                              │                                                     │
                              └─ ctx.Done() → drain in-flight → exit cleanly ────┘
```

### 3.2 Config

```go
type DispatcherConfig struct {
    Tick              time.Duration // default 5s; configurable via env
    LeaseSec          int           // default 60s
    MaxAttempts       int           // default 3
    WorkerCount       int           // default 5 (per Decision A)
    BatchSize         int           // claims per tick; default 4 (≤ WorkerCount)
    JobBufferSize     int           // bounded channel; default 64
    BackoffBase       time.Duration // default 1s (per Decision B)
    BackoffMax        time.Duration // default 30s
    InstanceID        string        // uuid; embedded in slog, debug-only
}

func (c DispatcherConfig) normalized() DispatcherConfig { /* defaults */ }
```

`InstanceID` exists purely for log-correlation. SKIP-LOCKED + lease is the actual correctness lever.

### 3.3 Submit-side: idempotency hinge already wired

The dispatcher calls `pipelineUC.DeployByTemplateID(ctx, templateID, "", assetIDs, deployOpts)` with the per-item deterministic name in `deployOpts.PreallocatedWorkflowName`. All Phase 1 logic (409 adopt, Terminating reject) lights up automatically.

The dispatcher's job is **only to manage the lease + state machine + reaper**. Submit correctness was solved in Phase 1 — the dispatcher just doesn't trash that work.

### 3.4 Concurrency — Decision A

The dispatcher uses **one ticker + a buffered in-memory channel + a pool of N workers** (Decision A, approved 2026-07-05):

```
                    ┌──► worker[0]  ──► submitOne ──┐
                    │                                │
                    ├──► worker[1]  ──► submitOne ──┤
   ticker  ──items─►├──► worker[2]  ──► submitOne ──┼─► Argo
                    │                                │
                    └──► worker[N-1]──► submitOne ──┘
```

- **Single ticker goroutine** claims items via `ClaimNextDispatch` (each iteration runs **one** claim with SKIP LOCKED + lease). This guarantees one claim per tick per replica → fairness + no in-process contention on claim.
- **Bounded channel `jobs` (default cap 64)** holds claimed items awaiting a free worker. **Backpressure**: if the channel is full, the ticker drops the new claim (lease will expire naturally on the next reaper cycle). This prevents OOM during burst-heavy batches.
- **Worker pool (default 5)**: each worker reads from the channel, runs the same `processItem` flow. Throughput ceiling: 5 concurrent submit calls per replica. Tunable via `BACKFILL_WORKER_COUNT`.
- **Multi-replica**: each replica runs the same loop; rows cross replicas via SKIP-LOCKED (one replica's claim excludes the row for the other). No leader election.
- The ticker ALSO runs the reaper SQL (`ResetStaleDispatchedItems`) once per cycle so a single component handles lease recovery.

Trade-off: the bounding is intentional, not lazy. Default 5 was chosen because Argo Server queue already rate-limits; cranking beyond 5 mostly stalls on the API side.

### 3.5 Backpressure + backoff — Decision B

Two related knobs:

**Backpressure** (channel overshoot): ticker's claim is rejected if the channel is full. Trade-off: at high batch-volume, throughput is bounded by workers (e.g. 5 × submit_p99), not by claim rate. **For graceful shutdown**, the ticker exits when ctx is done, then closes the channel; workers exit cleanly when drained.

**Exponential backoff via lease_expires_at** (Decision B, approved 2026-07-05). On retryable failure, instead of a separate `next_retry_at` column, we **set `dispatch_lease_expires_at = NOW() + backoff`**. The `ClaimNextDispatch` WHERE clause already filters on `(lease IS NULL OR lease < NOW())`, so the row is naturally excluded until backoff expires. This is the **no-perception backoff lock** — zero new schema surface, single timestamp serves both "currently being worked on" and "wait this long before retrying".

Backoff schedule (per attempt index, base=1s, max=30s, jitter ±20%):
- attempt 1: no backoff (lease = +1s = small buffer, immediate retry)
- attempt 2: backoff = 1s × 2 = 2s
- attempt 3: backoff = 4s × 2 = 8s
- attempts > MaxAttempts (3): state = dead, lease cleared

Two-equation summary:
- `backoff(attempt) = min(max, base × 2^(attempt-1)) ± 20% jitter`
- `effective_retry_after = max(tick, backoff(attempt))`

---

## 4. Migration of entry points — what each call site looks like after Phase 4

### 4.1 `CreateBackfill` (usecase.go)

Current (partial):
```go
go func() {
    <-uc.materializeAndRunBatch(job.ID, templateID, ...)
    // which itself does go runItems(...)
}()
```
After:
```go
// Items already inserted with dispatch_state='pending' (default).
// No goroutine. Dispatcher picks up next tick.
```

### 4.2 `ResumeJob`

Current: `go uc.runItems(...)` →
After: just call `uc.repo.ResetStaleDispatchedItems(...)` once at start of `ResumeJob`, then signal the dispatcher (in-process if same instance: bump shared channel; cross-instance: just rely on SKIP-LOCKED).

### 4.3 `Rerun`

Current:
```go
uc.repo.PrepareItemsForRerun(ctx, itemIDs)  // resets status='pending', clears timestamps
// then for each runnable:
uc.pipelineUC.UpsertBatchSubtaskRun(..., ForceNewAttempt: true)  // creates new pipeline_run_id (new runID)
go uc.runItems(...)
```
After:
- Rewrite `PrepareItemsForRerun` to: bump `dispatch_generation+1`, set `dispatch_state='pending'`, clear `workflow_name_planned` so the next claim re-derives a NEW name (old gen name would collide with already-reaped workflows or stale `submitted` rows).
- Drop the `UpsertBatchSubtaskRun(..., ForceNewAttempt: true)` call — that creates a new pipeline_run row which we no longer need if we just bump dispatch_generation. The dispatcher loop, upon seeing `dispatch_generation > 1`, derives a fresh workflow_name and a fresh pipeline_run_id. The new pipeline_run_id is just the existing one + dispatch_generation as a discriminator. **Wait — actually we DO need a new pipeline_run row per dispatch_generation** because each rerun has its own lifecycle/cost/etc. Leave `UpsertBatchSubtaskRun(..., ForceNewAttempt: true)` as-is; the new logic adds a final `UPDATE backfill_items SET dispatch_state='pending', dispatch_generation=dispatch_generation+1 WHERE id IN ($itemIDs)`.

### 4.4 `ContinueFull` (pilot → full)

Same shape as `ResumeJob`: `dispatch_state='pending'` already after `PrepareItemsForRun` (or equivalent).

---

## 5. Patch-cleanup plan

After step 4 lands, delete these (now redundant or actively harmful):

| What | Where | Why deletable |
| --- | --- | --- |
| `runItems` | backfill/usecase.go:435 | dispatcher replaces it |
| `executeItem` | backfill/usecase.go:481 | dispatcher replaces it |
| `runAlreadySubmitted` | backfill/usecase.go:1559 | Deterministic name already removes the heuristic race window |
| `ClaimNextItem` | postgres/backfill_repo.go:918 | dispatcher claims via `dispatch_state`, not `status` |
| `ResetStaleItems` | postgres/backfill_repo.go:946 | replaced by `ResetStaleDispatchedItems` |
| `StartReaper` / `StopReaper` reaper ticker | backfill/usecase.go:83 | dispatcher reimplements reaper inline |
| `ResumeIncompleteBatches` startup scan | backfill/usecase.go:120 | dispatcher restart recovery is automatic via SKIP-LOCKED; no explicit scan needed |
| `batchCancels` map (StopBatchRuns cancellation) | pipeline/usecase.go:84 | dispatcher needs a different cancellation model (ctx cancel + lease expiry handles "stop") |

The patch-cleanup is the *only* phase that's really risky (large blast radius, lots of tests). Split it into a **separate commit** from the dispatcher introduction. We'd merge:

- **Commit A:** dispatcher.go + repo methods + migration. **Keeps** legacy paths for now (just routes new entries through dispatcher while legacy path is wired through `BACKFILL_DISPATCH_MODE=legacy` env var).
- **Commit B:** deletion of legacy + flag removal + UI/test updates. Per-file small.

This matches the "Per-phase commit cadence" rule (#12).

---

## 6. Feature flag & rollout

`BACKFILL_DISPATCH_MODE = legacy | outbox` (default `legacy`). When `outbox`:
- The 4 entry points go to the new path (enqueue only).
- `Dispatcher.Run` is started from `cmd/server/core.go`.
- Legacy `runItems`/`executeItem` call sites get `if dispatchMode == legacy { go runItems(...) } else { /* nothing — dispatcher handles it */ }`.

Two safe ways to flip:
1. **Configurable per-deployment** (`BACKFILL_DISPATCH_MODE` env). Dev can flip with one redeploy.
2. **Per-job in DB** (`backfill_jobs.dispatch_mode`). More flexibility but more surface; defer to a follow-up.

**Critical: never run legacy + outbox on the same `backfill_items` table concurrently against overlapping items.** Why: legacy `ClaimNextItem` and `dispatcher ClaimNextDispatch` look at different columns, so they WILL both claim the same row, and one will dispatch successfully while the other attempts to re-dispatch (now adversarial with the deterministic name → `AlreadyExists` → adopt, which is *safe* but produces noisy logs). Acceptable in dev but ugly in prod. **Sequence: drain legacy first.**

### 6.1 Recommended rollout sequence

| Step | What | Risk | Rollback |
| --- | --- | --- | --- |
| 1 | Merge migration `064` only | low — additive, defaults match legacy | `ALTER TABLE DROP COLUMN ...` |
| 2 | Merge dispatcher + repo methods (Commit A), still no entry point switched | low — dispatcher has nothing to claim yet | revert commit |
| 3 | Per-deployment flip dev `BACKFILL_DISPATCH_MODE=outbox` and verify | medium — first time it touches prod-like load | env flip revert |
| 4 | Per-deployment flip staging outbox, monitor for 24h | medium-high | env flip |
| 5 | Drain legacy: wait until all "in-flight" items have `dispatch_state=submitted` or `dead` | medium — requires reading the rows | tolerate (legacy was non-fatal anyway) |
| 6 | Merge patch-cleanup (Commit B): delete legacy code | low-medium — covered by tests | `git revert` |

---

## 7. Risks beyond what's already in the plan

These are surfaced for explicit discussion:

1. **PG advisory lock surprise**: SKIP-LOCKED can interact badly with LOCK TABLE / row-level triggers. Audit `backfill_items` triggers before merge — none expected (no CDC triggers on it; `asset_events` table is the only CDC table). **Mitigation:** confirm with DBA team that `backfill_items` is CDC-free.

2. **Mixed-mode lock contention**: once a few hundred items are mid-dispatch, the hot partial index gets high BTree contention during batched updates. PostgreSQL handles this well at < 5K TPS; with backfill_total_count_max=100k we'd want to benchmark. Out of scope for v1.

3. **`prepareItemsForRerun` atomicity**: rewriting the SQL to bump generation+reset dispatch_state in one UPDATE — race window if a dispatcher is mid-submit on that exact row. **Mitigation:** `WHERE dispatch_state IN ('submitted','dead','pending')` only — i.e. don't touch rows that are claimed/submitting (claim will fail naturally afterwards because lease is unset but state is `pending`).

4. **Phased `MarkDispatched` vs `ReplaceDispatch`**: also need to write back to `backfill_items.pipeline_run_id` and `backfill_items.workflow_name` so the existing `ReportBatchSubtaskFailure` / `SyncJobProgress` paths still work. Two new repo methods OR we extend `MarkDispatched` to do both. **Design choice:** extend `MarkDispatched` to be transactional about both tables (call as in-process orchestration, no WithTx needed because `backfill_items` is the only fork between two rows).

5. **`TestListExecutionTargets_Default` and `TestDeployByTemplate_*`** : legacy test paths still use `runItems` indirectly via `materializeAndRunBatch`. After Phase 4 step 5, those tests must be updated. **Mitigation:** flag-gate in dispatcher and leave test infra intact for now; update tests in Commit B.

---

## 8. Open questions for the user

These need answers before coding:

1. **Default `Tick` value** — 2s or 5s? Plan said 2s. Workflow: 100k items, single replica, expected throughput 50-200 items/min ⇒ 2s is fine. Default 2s.
2. **Default `LeaseSec`** — 30s or 60s? Argo submit p99 historically under 30s in dev; 60s is safer. **Default 60s; dispatcher extend-rate (refresh lease at 80% of leaseSec) reduces risk of running over.**
3. **`MaxAttempts`** — 3 matches existing reaper; keep.
4. **Do we want **fast-fail** on first failure (no backoff) or **exponential backoff** per attempt?** Plan doesn't specify. Recommend: keep first-attempt immediate, then exponential with jitter (1s, 5s, 30s) — burned-in backoff to handle transient Argo Server blips.
5. **`go` library**: any preference? Default **`golang.org/x/sync/errgroup`** for the per-item workers (we don't have anything exotic in dependencies yet). Easy to swap later.

These are all small; I'd take your answer on (4) since it affects behaviour, the rest are defaults I can document.
