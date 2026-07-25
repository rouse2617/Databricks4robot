# Design — CYB-TBD

## Architecture Context
- **Constraints**: Go backend with Gin; PostgreSQL is the authoritative batch queue; target-to-cluster resolution currently lives in runtime/pipeline configuration, not as a persisted `backfill_jobs.cluster_id`.
- **Goals**: Prevent global candidate-window head-of-line blocking across execution targets with a small backend-only change.
- **Non-Goals**: Do not add migrations or persist `cluster_id` in this PR; do not redesign the submitter governor or Argo backpressure model.

## Affected Modules
- `backend/internal/postgres/backfill_submit_queue.go` — change the candidate query to use per-target ranking.
- `backend/internal/postgres/backfill_submit_queue_integration_test.go` — verify target fields still round-trip through the queue selection path.
- `backend/internal/usecase/backfill/submitter_cyb3678_test.go` — add an in-memory starvation regression.

## Architecture Decisions

### Decision 1: Fair by target key in the query layer
- **Approach**: Compute a stable target key from `filter_json` (`targetId`, `target_id`, `executionTargetId`, `execution_target_id`, default bucket when missing), rank jobs by `created_at` within each target, and keep up to a per-target cap before applying the global limit.
- **Alternative**: Persist `cluster_id` on `backfill_jobs` and perform true per-cluster SQL scheduling.
- **Rationale**: Persisted `cluster_id` is the ideal long-term model, but it requires a migration and target-resolution lifecycle decisions. Target-level fairness matches the current data model and removes the observed starvation shape without touching off-limits migration files.
- **Trade-off**: Multiple targets may resolve to the same cluster, so this is target-fair rather than perfectly cluster-fair.
- **Rollback**: Restore the previous global oldest-first query; no data rollback is needed.

### Decision 2: Keep submitter cluster grouping unchanged
- **Approach**: Leave `runSubmitterCycle` grouping by `ResolveTargetClusterID` as-is. The repository returns a more diverse candidate window; existing per-cluster locks/governors/backpressure continue to own dispatch decisions.
- **Alternative**: Move target/cluster resolution entirely into PostgreSQL.
- **Rationale**: Keeping runtime target resolution in Go avoids duplicating config semantics in SQL and limits the blast radius to candidate selection.

## Data Flow

```mermaid
flowchart LR
  PG["PostgreSQL FindSubmittableJobs"] --> Fair["rank pending jobs per target key"]
  Fair --> Jobs["bounded fair candidate set"]
  Jobs --> Group["submitter groups by resolved cluster"]
  Group --> Channel["per-cluster channel / governor / lock"]
```

## Data Model Changes

None.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Target-fair is not true cluster-fair | Several targets in one unhealthy cluster can still consume more candidates than one target in another cluster | Keep per-target cap bounded; leave persisted `cluster_id` as follow-up for exact fairness |
| SQL target-key extraction misses a legacy key | Such jobs fall into the default bucket | Include all currently used target keys and cover missing-target behavior |
