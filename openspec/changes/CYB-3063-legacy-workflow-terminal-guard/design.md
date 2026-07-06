# Design — CYB-3063

## Architecture Context
- **Constraints**:
  - 响应形状(`GET /api/v1/workflows/:name` 的 JSON 结构)必须保持不变,不能破坏前端现有消费方式
  - 复用已有的 `isActiveDeploymentStatus` 判断逻辑(pipeline usecase 包内已大量使用,保持判断标准一致)
  - `buildWorkflowDetailNodes`/`buildWorkflowDagEdges` 两个现有函数接受的类型是 `*wfv1.Workflow`,不是自定义类型
- **Goals**: 终态且 workflow 对象仍存在的 run,不再触发实时 Argo 调用
- **Non-Goals**: 不改响应形状;不处理 workflow 已被 TTL 清理的场景(现状已合理);不动前端并发请求模式

## Affected Modules
- `backend/internal/handlers/workflow/handler.go` — `GetWorkflow` 加终态判断分支;新增"从 DB 重建 wf 对象"的辅助函数

## Architecture Decisions

### Decision 1: 终态降级路径,重建一个 `*wfv1.Workflow` 对象喂给现有函数,而不是新写一套转换逻辑
- **Approach**: `pipeline_runs.manifest` 字段已经存了这个 run 提交时的完整 Argo Workflow spec(YAML,含 `Spec.Templates`)。终态时:反序列化 `manifest` 拿到 `Spec`,再用 `pipeline_run_nodes` 表(`FindByRunID`)查到的节点记录组装出 `Status.Nodes` map,拼成一个完整的 `*wfv1.Workflow`。这个"重建对象"直接传给现有的 `buildWorkflowDetailNodes(h, run, reconstructedWf)` 和 `buildWorkflowDagEdges(reconstructedWf)`,两个函数体完全不用改。
- **Alternative**: 新写一套独立的"DB 记录 → workflowNodeItem/workflowDagEdge"转换函数,直接操作 `[]models.PipelineRunNode`,不经过 `wfv1.Workflow` 这个中间形态。
- **Rationale**: 选重建对象而不是新写转换逻辑,是因为 `buildWorkflowDetailNodes` 现在有不少字段映射逻辑(`resolveWorkflowPodName`、`workflowNodeTaskNameCandidates`、`workflowStaticDAGTasks` 等),新写一套等价逻辑意味着以后每次这些函数改动都要同步改两遍,容易在两条路径间产生细微不一致。重建对象的做法只需要保证"组装出的 wf 对象在两个函数关心的字段上语义正确",后续维护只有一套代码路径。
- **Trade-off**: 重建对象本身需要一个新的映射函数(`models.PipelineRunNode` → `wfv1.NodeStatus`),但这个映射逻辑比"新写两个响应转换函数"要薄得多。
- **Risk**: `manifest` 字段为 NULL 的历史 run(老版本代码路径可能没写入这个字段)——终态判断成立但 manifest 缺失时,不能构造重建对象。见 Decision 2。
- **Rollback**: 判断分支本身是新增代码,不改动现有直连 Argo 的路径,出问题直接去掉终态分支即可回到原行为。

### Decision 2: `manifest` 缺失时,回退到直连 Argo,不返回残缺响应
- **Approach**: 终态判断成立、但 `run.Manifest == nil` 或反序列化失败时,直接走原有的直连 Argo 路径(等价于回退到修复前的行为),不尝试用不完整的数据拼一个可能字段缺失或错误的响应。
- **Alternative**: 用 `pipeline_run_nodes` 里能查到的数据尽量拼一个残缺响应,拼不出来的字段留空。
- **Rationale**: 数据完整性优先于"绝对不打 Argo"这个目标——manifest 缺失的场景应该是少数历史遗留数据,直连 Argo 兜底比返回一个可能误导用户的残缺响应更安全。且这类 run 大概率也已经跑完很久,即便偶尔多打一次 Argo,代价也可控。
- **Trade-off**: 这批历史 run 无法享受到这次优化,继续保持原有的每次直连行为。

### Decision 3: 判断终态发生在调用 Argo 之前,而不是之后
- **Approach**: 把 `h.findPipelineRunByWorkflow(ctx, name)` 提前到 `h.wfClient.GetWorkflow(...)` 调用之前执行,先拿到 `run`、判断 `isActiveDeploymentStatus(run.Status)`,再决定走哪条分支。
- **Alternative**: 保持现在的调用顺序(先查 Argo,后查 run),只是在拿到两者之后决定用哪份数据拼响应——这样虽然逻辑上也能工作,但完全没有省掉 Argo 调用,违背了本次改动的目的。
- **Rationale**: 本次改动的核心目标就是"终态时不打 Argo",调用顺序必须调整。
- **Risk**: `findPipelineRunByWorkflow` 查不到 run(`run == nil`)时,保持现状直连 Argo(与 proposal.md 的 Out of scope 一致)。

## Data Flow
```
GET /api/v1/workflows/:name
  └─ run, _ := h.findPipelineRunByWorkflow(ctx, name)   [提前到 Argo 调用之前]
  └─ if run != nil && !isActiveDeploymentStatus(run.Status):
       └─ reconstructed, ok := h.reconstructWorkflowFromDB(ctx, run)
            ├─ 反序列化 run.Manifest → wfv1.WorkflowSpec
            └─ h.runNodeRepo.FindByRunID(ctx, run.ID) → 映射成 wfv1.NodeStatus map
       └─ if ok:
            └─ 用 reconstructed 走原有的 buildWorkflowDetailNodes / buildWorkflowDagEdges,拼响应,返回 —— 不调 Argo
       └─ else (manifest 缺失/反序列化失败):
            └─ 走下面的原有路径(直连 Argo)
  └─ (原有路径,活跃 run 或降级失败时都走这里)
       wf, err := h.wfClient.GetWorkflow(ctx, name, namespace)
       └─ buildWorkflowDetailNodes(h, run, wf) / buildWorkflowDagEdges(wf) → 拼响应
```

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 重建的 wf 对象在某些边缘字段上与真实 Argo 返回不完全一致 | 详情页个别字段(如 `Progress`/`EstimatedDuration`)对终态 run 显示为空而非曾经的实时值 | 这两个字段对已终态的 run 本身没有实际意义(不会再变化),明确记录为可接受的降级,proposal.md 已列出 |
| "能否进容器执行终端命令"这个能力判断,在降级路径下如果沿用原逻辑可能误判成可用 | 用户点了终端却连不上一个早已不存在的 pod | 降级路径下 `terminalCapabilityForNode` 直接返回不可用,不依赖重建对象里可能不准确的 pod 存活信息 |
| `manifest` 反序列化逻辑本身出 bug(而不是缺失) | 可能拼出一个字段错误的响应,比静默失败更隐蔽 | 反序列化失败(而不是"manifest 为 nil")也一并归为 Decision 2 的回退条件,不区分"缺失"和"损坏",两者都回退直连 Argo |
