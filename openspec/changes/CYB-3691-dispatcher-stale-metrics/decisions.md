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
