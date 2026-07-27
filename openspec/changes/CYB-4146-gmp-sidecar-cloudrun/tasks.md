# Tasks — CYB-4146 GMP sidecar for Cloud Run backend

## Phase 1 [code] backend-dev.sh 双容器改造(gate OFF 默认)
- [ ] [deploy] 新增 gate 变量 `ENABLE_GMP_SIDECAR`(默认 `false`)+ `APP_CPU/APP_MEMORY/COLLECTOR_CPU/COLLECTOR_MEMORY/COLLECTOR_IMAGE` 默认值
- [ ] [deploy] 删除 `PROMETHEUS_SCRAPE` 变量(line 56)+ 错误注释(line 53-55)
- [ ] [deploy] 删除 `--update-annotations run.googleapis.com/prometheus_scrape` 分支(line 567-569)
- [ ] [deploy] 抽出 service-level flag 组(min/max-instances、timeout、cpu-boost、cpu-throttling、no-traffic、allow-unauthenticated、vpc)为共享片段,两分支复用
- [ ] [deploy] ON 分支:`--container app`(image/port 8080/cpu/memory/env-vars-file/secrets)+ `--container collector`(image/cpu/memory/--depends-on app,无 port)
- [ ] [deploy] OFF 分支:保留今日单容器 `deploy_args` 逐字不变

## Phase 2 [code] caller 开 gate(仅 dev 路径)
- [ ] [ci] `.github/workflows/deploy-dev.yml` backend-dev.sh 调用处加 `ENABLE_GMP_SIDECAR=true`
- [ ] [deploy] `deploy/cloudrun/local-build-deploy.sh` backend-dev.sh 调用处加 `ENABLE_GMP_SIDECAR=true`
- [ ] [deploy] `deploy/preview/cloudbuild.yaml` **不动**(默认 OFF,单容器)

## Phase 3 [verify offline] 无沙箱 googleapis egress,离线验证
- [ ] `bash -n deploy/cloudrun/backend-dev.sh`(语法)
- [ ] arg-array dry-run:`ENABLE_GMP_SIDECAR=true` 下 echo `deploy_args`,肉眼核对容器分段 / port 只在 app / secret 落在 app 段 / CPU 合计=4
- [ ] arg-array dry-run:`ENABLE_GMP_SIDECAR=false`(preview 模拟,CPU=1)确认与旧 diff 等价
- [ ] `openspec validate CYB-4146-gmp-sidecar-cloudrun --strict`

## Phase 4 [PR] 提交(PR 不触发部署)
- [ ] commit(subject 全小写),push `feat/CYB-4146-gmp-sidecar-cloudrun`
- [ ] 开 PR → `dev`;PR body:说明双容器 + gate + CPU 假设 + 合并前需手工 `!` 部署验证
- [ ] deploy-before-commit:PR 本身不部署;**合并到 dev 才触发** deploy-dev.yml

## Phase 5 [live verify] 合并前用户手工 `!` 部署(沙箱无法 gcloud)
- [ ] 用户跑给定 `! gcloud run deploy … --container app … --container collector …`(或 `ENABLE_GMP_SIDECAR=true … bash backend-dev.sh`)到 dev,`--no-traffic` 保护
- [ ] 若 CPU 3+1 被拒 → `APP_CPU=2 COLLECTOR_CPU=2` 重试(env 覆盖,不改代码)
- [ ] `describe` 见 2 容器 + `/healthz` 200 → 切流量
- [ ] Metrics Explorer 见 `prometheus.googleapis.com/backend_*` ≥ 2 min 数据 → 用户确认 → 合并 PR

## Context files
- `deploy/cloudrun/backend-dev.sh`(唯一实体改动)
- `.github/workflows/deploy-dev.yml` / `deploy/cloudrun/local-build-deploy.sh`(开 gate)
- `deploy/preview/cloudbuild.yaml`(确认不受影响)
- `backend/routes/routes.go:115`(`/metrics` 已存在)
- 父:`openspec/changes/CYB-3671-monitoring-baseline`(R5 + Sprint 2 sidecar 任务)
