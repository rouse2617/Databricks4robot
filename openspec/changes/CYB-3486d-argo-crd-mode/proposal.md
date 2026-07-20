# CYB-3486 PR 4d — argo client 增 K8s CRD 模式(不再依赖 argo-server HTTP)

## Why

`argo.ClientFactory` 目前只有一种实现:HTTP POST 到 argo-server。
非默认集群(比如新接的 `delivery-clust`)受此束缚:

1. **argo-server 必须公网可达** — 目前 delivery-clust 的 URL 是 `.svc.cluster.local` cluster-internal DNS,Cloud Run backend 打不到。想通得给每个新集群加公网 LB 或 VPC peering,每加一朵云都要开一次口子。
2. **每集群独立 argo bearer token** — 存 Secret Manager,rotate 策略,N 个客户 = N 份 secret。
3. **网络维度上的运维成本随集群数线性增加**。

同时,**Argo Workflows 本身是 K8s CRD**,argo-server HTTP 只是一层薄的 REST wrapper。 已验证(2026-07-15 F smoke):
- `kubectl create -f workflow.yaml` 直接提交到 delivery-clust,argo controller 12s 内接管、跑完、更新 status
- 真 transpiler workflow(tolerations / onExit / arguments / workflow-runner KSA)通过 CRD 提交完全等价于 HTTP 路径
- Backend GSA 已经能通过 K8s API 打到 delivery-clust(WIF binding + RBAC 已就位,cross-project workload identity 通了)

**结论:** K8s API 已经是我们和 delivery-clust 之间的通信通道。让 argo client 也走这条通道,而不是要求每个新集群再开一条公网 argo-server 通道。

## What Changes

### 后端 argo.WorkflowClient 增第二种实现

现在:
```go
// http_client.go — argo-server HTTP 实现
type Client struct { httpClient, serverURL, token ... }
```

之后:
```go
// http_client.go — 保留(cyber-clust 继续用)
type httpWorkflowClient struct { ... }

// crd_client.go — 新增
type crdWorkflowClient struct {
    dynamic       dynamic.Interface   // 通过 k8s.ClientFactory 拿
    typedPods     kubernetes.Interface // 拿 pod logs
    namespace     string               // 默认 ns
    resourceVersionOnListNil bool     // K8s API-specific tweaks
}
```

两者都实现 `argo.WorkflowClient` interface(13 个 method)。

### Factory 按 cluster 行选择实现

`argo.ClientFactory.ForCluster(clusterID)` 内部判断:
- `cluster.argo_server_url != ""` → 返回 `httpWorkflowClient`(HTTP path,cyber-clust)
- `cluster.argo_server_url == ""` → 返回 `crdWorkflowClient`(CRD path,delivery-clust 及未来新集群)

零回归:cyber-clust 的 `argo_server_url = http://10.2.1.211:2746` 非空,继续走 HTTP。

### 分子 PR 拆分(每个 <1 天 + tests + smoke)

**4d.1 CRD skeleton + create/get/delete/list/status(半天)**
- `crdWorkflowClient` 实现 5 个 read/create/delete methods:
  - `CreateWorkflow` → `dynamic.Resource(workflowsGVR).Namespace(ns).Create(obj)`
  - `GetWorkflow` / `GetWorkflowStatus` → `dynamic.Get`
  - `DeleteWorkflow` → `dynamic.Delete`
  - `ListWorkflows` → `dynamic.List(labelSelector)`
- Factory 加 mode 选择逻辑 + tests
- delivery-clust 上跑一个 e2e workflow(用 SDK 或前端 deploy 到 delivery-clust-dev target),验证提交、状态回读、删除都通

**4d.2 lifecycle ops(半天)**
- `StopWorkflow`: patch `spec.shutdown = "Stop"`
- `TerminateWorkflow`: patch `spec.shutdown = "Terminate"`
- `SuspendWorkflow`: patch `spec.suspend = true`
- `ResumeWorkflow`: patch `spec.suspend = false`
- Tests + delivery-clust smoke 各个操作

**4d.3 Retry / Resubmit(半天,较硬)**
- Retry: 读 workflow → 修改 `spec.retryStrategy` / 或 clone with `argoproj.io/retry-attempt` 注解 → create
- Resubmit: 读 workflow spec → 生成新 workflow(带 `spec.arguments` 复制)→ create
- 参考 argo v3 pkg/apiclient/workflow-service.go 源码,复刻边界(node reset / status clear)
- 简化:先只支持"整个 workflow 重跑",高级 partial-retry 后期做

**4d.4 Logs(半天)**
- `GetWorkflowLogs` / `GetWorkflowLogStream` 走 K8s Pod API:
  - 读 workflow.status.nodes → 找 pod 名
  - 用 `typedPods.CoreV1().Pods(ns).GetLogs(pod, opts).Stream(ctx)`
  - 多 pod 时按 startedAt 排序 interleave(和 argo-server 语义对齐)
- Tests(mock pod client)+ delivery-clust smoke

**4d.5 run_watcher per-cluster goroutine(半天,可选)**
- 现在 `pipeline_run_watcher` 是单 goroutine 用 singleton wfClient
- 改成:enumerate active runs 的 distinct clusterIDs → per-cluster goroutine
- 集群增删 reconcile(30s ticker)
- 这个也可以后置,不是 F blocker(默认路径 status 靠 exit webhook 推送,run_watcher 只是 backstop)

**总工时估算:** 4d.1~4d.4 各半天 + 4d.5 半天 + 集成/smoke/文档 = **~2-3 天**。

## What NOT in this PR series

- ❌ Argo v4 迁移(v3 CRD schema 稳定)
- ❌ 拆掉 cyber-clust 的 argo-server(还有 UI 用户;而且 http_client 保留)
- ❌ 前端 cluster picker on Deploy 弹窗(已有 target 下拉,选 target 就选了 cluster;单独 issue)
- ❌ 拆 argo v3.5 → v3.6 客户端库(依赖锁定)

## Deploy prereqs(已经全部就绪 · 2026-07-15)

- ✅ 跨项目 WIF IAM binding
- ✅ delivery-clust cyber-databrew-backend-argo KSA + User RBAC(workflow / pods / nodes / elasticquotas)
- ✅ delivery-clust workflow-runner KSA(为 pod SA)+ argo-workflow role(workflowtaskresults)
- ✅ delivery-clust cluster 表行 endpoint / audience / caData

## 决策 checkpoint

请 review `proposal.md` + `design.md` + `tasks.md`。 确认后我进 4d.1 code。
