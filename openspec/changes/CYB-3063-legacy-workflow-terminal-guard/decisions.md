## 2026-07-06 — 实现前查代码发现的两处方案调整

- **Context**: design.md 原方案设想"完全重建一个 `*wfv1.Workflow` 对象,两个现有函数(`buildWorkflowDetailNodes`/`buildWorkflowDagEdges`)无缝复用,不改一行"。实现前逐行确认这两个函数的真实依赖,发现两处需要调整:
  1. `models.PipelineRunNode`(DB 表)的 `Inputs`/`Outputs`/`ResourcesDuration` 字段类型是 `map[string]interface{}`(已拍扁的 JSON blob),而 `wfv1.NodeStatus` 对应字段是 Argo 原生结构体指针(`*wfv1.Inputs` 等),两者不能直接赋值。但确认 `buildWorkflowDagEdges` 只读 `NodeStatus` 的拓扑字段(ID/Name/Children/Type/BoundaryID),完全不涉及 Inputs/Outputs;而 `workflowNodeItem.Inputs/Outputs/ResourcesDuration` 字段类型本身是 `any`。
  2. `terminalCapabilityForNode` 内部通过 `isWorkflowNodeRunning(node)` 判断节点是否运行中——降级路径构造的节点 `Phase` 来自 DB 持久化的终态值,天然会让这个判断返回 false,已经给出正确保守的结果,不需要额外写"强制返回不可用"的特殊分支。
- **Decision**:
  1. 构造降级用的 `wfv1.NodeStatus` 时,`Inputs`/`Outputs`/`ResourcesDuration` 留空(零值),不做类型还原;`buildWorkflowDetailNodes` 返回 `[]workflowNodeItem` 后,再用 DB 节点记录做一轮后处理,把这三个字段的原始 map 直接回填进对应 `workflowNodeItem`(按 ID 匹配)。
  2. `terminalCapabilityForNode` 不做任何特殊处理,直接复用现有调用,信任其已有的 `isWorkflowNodeRunning` 判断。
- **Alternatives**: (1) 把 DB 的 map 通过 JSON marshal/unmarshal 往返转换回 `*wfv1.Inputs` 等原生类型,让 `NodeStatus` 在这几个字段上也"看起来完全原生"。(2) 在降级路径下给 `terminalCapabilityForNode` 传一个特殊标志,强制跳过判断直接返回不可用。
- **Rationale**: (1) 的往返转换只是为了让类型"看起来讲究",但下游只有 `workflowNodeItem.Inputs`(`any`类型)在用这份数据,直接赋值 map 语义上完全等价,没必要多绕一层。(2) 会引入一个新的特殊分支,而现有判断逻辑对本场景已经天然正确,新增分支反而增加了以后需要同步维护两处判断标准的风险。两处都遵循"改动面越小、依赖越少越好"的原则。
