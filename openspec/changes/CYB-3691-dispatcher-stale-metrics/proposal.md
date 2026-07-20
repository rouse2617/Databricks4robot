# Proposal — CYB-3691

## Why

当前批量任务的下发链路没有 webhook，状态回写完全依赖 watcher 更新 pipeline_runs + reconciler 同步 backfill_items。缺少一个关键监控：**watcher 已经看到 terminal 但 backfill_items 还没同步的 gap 有多大**。没有这个指标，用户可能看到 UI 上 item 还处于 "submitted" 但实际已经在 Argo 跑完了。

同时，这些 Prometheus 指标目前只在自建 Grafana 上可见，GCP 一侧的 Cloud Monitoring 看板看不到 backend 层面的 dispatch 健康状况。

## What Changes

### New Capabilities

1. **dispatcher stale-items gauge** — 后端新增 `backend_dispatcher_stale_items` Prometheus gauge，在 reconciler 每次轮回时计算 terminal pipeline_runs 但 backfill_items 未同步的数量
2. **Grafana dashboard panel** — 在 `backend-observability.json` 中新增 stale items 面板（stat + timeseries）
3. **GCP Monitoring 集成** — 配置 Prometheus remote_write 或 GMP，使 `backend_dispatcher_*` 指标出现在 GCP Cloud Monitoring

### Modified Capabilities

无修改，纯新增监控层。

## Impact

- **Affected code**: `backend/internal/metrics/backend.go`, `backend/internal/postgres/backfill_repo.go`, `backend/internal/usecase/backfill/usecase.go`, `deploy/k8s/monitoring/dashboards/backend-observability.json`, `deploy/k8s/monitoring/prometheus-configmap.yaml`
- **New APIs**: 无（纯 Prometheus metrics + dashboard）
- **Dependencies**: 无新增

## Scope

- **In scope**: stale items gauge + 埋点 + dashboard + GCP Monitoring 集成方案
- **Out of scope**: 其他 dispatch metric（DLQ、breaker、concurrency 等已有）；webhook 链路

## Success Criteria

- [ ] `backend_dispatcher_stale_items` 在 `/metrics` 端点出现，数值与 DB 真实 gap 一致
- [ ] Grafana backend-observability 看板有 stale items 面板
- [ ] GCP Cloud Monitoring 能看到 `backend_dispatcher_*` 指标
