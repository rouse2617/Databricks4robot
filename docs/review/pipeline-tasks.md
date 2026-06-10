# Pipeline 智算平台 — 完整 Task 分解（100+）

> 版本：2026-05-27 v2
> 基准：feat/pipeline-integration 分支
> 角色：产品经理视角，覆盖完整平台能力

---

## 说明

每个 Task 格式：

| 字段 | 说明 |
|------|------|
| **目标** | 一句话描述做什么 |
| **用户故事** | 谁在什么场景下用 |
| **文件** | 涉及文件（含行号参考） |
| **验收** | 怎么算做完 |
| **依赖** | 前置 Task |
| **估行** | 后端 / 前端 / 迁移 |

EPIC 按产品域分组，每个 EPIC 有独立的用户故事和成功指标。

---

## 目录

| EPIC | 领域 | Task 数 |
|------|------|---------|
| — | Phase 0 清理 | 3 |
| A | Pipeline Designer 画布体验 | 16 |
| B | 组件注册表管理 | 11 |
| C | Transpiler 引擎增强 | 12 |
| D | 执行与部署 | 10 |
| E | 运行监控与可观测性 | 12 |
| F | Asset-Pipeline 集成 | 9 |
| G | Backfill 批回放 | 7 |
| H | 调度与触发器 | 8 |
| I | 密钥与配置管理 | 7 |
| J | 通知与告警 | 6 |
| K | 权限与治理 | 9 |
| L | Template 管理增强 | 8 |
| M | API 与开发者工具 | 8 |
| N | 协作与市场 | 8 |
| O | UX 体验打磨 | 8 |
| P | 高级功能 | 11 |
| Q | 资产平台核心能力增强 | 35 |
| R | 资产-Pipeline 深度集成 | 10 |
| | **合计** | **~190** |

---

## Phase 0 清理（P0-CLEANUP）

### P0-CLEANUP-1：修复 Usecase.Deploy() 缺少 template_id

**目标**：`DeployByTemplateID` → `Deploy` 时传入 template_id，避免 pipeline_deployments.template_id 始终 NULL。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `Deploy()` 签名增加 `templateID string` |
| `backend/internal/handlers/pipeline/handler.go` | 两处调用处补齐 |

**验收**：从 template 部署后 DB 中 template_id 不为 NULL；从画布直接部署为 NULL。

**估行**：后端 ~10 行

---

### P0-CLEANUP-2：Pipeline 专属错误码

**目标**：当前复用 ASSET_NOT_FOUND，需独立错误码体系。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | ASSET_NOT_FOUND → 新码 |
| `backend/internal/usecase/pipeline/usecase.go` | 新增 ErrTemplateNotFound / ErrDeploymentNotFound / ErrInvalidDAG / ErrTranspileError / ErrDeployFailed |

**验收**：无效 template ID → `PIPELINE_TEMPLATE_NOT_FOUND`；无效 DAG → `PIPELINE_INVALID_DAG`。

**估行**：后端 ~20 行

---

### P0-CLEANUP-3：assetSelection 后端同步

**目标**：前端 `Pipeline.assetSelection` 已定义，后端 transpiler `Pipeline` struct 无对应字段，静默丢数据。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | Pipeline 增加 `AssetSelection *AssetSelection`；新增 `AssetSelection` 类型 |

**验收**：前后端传输 assetSelection 不丢失。

**估行**：后端 ~15 行

---

## EPIC-A：Pipeline Designer 画布体验

**用户故事**：作为数据工程师，我每天在画布上编排 pipeline，希望工具顺手、不卡顿、操作可逆。
**成功指标**：用户从打开到完成编排 < 3 分钟（对 5 节点 DAG）；误操作可恢复。

---

### A-1：Undo / Redo

**目标**：画布操作（添加/删除/移动/连线/改参）支持撤销重做，快捷键 Ctrl+Z / Ctrl+Shift+Z。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | 集成 undo/redo 状态管理（useReducer / zustand） |
| `Frontend/src/components/pipeline/CanvasHistory.tsx` | 新建：历史操作栈，记录每一步变更的快照 |
| `Frontend/src/components/pipeline/PipelineToolbar.tsx` | 新增撤销/重做按钮 |

**验收**：
1. 删除节点 → Ctrl+Z → 节点恢复
2. 拖动节点 → Ctrl+Z → 回到原位置
3. 连续操作 20 步 → Ctrl+Z 依次回退
4. 画布 Toolbar 显示撤销/重做按钮，灰色时不可用

**估行**：前端 ~200 行

---

### A-2：自动布局（Dagre）

**目标**：一键自动排列 DAG，按拓扑序从上到下布局，减少手动拖拽。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/package.json` | 新增 `dagre` 依赖 |
| `Frontend/src/components/pipeline/PipelineToolbar.tsx` | 新增"自动布局"按钮 |
| `Frontend/src/pages/PipelinePage.tsx` | 自动布局函数：解析 edges → dagre 计算位置 → setNodes |

**验收**：
1. 乱序放置的 5 个节点 → 点"自动布局" → 整齐排列
2. 有依赖关系的节点在不同层级
3. 多次点击不爆炸（每次重新计算）

**估行**：前端 ~80 行

---

### A-3：画布小地图（Minimap）

**目标**：大 pipeline（20+ 节点）时小地图辅助导航。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | ReactFlow 增加 `<MiniMap>` 组件 |

**验收**：画布右下角显示缩略图，拖拽可移动视口。

**估行**：前端 ~10 行

---

### A-4：键盘快捷键

**目标**：Delete 删选中节点、Ctrl+C/V 复制粘贴、Ctrl+A 全选、Ctrl+S 保存。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | 注册 keydown 事件监听，调用对应 React Flow API |

| 快捷键 | 操作 |
|--------|------|
| Delete / Backspace | 删除选中节点/边 |
| Ctrl+C | 复制选中节点 |
| Ctrl+V | 粘贴 |
| Ctrl+A | 全选 |
| Ctrl+S | 保存当前 pipeline |
| Ctrl+Z / Ctrl+Shift+Z | Undo / Redo |
| Ctrl+D | 复制选中节点并粘贴（duplicate） |
| Escape | 取消选中 / 关闭面板 |

**验收**：每种快捷键有效，不影响输入框中的正常输入（通过 `target.tagName` 过滤）。

**估行**：前端 ~120 行

---

### A-5：多选 + 批量操作

**目标**：框选或 Ctrl+Click 选中多个节点，批量删除/移动/复制。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | ReactFlow `selectionOnDrag` 开启框选；`multiSelectionKeyCode` 设 Ctrl |
| `Frontend/src/components/pipeline/PipelineNode.tsx` | 选中态样式高亮 |
| `Frontend/src/components/pipeline/PipelineToolbar.tsx` | 批量删除按钮（选中 >1 时显示） |

**验收**：
1. 画布上拖出矩形 → 框选多个节点
2. 右键 → "批量删除" → 移除所有选中节点及其连线

**估行**：前端 ~60 行

---

### A-6：节点分组 / 文件夹

**目标**：选中多个节点 → 右键"创建分组" → 折叠/展开，降低画布复杂度。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | 分组逻辑：创建 Group Node，子节点变为 group 的 children |
| `Frontend/src/components/pipeline/PipelineNode.tsx` | Group Node 渲染（背景色、标题栏、折叠按钮） |
| `Frontend/src/components/pipeline/types.ts` | `GroupNodeData` 类型 |

**验收**：
1. 选中 3 个节点 → 右键"分组" → 3 个节点被包含在 Group Node 内
2. 点 Group Node 的折叠按钮 → 子节点隐藏
3. 展开 → 子节点恢复
4. 移动 Group Node → 子节点跟随

**估行**：前端 ~200 行

---

### A-7：注释 / 便签节点

**目标**：画布上添加文本框节点（不可执行），用于标注、说明、TODO。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/PipelineNode.tsx` | 新增 NoteNode 类型（不同颜色、虚线边框） |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 组件面板增加"便签"项 |
| `Frontend/src/components/pipeline/types.ts` | `NoteNodeData` 类型 |

**验收**：拖入便签节点 → 双击编辑文本 → 只做展示不参与 transpile。

**估行**：前端 ~80 行

---

### A-8：全屏模式

**目标**：点"全屏"让画布占满浏览器窗口，排除侧边栏干扰。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | Fullscreen API + 退出按钮 |

**验收**：点全屏 → 画布充满窗口 → 按 Esc 退出。

**估行**：前端 ~30 行

---

### A-9：网格对齐吸附

**目标**：拖动节点时自动吸附到网格线，排列整齐。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | ReactFlow `snapToGrid` + `snapGrid` 设置 20x20 |

**验收**：拖节点时自动吸附。

**估行**：前端 ~5 行

---

### A-10：右键上下文菜单

**目标**：右键画布/节点弹出操作菜单（删除、复制、分组、配置）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/ContextMenu.tsx` | 新建：右键菜单组件，按点击对象显示不同选项 |
| `Frontend/src/pages/PipelinePage.tsx` | 集成 ContextMenu |

**验收**：右键节点 → 菜单含"删除/复制/配置"；右键空白处 → "粘贴/自动布局/全选"。

**估行**：前端 ~150 行

---

### A-11：Pipeline 缩略图预览

**目标**：在 template 列表 / 部署列表中展示 pipeline 的缩略 SVG 图。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/PipelineThumbnail.tsx` | 新建：接收 nodes+edges 渲染小型 React Flow 图（只读、fitView） |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署列表行中展示缩略图 |
| `Frontend/src/pages/PipelinePage.tsx` | template 列表卡片展示缩略图 |

**验收**：template 列表 > 每项左侧展示迷你 DAG 图。

**估行**：前端 ~100 行

---

### A-12：画布缩放控件增强

**目标**：状态栏显示当前缩放比例，提供"适应画布"、"100%"、"缩放到选中"。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/PipelineToolbar.tsx` | 缩放指示器 + 按钮组 |

**验收**：缩放时百分比变化；点"适应" → 全部节点可见；点"100%" → 回到 1:1。

**估行**：前端 ~50 行

---

### A-13：参数配置面板优化（Schema 驱动）

**目标**：节点配置面板根据组件声明的参数 schema（类型/枚举/默认值/必填）自动渲染表单，而非纯文本输入。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/NodeConfigPanel.tsx` | 根据 `component.args` schema 动态渲染 input/select/switch/number |
| `Frontend/src/components/pipeline/types.ts` | ArgumentSchema 类型增加 `type`, `required`, `options`, `default` |

**验收**：
1. 组件声明 `{name: "threshold", type: "number", default: 0.5, min: 0, max: 1}` → 渲染数字滑动条
2. 组件声明 `{name: "mode", type: "enum", options: ["fast", "accurate"]}` → 渲染下拉框
3. 必填参数未填时部署按钮灰色 + 提示

**估行**：前端 ~180 行

---

### A-14：Pipeline 模板预设参数覆盖

**目标**：保存 template 时可以为参数设默认值；加载后用户在部署弹窗中可覆盖。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/NodeConfigPanel.tsx` | 参数面板中区分"默认值"和"当前值" |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署弹窗增加"参数覆盖"区域 |
| `Frontend/src/api/pipelineApi.ts` | deploy API 增加 `params` 覆盖字段 |

**验收**：
1. 参数有默认值 → 加载 template 时自动填入
2. 部署弹窗中可修改参数 → 生效
3. 不改参数 → 使用默认值

**估行**：前端 ~120 行

---

### A-15：Pipeline 名称 / 描述 / 标签

**目标**：保存和编辑 pipeline 时填写名称、描述、颜色标签。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/PipelineInfoPanel.tsx` | 新建：标题 + 描述 + 标签编辑区域 |
| `Frontend/src/pages/PipelinePage.tsx` | 在画布上方展示信息栏 |
| `Frontend/src/components/pipeline/types.ts` | Pipeline 增加 `description`, `tags`, `color` |

**验收**：
1. 未命名 pipeline → 标题显示"未命名流水线"
2. 编辑名称/描述/标签 → 保存 → 重新加载后保留

**估行**：前端 ~100 行

---

### A-16：组件面板搜索/筛选

**目标**：组件库支持搜索和分类筛选（system / custom / GPU / 按名称）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 搜索框 + 分类 tab / 下拉筛选 |

**验收**：输入关键词实时过滤；切分类只显示该类组件。

**估行**：前端 ~50 行

---

## EPIC-B：组件注册表管理

**用户故事**：作为算法工程师，我写完镜像后能在平台注册为组件，管理版本、声明接口、写文档。
**成功指标**：从推送镜像到可在画布拖出组件 ≤ 2 分钟。

---

### B-1：组件 CRUD API + 表

**目标**：组件注册表增删改查（同原有 P1-F1.1）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/040_component_tables.sql` | CREATE TABLE pipeline_components |
| `backend/internal/models/pipeline.go` | `PipelineComponent` struct |
| `backend/internal/repository/pipeline_repository.go` | `PipelineComponentRepository` interface |
| `backend/internal/postgres/pipeline_repo.go` | PG 实现 |
| `backend/internal/handlers/pipeline/handler.go` | handler |
| `backend/internal/usecase/pipeline/usecase.go` | usecase |
| `backend/routes/routes.go` | 路由注册 |

**验收**：标准 CRUD 全部可用。

**估行**：后端 ~200 行 + 1 迁移

---

### B-2：组件管理前端页面

**目标**：组件列表页（CRUD 表格/卡片），从硬编码迁移到 API（同原有 P1-F1.2）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/ComponentManager.tsx` | API 加载、表格展示、新增/编辑弹窗 |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | API 加载组件列表 |

**验收**：组件面板数据来自 API；新增/编辑/删除组件有效。

**估行**：前端 ~200 行

---

### B-3：组件版本管理

**目标**：同一组件名可发布多版本，画布使用时选择版本。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/040_component_tables.sql` | `pipeline_component_versions` 表（id, component_id, version, spec JSONB, changelog, created_at） |
| `backend/internal/handlers/pipeline/handler.go` | `POST /components/:id/versions`, `GET /components/:id/versions` |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 组件卡片显示版本号，下拉选版本 |

**验收**：
1. 组件 v1 → 使用中 → 发版 v2 → v1 的 pipeline 不受影响
2. 新拖入的组件默认最新版

**估行**：后端 ~100 行 + 前端 ~80 行 + 1 迁移

---

### B-4：组件搜索 / 高级筛选

**目标**：按名称/镜像/标签/分类/GPU 支持搜索组件。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /components` 支持 `?q=&category=&has_gpu=` |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 搜索框 + 筛选条件 |

**验收**：搜索"ocr" → 只显示含 OCR 的组件；筛选"GPU" → 只显示 GPU 组件。

**估行**：后端 ~30 行 + 前端 ~50 行

---

### B-5：组件测试（Sample Data）

**目标**：在组件详情页点"测试" → 输入测试参数 → 创建一个临时 workflow 运行 → 实时看日志和结果。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `TestComponent()`：构建单节点 workflow + 提交 Argo |
| `backend/internal/handlers/pipeline/handler.go` | `POST /components/:id/test` |
| `Frontend/src/pages/ComponentDetailPage.tsx` | "测试" Tab：参数输入 → 运行 → 日志输出框 |

**验收**：
1. 填参数 → 点"测试" → 创建单步 workflow
2. 页面实时显示日志流
3. 完成后显示 exit code + 耗时

**估行**：后端 ~80 行 + 前端 ~150 行

---

### B-6：组件文档（Markdown）

**目标**：组件详情支持写 Markdown 文档（用法、注意事项、示例），保存到 DB。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/040_component_tables.sql` | `pipeline_components` 增加 `documentation TEXT` |
| `Frontend/src/pages/ComponentDetailPage.tsx` | "文档" Tab：Markdown 编辑器 + 渲染预览 |

**验收**：保存后重新打开 → Markdown 渲染正常。

**估行**：前端 ~100 行 + 后端 ~10 行

---

### B-7：组件分类管理

**目标**：Admin 管理组件分类（新增/编辑/删除），组件可归属分类。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/040_component_tables.sql` | `component_categories` 表 |
| `backend/internal/handlers/pipeline/handler.go` | 分类 CRUD |
| `Frontend/src/pages/ComponentCategoriesPage.tsx` | 新建：分类管理页 |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 按分类展示组件 |

