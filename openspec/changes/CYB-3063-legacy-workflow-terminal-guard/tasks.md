# Tasks — CYB-3063

## Context files
```
backend/internal/handlers/workflow/handler.go   # GetWorkflow(~L203)、buildWorkflowDetailNodes
backend/internal/handlers/workflow/dag_edges.go  # buildWorkflowDagEdges
backend/internal/postgres/pipeline_repo.go       # PipelineRunNodeRepo.FindByRunID(~L1399)
backend/internal/models/pipeline.go              # PipelineRun.Manifest 字段(~L47/93)
backend/internal/usecase/pipeline/usecase.go     # isActiveDeploymentStatus 判断逻辑参照
```

## Implementation

- [ ] [backend] `handler.go`: 把 `h.findPipelineRunByWorkflow(ctx, name)` 提前到 `h.wfClient.GetWorkflow(...)` 调用之前
- [ ] [backend] 新增 `reconstructWorkflowFromDB(ctx, run) (*wfv1.Workflow, bool)`:反序列化 `run.Manifest` 拿 `Spec`,查 `PipelineRunNodeRepo.FindByRunID` 拿节点记录并映射成 `Status.Nodes` map;`Manifest` 为空或反序列化失败时返回 `ok=false`
- [ ] [backend] `GetWorkflow`: `run != nil && !isActiveDeploymentStatus(run.Status)` 时,先尝试 `reconstructWorkflowFromDB`;成功则用重建对象走现有 `buildWorkflowDetailNodes`/`buildWorkflowDagEdges` 拼响应并直接返回,不调 Argo;失败或 run 为 nil 或 run 活跃,走原有直连 Argo 路径
- [ ] [backend] 降级路径下,`terminalCapabilityForNode` 相关的终端能力判断直接返回不可用(不依赖重建对象里的 pod 存活信息)

## Scenario coverage(测试)

- [ ] [backend] 单测:终态 run + 有效 manifest + 有节点记录 → 走降级路径,不调用 `wfClient.GetWorkflow`,响应核心字段(nodes/edges/status)与直连路径语义一致 —— 覆盖 *"已终态且底层对象仍存在的 run"*
- [ ] [backend] 单测:活跃 run → 仍然调用 `wfClient.GetWorkflow`,行为不变 —— 覆盖 *"活跃(未终态)run 保持原有行为"*
- [ ] [backend] 单测:终态 run 但 `Manifest` 为 nil / 反序列化失败 → 回退直连 Argo,不返回残缺响应 —— 覆盖 *"构造降级响应所需的历史数据缺失"*

## API contract sync
N/A —— 响应形状不变,不新增/修改 HTTP 路由、请求、响应结构。

## 验证(Tier M —— 单文件逻辑改动,涉及新的分支判断)
- [ ] `make fmt && make vet`
- [ ] `go test ./internal/handlers/workflow/...`
- [ ] `go test ./...`(全量,按本次会话的一贯做法)

## 部署验证
- [ ] 部署到 Cloud Run dev,独立核实 revision 健康(Ready/ENV/流量)
- [ ] 找一个已知终态、且 Argo 对象大概率还没被 TTL 清理的 run(比如今天/昨天跑完的),调 `GET /api/v1/workflows/:name`,确认响应字段正常
- [ ] 通过后端日志或响应耗时,确认这次请求确实没有触发对 Argo 的实时调用(降级路径应明显更快,且不应出现调用 Argo 相关的日志行)
- [ ] 找一个活跃(仍在跑)的 run,确认行为不变(照常直连 Argo,数据是实时的)
