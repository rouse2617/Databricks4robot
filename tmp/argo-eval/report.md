# Argo HTTP Client 可行性评估

## 结论

可行。Argo API Server 的 HTTP REST API 足够覆盖 `WorkflowClient` 当前 7 个方法。建议新增 `ArgoHTTPClient` 与现有 `ArgoClient` 共存，通过环境变量选择实现；Cloud Run 默认走 HTTP，集群内或本地开发仍可保留 CRD clientset 路径作为回退。

主要注意点不是 API 覆盖，而是本仓库的类型/依赖边界：当前 `WorkflowClient` interface 直接暴露 `wfv1.Workflow` / `wfv1.WorkflowPhase`，transpiler 和 pipeline usecase 也直接使用 Argo/K8s 类型。因此 `ArgoHTTPClient` 自身可以不依赖 `client-go` 和 Argo clientset，但如果不重构 interface 和 transpiler，整个 backend 不能完全移除 `github.com/argoproj/argo-workflows/v3`、`k8s.io/api`、`k8s.io/apimachinery` 等类型依赖。

参考来源：
- 本地代码：`backend/internal/k8s/workflow_client.go`, `backend/internal/k8s/client.go`, `backend/cmd/server/infra.go`, `backend/cmd/server/core.go`, `backend/internal/usecase/pipeline/usecase.go`, `backend/internal/k8s/metrics.go`
- Argo REST API docs: https://raw.githubusercontent.com/argoproj/argo-workflows/main/api/openapi-spec/swagger.json
- Argo auth docs: https://argo-workflows.readthedocs.io/en/latest/argo-server-auth-mode/
- Argo access token docs: https://argo-workflows.readthedocs.io/en/latest/access-token/
- Argo TLS docs: https://argo-workflows.readthedocs.io/en/latest/tls/
- Argo CLI server mode docs: https://argo-workflows.readthedocs.io/en/release-3.4/cli/argo/

## 1. 可行性：7 个方法覆盖情况

全部可覆盖：

| `WorkflowClient` 方法 | 当前实现 | Argo REST 对应能力 | 评估 |
|---|---|---|---|
| `CreateWorkflow` | `Workflows(namespace).Create` | `POST /api/v1/workflows/{namespace}`，body 为 `WorkflowCreateRequest` | 支持 |
| `GetWorkflowStatus` | `GET Workflow` 后读 `status.phase` | `GET /api/v1/workflows/{namespace}/{name}`，响应为 `Workflow` | 支持；可只读 `status.phase` |
| `DeleteWorkflow` | `DELETE Workflow` | `DELETE /api/v1/workflows/{namespace}/{name}` | 支持 |
| `ListWorkflows` | `List` + label selector | `GET /api/v1/workflows/{namespace}`，支持 `listOptions.labelSelector` | 支持 |
| `GetWorkflow` | `GET Workflow` | `GET /api/v1/workflows/{namespace}/{name}` | 支持 |
| `StopWorkflow` | `GET` 后设置 `spec.shutdown=Stop` 再 `Update` | `PUT /api/v1/workflows/{namespace}/{name}/stop` | 支持，且比当前实现更直接 |
| `GetWorkflowLogs` | K8s 查 pod label，再读 pod logs | `GET /api/v1/workflows/{namespace}/{name}/log`，另有 deprecated pod log endpoint | 支持，但实现方式要调整 |

没有发现 `WorkflowClient` 现有方法在 Argo API Server REST 中不支持。

## 2. 依赖消除

分两层看：

`ArgoHTTPClient` 实现本身可以完全不依赖：
- `k8s.io/client-go`
- `github.com/argoproj/argo-workflows/v3/pkg/client/clientset`
- `k8s.io/metrics/pkg/client/clientset`

但整个 backend 当前不能仅靠新增 `ArgoHTTPClient` 就完全移除 Argo/K8s 依赖，原因：
- `WorkflowClient` interface 使用 `wfv1.Workflow` 和 `wfv1.WorkflowPhase`。
- pipeline transpiler 产出 `*wfv1.Workflow`，并使用 `corev1.EnvVar`、`resource.Quantity`、`metav1` 等 K8s 类型。
- `GetResourceUsage` 通过 `MetricsClient` 走 `backend/internal/k8s/metrics.go`，仍依赖 K8s pods + metrics API。Argo HTTP client 只能解决 workflow 生命周期和 logs，不覆盖当前资源用量接口。

因此建议目标表述为：
- 第一阶段：Cloud Run 不再初始化 `k8s.Client`，pipeline/workflow handlers 可用；`ArgoHTTPClient` 不引入 client-go/clientset。
- 第二阶段：如需从 `go.mod` 移除 K8s/Argo 类型依赖，需要重构 `WorkflowClient` 接口、transpiler 输出模型、resource usage 能力，工作量明显更大。

## 3. GetWorkflowLogs

Argo API Server 提供 logs 端点：`GET /api/v1/workflows/{namespace}/{name}/log`。这个端点支持 `podName`、`logOptions.container`、`logOptions.follow`、`tailLines`、`grep`、`selector` 等 query 参数，并返回 streaming log entries。