**验收**：新增分类 → 组件可归入 → 侧边栏按分类展示。

**估行**：后端 ~80 行 + 前端 ~120 行 + 1 迁移

---

### B-8：从 Docker Registry 导入

**目标**：输入镜像名称（`registry.example.com/my-processor:latest`），自动拉取镜像元信息（labels、env、entrypoint）填充组件字段。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `ImportFromDocker(image)`: 拉取 manifest + inspect → 解析 label → 生成组件 |
| `backend/internal/handlers/pipeline/handler.go` | `POST /components:import` Body: `{image: "..."}` |
| `Frontend/src/components/pipeline/ComponentImporter.tsx` | 新建：输入镜像 → 确认导入 |

**验收**：输入镜像 → 自动填充名称、描述、entrypoint、默认 env。

**估行**：后端 ~60 行 + 前端 ~80 行

---

### B-9：组件使用统计

**目标**：查看组件被多少 pipeline 引用、最近运行次数、成功率。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `GetComponentStats(id)`: 查询 pipeline_templates JSONB + pipeline_deployments |
| `backend/internal/handlers/pipeline/handler.go` | `GET /components/:id/stats` |
| `Frontend/src/pages/ComponentDetailPage.tsx` | "统计" Tab |

**验收**：组件详情页显示"被 X 个 pipeline 使用，近 7 天运行 Y 次，成功率 Z%"。

**估行**：后端 ~50 行 + 前端 ~60 行

---

### B-10：组件废弃/弃用警告

**目标**：标记组件为 deprecated，已有 pipeline 显示黄色警告"此组件已废弃，建议升级到 vX"。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/040_component_tables.sql` | `pipeline_components` 表增加 `status TEXT DEFAULT 'active'`, `deprecation_notice TEXT`, `replacement_id TEXT` |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | deprecated 组件标灰、显示替换建议 |
| `Frontend/src/components/pipeline/PipelineNode.tsx` | 使用已废弃组件的节点显示 ⚠️ 图标 + tooltip |

**验收**：标记废弃后组件面板显示弃用标识；已使用的 pipeline 节点上出现警告。

**估行**：后端 ~40 行 + 前端 ~80 行

---

### B-11：组件 YAML 定义预览

**目标**：组件详情页展示其 transpile 后的 Argo template YAML 片段。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/ComponentDetailPage.tsx` | "YAML" Tab：展示组件被 transpile 后的 Argo template 格式 |

**验收**：打开 YAML Tab → 语法高亮显示 Argo YAML。

**估行**：前端 ~50 行

---

## EPIC-C：Transpiler 引擎增强

**用户故事**：作为算法工程师，我希望 pipeline 在 K8s 上执行时稳定、高效、不踩坑。
**成功指标**：transpiler 生成的 YAML 能被 Argo 直接接受，无需手动修改。

---

### C-1：RetryStrategy

**目标**：组件支持重试配置（次数 + backoff），transpiler 写入 YAML（同原有 P1-F3.1）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Component` 增加 `RetryPolicy` |
| `backend/internal/transpiler/transpiler.go` | `buildNodeTemplate()` 追加 retryStrategy |
| `backend/internal/transpiler/transpiler_test.go` | 测试用例 |

**验收**：RetryPolicy 为空 → YAML 不含 retryStrategy；Limit=3 → YAML 含 limit: "3"。

**估行**：后端 ~40 行

---

### C-2：超时配置

**目标**：Step 级别 + Workflow 级别超时（同原有 P1-F3.2）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Component` 增加 `ActiveDeadlineSeconds`；`Options` 增加 `Timeout` |
| `backend/internal/transpiler/transpiler.go` | 写入 `activeDeadlineSeconds` |

**验收**：无超时 → YAML 不含；设 3600 → YAML 出现。

**估行**：后端 ~20 行

---

### C-3：GPU 资源支持

**目标**：`ResourceRequirements` 支持 GPU，transpiler 写 `nvidia.com/gpu`（同原有 P1-F3.3）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `ResourceRequirements` 增加 `GPU string` |
| `backend/internal/transpiler/transpiler.go` | `buildNodeTemplate()` 处理 GPU |
| `Frontend/src/components/pipeline/types.ts` | 同步加 `gpu?: string` |

**验收**：GPU: "1" → YAML 含 `nvidia.com/gpu: "1"`；前端节点配 GPU。

**估行**：后端 ~10 行 + 前端 ~20 行

---

### C-4：并行度控制（Parallelism）

**目标**：Workflow 级别限制最大并行 step 数，防止打爆集群。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Options` 增加 `Parallelism int32` |
| `backend/internal/transpiler/transpiler.go` | `Transpile()` → `Spec.Parallelism = &opts.Parallelism` |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署弹窗增加并行度配置 |

**验收**：Parallelism=3 → YAML 含 `parallelism: 3`；默认不设时 YAML 不包含。

**估行**：后端 ~10 行 + 前端 ~30 行

---

### C-5：Volume 挂载

**目标**：组件声明需要挂载的 volume（ConfigMap / PVC / EmptyDir），transpiler 写入。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Component` 增加 `Volumes []VolumeMount`；`VolumeMount` struct |
| `backend/internal/transpiler/transpiler.go` | buildNodeTemplate 处理 volume mounts + 合并 Workflow-level volumes |
| `Frontend/src/components/pipeline/types.ts` | 前端类型同步 |

```go
type VolumeMount struct {
    Name       string `json:"name"`       // 在 workflow volumes 中引用
    MountPath  string `json:"mountPath"`  // 容器内挂载路径
    SubPath    string `json:"subPath,omitempty"`
    ReadOnly   bool   `json:"readOnly,omitempty"`
    VolumeType string `json:"volumeType"` // pvc / configmap / emptydir
    VolumeSpec interface{} `json:"volumeSpec"` // 具体配置
}
```

**验收**：组件声明 EmptyDir → YAML 含 volumes + volumeMounts；下发的 Pod 中挂载点正常。

**估行**：后端 ~60 行 + 前端 ~40 行

---

### C-6：Sub-DAG / 嵌套 Pipeline

**目标**：一个 pipeline 节点引用另一个 pipeline template 作为子步骤，支持递归嵌套。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/transpiler.go` | `buildNodeTemplate()` 判断节点类型为 "sub-dag" 时生成 `TemplateRef` 而非 Container |
| `backend/internal/transpiler/pipeline.go` | `Node` 增加 `Type string`（container / sub-dag）；`SubDAGRef` 字段 |
| `Frontend/src/components/pipeline/ComponentPalette.tsx` | 已保存的 template 作为可拖入的 Sub-DAG 组件 |

**验收**：拖入 Sub-DAG 节点 → transpile → YAML 使用 `templateRef` 引用子 workflow。

**估行**：后端 ~80 行 + 前端 ~100 行

---

### C-7：条件分支（Conditional）

**目标**：根据前置 step 的输出或状态决定是否执行后续 step（如：失败才执行告警 step）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Edge` 增加 `Condition string`（如 `"{{steps.step1.outputs.parameters.status}} == failed"`） |
| `backend/internal/transpiler/transpiler.go` | `buildDAGTemplate()` 处理 Edge.Condition → 追加 `when` 字段 |
| `Frontend/src/components/pipeline/NodeConfigPanel.tsx` | 边配置面板增加"条件表达式"输入 |

**验收**：连线带条件 → YAML 含 `when: "{{steps...}} == failed"`。

**估行**：后端 ~40 行 + 前端 ~60 行

---

### C-8：循环 / Foreach

**目标**：节点配置为"对输入列表的每个元素执行一次"，类似 Argo `withItems`。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Node` 增加 `Foreach *ForeachConfig` |
| `backend/internal/transpiler/transpiler.go` | `buildDAGTemplate()` 处理 Foreach → 生成 `withItems` |
| `Frontend/src/components/pipeline/NodeConfigPanel.tsx` | 节点配置增加"循环"开关 + 数据源字段 |

```go
type ForeachConfig struct {
    ItemsFrom string `json:"itemsFrom"` // 来源：前置节点输出参数名
    ItemVar   string `json:"itemVar"`   // 循环内变量名，如 "item"
}
```

**验收**：设 Foreach → YAML 含 `withItems: {{tasks...}}` + `{{item}}` 参数传递。

**估行**：后端 ~50 行 + 前端 ~60 行

---

### C-9：Step 缓存（跳过不变输入）

**目标**：如果 step 的输入参数和上次运行时一致，自动跳过复用上次结果。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Node` 增加 `Cache *CacheConfig` |
| `backend/internal/transpiler/transpiler.go` | 根据组件版本 + 参数计算 hash → 写入 Argo `podSpecPatch` 或 annotation |

**策略**：使用 Argo 的 `onExit` + 手动判断，或通过 input artifact hash + 自定义控制器。简化方案：不做精确缓存，只做**手动缓存标记**——用户在节点上勾选"启用缓存"，transpiler 生成唯一 cache key 写入 label。

**验收**：勾选缓存后 YAML 含 cache label；相同参数二次运行时 Argo 判断命中 → 跳过执行。

**估行**：后端 ~60 行 + 前端 ~30 行

---

### C-10：Init Container / Sidecar

**目标**：组件配置 init container（前置初始化）和 sidecar（伴生容器，如日志采集）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Component` 增加 `InitContainers []SidecarConfig`, `Sidecars []SidecarConfig` |
| `backend/internal/transpiler/transpiler.go` | buildNodeTemplate 处理 init 容器和 sidecar |

**验收**：配置 init 容器 → YAML 含 `initContainers`；配置 sidecar → 主容器运行期间 sidecar 同时运行。

**估行**：后端 ~50 行

---

### C-11：Argo Workflow Template / CronWorkflow / WorkflowTemplate 生成模式

**目标**：transpiler 可选生成三种 Argo CRD：一次性 Workflow、可复用 WorkflowTemplate、定时 CronWorkflow。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/transpiler.go` | `Options` 增加 `Kind string`（workflow / workflow-template / cron-workflow）；`CronSchedule string` |
| `backend/internal/transpiler/transpiler.go` | 根据 Kind 生成不同的 TypeMeta + Spec |

**验收**：Kind=workflow-template → YAML kind 为 WorkflowTemplate；Kind=cron-workflow → YAML 含 cron schedule。

**估行**：后端 ~80 行

---

### C-12：Transpiler 输入校验

**目标**：transpile 前校验 DAG 合法性（有环？节点不存在？参数类型不匹配？）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/transpiler.go` | 新增 `Validate(p *Pipeline) error` 函数：检测环、孤点、缺失引用 |
| `backend/internal/transpiler/transpiler_test.go` | 测试用例：合法 DAG、有环 DAG、缺参数 DAG |
| `backend/internal/usecase/pipeline/usecase.go` | Deploy 前先 Validate |

**验收**：有环 DAG → 返回 `PIPELINE_INVALID_DAG` + 环路径详情。

**估行**：后端 ~80 行

---

## EPIC-D：执行与部署

**用户故事**：作为数据工程师，我点了"运行"之后，希望能可靠地执行、知道状态、出了问题能控制。
**成功指标**：从点击运行到 Argo 开始执行 ≤ 3 秒；部署失败时提供明确错误原因。

---

### D-1：部署携带 asset_ids

**目标**：deploy API 和 transpiler 支持 asset 参数注入（同原有 P1-F4.1）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `DeployRequest.AssetIDs` |
| `backend/internal/transpiler/transpiler.go` | `Options.AssetIDs` → workflow 参数 |

**验收**：带 asset_ids 部署 → 容器 env `INPUT_ASSET_IDS` 存在；不带 → 兼容。

**估行**：后端 ~50 行

---

### D-2：Deploy Dry-Run

**目标**：部署前点"预览" → 只生成 YAML 不提交到 Argo，让用户确认内容。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /deploy?dry_run=true` → transpile 但不 submit |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署弹窗增加"预览 YAML"按钮 → modal 展示 YAML |

**验收**：dry_run → 返回 YAML 但 Argo 无新 workflow；正常 deploy → Argo 有 WF。

**估行**：后端 ~20 行 + 前端 ~50 行

---

### D-3：部署前参数校验

**目标**：提交 deploy 前后端校验参数完整性（必填项、类型、范围）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `ValidateDeployParams(deployReq)`：遍历 node args → 检查必填、类型 |
| `backend/internal/handlers/pipeline/handler.go` | deploy handler 调用校验 |

**验收**：必填参数未填 → 400 + 指明哪个节点哪个参数缺失；参数类型不匹配 → 400 + 说明。

**估行**：后端 ~60 行

---

### D-4：并发部署限制

**目标**：防止同一用户短时间内重复部署同一 pipeline，加锁或去重。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | Deploy() 中 `SELECT ... FOR UPDATE` 或内存锁（基于 pipelineName + hash(assetIDs)） |
| `backend/internal/usecase/pipeline/usecase.go` | 最近 5 秒内相同 pipeline + 相同参数 → 返回已有 deployment |

**验收**：连点 3 次部署 → 只创建 1 个 workflow（后 2 次返回已有 ID）。

**估行**：后端 ~40 行

---

### D-5：部署队列 + 优先级

**目标**：资源不足时 deploy 请求排队，支持设置优先级（urgent / normal / low）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | 新增 `DeployQueue`：channel + goroutine worker |
| `backend/internal/models/pipeline.go` | `DeployRequest` 增加 `Priority int`（1=urgent, 2=normal, 3=low） |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 优先级选择器 |

**验收**：提交 5 个部署 → 队列中有序执行；高优先级插队。

**估行**：后端 ~120 行 + 前端 ~30 行

---

### D-6：部署历史状态看板

**目标**：部署列表中直观展示每个 deployment 的生命周期状态视图。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 增加时间线视图：Submitted → Running → Success/Failed（带时间戳） |

**验收**：点击 deployment → 展开显示状态时间线。

**估行**：前端 ~80 行

---

### D-7：部署回滚（Redeploy Previous）

**目标**：从部署历史中选择一个之前的成功版本，一键回滚。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /deployments/:id/rollback` → 读取历史 deployment 的 pipeline_json → 重新 deploy |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 历史行增加"回滚"按钮 |

**验收**：点"回滚" → 用该次 deployment 的 pipeline 快照重新创建新 deployment。

**估行**：后端 ~40 行 + 前端 ~40 行

---

### D-8：定时部署（指定未来时间）

**目标**：部署时选择"定时执行"，设定未来某个时间点自动触发。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `DeployRequest` 增加 `ScheduledAt *time.Time` |
| `backend/internal/usecase/pipeline/usecase.go` | 处理定时部署：创建 deployment + 插入 `scheduled_tasks` 表或 cron |
| `backend/migrations/043_scheduled_tasks.sql` | `pipeline_scheduled_tasks` 表 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 日期时间选择器 |

**验收**：设定 1 小时后 → 立即创建 deployment(status=scheduled) → 到达时间后自动提交 Argo。

**估行**：后端 ~80 行 + 前端 ~40 行 + 1 迁移

---

### D-9：部署结果快速诊断

**目标**：部署失败时，页面上直接显示失败原因摘要而非"状态: failed"。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | Deployment 增加 `error_summary TEXT` 字段 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 失败行显示红色摘要 + 详情展开 |

**验收**：Argo 提交失败 → 页面上显示"K8s API 错误: namespace not found"; 容器执行失败 → 显示"exit code 1: OOM killed"。

**估行**：后端 ~30 行 + 前端 ~40 行

---

### D-10：Deployment 批量清理

**目标**：列表支持批量选中旧 deployment → 批量删除（含 K8s workflow 清理）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /deployments/batch-delete` Body: `{ids: [...]}` |
| `backend/internal/usecase/pipeline/usecase.go` | 遍历删除 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 表格 checkbox + "批量删除"按钮 + 确认弹窗 |

