# Pipeline 下一步开发指南

> **读者**：前端 / 后端 / 全栈开发  
> **版本**：2026-05-27  
> **状态**：可开工  
> **前置阅读**：[pipeline-requirements.md](./pipeline-requirements.md)、[pipeline-frontend-guide.md](./pipeline-frontend-guide.md)、[pipeline-ship-readiness.md](./pipeline-ship-readiness.md)

---

## 0. 文档目的

本页是 **「接下来写什么代码」** 的执行清单：按优先级拆任务、标明改哪些文件、API 契约、验收标准与测试步骤。  
PM/决策摘要见 [pipeline-ship-readiness.md](./pipeline-ship-readiness.md)；契约细节见 [pipeline-frontend-guide.md](./pipeline-frontend-guide.md)。

---

## 1. 平台定位（开发前先读）

Pipeline **不是** 整个数据闭环的「中枢」（数据计划 / 计算编排 / 湖仓生产在别的系统）。本模块的定位：

```text
┌─────────────────────────────────────────────────────────────┐
│  Cyber Databrew Pipeline（本仓库 CYB-1254）                  │
│  画布 DAG → Pipeline JSON → Transpiler → Argo → K8s         │
│  可选绑定 asset → 容器 env + 血缘事件                        │
│  监控：/workflows 列表 + 详情 DAG/时间线                      │
└─────────────────────────────────────────────────────────────┘
         │ 输入                    │ 输出
         ▼                         ▼
   资产管理 / 检索            asset_events、RegisterOutput
   （/api/v1/assets）        （/api/v1/pipeline-assets）
```

**与集团数据平台文档的关系**（`~/yx-files-extracted/yx-files/`）：

| 外部原则 | 对本项目的约束 |
|----------|----------------|
| 数据按地域 **就近处理** | 「带资产运行」未来要展示/校验 asset 集群，避免跨域读 `STORAGE_URI` |
| **控制面统一** | 部署记录、Workflow 监控在 Databrew API；执行在 GKE Argo |
| **数据计划 / DataClaw** | 批量跑 pipeline 应走「计划」层，不在画布 fan-out N 次（后续集成） |

---

## 2. 当前实现快照（截至 dev 复验）

### 2.1 已完成（不要再做一遍）

| 能力 | 实现要点 |
|------|----------|
| 三 Tab 页面 | `Frontend/src/pages/PipelinePage.tsx`：画布 / 组件 / 部署 |
| JSON 契约 | `Frontend/src/lib/pipelineContract.ts`：`toTranspilerPipeline` / `fromTranspilerPipeline` |
| 画布部署 | `savePipeline` → `deployTemplate(saved.id)`，避免 FK（`handleDeploy`） |
| 模板一键运行 + 资产下拉 | `DeployPanel`：`handleDirectRun` + Dropdown「选择资产运行」 |
| 模板编辑灌画布 | `DeployPanel` `onEditTemplate` 回调（**仅部署 Tab 内有效**，见 §3.1） |
| 部署记录 → Workflow | `navigate("/workflows/" + d.workflowName)` |
| 命名 | 部署 Tab「部署记录」；`/workflows`「流水线运行」 |
| Transpiler + K8s | `backend/internal/transpiler/*`、`backend/internal/k8s/workflow_client.go` |
| 带资产部署（API） | `usecase.Deploy` 注入 `ASSET_*` env + `_input_asset_ids` + `asset_events` |
| 产出血缘 API | `POST /api/v1/pipeline-assets`、`GET /api/v1/assets/:id/pipeline-lineage` |

### 2.2 未完成 / 有缺陷（本指南重点）

