# Design — argo.WorkflowClient K8s CRD mode

## Constraint recap

- Preserve full backward compat for cyber-clust — HTTP path 一行代码不动
- 保留 `argo.WorkflowClient` interface(13 method),不引入新 wrapping 层
- 复用现有 `k8s.ClientFactory`(PR 4a)拿 dynamic + typed client,不新增依赖
- 复用 `models.Cluster` 那一行,不加新列(至少 4d.1-4d.4 不加)
- 每一子 PR 单独可 revert,零跨 PR 依赖

## 架构决定

### D1. 模式选择放在 `argo.ClientFactory.configForCluster`,不放在业务代码

**选:** factory 内部 switch:
```go
func (f *dbClientFactory) buildClient(c *models.Cluster) (WorkflowClient, error) {
    if strings.TrimSpace(c.ArgoServerURL) != "" {
        cfg := f.configForCluster(c)  // 现有 http config
        return NewClientFromConfig(cfg), nil
    }
    // CRD mode
    dc, err := f.k8sFactory.DynamicForCluster(ctx, c.ID)
    if err != nil { return nil, err }
    typed, err := f.k8sFactory.ForCluster(ctx, c.ID)
    if err != nil { return nil, err }
    return newCRDWorkflowClient(dc, typed, argoNamespace(c)), nil
}
```

**拒绝:** 让 Deploy usecase 自己判断"这个 target 用 http 还是 crd"。 理由:
- 消费方(deploy / run_watcher / handler)不该知道传输机制
- factory 是 Karmada-ready 抽象点,选实现是 factory 的职责

### D2. `argo_server_url == ""` = CRD 模式(隐式)

**选:** 用现有 `clusters.argo_server_url` 列的 empty/non-empty 分岔,不加新列。

**理由:**
- cyber-clust 迁移零改动(URL 已非空)
- delivery-clust 迁移只需一次 `UPDATE clusters SET argo_server_url = '' WHERE id = <...>`
- 语义上 URL 就是"http 提交端点";没端点自然走别的路径
- 未来加新集群时如果客户环境有 argo-server 且公网可达,依然可以填 URL 走 HTTP

**拒绝的 alternatives:**
- Alt A. 加 `argo_transport` 列 (enum: http | crd) — 冗余,URL 已能表达
- Alt B. 从 client 侧探测(先 ping HTTP,不通 fallback CRD) — 30s 探测 + 首次请求延迟不可接受

### D3. Retry / Resubmit 复刻 argo-server 语义,不自研

**选:** 参考 argo v3.5 源码 `pkg/apiclient/workflow/workflow-server.go`:
- `Retry`: 读 workflow → 找 node phase=Failed → reset node status → clear workflow phase → update
- `Resubmit`: 读 workflow spec → 深拷贝 → clear metadata (name, uid, resourceVersion) → set generateName → create

**为什么复刻不重新设计:**
- 保证 argo UI 看到 F 提交的 retry/resubmit 行为完全一致(UI 用 argo-server 查同一份 CRD)
- Argo 的 retry 语义(哪些 status 保留、哪些 clear)是有历史的,自研会踩之前踩过的坑

**简化范围(4d.3 iteration 1):**
- Retry: 只支持 `--restart-successful=false --node-field-selector phase=Failed`(即失败节点从头跑)
- Resubmit: 生成全新 workflow(不做 `--memoized`)
- 高级 flag(retry-successful / memoized / parameter override)延后到需要时

### D4. Logs 用 typed K8s API,不重新实现 argo-server logs

**选:**
- `GetWorkflowLogs`: enumerate workflow.status.nodes → 找 podName / template → CoreV1().Pods(ns).GetLogs(pod, PodLogOptions).DoRaw → text
- `GetWorkflowLogStream`: 同上但 `.Stream(ctx)` 返回 io.ReadCloser
- 多 pod 时:按 node.startedAt 排序,顺序拼接(argo-server 也是这样)

**Container 选择:** transpiler 产的 workflow 主容器叫 `main`。用 `PodLogOptions.Container: "main"`。 wait sidecar / init container 不含用户日志,不 stream。

**Tail / limit:** argo `WorkflowLogOptions.TailLines / LimitBytes` 直接映射到 K8s PodLogOptions.TailLines / LimitBytes。

### D5. `getWorkflowWithUID` / retry loop 保留 client-abstract

`getWorkflowWithUID` 已经在 PR 4c.1 里改成接受 `client argo.WorkflowClient`。 CRD client 的 `GetWorkflow` 也返回 `*wfv1.Workflow`(same shape),所以上层零感知。

### D6. Argo namespace 从 cluster 行拿,不用全局 env

`clusters.argo_namespace` 列已有(PR 1)。 CRD client 需要一个默认 ns 用于 List(不带 ns 时)。 从这一列读。

### D7. Nil-safety: `k8sFactory == nil` 不 crash

如果部署没配 k8sFactory(单元测试 fixture,或纯前端 dev 环境),`argo.ClientFactory.ForCluster` 应该:
- HTTP 路径:老逻辑不变(用 env config)
- CRD 路径:返回 ErrClusterMisconfigured(而不是 nil-pointer dereference)

## Non-goals

- **不改 `models.Cluster` schema** — 这一系列 PR 全走现有列
- **不重构 argo `http_client.go`** — 保留代码 mush,只加不删
- **不追求 argo-server API 语义 1:1 完美** — 追求 "cyber-databrew backend 需要的 method 完全等价",不追求"完全可以替代 argo-server 给所有 CLI 使用"

