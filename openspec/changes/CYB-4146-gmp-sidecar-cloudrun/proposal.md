# Proposal — CYB-4146

## Why

CYB-3671 (monitoring-baseline) 的 **R5** 要求 Cloud Run backend-dev/prod 的 `/metrics` 端点被 GCP Managed Service for Prometheus (GMP) 采集,频率 ≥ 每分钟 1 次。该需求至今未落地。

原因:`backend-dev.sh` 现有的 `--update-annotations run.googleapis.com/prometheus_scrape=true`(line 567-569)是 **GKE PodMonitoring 的概念,在 Cloud Run 上无效** —— Cloud Run 会静默丢弃该 annotation,不产生任何 GMP 采集。脚本里那句注释 "GCP automatically scrapes /metrics" 是错的。CYB-3691 因此只能对单个 gauge 走 Custom Metric API direct-write 兜底,其余 ~40 个 `backend_*` 指标在 GCP 侧仍然是空白。

## What Changes

### New Capabilities

1. **GMP sidecar collector** — 在 Cloud Run backend-dev 服务里以 multi-container 方式挂官方 `run-gmp-sidecar` collector 边车,scrape 主容器 `localhost:8080/metrics`,推送到 GMP。全部 `backend_*` 指标出现在 `prometheus.googleapis.com/backend_*`。

### Modified Capabilities

- **`deploy/cloudrun/backend-dev.sh`**:单容器 `gcloud run deploy` 改造为双容器(app + collector);删除失效的 `PROMETHEUS_SCRAPE` 变量与 annotation 分支。

## Impact

- **Affected code**: `deploy/cloudrun/backend-dev.sh`(唯一改动文件)
- **New APIs**: 无(纯部署形态变更,不动 Go 代码)
- **Dependencies**: 新增 collector 镜像 `us-docker.pkg.dev/cloud-ops-agents-artifacts/cloud-run-gmp-sidecar/cloud-run-gmp-sidecar:1.2.0`(GCP 一方,无需自建)
- **Runtime SA / IAM**: 无需变更,`roles/monitoring.metricWriter` 已授予
- **Secret**: 不新增(零配置 collector,不需要 RunMonitoring secret)

## Scope

- **In scope**:改造 dev 部署脚本为双容器;CPU 拆分 app=3 + collector=1(合计维持 4,零成本增量);删除死 annotation。
- **Out of scope**:prod 部署脚本(dev 验证后另起 issue);metric 白名单(Option 2,需 RunMonitoring secret,留作 CYB-3989 成本治理杠杆);Go 代码 / `/metrics` 端点本身(已存在,routes.go:115)。

## Success Criteria

- [ ] dev 新 revision 以双容器启动,主容器正常 serve 流量(`/healthz` 200)
- [ ] Metrics Explorer 查得到 `prometheus.googleapis.com/backend_dispatcher_stale_items/gauge` 等 `backend_*` 指标
- [ ] 实例 CPU 合计 = 4(与现状一致),月成本无增量
- [ ] CI `deploy-dev.yml`(no-traffic → migrate → update-traffic)全流程不受影响
