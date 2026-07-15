# CYB-3425 Tasks

## Phase A · 新集群基础设施(SRE / infra 层,~1 天)

- [ ] A1. 跨项目 IAM:在 `cyberorigin-delivery` 建 KSA `cyber-databrew` + RBAC(namespace-scoped),给 `cyber-databrew-cloudrun-{dev,prod}@green-valley-442103.iam` 绑定 `roles/iam.workloadIdentityUser`
- [ ] A2. delivery-clust 装 Argo Workflow(controller + argo-server + minio 视需要)
- [ ] A3. delivery-clust 装 Koordinator 精简档 helm(koord-scheduler + koord-manager + koord-descheduler)
- [ ] A4. delivery-clust 建 4 个 PriorityClass(koord-prod/mid/batch/free,`preemptionPolicy: Never`,`app.kubernetes.io/managed-by: Koordinator-Custom`)
- [ ] A5. delivery-clust 建 3 个 ElasticQuota(cyberorigin-delivery-high/mid/low,生产配比 min 4/2/1 max 24/14/10)
- [ ] A6. 部署 GMP PodMonitoring 采样 koord scheduler metrics
- [ ] A7. 从 Cloud Run backend SA smoke test:`kubectl get elasticquota` 能列出 3 个
- [ ] A8. 提取 delivery-clust 的 `api_endpoint / audience / ca_data`,写入 gcloud secret manager 或直接手写到 seed migration

## Phase B · Backend 实现(~3-5 天)

### B1. Schema
- [ ] B1.1 Atlas migration `<yyyymmdd>_create_clusters_table.sql`
- [ ] B1.2 Atlas migration `<yyyymmdd>_add_execution_targets_cluster_id_nullable.sql`
- [ ] B1.3 Atlas migration `<yyyymmdd>_seed_default_cluster_row.sql`(从现有 env vars 派生 default cluster;prod / dev 各一次)
- [ ] B1.4 Atlas migration `<yyyymmdd>_backfill_execution_targets_cluster_id.sql`
- [ ] B1.5 Atlas migration `<yyyymmdd>_set_execution_targets_cluster_id_not_null.sql`(第 3 阶段,等所有 target 都 backfill 完再上)

### B2. Repo / Model
- [ ] B2.1 `backend/internal/models/cluster.go` — `Cluster` struct + JSON tags
- [ ] B2.2 `backend/internal/repository/cluster_repository.go` — 接口
- [ ] B2.3 `backend/internal/postgres/cluster_repo.go` — 实现 + 单测(mock DB)

### B3. Client Factory
- [ ] B3.1 `backend/internal/k8s/factory.go` — `ClientFactory` 接口 + `dbClientFactory` 实现(从 clusters 表加载,60s TTL 缓存,每 cluster 独立 metadata token source)
- [ ] B3.2 `backend/internal/argo/factory.go` — `ClientFactory` 接口 + 实现
- [ ] B3.3 factory 加 `ForCluster(clusterID string)` 返回 client + `ForTarget(target *ExecutionTarget)` 便捷方法
- [ ] B3.4 单测:mock cluster repo,factory 缓存命中 / miss / TTL 过期 / cluster 不存在报错

### B4. 替换所有 client 消费点
- [ ] B4.1 `k8s.NewClientset("")` 全 repo grep → 改用 factory(handler / usecase / worker 里的调用点)
- [ ] B4.2 `k8s.NewDynamicClient("")` 同上 —— **ElasticQuota handler(CYB-3422 P3.1a)也要改**
- [ ] B4.3 `argo.NewClient(...)` 单例改工厂
- [ ] B4.4 所有旧的 `NewClientset` 保留 wrapper `NewClientsetForDefault()` 兼容(下 iter 迁移完删)

### B5. Handler / API
- [ ] B5.1 `backend/internal/handlers/admin/cluster_handler.go` — GET/POST/PUT/DELETE
- [ ] B5.2 `routes/routes.go` — `/api/v1/admin/clusters`,adminAuth 中间件
- [ ] B5.3 ElasticQuota handler 加 `?cluster=<id>` filter,不指定则聚合全 cluster(returns cluster 信息在每条 item)
- [ ] B5.4 单测覆盖 4 个 CRUD path + 权限拒绝(non-admin 401)

### B6. Deploy / Runtime
- [ ] B6.1 pipeline usecase 里所有 argo submit / k8s exec 用 `factory.ForTarget(target)` 拿 client
- [ ] B6.2 `run_watcher` 从"单 goroutine"改成"per-cluster goroutine",从 clusters 表读活跃 cluster 列表启动
- [ ] B6.3 Circuit breaker per cluster(现有 CB middleware 加 cluster 维度)
- [ ] B6.4 单测:一个 cluster API 503 时,另一个 cluster deploy 仍成功

### B7. Feishu / 监控
- [ ] B7.1 Feishu batch 通知 msg 加 `cluster` 字段
- [ ] B7.2 Cloud Monitoring PodMonitoring label 加 `cluster` 维度

## Phase C · Frontend(~1-2 天)

- [ ] C1. `Frontend/src/api/pipelineApi.ts` 加 `Cluster` type + `listClusters()` / `createCluster()` / `updateCluster()` / `deleteCluster()`
- [ ] C2. Registry pools tab 新增子 section「集群管理」(和 ExecutionTarget、ElasticQuota 并列),admin 可编辑
- [ ] C3. PoolManager Create/Edit modal 加 `cluster` 下拉(依赖 listClusters,预填 default)
- [ ] C4. Target 表格新增"集群"列(显示 display_name)
- [ ] C5. ElasticQuota panel 按 cluster 分组渲染(group header 显示 cluster name);空列表隐藏
- [ ] C6. 前端单测:cluster CRUD 弹窗 / target 绑定 cluster / EQ 分组

## Phase D · 数据 & 灰度(~0.5 天)

- [ ] D1. 上 dev:migrations 生效 + default cluster row 自动 seed
- [ ] D2. 回归测试:现有 target 部署 pipeline 仍走 cyber-clust,行为不变
- [ ] D3. 通过 admin UI 建 delivery-clust cluster row
- [ ] D4. 通过 UI 新建 target `delivery-prod` 绑 delivery-clust
- [ ] D5. 部署一条 smoke pipeline 到 delivery-clust,pod 落到新集群
- [ ] D6. UI Registry pools tab 里看到 delivery-clust 的 3 个 ElasticQuota

## Phase E · Tier L + PR + 回归

- [ ] E1. `go fmt && go vet && go test ./...` 全绿
- [ ] E2. `npm run build` 通过
- [ ] E3. PR to dev + auto-merge + deploy-dev
- [ ] E4. deploy-dev 完成后拉 backend log,确认没引入新的 spam
- [ ] E5. Chrome MCP 前端验收:cluster CRUD / target 绑 cluster / EQ 分组显示
- [ ] E6. Circuit breaker 验收:临时把 delivery-clust api_endpoint 改错,default cluster 依然正常

## Phase F · 生产上线(单独 PR,后续)

- [ ] F1. Prod IAM 跨项目 binding(和 dev 分开,权限最小化)
- [ ] F2. 打 `v0.4.0` tag,触发 deploy-prod.yml
- [ ] F3. 逐步在 prod UI 加 delivery-clust(要等 prod DB migration 跑完)
