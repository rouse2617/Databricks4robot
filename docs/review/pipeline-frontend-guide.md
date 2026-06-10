# Pipeline 前端对接指南

> 版本：2026-05-27
> 读者：做 `Frontend/src/pages/PipelinePage` 及相关组件的前端 / 全栈
> 配套：[pipeline-requirements.md](./pipeline-requirements.md)（产品需求）、[CYB-1254 design](../../openspec/changes/CYB-1254-pipeline-integration/design.md)（集成设计）

---

## 1. 为什么前端「很难做」

难不在 React Flow 本身，而在 **三套模型没对齐**：

| 模型 | 谁定义 | 当前问题 |
|------|--------|----------|
| **画布模型** | React Flow：`Node.id` + `Edge.source/target`（节点 id） | 连线只表示「谁连谁」，没有 port |
| **API 契约** | `transpiler.Pipeline`：`Edge` 为 `node-id.port-name` | 后端按 port 做参数传递 |
| **组件注册表** | `PipelineComponent`（PG + `/api/v1/components`） | 前端仍用 `localStorage` 硬编码 BusyBox 等 |

结果是：

- **依赖顺序**（先跑 A 再跑 B）有时能对上——`buildDAGTemplate` 能从边的 target 推断 `dependencies`。
- **数据传递**（A 的输出作为 B 的输入）在 UI 保存的 JSON 上 **基本不工作**——因为缺少 `a.output → b.input` 这种边，也没有节点的 `inputs`/`outputs` 声明。

后端方向是对的；前端需要 **先认一份唯一契约**，再改 UI，否则会一直在「能画不能跑」里打转。

---

## 2. 唯一契约：Pipeline JSON（与 transpiler 对齐）

后端真源：`backend/internal/transpiler/pipeline.go` + `transpiler_test.go`。

### 2.1 顶层结构

```json
{
  "name": "my-pipeline",
  "version": "1",
  "parallelism": 0,
  "nodes": [ ... ],
  "edges": [ ... ],
  "assetSelection": {
    "assetIds": ["asset-id-1"]
  }
}
```

`assetSelection` 仅前端编排用；部署时把 `assetIds` 放到 `POST /deploy` 的 `asset_ids` 字段（见 §5）。

### 2.2 Node

```json
{
  "id": "step-1",
  "component": {
    "name": "Pass Through",
    "image": "busybox:latest",
    "command": ["sh", "-c"],
    "args": [{ "name": "script", "value": "echo hello" }],
    "resources": { "cpu": "500m", "memory": "256Mi", "disk": "1Gi" }
  },
  "inputs": [{ "name": "input", "type": "asset" }],
  "outputs": [{ "name": "output", "type": "asset" }]
}
```

- `inputs` / `outputs`：**必须存在**（至少各一个默认 port），否则 transpiler 无法生成 `outputs.parameters` 与 DAG task arguments。
- 系统组件 seed 示例：`input` + `output`（见 `pipeline_component` usecase `sys-pass-through`）。

### 2.3 Edge（关键）

```json
{
  "source": "step-1.output",
  "target": "step-2.input"
}
```

格式固定：`{sourceNodeId}.{sourcePortName}` → `{targetNodeId}.{targetPortName}`。

**不要**再导出：

```json
{ "source": "step-1", "target": "step-2" }
```

### 2.4 最小可运行示例（两节点串联）

保存 / 部署 / 单测都应能通过这份 JSON：

```json
{
  "name": "demo-chain",
  "version": "1",
  "nodes": [
    {
      "id": "a",
      "component": {
        "name": "a",
        "image": "busybox:latest",
        "command": ["sh", "-c"],
        "args": [{ "name": "script", "value": "echo a > /tmp/outputs/output" }]
      },
      "outputs": [{ "name": "output", "type": "string" }]
    },
    {
      "id": "b",
      "component": {
        "name": "b",
        "image": "busybox:latest",
        "command": ["sh", "-c"],
        "args": [{ "name": "script", "value": "cat {{inputs.parameters.input}}" }]
      },
      "inputs": [{ "name": "input", "type": "string" }],
      "outputs": [{ "name": "output", "type": "string" }]
    }
  ],
  "edges": [
    { "source": "a.output", "target": "b.input" }
  ]
}
```

