# Task 🅲: Infra 改造 + 清理 k8s 包

## 目标
使用 `/Users/rick/cyber-databrew/backend/internal/argo/` 包替换 `backend/internal/k8s/` 包，删除 `k8s` 包，清理 go.mod。

run `cd backend && go build ./...` after each change to verify.

## 具体改动

### 1. `cmd/server/infra.go`
- 去掉 `k8sClient *k8s.Client`
- 去掉 `k8s.NewClient` 调用（初始化 K8s clientset）
- 改为创建 `argo.Client`（从 `ARGO_SERVER_URL` + `ARGO_AUTH_TOKEN` env）
- 保存 `workflowClient argo.WorkflowClient` 给 handler 使用

### 2. `cmd/server/core.go`
- `setupCore`: 
  - 用 `inf.workflowClient` 初始化 pipeline usecase（需要 `*argo.Client` 实现了 `WorkflowClient`）
  - 不再需要 `if kc := inf.k8sClient; kc != nil` 条件判断
  - pipeline usecase 现在始终可用（不需要 K8s）
  - `MetricsClient` 不再初始化（`GetResourceUsage` 可以暂时返回空）
  - `backfillUC` 仍然需要 `pipelineUC.Usecase`，这个不受影响

### 3. `cmd/server/main.go`
- `infra` 结构体去掉 `k8sClient *k8s.Client`
- 引用改为 `argo.WorkflowClient` 接口

### 4. 删除 `backend/internal/k8s/` 整个包
- 删除 `client.go`（不再需要）
- 删除 `workflow_client.go`（ArgoClient 不再需要，接口已迁移到 argo 包）
- 删除 `metrics.go`（MetricsClient，暂时不替代）

### 5. 更新所有 `k8s.WorkflowClient` 引用 → `argo.WorkflowClient`
- 搜索 `k8s.WorkflowClient` 并改为 `argo.WorkflowClient`
- 更新 import path

### 6. 更新 `backend/internal/handlers/pipeline/handler.go`
- 去掉 `unavailable()` 方法（现在 handler 始终可用）
- 所有 method 去掉 `if h.unavailable(c)` guard
- `New()` 构造函数简化（不再需要 unavailableMessage）

### 7. 更新 `backend/internal/handlers/workflow/handler.go`
- 同样去掉 nil/unavailable guard

### 8. 清理 go.mod
- 运行 `go mod tidy` 去掉不再需要的依赖

## 验证
```bash
cd backend && go build ./...
cd backend && go vet ./...
cd backend && go test ./routes/... ./internal/handlers/pipeline/... ./internal/handlers/workflow/... ./internal/k8s/... 2>&1
```

## 参考文件
- `backend/cmd/server/infra.go`
- `backend/cmd/server/core.go`
- `backend/cmd/server/main.go`
- `backend/internal/k8s/client.go`
- `backend/internal/k8s/workflow_client.go`
- `backend/internal/k8s/metrics.go`
- `backend/internal/argo/client.go` — 新的 WorkflowClient 接口
- `backend/internal/argo/auth.go` — 配置加载
- `backend/internal/handlers/pipeline/handler.go`
- `backend/internal/handlers/workflow/handler.go`
