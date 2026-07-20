# Dispatcher SLO 告警定义(CYB-3679)

告警策略在 GCP Cloud Monitoring(Managed Prometheus)侧配置,本文件是
仓库内的权威定义 —— 指标名以 `backend/internal/metrics/backend.go` 为准。

| 告警 | PromQL(示意) | 阈值 / 窗口 | 含义与处置 |
|---|---|---|---|
| 通道被暂停过久 | `backend_dispatcher_channel_paused == 1` | 持续 >10min | 有人暂停后忘记恢复;确认后在「调度调参」页恢复 |
| 背压压制过久 | `backend_dispatcher_effective_concurrency / on(cluster) group_left() (配置值)` < 0.5 | 持续 >10min | AIMD 长期压半:集群容量不足或 Argo controller 饱和,查 workqueue 深度 |
| 通道熔断突增 | `increase(backend_dispatcher_channel_breaker_total[10m]) > 3` | 10min | 集群持续瞬态失败(API server / 网络);看 submitter 日志 `channel breaker tripped` |
| DLQ 突增 | `increase(backend_dispatcher_dlq_total[10m]) > 20` | 10min | 批次大面积永久失败(坏模板 / 资产缺失);到批次详情页看人话原因,修复后 `dlq:retry` |
| 模板钉版回退 | `increase(backend_dispatcher_template_fallback_total[1h]) > 0` | 1h | 新批次未写 template_version(写入路径回归)——立即排查 |
| 下发时延劣化 | `histogram_quantile(0.95, rate(backend_dispatcher_submit_duration_seconds_bucket[10m])) > 2` | 10min | Argo API 变慢;AIMD 会自动降速,持续则扩容 controller |

**backlog 年龄(backlog_age>10min)**:pending 项最早创建时间无直接指标,用
DB 查询型告警(Cloud SQL insights / 定时探针)或在批次页人工观察;若需要
指标化,后续在 submitter cycle 里导出 `backend_dispatcher_oldest_pending_age_seconds`。
