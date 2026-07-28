# Decisions — CYB-3691

## 2026-07-20: gauge placement

Chose to set `DispatcherStaleItems` at the **start** of `reconcileActiveJobs` (pre-sync gap), not after. Reasoning: if set after reconciler has already fixed the gap, the gauge is always 0 in steady state and provides no signal. Pre-sync gap > 0 means the gap exceeds one reconciler cycle (60s), which is actionable.

## 2026-07-20: GMP vs remote_write

Recommended GMP (Managed Prometheus) over self-managed remote_write because:
- GKE cluster already has Prometheus exporter running
- GMP auto-discovers pod annotations, no additional configmap
- Cloud Run supports `run.googleapis.com/prometheus_scrape` annotation
- remote_write requires managing GCP Monarch endpoint + auth

Deferred to follow-up: actual GMP enablement depends on GKE cluster configuration and may require cluster-admin privileges.

## 2026-07-27: writer was wired but never called (gap fix)

Follow-up audit found the Cloud Monitoring path shipped only half-done:
`cloudmonitoring.Writer` was implemented and wired onto the backfill usecase
(`core.go` → `SetMonWriter`), and the backend SA `cyber-databrew-dev@` already
holds `roles/monitoring.metricWriter` — but `reconcileActiveJobs` only called
`metrics.DispatcherStaleItems.Set(n)` (the in-process Prometheus gauge) and
**never invoked `monWriter.WriteInt64Metric`**. Result: the GCP "Dispatcher
Metrics" dashboard (queries `custom.googleapis.com/dispatcher/stale_items`) had
no data source and rendered empty.

- **Decision**: add the missing `monWriter.WriteInt64Metric(ctx,
  "dispatcher/stale_items", n)` call right after the gauge `Set`, nil-guarded
  and best-effort (write errors logged, never block the reconcile cycle). Runs
  every reconciler tick (~60s) on the Cloud Run instance.
- **Also found**: the GMP path via `run.googleapis.com/prometheus_scrape` on
  Cloud Run does **not** work — that annotation is not a supported Cloud Run
  scrape feature and is silently dropped (absent on the serving revision despite
  `backend-dev.sh` setting it). Full `backend_*` Prometheus suite in GCP would
  need a collector sidecar (tracked separately, not this change). This change
  only closes the already-built custom-metric path for the dispatcher gauge.
- **No new IAM**: `monitoring.metricWriter` already granted to the runtime SA.
