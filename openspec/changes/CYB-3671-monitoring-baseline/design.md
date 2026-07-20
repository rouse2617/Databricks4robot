# Design — CYB-monitoring-baseline

## Architecture Context
- **既有基础设施**(免动):
  - GKE `cyber-databrew-dev` namespace 已有 Prometheus(:9090)、Grafana(:3000)、ES exporter(:9114)
  - Backend `/metrics` 端点:promhttp Handler 已挂在 `/metrics`,吐 backend_dependency_up、backend_http_*、backend_elasticsearch_* 等
  - `BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL` env + `feishu-monitoring-relay` Cloud Run 服务已就位
  - Cloud Run 服务:backend-dev/prod、frontend-dev/prod、grace-service 都有公网 URL
- **约束**:
  - Prometheus 是 GKE ClusterIP 内网,Cloud Run 后端在公网,scrape 不通(NAT egress 可解决但费带宽)
  - 团队规模小,方案要**低运维成本**
  - dev 和 prod 共用 project `green-valley-442103`
- **Non-Goals**:
  - APM/Trace 全链路(留到 L4+)
  - K8s pod 级监控(GKE 已有 GCP 自带 metrics)

## Goals
1. **可发现**:所有公网入口的 down/latency 5min 内到 Feishu
2. **可衡量**:Backend HTTP P50/P95/P99 + 5xx rate 在 Grafana 有面板
3. **可追踪**:错误预算烧尽自动 P1 告警
4. **低运维**:不引入新 SaaS(除可选 Sentry 免费 tier)

## Architecture

```
┌────────────────────── Cloud Run (public) ──────────────────────┐
│  backend-dev/prod  frontend-dev/prod  grace-service            │
│     ├─ /metrics ─────────┐                                     │
│     └─ /healthz ─────────┼──┐                                  │
└──────────────────────────┼──┼──────────────────────────────────┘
                           │  │
        ┌──────────────────┴──┴───────────────────┐
        │  ⓐ GCP Uptime Check (5 个)              │
        │  ⓑ GCP Managed Prometheus (GMP)         │
        └──┬────────────────────────────┬─────────┘
           │                            │
           ↓ metrics                    ↓ alerts
   ┌──────────────────┐         ┌──────────────────┐
   │  Grafana         │◄────────│ Alert Policies   │
   │  (GKE ClusterIP) │         │  (GCP Monitoring)│
   │  data sources:   │         └────────┬─────────┘
   │    - GMP         │                  │
   │    - GKE Prom    │                  ↓
   │  dashboards:     │         ┌──────────────────┐
   │    - Backend HTTP│         │ feishu-monitor-  │
   │    - Deps        │         │ ing-relay        │
   │    - Cloud Run   │         │ (Cloud Run)      │
   └──────────────────┘         └────────┬─────────┘
                                         │
                                         ↓
                                    ┌─────────┐
                                    │ Feishu  │
                                    └─────────┘
```

## Approach Options + Rationale

### 抓 Cloud Run `/metrics` 三选一

| 方案 | 说明 | 优点 | 缺点 | 决定 |
|---|---|---|---|---|
| A) GMP(GCP Managed Prometheus) | Cloud Run 用 sidecar 或写入模式,GMP 直接从 endpoint 抓 | 全托管、免运维、和 GCP 生态原生集成 | 需要开 API + 少量配置;账单额外(通常 <$5/月) | ✅ **推荐** |
| B) 现有 GKE Prom 加 static_configs 走 NAT | 让 GKE Prom 通过 Cloud NAT 出网抓公网 URL | 沿用现有 Prom | NAT 出网费 + 公网抓延迟高 + Prom 单点 | 备选 |
| C) Pushgateway | Backend 主动 push metrics | 简单 | 不适合长期(Pushgateway 反模式) | ✗ 拒绝 |

**选 A**:GMP 官方就是给 Cloud Run 用的,后续加新服务只加一行 scrape config 即可,零维护。

### 前端 error 上报三选一

| 方案 | 说明 | 优点 | 缺点 | 决定 |
|---|---|---|---|---|
| a) Sentry(免费 tier) | 5k events/month 免费 | UI 好、trace/breadcrumbs 完善 | 数据出海,团队小可能吃不完额度 | 备选 |
| b) GCP Error Reporting | 同 project 内,`console.error` 自动聚类 | 无成本、内部化、和 uptime 同 UI | UI 一般、无 breadcrumbs | ✅ **推荐** |
| c) 自建 | 前端 fetch → 后端 endpoint → GCP Log | 完全可控 | 造轮子、缺去重 | ✗ 拒绝 |

**选 b**:数据不出 GCP,零成本,和 alert policy 复用同一套告警链路。

### Alert 触发条件(L1 baseline)

| SLI | 阈值 | 严重级 | 目标 |
|---|---|---|---|
| Uptime check 连续 2 次失败 | 60s window | P1 | Backend/Frontend down 检测 |
| Cloud Run 5xx rate | > 5% for 5min | P1 | 应用异常 |
| Cloud Run request latency P95 | > 1s for 10min | P2 | 性能退化 |
| `backend_dependency_up{dependency=X}` = 0 | 5min | P1 | ES/PG/BQ 依赖挂 |
| Log-based:出现 "stranded" 或 "already-exists but run still has no uid" | any | P2 | 复现今天 batch 8f5cd676 事故的信号 |

## Affected Modules
- `deploy/terraform/monitoring/`(**新目录**)存 uptime_check.tf + alert_policy.tf + notification_channel.tf
- `deploy/cloudrun/backend-dev.sh` 加 `--monitoring` 参数(如需 GMP sidecar)
- `Frontend/src/main.tsx` 或新 `Frontend/src/lib/errorReporting.ts` 加 `window.addEventListener('error'/'unhandledrejection', ...)` 上报 Cloud Logging
- `docs/runbook/README.md`(**新**)
- `docs/runbook/backend-5xx.md`
- `docs/runbook/batch-stuck.md`(结合今天 batch 8f5cd676 事故:Argo GC + submitter 死循环 → 检查方法)
- `docs/runbook/db-slow-query.md`

## Rollback
- Uptime + alert policy 是纯 terraform 声明,`terraform destroy` 即可撤
- 前端错误上报是可选启用的 SDK 初始化,注释即禁
- GMP 是可选启用的 API,禁用 API 即停(数据保留 6 周)

## Migration Order (Rollout)
1. **Sprint 1**(L1,~1 天):Uptime + Alert + Feishu 路由 —— 立即消除"backend down 无告警"最大盲区
2. **Sprint 2**(L2,~3 天):GMP 接入 + 3 张 Grafana dashboard —— 有可视化
3. **Sprint 3**(L3,~2 天):SLO + 错误预算 alert + 前端 error reporting + runbook —— SRE-ready
