# Decisions — CYB-3677

## INV-1 audit outcome: advisory lock, NOT a unique index

Design v1.5 called for auditing whether `pipeline_runs (batch_job_id,
asset_id)` needs a unique index against multi-instance double-mint. Audit
result: **a unique index is wrong here** — `ForceNewAttempt` reruns
legitimately create multiple runs per (batch, asset) (see
`batchSubtaskWorkflowName(_, _, _, runID, unique=true)`), and a partial index
cannot express "one non-superseded attempt". Cross-instance economy is instead
guaranteed by the **cycle-level advisory lock**, and correctness by INV-2.

## INV-2 audit outcome: deterministic naming holds

`batchSubtaskWorkflowName` with `unique=false` (the submitter's normal path)
is `<pipeline>-batch-<job8>-<asset12>` — fully deterministic per (job, asset).
Racing instances mint the SAME name and converge at the cluster via
AlreadyExists + UID backfill. `unique=true` (explicit reruns) is intentionally
non-convergent.

## Lock design: session advisory lock on a pinned connection

- `pg_try_advisory_lock` on a **dedicated pooled connection** (session locks
  are per-session; routing through the pool's per-call dispatch would lock a
  random connection).
- Auto-release on connection drop = natural lease; a crashed holder never
  wedges peers; Cloud Run rollout hands over within one tick.
- **Not** the CYB-3491 failure mode: no row locks, no transaction spanning
  Argo calls — fn's DB work still uses the normal pool.
- Lock ERROR (transient DB trouble) degrades to running unguarded:
  availability over economy, correctness already held by idempotency.
- Fixed single key this change; CYB-3678 shards to per-cluster keys.

## Pinning precedence: column > filter_json > active(+metric)

The column is authoritative and, when set, the submitter never consults the
active version. filter_json remains a fallback for pre-3677 rows and is still
mirrored on new rows for exactly one release so a legacy-flag rollback
inherits the pin.

## Flag default: submitter

`BATCH_DISPATCH_MODE` defaults to the new path; `legacy` is rollback-only and
scheduled for removal one release after G4 (dedicated removal CYB, owner:
ruipeng).