**验收**：选中 N 个 → 删除 → 前端列表移除 + K8s workflow 删除。

**估行**：后端 ~30 行 + 前端 ~60 行

---

## EPIC-E：运行监控与可观测性

**用户故事**：作为算法/数据工程师，我每天要看 pipeline 跑得怎么样——成功了没有、多久、为什么失败。
**成功指标**：30 秒内能定位任意一次失败的原因。

---

### E-1：Retry / Stop / Logs API

**目标**：后端提供 workflow 操作 API（同原有 P1-F5.1）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/k8s/workflow_client.go` | `RetryWorkflow`, `StopWorkflow`, `GetWorkflowLogs` |
| `backend/internal/handlers/workflow/handler.go` | handler |
| `backend/routes/routes.go` | 注册 |

**验收**：retry → Argo 重试；stop → 变 Stopped；logs → 返回 log 内容。

**估行**：后端 ~120 行

---

### E-2：前端操作按钮 + 日志面板

**目标**：Workflow 详情页增加操作按钮和日志查看（同原有 P1-F5.2）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/WorkflowDetailPage.tsx` | 操作栏 + 日志面板 |
| `Frontend/src/api/workflowApi.ts` | API 调用 |

**验收**：重试/停止/日志全部可用。

**估行**：前端 ~180 行

---

### E-3：Pipeline 运行看板

**目标**：新增 Dashboard 卡片 / 独立页面，展示 pipeline 全局指标。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelineDashboardPage.tsx` | 新建：统计卡片 + 图表 |
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/stats` |

**指标**：
- 今日运行次数 / 成功数 / 失败数
- 平均运行时长（最近 7 天趋势）
- 活跃 pipeline 数
- 失败率 Top 5 pipeline
- 最常用组件 Top 5

**验收**：页面打开显示真实指标，有数据时图表可见，无数据时 empty state。

**估行**：后端 ~60 行 + 前端 ~200 行

---

### E-4：运行历史趋势图

**目标**：按时间维度看 pipeline 运行趋势（每小时/每天/每周）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/stats/history?range=7d&bucket=1d` → 返回时间序列 |
| `Frontend/src/pages/PipelineDashboardPage.tsx` | 折线图：运行次数 / 成功率 / 平均耗时 |

**验收**：7 天每天有一条数据；hover 显示具体数值。

**估行**：后端 ~60 行 + 前端 ~120 行

---

### E-5：Run 对比

**目标**：从部署列表中勾选 2 个 run → 点"对比" → 并排展示节点状态、耗时、参数差异。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/RunComparisonPage.tsx` | 新建：两个 React Flow 并排 + 差异高亮 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 选中 2 行后显示"对比"按钮 |

**对比内容**：
- 节点状态差异（节点 A 第一次成功第二次失败 → 红色高亮）
- 各节点耗时对比（柱状图）
- 参数差异（diff 展示）

**验收**：选 2 个 deployment → 对比 → 能看到参数不同、状态不同。

**估行**：前端 ~250 行 + 后端 ~30 行（`GET /deployments/compare?ids=a,b`）

---

### E-6：Step 资源使用展示

**目标**：节点详情中展示实际 CPU/Mem 使用量（从 K8s metrics API 获取）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/k8s/workflow_client.go` | `GetPodResourceUsage(name, nodeID)` → K8s metrics API |
| `backend/internal/handlers/workflow/handler.go` | `GET /workflows/:name/nodes/:id/resources` |
| `Frontend/src/pages/WorkflowDetailPage.tsx` | 节点详情面板增加资源使用指示条 |

**验收**：运行中的节点 → 显示实时 CPU/Mem 使用 vs 请求量。

**估行**：后端 ~50 行 + 前端 ~80 行

---

### E-7：Workflow 实时事件流（SSE）

**目标**：Workflow 详情页通过 Server-Sent Events 实时推送状态变更，无需手动刷新。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/workflow/handler.go` | `GET /workflows/:name/events` → SSE endpoint（轮询 Argo + push 变更） |
| `Frontend/src/pages/WorkflowDetailPage.tsx` | EventSource 连接 SSE → 节点状态自动更新 |

**验收**：打开页面 → 节点状态变更无需刷新自动更新。

**估行**：后端 ~60 行 + 前端 ~60 行

---

### E-8：运行 Gantt 图

**目标**：Workflow 详情中切换"时间线"视图 → 每个 step 作为横条，展示耗时和依赖关系。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/WorkflowDetailPage.tsx` | Tab 切换 "DAG 视图" / "Gantt 视图" |
| `Frontend/src/components/pipeline/WorkflowGantt.tsx` | 新建：Gantt 图组件（基于 step.startAt/finishedAt） |

**验收**：切换 Gantt 视图 → 每个 step 一条横条，按时间轴排列，颜色标状态。

**估行**：前端 ~200 行

---

### E-9：节点日志内联预览

**目标**：Workflow 详情中鼠标悬停或点击节点 → 浮层预览最近几条日志。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/WorkflowDetailPage.tsx` | 节点点击 → 浮层展示 tail 20 行日志 + "查看全部"链接 |
| `Frontend/src/api/workflowApi.ts` | `getWorkflowLogs(name, nodeID, tail=20)` |

**验收**：点击节点 → 浮层显示最近日志；关联即可跳转到完整日志。

**估行**：前端 ~80 行 + 后端 ~20 行

---

### E-10：Pipeline SLA 追踪

**目标**：为 pipeline 设定 SLA（期望最大执行时间），超过则标记为 SLA 违规。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/models/pipeline.go` | `PipelineTemplate` 增加 `sla_seconds INT` |
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/sla-violations` |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署列表 SLA 状态列（正常 / 警告 / 违规） |
| `Frontend/src/pages/PipelineDashboardPage.tsx` | SLA 看板卡片 |

**验收**：pipeline 执行超 SLA → 列表显示红色违规标记；看板展示 SLA 达标率。

**估行**：后端 ~50 行 + 前端 ~100 行

---

### E-11：失败模式聚合（类似 P0 Failure Mining 但针对 Pipeline）

**目标**：聚合 pipeline 失败原因（OOM / CrashLoop / Timeout / 参数错误 / 镜像拉取失败），展示分布饼图。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `GetFailureClusters(days)` → 从 deployment 日志/状态聚合 |
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/failure-clusters?days=7` |
| `Frontend/src/pages/PipelineDashboardPage.tsx` | 饼图展示失败模式分布 |

**验收**：7 天内有失败 → 饼图展示 OOM / CrashLoop 等分布。

**估行**：后端 ~60 行 + 前端 ~80 行

---

### E-12：Workflow 搜索/过滤

**目标**：Workflow 列表支持按状态/名称/时间/组件名搜索。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/workflow/handler.go` | `GET /workflows` 支持 `?q=&status=&component=&since=&until=` |
| `Frontend/src/pages/WorkflowListPage.tsx` | 搜索框 + 筛选条件 row |

**验收**：搜索"ocr" → 只显示名称含 ocr 的 workflow；筛选 Failed → 只显示失败。

**估行**：后端 ~40 行 + 前端 ~80 行

---

## EPIC-F：Asset 深度集成

**用户故事**：作为业务运营，我在 pipeline 处理前后都能和资产管理无缝衔接——选资产、看产出、查血缘。
**成功指标**：pipeline 的输入和产出在资产管理页面中可直接追溯到对方。

---

### F-1：前端部署弹窗选 Asset

**目标**：部署弹窗中搜索、选择 asset 作为输入（同原有 P1-F4.2）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/DeployPanel.tsx` | Asset 选择器（搜索 + 列表 + 多选） |
| `Frontend/src/api/pipelineApi.ts` | `deployPipeline` / `deployTemplate` 参数 |

**验收**：搜索 → 选择 → 部署 → asset_ids 传到后端。

**估行**：前端 ~120 行

---

### F-2：Asset Filter Presets

**目标**：将常用的 asset 筛选条件保存为 preset，部署时直接选用预置条件。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/api/pipelineApi.ts` | `GET /api/v1/saved-queries?scope=pipeline` |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 预置条件下拉选择 |

**验收**：保存"近 7 天 ready asset"为 preset → 部署时选中 → 自动注入参数。

**估行**：前端 ~60 行（复用已有 saved-queries）

---

### F-3：产出 Asset 自动注册（容器回调）

**目标**：容器处理完成后调用 `POST /api/v1/assets` 注册新 asset，关联 pipeline deployment。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/deployments/:id/register-output` Body: `{asset_id, name, type, ...}` |
| `docs/pipeline/container-contract.md` | 新建：容器回调协议文档 |

**容器协议**：

```
容器在完成处理后调用平台 API：
POST /api/v1/deployments/{deployment_id}/register-output
{
  "asset_id": "abc123",
  "name": "processed-segment",
  "asset_type": "processed",
  "uri": "gs://bucket/outputs/abc123.bin",
  "metadata": {...}
}
```

**验收**：容器成功调用 → 新 asset 创建 + deployment 记录关联。

**估行**：后端 ~80 行

---

### F-4：Pipeline → Asset 血缘查询

**目标**：asset 详情页展示"来自哪个 pipeline run"；deployment 详情页展示"产出了哪些 asset"。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/deployments/:id/outputs` |
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/assets/:id/pipeline-lineage` |
| `Frontend/src/pages/AssetDetailPage.tsx` | 新增"Pipeline" Tab 或 section |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署详情展示产出 asset 列表 |

**验收**：asset 详情显示"由 pipeline X 的部署 Y 产出"；部署详情显示"产出了 N 个 asset"。

**估行**：后端 ~60 行 + 前端 ~100 行

---

### F-5：asset_events 集成

**目标**：pipeline deploy 和 finish 时写入 asset_events outbox。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | Deploy 成功 → 写 `pipeline_deployed` 事件；完成 → 写 `pipeline_finished` 事件 |
| `backend/schemas/events/` | 新增事件 schema：`pipeline_deployed.v1.json`、`pipeline_finished.v1.json` |

**事件类型**：

```
pipeline_deployed   — pipeline 部署成功
pipeline_finished   — pipeline 执行完成（含状态）
pipeline_step_failed— 某个 step 失败
```

**验收**：部署 pipeline → asset_events 中看到 `pipeline_deployed` 事件。

**估行**：后端 ~60 行 + 2 个 schema 文件

---

### F-6：批量 Asset → Pipeline 绑定

**目标**：在资产列表中选中 N 个 asset → 右键"用 pipeline 处理" → 选 template → 批量创建 deployment（每个 asset 一个）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/AssetsPage.tsx` | 选中行 → "用 pipeline 处理"按钮 |
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/:id/batch-deploy` Body: `{asset_ids: [...]}` |

**验收**：选中 10 个 asset → 选 template → 创建 10 个 deployment（或 1 个 backfill job）。

**估行**：后端 ~50 行 + 前端 ~100 行

---

### F-7：Pipeline 输入校验（Asset 存在性）

**目标**：部署时校验所选 asset 是否存在、lifecycle 状态是否可处理。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `ValidateAssets(assetIDs)` → 批量查询 assets 表 |

**验收**：选不存在的 asset → 部署失败 + "asset X 不存在"；选已归档 asset → 提示"该 asset 已归档"。

**估行**：后端 ~40 行

---

### F-8：Asset 输出预览

**目标**：部署详情中产出的 asset 可预览（如：图片类 asset 显示缩略图、文本类显示前几行）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 产出 asset 列表每项加"预览" |
| `Frontend/src/components/pipeline/AssetPreview.tsx` | 新建：根据 asset type 渲染预览 |

**验收**：产出 asset 为图片 → 显示缩略图；为文本 → 显示前 500 字符。

**估行**：前端 ~120 行

---

### F-9：Asset → Pipeline 反向触发

**目标**：asset 状态变更时自动触发 pipeline（如：asset 变为 ready → 自动执行处理 pipeline）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/:id/asset-triggers` 设定绑定规则 |
| `backend/migrations/044_asset_triggers.sql` | `pipeline_asset_triggers` 表 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | "自动触发"配置区 |

**验收**：设触发规则 → asset 状态变更 → 自动创建 deployment。

**估行**：后端 ~120 行 + 前端 ~80 行 + 1 迁移

---

## EPIC-G：Backfill 批回放

**用户故事**：作为数据工程师，我升级了算法版本，要对历史数据重新跑一遍 pipeline。
**成功指标**：选中 1000 个 asset、选 template、一键启动，30 分钟内跑完。

---

### G-1：Backfill 建表

**目标**：`backfill_jobs` + `backfill_items` 表（同原有 P1-F6.1）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/041_backfill_tables.sql` | 建表（同原有设计） |

**估行**：1 迁移

---

### G-2：Backfill CRUD API

**目标**：创建/列表/详情/暂停/继续/重试 API（同原有 P1-F6.2）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/models/pipeline.go` | `BackfillJob`, `BackfillItem` |
| `backend/internal/repository/pipeline_repository.go` | BackfillRepository |
| `backend/internal/postgres/pipeline_repo.go` | PG impl |
| `backend/internal/usecase/pipeline/usecase.go` | 核心逻辑 |
| `backend/internal/handlers/pipeline/handler.go` | handler |
| `backend/routes/routes.go` | 路由 |

**验收**：创建 → items 生成 → 逐个 deploy → 进度更新 → 暂停/继续/重试失效。

**估行**：后端 ~250 行

---

### G-3：Backfill 前端页面

**目标**：Backfill 列表 + 创建弹窗 + 进度详情（同原有 P1-F6.3）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/BackfillPage.tsx` | 列表 + 创建 + 进度 |
| `Frontend/src/api/pipelineApi.ts` | API 调用 |
| `Frontend/src/App.tsx` | 路由 |

**验收**：创建 → 进度条动态更新 → 可暂停/继续/重试。

**估行**：前端 ~250 行

---

### G-4：Backfill 并发控制

**目标**：限制同时运行的 backfill item 数（默认 5，可配置），避免打爆集群。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `CreateBackfill()` 增加 `concurrency int` 参数；worker goroutine 用 channel/semaphore 控制 |

**验收**：并发数=3 → 同时最多 3 个 workflow 运行。

**估行**：后端 ~40 行

---

### G-5：Backfill 结果摘要

**目标**：Backfill 完成后展示汇总（成功率、平均耗时、失败原因分布）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `GetBackfillSummary(id)` |
| `Frontend/src/pages/BackfillPage.tsx` | 完成后的结果摘要卡片 |

**验收**：回放完成 → 显示"成功 95/100，耗时 12m，失败原因：5 OOM"。

**估行**：后端 ~30 行 + 前端 ~60 行

---

### G-6：Backfill 部分取消

**目标**：运行中的 backfill 可取消特定 item（如某个 asset 一直失败 → 单独取消）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /backfill/:id/items/:item_id/cancel` |
| `backend/internal/usecase/pipeline/usecase.go` | 停止对应 workflow + 更新 item status=cancelled |
| `Frontend/src/pages/BackfillPage.tsx` | 每行 item 的取消按钮 |

**验收**：取消单个 item → 对应 workflow 被停止，job 进度统计更新。

**估行**：后端 ~30 行 + 前端 ~40 行

---

### G-7：Backfill 定时执行

