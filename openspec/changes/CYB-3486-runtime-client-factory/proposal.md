# CYB-3486 PR 4 — runtime client factory(多集群按 target 路由)

## Why

[CYB-3486](https://linear.app/cyberorigin/issue/CYB-3486) PR 1-3 已经在 dev 上落地了 `clusters` 表 + admin CRUD + 前端下拉 + delivery-clust cluster row。但 **backend runtime 仍然是全局单例 K8s / Argo client**,拿 target.cluster_id 路由到不同 cluster 的能力还没有。

现在 dev 上:
- `k8s.NewClientset("")` / `k8s.NewDynamicClient("")` / `argo.NewClient(...)` 都是 startup 时读 env vars 建的**全局单例**
- 所有 handler / usecase / run_watcher 都直接消费单例
- delivery-clust cluster row 存了 endpoint / audience / CA / argo URL,但**没人读**

必须把 client 建构从"env 单例"抽成"按 clusterID 拉取的 factory",才能让 target.cluster_id=delivery-clust 的 pipeline 真正落到 delivery-clust 上。

## What Changes

### 后端接口(Karmada-ready 抽象)

```go
// backend/internal/k8s/factory.go
type ClientFactory interface {
    ForCluster(ctx, clusterID) (kubernetes.Interface, error)
    DynamicForCluster(ctx, clusterID) (dynamic.Interface, error)
    ForTarget(ctx, target) (kubernetes.Interface, error)  // 便捷方法
    Invalidate(clusterID)  // cluster CRUD 后主动失效
}

// backend/internal/argo/factory.go
type ClientFactory interface {
    ForCluster(ctx, clusterID) (WorkflowClient, error)
    ForTarget(ctx, target) (WorkflowClient, error)
    Invalidate(clusterID)
}
```

Stage 1 实现 `dbClientFactory`(从 clusters 表读 + 60s TTL 缓存);未来 N > 5 换 `karmadaClientFactory` 时不动接口。

### 消费点迁移(逐个替换)

| 消费方 | 现状 | 改成 |
|-------|------|------|
| `elastic_quota.go`(CYB-3422)| `k8s.NewDynamicClient("")` 单例 | `factory.DynamicForCluster(clusterID)` |
| `resource_quota.go` | `k8s.NewClientset("")` | `factory.ForCluster(clusterID)` |
| Pod diagnostics / exec / logs | 单例 pod client | `factory.ForTarget(target)` |
| Deploy usecase(argo submit)| 单例 argo client | `argoFactory.ForTarget(target)` |
| `run_watcher` | 单 goroutine 订阅一个 argo | **N 个 goroutine per active cluster** |

### 故障隔离

每 cluster 独立 client + 独立 lastError。某 cluster API 挂,只该 cluster 快速失败,其他 cluster deploy 不受影响。Circuit breaker per cluster(可选,PR 4e)。

### Cluster 配置热更新

Admin 通过 `/api/v1/admin/clusters` PUT/POST 后,handler 调 `factory.Invalidate(clusterID)`,下次 ForCluster 走 fresh config。不用等 60s TTL。

## What NOT in this PR

- ❌ Karmada / OCM 换实现(等 N > 5)
- ❌ ExecutionTarget schema 加 elastic_quota_name(那是 CYB-3422 P3.1b)
- ❌ Cross-project IAM binding(需要 SRE 手动执行 + backend deploy 环境变量;单独 PR)

## 分子 PR 拆分(降 blast radius)

- **4a**:接口 + 骨架实现 + 单测,**不接入任何消费方**(骨架 land 到 dev 无副作用)
- **4b**:替换 elastic_quota / resource_quota handler
- **4c**:替换 Deploy usecase + argo submit
- **4d**:替换 run_watcher(per-cluster goroutine)
- **4e**(可选):circuit breaker + Feishu cluster tag

本 iteration 目标 = PR 4a。
