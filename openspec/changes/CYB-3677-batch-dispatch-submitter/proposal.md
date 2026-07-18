# Proposal — CYB-3677

## Why

The legacy batch entry (`POST /api/v1/runs/batch` → `CreateBatchJob`) dispatches
runs from a **one-shot in-memory goroutine** (`processBatchJob`, 20-worker
pool). A backend restart (rollout/OOM) mid-batch silently strands every
not-yet-submitted item: the job is stuck in `processing` (a status the durable
submitter never selects), items stay `pending` forever, and the reconciler only
repairs ledger rows — it never submits. Field-observed as "batch stuck running,
tasks silently missing".

A second, independent defect rides the same path: the template version pinned
at batch creation is stored only in `filter_json`, never in the
`backfill_jobs.template_version` column, and `FindSubmittableJobs` doesn't
select that column — so any submitter-driven dispatch would silently fall back
to the template's **current active version**, mixing versions within one batch
(P0: same-batch semantic inconsistency).

Design doc (v1.5, five review rounds):
https://claude.ai/code/artifact/f618dcf2-ec3e-4a99-a924-4daaf3abad90

## What Changes

### Modified Capabilities
- **pipeline** (batch dispatch): `CreateBatchJob` becomes persist-only — write
  job (status=`running`, `template_version` column populated) + items, kick the
  existing durable backfill submitter, return. The submitter owns submission:
  crash-resumable (boot eager + re-list), idempotent (deterministic workflow
  name + AlreadyExists convergence), and now **template-version-pinned** (the
  column value is authoritative; NULL legacy rows fall back to
  `filter_json.template_version`, then active, with a warning metric).
- A **cycle-level Postgres advisory lock** makes multi-instance submitters
  mutually exclusive (session-scoped; auto-released on connection loss = a
  natural lease). Losing instances skip the cycle.
- `BATCH_DISPATCH_MODE=submitter|legacy` feature flag (default `submitter`);
  legacy path retained for one release as rollback.
- One-time data migration: stranded `processing` jobs that still have pending
  items → `running`, so the submitter finishes them.

## Impact
- Affected code: `usecase/pipeline/batch.go` + `usecase.go` (setters),
  `usecase/backfill/submitter.go`, `postgres/backfill_submit_queue.go`,
  `internal/config`, `internal/metrics`, `cmd/server/core.go`, one migration.
- No API-surface change (`POST /runs/batch` request/response unchanged; 202
  semantics identical). No transpiler/CRD change.
- Cancel path unchanged: `StopBatchRuns` already sets job `cancelled`, which
  the submitter never selects; the in-memory `cancelBatch` becomes a harmless
  no-op in submitter mode.

## Scope
- In: the four changes above + broad tests (pinning acceptance, crash-resume
  semantics, flag parity, lock mutual exclusion, migration idempotency).
- Out (follow-up CYBs): per-cluster sharding/self-kick/DLQ/governor
  (CYB-3678), online tuning + metrics UI (CYB-3679), CM dedup (CYB-3680),
  exit-pod removal + DB-driven writeback (CYB-3681).

## Success Criteria
- [ ] Batch pinned to version X keeps submitting X even after the template's
      active version moves to Y mid-batch (acceptance test).
- [ ] Kill the process mid-batch → restart → remaining pending items are
      submitted; no duplicates (deterministic name convergence).
- [ ] `BATCH_DISPATCH_MODE=legacy` restores the previous behavior byte-for-byte.
- [ ] Two concurrent submitter instances: exactly one runs a cycle (lock),
      the other skips without error.
- [ ] Migration moves only `processing`-with-pending jobs; re-running it is a
      no-op.