**目标**：设置 backfill 在指定时间自动运行（如：每晚凌晨 2 点重跑当天所有失败资产）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/041_backfill_tables.sql` | `backfill_jobs` 增加 `schedule_cron TEXT`, `next_run_at TIMESTAMPTZ` |
| `backend/internal/usecase/pipeline/usecase.go` | 后台 goroutine 定期检查并触发定时 backfill |
| `Frontend/src/pages/BackfillPage.tsx` | 创建时可选"定时执行" |

**验收**：设 cron `0 2 * * *` → 每天凌晨 2 点自动执行该 backfill。

**估行**：后端 ~80 行 + 前端 ~40 行

---

## EPIC-H：调度与触发器

**用户故事**：作为数据工程师，我想让 pipeline 在特定条件发生时自动执行——定时、事件、webhook。
**成功指标**：配置一次触发规则后无需人工介入。

---

### H-1：Cron 定时调度 UI

**目标**：pipeline 详情中配置 cron 表达式 + 时区，创建 CronWorkflow。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `CreateCronWorkflow()`：transpile → 生成 CronWorkflow CRD → 提交 K8s |
| `backend/internal/handlers/pipeline/handler.go` | `POST /pipelines/:id/cron` |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | 新建：cron 输入（支持预设：每小时/每天/每周）+ 时区选择 |

| Cron 预设 | 表达式 |
|-----------|--------|
| 每小时 | `0 * * * *` |
| 每天午夜 | `0 0 * * *` |
| 每工作日早 9 点 | `0 9 * * 1-5` |
| 自定义 | 自由输入表达式 |

**验收**：设 cron → K8s 创建 CronWorkflow → 到达时间自动执行。

**估行**：后端 ~80 行 + 前端 ~120 行

---

### H-2：Webhook 触发

**目标**：为 pipeline 生成唯一 webhook URL，外部系统 POST 即可触发执行，可携带参数。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/045_webhook_triggers.sql` | `pipeline_webhooks` 表（id, pipeline_id, secret, params_template, created_at） |
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/webhooks/:webhook_id/trigger` |
| `backend/internal/usecase/pipeline/usecase.go` | 验证 secret → 解析参数 → 创建 deployment |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | "Webhook" Tab：生成 URL、复制、重置 secret |

**验收**：生成 URL → `curl -X POST <url> -d '{"param":"value"}'` → pipeline 执行。

**估行**：后端 ~100 行 + 前端 ~80 行 + 1 迁移

---

### H-3：Asset 事件触发

**目标**：监听 `asset_events` 中的特定事件类型（如 `algo_finished`），条件匹配则触发 pipeline。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/046_event_triggers.sql` | `pipeline_event_triggers` 表 |
| `backend/internal/usecase/pipeline/usecase.go` | 后台消费 asset_events → 匹配触发器 → 创建 deployment |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | "事件触发" Tab：选事件类型 + 条件 |

**验收**：设"algo_finished → 跑处理 pipeline" → 算法完成 → pipeline 自动启动。

**估行**：后端 ~120 行 + 前端 ~80 行 + 1 迁移

---

### H-4：Pipeline 链式触发（Pipeline Done → Run Next）

**目标**：pipeline A 完成后自动触发 pipeline B（A 的输出作为 B 的输入）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/047_chain_triggers.sql` | `pipeline_chains` 表（source_pipeline_id, target_pipeline_id, params_mapping JSONB） |
| `backend/internal/usecase/pipeline/usecase.go` | Pipeline finish 时检查链式触发器 |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | "链路" Tab：选下游 pipeline + 参数映射 |

**验收**：A 完成 → B 自动开始执行，且 A 的产出 asset_ids 传入 B。

**估行**：后端 ~80 行 + 前端 ~100 行 + 1 迁移

---

### H-5：触发历史日志

**目标**：所有触发事件（cron / webhook / event / chain）记录到触发器历史表，可查看最近触发记录。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/048_trigger_logs.sql` | `trigger_logs` 表 |
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/:id/trigger-logs` |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | "触发历史" Tab |

**验收**：查看 pipeline 详情 → 触发历史 → 显示"2026-05-27 10:00 cron 触发 → 部署成功"。

**估行**：后端 ~50 行 + 前端 ~60 行 + 1 迁移

---

### H-6：触发器启用/禁用

**目标**：每个触发器有独立开关，禁用后不再响应。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /pipelines/:id/triggers/:trigger_id/toggle` |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | 开关 Switch |

**验收**：开关关闭 → 触发事件不再执行 pipeline。

**估行**：后端 ~20 行 + 前端 ~20 行

---

### H-7：手动触发带参数

**目标**：从 pipeline 详情页点"手动触发" → 输入参数 → 立即执行（不经过调度器等待）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelineDetailPage.tsx` | "手动触发"按钮 + 参数弹窗 |
| `Frontend/src/api/pipelineApi.ts` | `triggerPipeline(pipelineId, params)` |

**验收**：点手动触发 → 填参数 → 创建 deployment。

**估行**：前端 ~80 行 + 后端 ~20 行

---

### H-8：触发器统计

**目标**：展示每个触发器的执行次数、成功率、平均响应时间。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/:id/triggers/stats` |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | 统计卡片 |

**验收**：触发器旁显示"已触发 24 次，成功率 95%，平均延迟 3s"。

**估行**：后端 ~30 行 + 前端 ~50 行

---

## EPIC-I：密钥与配置管理

**用户故事**：作为算法工程师，我的镜像需要访问数据库、云存储，不希望把密钥写在 pipeline 参数里。
**成功指标**：配置一次密钥，任意 pipeline 引用，密钥不在任何日志中明文出现。

---

### I-1：Secrets CRUD

**目标**：平台层面管理 K8s Secrets（新增/查看/删除/列表），前端可操作。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/k8s/workflow_client.go` | `CreateSecret()`, `ListSecrets()`, `DeleteSecret()` |
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/secrets`, `GET /api/v1/secrets`, `DELETE /api/v1/secrets/:name` |
| `Frontend/src/pages/SecretsPage.tsx` | 新建：密钥管理页 |
| `Frontend/src/App.tsx` | 路由 `/settings/secrets` |

**注意**：K8s Secret value 一旦创建不可读取（只展示 metadata），前端只做创建和删除。

**验收**：新增 secret → K8s 创建 Secret 资源 → 列表可见 → 删除。

**估行**：后端 ~80 行 + 前端 ~120 行

---

### I-2：Secret 绑定到 Step

**目标**：在节点配置面板中选择 secret → 映射为容器环境变量或 volume 挂载。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/NodeConfigPanel.tsx` | "密钥"区域：选择 secret + 映射为 env |
| `backend/internal/transpiler/pipeline.go` | `Component` 增加 `SecretRefs []SecretRef` |
| `backend/internal/transpiler/transpiler.go` | 生成 envFrom / valueFrom.secretKeyRef |

**验收**：绑定 secret → transpile → YAML 含 `valueFrom.secretKeyRef`。

**估行**：后端 ~40 行 + 前端 ~100 行

---

### I-3：全局 Pipeline 变量

**目标**：定义平台级别的全局变量（如 `GCS_BUCKET`, `DB_HOST`），所有 pipeline 自动注入。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET/PUT /api/v1/pipeline-variables` |
| `backend/internal/postgres/pipeline_repo.go` | `pipeline_variables` 表 |
| `backend/internal/transpiler/transpiler.go` | `Options` 增加 `GlobalVars map[string]string` → 注入每个 step |
| `Frontend/src/pages/SettingsPage.tsx` | 全局变量配置区 |

**验收**：设 `GCS_BUCKET=my-bucket` → 所有新部署的容器中该环境变量存在。

**估行**：后端 ~60 行 + 前端 ~80 行

---

### I-4：环境配置（Dev / Staging / Prod）

**目标**：支持多环境配置，同一 pipeline 在不同环境下使用不同的变量/密钥。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/models/pipeline.go` | `EnvironmentConfig` struct |

| 变量 | Dev 值 | Prod 值 |
|------|--------|---------|
| DB_HOST | localhost:5432 | prod-db.internal:5432 |
| GCS_BUCKET | dev-bucket | prod-bucket |

`Frontend/src/components/pipeline/DeployPanel.tsx`：部署时选环境 → 自动替换变量。

**验收**：Dev/Prod 切换 → 容器中收到的环境变量值不同。

**估行**：后端 ~80 行 + 前端 ~80 行

---

### I-5：ConfigMap 集成

**目标**：支持挂载 K8s ConfigMap 作为配置文件注入容器。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Component` 增加 `ConfigMapRefs []ConfigMapRef` |
| `backend/internal/transpiler/transpiler.go` | 处理 volume + volumeMount（ConfigMap） |

**验收**：声明引用的 ConfigMap → YAML 含 configMap volume mount。

**估行**：后端 ~40 行

---

### I-6：Secret 自动轮转通知

**目标**：密钥快过期时（如 90 天未更新）在 dashboard 显示通知。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | 定时检查 secret 创建时间 → 返回即将过期的列表 |
| `Frontend/src/pages/PipelineDashboardPage.tsx` | 告警卡片"N 个密钥 90 天未更新" |

**验收**：超过阈值的 secret → Dashboard 显示提醒。

**估行**：后端 ~30 行 + 前端 ~30 行

---

### I-7：变量继承与覆盖

**目标**：全局变量 → 环境变量 → pipeline 参数 → 节点参数，层级覆盖。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/transpiler.go` | 合并变量时按优先级：节点参数 > pipeline 参数 > 环境 > 全局 |

**验收**：全局设 `THRESHOLD=0.5` → 节点参数设 `THRESHOLD=0.8` → 容器中 THRESHOLD=0.8。

**估行**：后端 ~30 行

---

## EPIC-J：通知与告警

**用户故事**：作为算法工程师，pipeline 跑完或失败了我想立刻知道，不用打开页面刷。
**成功指标**：从 pipeline 完成到收到通知 ≤ 10 秒。

---

### J-1：Run 完成通知（飞书/Webhook）

**目标**：pipeline 完成时发送飞书消息/通用 webhook，内容包括名称、状态、耗时、节点统计。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | Pipeline finish 回调时触发通知 |
| `backend/pkg/notify/feishu.go` | 飞书消息卡片发送 |
| `backend/pkg/notify/webhook.go` | 通用 webhook |
| `backend/internal/models/pipeline.go` | `PipelineTemplate` 增加 `notify_channels JSONB` |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | 通知配置（部署弹窗底部"完成通知"） |

**飞书消息示例**：

```
✅ Pipeline "数据处理" 执行完成
━━━━━━━━━━━━━━━━━━━━
状态: 成功 ✓
耗时: 3m 12s
节点: 5/5 成功
产出: 3 个 asset
```

**验收**：pipeline 完成 → 飞书消息到达。

**估行**：后端 ~100 行 + 前端 ~60 行

---

### J-2：Step 失败即时告警

**目标**：单个 step 失败时立即发送告警（不等整个 workflow 完成），附带错误日志摘要。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | 监听 Argo workflow node status change → 检测失败 → 立即发通知 |
| `backend/pkg/notify/feishu.go` | 告警消息格式（红色卡片） |

**验收**：step 失败 → 10 秒内收到飞书告警 + 错误摘要。

**估行**：后端 ~60 行

---

### J-3：每日/每周 Pipeline 运行摘要

**目标**：定时发送运行摘要（今日运行 N 次，成功率 X%，平均耗时 Y）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | 定时器 → 聚合统计数据 → 发送摘要 |
| `backend/pkg/notify/feishu.go` | 摘要消息格式 |

**验收**：每天 10:00 收到摘要 → 数据和页面一致。

**估行**：后端 ~60 行

---

### J-4：通知渠道管理

**目标**：用户可配置通知渠道类型和 Webhook URL。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET/PUT /api/v1/notification-channels` |
| `Frontend/src/pages/SettingsPage.tsx` | 通知配置区域：飞书 Webhook URL、Slack Webhook、邮箱 |

**验收**：配置飞书 webhook → 通知发到飞书群；不配置则不发送。

**估行**：后端 ~40 行 + 前端 ~80 行

---

### J-5：告警静默规则

**目标**：设定静默时间段（如凌晨 2-6 点），期间的失败不发送告警。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | 发送前检查静默规则 |
| `Frontend/src/pages/SettingsPage.tsx` | 静默规则配置 |

**验收**：设静默 2:00-6:00 → 期间失败不告警 → 6:01 失败正常告警。

**估行**：后端 ~40 行 + 前端 ~40 行

---

### J-6：通知模板自定义

**目标**：用户自定义通知消息模板（标题、内容、颜色），使用变量占位符。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/pkg/notify/template.go` | Go template 渲染 |
| `Frontend/src/pages/SettingsPage.tsx` | 模板编辑器（标题 + 正文 + 预览） |

**变量**：`{{.PipelineName}}`, `{{.Status}}`, `{{.Duration}}`, `{{.NodeCount}}`, `{{.FailedNodes}}`

**验收**：编辑模板 → 发送后按自定义格式展示。

**估行**：后端 ~50 行 + 前端 ~100 行

---

## EPIC-K：权限与治理

**用户故事**：作为 Admin，我要控制谁能看/改/跑 pipeline，变更要可追溯，关键操作要审批。
**成功指标**：新成员加入团队后 5 分钟完成权限配置。

---

### K-1：Pipeline 级权限

**目标**：每个 pipeline 有 owner + 权限角色（viewer / editor / runner / admin）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/049_pipeline_permissions.sql` | `pipeline_permissions` 表 |
| `backend/internal/middleware/pipeline_auth.go` | 新建：检查当前用户对 pipeline 的操作权限 |
| `backend/internal/handlers/pipeline/handler.go` | 写操作前调 auth middleware |
| `Frontend/src/components/pipeline/PermissionPanel.tsx` | 新建：权限配置 UI |

| 角色 | 查看 | 编辑 | 运行 | 删除 | 改权限 |
|------|------|------|------|------|--------|
| viewer | ✅ | ❌ | ❌ | ❌ | ❌ |
| editor | ✅ | ✅ | ✅ | ❌ | ❌ |
| runner | ✅ | ❌ | ✅ | ❌ | ❌ |
| admin | ✅ | ✅ | ✅ | ✅ | ✅ |

**验收**：viewer → 看到列表但编辑按钮灰色；editor → 可编辑保存；runner → 只能部署。

**估行**：后端 ~150 行 + 前端 ~150 行 + 1 迁移

---

### K-2：审批门禁（Approval Gate）

**目标**：pipeline 中增加"审批"节点，执行到该节点时暂停，等待审批人通过后才继续。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `Node` 增加 `Type: "approval"`, `Approvers []string` |
| `backend/internal/transpiler/transpiler.go` | Approval Node → 生成 `suspend` template |
| `backend/internal/handlers/pipeline/handler.go` | `POST /workflows/:name/approve` → Resume suspend |
| `Frontend/src/components/pipeline/PipelineNode.tsx` | 审批节点渲染（特殊图标 + 审批人列表） |
| `Frontend/src/pages/WorkflowDetailPage.tsx` | 暂停节点显示"等待审批" + 审批人可点"通过/拒绝" |
| `backend/pkg/notify/feishu.go` | 审批通知（@审批人） |

**验收**：pipeline 执行到审批节点 → 暂停 → 飞书通知审批人 → 点通过 → 继续执行。

**估行**：后端 ~120 行 + 前端 ~100 行

---

### K-3：Pipeline 变更审计日志

**目标**：谁在什么时候改了什么（保存、部署、删除、修改权限），全部记录。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/050_pipeline_audit_log.sql` | `pipeline_audit_log` 表 |
| `backend/internal/usecase/pipeline/usecase.go` | 所有写操作记录 |
| `Frontend/src/pages/PipelineAuditPage.tsx` | 新建：审计日志页 |

**记录内容**：
- `actor` — 谁
- `action` — 操作类型（save / deploy / delete / update_permissions）
- `target` — 操作对象（pipeline id / deployment id）
- `detail` — 变更详情 JSON（diff / 参数）
- `created_at` — 时间

**验收**：每次保存 → 审计日志多一条记录；打开审计页 → 可按 pipeline 筛选。

**估行**：后端 ~80 行 + 前端 ~150 行 + 1 迁移

---

### K-4：部署窗口限制

**目标**：限制生产环境 pipeline 只能在特定时间段部署（如：工作日 9:00-18:00）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | Deploy 前检查部署窗口（配置在 pipeline 级别） |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 窗口外部署时提示"当前不在部署窗口内" |

**验收**：周六尝试部署 → 被拒绝 + 提示原因。

**估行**：后端 ~40 行 + 前端 ~30 行

---

### K-5：Pipeline 冻结

**目标**：Admin 可冻结 pipeline（冻结期间不允许任何修改和部署），用于发布窗口。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/:id/freeze`, `POST /api/v1/pipelines/:id/unfreeze` |
| `Frontend/src/pages/PipelineDetailPage.tsx` | 冻结状态 banner + 冻结操作按钮 |