## 迁移路径

**阶段 1(此 PR 系列):**
- Land 4d.1 → 4d.4
- 更新 `delivery-clust` cluster 行:`UPDATE clusters SET argo_server_url = '' WHERE name = 'delivery-clust'`
- 验证:通过前端 deploy 一个 pipeline 到 `delivery-clust-dev` target,workflow 完整跑通

**阶段 2(未来接第一个非 GCP 客户):**
- 加 `clusters.auth_type` + `clusters.auth_secret_ref` 列
- factory 认证层再抽象一次(GKE metadata → bearer token → mTLS)
- 走 CRD path 无差

## 风险 & 回退

| 风险 | 缓解 | 回退动作 |
|---|---|---|
| CRD path 有边界 bug(retry 某个 flag 行为不对)| 只有走 CRD path 的 cluster 受影响,cyber-clust 完全隔离 | 恢复 `argo_server_url` 非空,回退到 HTTP path。无 code revert 需要 |
| Argo v3 CRD schema 未来变更 | 我们用 typed `wfv1.Workflow` 反序列化 dynamic 结果,GKE 会随 argo 一起升级 | 锁定 argo v3.5 helm chart(已锁) |
| Retry/Resubmit 复刻不完整,某些 flag 缺失 | 4d.3 只覆盖高频用法,详见 D3 简化范围 | 用户可以 kubectl retry / argo CLI 手工做,或走前端 Deploy 重新起 |
| Logs stream 多 pod 排序抖动 | 用 pod.startedAt sort;和 argo-server 同一策略 | 单 pod workflow(默认)无差异 |

## 实际实现与计划的差异 (post-hoc addendum)

### D3 补充 —— 不引入 `argo-workflows/v3/workflow/util`

原本设想复用官方 `wfutil.FormulateRetryWorkflow` / `FormulateResubmitWorkflow` 拿"官方语义"。
真上手时发现:该 file 传递依赖 HDFS / Kerberos / OpenTelemetry / cron —— 拉进来 backend
镜像 +几十 MB 依赖树,违背 [[feedback_prefer_minimal_infra]] 的最小依赖原则。 D3 里
预留的"简化范围"正好允许 hand-roll,所以走了 hand-roll:

- Retry: workflow 必须是 Failed/Error/Succeeded;失败节点 phase→Pending + 删对应 pod;
  workflow status.phase→Running,清 finishedAt/message;Update。
- Resubmit: deep-copy spec + non-argo labels/annotations;新 generateName(源名 strip
  argo hash suffix);清 metadata;Create。

不含:`--restart-successful`、`--memoized`、partial retry via node-field selectors,
参数覆盖。 需要这些的用户走 argo CLI / kubectl。

### D8(新增)—— 破解 `argo → k8s → pipeline → argo` import cycle

计划里 D1 说 factory 内部依赖 `k8s.ClientFactory` 拿 dynamic/typed client。 实操时
发现 `internal/k8s/runtime_config.go` import 了 `usecase/pipeline`(为了拿
RuntimeConfigProjection 类型),而 `usecase/pipeline/usecase.go` import `argo`,
成环。

解决:`internal/argo/factory.go` 里声明局部 `K8sFactory` interface 只包含用到的两个
方法(`ForCluster` + `DynamicForCluster`)。`k8s.ClientFactory` 结构性满足这个接口,
`infra.go` 里 `argo.WithK8sFactory(k8sFactory)` 直接传就行。 argo package 不再
import `internal/k8s`,cycle 解开。

### 4d.5 拆成两个 commit

原本 4d.5 一个 commit 做完 per-cluster goroutine + reconcile 循环。 实际发现:
"每 run 走对客户端"(correctness,delivery-clust runs 不再被误发到 cyber-clust)和
"per-cluster goroutine 并行"(scaling,一 cluster 卡不会拖住别的 cluster)是两个
独立价值。 拆:

- **4d.5** = correctness half. `resolveArgoClientForRun(ctx, run)` + 7 个 caller
  重构。 单独可 revert,单独可 merge,已经解掉了核心 bug。
- **4d.5.b** = scaling half. `SyncActiveRunEvents` 按 cluster 分组 fan-out 一 goroutine
  per cluster。 correctness 已通,这一步是性能优化。

原始 "reconcile 循环(新集群出现 → spawn goroutine,idle > N min → drain)" 没做 ——
现在集群数很少(2 个),goroutine 每次 SyncActiveRunEvents 都 fresh 起,自然对齐
新集群加减。 长期主动的 goroutine 池等真正接客户扩到 10+ cluster 再做。

### 4d 之外顺路做的 (branch scope 扩展)

用户 review 后一次性把这些也塞进了 `feat/cyb-3486d-remaining`:

- **auth.1** — `clusters.auth_type` + `auth_secret_ref` schema + factory dispatch。
  只加 `gke_wif` 一个 case,其他 case 返 `ErrClusterMisconfigured`,给未来接 ACK/EKS
  留 seam。
- **pool.1** — `execution_targets.elastic_quota_name`;transpiler 注入
  `quota.scheduling.koordinator.sh/name` label。 pool 概念的细粒度控制(EQ 层)。
- **pool.2** — `execution_targets.priority_class_name`;transpiler 设
  `wf.Spec.PodPriorityClassName`。 池内调度顺序。
- **ux.1** — PoolManager 集群列 muted;EQ 面板收成"池"单列。
- **ux.2** — Target modal 加 EQ + PriorityClass 两个 Input。