| ID | 问题 | 优先级 |
|----|------|--------|
| **T-01** | `GET /workflows/:name/logs` 未注册路由 | P0 |
| **T-02** | Workflow 详情无日志 UI | P0 |
| **T-03** | 编辑模板：`sessionStorage` 回退路径在同页不生效 | P1 |
| **T-04** | 画布部署弹窗无「可选资产」 | P1 |
| **T-05** | Workflow 列表无分页/筛选 | P1 |
| **T-06** | 组件注册仍用 `localStorage` | P1 |
| **T-07** | 部署成功弹窗无「查看 Workflow」主按钮 | P2 |
| **T-08** | 资产 Modal 文案仍像「必选」 | P2 |
| **T-09** | 画布节点无 port handle，连线靠默认 port | P2 |
| **T-10** | 资产详情页无 pipeline 血缘展示 | P2 |
| **T-11** | OpenAPI 未收录 workflow logs | P3 |

---

## 3. 阶段一：排障闭环 + 小 bug（约 3–4 人日）

目标：**Failed 能看日志**；**编辑模板任意入口可用**；列表可扩展。

### T-01 — 注册 Workflow 日志 API（后端）

**现象**：`GET /api/v1/workflows/{name}/logs?nodeId=xxx` 返回 404。Handler 已实现，路由未挂。

**改动**：

```go
// backend/routes/routes.go — workflowHandler 块内追加
api.GET("/workflows/:name/logs", workflowHandler.GetWorkflowLogs)
```

**注意**：路由顺序保持 `/:name/logs` 在 `/:name` **之后** 已满足（更具体路径需先注册；Gin 按注册顺序匹配，当前 `/:name` 在前会吞掉 `/logs`——应改为：

```go
api.GET("/workflows/:name/logs", workflowHandler.GetWorkflowLogs)
api.GET("/workflows/:name", workflowHandler.GetWorkflow)
```

**验收**：

```bash
curl -s -H "X-Databrew-Token: dev-token" \
  "https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app/api/v1/workflows/<wf-name>/logs?nodeId=<node-id>"
# 期望 200: {"logs":"..."} 或空字符串；非 404
```

**实现参考**：`backend/internal/handlers/workflow/handler.go` `GetWorkflowLogs`；`backend/internal/k8s/workflow_client.go` `GetWorkflowLogs`（按 nodeId 找 Pod）。

**部署**：改 backend 后必须 **deploy backend-dev**（仅 deploy 前端不够）。

---

### T-02 — Workflow 详情日志面板（前端）

**文件**：

- `Frontend/src/api/workflowApi.ts` — 新增
- `Frontend/src/pages/WorkflowDetailPage.tsx` — UI

**API 客户端**：

```typescript
export function getWorkflowLogs(
  workflowName: string,
  nodeId: string,
): Promise<{ logs: string }> {
  return request(
    "GET",
    `/workflows/${encodeURIComponent(workflowName)}/logs?nodeId=${encodeURIComponent(nodeId)}`,
  );
}
```

**UI 行为**（建议）：

1. 右侧详情区在现有 Descriptions 下增加 **Tabs**：`详情` | `日志`。
2. 选中 DAG/时间线上的节点时，`nodeId` 用 `WorkflowNodeStatus.id`（与 Argo template 名一致，与 backend 查 pod 一致）。
3. `日志` Tab：点击时 `getWorkflowLogs(name, nodeId)`；`Spin` + `<pre>`  monospace 展示；失败 toast。
4. 无选中节点时日志 Tab disabled 或提示「请先选择节点」。

**验收**：

1. 打开 `/workflows/<Failed 的 wf>` → 点失败节点 → 日志 Tab 有内容或明确「无日志」。
2. Network 出现 `GET .../logs?nodeId=` 200。

---

### T-03 — 修复模板编辑加载（前端）

**根因**：`DeployPanel.handleEditTemplate` 在无 `onEditTemplate` 时写 `sessionStorage` 并 `navigate("/pipeline")`；用户已在 `/pipeline` 时 **组件不 remount**，`useEffect` 不再次执行。

**现状**：`PipelinePage` 已传 `onEditTemplate`（仅当 `view === "deploy"` 渲染 `DeployPanel` 时）。若从 **侧栏切到流水线且默认画布 Tab**，仍可能不走回调。

**推荐改法（二选一或都做）**：

