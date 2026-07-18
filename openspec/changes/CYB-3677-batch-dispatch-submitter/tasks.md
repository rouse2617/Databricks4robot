# Tasks

- [x] `CreateBatchJob`: persist job born `running` with `template_version`
      column pinned (mirrored in filter_json for one release), kick the
      submitter, no in-memory goroutine; `BATCH_DISPATCH_MODE=legacy` keeps the
      old path (rollback flag, default `submitter`).
- [x] `FindSubmittableJobs`: select + scan `COALESCE(template_version, 0)`.
- [x] Submitter pinning semantics: column > filter_json > active-with-warning
      (`backend_dispatcher_template_fallback_total` metric).
- [x] Cross-instance cycle mutual exclusion: `pg_try_advisory_lock` on a
      dedicated pooled connection (`realDB.WithAdvisoryLock` →
      `BackfillRepo.WithSubmitterCycleLock` → optional interface in
      `runSubmitterCycle`; lock error degrades to unguarded).
- [x] Wiring: `cmd/server/core.go` sets dispatch mode + kicker after the
      submitter starts.
- [x] Migration `20260719120000_batch_jobs_processing_to_running.sql`
      (stranded `processing`-with-pending jobs → `running`; idempotent) +
      atlas.sum rehash.
- [x] Tests (new code 100%):
      - pinning acceptance (column wins / filter_json fallback / active
        fallback + metric), `templateVersionFromBackfillFilter` table
      - cycle lock (skip when held / run under lock / error → unguarded /
        no-locker fallback / nil-wiring no-op / find-error abort)
      - batch dispatch (submitter mode persists+kicks+no-goroutine, nil
        kicker, explicit + active version pin, legacy mode byte-for-byte,
        mode parsing, all error paths)
      - postgres unit (`WithSubmitterCycleLock` all branches) + integration
        (`TestFreshDB_SubmitterCycleLock` two-session exclusion,
        template_version round-trip) — integration runs in CI fresh-db job
- [x] Local gates: gofmt, go vet (incl. integration tag), full `go test ./...`
      (57 pkgs green).
- [ ] Apply migration to dev, deploy dev, live verification (batch create →
      submitter dispatch → workflows in cluster; kill-resume drill).
- [ ] PR → dev, merge, CI/CD watch, post-merge dev verification.
