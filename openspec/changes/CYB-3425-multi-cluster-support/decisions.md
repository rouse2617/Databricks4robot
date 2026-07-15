# CYB-3425 Decisions

## D1. Explicit target-cluster binding, not smart scheduling

### 选择
`ExecutionTarget` 加 `cluster_id` FK,用户显式选 target 就选中了 cluster;backend 不做"看容量选 cluster"这类动态调度。

### 备选
- **标签路由**:pipeline 打 label `customer: kimi` 自动选 cluster —— 需要写规则引擎
- **就近调度**:根据 asset 数据 region 选 —— 需要 asset metadata 补 region 字段
- **智能调度**:看 cluster 容量剩余选 —— 需要跨集群实时容量指标

### 拒绝理由
- 我们规模 1-5 集群,显式绑定完全够用;规则引擎属于过度设计
- 智能调度和 Koordinator 的定位重叠(quota 已经是集群内的动态分配),集群间再加一层"智能选 cluster"违反 [feedback_prefer_minimal_infra](../../.claude/projects/-Users-rick-cyber-databrew-dev/memory/feedback_prefer_minimal_infra.md)
- 显式绑定 debug 直接:pipeline 跑不起来 → 看 target.cluster_id → 检查该 cluster 状态,一路可查

## D2. Cluster 配置在 DB 表 + CRUD API,不是 env var

### 选择
新增 `clusters` 表 + `/api/v1/admin/clusters` CRUD API,admin 可通过 UI 增删改集群。

### 备选
- **env var `CLUSTERS_JSON`**:backend 启动读 JSON 数组,重启才生效
- **静态 config file**:mount 一个 configmap
- **不做多集群**:接受"单集群 + 多 namespace 分池"作为 workaround

### 拒绝理由
- env var:每次加集群都要动 Cloud Run env,还要 deploy 一次;prod dev 独立配置容易漂移
- config file:同 env var,+ configmap 变更后 pod 不会热重载
- 单集群 workaround:blast radius 大,delivery 客户 pipeline 崩会牵连 dev / video-proc

### DB 表方案的优点
- Admin UI 里加集群 = 一次 API 调用,不需要 deploy
- prod / dev 数据库各自独立,配置漂移天然分开
- 支持 `deleted_at` 软删除,历史 pipeline 的 target.cluster_id 还能查回来
- 是 Karmada-ready 的:未来 Karmada 阶段,表还在,只是 factory 实现变了

## D3. 认证走 Workload Identity Federation,不是 static bearer token

### 选择
Backend 到 GKE cluster 的认证复用**现有 dev 的 WIF metadata-token 路径**(`K8S_USE_METADATA_TOKEN=true`),只是每 cluster 一份 `audience + api_endpoint + ca_data`。跨项目认证 = `cyber-databrew-cloudrun-{dev,prod}@green-valley-442103.iam` 绑到 `cyberorigin-delivery` 项目的 KSA。

### 备选
- **Static bearer token**:创建 SA + Token,base64 存 Secret Manager,`K8S_<CLUSTER>_BEARER_TOKEN` env
- **Cross-cluster kubeconfig 文件**:把两集群的 kubeconfig 打包成一个 file mount
- **impersonation via `iam.serviceAccounts.getAccessToken`**:backend SA 显式 impersonate 目标 KSA

### 拒绝理由
- Bearer token:半年后必有"忘轮换 → prod 认证失败"事故([memory_dev_deploy_env_traffic](../../.claude/projects/-Users-rick-cyber-databrew-dev/memory/project_dev_deploy_env_traffic.md) 类事故);Secret Manager 也不解决轮换问题
- Kubeconfig 文件:审计难,权限颗粒度差
- 显式 impersonation:多一次 API 调用,latency 增加;WIF 已经是 impersonation 的正规封装,没必要再手撸

### WIF 优点
- 已经在用,添加新 cluster 只是**多一次 IAM binding** + 多一个 audience
- Token 每次请求自动刷新,零轮换负担
- audience 是绑定 KSA 的天然作用域,权限隔离清晰