**方案 A（推荐）— 抽公共函数，去掉 sessionStorage**

```typescript
// PipelinePage.tsx — PipelineCanvas 内
const loadPipelineToCanvas = useCallback((pipeline: Pipeline) => {
  const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
  setNodes(n);
  setEdges(e);
  if (pipeline.name) setPipelineName(pipeline.name);
  setView("pipeline");
  setSelectedNode(null);
  setJsonOutput(null);
}, [setNodes, setEdges]);

// DeployPanel
<DeployPanel onEditTemplate={loadPipelineToCanvas} />
```

删除或保留 `sessionStorage` 仅作深链接备用（可选）。

**方案 B — 保留 sessionStorage，加自定义事件**

```typescript
// 写入后
window.dispatchEvent(new CustomEvent("pipeline-edit", { detail: pipeline }));
// PipelineCanvas 内 listen 并 load
```

**验收**：

1. 部署 Tab → 某模板「编辑」→ **自动切到画布 Tab** 且节点/边可见。
2. 无需 F5。
3. `pipeline.name` 写入顶部名称输入框。

---

### T-05 — Workflow 列表分页与状态筛选（前端）

**文件**：`Frontend/src/pages/WorkflowListPage.tsx`

**改动**：

```typescript
const [statusFilter, setStatusFilter] = useState<string | undefined>();
const filtered = useMemo(
  () => (statusFilter ? items.filter((i) => i.status === statusFilter) : items),
  [items, statusFilter],
);

<Select allowClear placeholder="状态" options={[...]} onChange={setStatusFilter} />
<Table
  dataSource={filtered}
  pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
/>
```

后端若未来支持 `?status=&page=` 再改 `listWorkflows`；当前 **前端过滤 + 分页** 即可满足 U7。

**验收**：`/workflows` 超过 20 条时分页；筛选 `Failed` 只显示失败项。

---

## 4. 阶段二：运行体验对齐（约 3–4 人日）

目标：**画布与模板「运行」语义一致**；资产为可选增强。

### T-04 — 画布部署弹窗支持可选资产（前端）

**文件**：`Frontend/src/pages/PipelinePage.tsx`（部署 Modal）

**交互**（与 DeployPanel 资产 Modal 对齐）：

1. 部署 Modal 增加折叠区 **「高级：绑定资产（可选）」**。
2. 复用 `searchApi.searchAssets` + `Table` 多选（可从 `DeployPanel` 抽 `AssetPicker` 组件到 `Frontend/src/components/pipeline/AssetPicker.tsx`）。
3. `handleDeploy` 在 `deployTemplate(saved.id, assetIds)` 传入所选 ID；未选则 `undefined`。

**后端**：已支持 `POST /deploy/template/:id` body `{ "asset_ids": ["..."] }`，无需改 API。

**验收**：

1. 画布部署时选 1 个 asset → Argo Pod env 含 `ASSET_0_ID`（在 manifest 或集群 describe 验证）。
2. 不选资产 → 与现网一致，仍可成功。

---

### T-07 — 部署成功弹窗增强（前端）

**文件**：`PipelinePage.tsx` 部署成功区块

**改动**：

- Primary：`查看 Workflow` → `navigate(\`/workflows/${result.workflowName}\`)`
- Secondary：`关闭` / `留在部署记录`（`setView("deploy")`）

**验收**：画布部署成功后一键进详情页。

---

### T-08 — 资产 Modal 文案（前端）

**文件**：`DeployPanel.tsx` 资产 Modal

| 现文案 | 改为 |
|--------|------|
| 标题「选择处理资产」 | **「可选：绑定处理资产」** |
| 底部「不选择则直接部署」 | 保留，移到 Modal 顶部 `Alert type="info"` |

---

### T-06 — 组件注册接 API（前端，Phase B）

**真源**：`GET/POST /api/v1/components`（见 `backend/internal/handlers/pipeline_component/handler.go`）

**文件**：