容器约定：带 `outputs` 的节点应把结果写到 `/tmp/outputs/{portName}`（transpiler 用 `ValueFrom.Path` 采集）。

---

## 3. 推荐路线：分三期，不要一次重做画布

### Phase A — 契约层（1–2 天，优先）

**目标**：不改画布交互，先让「保存 / 部署」的 JSON 合法。

1. 新增 `Frontend/src/lib/pipelineContract.ts`（或 `components/pipeline/serialize.ts`）集中处理：
   - `toTranspilerPipeline(nodes, edges, meta): Pipeline`
   - `fromTranspilerPipeline(pipeline): { nodes, edges }`
2. 在 `buildPipelineJSON` 里 **只调用** `toTranspilerPipeline`，禁止手写 map。
3. **默认 port 策略**（在仍用「整节点 handle」期间）：
   - 每个节点从组件注册表拷贝 `inputPorts` / `outputPorts`；若无，用 `{ name: "input" }` / `{ name: "output" }`。
   - 每条 React Flow 边导出为
     `source: ${edge.source}.output`（或组件第一个 output port）
     `target: ${edge.target}.input`（或组件第一个 input port）。
4. 单测：`pipelineContract.test.ts`，断言 §2.4 golden JSON。

这样 **Phase A 不强迫做多 handle UI**，但后端已能正确 transpile。

> 可选：与后端协商在 transpiler 增加「仅 node-id 边 → 默认 port」兼容；**短期仍建议前端显式导出 port**，避免隐式行为。

### Phase B — 组件注册表接 API（1–2 天）

**目标**：去掉 `localStorage`，与 F1 对齐。

| 步骤 | 文件 | 动作 |
|------|------|------|
| 1 | 新建 `api/componentApi.ts` | `listComponents`, `createComponent`, `updateComponent`, `deleteComponent` → `/api/v1/components` |
| 2 | 扩展 `types.ts` | `PipelineComponent` 对齐后端 `inputPorts` / `outputPorts` / `source` |
| 3 | `PipelinePage.tsx` | 删除 `STORAGE_KEY`；`useEffect` 拉 `GET /components` |
| 4 | `ComponentManager.tsx` | CRUD 走 API；创建时带默认 ports |
| 5 | `createPipelineNode` | 从 component 的 ports 写入 `node.data.inputPorts` / `outputPorts` |

后端 `PipelineComponent` 字段参考：`backend/internal/models/pipeline_component.go`。

### Phase C — 画布体验（按需，3–5 天）

**目标**：多 port、可维护的 DAG 编辑。

1. **多 Handle**：`PipelineNode.tsx` 按 `inputPorts`/`outputPorts` 渲染多个 `<Handle id={port.name} />`。
2. **连线带 port**：`onConnect` 使用 `connection.sourceHandle` / `targetHandle`；存到 `edge.data` 或直接用 handle id 拼 §2.3 格式。
3. **部署前选资产**：画布「运行」复用 `DeployPanel` 的资产选择 Modal；`deploy(pipeline, name, assetIds)`。
4. **加载 template**：`GET /pipelines/:id` → `fromTranspilerPipeline` 还原画布。
5. **空态 / 清空确认**：onboarding 文案；`Modal.confirm` 包「清空」。

---

## 4. React Flow ↔ Transpiler 映射表

| React Flow | Pipeline JSON | 说明 |
|------------|---------------|------|
| `node.id` | `node.id` | 保持稳定，勿用随机数重建 |
| `node.data`（镜像、command、资源） | `node.component` | 见 `NodeConfigPanel` 字段 |
| `node.data.inputPorts` / `outputPorts` | `node.inputs` / `node.outputs` | 来自组件注册表 |
| `edge.source` + `sourceHandle` | `edge.source` = `` `${id}.${handle}` `` | handle 默认 `output` |
| `edge.target` + `targetHandle` | `edge.target` = `` `${id}.${handle}` `` | handle 默认 `input` |
| `pipelineName` state | `pipeline.name` | |
| — | `asset_ids`（部署 body） | 不在 pipeline JSON 内，见 §5 |

