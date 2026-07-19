# Tasks — CYB-3681

- [x] Bulk-pull watcher: per-(cluster,ns) LIST `completed=false`, RV change
      gate + every-10th recalibration, bounded residual GETs, LIST-failure
      fallback, `WATCHER_MODE` flag (`watcher_bulk.go`)
- [x] Exit-hook opt-in: `ARGO_EXIT_HOOK_ENABLED` (default false) gates
      SetArgoRunWebhook in core.go; webhook endpoint retained
- [x] Transpiler step-count guardrails (warn 200 / reject 500)
- [x] Tests: 100% coverage on all new functions (RV gate, apply/gate/
      recalibrate, terminal eviction, residual budget, LIST-error fallback,
      resolve-error fallback, empty-name residual, e2e bulk + legacy modes,
      step-count table, config flag)
- [x] Live dev verification: batch completes via pull-only writeback, no
      exit-notify pod in submitted workflows
