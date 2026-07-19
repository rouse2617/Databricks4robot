# Proposal — CYB-3679

## Why

Dispatch tuning (per-cluster concurrency, submit batch, token-bucket rate)
was compiled env config: retuning a cluster or pausing dispatch required a
deploy. Operators need per-cluster knobs that bite within one tick, plus
visibility into WHY a cluster dispatches below its configured capacity.
Design v1.5 §04/§07.

## What Changes

### Added Capabilities
- **dispatcher_configs table** — one row per cluster (max_concurrency,
  submit_batch, rate_per_sec, paused, updated_by). Absent row = compiled
  defaults. The submitter re-reads all rows at the start of every cycle
  (≤15s to take effect). Refresh failure keeps the last snapshot — a DB blip
  cannot silently un-pause a cluster.
- **Admin API** — `GET /dispatcher/clusters` (configured vs effective +
  suppression reason), `PUT /dispatcher/clusters/:cluster` (range-validated:
  concurrency [1,256], batch [1,200], rate [0.1,100]),
  `DELETE /dispatcher/clusters/:cluster` (back to defaults). Save/delete
  refresh the snapshot and kick a cycle immediately.
- **Governor online retune** — applyConfig adjusts ceiling/floor (effective
  clamps into range; AIMD keeps operating around the new ceiling), token
  bucket SetLimit/SetBurst, per-job batch override. Paused wins over
  everything: the channel skips its whole cycle,
  `backend_dispatcher_channel_paused{cluster}` = 1.
- **UI「调度调参」page** — table of clusters: 生效/配置 concurrency with
  suppression chip (已暂停 red / 背压降速中 orange), batch, rate, pause
  switch, range-validated edit modal, restore-defaults. Polls every 15s.
- **DLQ 人话** — batch detail failure reasons rendered in plain language
  (step-count over limit, resource over ceiling, template/asset missing,
  transpile error, attempts exhausted); raw message stays in the tooltip.
- **SLO alert definitions** — docs/review/dispatcher-slo-alerts.md (paused
  >10min, suppression >10min, breaker spikes, DLQ spikes, pin fallback,
  submit p95) over the exported dispatcher_* metrics.

### Non-Goals
- Alert POLICY provisioning (lives in GCP Monitoring, not this repo).
- `oldest_pending_age_seconds` metric (documented follow-up).