- 新建 `Frontend/src/api/pipelineComponentApi.ts`（或扩展现有 API 模块）
- `Frontend/src/components/pipeline/ComponentManager.tsx`
- `Frontend/src/pages/PipelinePage.tsx` — 初始 load

**流程**：

```typescript
// 启动时
const [components, setComponents] = useState<RegisteredComponent[]>([]);
useEffect(() => {
  listComponents().then(setComponents).catch(() => setComponents(loadComponents())); // fallback localStorage
}, []);

// ComponentManager 保存 → POST/PUT /components，成功后 refresh 列表
```

**类型映射**：API `PipelineComponent` ↔ 画布 `RegisteredComponent`（`name`, `image`, `command`, `args`, `resources`）见 `pipeline-frontend-guide.md` §4。

**验收**：

1. 清 localStorage 后刷新，组件仍从 API 加载（Network `GET /components`）。
2. 新建组件后其他浏览器/session 可见（同源 dev）。

**不在本期**：端口类型 UI（F1.2/F1.3 完整表单）可只读展示默认 input/output。

---

## 5. 阶段三：契约与节点 UX（约 4–5 人日）

### T-09 — React Flow 多 handle（前端）

**文件**：

- `Frontend/src/components/pipeline/PipelineNode.tsx`
- `pipelineContract.ts`（已支持 `sourceHandle` / `targetHandle`）

**改动**：

- 每节点左侧 `Handle type="target" id="input"`，右侧 `Handle type="source" id="output"`。
- `onConnect` 无需改；导出已通过 `formatEdgeEndpoint` 生成 `step-1.output`。

**验收**：连两条边导出 JSON 为 `"source":"step-1.output","target":"step-2.input"`。

---

### T-10 — 资产详情展示 Pipeline 血缘（前端）

**API**：`GET /api/v1/assets/:id/pipeline-lineage`（已实现）

**文件**：资产详情页相关 Tab（搜索 `AssetDetail` / `pipeline-lineage` 引用）

**UI**：展示 `workflow_name`、`pipeline_name`、`deployment_id`、`input_assets`；链到 `/workflows/:name`。

**验收**：对跑过 pipeline 的 asset 能看到 run 链接。

---

### T-11 — OpenAPI 同步（后端/文档）

**文件**：`api/openapi.yaml`

补充：

```yaml
/workflows/{name}/logs:
  get:
    parameters:
      - name: nodeId
        in: query
        required: true
```

---

## 6. 阶段四：生产化增强（按需排期）

以下 **不挡内测**，但对齐数据平台「就近 + 审计」原则。

### T-12 — 部署前 asset 存在性校验（后端）

**文件**：`backend/internal/usecase/pipeline/usecase.go` `Deploy`

当 `len(assetIDs) > 0` 时：

- 任一 ID `assetRepo.Get` 失败 → 返回 400 `INVALID_ARGUMENT`（不要静默跳过 env）。
- 可选：校验 `lifecycle_state` 是否允许处理（见 requirements §9）。

### T-13 — 展示 asset 存储地域/URI（前端）

资产选择表中增加列：`storage_uri`（截断）、`region`（若 API 有）。

部署前若未来有「执行集群」配置，可 Warn：**资产与执行集群不一致**（需产品给集群字段来源）。

### T-14 — 与数据计划集成（架构预留）

**不在本仓库一次做完**；在 `openspec/changes/CYB-1254-pipeline-integration/design.md` 追加 ADR：

- 批量 asset 跑同一 template = **数据计划** 创建 N 次 deployment 或一次 plan-run 带 `asset_ids[]`。
- Pipeline 模板 ID = 计划节点类型引用。

---

## 7. 关键 API 速查（开发拷贝用）

### 7.1 Pipeline

