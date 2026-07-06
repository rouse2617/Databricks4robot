# Proposal — CYB-3063

## Why
`GET /api/v1/workflows/:name` 无条件直连 Argo 查询实时状态,不管这个 run 是不是早已终态完成——每次打开 WorkflowDetailPage 都会真实打一次 Argo API,即使 run 是三个月前跑完的。新的 Run Kernel 路径已经有终态 guard,这条旧接口是唯一的漏网之鱼。

## What Changes

### Modified Capabilities
- pipeline: `GetWorkflow` 旧接口在 run 已终态且底层 workflow 对象仍存在(未被 Argo TTL 清理)时,改为从数据库已持久化的节点数据构造响应,不再直连 Argo

## Impact
- **Affected code**: `backend/internal/handlers/workflow/handler.go`
- **New APIs**: 无,响应形状不变
- **Dependencies**: 无新增

## Scope
- **In scope**:
  - 判断对应 run 是否终态(复用 `isActiveDeploymentStatus` 同款判断逻辑)
  - 终态且 workflow 对象仍存在时,从 `pipeline_run_nodes` 表构造等价的节点列表,替代遍历 `wf.Status.Nodes`
  - `Progress`、`EstimatedDuration` 字段(DB 未持久化,且对终态 run 本身没有实际意义)在降级路径下留空/零值
  - "能否进容器执行终端命令"(`terminalCapabilityForNode`)在降级路径下直接返回不可用,而不是尝试连接一个可能早已不存在的 pod
- **Out of scope**:
  - workflow 对象已被 Argo TTL 清理的场景——现有行为(404 + 前端"底层 Runtime 已不可用"降级提示)已经合理,不用改
  - 前端 `useWorkflowDetail.ts` 一次 mount 并发 6 个 Run Kernel 子资源请求的问题——建议单独开 singleflight 去重的 ticket,不在本次范围
  - run 找不到对应记录的场景(`h.findPipelineRunByWorkflow` 返回 nil)——保持现有行为,照常直连 Argo

## Success Criteria
- [ ] 已终态、且 Argo workflow 对象仍存在的 run,打开详情页/调用该接口不再触发 `wfClient.GetWorkflow` 调用
- [ ] 活跃(未终态)run 的行为不变,仍然直连 Argo 拿实时状态
- [ ] workflow 已被 TTL 清理的场景行为不变(仍是 404)
- [ ] 降级路径返回的节点列表字段与原 Argo 直连路径在 DB 有对应数据的字段上一致(ID/Name/DisplayName/Type/TemplateName/Phase/Message/PodName/Inputs/Outputs/ResourcesDuration/HostNodeName/Children/StartedAt/FinishedAt)
