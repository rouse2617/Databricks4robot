# CYB-3425 — DataBrew 多集群支持

## Why

`delivery-clust`(独立 GCP project `cyberorigin-delivery`,us-central1)已经建好,需要接入 DataBrew 作为**独立执行集群**跑客户交付流水线,并**保留现有集群 `cyber-clust` 继续跑 dev / video-proc 等业务**。当前 backend 架构假设"单集群":

- `k8s.NewClientset("")` 和 `argo.NewClient(...)` 都是全局单例,从 env vars(`K8S_API_ENDPOINT` / `ARGO_SERVER_URL` / `K8S_AUDIENCE`)读一次
- `ExecutionTarget` 虽然有 `cluster` 字段,但代码里没被消费 —— 所有 target 落在同一集群不同 namespace
- `run_watcher` 单例订阅一个 Argo server 的 workflow event stream
- ElasticQuota 面板只查询一个集群

要接入 delivery-clust 就必须把这些"单集群假设"打破。

### 为什么现在做

- delivery-clust 已经装好,等着 DataBrew 接
- 未来 12 个月预期 1-5 集群(交付驱动,个位数客户)
- 每接一个新客户就再等一次多集群改造,总成本远高于现在一次做透

### 为什么不用 Karmada / OCM

调研了业界方案(见 [decisions.md D5](decisions.md#d5-per-cluster-argo-installation-not-karmada--ocm)):
- **Karmada**:CNCF sandbox,但控制面复杂,踩坑资料对我们的规模是 overkill
- **OCM + argo-workflow-multicluster**:Red Hat 系,Argo 官方支持不明确
- **MultiKueue**:用 Kueue Job API 不是 Argo,方向不对
- **GKE Fleet / Anthos**:licensing 贵,面向流量不是 workflow dispatch

自建 factory(N ≤ 5 时最优)+ schema 为 Karmada-ready 保留(N > 5 时无痛迁移),是最匹配我们规模和成本的路径。

## What Changes

### Backend(medium,~3-5 天)

**Schema(Atlas migration,3 阶段避免 downtime)**:
- 新表 `clusters`:`id uuid PK, name text UNIQUE, display_name text, api_endpoint text, argo_url text, argo_namespace text, audience text, ca_data bytea, is_default bool, status text, created_at, updated_at, deleted_at`
- `execution_targets` 加 `cluster_id uuid` FK,3 阶段:nullable → backfill default → NOT NULL

**Client factory 抽象**:
- `k8s.ClientFactory` 接口:`ForCluster(clusterID string) (kubernetes.Interface, error)` / `DynamicForCluster(...)`
- `argo.ClientFactory` 接口:`ForCluster(clusterID string) (WorkflowClient, error)`
- 实现:内部持有 `clusters` 表 + `map[clusterID]*rest.Config` 缓存,60s TTL 刷新;每 cluster 独立 WIF token source
- **接口设计为 Karmada-ready**:Stage 2 只换实现(单点 hub client),接口不变

**Handler / API**:
- `GET/POST/PUT/DELETE /api/v1/admin/clusters`(admin auth)
- `GET /api/v1/elastic-quotas?cluster=<id>`(不指定则聚合全集群)
- 现有 `/deploy` 等 endpoint 内部按 `target.cluster_id` 选 client,签名不变

**Runtime**:
- `run_watcher` 变成"按 cluster 起 N 个 goroutine",各自独立订阅 argo server 的 workflow event stream,一个 cluster 挂不影响其他
- Circuit breaker per cluster
- 后台 job(outbox 等)不动 —— 都是纯 DB 层,无需多集群

**Frontend(small,~1-2 天)**:
- Registry pools tab 新增子面板 "集群管理"(admin 可见,create/edit/delete)
- Pool 创建/编辑弹窗加 `cluster` 下拉(依赖 `listClusters`)
- Target 表格加"集群"列
- ElasticQuota panel 按 cluster 分组(header 显示 cluster name)

**Infra(external,SRE 支持,~1 天)**:
- **跨项目 IAM**:`cyber-databrew-cloudrun-{dev,prod}@green-valley-442103.iam` → `cyberorigin-delivery` 项目 KSA WIF binding
- **delivery-clust 装 3 件套**:
  - Argo Workflow controller + argo-server(独立 helm)
  - Koordinator(koord-scheduler + koord-manager + koord-descheduler,和 cyber-clust 同版本 + preemptionPolicy: Never)
  - GMP PodMonitoring 采样 koord metrics
- **业务对象**:4 PriorityClass(koord-prod/mid/batch/free)+ 3 ElasticQuota(cyberorigin-delivery-high/mid/low,生产配比)
- **Cloud Monitoring dashboard**:变量加 cluster label filter
- **Feishu 通知**:msg 加 `cluster` tag

## 数据 & 迁移

1. Atlas migration 部署到 dev DB → 现有 dev backend 起 `default` cluster row(从 env vars 初始化,兼容旧配置)
2. 现有 targets 全部 backfill `cluster_id = default_cluster.id`
3. Admin 通过 UI 或 SQL 新建 `delivery-clust` cluster row
4. 新建 `delivery-prod` target 绑 `delivery-clust`
5. Smoke pipeline 部署到 delivery-clust 验收
6. 全部通过后,`cluster_id NOT NULL` 加约束(第 3 阶段 migration)

## 兼容性 & 回滚

- **零回归**:现有 target 全部指向 default cluster,行为不变
- **回滚**:revert PR → clusters 表和 factory 代码消失 → backend 回退到旧的"单集群 env-driven"路径。schema 上 `execution_targets.cluster_id` 可以留(nullable),不做数据 loss。
- **未装 delivery-clust 的环境**:clusters 表只有 default 一条,factory 只维护 1 个 client,和现在无差异

## Out of Scope

- ❌ **Karmada / OCM 层**:延迟到 N > 5 集群才引入([decisions D5](decisions.md#d5-per-cluster-argo-installation-not-karmada--ocm))
- ❌ **transpiler 注入 Koordinator 三要素**:这是 [CYB-3422 P3.1b Task #83/#84](../CYB-3422-koord-quota-panel/tasks.md) 的事,不叠在本 PR
- ❌ **智能调度 / 动态路由**:target 硬绑 cluster,不做"看容量选 cluster"([decisions D1](decisions.md#d1-explicit-target-cluster-binding-not-smart-scheduling))
- ❌ **MCap-preview 多集群**:mcap-preview 服务独立,不涉及
- ❌ **多集群 secret 中央化(Secret Manager)**:现在 secret 还在 K8s namespace 内,先不改;未来若 secret 变多再中央化

## 后续(不在本次)

- **P4.2**:CYB-3422 P3.1b transpiler 接入 Koord(和多集群完全独立,可并行)
- **P4.3**:当 N > 5 或需要"跨集群 failover"时,factory 实现替换为 Karmada hub client(schema 不变)
- **P4.4**:多集群成本归因 —— 当前 [billing-cost skill](../../.claude/skills/billing-cost/) 按 project 拆,delivery-clust 独立 project 会自然分开,不需要额外改动
