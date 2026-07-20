# Proposal — CYB-monitoring-baseline

## Why

cyber-databrew 已经有 Prometheus/Grafana/ES-exporter 部署在 GKE 里跑了 34 天,后端 `/metrics` 也吐出了丰富的 backend_http_*/backend_dependency_up/backend_elasticsearch_* 指标 —— **但整套系统的 SRE-facing 监控是残缺的**:

1. **零 uptime check** 覆盖 cyber-databrew(GCP project `green-valley-442103` 只有 1 个 uptime,还是别的项目 `cyberorigin-os` 的)。后端 Cloud Run 5xx 或完全 down,Google 不会通知任何人。
2. **零 alert policy** 定义在 cyber-databrew 的组件上;`BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL` 只用于 batch 完成通知,没有异常告警。
3. **Cloud Run 后端 `/metrics` 未被 Prometheus 抓** —— Prometheus 在 GKE ClusterIP 内网,Cloud Run 在公网,scrape 未打通,所以现有 Prometheus 面板上看不到 backend 指标。
4. **无 SLO 定义**,故障判定靠肉眼刷日志。
5. **前端**(Cloudflare Worker + Cloud Run 混合)**监控为 0**。

结果:如果 backend-dev 挂 30 分钟,发现路径是"用户报"而不是"监控叫"。这不是 SRE-ready 状态。

## What Changes

### New Capabilities
- **L1 存活告警**:cyber-databrew 相关的 5 个公网入口(backend-dev、backend-prod、frontend-dev、frontend-prod、grace-service)全部覆盖 GCP Uptime Check + Alert Policy → 现有 Feishu webhook。
- **L2 指标可视**:backend `/metrics` 通过 GCP Managed Service for Prometheus(GMP)方式被 GKE 现有 Prometheus 联邦,或者切换到 GMP 完全托管,把 backend HTTP P95 / 5xx / dependency_up 三条核心 SLI 上 Grafana。
- **L3 SLO 契约**:公开定义 availability 99.5%、latency P95 < 500ms 目标,错误预算烧尽触发 P1 告警。
- **前端错误捕获**:Cloudflare Worker + Cloud Run frontend 前端 JS 错误上报(Sentry 或 GCP Error Reporting)。

### Modified Capabilities
- 现有 `feishu-monitoring-relay` Cloud Run 从"batch 完成通知"扩展成"通用告警渠道",支持不同严重级(P1/P2/P3)+ runbook 链接。

## Impact
- **Affected code**:
  - `deploy/terraform/monitoring/`(新目录,uptime + alert policy)
  - `deploy/k8s/base/prometheus-scrape-config.yaml` 或迁移到 GMP
  - `Frontend/src/main.tsx` 加错误上报 SDK 初始化
  - `docs/runbook/`(新)存故障处理手册
- **新配额**:
  - GCP Uptime Check:5 个 × 1/min = 免费额度内(前 3M/月)
  - GMP 或 alert policy 无额外硬费用,评估 <$10/月
- **依赖**:可选 Sentry account(免费 tier 够小团队用)或用 GCP Error Reporting(项目内已有)
- **Schema**:无 SQL 迁移

## Scope
- **In scope**:
  - Uptime checks + alert policies + Feishu 通知路由(L1)
  - Cloud Run backend `/metrics` 打通 Prometheus 或 GMP(L2)
  - 3 张 Grafana dashboard:Backend HTTP、Backend Dependencies、Cloud Run 资源(L2)
  - SLI/SLO 定义 + 错误预算告警(L3)
  - 前端 error 上报接入(L3)
  - runbook 骨架
- **Out of scope**:
  - APM/Trace(OpenTelemetry 全链路)—— 单独 change 处理
  - 日志聚合/结构化(现有 gcloud logging 够用)
  - 成本告警(GCP Budget 独立机制)
  - 安全监控(SIEM)

## Success Criteria
- [ ] `gcloud monitoring uptime list-configs` 列出 ≥5 个 cyber-databrew 相关 uptime check
- [ ] `gcloud monitoring policies list --filter="displayName~cyber-databrew"` 返回 ≥5 个 policy
- [ ] Backend Cloud Run 主动关闭一个 revision 后 5 分钟内 Feishu 收到告警
- [ ] Grafana dashboard "Backend HTTP" 显示 P50/P95/P99 latency + 5xx rate 曲线(读 GMP 或 Prometheus)
- [ ] 前端 throw 一个测试 error 后 5 秒内 Sentry/Error Reporting 收到
- [ ] `docs/runbook/README.md` 至少 3 个场景手册(backend 5xx / batch stuck / DB 慢查询)

## Goals (SLO,骨架待 L3 定稿)
- **Backend availability**: monthly uptime ≥ 99.5%(错误预算 ~3.6h/月)
- **Backend latency**: HTTP P95 < 500ms(排除 batch 大查询)
- **Frontend error rate**: JS uncaught < 0.1% 会话
- **Alert MTTR**: P1 告警到人 < 5min(Feishu 推送延迟)