**验收**：冻结 → 编辑按钮灰色 + 部署按钮灰色 + 页面顶部横幅"该 pipeline 已冻结"。

**估行**：后端 ~30 行 + 前端 ~50 行

---

### K-6：镜像安全扫描集成

**目标**：组件注册时自动扫描镜像漏洞（集成 Trivy / Grype），高危漏洞禁止注册或展示警告。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `ScanImage(image)`: 调用 Trivy CLI / API |
| `backend/migrations/040_component_tables.sql` | `pipeline_components` 增加 `scan_status TEXT`, `scan_summary JSONB` |
| `Frontend/src/pages/ComponentDetailPage.tsx` | "安全" Tab 展示扫描结果 |

**验收**：注册含高危漏洞的镜像 → 接口拒绝 / 前端展示红色"高危"标识。

**估行**：后端 ~80 行 + 前端 ~60 行

---

### K-7：组件审批

**目标**：新组件注册后需要 Admin 审批才能使用（避免引入不安全的镜像）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /components/:id/approve`, `POST /components/:id/reject` |
| `backend/internal/repository/pipeline_repository.go` | 组件增加 `approval_status` |
| `Frontend/src/components/pipeline/ComponentManager.tsx` | Admin 看到"待审批" Tab + 审批按钮 |

**验收**：注册组件 → 状态为 pending → Admin 审批 → 变为 active → 可在画布使用。

**估行**：后端 ~50 行 + 前端 ~80 行

---

### K-8：资源配额管理

**目标**：限制每个用户/团队的最大并发 pipeline 数和总资源使用量。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/051_resource_quotas.sql` | `resource_quotas` 表 |
| `backend/internal/usecase/pipeline/usecase.go` | Deploy 前检查配额 |
| `Frontend/src/pages/AdminQuotaPage.tsx` | 新建：Admin 配额管理页 |

**配额维度**：
- 最大并发 workflow 数
- 最大 CPU / Memory 总量
- 每月最大运行次数
- 最大 pipeline 数

**验收**：超配额 deploy → 返回 429 + "已达到并发限制（5/5）"。

**估行**：后端 ~100 行 + 前端 ~120 行 + 1 迁移

---

### K-9：操作确认双签（Break-Glass）

**目标**：高危操作（删除 pipeline、清空部署记录、冻结解冻）需要二次确认 + 输入原因。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/ConfirmDialog.tsx` | 新建：高危操作确认弹窗（操作名 + 原因输入 + 输入"确认"文本） |

**验收**：点"删除" → 弹窗 → 输入原因 + 输入"确认" → 按钮才可用。

**估行**：前端 ~60 行

---

## EPIC-L：Template 管理增强

**用户故事**：作为数据工程师，我积累了很多好用的 pipeline template，希望能管理版本、做 diff、分享给团队。
**成功指标**：找到要用的 template ≤ 10 秒；误更新可回滚。

---

### L-1：Template 版本化

**目标**：保存时版本递增，部署引用固定版本（同原有 P1-F7.1）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/042_template_version.sql` | 增加 version 列 |
| `backend/internal/usecase/pipeline/usecase.go` | 版本递增 |
| `Frontend/src/components/pipeline/ComponentManager.tsx` | 显示版本号 |

**验收**：同 name 多次保存 → version 递增。

**估行**：后端 ~50 行 + 前端 ~30 行 + 1 迁移

---

### L-2：Template Fork / Copy

**目标**：从已有 template 创建副本（fork），继承所有配置，不关联原 template。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/:id/fork` Body: `{name: "新模板"}` |
| `Frontend/src/components/pipeline/ComponentManager.tsx` | 列表行"复制"按钮 |

**验收**：fork → 新 template 内容完全相同但 id 不同，修改原模板不影响副本。

**估行**：后端 ~20 行 + 前端 ~20 行

---

### L-3：Template Diff

**目标**：对比两个版本/两个 template 的差异（节点、参数、连接关系差异）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /templates/:id/diff?old=v1&new=v2` |
| `backend/internal/usecase/pipeline/usecase.go` | 计算 DAG diff（节点增删、参数变化、边变化） |
| `Frontend/src/pages/TemplateDiffPage.tsx` | 新建：并排展示 + 差异高亮 |

**差异展示**：
- 新增节点 → 绿色
- 删除节点 → 红色
- 参数变化 → 黄色 + 旧值/新值

**验收**：选 v1 和 v2 → 展示差异清单："新增节点 B，删除节点 C，A 的参数 threshold 0.5→0.8"。

**估行**：后端 ~80 行 + 前端 ~200 行

---

### L-4：Template 回滚

**目标**：从版本历史中选择任一版本 → 一键回滚到该版本。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/:id/rollback?version=1` |
| `Frontend/src/pages/PipelinePage.tsx` | 版本历史列表 → "回滚到此版本"按钮 |

**验收**：当前 v5，回滚到 v2 → pipeline 内容变为 v2 状态，版本递增为 v6。

**估行**：后端 ~30 行 + 前端 ~40 行

---

### L-5：Template 变量替换

**目标**：template 中声明变量占位符 `${INPUT_BUCKET}`，加载时用户输入实际值。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/NodeConfigPanel.tsx` | 加载未填变量的 template → 弹出变量填写窗口 |
| `backend/internal/transpiler/transpiler.go` | `Transpile()` 支持变量替换 |

**验收**：template 含 `${BUCKET}` → 加载时弹窗要求输入 → 填写后部署 → 容器收到正确值。

**估行**：后端 ~40 行 + 前端 ~100 行

---

### L-6：Template 验证

**目标**：手动触发 template 校验（检查 DAG 合法性 + 参数完整性 + 组件是否存在）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/:id/validate` |
| `Frontend/src/pages/PipelinePage.tsx` | "验证"按钮 |

**验收**：合法 → "验证通过 ✓"；不合法 → "错误: 节点 A 和 B 之间存在环"。

**估行**：后端 ~20 行 + 前端 ~30 行（复用 C-12）

---

### L-7：Template 使用分析

**目标**：查看 template 被部署次数、最近运行时间、平均耗时、成功率。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /templates/:id/analytics` |
| `Frontend/src/pages/TemplateDetailPage.tsx` | "分析" Tab |

**验收**：template 详情 → 显示"部署 24 次，成功率 91%，平均耗时 3m12s"。

**估行**：后端 ~40 行 + 前端 ~80 行

---

### L-8：从 Deployment 保存为 Template

**目标**：成功运行后 → 点"保存为 template" → 将此次运行的 pipeline_json 保存为新 template。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /deployments/:id/save-template` |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 成功 deployment 行 →"保存为模板"按钮 |

**验收**：运行完成 → 保存 → template 列表中新增。

**估行**：后端 ~20 行 + 前端 ~30 行

---

## EPIC-M：API 与开发者工具

**用户故事**：作为 DevOps / SRE，我想用 API 或 CLI 管理 pipeline，集成到现有 CI/CD 流程。
**成功指标**：一条 curl 命令可以完成 pipeline 部署；CI 脚本可以完整编排 pipeline。

---

### M-1：Pipeline YAML/JSON 导出

**目标**：从 UI 或 API 导出 pipeline 为 YAML/JSON 文件，包含完整的 DAG 定义。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /templates/:id/export?format=yaml|json` |
| `Frontend/src/pages/PipelinePage.tsx` | 工具栏"导出"按钮 → 下载文件 |

**验收**：导出 → 下载 `.yaml` 文件 → 内容包含完整 pipeline 定义。

**估行**：后端 ~30 行 + 前端 ~40 行

---

### M-2：Pipeline YAML/JSON 导入

**目标**：上传 YAML/JSON 文件 → 解析生成 pipeline，加载到画布。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/import` Body: 文件内容 |
| `Frontend/src/pages/PipelinePage.tsx` | 工具栏"导入"按钮 → 文件选择器 |

**验收**：导出 → 导入 → 画布内容完全一致。

**估行**：后端 ~40 行 + 前端 ~50 行

---

### M-3：Pipeline as Code（Git 同步）

**目标**：关联 Git 仓库，push pipeline YAML 到指定分支 → 自动同步到平台。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/git-webhook` |
| `backend/internal/usecase/pipeline/usecase.go` | 解析 webhook payload → clone repo → 读取 pipeline YAML → upsert template |
| `Frontend/src/pages/PipelineSettingsPage.tsx` | "Git 同步"配置区 |

**验收**：Git push → webhook 到达 → pipeline 自动更新。

**估行**：后端 ~150 行 + 前端 ~60 行

---

### M-4：Pipeline API Token

**目标**：生成 pipeline 专用的 API token（scope 限制为 pipeline 操作），用于 CI 集成。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipeline-tokens` |
| `Frontend/src/pages/SettingsPage.tsx` | Token 管理（生成/撤销） |

**验收**：生成 token → `curl -H "Authorization: Bearer <token>"` → 可调 pipeline API。

**估行**：后端 ~50 行 + 前端 ~60 行

---

### M-5：OpenAPI 契约同步

**目标**：所有 pipeline API 端点写入 `api/openapi.yaml`，保证契约完整性。

**文件**：

| 文件 | 修改 |
|------|------|
| `api/openapi.yaml` | 新增 pipeline / workflow / component / backfill / secret 等路径 |

**验收**：`openapi.yaml` 包含所有 pipeline 相关端点。

**估行**：~200 行 YAML

---

### M-6：CLI 工具

**目标**：提供 `databrew-pipeline` CLI（Go 单二进制），支持 `deploy`, `list`, `logs`, `retry`。

**文件**：

| 文件 | 修改 |
|------|------|
| `cmd/databrew-pipeline/main.go` | 新建：CLI 入口（cobra） |
| `cmd/databrew-pipeline/deploy.go` | `deploy --template <id> --param key=val` |
| `cmd/databrew-pipeline/list.go` | `list [--status running|failed]` |
| `cmd/databrew-pipeline/logs.go` | `logs <workflow-name> [--node step-1]` |
| `cmd/databrew-pipeline/retry.go` | `retry <workflow-name>` |

**验收**：`databrew-pipeline deploy --template abc` → deployment 创建成功。

**估行**：Go ~400 行

---

### M-7：Pipeline Webhook Outgoing

**目标**：pipeline 完成/失败时通过 webhook 通知外部系统（如：通知下游 CI 或训练平台）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | Pipeline finish 回调 → POST 到配置的 webhook URL |
| `Frontend/src/components/pipeline/SchedulePanel.tsx` | "Webhook 通知"配置（URL + secret + 事件类型） |

**验收**：配置 webhook → pipeline 完成 → 外部系统收到 POST 请求。

**估行**：后端 ~50 行 + 前端 ~40 行

---

### M-8：Pipeline 健康 Check API

**目标**：`GET /api/v1/pipelines/health` → 返回 pipeline 子系统状态（Argo 连接、PG 连接、最近成功 deploy）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/health` |

**验收**：Argo 正常 → `{"argo": "ok", "postgres": "ok"}`；Argo 断开 → `{"argo": "down"}`。

**估行**：后端 ~30 行

---

## EPIC-N：协作与市场

**用户故事**：作为数据工程师，我希望找到别人做好的 pipeline，复用而不是从零搭。
**成功指标**：新场景 50% 可以从已有 template fork 开始。

---

### N-1：Template Marketplace

**目标**：平台内置一个 template 市场页，展示所有公开 template，支持分类浏览。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/TemplateMarketplace.tsx` | 新建：市场首页（分类 + 搜索 + 推荐） |
| `Frontend/src/App.tsx` | 路由 `/templates/marketplace` |
| `Frontend/src/components/AppLayout.tsx` | 菜单项"模版市场" |

**市场页布局**：分类导航栏 | 卡片网格（名称、描述、缩略图、使用次数、评分）

**验收**：打开市场 → 看到公开 template → 点"使用"→ Fork 到我的 pipeline。

**估行**：前端 ~200 行

---

### N-2：Pipeline 收藏/点赞

**目标**：用户收藏常用的 pipeline/template，仪表盘展示收藏列表。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/052_favorites.sql` | `user_favorites` 表 |
| `backend/internal/handlers/pipeline/handler.go` | `POST /favorites`, `GET /favorites`, `DELETE /favorites/:id` |
| `Frontend/src/components/pipeline/PipelineCard.tsx` | 星标按钮 |

**验收**：点星标 → 收藏 → "我的收藏"列表出现。

**估行**：后端 ~40 行 + 前端 ~60 行 + 1 迁移

---

### N-3：Pipeline 标签/分类

**目标**：用户自定义标签对 pipeline 分类，支持多标签、按标签筛选。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/PipelineInfoPanel.tsx` | 标签编辑（输入 + 自动补全） |
| `backend/internal/handlers/pipeline/handler.go` | `GET /templates?tag=` 筛选 |

**验收**：打标签"CV" → 搜索 tag=CV → 显示所有 CV 相关 pipeline。

**估行**：后端 ~20 行 + 前端 ~50 行

---

### N-4：Template 评分与评论

**目标**：用户评分（1-5）和评论 template，帮助其他人选择。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/053_template_reviews.sql` | `template_reviews` 表 |
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/:id/reviews`, `GET /templates/:id/reviews` |
| `Frontend/src/pages/TemplateDetailPage.tsx` | "评价" Tab：星标 + 评论输入 + 评论列表 |

**验收**：评分 → 平均分更新 → 评论出现在列表。

**估行**：后端 ~60 行 + 前端 ~100 行 + 1 迁移

---

### N-5：分享 Pipeline 链接

**目标**：生成分享链接，带权限的协作成员点击即可查看/编辑 pipeline。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/:id/share` → 生成分享 token |
| `Frontend/src/components/pipeline/PipelineInfoPanel.tsx` | "分享"按钮 → 弹窗显示链接 + 复制 + 权限选择 |

**验收**：生成链接 → 其他用户打开 → 根据权限可查看/编辑。

**估行**：后端 ~40 行 + 前端 ~60 行

---

### N-6：团队空间

**目标**：pipeline 按团队组织（team workspace），成员自动继承权限。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/054_team_spaces.sql` | `team_spaces`, `team_members`, `space_pipelines` 表 |
| `backend/internal/handlers/pipeline/handler.go` | 空间 CRUD + 成员管理 |
| `Frontend/src/components/pipeline/TeamSpacePanel.tsx` | 新建：空间列表 + 成员管理 |

**验收**：创建空间 → 邀请成员 → 空间内 pipeline 成员可见。

**估行**：后端 ~150 行 + 前端 ~150 行 + 1 迁移

---

### N-7：最近使用/常用 Pipeline

**目标**：用户主页展示最近使用和常用的 pipeline，一键快速部署。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/recent`, `GET /api/v1/pipelines/frequently-used` |
| `Frontend/src/pages/DashboardPage.tsx` | "常用流水线"卡片区域 |

**验收**：部署 3 次 pipeline A → Dashboard 显示 A 在"常用"列表。

**估行**：后端 ~30 行 + 前端 ~60 行

---

### N-8：Pipeline 模板入门向导

**目标**：首次使用 pipeline 时，弹出入门向导步骤指引（3 步：选组件→连线上→点运行）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/OnboardingWizard.tsx` | 新建：分步指引浮层 |
| `Frontend/src/pages/PipelinePage.tsx` | 首次打开 → 显示向导 |

**验收**：首次进入 → 3 步指引 → "完成"后不再显示。

**估行**：前端 ~150 行

---

## EPIC-O：UX 体验打磨

**用户故事**：我不是高频用户，但每次打开页面不希望看到白屏、报错、空荡荡的列表。
**成功指标**：每次页面加载 ≤ 2 秒；任何状态下页面都有用（有数据、empty state、error state）。

---

### O-1：Empty State 插画

**目标**：空列表/空画布展示友好空状态，而不是空白区域。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/EmptyCanvasState.tsx` | 新建：画布空状态（插画 + "从左侧组件库拖拽组件到画布"） |
| `Frontend/src/components/pipeline/EmptyListState.tsx` | 新建：列表空状态 |
| `Frontend/src/pages/PipelinePage.tsx` | 无节点时展示 EmptyCanvasState |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 无部署记录时展示 EmptyListState |

