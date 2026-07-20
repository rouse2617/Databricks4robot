# Specs — Observability Baseline (CYB-monitoring-baseline)

## ADDED Requirement: 公网入口存活告警

**Priority**: P0

**Rationale**: 现状 cyber-databrew 无任何 uptime check + alert policy,后端 Cloud Run 服务 down 时无告警路径。属 SRE-critical 盲区。

The system SHALL:

- **R1**: 对以下 5 个公网 Cloud Run 服务分别配置 GCP Uptime Check,轮询周期 ≤ 1 min,超时 ≤ 10s,check regions ≥ 3 大洲:
  1. cyber-databrew-backend-dev — `/healthz` (无 auth)
  2. cyber-databrew-backend-prod — `/healthz`
  3. cyber-databrew-frontend-dev — `/`
  4. cyber-databrew-frontend-prod — `/`
  5. grace-service — `/healthz`(依赖服务)
- **R2**: 每个 uptime check MUST 绑定一个 P1 严重级 alert policy,触发条件:任意 2 个连续采样窗口 down。
- **R3**: 所有 P1 alert MUST 通过 `feishu-monitoring-relay` Cloud Run 服务发送到 Feishu 群,附带:服务名、URL、错误码/超时、GCP console 快捷链接。
- **R4**: 告警冷却(auto-close)MUST 在服务恢复后 5 min 内触发,发送恢复通知。

### Scenario: backend-dev down 触发告警

**Given** cyber-databrew-backend-dev 处于健康状态,uptime check 通过
**When** 手动 `gcloud run services update-traffic --to-latest=0`(或删除 revision)导致 5 min 内 3 次以上 uptime 失败
**Then** ≤ 5 min 内 Feishu 告警群收到 P1 通知,消息含 "backend-dev DOWN" + 具体 URL + GCP 控制台链接

### Scenario: 服务恢复自动关告警

**Given** 上一步告警已触发
**When** 恢复 traffic 到 healthy revision,uptime check 连续 3 次成功
**Then** ≤ 5 min 内 Feishu 收到恢复通知,alert incident status = CLOSED

### Scenario: 误报保护

**Given** 单一 region 短暂网络抖动导致 1 次 uptime 失败
**When** 其他 region 全部成功
**Then** SHALL NOT 触发告警(需连续 2 次多 region 失败)

## ADDED Requirement: 后端 SLI 指标可见

**Priority**: P1

**Rationale**: `/metrics` 端点已有丰富指标但未接入 Prometheus,Grafana 面板空白,故障时无可视化。

The system SHALL:

- **R5**: Cloud Run backend-dev/prod 的 `/metrics` 端点 MUST 被 GCP Managed Service for Prometheus (GMP) 采集,采集频率 ≥ 每分钟 1 次。
- **R6**: 至少提供 3 张 Grafana dashboard:Backend HTTP、Backend Dependencies、Cloud Run Resources。
- **R7**: Backend HTTP dashboard MUST 显示按 route 聚合的 P50/P95/P99 latency + 5xx rate + in-flight requests,时间粒度 ≤ 1 min。

### Scenario: 面板显示实时数据

**Given** GMP 采集配置生效 ≥ 2 min
**When** 用户打开 Grafana Backend HTTP dashboard
**Then** SHALL 显示最近 15 分钟内的 P95 曲线,数据点密度 ≥ 每分钟 1 个

### Scenario: 5xx 骤增可视

**Given** backend-dev 某 endpoint 突然 100% 返回 500
**When** 5xx 事件持续 2 min
**Then** Grafana Backend HTTP dashboard 的 5xx rate 曲线 SHALL 在下一次 scrape (≤ 1 min) 后显示 spike

## ADDED Requirement: 日志级异常信号告警

**Priority**: P1

**Rationale**: 2026-07-17 batch 8f5cd676 事故里,submitter 陷入 "already-exists but run still has no uid" 死循环,后端日志高频吐 warning 但无告警,直到用户手动打开页面才发现。

The system SHALL:

- **R8**: 定义 log-based alert policy 匹配以下 pattern,触发条件:5 min 内 ≥ 20 次:
  - "already-exists but run still has no uid, leaving pending"
  - "submitter: submit incomplete (no uid yet), leaving pending for retry"
  - "stranded" 或 "stuck"
- **R9**: log-based alert MUST 严重级 P2,推到 Feishu 但不 page on-call。

### Scenario: batch stranding 早期识别

**Given** 某 batch 由于 Argo GC + pre-#471 bug 陷入 submitter 死循环
**When** 5 min 内后端日志出现 ≥ 20 次 "already-exists but run still has no uid"
**Then** ≤ 5 min 内 Feishu 收到 P2 告警,含 jobID + 相关 pipeline_run 数量提示

## ADDED Requirement: 前端错误可采集

**Priority**: P2

**Rationale**: 前端 JS 错误目前完全静默,用户遇到白屏无信号。

The system SHALL:

- **R10**: 前端 (Cloud Run frontend-dev/prod) MUST 全局注册 `window.onerror` + `unhandledrejection` handler,将错误上报到 GCP Error Reporting(通过前端 Cloud Run 的 stdout structured log)。
- **R11**: 上报 payload MUST 包含:message、stack、url、userAgent、session-id(不含 PII)。

### Scenario: 未捕获 error 上报

**Given** frontend 加载完成
**When** 页面代码 throw 一个 uncaught error
**Then** GCP Error Reporting 界面 SHALL 在 30 s 内出现该 error,并按 stack trace 自动聚类

## ADDED Requirement: SLO 契约

**Priority**: P2

**Rationale**: 无 SLO 则无客观故障判定标准,团队对"多严重才算 incident"没共识。

The system SHALL:

- **R12**: 定义并落地以下 SLO(通过 `google_monitoring_slo` 资源):
  - Backend availability:滚动 30 天 uptime ≥ 99.5%
  - Backend latency:滚动 30 天 HTTP P95(排除 `/api/v1/backfill/*` batch 端点)< 500 ms
  - Frontend JS uncaught error rate:滚动 7 天 < 0.1% of pageviews
- **R13**: 错误预算多窗口烧尽率告警(1h 14.4x + 6h 6x):烧尽速率超阈值 → P1 alert。

### Scenario: 错误预算烧尽预警

**Given** 30 天窗口内 backend availability = 99.5%(错误预算刚好用完)
**When** 最近 1 小时内 downtime 达 14.4× 正常 burn rate(即 6 分钟连续 down 就烧尽本月剩余预算)
**Then** 触发 P1 alert(多窗口 burn rate 策略)
