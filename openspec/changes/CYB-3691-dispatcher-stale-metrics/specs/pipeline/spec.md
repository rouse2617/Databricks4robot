## ADDED Requirements

### Requirement: 批量任务 watcher→reconciler gap metric

The system SHALL expose a Prometheus gauge (`backend_dispatcher_stale_items`) that reports how many backfill_items are in a non-terminal status (`pending` or `submitted`) while their linked pipeline_run is already terminal (`Succeeded`/`Failed`/`Error`). The gauge SHALL be refreshed each reconciler cycle (~60s) and SHALL reflect the value observed at the start of the cycle (the "pre-sync gap" before the reconciler attempts to resolve it).

**Priority**: P2 (Nice-to-have)
**Rationale**: 没有 webhook 的链路中，watcher→reconciler gap 没有可见性。gauge > 0 持续多个 cycle 意味着 item 可能永久卡住。

### Requirement: GCP Cloud Monitoring 可观测 dispatch 指标

The system SHALL make its Prometheus dispatch metrics (`backend_dispatcher_*`) discoverable by GCP Cloud Monitoring via Google Managed Prometheus (GMP) or Prometheus remote_write. The backend SHALL NOT add a new direct dependency on the Cloud Monitoring API.

**Priority**: P3 (Nice-to-have, deferred)
**Rationale**: 配置层改动，不影响 backend 代码。需评估现有 GKE 集群是否已启用 GMP。
