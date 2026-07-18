# Tasks — CYB-monitoring-baseline

## Sprint 1 [L1] Uptime + Alert(~1 天)
- [ ] [backend] `deploy/terraform/monitoring/` 新建目录 + `main.tf`(声明 `google_monitoring_uptime_check_config` × 5)
- [ ] [backend] `deploy/terraform/monitoring/notification.tf` 声明 `google_monitoring_notification_channel` 指向 `feishu-monitoring-relay` Cloud Run 的公开触发 URL
- [ ] [backend] `deploy/terraform/monitoring/policies.tf` 声明 `google_monitoring_alert_policy` × 5(每个 uptime 一个 P1 policy,链到通知渠道)
- [ ] [backend] `deploy/terraform/monitoring/README.md` 说明 apply 步骤 + 手工验证方法
- [ ] [backend] `terraform apply` 到 dev(在维护窗口),手工 stop 一次 backend-dev revision 触发告警验证
- [ ] Runbook `docs/runbook/README.md` 骨架

## Sprint 2 [L2] Cloud Run /metrics 接入 GMP(~3 天)
- [ ] [backend] 启用 GCP Managed Service for Prometheus API(`monitoring.googleapis.com` 已开;`monitoring.googleapis.com/PrometheusEnabled` 需确认)
- [ ] [backend] Cloud Run backend-dev 挂 GMP sidecar 或改用 `google-cloud-metric-writer` 直接 push(评估两者)
- [ ] [backend] Grafana 加 GMP 数据源(GKE ClusterIP 出网到 GCP 需 workload identity 或 SA JSON)
- [ ] [Frontend] `Frontend/src/lib/errorReporting.ts` 初始化 Cloud Logging structured error 上报
- [ ] Grafana dashboard JSON:Backend HTTP(P50/P95/P99 + 5xx rate + in_flight)
- [ ] Grafana dashboard JSON:Backend Dependencies(dep_up + ES/PG/BQ latency histogram)
- [ ] Grafana dashboard JSON:Cloud Run 资源(CPU/Mem/Instance count/Request count,读 GCP built-in metrics)

## Sprint 3 [L3] SLO + 错误预算 + 前端 + runbook(~2 天)
- [ ] [backend] SLO 声明:availability 99.5% + latency P95 < 500ms(在 monitoring 里定义 `google_monitoring_slo`)
- [ ] [backend] 错误预算烧尽告警(1h/6h/24h burn rate multi-window)
- [ ] [Frontend] Cloud Run frontend 加 GCP Error Reporting SDK
- [ ] [Frontend] `main.tsx` 挂 `window.onerror` + `unhandledrejection` 上报
- [ ] Runbook `docs/runbook/backend-5xx.md` — 5xx 骤增排查(gcloud logs → 定位 revision → 回滚步骤)
- [ ] Runbook `docs/runbook/batch-stuck.md` — 引用今天 batch 8f5cd676 事故:pipeline_run pending 无 uid → Argo GC → submitter 死循环;含 SQL 修复命令模板
- [ ] Runbook `docs/runbook/db-slow-query.md` — Cloud SQL 慢查询 top-N + 缓解手段

## Dev verification(全量 apply 后)
- [ ] `gcloud monitoring uptime list-configs` 显示 ≥ 5 个 cyber-databrew 条目
- [ ] 手动 stop backend-dev revision → 5min 内 Feishu 群收到 P1 告警
- [ ] Grafana → Backend HTTP dashboard 点开有实时数据(≥ 12h 保留)
- [ ] 前端 `throw new Error('e2e-test-monitoring')` 后 → GCP Error Reporting 30s 内看到该 error
- [ ] `openspec/changes/CYB-monitoring-baseline` 归档(tasks 全勾)

## Context files
- `openspec/changes/CYB-monitoring-baseline/proposal.md`
- `openspec/changes/CYB-monitoring-baseline/design.md`
- `openspec/changes/CYB-monitoring-baseline/specs/observability/spec.md`
- `backend/routes/routes.go`(L113: `/metrics` handler 已存在)
- `backend/internal/metrics/backend.go`(现有指标定义)
- `deploy/k8s/base/`(现有 GKE 部署,Prometheus/Grafana 参考)
- `docs/agents/AI-RULES.md`(deploy 流程)