**场景**：
- 画布无节点 → "拖拽组件开始编排" + 示例截图
- 部署列表空 → "还没有运行记录，点击上方按钮开始运行"
- 组件库空 → "还没有注册组件，点击管理按钮添加"
- backfill 空 → "还没有回放记录"

**验收**：每种空状态都有对应的引导文案和操作入口。

**估行**：前端 ~150 行

---

### O-2：Loading Skeleton

**目标**：加载中展示骨架屏，而非全屏 Spin。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/SkeletonCard.tsx` | 新建：骨架屏卡片组件 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | Loading 时显示骨架 |
| `Frontend/src/pages/WorkflowDetailPage.tsx` | Loading 时显示骨架 |
| `Frontend/src/pages/WorkflowListPage.tsx` | Loading 时显示骨架 |

**验收**：页面加载 → 先出现灰色骨架 → 内容加载后平滑替换。

**估行**：前端 ~120 行

---

### O-3：Error Boundary + 优雅降级

**目标**：每个独立区域（画布、组件面板、部署面板）有自己的 ErrorBoundary，单个区域崩溃不影响其他区域。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/CanvasErrorBoundary.tsx` | 新建：画布区域的 ErrorBoundary |
| `Frontend/src/components/pipeline/PanelErrorBoundary.tsx` | 新建：面板区域的 ErrorBoundary |
| `Frontend/src/pages/PipelinePage.tsx` | 各区域包裹 ErrorBoundary |

**错误展示**：区域显示"该区域加载失败" + "重试"按钮，不影响整页。

**验收**：手动模拟组件面板崩溃 → 画布和部署面板仍然可用。

**估行**：前端 ~80 行

---

### O-4：Pipeline 列表搜索/筛选/排序

**目标**：template 列表和部署列表支持搜索、多条件筛选（状态/时间/标签）和排序。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/ComponentManager.tsx` | 搜索框 + 状态筛选 + 时间范围 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 同上 |
| `backend/internal/handlers/pipeline/handler.go` | `GET /templates?q=&status=&sort=` |

**验收**：搜索"ocr" → 只显示含 ocr 的行；筛选"failed" → 只看失败。

**估行**：后端 ~30 行 + 前端 ~100 行

---

### O-5：部署列表批量操作

**目标**：列表多选 → 批量删除、批量重试失败。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/DeployPanel.tsx` | checkbox + 顶部批量操作栏 |

**验收**：选 N 行 → "批量删除" → 确认 → 全部删除。

**估行**：前端 ~60 行

---

### O-6：确认弹窗统一化

**目标**：所有危险操作（删除/清空/停止/回滚）使用统一的确认弹窗，显示操作名 + 影响范围 + 需输入确认文字。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/pipeline/ConfirmDialog.tsx` | 新建：统一确认弹窗组件 |
| `Frontend/src/pages/PipelinePage.tsx` | 替换现有的 window.confirm |

**验收**：点"删除" → 弹窗显示"确定要删除 pipeline X？该操作不可恢复" + 输入"确认"。

**估行**：前端 ~60 行

---

### O-7：Pipeline 加载历史（最近 10 个）

**目标**：画布工具栏显示最近打开/编辑过的 pipeline 列表，一键切换。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/PipelinePage.tsx` | 保存最近编辑记录到 localStorage，工具栏下拉展示 |

**验收**：编辑 pipeline A → B → 下拉显示 A、B → 选 A → 加载 A。

**估行**：前端 ~50 行

---

### O-8：国际化准备（i18n 抽象）