### 当前代码要改的一处（核心）

`PipelinePage.tsx` 中：

```ts
// 今天（错误）
edges: edges.map((e) => ({ source: e.source, target: e.target })),

// Phase A 目标（示例）
edges: edges.map((e) => ({
  source: `${e.source}.${e.sourceHandle ?? defaultOutputPort(e.source)}`,
  target: `${e.target}.${e.targetHandle ?? defaultInputPort(e.target)}`,
})),
```

并在 `nodes.map` 中补上 `inputs` / `outputs`。

---

## 5. API 调用清单（前端已实现 / 待接）

### 5.1 已有 `pipelineApi.ts`

| 方法 | HTTP | 用途 |
|------|------|------|
| `listPipelines` | GET `/pipelines` | 运行记录 Tab 模板列表 |
| `savePipeline` | POST `/pipelines` | 保存 template |
| `deploy` | POST `/deploy` | 画布运行（**需加 `asset_ids`**） |
| `deployTemplate` | POST `/deploy/template/:id` | 模板运行（DeployPanel 已支持 asset） |
| `listDeployments` | GET `/deployments` | 运行历史 |

**待改 `deploy` 签名**：

```ts
export function deploy(
  pipeline: unknown,
  name?: string,
  assetIds?: string[],
): Promise<Deployment> {
  return request<Deployment>("POST", "/deploy", {
    pipeline,
    name,
    asset_ids: assetIds,
  });
}
```

### 5.2 待建 `componentApi.ts`

| 方法 | HTTP |
|------|------|
| `listComponents(q?, source?)` | GET `/components` |
| `createComponent` | POST `/components` |
| `getComponent(id)` | GET `/components/:id` |
| `updateComponent` | PUT `/components/:id` |
| `deleteComponent` | DELETE `/components/:id` |

### 5.3 监控页（已存在，与 Pipeline 页分工）

| 页面 | API | 用户心智 |
|------|-----|----------|
| `/pipeline` Tab「运行记录」 | `/deployments`, `/pipelines` | **我发起的** pipeline run（有 template、可再部署） |
| `/workflows` | `/workflows`, `/workflows/:name` | **集群里** Argo Workflow（排障、看节点 phase、日志） |

侧栏已区分：`流水线` → `/pipeline`，`流水线运行` → `/workflows`。Pipeline 内 Tab 建议改名为 **「部署记录」**，避免与「算法运行记录」混淆。

---

## 6. 文件级改动清单（抄作业用）

```
Frontend/src/
├── api/
│   ├── pipelineApi.ts          # deploy 增加 asset_ids
│   └── componentApi.ts         # 新建
├── lib/  (或 components/pipeline/)
│   ├── pipelineContract.ts     # to/from Transpiler JSON
│   └── pipelineContract.test.ts
├── components/pipeline/
│   ├── types.ts                # PipelineComponent, PortDef, 边上 port
│   ├── PipelineNode.tsx        # Phase C: 多 Handle
│   ├── ComponentManager.tsx    # Phase B: API CRUD
│   ├── DeployPanel.tsx         # 可把 AssetPicker 抽成共享组件
│   └── NodeConfigPanel.tsx     # 可选：展示/编辑 port（P1）
└── pages/
    ├── PipelinePage.tsx        # 串起 contract + API；删 localStorage
    ├── WorkflowListPage.tsx    # 一般不用动
    └── WorkflowDetailPage.tsx  # 一般不用动
```

**不要动**（除非修 bug）：`transpiler/`、`k8s/workflow_client.go` — 契约以它们为准。

---

## 7. 类型定义建议（`types.ts`）