| 方法 | 路径 | Body | 说明 |
|------|------|------|------|
| POST | `/api/v1/pipelines` | `{ name, pipeline }` | 保存模板 |
| GET | `/api/v1/pipelines` | — | 列表 |
| GET | `/api/v1/pipelines/:id` | — | 单模板（编辑用） |
| POST | `/api/v1/deploy` | `{ pipeline, name?, asset_ids? }` | 直部署（画布旧路径） |
| POST | `/api/v1/deploy/template/:id` | `{ asset_ids?: [] }` | **推荐** 模板部署 |
| GET | `/api/v1/deployments` | — | 部署记录 |

### 7.2 Workflow 监控

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/workflows` | 列表 |
| GET | `/api/v1/workflows/:name` | 详情 + nodes |
| GET | `/api/v1/workflows/:name/logs?nodeId=` | Pod 日志（T-01 注册） |

### 7.3 资产 / 血缘

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/search/assets?q=` | 资产搜索（部署选 asset） |
| POST | `/api/v1/pipeline-assets` | 容器回调登记产出 |
| GET | `/api/v1/assets/:id/pipeline-lineage` | 反查 run |

### 7.4 带资产时容器环境变量（已由后端注入）

```text
PIPELINE_DEPLOYMENT_ID
ASSET_IDS          # 逗号分隔
ASSET_COUNT
ASSET_0_ID
ASSET_0_STORAGE_URI
ASSET_0_TYPE
ASSET_1_...
```

算法镜像需自行读取；BusyBox 演示可忽略。

---

## 8. Pipeline JSON 契约（摘录）

完整版见 [pipeline-frontend-guide.md](./pipeline-frontend-guide.md) §2。

```json
{
  "name": "my-pipeline",
  "version": "1",
  "nodes": [{
    "id": "step-1",
    "component": { "name": "...", "image": "...", "command": [], "args": [] },
    "inputs": [{ "name": "input", "type": "string" }],
    "outputs": [{ "name": "output", "type": "string" }]
  }],
  "edges": [{ "source": "step-1.output", "target": "step-2.input" }]
}
```

**所有** 保存/导出/部署/导入 必须经过 `toTranspilerPipeline` / `fromTranspilerPipeline`。

单测参考：`backend/internal/transpiler/transpiler_test.go`（改 golden 前先读现有 case）。

---

## 9. 目录与文件索引

```text
Frontend/
  src/pages/PipelinePage.tsx          # 主页面、画布部署、loadPipelineToCanvas
  src/pages/WorkflowListPage.tsx      # T-05 分页筛选
  src/pages/WorkflowDetailPage.tsx    # T-02 日志 Tab
  src/components/pipeline/
    DeployPanel.tsx                   # 模板运行/编辑/部署记录
    ComponentManager.tsx              # T-06 接 API
    PipelineNode.tsx                  # T-09 handles
  src/lib/pipelineContract.ts         # JSON 契约（勿绕开）
  src/api/pipelineApi.ts
  src/api/workflowApi.ts              # T-02 getWorkflowLogs

backend/
  routes/routes.go                    # T-01 logs 路由
  internal/usecase/pipeline/usecase.go  # Deploy、asset env、lineage
  internal/transpiler/                # DAG → Argo
  internal/k8s/workflow_client.go       # Argo + logs
  internal/handlers/workflow/handler.go
  internal/handlers/pipeline/handler.go
  internal/postgres/pipeline_repo.go    # deployment/template 持久化
```

---

## 10. 测试与部署检查清单

### 10.1 本地

```bash
# 前端
cd Frontend && npx tsc -b --noEmit && npm run test -- --runInBand  # 若有相关单测

# 后端
cd backend && go test ./internal/transpiler/... ./internal/usecase/pipeline/...
```

### 10.2 Dev 部署（必做）

| 变更范围 | 脚本 |
|----------|------|
| 仅 `Frontend/` | `USE_CLOUD_BUILD=false bash deploy/cloudrun/frontend-dev.sh` |
| `backend/` 含 routes | **先** backend-dev **再** frontend-dev |

记录 revision、URL 到 PR。验证步骤见 [deploy-verification.md](../agents/deploy-verification.md)。

### 10.3 冒烟路径（10 分钟）