**目标**：前端字符串抽取为 i18n 资源文件（中文 + 英文），不要求立刻翻译但做好结构。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/i18n/zh-CN/pipeline.ts` | 新建：pipeline 模块中文文案 |
| `Frontend/src/i18n/en-US/pipeline.ts` | 新建：pipeline 模块英文文案 |
| `Frontend/src/components/pipeline/*.tsx` | 替换硬编码中文 → `t('key')` |

**验收**：切换语言 → pipeline 页显示对应语言。

**估行**：前端 ~200 行（体力活）

---

## EPIC-P：高级功能

**用户故事**：作为高阶用户，我希望平台能帮我自动生成 pipeline、预估成本、和现有 CI 打通。
**成功指标**：NL 描述可生成基础 pipeline；查看 pipeline 前就知道要花多少钱。

---

### P-1：AI 辅助 Pipeline 生成

**目标**：输入自然语言描述 → AI 自动生成 pipeline 草图（节点+连线），用户手动微调。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/ai-generate` Body: `{prompt: "用 OCR 处理图片然后上传到 GCS"}` |
| `backend/internal/usecase/pipeline/usecase.go` | 调用 LLM API → 解析响应 → 生成 Pipeline struct |
| `Frontend/src/components/pipeline/AIGenerator.tsx` | 新建：AI 生成对话框（输入框 + 生成按钮 + 预览） |

**验收**：输入"用模型 A 处理后通知飞书" → 生成含 2 个节点的 pipeline（处理节点 + 通知节点）。

**估行**：后端 ~100 行 + 前端 ~120 行

---

### P-2：Pipeline 执行模拟（Dry Run on Sample）

**目标**：用样本数据模拟运行 pipeline，不提交 Argo，预览每个 step 的输入输出。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `Simulate(pipeline, sampleInput)` → 本地模拟执行 |
| `Frontend/src/pages/PipelinePage.tsx` | "模拟"按钮 + 结果展示 |

**模拟模式**：
1. 不调 K8s，纯本地遍历 DAG
2. 每个 step 输出 mock 结果（或空）
3. 展示执行顺序和参数传递

**验收**：点"模拟" → 展示执行顺序 A→B→C，每步的输入输出参数。

**估行**：后端 ~80 行 + 前端 ~100 行

---

### P-3：Pipeline 成本估算

**目标**：部署前展示预估成本（基于节点资源 × 预估耗时 × 单价）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `EstimateCost(pipeline)` → 计算资源需求 × 单价 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署弹窗中展示"预估成本: ¥0.35" |

**计算逻辑**：求和（每节点 CPU 核数 × 预估分钟 × CPU 单价 + 每节点 Mem GB × 分钟 × Mem 单价 + GPU 数量 × 分钟 × GPU 单价）

**验收**：设资源需求 → 弹窗显示"预估成本 ¥0.35/次"。

**估行**：后端 ~40 行 + 前端 ~40 行

---

### P-4：Image Builder 集成（Code → Image → Component）

**目标**：用户在平台提交代码（Git 仓库或 zip）→ 自动构建 Docker 镜像 → 注册为组件。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `BuildImage(gitURL, dockerfile, imageName)` → 调用 Cloud Build / Kaniko |
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/components:build` |
| `Frontend/src/pages/ComponentBuilderPage.tsx` | 新建：构建页面（Git URL + Dockerfile + 镜像名） |

**验收**：提交 Git URL → Cloud Build 运行 → 完成后自动注册组件。

**估行**：后端 ~120 行 + 前端 ~150 行

---

### P-5：Git Webhook（Push → Build → Deploy）

**目标**：Git push 触发：拉取代码 → 构建镜像 → 更新组件 → 部署 pipeline。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/git-actions-webhook` |
| `backend/internal/usecase/pipeline/usecase.go` | 完整 CI/CD 链路 |

**验收**：Git push → pipeline 自动更新部署。

**估行**：后端 ~100 行

---

### P-6：Pipeline 分析（最常用组件/平均耗时/瓶颈检测）

**目标**：分析所有 pipeline 运行数据，展示最常用组件 Top 10、平均各 step 耗时、检测瓶颈。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/analytics` |
| `Frontend/src/pages/PipelineAnalyticsPage.tsx` | 新建：分析页（图表 + 表格） |

**分析内容**：
- 各 step 平均耗时（发现瓶颈）
- 组件使用频率排名
- 失败率最高的组件
- 资源使用分布

**验收**：打开分析页 → 图表展示真实运行数据。

**估行**：后端 ~80 行 + 前端 ~200 行

---

### P-7：Pipeline 推荐

**目标**：基于用户常用组件和场景，推荐可能需要的 pipeline template。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `Recommend(userID)` → 基于使用历史和组件关联推荐 |
| `Frontend/src/pages/PipelineDashboardPage.tsx` | "为你推荐"卡片区域 |

**验收**：用户常用 OCR 组件 → 推荐包含 OCR 或相关组件的 template。

**估行**：后端 ~60 行 + 前端 ~60 行

---

### P-8：Pipeline 依赖图 / 影响分析

**目标**：修改组件/模板时展示"哪些 pipeline 使用了此组件"，评估变更影响。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/components/:id/impact-analysis` |
| `Frontend/src/pages/ComponentDetailPage.tsx` | "影响分析" Tab |

**验收**：查看组件 → "影响分析" Tab → 展示"被 3 个 pipeline 引用，涉及 5 次部署"。

**估行**：后端 ~40 行 + 前端 ~80 行

---

### P-9：Pipeline 归档/冷存储

**目标**：长期不用的 pipeline 自动归档（从活跃列表隐藏但可恢复），减少列表噪音。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /templates/:id/archive`, `POST /templates/:id/unarchive` |
| `Frontend/src/components/pipeline/ComponentManager.tsx` | 归档/取消归档按钮 + "显示归档"开关 |

**验收**：归档 → 列表隐藏 → 开"显示归档" → 出现。

**估行**：后端 ~20 行 + 前端 ~40 行

---

### P-10：Pipeline Run 快照对比（参数 vs 结果）

**目标**：两次 run 之间对比：参数 diff + 产出物 diff + 耗时 diff + 资源使用 diff。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/RunComparisonPage.tsx` | 增强：增加参数 diff 和产出物对比 |
| `backend/internal/handlers/pipeline/handler.go` | `GET /deployments/compare?ids=a,b` 增强 |

**验收**：对比 → 展示"参数 A: 0.5→0.8，耗时: 3m→5m，产出: 3 个 vs 5 个 asset"。

**估行**：前端 ~120 行 + 后端 ~40 行

---

### P-11：Pipeline 模板预设场景

**目标**：平台内置 5-10 个预置模板（OCR 处理、视频转码、数据 ETL、模型推理等），开箱即用。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/055_seed_templates.sql` | 种子数据插入预设模板 |
| `Frontend/src/pages/TemplateMarketplace.tsx` | "官方模板" Tab |

**预设模板列表**：
1. 图片批量处理（下载 → 处理 → 上传）
2. 视频抽帧分析（下载 → 抽帧 → 算法分析 → 结果入库）
3. 数据 ETL（提取 → 转换 → 加载到 BQ）
4. 模型推理管线（前处理 → 推理 → 后处理 → 结果回写）
5. 日报生成（数据查询 → 分析 → 图表生成 → 飞书推送）

**验收**：新用户 → 市场页看到官方模板 → 点"使用" → 5 分钟搭建完成。

**估行**：~10 个预设 JSON 文件 + 1 迁移 + 前端 ~50 行标记

---

---

## EPIC-Q：资产平台核心能力增强

**用户故事**：作为业务运营/算法工程师，我每天管理 35 万+ 资产，需要批量操作、质量管控、生命周期自动化、数据可视化。
**成功指标**：日常资产管理操作不需要写 SQL 或脚本；新资产从注册到可查询 ≤ 3 秒。

---

### Q-1：资产详情页重构（Tab 化）

**目标**：资产详情页从单页长滚动改为 Tab 结构（基本信息 / 算法 / 事件 / 血缘 / 交付 / Pipeline / 质量），每 Tab 独立加载。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/AssetDetailPage.tsx` | 重构为 Tab 布局，每个 Tab 懒加载 |
| `Frontend/src/components/asset/AssetInfoTab.tsx` | 新建：基本信息（当前详情内容） |
| `Frontend/src/components/asset/AssetAlgoTab.tsx` | 新建：算法状态列表 |
| `Frontend/src/components/asset/AssetLineageTab.tsx` | 新建：血缘 DAG 图 |
| `Frontend/src/components/asset/AssetPipelineTab.tsx` | 新建：关联的 pipeline run |
| `Frontend/src/components/asset/AssetEventsTab.tsx` | 新建：事件时间线（从现有迁移） |
| `Frontend/src/components/asset/AssetDeliveriesTab.tsx` | 新建：交付历史 |
| `Frontend/src/components/asset/AssetQualityTab.tsx` | 新建：质量评分看板 |

**验收**：打开资产详情 → Tab 栏 7 个 Tab → 切 Tab 懒加载 → 页面加载速度比单页面快。

**估行**：前端 ~400 行（从现有页面拆解）

---

### Q-2：资产批量生命周期切换

**目标**：资产列表中多选 → "变更生命周期" → 选目标状态 → 批量更新（含确认弹窗 + 结果报告）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/batch-lifecycle` Body: `{asset_ids, target_state, reason}` |
| `backend/internal/usecase/asset/usecase.go` | 遍历校验（每个 asset 的当前状态是否允许转换）→ 逐条更新 |
| `Frontend/src/pages/AssetsPage.tsx` | 选中行 → "变更生命周期"按钮 → 选择目标状态弹窗 |

**验收**：选 50 个 asset → 改成 archived → 成功 48 个（2 个状态不允许）→ 显示结果汇总。

**估行**：后端 ~60 行 + 前端 ~80 行

---

### Q-3：资产批量 Tag 编辑

**目标**：批量添加/删除/替换 tag（同 Q-2 的批量操作能力）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/batch-tags` Body: `{asset_ids, add: {}, remove: [], replace: {}}` |
| `Frontend/src/pages/AssetsPage.tsx` | 批量操作栏 → "编辑标签" |

**验收**：选 50 个 asset → "添加标签 priority:high" → 全部成功。

**估行**：后端 ~40 行 + 前端 ~60 行

---

### Q-4：资产批量交付增强

**目标**：当前批量交付只有基础功能，增加交付校验（asset 是否已交付给同一客户）、批量交付结果预览、幂等检测。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/delivery/handler.go` | `POST /api/v1/deliveries/batch` 增强校验 |
| `backend/internal/usecase/delivery/usecase.go` | 交付前检查是否已有相同 asset→customer 记录 |
| `Frontend/src/components/asset/BatchDeliveryDialog.tsx` | 新建：批量交付弹窗（预览 → 确认 → 结果） |

**验收**：选 10 个 asset → 批量交付 → 弹窗显示"3 个已交付过" + 7 个可交付 →确认 → 7 个成功。

**估行**：后端 ~50 行 + 前端 ~80 行

---

### Q-5：资产自动归档策略

**目标**：Admin 配置自动归档规则（如：last_delivered_at > 90 天 → 自动 archived），后台定时执行。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/056_archival_policies.sql` | `asset_archival_policies` 表 |
| `backend/internal/usecase/asset/usecase.go` | `ApplyArchivalPolicies()`: 定时任务，查询匹配资产 → 批量归档 |
| `backend/cmd/server/jobs.go` | 注册定时任务（cron daily） |
| `Frontend/src/pages/SettingsPage.tsx` | "归档策略"配置区 |

**验收**：设"90 天未交付 → 归档" → 第二天匹配的 asset 自动归档。

**估行**：后端 ~80 行 + 前端 ~60 行 + 1 迁移

---

### Q-6：资产过期通知

**目标**：资产即将过期（如 30 天后自动删除）时，资产列表显示标注 + 可选通知 owner。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/asset/usecase.go` | `GetExpiringAssets(days)` → 查询即将过期的 |
| `Frontend/src/components/asset/AssetExpiryBadge.tsx` | 新建：过期标签（>60 天黄色 >90 天红色） |
| `Frontend/src/pages/AssetsPage.tsx` | 列表行集成过期标签 |

**验收**：资产还有 5 天过期 → 列表行显示红色"即将过期"标签。

**估行**：后端 ~30 行 + 前端 ~40 行

---

### Q-7：资产自定义元数据（动态字段）

**目标**：允许用户为 asset_type 定义自定义字段 schema（如 "scene" 类型的 asset 有 `location`, `weather` 字段），查询时可筛选。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/057_asset_metadata_schema.sql` | `asset_metadata_schemas` 表（asset_type → 字段定义列表 JSONB） |
| `backend/internal/handlers/asset/handler.go` | `GET/PUT /api/v1/asset-types/:type/metadata-schema` |
| `backend/internal/usecase/asset/usecase.go` | 资产创建/更新时校验自定义字段合法性 |
| `Frontend/src/pages/AssetTypeSettingsPage.tsx` | 新建：资产类型元数据配置页 |
| `Frontend/src/components/asset/AssetMetaFields.tsx` | 新建：根据 schema 动态渲染表单字段 |

**验收**：为"scene"类型配置 `location:text, weather:enum{sunny,rainy}` → 创建 scene 资产时显示这些字段 → 列表可按 weather 筛选。

**估行**：后端 ~120 行 + 前端 ~150 行 + 1 迁移

---

### Q-8：资产模板（预定义元数据模板）

**目标**：保存常用资产创建模板（预设 asset_type + 元数据 + tag），快速创建同类资产。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/058_asset_templates.sql` | `asset_templates` 表 |
| `backend/internal/handlers/asset/handler.go` | 模板 CRUD |
| `Frontend/src/components/asset/AssetTemplateSelector.tsx` | 新建：创建资产时选模板 |

**验收**：创建模板"scene-模板"（type=scene, tag=camera:A）→ 用模板创建 → 预填字段。

**估行**：后端 ~60 行 + 前端 ~80 行 + 1 迁移

---

### Q-9：资产搜索增强（保存搜索 + 搜索告警）

**目标**：搜索条件可保存，匹配新资产时通知用户。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/AssetsPage.tsx` | 搜索栏 + "保存搜索"按钮 |
| `backend/internal/handlers/search/handler.go` | `POST /api/v1/saved-searches` Body: `{name, query_ir, notify_on_match: true}` |
| `backend/internal/usecase/asset/usecase.go` | 定时检查已保存搜索 → 新匹配资产 → 发送通知 |
| `Frontend/src/pages/SavedSearchesPage.tsx` | 新建：已保存搜索列表 + 新匹配结果 |

**验收**：保存搜索"type=scene AND quality=good" → 第二天有新匹配 → 收到通知。

**估行**：后端 ~80 行 + 前端 ~100 行

---

### Q-10：资产血缘 DAG 可视化（升级）

**目标**：从简单的箭头列表升级为 React Flow DAG 图，展示完整的 upstream/downstream 链路。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetLineageDAG.tsx` | 新建：React Flow 血缘图（可交互、缩放、点击节点跳转） |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/:id/lineage` 增强：返回多级上下游 |
| `Frontend/src/components/asset/AssetLineageTab.tsx` | 使用新 DAG 组件 |

**验收**：打开血缘 Tab → DAG 图显示 MCAP→Asset→Algo→Delivery 全链路，点击节点可跳转。

**估行**：后端 ~60 行 + 前端 ~200 行

---

### Q-11：资产关系图浏览器

**目标**：独立"血缘探索"页，从任意资产开始，拖拽探索上下游关系网络。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/LineageExplorerPage.tsx` | 新建：全屏血缘探索（输入 asset_id → 展开 DAG → 双击节点展开更多） |
| `Frontend/src/App.tsx` | 路由 `/lineage` |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/:id/lineage/expand?direction=upstream&depth=2` |

**验收**：输入 asset_id → 展示 DAG → 双击"算法节点"→ 展开该算法处理的更多资产。

**估行**：后端 ~80 行 + 前端 ~250 行

---

### Q-12：资产影响分析

**目标**：删除/修改资产前展示"影响分析"：此资产被哪些下游资产/pipeline/交付引用。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/:id/impact-analysis` |
| `Frontend/src/pages/AssetDetailPage.tsx` | "影响分析"面板（删除前强制展示） |

**验收**：点"删除"→弹窗显示"此资产被 3 个算法、2 个交付引用"→确认后删除。

**估行**：后端 ~50 行 + 前端 ~60 行

---

### Q-13：资产质量评分

**目标**：基于预配置规则（算法成功率、交付频次、tag 覆盖度、元数据完整度）计算资产质量分数，显示在列表和详情。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/asset/usecase.go` | `CalculateQualityScore(asset)` → 加权计算 |
| `backend/migrations/059_asset_quality.sql` | `assets` 表增加 `quality_score DECIMAL, quality_updated_at` |
| `Frontend/src/components/asset/QualityBadge.tsx` | 新建：质量徽章（A/B/C/D 级，颜色区分） |
| `Frontend/src/pages/AssetsPage.tsx` | 列表增加质量列 |

| 维度 | 权重 | 数据来源 |
|------|------|---------|
| 算法覆盖率 | 30% | 通过的算法数 / 总算法数 |
| 交付活跃度 | 25% | 近 30 天交付次数 |
| Tag 覆盖度 | 20% | 已填 tag 数 / 期望 tag 数 |
| 元数据完整度 | 15% | 必填字段填充率 |
| 时效性 | 10% | 创建到现在的天数（越新越高） |

**验收**：资产列表显示质量 A/B/C/D 标签；筛选可按质量等级。

**估行**：后端 ~80 行 + 前端 ~100 行 + 1 迁移

---

### Q-14：资产认证/审批工作流

**目标**：资产从 created 到 ready 需要审批（类似数据发布流程），审批人可批量操作。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/060_asset_approvals.sql` | `asset_approvals` 表 |
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/:id/approve`, `POST /assets/batch-approve`, `GET /assets/pending-approval` |
| `Frontend/src/pages/AssetApprovalPage.tsx` | 新建：待审批资产列表 + 批量通过/驳回 |
| `backend/pkg/notify/feishu.go` | 审批通知 |

**验收**：新资产 → 状态为 pending_approval → 审批人收到通知 → 在审批页通过 → 变为 ready。

**估行**：后端 ~100 行 + 前端 ~150 行 + 1 迁移

---

### Q-15：资产数据预览

**目标**：在资产详情页预览数据内容（图片类显示缩略图、点云/视频类显示元信息、表格类显示前 20 行）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetPreviewFactory.tsx` | 新建：根据 asset_type 选择预览组件 |
| `Frontend/src/components/asset/preview/ImagePreview.tsx` | 新建：图片预览（缩略图 + 原图切换） |
| `Frontend/src/components/asset/preview/VideoPreview.tsx` | 新建：视频预览（首帧 + 时长信息） |
| `Frontend/src/components/asset/preview/TablePreview.tsx` | 新建：表格预览（CSV 前 N 行） |
| `Frontend/src/components/asset/preview/PointCloudPreview.tsx` | 新建：点云预览（基本信息 + 采样数据量） |
| `Frontend/src/pages/AssetDetailPage.tsx` | 基本信息区域嵌入预览 |

**验收**：打开图片类 asset → 显示缩略图 → 点放大 → 原图全屏。

**估行**：前端 ~250 行

---

### Q-16：资产审计时间线增强

**目标**：当前事件流已存在，增强为完整的审计时间线（事件类型过滤、日期范围、操作者筛选、搜索）。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetAuditTimeline.tsx` | 新建：时间线组件（分组按日期、颜色按事件类型、搜索框） |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/:id/events` 增强：支持 `?type=&since=&until=&actor=` |

**验收**：打开时间线 → 按日期分组 → 选"algo_failed"→ 只显示算法失败事件。

**估行**：后端 ~30 行 + 前端 ~120 行

---

### Q-17：资产 Dashboard

**目标**：独立资产仪表盘页面，展示全平台资产统计。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/AssetDashboardPage.tsx` | 新建 |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/dashboard/stats` |

**统计卡片**：
- 总资产数 + 日增长趋势
- 按 lifecycle_state 分布（饼图）
- 按 asset_type 分布（柱状图）
- 近 7 天新增/归档趋势（折线图）
- 存储用量（总量 + 按类型）
- Top 10 Owner
- 算法覆盖率（已覆盖/未覆盖资产数）

**验收**：Dashboard 展示所有统计卡片，数据与 DB 一致。

**估行**：后端 ~80 行 + 前端 ~250 行

---

### Q-18：资产过期/冷数据报告

**目标**：查看"超过 N 天未访问的资产"报告，辅助判断归档/删除。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/reports/stale?days=90` |
| `Frontend/src/pages/AssetDashboardPage.tsx` | "冷数据"报告 Tab |

**验收**：90 天未访问 → 报告列出 → 可全选归档。

**估行**：后端 ~40 行 + 前端 ~80 行

---

### Q-19：资产存储成本分析

**目标**：按资产类型/项目展示存储成本（基于 GCS 用量或估算）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/reports/storage-cost` |
| `Frontend/src/pages/AssetDashboardPage.tsx` | "存储成本" Tab |

**验收**：展示各类型资产存储用量及估算月成本。

**估行**：后端 ~40 行 + 前端 ~80 行

---

### Q-20：资产导入（CSV / JSON）

**目标**：上传 CSV/JSON 文件批量创建/更新资产，支持字段映射预览。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/import` Content-Type: multipart |
| `backend/internal/usecase/asset/usecase.go` | 解析文件 → 逐行校验 → 批量创建/更新 |
| `Frontend/src/components/asset/AssetImportDialog.tsx` | 新建：导入弹窗（选文件 → 字段映射 → 预览 → 确认） |
| `Frontend/src/components/asset/AssetImportResult.tsx` | 新建：导入结果（成功/失败行） |

**验收**：上传 CSV（100 行）→ 映射 asset_id, type, tags → 预览 5 行 → 确认 → 成功 98 行（2 行校验失败）。

**估行**：后端 ~100 行 + 前端 ~150 行

---

### Q-21：资产导出

**目标**：筛选条件 → 导出为 CSV / Parquet / JSON，支持大结果集流式下载。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/export` Body: `{query_ir, format, fields}` |
| `backend/internal/usecase/asset/usecase.go` | 流式查询 → 写入文件 → 返回下载链接 |
| `Frontend/src/pages/AssetsPage.tsx` | 筛选后点"导出" → 选格式 → 下载 |

**验收**：筛选 10000 条 → 导出 CSV → 下载文件包含正确字段和行数。

**估行**：后端 ~80 行 + 前端 ~60 行

---

### Q-22：资产 ownership 转移

**目标**：资产 owner 可转移给其他用户，支持单个和批量。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/batch-transfer` Body: `{asset_ids, new_owner}` |
| `Frontend/src/pages/AssetsPage.tsx` | 批量操作栏 → "转移 Owner" → 选择用户 |

**验收**：转移 → asset owner 字段更新 → 原 owner 列表减少 → 新 owner 列表增加。

**估行**：后端 ~30 行 + 前端 ~60 行

---

### Q-23：资产锁定（防修改/防删除）

**目标**：重要资产可锁定，锁定后不可修改/删除（管理员可强制解锁）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/061_asset_locks.sql` | `assets` 表增加 `locked BOOLEAN DEFAULT false`, `locked_by TEXT`, `locked_at` |
| `backend/internal/middleware/asset_lock.go` | 新建：写操作前检查是否锁定 |
| `Frontend/src/components/asset/AssetLockButton.tsx` | 新建：锁定/解锁按钮 |
| `Frontend/src/pages/AssetsPage.tsx` | 列表行显示锁定图标 |

**验收**：锁定 → 编辑按钮灰色 + 提示"该资产已锁定"；Admin 可强制解锁。

**估行**：后端 ~40 行 + 前端 ~60 行 + 1 迁移

---

### Q-24：资产备注/评论

**目标**：在资产详情中添加备注/评论（类似 annotation），支持 @提及和通知。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/062_asset_comments.sql` | `asset_comments` 表 |
| `backend/internal/handlers/asset/handler.go` | 评论 CRUD |
| `Frontend/src/components/asset/AssetComments.tsx` | 新建：评论区（输入框 + 列表 + @提及） |

**验收**：添加评论 → 显示在详情 → @user → 对方收到通知。

**估行**：后端 ~60 行 + 前端 ~120 行 + 1 迁移

---

### Q-25：资产重复检测与合并

**目标**：基于 hash / 文件名 / 相似度检测重复资产，展示重复组并支持合并。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/asset/usecase.go` | `FindDuplicates()` → 基于 `raw_hash_md5` 和 `uri` 分组 |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/duplicates`, `POST /assets/duplicates/:group_id/merge` |
| `Frontend/src/pages/AssetDuplicatesPage.tsx` | 新建：重复资产列表 + 合并操作 |

**验收**：检测到重复组 → 展示 → 合并 → 保留一个资产，其他标记为 duplicate_of。

**估行**：后端 ~80 行 + 前端 ~120 行

---

### Q-26：资产可访问性控制（ACL）

**目标**：资产级别权限控制（谁可以查看/修改/删除）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/063_asset_acl.sql` | `asset_acl` 表 |
| `backend/internal/middleware/asset_acl.go` | 新建：资产查询/写入时检查 |
| `Frontend/src/components/asset/AssetACLPanel.tsx` | 新建：ACL 配置面板 |

**验收**：设某资产仅 team-A 可见 → team-B 用户搜索不到该资产。

**估行**：后端 ~120 行 + 前端 ~100 行 + 1 迁移

---

### Q-27：资产 Schema 校验

**目标**：创建/更新资产时根据 asset_type 的 schema 定义校验必填字段和字段类型。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/asset/usecase.go` | `ValidateAssetSchema(asset)` → 读取 asset_type schema → 校验 |
| `backend/internal/handlers/asset/handler.go` | POST/PATCH 时调用校验 |

**验收**：必填字段缺失 → 400 + "字段 location 为必填"；字段类型不匹配 → 400。

**估行**：后端 ~50 行

---

### Q-28：资产数据 Profiling

**目标**：对资产存储的数据进行概要分析（行数、大小、空值率、枚举值分布），展示在详情页。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/asset/usecase.go` | `ProfileAssetData(asset)` → 读取 GCS 文件 → 分析 |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/:id/profile` |
| `Frontend/src/components/asset/AssetProfileTab.tsx` | 新建：数据画像 Tab |

**验收**：查看资产画像 → 显示"行数 5000，大小 2.3MB，字段 A 空值率 5%，字段 B 分布{cat:3000, dog:2000}"。

**估行**：后端 ~100 行 + 前端 ~100 行

---

### Q-29：资产合规标签

**目标**：为资产添加合规分类标签（PII / 敏感 / 公开 / 机密），列表和详情中可见，可筛选。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/064_compliance_labels.sql` | `assets` 表增加 `compliance_label TEXT`, `compliance_reviewed_at` |
| `backend/internal/handlers/asset/handler.go` | `PATCH /api/v1/assets/:id/compliance` |
| `Frontend/src/components/asset/ComplianceBadge.tsx` | 新建：合规标签徽章（颜色编码） |

| 标签 | 颜色 | 说明 |
|------|------|------|
| public | 绿色 | 可公开 |
| internal | 蓝色 | 内部可用 |
| sensitive | 黄色 | 限特定团队 |
| pii | 红色 | 含个人隐私数据 |
| restricted | 紫色 | 限特定项目 |

**验收**：标记 PII → 详情显示红色 PII 标签 → 非授权用户需审批才能查看。

**估行**：后端 ~40 行 + 前端 ~60 行 + 1 迁移

---

### Q-30：资产生命周期自动化规则引擎

**目标**：配置规则引擎自动触发资产生命周期变更（如：交付 3 次后状态从 ready → active）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/065_lifecycle_rules.sql` | `lifecycle_rules` 表 |
| `backend/internal/usecase/asset/usecase.go` | `EvaluateLifecycleRules(event)` → 匹配规则 → 执行动作 |
| `Frontend/src/pages/SettingsPage.tsx` | "生命周期规则"配置区 |

**规则示例**：
```
WHEN: delivery_count >= 3
THEN: lifecycle_state = 'active'

WHEN: last_delivery_at > 90d AND lifecycle_state = 'ready'
THEN: lifecycle_state = 'archived'

WHEN: algo_failed_count >= 5 AND lifecycle_state = 'ready'
THEN: lifecycle_state = 'quarantined'
```

**验收**：设规则 → asset 满足条件 → 自动状态变更 → 事件流记录。

**估行**：后端 ~120 行 + 前端 ~100 行 + 1 迁移

---

### Q-31：资产批量操作队列 + 进度

**目标**：批量操作（导入/导出/生命周期变更/打标签）超过 1000 条时走后台队列，前端实时看进度。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/066_batch_operations.sql` | `batch_operations` 表 |
| `backend/internal/usecase/asset/usecase.go` | 后台 goroutine 逐批处理 |
| `Frontend/src/components/asset/BatchOperationProgress.tsx` | 新建：操作进度条 + 详情展开 |

**验收**：批量归档 5000 个 asset → 显示进度 0/5000 → 实时更新 → 5000/5000 完成报告。

**估行**：后端 ~80 行 + 前端 ~100 行 + 1 迁移

---

### Q-32：资产列自定义显示

**目标**：用户在资产列表页可自定义显示列（列选择、顺序拖拽、宽度调整），设置保存到本地或用户配置。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetColumnSettings.tsx` | 新建：列设置弹窗（checkbox + 拖拽排序） |
| `Frontend/src/pages/AssetsPage.tsx` | 集成列设置 |

**验收**：隐藏"owner"列 → 列表不再显示 → 下次打开依然隐藏。

**估行**：前端 ~120 行

---

### Q-33：资产快捷筛选（收藏筛选条件）

**目标**：将常用筛选条件（如"我的待交付资产"）添加为快捷筛选，侧边栏一键切换。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/QuickFilters.tsx` | 新建：快捷筛选栏（可拖拽排序） |
| `Frontend/src/pages/AssetsPage.tsx` | 列表上方展示快捷筛选 |

**验收**：添加"我的待交付"→ 侧边栏显示 → 点击 → 自动筛选。

**估行**：前端 ~100 行

---

### Q-34：资产标签管理页

**目标**：独立标签管理页，查看所有已使用的标签、使用频次、批量重命名/删除/合并。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/TagManagementPage.tsx` | 新建：标签管理（搜索 + 列表 + 频次 + 操作） |
| `backend/internal/handlers/tag/handler.go` | `GET /api/v1/tags/manage` 返回标签使用统计；`POST /tags/rename`, `POST /tags/merge` |
| `Frontend/src/App.tsx` | 路由 `/settings/tags` |

**验收**：重命名 tag "label" → "annotation" → 所有资产中的该 tag 更新；合并 tag "old"→"new" → 自动迁移。

**估行**：后端 ~80 行 + 前端 ~150 行

---

### Q-35：资产通知规则（订阅感兴趣的变化）

**目标**：用户可订阅资产变更通知（如"当 asset A 状态变为 ready 时通知我"）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/migrations/067_asset_subscriptions.sql` | `asset_subscriptions` 表 |
| `backend/internal/handlers/asset/handler.go` | `POST /api/v1/assets/:id/subscribe` |
| `backend/internal/usecase/asset/usecase.go` | 事件发生时检查订阅 → 发送通知 |
| `Frontend/src/pages/AssetDetailPage.tsx` | "订阅"按钮 |

**验收**：订阅 asset → asset 变更 → 收到飞书/邮件通知。

**估行**：后端 ~60 行 + 前端 ~40 行 + 1 迁移

---

## EPIC-R：资产-Pipeline 深度集成

**用户故事**：作为运营/算法工程师，资产管理 和 pipeline 执行不再割裂——从资产可以发起 pipeline，pipeline 产出自动成为资产。
**成功指标**：80% 的 pipeline 执行可以从资产页面一键发起；pipeline 产出物无需手动注册。

---

### R-1：资产详情页"一键 Pipeline"

**目标**：资产详情页新增"处理"按钮 → 选择 pipeline template → 立即部署，资产 ID 自动传入。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetPipelineActions.tsx` | 新建："处理"按钮 + template 选择弹窗 |
| `Frontend/src/pages/AssetDetailPage.tsx` | 操作栏集成该组件 |
| `Frontend/src/api/pipelineApi.ts` | `deployTemplateOnAsset(templateId, assetId)` |

**验收**：打开资产详情 → 点"处理" → 选 template → 部署 → 跳转到 workflow 详情。

**估行**：前端 ~120 行

---

### R-2：资产列表批量发起 Pipeline

**目标**：资产列表中多选 → "用 Pipeline 批量处理" → 选 template → 创建 backfill job。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/pages/AssetsPage.tsx` | 批量操作栏 → "Pipeline处理"按钮 |
| `Frontend/src/components/pipeline/BatchPipelineDialog.tsx` | 新建：批量处理弹窗（选 template + 参数） |
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/pipelines/batch-from-assets` Body: `{asset_ids, template_id, params}` |

**验收**：选中 100 个 asset → 选 template → 确认 → 创建 backfill job → 进度可见。

**估行**：后端 ~60 行 + 前端 ~150 行

---

### R-3：Pipeline 产出自动注册 Asset（后端回调）

**目标**：pipeline 节点完成后，容器通过回调 API 注册产出 asset，自动关联 parent pipeline 和输入资产血缘。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `POST /api/v1/deployments/:id/outputs` Body: `{asset_id, name, type, uri, parent_asset_ids}` |
| `backend/internal/usecase/pipeline/usecase.go` | 注册资产后自动写入 asset_relations（input→output） |
| `docs/pipeline/container-output-spec.md` | 新建：容器输出协议文档 |

**容器协议**：
```
# 容器在处理完成后调用（通过 curl 或 SDK）：
POST https://platform/api/v1/deployments/{deployment_id}/outputs
{
  "assets": [
    {
      "name": "processed-segment-001",
      "asset_type": "processed",
      "uri": "gs://bucket/outputs/001.bin",
      "metadata": {"algo_version": "1.2", "quality": "good"},
      "parent_asset_ids": ["V4ftKjqX"]
    }
  ]
}
```

**验收**：容器完成 → 调用回调 → 资产列表出现新 asset → 资产血缘显示来源于原始 input asset。

**估行**：后端 ~80 行 + 文档 ~30 行

---

### R-4：Pipeline 产出预览（在资产列表中）

**目标**：pipeline 部署详情中产出的 asset 可直接预览，资产列表中 pipeline 产出的资产显示"来自 pipeline"标签。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetSourceTag.tsx` | 新建：来源标签（system / pipeline / manual / import） |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署详情展示产出资产缩略 |
| `backend/migrations/068_asset_source.sql` | `assets` 表增加 `source TEXT`, `source_deployment_id TEXT` |

**验收**：pipeline 产出的资产 → 列表显示"Pipeline"标签 → 点击跳转到部署详情。

**估行**：后端 ~30 行 + 前端 ~80 行 + 1 迁移

---

### R-5：资产搜索作为 Pipeline 输入源

**目标**：pipeline 输入可以不指定具体 asset_ids，而是一个搜索查询（query_ir），运行时动态查询匹配的 asset。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/transpiler/pipeline.go` | `AssetSelection` 增强增加 `QueryIR json.RawMessage` |
| `backend/internal/transpiler/transpiler.go` | Pipeline 启动时注入查询结果 or 运行时容器自查询 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 输入源切换："选择资产" / "搜索条件" |
| `Frontend/src/components/asset/AssetQueryBuilder.tsx` | 新建：搜索条件构建器（复用 queries/validate 的 IR builder） |

**验收**：输入搜索条件"type=scene AND quality=good" → 部署时自动注入匹配的 asset_ids。

**估行**：后端 ~80 行 + 前端 ~150 行

---

### R-6：Pipeline 触发 Asset 事件

**目标**：pipeline 全生命周期事件写入 asset_events outbox（deploy / start / step_complete / step_fail / finish）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | 各生命周期点写入事件 |
| `backend/schemas/events/pipeline_deployed.v1.json` | 新建：事件 schema |
| `backend/schemas/events/pipeline_step_finished.v1.json` | 新建：事件 schema |
| `backend/schemas/events/pipeline_finished.v1.json` | 新建：事件 schema |

**事件定义**：

```
pipeline_deployed:
  deployment_id, template_id, pipeline_name, input_asset_ids

pipeline_step_finished:
  deployment_id, step_name, step_status, duration_ms, error_message

pipeline_finished:
  deployment_id, status, total_duration_ms, node_count, output_asset_ids
```

**验收**：部署 pipeline → asset_events 表出现 pipeline_deployed 事件 → 在资产事件流中可见。

**估行**：后端 ~60 行 + 3 个 schema 文件

---

### R-7：资产驱动 Pipeline 自动触发

**目标**：监听 asset_events 中的特定事件类型，条件匹配时自动触发指定 pipeline（复用 H-3 的事件触发机制）。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `HandleAssetEvent(event)` → 匹配 event_trigger 规则 → 创建 deployment |
| `backend/internal/handlers/pipeline/handler.go` | 从 H-3 的 event trigger 扩展 |
| `Frontend/src/components/pipeline/AssetTriggerConfig.tsx` | 新建：事件触发配置面板 |

**规则配置**：
```
WHEN: event_type = algo_finished AND event_payload.status = ok
THEN: pipeline = "post-process-v1" WITH params: {asset_id: event.asset_id}
```

**验收**：算法完成 → 事件发出 → 事件触发 pipeline 自动执行。

**估行**：后端 ~80 行 + 前端 ~100 行

---

### R-8：Asset-Pipeline 联合血缘图

**目标**：在血缘探索页中同时展示资产和 pipeline 节点——asset → pipeline step → asset 的完整链路。

**文件**：

| 文件 | 修改 |
|------|------|
| `Frontend/src/components/asset/AssetLineageDAG.tsx` | 增强：支持 pipeline step 节点（菱形图标）、deployment 节点 |
| `backend/internal/handlers/asset/handler.go` | `GET /api/v1/assets/:id/lineage` 返回混合图 |

**节点类型**：
- 资产 → 圆形
- pipeline step → 菱形
- pipeline deployment → 圆角矩形
- 算法 → 方形

**验收**：打开血缘 → 显示 asset → pipeline step（箭头）→ 产出 asset 的完整链路。

**估行**：后端 ~60 行 + 前端 ~100 行

---

### R-9：Pipeline 输入资产完整性检查

**目标**：部署前检查所有输入 asset 是否存在且状态可处理，缺失或状态异常时给出明确报告。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/usecase/pipeline/usecase.go` | `ValidateInputAssets(assetIDs)` → 批量查询 → 分组报告 |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 部署弹窗显示校验结果 |

**校验内容**：
- 是否存在（缺失列表）
- lifecycle 是否允许处理（已归档/已删除的有提示）
- 是否有必要的算法结果（如 pipeline 依赖 algo A 的结果 → 检查 asset 是否已跑过 algo A）

**验收**：输入含 3 个不存在的 asset_id → 部署失败 → "资产 [xxx, yyy, zzz] 不存在"。

**估行**：后端 ~50 行 + 前端 ~40 行

---

### R-10：Pipeline 耗时与资产数量关联分析

**目标**：分析 pipeline 运行耗时与输入资产数量的关系，帮助用户估算未来需要的时间。

**文件**：

| 文件 | 修改 |
|------|------|
| `backend/internal/handlers/pipeline/handler.go` | `GET /api/v1/pipelines/:id/performance-model` |
| `Frontend/src/components/pipeline/DeployPanel.tsx` | 根据输入资产数预估耗时 |

**验收**：选 100 个 asset → 部署弹窗显示"预估耗时: 3-5 分钟（基于历史 12 次运行）"。

**估行**：后端 ~60 行 + 前端 ~40 行

---

## 总览汇总

| EPIC | 领域 | Task 数 | 后端估行 | 前端估行 | 迁移 |
|------|------|---------|---------|---------|------|
| — | Phase 0 清理 | 3 | ~45 | 0 | 0 |
| A | Pipeline Designer 画布 | 16 | 0 | ~1,395 | 0 |
| B | 组件注册表 | 11 | ~550 | ~790 | 3 |
| C | Transpiler 引擎 | 12 | ~540 | ~130 | 0 |
| D | 执行与部署 | 10 | ~500 | ~440 | 2 |
| E | 运行监控 | 12 | ~440 | ~1,030 | 0 |
| F | Asset-Pipeline 集成 | 9 | ~480 | ~560 | 2 |
| G | Backfill 批回放 | 7 | ~400 | ~390 | 1 |
| H | 调度与触发器 | 8 | ~500 | ~590 | 4 |
| I | 密钥与配置管理 | 7 | ~320 | ~300 | 0 |
| J | 通知与告警 | 6 | ~310 | ~280 | 0 |
| K | 权限与治理 | 9 | ~600 | ~740 | 4 |
| L | Template 管理增强 | 8 | ~310 | ~550 | 1 |
| M | API 与开发者工具 | 8 | ~620 | ~170 | 0 |
| N | 协作与市场 | 8 | ~340 | ~720 | 3 |
| O | UX 体验打磨 | 8 | ~30 | ~820 | 0 |
| P | 高级功能 | 11 | ~680 | ~870 | 1 |
| Q | 资产平台核心增强 | 35 | ~2,090 | ~3,770 | 11 |
| R | 资产-Pipeline 深度集成 | 10 | ~630 | ~870 | 2 |
| | **合计** | **~190** | **~9,385** | **~14,425** | **34** |

---

## 依赖关系

```
Phase 0 (P0-CLEANUP-1,2,3) → 所有后续 Task 的基础
                                │
        ┌───────┬───────┬──────┬┴──────┬───────┬──────┬──────┬──────┐
        ▼       ▼       ▼      ▼       ▼       ▼      ▼      ▼      ▼
       A-*    B-1,2   C-*    D-*     E-*     L-*    M-*    Q-*    R-*
        │       │                      │              │             │
        ▼       ▼                      ▼              ▼             ▼
       F,G     B-3~11                H,I,J,N     Q-3~35         R-2~10
        │       │                      │              │
        ▼       ▼                      ▼              ▼
       K,O,P   K,O,P                K,O,P          K,O,P
```

**核心依赖链**：
- `Q-1 (Asset Detail Tab重构)` → 许多 Q 子 Task 依赖 Tab 架构
- `Q-10 (Lineage DAG)` → R-8 (联合血缘图)
- `R-3 (产出自动注册)` → R-4 (产出预览), R-8 (联合血缘)
- `R-5 (搜索作为输入源)` → 大量 Pipeline 自动化场景
- `Q-30 (生命周期规则引擎)` → 资产自动化治理

**推荐启动顺序**：

| 波次 | Task | 理由 |
|------|------|------|
| Wave 1 | P0-CLEANUP + A-1~4 + C-1~4 + B-1 + E-1 + L-1 + Q-1~3 | 基础设施 + 资产详情重构 |
| Wave 2 | A-5~10 + B-2~3 + C-5~8 + D-1~3 + E-2~4 + F-1 + M-1~2 + Q-4~10 + R-1 | 核心功能 + 资产增强 |
| Wave 3 | D-4~8 + E-5~8 + F-3~5 + G-1~3 + H-1~3 + I-1~3 + J-1~2 + K-1~3 + Q-11~20 + R-2~5 | 规模功能 + 资产批量治理 |
| Wave 4 | C-9~12 + E-9~12 + F-6~9 + G-4~7 + H-4~8 + I-4~7 + J-3~6 + K-4~9 + L-2~8 + M-3~8 + N-1~8 + O-1~8 + P-1~11 + Q-21~35 + R-6~10 | 高级功能 + 锦上添花 |

**Phase 1 (~190 Task) 总估算**：~24,000 行代码 + 34 个迁移文件
**Phase 1 预计工期**：6-8 个月（1 个全栈开发者）
**建议**：按 Wave 分 PR，每个 Wave 独立可交付
