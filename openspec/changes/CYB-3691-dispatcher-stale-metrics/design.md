# Design — CYB-3691

## 1. Stale Items Gauge

### 1.1 Metric Definition

```
Name: backend_dispatcher_stale_items
Type: Gauge
Labels: (none — first version; cluster label added later if needed)
Help: Number of terminal pipeline_runs whose backfill_items have not yet been synced (watcher→reconciler gap)
```

### 1.2 SQL Query

```sql
SELECT COUNT(*) FROM backfill_items bi
INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id
WHERE bi.status IN ('pending', 'submitted')
  AND pr.status IN ('Succeeded', 'Failed', 'Error')
```

### 1.3 Placement

在 `reconcileActiveJobs` 轮回的**末尾**设置 gauge。理由：

- 设置在 reconciler 执行之后 → 反映 residual gap（reconciler 修不了的遗留项）
- 设置在 reconciler 之前 → 反映最大 gap（reconciler 刚修完就归零，不反映问题）
- 选择"之后": 如果 residual > 0，说明有明确 bug（item 没有 pipeline_run_id、或 status 映射不完整）

但如果 reconciler 每次都能修完，gauge 永远是 0 —— 那就失去了"gap 是否存在"的信号。所以折衷：**在 reconcileActiveJobs 开始时设置一次**（pre-sync gap），**结束时再设置一次**（residual gap）。两个 gauge 合并为一个，取 pre-sync 值（最大 gap）：因为 pre-sync 如果为 0，说明 watcher→reconciler 间隙控制在 1 cycle 内。如果有 spike，说明 gap 超过 60s。

### 1.4 新增方法

- `BackfillRepository.CountStaleBackfillItems(ctx) (int, error)` — 接口方法
- `BackfillRepo.CountStaleBackfillItems(ctx) (int, error)` — Postgres 实现

### 1.5 调用链路

```
StartJobReconciler(60s)
  └─ reconcileActiveJobs
       ├─ metrics.DispatcherStaleItems.Set(CountStaleBackfillItems)  ← NEW
       ├─ FindActiveJobs → per-job sync
       └─ ...
```

## 2. Grafana Dashboard Panel

在 `deploy/k8s/monitoring/dashboards/backend-observability.json` 中新增一行面板：

- **Type**: Stat (当前值) + Timeseries (趋势)
- **Query**: `backend_dispatcher_stale_items`
- **Alert**: > 0 持续 5min → 健康门控

## 3. GCP Cloud Monitoring 集成

### 方案对比

| 方案 | 复杂度 | 成本 | 说明 |
|------|--------|------|------|
| **GMP (Managed Prometheus)** | 低 | 按摄入量计费 | GKE 侧自动 scrape pod annotations；Cloud Run 需 annotation `run.googleapis.com/prometheus_scrape: "true"` |
| Prometheus remote_write | 中 | 按摄入量计费 | 现有自建 Prometheus 加 remote_write 到 GCP Monarch endpoint |
| Custom Metric API (direct write) | 高 | 按 API 调用计费 | 后端新增 Cloud Monitoring client，直接写 custom metric |

**推荐**: GMP。Cloud Run 服务加 annotation 即可，GKE side 已有 prometheus exporter，配置 GMP Operator 自动发现。

### 具体配置

**Cloud Run**: `gcloud run deploy` 加 `--set-build-env-vars=...` 或在 service YAML 中加：

```yaml
annotations:
  run.googleapis.com/prometheus_scrape: "true"
  run.googleapis.com/prometheus_port: "8080"
```

**GKE Prometheus**: 启用 GMP Operator，现有 scrape config 加 `collector` 标记即可。

### 指标前缀

一旦接入 GMP，GCP Cloud Monitoring 中指标名变为：
```
prometheus.googleapis.com/backend_dispatcher_stale_items/gauge
```
用户可在 Cloud Monitoring 看板中通过 MQL 查询：
```
fetch prometheus_target
| metric 'prometheus.googleapis.com/backend_dispatcher_stale_items/gauge'
| align next_older(1m)
```
