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