对当前 `GetWorkflowLogs(ctx, workflowName, nodeId, namespace)` 来说，不必再直接访问 K8s API。可选实现：

1. 首选：调用 workflow logs endpoint，并传 selector：
   `selector=workflows.argoproj.io/workflow=<workflowName>,workflows.argoproj.io/node-id=<nodeId>`，同时传 `logOptions.container=main`。
2. 备选：先 `GET /api/v1/workflows/{namespace}/{name}`，从 `status.nodes[nodeId]` / children 信息定位 pod 名，再用 `podName` 调 workflow logs endpoint。
3. 不建议继续让 Cloud Run 直接查 K8s pods/log stream；这正是当前不可用的依赖。

实现注意：Argo logs 是流式响应，通常需要按行 decode streaming JSON，把每个 `result.content` / log message 拼成当前 handler 需要的 plain string，再返回 `{ "logs": "..." }`。

## 4. 认证方式

Argo Server 认证模式主要是：

- `server`：Argo Server 使用自己的 ServiceAccount 访问 Kubernetes。客户端对 Argo Server 的身份控制通常应放在外层网关/IAP/Ingress 上。
- `client`：客户端传 Kubernetes bearer token，Argo Server 用客户端身份做 Kubernetes RBAC。Argo v3.0 之后默认是 `client`。
- `sso`：OAuth2/OIDC SSO，可结合 SSO RBAC。

HTTP 客户端侧可支持：
- Bearer token：`Authorization: Bearer <token>`。官方 access token 文档给出的自动化方式是创建最小权限 ServiceAccount token。
- Basic auth：Argo CLI 支持 `ARGO_TOKEN` 以 `Basic ...` 开头，也有 `--username` / `--password` 选项；实际常用于前置代理或 basic header 场景，是否由 Argo Server 本身接受取决于部署/auth mode。
- TLS：Argo Server 支持 HTTP/HTTPS；生产建议通过 HTTPS proxy 做可验证证书。CLI/客户端也支持 client certificate/key 参数，但这更多是传输层/网关或 Kubernetes client auth 能力，不应假设所有 Argo Server 部署都启用了 mTLS。

本项目建议的 Cloud Run 配置：
- `ARGO_SERVER_URL`
- `ARGO_AUTH_TOKEN` 或 Secret Manager 注入的 bearer token
- `ARGO_INSECURE_SKIP_VERIFY=false`，仅本地/临时 dev 允许 true
- 如走 IAP/API Gateway，可加自定义 header 或使用 Cloud Run service-to-service auth，但这属于 Argo Server 前置层设计。

## 5. 架构建议

建议新增 `ArgoHTTPClient`，与 `ArgoClient` 共存，用策略模式按 env 选择，不建议直接替换。

理由：
- 当前 `ArgoClient` 已有测试和本地/in-cluster 使用路径，保留可降低回归风险。
- Cloud Run 的问题在 `setupInfra` 强依赖 `k8s.NewClient`：失败后 `inf.k8sClient=nil`，`setupCore` 不创建 pipeline/workflow handler，最终相关路由不可用/503。HTTP client 应该成为 Cloud Run 初始化路径，不需要 kubeconfig/in-cluster config。
- 共存后可逐步迁移：`ARGO_CLIENT_MODE=http|k8s|auto`。`auto` 可优先 `ARGO_SERVER_URL`，否则回退现有 K8s client。
- `MetricsClient` 是独立能力。HTTP client 上线后，`GetResourceUsage` 可先返回空 pods 或明确“metrics unavailable”，后续再考虑 Prometheus/Argo archive/metrics API 替代。

建议 wiring：
- 新增 `backend/internal/k8s/argo_http_client.go` 或拆到 `backend/internal/argo/`。
- 新增 config：`ARGO_SERVER_URL`, `ARGO_AUTH_TOKEN`, `ARGO_CLIENT_MODE`, `ARGO_TLS_CA_CERT_BASE64`, `ARGO_INSECURE_SKIP_VERIFY`。
- `infra` 不应只保存 `*k8s.Client`，可保存 `workflowClient k8s.WorkflowClient` 和可选 `metricsClient k8s.MetricsClient`。
- `setupCore` 用 `inf.workflowClient` 初始化 pipeline/workflow handler；`metricsClient` 有则注入，无则跳过。

## 6. 工作量估计

最小可用版本：
- 约 250-400 行 Go。
- 1-2 天实现并完成单元测试。

拆分：
- HTTP client struct、request helper、auth/TLS config：80-120 行。
- 7 个 `WorkflowClient` 方法：120-180 行。
- streaming logs decode：50-100 行，取决于对错误流/partial line 的处理。
- config + server wiring：50-100 行。
- 单元测试（httptest 覆盖 create/get/list/delete/stop/logs）：150-250 行。

如果同时重构 interface 以移除 `wfv1` / K8s 类型依赖，或替换 resource usage metrics，预计增加 2-4 天；如果还要做 dev 部署验证和 API smoke，按项目流程整体排期建议预留 3-5 天。
