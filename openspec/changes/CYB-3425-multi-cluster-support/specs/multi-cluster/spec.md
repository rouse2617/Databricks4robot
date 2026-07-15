# 多集群支持 · Spec Delta

## ADDED Requirements

### Requirement: Admin 可以通过 UI 管理集群清单

Admin 用户可以在 `/registry` → 「集群管理」子面板 CRUD 集群配置。

#### Scenario: Admin 创建新集群

- **GIVEN** 我是 admin 用户(`ADMIN_EMAILS` 包含我)
- **WHEN** 我打开 `/registry` → 「集群管理」子面板 → 点击「新建集群」→ 填入 {name: "delivery-clust", display_name: "客户交付集群", api_endpoint: "https://34.44.27.160", argo_url: "https://argo.delivery.internal", argo_namespace: "cyber-databrew-dev", audience: "…", ca_data: "…"} → 提交
- **THEN** POST `/api/v1/admin/clusters` 返回 201 + cluster.id
- **AND** 「集群管理」表格新增一行
- **AND** backend 60s 内(TTL 缓存刷新)可以对这个 cluster 拉起 client(下次调用 `factory.ForCluster(id)` 命中新 cluster)

#### Scenario: 非 admin 用户尝试创建集群

- **GIVEN** 我不是 admin
- **WHEN** 我直接 POST `/api/v1/admin/clusters` 或打开 admin 页
- **THEN** 返回 403 forbidden
- **AND** 「集群管理」子面板对非 admin 完全不可见

#### Scenario: 软删除集群

- **GIVEN** cluster `delivery-clust` 存在,但**没有任何** target 绑定它
- **WHEN** admin 点删除 → 确认
- **THEN** DELETE `/api/v1/admin/clusters/:id` 返回 204,`clusters.deleted_at` 被设置
- **AND** 后续 `listClusters()` 不返回它;绑定过它的历史 pipeline 依然可以查回 cluster 名字(通过 `?include_deleted=true`)

#### Scenario: 删除仍被 target 引用的集群

- **GIVEN** cluster `delivery-clust` 存在,且 target `delivery-prod` 的 `cluster_id` 指向它
- **WHEN** admin 尝试删除
- **THEN** 返回 409 conflict + message "1 target still references this cluster"

### Requirement: ExecutionTarget 必须绑定到一个集群

Target 有必填的 `cluster_id`,决定了它的 pipeline 落哪个 K8s cluster。

#### Scenario: 老 target 自动归到 default cluster

- **GIVEN** Atlas migration 已经跑过(default cluster row 已 seed,所有老 target `cluster_id` 已 backfill)
- **WHEN** 查询 `/api/v1/execution-targets`
- **THEN** 每个 target 都带 `cluster_id`,老 target 都是 `default cluster.id`
- **AND** 部署到老 target 的 pipeline 依然落到原来的 cluster(`cyber-clust`),行为不变

#### Scenario: 新建 target 绑到 delivery-clust

- **GIVEN** `delivery-clust` cluster row 存在
- **WHEN** admin 建新 target `delivery-prod` + cluster 下拉选 `delivery-clust` + namespace `cyber-databrew-dev`
- **THEN** target 创建成功
- **AND** 部署 pipeline 到该 target 时,Argo workflow 提交到 delivery-clust 的 argo-server,pod 起在 delivery-clust 上

#### Scenario: 绑到不存在(或已删除)的 cluster

- **WHEN** admin 尝试建 target 时 cluster_id 传了不存在的 UUID
- **THEN** 返回 400 + message "cluster not found"

### Requirement: Backend 对每个 cluster 独立通信

Backend 用 factory 为每个 cluster 维护独立的 K8s client、Argo client、run_watcher goroutine。

#### Scenario: 一个 cluster API 不可达,其他 cluster 不受影响

- **GIVEN** `delivery-clust` 的 API server 突然 503(网络断 / kube-apiserver 挂)
- **WHEN** 有并发 deploy 请求:一批目标 `delivery-prod`(delivery-clust),一批目标 `video-proc-prod`(cyber-clust)
- **THEN** delivery-prod 的部署报 502 "cluster delivery-clust unavailable"
- **AND** video-proc-prod 的部署**全部成功**,不受影响
- **AND** backend `/readyz` 返回 200(不因 delivery-clust 挂而 unhealthy)

#### Scenario: Cluster 认证凭证失效

- **GIVEN** `cyberorigin-delivery` 项目的 WIF binding 被 admin 撤销
- **WHEN** backend 试图对 delivery-clust 调 K8s API
- **THEN** 客户端刷新 token 失败,返回 401
- **AND** backend 把这次调用记为 circuit breaker 一次 failure,连续 N 次后 open circuit
- **AND** 面向该 cluster 的所有请求快速失败(不再打真实 K8s API),15s 后 half-open 重试

### Requirement: ElasticQuota API 支持按 cluster 过滤/聚合

`GET /api/v1/elastic-quotas` 默认聚合所有 cluster;`?cluster=<id>` 只返回该 cluster 的 quota。

#### Scenario: 聚合模式(默认)

- **WHEN** curl `/api/v1/elastic-quotas`
- **THEN** 返回所有 cluster 上装了 Koordinator 的 ElasticQuota,每条 item 带 `cluster_id` 和 `cluster_name` 字段
- **AND** 前端可以按 cluster 分组显示

#### Scenario: Cluster filter

- **WHEN** curl `/api/v1/elastic-quotas?cluster=<delivery-clust-id>`
- **THEN** 只返回 delivery-clust 上的 quota
- **AND** cluster_id / cluster_name 字段依然带上,方便前端复用一个渲染组件

#### Scenario: 某 cluster 没装 Koordinator

- **GIVEN** cyber-clust 装了 Koordinator,delivery-clust 没装
- **WHEN** 聚合查询 `/api/v1/elastic-quotas`
- **THEN** 只返回 cyber-clust 的 quota;delivery-clust 那部分自动跳过(CRD not found,不当作错误)
- **AND** 响应 200 不报错

## MODIFIED Requirements

### Requirement: ExecutionTarget 表结构

**BEFORE**:`id, name, namespace, cluster (string legacy 字段, 未使用), argo_server_configured, ...`

**AFTER**:同上 + `cluster_id uuid NOT NULL FK → clusters(id)`。老字段 `cluster` 保留但 deprecated,新代码不读它;下 iteration 删除。

### Requirement: run_watcher 生命周期

**BEFORE**:backend 启动时起 1 个 run_watcher goroutine,订阅单个 argo server 的 workflow event stream。

**AFTER**:backend 启动时从 `clusters` 表读取所有 active cluster,每 cluster 起 1 个 run_watcher goroutine。cluster CRUD 后 60s 内(TTL)自动 spawn/kill 对应 watcher。

## RELATED

- 前置:CYB-3422 P3.1a(ElasticQuota 只读面板,已 done)
- 并行 / 无依赖:CYB-3422 P3.1b(transpiler 注入 Koord 三要素)
- 后续:Stage 2 Karmada 迁移(当 N > 5 或需要 failover)—— schema 和 factory 接口都保留兼容
