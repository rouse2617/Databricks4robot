## ADDED Requirements

### Requirement: Cloud Run backend `/metrics` 经 GMP sidecar 采集
The system SHALL 以 multi-container 形态在 Cloud Run backend-dev 服务上运行官方 `run-gmp-sidecar` collector 边车,scrape 主容器的 `localhost:8080/metrics` 并推送到 Google Managed Service for Prometheus (GMP),使全部 `backend_*` 指标出现在 `prometheus.googleapis.com/backend_*`。

**Priority**: P1

**Rationale**: 本需求把 CYB-3671 R5(“`/metrics` MUST 被 GMP 采集,频率 ≥ 每分钟 1 次”)及其 Sprint 2 未勾任务“Cloud Run backend-dev 挂 GMP sidecar”从目标落成具体机制;R5 父 spec 仍在未归档的 CYB-3671 change 中,归档时二者合并。此前 `backend-dev.sh` 的 `--update-annotations run.googleapis.com/prometheus_scrape=true`(及注释“GCP automatically scrapes /metrics”)是 GKE PodMonitoring 概念,在 Cloud Run 上无效——Cloud Run 静默丢弃该 annotation,不产生任何采集,导致 CYB-3691 只能对单个 gauge 走 Custom Metric API direct-write 兜底,其余 ~40 个 `backend_*` 指标在 GCP 侧全空白。官方支持的 Cloud Run 拉取式采集机制即 multi-container run-gmp-sidecar collector。

- collector zero-config 默认 scrape `localhost:8080/metrics`,周期 ≤ 每分钟 1 次(sidecar 默认 30s),无需 RunMonitoring secret。
- app 容器 MUST 是唯一 ingress 容器(持有 `--port 8080`);collector 无 ingress port。
- 实例 CPU 合计 MUST 为受支持值(1/2/4/8),与单容器现状(4)一致,月成本无增量。
- 采集 MUST 通过部署脚本 gate flag `ENABLE_GMP_SIDECAR`(默认 OFF)控制,仅 dev 部署路径(`deploy-dev.yml`、`local-build-deploy.sh`)开启;共享该脚本的 preview(`deploy/preview/cloudbuild.yaml`,CPU=1)MUST 保持单容器不受影响。
- 失效的 `PROMETHEUS_SCRAPE` 变量与 `run.googleapis.com/prometheus_scrape` annotation 分支 MUST 从 `backend-dev.sh` 移除。

#### Scenario: dev 双容器启动且主容器正常 serve
- **Given** `ENABLE_GMP_SIDECAR=true` 的 dev 部署完成
- **When** 新 revision 就绪并接管 100% 流量
- **Then** `gcloud run services describe` 显示 2 个容器(app + collector)
- **And** `/healthz` 返回 200,实例 CPU 合计 = 4

#### Scenario: backend_* 指标出现在 GMP
- **Given** 双容器 revision 稳定运行 ≥ 2 min
- **When** 在 Metrics Explorer 查询 `prometheus.googleapis.com/backend_dispatcher_stale_items/gauge` 等 `backend_*` 指标
- **Then** 查得到数据点,密度 ≥ 每分钟 1 个

#### Scenario: preview 环境不受影响
- **Given** `deploy/preview/cloudbuild.yaml` 未设 `ENABLE_GMP_SIDECAR`(默认 OFF)、CPU=1
- **When** preview 部署运行 `backend-dev.sh`
- **Then** 以单容器部署,不挂 collector
- **And** 运行时行为与本变更前等价(遗留 annotation 无害,被 Cloud Run 忽略)

#### Scenario: 死 annotation 不再写入
- **Given** 本变更后的 `backend-dev.sh`
- **When** 任意路径调用部署
- **Then** deploy_args 不含 `run.googleapis.com/prometheus_scrape`
- **And** 脚本不再定义 `PROMETHEUS_SCRAPE` 变量