```text
1. /pipeline → 拖组件 → 导出 JSON（含 inputs/outputs、port 边）
2. 画布「部署」→ 成功 → 「查看 Workflow」
3. 部署 Tab → 模板「运行」（无资产）→ 运行历史「查看」
4. 模板「编辑」→ 画布有节点（T-03）
5. /workflows → 分页/筛选（T-05）→ 点 Failed → 节点日志（T-01+T-02）
```

---

## 11. 建议分工与顺序

| 顺序 | 任务 | 建议负责人 | 依赖 |
|------|------|------------|------|
| 1 | T-01 | 后端 | — |
| 2 | T-02 | 前端 | T-01 部署 backend |
| 3 | T-03 | 前端 | — |
| 4 | T-05 | 前端 | — |
| 5 | T-04、T-07、T-08 | 前端 | 可并行 |
| 6 | T-06 | 全栈 | API 已存在 |
| 7 | T-09、T-10 | 前端 | — |
| 8 | T-12–T-14 | 后端/架构 | 产品确认后 |

**里程碑**：

- **M1（阶段一完成）**：可对外 Beta 的排障能力（日志 + 列表 + 编辑）。
- **M2（阶段二完成）**：运行路径体验统一（画布/模板 + 可选资产）。
- **M3（阶段三完成）**：组件 API 化 + 血缘可见。

---

## 12. 相关文档

| 文档 | 用途 |
|------|------|
| [pipeline-ship-readiness.md](./pipeline-ship-readiness.md) | 上线结论、Dev 复验 |
| [pipeline-qa-report.md](./pipeline-qa-report.md) | 全量用例表 |
| [pipeline-frontend-guide.md](./pipeline-frontend-guide.md) | 契约 Phase A/B/C |
| [openspec/.../CYB-1254/.../design.md](../../openspec/changes/CYB-1254-pipeline-integration/design.md) | 集成设计（需更新路由说明） |
| [deploy-verification.md](../agents/deploy-verification.md) | 部署后验证 |

---

## 13. 附录：UI 参考图

> 本附录回答「有没有图可以照着画 UI」。**`pipeline-next-steps.md` 正文无内嵌图片**；路径均为仓库内或本机剪存目录，在 IDE / Finder 中打开即可。

### 13.1 本仓库 — 流水线页面（优先）

| 路径 | 说明 | 可借鉴 |
|------|------|--------|
| [`pipeline-page.png`](../../pipeline-page.png) | 已接入 DataBrew 侧栏的流水线页（较早截图） | **三栏**：组件 \| 画布 \| 配置；顶栏运行/保存/导出；与现网布局一致 |
| [`pipeline-page-full.png`](../../pipeline-page-full.png) | 全页截图 | 与 dev 对照 |
| [`databrew-pipeline/argo-ui/pipeline-screenshot.png`](../../databrew-pipeline/argo-ui/pipeline-screenshot.png) | 独立 Argo 风原型（深色顶栏、橙色主色） | 顶栏 PIPELINE / REGISTRY / DEPLOY；组件卡片；**换肤/分段 Tab 的备选视觉** |
| `databrew-pipeline/argo-ui/full-screen.png` | 同套原型 | 满画布状态 |
| `databrew-pipeline/argo-ui/tmp-screen.png` | 同套原型 | 中间态 |

**建议**：默认以 **`pipeline-page.png` + 当前 dev** 为 UI 真源；仅调整视觉时可参考 **argo-ui** 原型，不必两套交互并存。

### 13.2 本仓库 — 架构图（定信息架构，非页面像素）

| 路径 | 说明 |
|------|------|
| [`docs/review/assets/architecture-overview.png`](./assets/architecture-overview.png) | 采集 → 算法 → DataBrew 服务（检索/资产/流转） |
| [`docs/review/assets/architecture-evolution.png`](./assets/architecture-evolution.png) | 平台 1.0 → 3.0 演进 |

用于：Pipeline 在「数据流转 / 算法处理」中的位置；侧栏菜单归类。**不**作为画布编辑器线框图。

