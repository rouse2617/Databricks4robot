# Spec Delta — argo.WorkflowClient K8s CRD mode

## ADDED

### `argo.crdWorkflowClient` — 第二个 WorkflowClient 实现

实现 13-method `argo.WorkflowClient` interface,用 K8s dynamic + typed client(通过 `k8s.ClientFactory` 拿)直接操作 `argoproj.io/v1alpha1/workflows` CRD,不打 argo-server HTTP endpoint。

**依赖注入:**
- `dynamic.Interface`(K8s dynamic client)— CRUD workflow CRD
- `kubernetes.Interface`(typed client)— 拿 pod logs
- default namespace — 兜底(现在都传 ns)

**每个 method 到 K8s API 的映射:**

| WorkflowClient method | K8s API 操作 |
|---|---|
| `CreateWorkflow(wf, ns)` | `dynamic.Resource(workflowsGVR).Namespace(ns).Create(unstructured, {})` |
| `GetWorkflow(name, ns)` | `dynamic.Resource(...).Namespace(ns).Get(name, {})` → convert to `wfv1.Workflow` |
| `GetWorkflowStatus(name, ns)` | `GetWorkflow` + return `wf.Status.Phase` |
| `DeleteWorkflow(name, ns)` | `dynamic.Resource(...).Namespace(ns).Delete(name, {})` |
| `ListWorkflows(ns, selector)` | `dynamic.Resource(...).Namespace(ns).List({LabelSelector: selector})` |
| `StopWorkflow(name, ns)` | JSON patch `spec.shutdown = "Stop"` |
| `TerminateWorkflow(name, ns)` | JSON patch `spec.shutdown = "Terminate"` |
| `SuspendWorkflow(name, ns)` | JSON patch `spec.suspend = true` |
| `ResumeWorkflow(name, ns)` | JSON patch `spec.suspend = false` |
| `RetryWorkflow(name, ns)` | Get → reset failed nodes + clear workflow phase → Update |
| `ResubmitWorkflow(name, ns)` | Get → clone spec, clear metadata → Create with new generateName |
| `ResubmitWorkflowWithResult(...)` | 同 Resubmit + 返回新 workflow |
| `GetWorkflowLogs(wf, pod, ns, opts)` | typed `CoreV1().Pods(ns).GetLogs(pod, PodLogOptions).DoRaw()` |
| `GetWorkflowLogStream(wf, pod, ns, opts)` | 同上 + `.Stream(ctx)` 返回 io.ReadCloser |

## MODIFIED

### `argo.ClientFactory.ForCluster` — 增 mode 选择

```go
func (f *dbClientFactory) buildClient(c *models.Cluster) (WorkflowClient, error) {
    if strings.TrimSpace(c.ArgoServerURL) != "" {
        // HTTP mode (unchanged, cyber-clust and any future cluster with argo-server URL)
        return NewClientFromConfig(f.configForCluster(c)), nil
    }
    // CRD mode (delivery-clust and any future cluster without argo-server exposure)
    if f.k8sFactory == nil {
        return nil, fmt.Errorf("%w: crd mode requires k8s factory for cluster %q",
            ErrClusterMisconfigured, c.Name)
    }
    dc, err := f.k8sFactory.DynamicForCluster(ctx, c.ID)
    if err != nil { return nil, err }
    typed, err := f.k8sFactory.ForCluster(ctx, c.ID)
    if err != nil { return nil, err }
    return newCRDWorkflowClient(dc, typed, argoNamespace(c)), nil
}
```

Cache semantics unchanged(60s TTL,cluster CRUD 触发 Invalidate)。

### `argo.WithK8sFactory(k8s.ClientFactory) FactoryOption` — 新 option

供 `NewClientFactory` 注入 k8s factory,CRD mode 用它拿 dynamic/typed client。

## What stays UNCHANGED

- `argo.WorkflowClient` interface(13 methods)不变
- `argo.Client` (httpWorkflowClient) 代码零改动
- `cluster-default`(cyber-clust)行的 `argo_server_url` 非空 → 继续 HTTP 路径,byte-identical
- `models.Cluster` schema 不动
- 前端 API `argoServerConfigured` 判断逻辑不动(只要 factory 能出 client,就是 configured)
- run_watcher / Deploy usecase / handlers 一行不改(它们只跟 `argo.WorkflowClient` interface 打交道)

## Compat matrix

| 集群 argo_server_url | k8sFactory | 返回的 WorkflowClient | 行为 |
|---|---|---|---|
| 非空(cyber-clust) | 任意 | httpWorkflowClient | pre-3486d,零变化 |
| 空(delivery-clust) | wired | crdWorkflowClient | K8s CRD 直提,workflow controller 接管 |
| 空 | nil(测试)| ErrClusterMisconfigured | 明确错误,不 nil-deref |

## Verify plan

- Backend `go test ./...` 全绿(4d.1 - 4d.4 每个都单独跑)
- 4d.1 后:delivery-clust 上 kubectl 能查到通过 backend 提交的 workflow
- 4d.2 后:UI 停止 / 挂起 / 恢复 delivery-clust 上跑的 workflow 生效
- 4d.3 后:UI Retry / Resubmit delivery-clust workflow 生效
- 4d.4 后:UI workflow log 页面显示 delivery-clust pod logs
- 4d.5 后(可选):delivery-clust pipeline 状态自动同步(不依赖 exit hook)
- 整体:通过 SDK 端到端 submit → 跑完 → 状态回读 → logs 可见,delivery-clust 完成第一个真实业务 pipeline