```ts
export interface PortDef {
  name: string;
  type: string;
  desc?: string;
  default_value?: string;
}

export interface PipelineComponent {
  id: string;
  name: string;
  description?: string;
  image: string;
  tag?: string;
  source: "system" | "custom" | "marketplace";
  inputPorts: PortDef[];
  outputPorts: PortDef[];
  resources?: { cpu?: string; memory?: string; disk?: string };
  envVars?: { name: string; value?: string }[];
}

export interface PipelineNodeData {
  label: string;
  image: string;
  command: string[];
  args: Argument[];
  cpu: string;
  memory: string;
  disk: string;
  componentId?: string;
  inputPorts: PortDef[];
  outputPorts: PortDef[];
}
```

`buildPipelineJSON` 时：`inputs` ← `inputPorts`，`outputs` ← `outputPorts`。

---

## 8. 自测清单（改完必做）

### 8.1 契约单测

- [ ] `toTranspilerPipeline` 对 §2.4 golden 输出一致
- [ ] 圆环依赖（若有）前端校验拒绝保存

### 8.2 联调 dev

- [ ] `GET /components` 返回 seed 组件（含 `input`/`output` port）
- [ ] 画布两节点一线 → 保存 → `POST /deploy` → deployment `status` 非立即 Failed
- [ ] 带 `asset_ids` 部署后，容器 env 含 `ASSET_0_ID`（后端已注入）
- [ ] `/workflows` 能看到对应 `workflow_name`
- [ ] 从 dashboard 点「流水线」进入，主区为 Pipeline（`App.tsx` 已有 `key={location.pathname}`）

### 8.3 UX

- [ ] 空画布有引导文案
- [ ] 「清空」有确认
- [ ] 运行失败有 `message.error` 展示后端 message

---

## 9. 常见坑

| 坑 | 说明 |
|----|------|
| 只连节点不连 port | Argo DAG 有依赖无数据；用 §2.3 边格式 |
| 产出路径不对 | 输出 port 对应 `/tmp/outputs/{name}` |
| localStorage 与 DB 双源 | 删 localStorage，只信 API |
| 部署不传 asset_ids | F4 血缘、env 注入都不会生效 |
| 节点 id 每次拖拽重建 | 用递增 `step-N`，加载 template 时保留原 id |
| 与 Workflow 列表混淆 | 产品记录看 deployments；排障看 workflows |

---

## 10. 和后端分工（减少扯皮）

| 话题 | 建议 |
|------|------|
| 边格式 | **前端按 §2.3 导出**；若要坚持「仅 node id」，由后端改 transpiler 并写进 OpenAPI |
| 默认 port 名 | 全栈约定：`input` / `output`（与 seed 一致） |
| 组件镜像展示 | `image` 已含 tag 或 `image:tag` 拼接规则写进 api-guide |
| GPU | 后端 transpiler 未支持前，前端资源表单可不展示 GPU |

---

## 11. 建议执行顺序（给 PM / TL）

1. **Phase A** 合并 — 阻塞「能跑通两节点 pipeline」
2. **Phase B** 合并 — 阻塞「团队共享组件」
3. **Phase C** 按优先级切片（资产选择 > 多 port UI > 布局美化）

每阶段 PR 附：golden JSON + dev 部署截图 + `GET /deployments` 一条成功记录。

---

## 12. 参考代码位置

| 内容 | 路径 |
|------|------|
| Transpiler 边解析 | `backend/internal/transpiler/transpiler.go` — `buildInputSpecs`, `buildDAGTemplate` |
| 单测示例 | `backend/internal/transpiler/transpiler_test.go` — `TestTranspileEdgePorts` |
| 部署与 asset env | `backend/internal/usecase/pipeline/usecase.go` — `Deploy` |
| 组件 API | `backend/internal/handlers/pipeline_component/handler.go` |
| 当前画布 | `Frontend/src/pages/PipelinePage.tsx` |
| 节点 UI | `Frontend/src/components/pipeline/PipelineNode.tsx` |

---

**总结**：前端难，是因为画布模型和 transpiler 契约脱节。**先做 Phase A 的序列化层**，一天内就能让「拖两个盒子、连一条线、点运行」在 Argo 上真跑起来；再接 API 和多 port UI。不要反过来先 polish 画布再补契约。