### 13.3 外部剪存 — `~/yx-files-extracted/yx-files/`

飞书剪存包；**带 `images/` 子目录的 `.md` 同目录** 最易用。HTML 单文件内多为 base64，不便浏览。

列出全部图片：

```bash
find ~/yx-files-extracted/yx-files -type f \( -iname '*.png' -o -iname '*.jpg' \)
```

**与 Pipeline / 编排相关的 subset：**

| 路径（相对 `yx-files/`） | 借鉴用途 |
|--------------------------|----------|
| `团队介绍_2026-04-14 22-01-13/images/jtmmH284W9cjGn0ZYvvTKgESpBabHMSM.jpg` | **智算平台产品架构图**：业务应用 → 算力服务（含「数据计划 / 工作流编排」）→ 引擎 → 调度 |
| `团队介绍_.../images/XSHpeo7dAjGRLD5kL92IOqhqepIFlQmJ.jpg` 等 | 研发方向分层（智算应用 / 统一编排 / 容器服务）— 能力卡片分组 |
| `智数平台脑暴_2026-04-14 21-48-19/images/CCsTMp6JPyIfNHvMtAeM1si4oIkkjj37.jpg` | Data+AI 平台总览：左数据源、中开发、右运维 — **Workflow 列表 + 运维能力** IA |
| `智数平台脑暴_.../T3Qxcb5KUtCjGXjbXVS4SUxp0ab6trxf.png` | 横向数据链路（入湖 → Daft → Lance → 训练）+「工作流编排」— **onboarding / 页脚说明** |
| `智数平台脑暴_.../AZwa5Vm8nHVDPLSlNIcqBOMlGgTjATBt.png`、`bAVxjhViul60KMMfAjtlh45lLtTrbh1C.png` | DataClaw 闭环示意 — 说明 Pipeline 非中枢，与数据计划衔接 |

**弱 UI 模版（背景了解即可）**：`智数平台脑暴_.../iF7ad9bmcCxwbZLC83p9ePJIglNeWAKN.png` 等为汇报风三列卡片，不宜直接当编辑器界面。

### 13.4 屏幕 ↔ 参考图对照（开发对标）

| 屏幕 | 优先参考 | 次要参考 |
|------|----------|----------|
| 画布 / 组件 / 部署 三 Tab | `pipeline-page.png`、`argo-ui/pipeline-screenshot.png` | `Frontend/src/styles/pipeline.css`、现 dev |
| 部署 Tab（模板 + 运行历史） | argo-ui **DEPLOY** 区；智算架构图中的「任务/模板」 | 现 `DeployPanel`（Ant Design Card/List） |
| `/workflows` 列表 | yx 运维列、资产列表页表格风格 | `WorkflowListPage.tsx` |
| `/workflows/:name` 详情（DAG + 时间线 + **日志 T-02**） | **行业**：Argo Workflows / KFP 节点详情 | 本仓库**无**专门 mock，需自设计 |
| 带资产运行 Modal | 资产搜索 / 资产列表页 | `DeployPanel` 资产 Table |

### 13.5 缺口与外部参考

| 缺口 | 建议 |
|------|------|
| Workflow 节点 **Pod 日志** 面板 | 参考 [Argo Workflows UI](https://argo-workflows.readthedocs.io/) 节点日志抽屉；实现见 **T-02** |
| 多 handle 端口连线 | 参考 React Flow 官方 [handles 示例](https://reactflow.dev/examples)；实现见 **T-09** |
| 高保真设计稿（Figma） | 仓库内暂无；若产品出稿，可放到 `docs/review/assets/pipeline-ui/` 并更新本表 |

---

## 14. 变更记录

| 日期 | 说明 |
|------|------|
| 2026-05-27 | 初版：基于 dev 复验 + 产品方向 A（一键运行 / 可选资产）+ yx 数据平台定位 |
| 2026-05-27 | 增加 §13 附录：UI 参考图（本仓库截图、架构图、yx-files 路径） |