## D4. Failure isolation per cluster,不是全或无

### 选择
每 cluster 独立 K8s + Argo client,独立 circuit breaker。一个 cluster 挂了,其他 cluster 的 deploy / status refresh 依然正常。backend 整体健康(`/readyz`)不依赖任何单个 cluster reachable。

### 备选
- **All-or-none**:任一 cluster unreachable → backend 返回 503。简单但爆炸
- **优先级降级**:default cluster 挂了整体不可用,delivery cluster 挂了只报"新客户不可用"

### 拒绝理由
- 多集群本来就是为了隔离故障域,任一挂就全崩不如不做
- 优先级降级需要在 backend 硬编码"哪个 cluster 更重要",不符合"cluster 平等"的抽象

## D5. Per-cluster Argo installation,不是 Karmada / OCM

### 选择
每 cluster 各装一套 Argo Workflow controller + argo-server + Koordinator + GMP。backend 维护 N 个 argo client。

### 备选
- **Karmada**:hub-and-spoke,backend 只提交一次 Workflow CR,Karmada 复制到 member cluster 本地 Argo 执行
- **OCM + argo-workflow-multicluster**(Red Hat 系)
- **KubeStellar**(新项目)

### 拒绝理由
- Karmada 控制面复杂,踩坑成本 2-3 周,对 1-5 集群规模是 overkill
- OCM 概念多、hub 集群单点、Argo 官方支持不明确
- KubeStellar 项目新、生态窄、生产事故风险高

### 何时会翻案
Stage 2 触发条件:**cluster 数 > 5** 或 **需要 failover 到备用 cluster**。届时:
- factory 接口保留,实现替换为 Karmada hub client(schema 不变)
- 每 member cluster 依然装 Koord(Karmada 只做调度决策,不管 pod QoS)
- 预期迁移工作量 ~2-3 周,可以按"新集群先接 Karmada、老集群逐步迁"渐进式做

## D6. 现有 target 归 default cluster,不迁移

### 选择
Atlas migration 建 `default` cluster row(从当前 backend 的 env vars 派生 api_endpoint / argo_url / audience / ca_data);现有所有 ExecutionTarget backfill `cluster_id = default_cluster.id`;新客户交付 pipeline 建**新 target** `delivery-prod` 绑 `delivery-clust`。

### 备选
- **迁移现有 target 到 delivery-clust**:比如 `video-proc-prod` 迁过去
- **保留 target 名字但换 cluster**:改 `video-proc-prod` 的 cluster_id 到 delivery-clust

### 拒绝理由
- 迁移现有 target 意味着**已在跑的 pipeline 会中断或 pod 分布突变**,blast radius 大
- 迁移是不可回滚的操作(pod 起在了新 cluster,老 cluster 的 quota 已经释放),不适合和多集群改造同时做
- 建新 target(`delivery-prod`)让业务侧可以显式选,不影响老逻辑

### 如需迁移,应作为独立 PR
先本次多集群 GA,后续如需把 `video-proc-prod` 迁到 delivery-clust,单独 issue 单独规划(涉及 pipeline 侧配合停机窗口)。

## D7. Factory 接口先行,Karmada-ready 抽象

### 选择
`k8s.ClientFactory` / `argo.ClientFactory` 定义成 interface,Stage 1 实现是 `dbClientFactory`(从 clusters 表加载),Stage 2 未来实现是 `karmadaClientFactory`(hub 单点 + cluster metadata)。所有 handler / usecase 消费方**只依赖 interface**,不知道底下是自建 factory 还是 Karmada。

### 好处
- Stage 1 → Stage 2 迁移只改 `wire.go` 一处 binding,不动业务代码
- 单测更好写(mock factory 就够,不用 mock K8s client)
- schema(`clusters` 表)在两 stage 都用,兼容平滑

### 代价
接口设计需要考虑得比"直接叫 client factory function"周到些,多花 0.5 天设计时间。
