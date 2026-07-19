# Tasks

- [x] Cluster sharding: group jobs by ResolveTargetClusterID, per-cluster
      goroutine + advisory lock (fnv-scoped keys, supersedes global lock).
- [x] Governor: token bucket + AIMD (floor/¼, ×1.5 recovery, slow-majority
      distress) + effective-concurrency gauge.
- [x] Self-kick on full batch; suppressed when breaker trips.
- [x] Error classification (permanent list; unknown→transient) + durable
      submit_attempts cap → DLQ; permanent path doesn't burn the cap.
- [x] DLQ endpoints (list + retry) with governed re-entry.
- [x] Boot jitter; channel breaker (10 consecutive transient).
- [x] Migration submit_attempts + atlas hash (NOT pre-applied — Atlas CI owns
      DDL, 3680 lesson).
- [x] Tests (new code 100% where unit-testable): governor math, classification
      table, sharding + isolation, self-kick both ways, breaker, attempt-cap
      lifecycle, permanent path, cancelled-ctx stop, DLQ list/retry + error
      branches, postgres attempts/reset SQL, per-cluster lock keys.
- [x] Gates: gofmt/vet/full suite green.
- [ ] PR → merge → CI (applies migration) → live verify.
