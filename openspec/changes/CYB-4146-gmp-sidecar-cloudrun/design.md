# Design — CYB-4146 GMP sidecar for Cloud Run backend

## Architecture Context

- **父需求**:CYB-3671 R5 + Sprint 2 未勾任务 “Cloud Run backend-dev 挂 GMP sidecar 或改用 metric-writer 直接 push(评估两者)”。本变更选定并落地 sidecar。
- **既有事实**:
  - `/metrics`(promhttp)已挂在 `backend/routes/routes.go:115`,吐 ~40 个 `backend_*` 指标。
  - `backend-dev.sh` 现为单容器 `gcloud run deploy`,`--no-cpu-throttling` 已开(sidecar scrape loop 需要),CPU=4 / MEMORY=4Gi。
  - 该脚本被 3 个 caller 共享:`deploy-dev.yml`(dev CI)、`local-build-deploy.sh`(本地→dev)、`deploy/preview/cloudbuild.yaml`(preview,CPU=1/512Mi)。
  - Runtime SA `cyber-databrew-dev@green-valley-442103` 已有 `roles/monitoring.metricWriter`,且脚本**不传** `--service-account`(redeploy 时 gcloud 从既有服务保留)。
- **Non-Goals**:prod 脚本(dev 验证后另起 issue);metric 白名单;动 Go / `/metrics` 本身。

## 决策

### D1 — 采集机制:sidecar vs direct-write

| 方案 | 说明 | 优点 | 缺点 | 决定 |
|---|---|---|---|---|
| A) run-gmp-sidecar | 官方 collector 边车 scrape `localhost:8080/metrics` → GMP | 一次接入全部 `backend_*`;零 Go 改动;GCP 一方镜像免自建 | 部署形态变双容器 | ✅ **选定** |
| B) Custom Metric API direct-write | Go 里逐指标 push(CYB-3691 对单 gauge 已做) | 无部署形态变更 | 需为 ~40 指标逐个写代码 + 维护;易漏 | ✗ 仅作兜底 |

**选 A**:一次性覆盖全部指标,零 Go 改动,契合 CYB-3671 “低运维” 原则。B 留作已存在的单 gauge 兜底,不扩大。

### D2 — gate flag `ENABLE_GMP_SIDECAR`,默认 OFF

脚本共享,preview 用 CPU=1 —— 若无条件改双容器,会把 preview 也变双容器并触碰其 CPU 预算。故用 gate:
- 默认 **OFF** → 单容器路径与本变更前**逐字节一致**(preview 零影响,回滚只需不设 flag)。
- 仅在 `deploy-dev.yml` 与 `local-build-deploy.sh` 显式设 `ENABLE_GMP_SIDECAR=true`。
- `local-build-deploy.sh` 必须一起开:否则本地 pre-commit 部署会静默把 sidecar 抹掉(单容器覆盖双容器 revision)。

### D3 — CPU / Memory 拆分

实例总量维持 **CPU=4 / MEM=4Gi**(月成本零增量)。默认拆分 app=2+2、mem=3584Mi+512Mi:

| 容器 | CPU | MEM | 依据 |
|---|---|---|---|
| app | `APP_CPU=2` | `APP_MEMORY=3584Mi` | 保留 line 27-31 注释的多核意图的大头;总量维持 4 时这是唯一合法的 app 份额 |
| collector | `COLLECTOR_CPU=2` | `COLLECTOR_MEMORY=512Mi` | sidecar 只需 ~0.1 vCPU,但总量不能取 3,故 collector 取 2 才能凑到合法总量 4 |

**规则(已由首次 dev 部署证实,更正原假设)**:gcloud 对 CPU 的校验是**逐容器**、不是取和 —— 每个容器的 `--cpu` 自身必须是 `{<=1, 1, 2, 4, 6, 8}` 之一(`3` 被拒:`Must be equal to one of [.08-1], 1.0, 2.0, 4.0, 6.0, 8.0`)。因此原 `3+1` 的 app=3 非法。`2+2` 是唯一"每容器合法 **且** 实例总量落在受支持值 4"的零成本拆分。校验发生在创建任何 revision **之前**(fail-fast,不产生半坏状态):首次合并 3+1 触发的 deploy-dev 即在此步红掉,dev 仍服务旧单容器 revision。四项 CPU/MEM 仍全部 env 可覆盖 —— 若 app 要拿回更多算力,`APP_CPU=4 COLLECTOR_CPU=2`(总 6,+50% CPU 成本)一行覆盖,不改已提交代码。

### D4 — 零配置 collector,不新增 secret

run-gmp-sidecar 默认即 scrape `localhost:8080/metrics`,**无需 RunMonitoring secret**。契合安全约束(“不新建/改 K8s Secret”)。白名单(RunMonitoring secret)属 Option 2,留给 CYB-3989 成本治理做杠杆,本变更 out of scope。

### D5 — 删死 annotation

`PROMETHEUS_SCRAPE` 变量(line 56)+ `--update-annotations run.googleapis.com/prometheus_scrape`(line 567-569)在 Cloud Run 上无效,连同错误注释 “GCP automatically scrapes /metrics” 一并删除。**范围**:仅停止**写入**该 annotation;既有 dev 服务上遗留的旧 annotation 无害(Cloud Run 忽略),不额外发 `--remove-annotations`,以免误碰系统 annotation,保持 diff 最小。

## 实现形态(imperative gcloud, 576.0.0)

flag 分层(576 `run deploy --help` 已确认):**service-level**(首个 `--container` 之前)= `--min/max-instances --timeout --cpu-boost --no-cpu-throttling --vpc-connector --vpc-egress --no-traffic --allow-unauthenticated`;**per-container**(某 `--container` 之后)= `--image --port --cpu --memory --env-vars-file --set-secrets --remove-secrets --depends-on`。

```
# ENABLE_GMP_SIDECAR=true 分支
run deploy SVC --quiet --project --region --platform managed \
  <service-level flags…> \
  --container app       --image $IMAGE --port 8080 --cpu $APP_CPU --memory $APP_MEMORY \
                        --env-vars-file $F [--set-secrets …] \
  --container collector --image $COLLECTOR_IMAGE --cpu $COLLECTOR_CPU --memory $COLLECTOR_MEMORY \
                        --depends-on app
```

- `--port 8080` 只在 app → app 为 ingress 容器;collector 无 port。
- secret 映射从 service-level 移到 `--container app` 之后(否则绑到 collector)。
- `--depends-on app`:collector 在 app 之后启动,scrape 目标先就绪。
- OFF 分支:保留今日单容器 `deploy_args`,一字不改。

## Success Criteria

- [ ] dev 双容器 revision 起来,`/healthz` 200,CPU 合计=4
- [ ] Metrics Explorer 查得 `prometheus.googleapis.com/backend_*`
- [ ] preview 仍单容器(flag 默认 OFF)
- [ ] `deploy-dev.yml`(no-traffic→migrate→update-traffic)全流程不受影响
- [ ] 月成本无增量
