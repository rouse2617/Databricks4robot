# Proposal — CYB-3678

## Why

The submitter ran one global cycle: a slow/unreachable cluster head-of-line
blocked every other cluster's dispatch; there was no rate protection for the
Argo controller (submission could outrun consumption); any non-AlreadyExists
error immediately failed the item (a network blip mass-failed innocent
items); and a poison item retried forever. Design v1.5 §04/§08 (C5, C8, C9,
C16) and review rounds 2–3.

## What Changes

### Modified Capabilities
- **pipeline** (batch dispatch): per-CLUSTER channels — each cluster gets its
  own goroutine, advisory lock (base^fnv64a(cluster)), governor, and breaker.
  Slow clusters only stall themselves.
- **Governor**: per-cluster token bucket (`BACKFILL_CLUSTER_RATE`, default
  10/s — sized to controller consumption per CNOE baseline) + AIMD
  concurrency (halve on distress, floor=configured/4, ×1.5 recovery).
- **Self-kick**: a channel that fills a whole batch re-kicks immediately —
  short-task/large-node clusters stay fed without waiting for the tick.
- **Error classification + DLQ**: only explicitly-permanent errors fail
  immediately; everything else retries with a durable per-item attempt cap
  (`BACKFILL_MAX_SUBMIT_ATTEMPTS`, default 5 → failed). DLQ surface:
  `GET /runs/batch/:id/dlq` + `POST /runs/batch/:id/dlq:retry` (revive with a
  fresh cap; rejoins the governed pipeline — mass retry can't storm).
- **Channel breaker**: 10 consecutive transient failures skip the rest of the
  cluster's cycle (items stay pending; a down cluster comes back).
- **Boot jitter** (0–5s): multi-instance cold starts de-align (C17).

## Impact
- `backfill/submitter{,_cluster}.go`, `postgres/backfill_submit_queue.go`,
  pipeline usecase (cluster resolution + DLQ), batch handler + routes,
  metrics, one migration (`submit_attempts`). API adds two endpoints;
  existing surface unchanged.

## Success Criteria
- [ ] Two clusters: one held/slow → the other dispatches unaffected.
- [ ] Transient error → pending with attempts++; cap → failed (DLQ); retry
      API revives with fresh cap and kicks.
- [ ] Permanent error fails immediately without burning the cap.
- [ ] Full batch → immediate self-kick; drained → none.
- [ ] Governor: distress halves to floor, health recovers ×1.5 to ceiling.
