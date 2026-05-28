# TASK: 评估 Argo HTTP Client 可行性

## 背景

当前 pipeline/workflow handler 依赖 `k8s.Client`（K8s CRD clientset），Cloud Run 上不可用，返回 503。

`k8s/internal/workflow_client.go` 定义了 `WorkflowClient` interface：
- CreateWorkflow / GetWorkflowStatus / DeleteWorkflow / ListWorkflows / GetWorkflow / StopWorkflow / GetWorkflowLogs

当前实现 `ArgoClient` 走 K8s CRD API。计划新增 `ArgoHTTPClient` 走 **Argo API Server** 的 HTTP REST API。

## 评估要点

请阅读代码并回答以下问题：

1. **可行性**：Argo API Server 的 REST API 是否能覆盖 `WorkflowClient` 全部 7 个方法？有没有个别方法 Argo API 不支持的？
2. **依赖消除**：`ArgoHTTPClient` 是否能完全消除对 `k8s.io/client-go` 和 `argoproj/argo-workflows/v3/pkg/client/clientset` 的依赖？
3. **GetWorkflowLogs**：当前实现通过 K8s API 查 pod label 再取 log stream。Argo API Server 是否提供 logs 端点？还是需要用其他方式？
4. **认证方式**：Argo API Server 支持什么认证（token / basic auth / mTLS）？
5. **架构建议**：是新增 `ArgoHTTPClient` 与 `ArgoClient` 共存（策略模式 by env），还是直接替换 `ArgoClient`？
6. **工作量估计**：实现 `ArgoHTTPClient` 大概多少行/多久？

## 代码位置

- `backend/internal/k8s/workflow_client.go` — WorkflowClient interface + ArgoClient
- `backend/internal/k8s/client.go` — k8s.Client + NewClient
- `backend/internal/handlers/pipeline/handler.go` — pipeline handler
- `backend/internal/handlers/workflow/handler.go` — workflow handler
- `backend/cmd/server/core.go` — handler 初始化，`inf.k8sClient` nil check
- `backend/cmd/server/main.go` — infra struct
- `backend/cmd/server/optional.go` — optional handlers（包含 Cloud Run 适配）

## 输出要求

一份简单评估报告，回答问题 1-6。不需要写代码，只需要评估和建议。
