# Tasks — CYB-3679

- [x] Migration `dispatcher_configs` (Atlas CI applies)
- [x] postgres.DispatcherConfigRepo (List/Upsert/Delete)
- [x] backfill: per-cycle config refresh (fail-safe snapshot), paused channel
      skip + gauge, governor applyConfig (ceiling/floor/rate/batch),
      submit_batch override, DispatcherStatus (configured vs effective +
      suppression reason)
- [x] Admin API GET/PUT/DELETE /dispatcher/clusters (range validation → 400)
- [x] UI: 调度调参 page (table + chips + edit modal + pause switch +
      restore-defaults), DLQ 人话 in batch detail
- [x] Metrics: backend_dispatcher_channel_paused; SLO alert definitions doc
- [x] Tests: backend new code 100% (validation table, fail-safe refresh,
      save/delete branches, applyConfig math, paused skip + gauge, batch
      override, status merge); frontend lib tests (ranges, chips, 人话)
- [x] Live dev verification: PUT config → paused batch stalls → unpause
      completes; GET shows effective vs configured
