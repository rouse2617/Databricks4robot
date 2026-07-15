# Design — CYB-3076

## Architecture Context
- **Constraints**: Argo pod name = `<wfName>-<templateName>-<hash>` (POD_NAMES=v2) — the pod name embeds the **template name**, so readability requires changing the template name. Template names must be RFC1123-ish and unique per workflow; pod names ≤ 253 chars. Pipeline node IDs are `node-<uuidv4>`; each node has `component.name` (e.g. `head-track-pycuvslam`). Historical `pipeline_run_nodes`/`pipeline_run_asset_nodes` rows store the OLD template name `step-node-<uuid>` as `pipeline_node_id`.
- **Goals**: readable pod names; zero data migration; historical runs keep working (dual-format); no scattered format branches (centralized).
- **Non-Goals**: renaming node IDs; DAG semantics; migrating old rows.

## Affected Modules
- `backend/internal/transpiler` — `templateName`, its callers, validation, output-param wiring (:620), + shared slug.
- `backend/internal/usecase/backfill` — `batchNodeOrderFromPipeline` / `normalizeBatchPipelineNodeID` (node-summary sort).
- `backend/internal/handlers/workflow` — `lookupWorkflowNodeRuntimeInfo` (:377) reverse lookup.
- `Frontend/src/lib/workflowNodeDisplay.ts` — `addPipelineNodeLabelVariants`.

## Architecture Decisions

### Decision 1: Readable template name = `step-<slug>-<uuid8>` via a shared key function
- **Approach**: Add `stepKey(componentName, nodeID) string` = `"step-" + slug(componentName) + "-" + uuid8(nodeID)`. `slug` reuses `sanitizePodNamePart` (lowercase, `[a-z0-9-]`, trim), truncated (~30 chars). `uuid8` = first 8 hex of the node UUID (after stripping the `node-` prefix). Empty component → fall back to the legacy `step-<nodeID>` so nothing regresses. This one function is the single source of the format; the transpiler and backend matchers all call it.
- **Alternative**: Argo DAG task-name trick (task name = node id → DisplayName carries id; template = readable) to avoid changing the join key — rejected: it juggles 3 outputs (pod name / pipeline_node_id / display_name) across 2 Argo fields and couples to DisplayName semantics; harder to reason about than a centralized dual-format key.
- **Rationale**: pod name must equal template name; a single shared formatter keeps it consistent and clean.
- **Risk**: collision if two nodes share component AND uuid8 → transpile map build detects a dup and falls back to a longer uuid for the colliding node (never rejects a valid pipeline).
- **Rollback**: revert; new runs return to `step-node-<uuid>`. No stored-data change.

### Decision 2: Fix the output-param wiring to use the template map (transpiler.go:620)
- **Approach**: `buildDAGTemplate`'s output-param reference `{{tasks.<name>.outputs...}}` currently recomputes `templateName(is.srcNode)` — but at that point there is no `component` for `srcNode`. Change it to look up `nodeTemplates[is.srcNode]` (the map already built from all nodes/subnodes, so it holds the correct readable name).
- **Rationale**: task names/dependencies already use the map; the output wiring is the only spot that recomputes — aligning it keeps the DAG internally consistent.

### Decision 3: Dual-format, candidate-key matching (no migration, centralized)
- **Approach**: Where a stored `pipeline_node_id` (old or new format) must be related to a pipeline-definition node, build a small **candidate-key set** per definition node — `{ stepKey(component,id), legacyKey(id)=step-<id> }` — and match membership. Confined to:
  - backfill `batchNodeOrderFromPipeline`/`normalizeBatchPipelineNodeID` (sort): read `component.name` too, generate candidates, match stored value → correct order for old + new runs.
  - workflow `lookupWorkflowNodeRuntimeInfo`: try new stepKey mapping AND the existing `TrimPrefix "step-"` path.
  - frontend `addPipelineNodeLabelVariants`: register the new `step-<slug>-<uuid8>` variant in addition to the existing variants (the frontend already registers multiple variants — idiomatic).
- **Alternative**: only-new matching (Option B) — rejected by user: old batch runs' sort/labels/runtime-info would degrade (data still queryable, but display worse).
- **Rationale**: candidate-key set is one cohesive pattern (already used in the frontend), not scattered conditionals; the legacy branch is removable once old runs age out.

## Data Flow
```
transpile:  node(id, component) --stepKey--> template "step-<slug>-<uuid8>"  -> pod name readable
run refresh: Argo NodeStatus.TemplateName ("step-<slug>-<uuid8>") -> pipeline_run_nodes.pipeline_node_id
match (sort/label/runtime): definition node -> {stepKey(component,id), step-<id>} candidate set
   -> matches BOTH new-format (new runs) and legacy (historical runs)
```

## Data Model Changes
- **None.** No migration. Old rows keep `step-node-<uuid>`; new rows get `step-<slug>-<uuid8>`; matching accepts both.

## Risks / Trade-offs
| 风险 | 影响 | 缓解 |
|------|------|------|
| component+uuid8 撞名 | 合法 pipeline 被拒 | transpile 建映射时检测重复→该节点回退更长 uuid |
| 历史 run 只认旧格式 | 排序/label/富信息降级 | 候选键集合含旧格式(dual);老 run 不降级 |
| slug 过长 | pod 名逼近 253 | slug 截断(~30);uuid8 定长 |
| 前后端 slug 不一致 | 前端 label 匹配不到 | slug 规则文档化 + 前端镜像后端逻辑 + 单测锁定 |
| 格式分支堆积 | 代码变乱 | 单一 stepKey 函数 + 候选键模式集中;旧支可在老 run 退役后删除 |
