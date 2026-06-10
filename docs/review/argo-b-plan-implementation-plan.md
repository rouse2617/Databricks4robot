# Argo UI 集成实施计划（B 方案 — 完整版）

## 0. 源码分析结论

### 0.1 Argo UI 架构速览

| 维度 | 值 |
|------|-----|
| React | 18.3.1 |
| 路由 | react-router / react-router-dom 4.x (`<Router>`, `<Switch>`, history v4) |
| HTTP 客户端 | `superagent`（非 axios/fetch） |
| UI 库 | `argo-ui`（Argo 自研，git 依赖）+ 自定义 SCSS |
| 构建 | webpack 5 + sass + esbuild-loader |
| TypeScript | 4.6.4 |
| SSE | 原生 `EventSource` 包装为 RxJS Observable |
| 代码编辑器 | monaco-editor 0.45.0 |
| 终端模拟器 | xterm 4.19.0（argo-ui 内 lazy import） |
| DAG | SVG/DOM，dagre 布局，自研 Graph 类 |

### 0.2 我们的前端架构速览

| 维度 | 值 |
|------|-----|
| React | 19.1.0 |
| 路由 | react-router-dom 7.6 (`<BrowserRouter>` / `<Routes>`) |
| HTTP 客户端 | 双客户端：axios（大多数 API）+ fetch（pipeline/workflow API） |
| UI 库 | antd 5.20 + Tailwind 3.4 |
| 构建 | vite 6.3 |
| TypeScript | 5.8.3 |
| 拖拽画布 | @ant-design/pro-flow 1.3（基于 @xyflow/react 12） |
| SSE | 原生 EventSource（asset events） |

### 0.3 关键不兼容点

| 差异项 | Argo UI | Cyber Frontend | 影响 |
|--------|---------|---------------|------|
| React | 18 (useSyncExternalStore 不可用) | 19 | 不能合并到同一 SPA |
| 路由体系 | react-router 4 (`<Switch>` / history) | react-router 7 (`<Routes>`) | 路由配置风格完全不同 |
| HTTP 层 | superagent | axios/fetch | 认证拦截器需要分别配置 |
| UI 组件 | argo-ui | antd | 样式体系完全不同 |
| 构建 | webpack | vite | 需要独立构建流水线 |
| TS | 4.6 | 5.8 | 类型系统差异 |

**结论：Fork 后的 Argo UI 必须独立构建运行，不能合并到 Cyber 的 Vite SPA 中。**

### 0.4 后端数据流

```
当前架构:
  Argo Server → (REST API) → Cyber Backend (Go) → (简化 JSON) → Cyber Frontend

Fork 后需要:
  Argo Server → (REST API) → Cyber Backend → (完整 Argo 模型) → Argo UI Fork
```

我们的后端 `argo.Client` 已经通过 `wfv1.Workflow` 拿到了完整的 Argo 数据模型，但当前 handler 把它简化为 `{name, status, nodeCount, ...}` 后再返回前端。Fork 方案需要在我们的后端新增「透传端点」或「字段完备端点」，让 Argo UI 能拿到完整数据。

---

## 1. 文件变更清单

### 1.1 Argo UI Fork 内部变更

```
argo-ui/ (独立目录，Fork 自 /Users/rick/src/argo-workflows/ui/)

A. 新增文件 (~12 个)
├── src/shared/adapters/
│   ├── http-client.ts          (替换 superagent → fetch，注入 cookie 认证，~60 行)
│   ├── api-paths.ts            (URL 路径映射：Argo 风格 → Cyber 风格，~40 行)
│   └── field-mapper.ts         (字段补齐：Cyber 响应 → Argo 期望模型，~80 行)
├── src/app/workflow-submit/
│   ├── index.ts                (模块导出，~5 行)
│   ├── workflow-submit-page.tsx(提交页主组件，~200 行)
│   ├── canvas-wrapper.tsx      (React 18 兼容的 iframe/portal 包装器，~60 行)
│   └── styles.scss             (提交页样式，~50 行)
├── src/index.html              (独立 HTML 入口，~20 行)
└── .env                        (开发环境变量，~5 行)

B. 修改文件 (~8 个)
├── src/shared/services/requests.ts
│   (替换 superagent → 改用 adapters/http-client.ts)
├── src/shared/services/workflows-service.ts
│   (submit 方法改为调用 Cyber 后端，字段映射)
├── src/shared/base.ts
│   (apiUrl 改为可配置的 Cyber 后端地址)
├── src/app-router.tsx
│   (新增 /workflows/new 路由，注入 canvas 组件)
├── src/workflows/index.ts
│   (注册新路由)
├── src/workflows/components/workflows-container.tsx
│   (添加 /new 路由)
├── src/workflows/components/workflows-list/
│   workflows-list.tsx (Submit New Workflow 按钮改为跳转 /new)
└── package.json
│   (新增依赖或脚本)

C. 不需要改的文件清单（保留原样）
├── src/workflows/components/workflow-dag/          (DAG 图组件)
├── src/workflows/components/workflow-logs-viewer/  (日志查看)
├── src/workflows/components/workflow-details/      (详情页)
├── src/workflows/components/workflow-yaml-viewer/  (YAML 查看)
├── src/workflows/components/workflow-node-info/    (节点信息)
├── src/workflows/components/workflow-panel/        (DAG 面板)
├── src/workflows/components/workflow-timeline/     (时间线)
├── src/workflows/components/events-panel.tsx       (事件面板)
├── src/shared/components/graph/                    (图布局引擎)
├── src/shared/models/                              (所有类型定义)
├── src/shared/services/(除 requests.ts 外)         (其他 service)
```

### 1.2 Cyber 主项目变更

```
Cyber Frontend/ (~3-5 个文件)

A. 修改文件
├── vite.config.ts               (添加 /argo-ui 子路径的代理规则，~10 行)
├── nginx.conf 或部署配置        (反向代理 /argo-ui → argo-ui dev server，~15 行)
├── Frontend/src/App.tsx         (WorkflowListPage 跳转路径指向 argo-ui，~5 行)
└── Frontend/src/pages/
    └── WorkflowDetailPage.tsx   (详情页改为 iframe 或直接跳转到 argo-ui，~20 行)

Cyber Backend/ (~3 个文件)

A. 新增文件
├── backend/internal/handlers/workflow/
│   └── argo_ui_handler.go      (Argo UI 透传端点，~200 行)

B. 修改文件
├── backend/routes/routes.go     (注册新端点，~15 行)

C. 新增 API 端点 (见第 4 节)
```

### 1.3 部署/基础设施变更

```
A. 新增文件
├── argo-ui/Dockerfile           (~15 行)
├── deploy/cloudrun/
│   └── argo-ui-dev.sh          (Cloud Run 部署脚本，~30 行)
└── nginx/
    └── argo-ui-proxy.conf      (nginx 反向代理规则，~20 行)
```

---

## 2. 实施阶段

### Phase 1: Fork & 环境搭建（1-2 天）

**目标：Argo UI 独立可运行，访问真实数据**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P1.1 | 复制 `argo-workflows/ui/` 到 `cyber-databrew/argo-ui/` | 全量复制 | — |
| P1.2 | 安装依赖 `cd argo-ui && yarn install` | — | — |
| P1.3 | 创建 `argo-ui/.env`，配置 `API_BASE_URL` 指向 Cyber 后端 | `.env` | 5 |
| P1.4 | 修改 `base.ts`，让 `apiUrl()` 可读取环境变量 | `base.ts` | 10 |
| P1.5 | 替换 `requests.ts` 的 HTTP 层：superagent → fetch + `credentials: include` | `adapters/http-client.ts`, `requests.ts` | 80 |
| P1.6 | 验证 `yarn start` 可运行，列表页能加载数据 | — | — |

**验证：浏览器打开 `localhost:8080`，Workflows 列表加载 Cyber 后端数据，详情页 DAG 图正常渲染。**

### Phase 2: Submit 入口替换（2-3 天）

**目标：拖拽画布替换 Argo 的 Submit 表单**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P2.1 | 修改 `workflows-list.tsx`，将 "Submit New Workflow" 按钮的 action 从 `sidePanel` 改为导航到 `/workflows/new` | `workflows-list.tsx` | 5 |
| P2.2 | 在 `workflows-container.tsx` 添加 `<Route path="/:namespace/new">` | `workflows-container.tsx` | 10 |
| P2.3 | 创建 `workflow-submit-page.tsx`：提交页主框架 | `workflow-submit-page.tsx` | 200 |
| P2.4 | 创建 `canvas-wrapper.tsx`：包装我们的拖拽画布组件（iframe 嵌入或 web component） | `canvas-wrapper.tsx` | 60 |
| P2.5 | 在 Cyber Frontend 的 PipelinePage 中抽取出可嵌入的「提交模式」 | TBD — [TODO: 需验证] | 100 |
| P2.6 | 实现 Submit 按钮：调用 Cyber 后端的 pipeline deploy 接口 | `workflow-submit-page.tsx` | 50 |
| P2.7 | 创建 `workflow-submit/styles.scss`：与 Argo 风格一致的样式 | `styles.scss` | 50 |

**验证：点击 "Submit New Workflow" → 跳转到 `/workflows/new` → 显示拖拽画布（初始桩：一个 `<div>Canvas Area</div>`）→ 可提交 pipeline。**

### Phase 3: API 适配层（2-3 天）

**目标：Argo UI 的所有功能（列表/详情/DAG/日志/操作）通过 Cyber 后端正常工作**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P3.1 | 后端：新增 Argo UI 透传端点（直接返回 wfv1.Workflow） | `argo_ui_handler.go` | 200 |
| P3.2 | 后端：注册新路由 | `routes.go` | 15 |
| P3.3 | 前端：字段映射 `field-mapper.ts`（Cyber 简化模型 ↔ Argo 完整模型） | `field-mapper.ts` | 80 |
| P3.4 | 前端：路径映射 `api-paths.ts`（Argo API 路径 → Cyber API 路径） | `api-paths.ts` | 40 |
| P3.5 | 修改 `workflows-service.ts`，所有 API 调用走 Cyber 后端 | `workflows-service.ts` | 100 |
| P3.6 | 验证列表/详情/DAG/日志全部正常 | — | — |

**验证：Argo UI 的列表页、详情页（DAG/时间线/概要）、日志查看、YAML 查看全部功能正常。**

### Phase 4: SSE / Watch 与实时状态（2-3 天）

**目标：实时状态更新，日志 streaming**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P4.1 | 后端：分析 Argo Server 的 SSE 端点格式，决定是透传还是自定义 | `argo_ui_handler.go` | 100 |
| P4.2 | 后端：实现 workflow watch SSE 端点（透传 Argo Server workflow-events） | `argo_ui_handler.go` | 80 |
| P4.3 | 后端：实现 workflow log SSE 端点（透传 Argo Server log streaming） | `argo_ui_handler.go` | 80 |
| P4.4 | 前端：`requests.ts` 的 `loadEventSource` 验证 SSE 连接正常 | — | — |
| P4.5 | 若 SSE 不可用，降级方案：前端轮询（30s 间隔） | `workflows-service.ts` | 50 |

**验证：运行中的 workflow 状态实时更新；日志 streaming 实时显示。**

### Phase 5: 认证打通（1-2 天）

**目标：Argo UI 与 Cyber 主站共享认证状态**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P5.1 | Argo UI 的 fetch client 配置 `credentials: "include"`（已完成于 P1.5） | `http-client.ts` | — |
| P5.2 | 修改 401 处理：Argo UI 发现 401 → 跳转 Cyber 登录页（非 Argo 登录页） | `http-client.ts` | 10 |
| P5.3 | 修改 `cookie.ts`：`setCookie`/`getCookie` 的 `SameSite` 和 `path` 策略 | `cookie.ts` | 5 |
| P5.4 | 后端：在透传端点添加 auth middleware | `argo_ui_handler.go` | 10 |

**验证：在主站登录后，访问 Argo UI 直接有数据；退出登录后 401 跳转 Cyber 登录页。**

### Phase 6: 完整画布集成（3-5 天）

**目标：拖拽画布完整功能嵌入 Argo UI，Pipeline 可以正式提交流水线**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P6.1 | 将 Cyber 的 PipelinePage 核心逻辑抽离为独立的 `<SubmitCanvas />` 组件 | Cyber Frontend | 300 |
| P6.2 | 构建该组件的独立 bundle（webpack/vite library mode） | Cyber Frontend | 100 |
| P6.3 | 在 Argo UI 中通过 iframe 或模块加载引入 | `canvas-wrapper.tsx` | 100 |
| P6.4 | 提交逻辑：用户点击 Submit → 调用 Cyber deploy API → 返回 workflow name → 跳转详情页 | `workflow-submit-page.tsx` | 80 |
| P6.5 | 部署后验证全链路 | — | — |

**验证：在 Argo UI 的 Submit 页面使用拖拽画布编排流水线 → 提交成功 → 跳转到 Argo 详情页 → 能看到运行的 DAG。**

### Phase 7: 路由全面接管与稳定性（2-3 天）

**目标：Argo UI 完全接管 Workflow 相关路由，移除 Cyber 自己的 WorkflowListPage/WorkflowDetailPage**

| # | 任务 | 文件 | 预估行数 |
|---|------|------|----------|
| P7.1 | Cyber 前端：`/workflows` 路由直接跳转到 Argo UI | `App.tsx` | 5 |
| P7.2 | Cyber 前端：移除 `WorkflowListPage` / `WorkflowDetailPage` 的入口 | `App.tsx` | 10 |
| P7.3 | Nginx/反向代理：配置 `/argo-ui` 路径规则 | nginx.conf | 20 |
| P7.4 | 面包屑/导航联动：Argo UI 的返回按钮链回 Cyber 主站 | `app-router.tsx` | 20 |
| P7.5 | 完整回归测试：所有 workflow 操作正常 | — | — |

### 预估总工期

| Phase | 内容 | 预估人天 |
|-------|------|----------|
| P1 | Fork & 环境搭建 | 1-2 |
| P2 | Submit 入口替换 | 2-3 |
| P3 | API 适配层 | 2-3 |
| P4 | SSE / Watch | 2-3 |
| P5 | 认证打通 | 1-2 |
| P6 | 完整画布集成 | 3-5 |
| P7 | 路由接管 | 2-3 |
| **总计** | | **13-21 人天** |

---

## 3. 字段映射表

### 3.1 Workflow 列表

| Cyber 当前返回 | Argo UI 期望 (WorkflowList.items[].metadata.*) | 映射方式 |
|---|---|---|
| `{items: [{name, status, nodeCount, createdAt, finishedAt, labels}]}` | `{items: [{metadata: {name, namespace, uid, creationTimestamp, labels}, status: {phase, message, finishedAt, startedAt, progress, estimatedDuration}, spec: {suspend, arguments}}]}` | 后端新增透传端点 |
| `status: string` | `status.phase: "Running" \| "Succeeded" \| ...` | 直接映射 |
| `nodeCount: number` | 无（Argo 不返回，客户端从 `status.nodes` 计算） | — |
| `createdAt: string` | `metadata.creationTimestamp: Time` | 格式一致 (RFC3339) |
| `finishedAt?: string` | `status.finishedAt: Time` | 格式一致 |
| `labels?: Record<string, string>` | `metadata.labels?: Record<string, string>` | 直接映射 |

### 3.2 Workflow 详情

| Cyber 当前返回 | Argo UI 期望 (Workflow) | 映射方式 |
|---|---|---|
| `{name, status, nodes: [...], edges: [...]}` | `{metadata: {...}, spec: {...}, status: {phase, nodes: {[id]: NodeStatus}, ...}}` | 后端新增透传端点 |
| `nodes: WorkflowNodeStatus[]` (数组) | `status.nodes: {[nodeId: string]: NodeStatus}` (字典) | 透传端点直接返回字典 |
| `edges: WorkflowDagEdge[]` | 无（Argo UI 从 nodes 的 children 字段计算边） | 后端直接返回完整 nodes |

### 3.3 NodeStatus 字段

| Cyber 当前返回 | Argo UI 期望 | 缺失字段 |
|---|---|---|
| `id, name, displayName, type, templateName, phase, message, children, startedAt, finishedAt, estimatedDuration` | 全部上述 + `boundaryID, podIP, daemoned, retryStrategy, outputs, outboundNodes, inputs, templateRef, templateScope, hostNodeName, memoizationStatus, progress, resourcesDuration` | `boundaryID`（影响 DAG 模板分组）, `outputs`, `inputs`, `templateRef`, `templateScope`, `hostNodeName`, `memoizationStatus`, `podIP`, `daemoned`, `retryStrategy`, `outboundNodes` |

**核心缺失字段对功能的影响：**

| 字段 | 影响的功能 |
|------|-----------|
| `boundaryID` | DAG 中 templateRef 分组渲染（`WorkflowDag.prepareGraph` 第 191-195 行） |
| `outputs` | 节点产出物（参数/artifact）展示、日志 artifact 路径判断 |
| `inputs` | 节点入参展示 |
| `outboundNodes` | DAG 图出口边计算（`WorkflowDag.getOutboundNodes`） |
| `templateRef` | 模板引用解析、节点标签显示 |
| `hostNodeName` | 节点信息面板展示 |

**结论：后端透传端点必须返回完整的 Argo `WorkflowStatus`，不能只返回简化版。我们后端的 `argo.Client.GetWorkflow()` 已经返回完整的 `wfv1.Workflow`，透传即可。**

### 3.4 Submit 操作

| Argo UI submit 参数 | Cyber 后端期望 | 映射方式 |
|---|---|---|
| `kind: "WorkflowTemplate"` / `"ClusterWorkflowTemplate"` / `"CronWorkflow"` | Pipeline + deploy | 前端适配：拖拽画布 → 组装 Pipeline JSON → 调用 deploy API |
| `name: string` (模板名) | 无模板概念（用户画 free-form pipeline） | N/A（不需要模板） |
| `namespace: string` | `X-Databrew-Token` 中的 namespace | 由后端从 cookie/session 读取 |
| `entryPoint: string` | 画布自动决定 | — |
| `parameters: string[]` | Pipeline 中每个节点的参数 | 画布组装 |
| `labels: string` | labels 在 deploy payload 中 | 画布组装 |

---

## 4. 新增 API 端点列表

### 4.1 Argo UI 透传端点（Cyber 后端新增）

这些端点直接透传 Argo Server 的响应，字段完备。

| 方法 | 路径 | 说明 | 替代 Argo 原始路径 |
|------|------|------|-------------------|
| `GET` | `/api/v1/workflows/{namespace}` | Workflow 列表（完整 Argo 模型） | `/api/v1/workflows/{namespace}` |
| `GET` | `/api/v1/workflows/{namespace}/{name}` | Workflow 详情（完整 Argo 模型） | `/api/v1/workflows/{namespace}/{name}` |
| `GET` | `/api/v1/workflow-events/{namespace}?{query}` | SSE Workflow 状态 watch | `/api/v1/workflow-events/{namespace}` |
| `GET` | `/api/v1/workflows/{namespace}/{name}/log?{query}` | SSE 日志 streaming | `/api/v1/workflows/{namespace}/{name}/log` |
| `POST` | `/api/v1/workflows/{namespace}/submit` | 提交 workflow | `/api/v1/workflows/{namespace}/submit` |
| `PUT` | `/api/v1/workflows/{namespace}/{name}/retry` | 重试 | `/api/v1/workflows/{namespace}/{name}/retry` |
| `PUT` | `/api/v1/workflows/{namespace}/{name}/resubmit` | 重新提交 | `/api/v1/workflows/{namespace}/{name}/resubmit` |
| `PUT` | `/api/v1/workflows/{namespace}/{name}/suspend` | 暂停 | `/api/v1/workflows/{namespace}/{name}/suspend` |
| `PUT` | `/api/v1/workflows/{namespace}/{name}/resume` | 恢复 | `/api/v1/workflows/{namespace}/{name}/resume` |
| `PUT` | `/api/v1/workflows/{namespace}/{name}/stop` | 停止 | `/api/v1/workflows/{namespace}/{name}/stop` |
| `PUT` | `/api/v1/workflows/{namespace}/{name}/terminate` | 终止 | `/api/v1/workflows/{namespace}/{name}/terminate` |
| `DELETE` | `/api/v1/workflows/{namespace}/{name}` | 删除 | `/api/v1/workflows/{namespace}/{name}` |

### 4.2 Auth/Info 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/userinfo` | 用户信息（Argo UI 用于获取 namespace） |
| `GET` | `/api/v1/info` | 系统信息（版本/links/navColor/managedNamespace） |

**注意：** `GET /api/v1/info` 的响应格式需要包含 `managedNamespace` 字段，Argo UI 据此判断是否展示 namespace 选择器。

---

## 5. 关键组件依赖链

### 5.1 DAG 图完整依赖链

```
WorkflowDetails (详情页)
  └─ WorkflowPanel
      └─ WorkflowDag (类组件, React.Component<WorkflowDagProps, WorkflowDagRenderOptions>)
          ├─ Props:
          │   workflowName: string
          │   artifactRepositoryRef?: ArtifactRepositoryRefStatus
          │   nodes: {[nodeId: string]: NodeStatus}     ← 核心数据：节点字典
          │   selectedNodeId?: string
          │   nodeSize?: number
          │   hideOptions?: boolean
          │   nodeClicked?: (nodeId: string) => any     ← 点击节点回调
          │
          ├─ 内部状态 State:
          │   expandNodes: Set<string>     (展开的节点)
          │   showArtifacts: boolean       (localStorage 持久化)
          │   showInvokingTemplateName: boolean  (localStorage 持久化)
          │   showTemplateRefsGrouping: boolean   (localStorage 持久化)
          │
          ├─ 核心方法:
          │   prepareGraph(): 遍历 nodes 树构建 Graph 对象
          │     - 使用 NodeStatus.children 确定父子关系
          │     - Collapse 逻辑: children.length > 3 时折叠中间节点
          │     - boundaryID 用于 templateRef 分组渲染
          │     - outboundNodes 用于 onExit handler 边计算
          │
          └─ 渲染: GraphPanel (shared/components/graph/graph-panel.tsx)
               └─ 依赖:
                   graph/types.ts (Graph, Node, Edge 等数据结构)
                   graph/layout.ts (dagre 布局算法)
                   graph/icon.tsx (节点图标)
                   graph/label.tsx (节点标签)
                   icons.ts (workflow node 类型 → 图标映射)
                   genres.ts (node type → 类型分类)
```

### 5.2 日志查看器完整依赖链

```
WorkflowDetails
  └─ SlidingPanel (sidePanel === 'logs')
      └─ WorkflowLogsViewer
          ├─ Props: {workflow: Workflow, initialNodeId?, initialPodName?, container, archived}
          │
          ├─ 核心逻辑:
          │   useEffect → services.workflows.getContainerLogs(workflow, podName, ...)
          │     → requests.loadEventSource(url)  ← SSE 连接
          │     → publishReplay() + refCount()   ← RxJS 多播
          │     → map(LogEntry → string)         ← 日志行转化
          │     → map(grep filter)               ← 搜索高亮
          │     → map(parseAndTransform timestamp)
          │
          ├─ UI: FullHeightLogsViewer
          │     └─ React.lazy → import LogsViewer from 'argo-ui'
          │         └─ LogsViewer (argo-ui 库内)
          │             └─ xterm (终端渲染, webpack dynamic import)
          │
          └─ 依赖数据:
              workflow.status.nodes[nodeId]          ← 获取 pod/container 信息
              workflow.metadata.namespace + name     ← API 路径参数
              getPodName(workflow, node)             ← pod 名解析
              execSpec(workflow).templates           ← 容器列表
              execSpec(workflow).podGC               ← GC 配置提示
```

### 5.3 节点详情面板依赖链

```
WorkflowDetails
  └─ 右侧 side panel → WorkflowNodeInfo
      ├─ Props: {node: NodeStatus, workflow, links, onShowContainerLogs, onShowEvents, ...}
      │
      ├─ 展示字段 (来自 NodeStatus):
      │   - displayName, name, id, type, phase
      │   - startedAt, finishedAt, estimatedDuration
      │   - message (错误/状态信息)
      │   - progress
      │   - resourcesDuration
      │   - hostNodeName
      │   - templateName / templateRef / templateScope
      │   - inputs (parameters + artifacts)
      │   - outputs (parameters + artifacts + result + exitCode)
      │   - memoizationStatus (hit, key, cacheName)
      │   - podIP (daemoned steps)
      │
      ├─ 依赖:
      │   getResolvedTemplates(workflow, node)  ← 模板解析
      │   links.filter(link => link.scope === '...')  ← 外部链接
      │   services.workflows.set()  ← set output parameters
      │   services.workflows.resume()  ← resume suspended node
      │
      └─ 子面板 (通过 onShowContainerLogs/onShowYaml/onShowEvents 切换):
          - WorkflowLogsViewer
          - WorkflowYamlViewer (SerializingObjectEditor)
          - EventsPanel
```

### 5.4 列表页实时 Watch 依赖链

```
WorkflowsList
  └─ ListWatch<Workflow>
      ├─ 初始加载: services.workflows.list(namespace, phases, labels, pagination, ...)
      │     → GET api/v1/workflows/{namespace}
      │     → 返回 WorkflowList {items: Workflow[], metadata: ListMeta}
      │
      └─ Watch 订阅: services.workflows.watchFields({namespace, phases, labels, resourceVersion})
            → requests.loadEventSource(url)
            → Observable<WatchEvent<Workflow>>
            → 根据 event.type (ADDED/MODIFIED/DELETED) 更新列表
```

### 5.5 详情页 Watch 依赖链

```
WorkflowDetails
  └─ RetryWatch<Workflow>
      ├─ services.workflows.watch({name, namespace})
      │     → GET api/v1/workflow-events/{namespace}?name={name}
      │     → 根据 event.type:
      │         'DELETED' → 标记为 archived 或显示错误
      │         其他 → setWorkflow(e.object) 更新详情
      │
      └─ 初始加载:
          services.workflows.get(namespace, name, uid?)
            → 404 → try archived workflow
```

---

## 6. 技术决策记录

### 6.1 认证方案

**决策：Cookie 同源共享方案**

- Argo UI 部署在与 Cyber 主站**同域**（例如 Cyber 在 `databrew.example.com/`，Argo UI 在 `databrew.example.com/argo-ui/`）
- Argo UI 的 HTTP 客户端（替换 superagent → fetch）设置 `credentials: "include"`
- 所有请求自动携带 Cyber 后端的 session cookie
- Cyber 后端在 Argo UI 透传端点前放置 auth middleware，校验 cookie/header
- 401 处理：Argo UI 跳转到 `/login`（Cyber 登录页，不是 Argo 的 login 组件）

**备选（如果必须不同域）：**
- Cyber 后端在透传端点同时支持 `X-Databrew-Token` header
- Argo UI 从 localStorage 或 cookie 读取 token 并注入到每个请求 header

**理由：** 同域 cookie 是 Web 标准中最可靠的认证共享方式，不需要额外的 OAuth/SSO/JWT 桥接。

### 6.2 路由融合方案

**决策：Nginx 路径分发 + SPA historyApiFallback**

```
location /argo-ui/ {
    # Argo UI 的静态资源和 SPA 路由
    alias /app/argo-ui/dist/;
    try_files $uri $uri/ /argo-ui/index.html;
}

location /api/v1/workflows {
    # Workflow API 由 Cyber 后端处理（透传 Argo Server）
    proxy_pass http://cyber-backend;
}

location / {
    # 其他路径走 Cyber 主站
    proxy_pass http://cyber-frontend;
}
```

Cyber Frontend 的 `App.tsx` 中 `/workflows` 路由直接重定向到 `/argo-ui/workflows/`。

**理由：** 路径分发避免了 SPA 内嵌 SPA 的复杂度，且每个 SPA 保持独立的 history API。

### 6.3 构建部署方案

**决策：Argo UI 保持 webpack 独立构建，与 Cyber Vite 并行**

- `argo-ui/package.json` 保留原始 webpack 构建脚本
- `argo-ui/Dockerfile` 单独构建，产出静态资源
- Cloud Run 部署：Argo UI 作为共享的静态资源层，与 Cyber 后端同容器部署
- Dev 模式：`webpack-dev-server` + Vite dev server，通过 nginx/haproxy 统一入口

**理由：** 迁移到 Vite 需要处理 monaco-editor-webpack-plugin、argo-ui 的 webpack 配置、scss loader 等问题，性价比极低。保持 webpack 是最小摩擦的方案。

### 6.4 画布嵌入方案

**决策：iframe 嵌入 Cyber 的拖拽画布**

```
Argo UI 的 /workflows/new 页面:
┌─────────────────────────────────────────┐
│ Argo 导航栏 (Layout navBar)              │
├─────────────────────────────────────────┤
│ 面包屑: Workflows > New Pipeline         │
│                                          │
│ ┌─────────────────────────────────────┐  │
│ │ iframe: /pipeline?mode=submit       │  │
│ │ (Cyber PipelinePage 的提交模式)     │  │
│ │                                     │  │
│ └─────────────────────────────────────┘  │
│                                          │
│ [Submit] [Cancel]                        │
└─────────────────────────────────────────┘
```

iframe 与父页面通过 `postMessage` 通信：
- 画布 ready → `postMessage({type: 'canvas-ready'})`
- 提交 → `postMessage({type: 'submit', pipeline: {...}})`
- 父页面收到 pipeline → 调用 deploy API → 跳转详情页

**理由：** 避免 React 18/19 冲突、react-router 4/7 冲突、antd/argo-ui 冲突。iframe 边界清晰，通信协议简单。

### 6.5 上游维护策略

**决策：最小 diff 的「补丁集」方式**

- Fork 后不重构 Argo UI 的整体架构
- 只修改「必要的胶水层」（requests.ts, base.ts, workflows-service.ts, app-router.tsx, workflows-list.tsx）
- 所有修改用 `/// CYB-PATCH:` 注释标记（便于 `grep` 定位差异）
- 上游更新时：
  1. 拉取上游最新代码到临时分支
  2. `diff -r upstream-ui/src argo-ui/src | grep -v CYB-PATCH` 查看非补丁的差异
  3. 选择性合入
  4. 运行回归测试

**理由：** Argo UI 是一个成熟稳定的项目（上游更新频率低）。最小化改动面意味着维护成本最低。标记补丁行号便于自动化 diff 和冲突检测。

### 6.6 监控与可观测性

分析 Argo UI 源码后确认的监控点：

| 监控点 | 优先级 | 说明 |
|--------|--------|------|
| `GET /api/v1/workflows` 透传 | P0 | 列表页关键路径 |
| `GET /api/v1/workflow-events` SSE | P0 | Watch 实时状态 |
| `GET /api/v1/workflows/*/log` SSE | P1 | 日志 streaming |
| `POST /api/v1/workflows/*/submit` | P1 | Pipeline 提交 |
| 401 跳转事件 | P0 | 认证过期 |
| webpack 构建失败 | P0 | 部署阻断 |

---

## 7. 风险点与缓解方案

| # | 风险 | 影响 | 概率 | 缓解方案 |
|---|------|------|------|----------|
| R1 | **argo-ui 包依赖**：Argo UI 依赖 `argo-ui` git 私有包（`git+https://github.com/argoproj/argo-ui.git#54f36c7`），下游组件（LogsViewer、Layout、Page、SlidingPanel 等）来自这个包。如果包不可安装或不兼容 Node 22，整个 Fork 无法运行。 | 阻塞 | 中 | **P1.2 优先验证**。先 `yarn install` 确认依赖可解析。如果失败，使用 `yarn pack` 离线保存 `argo-ui` 包快照；或升级到兼容版本。 |
| R2 | **SSE 透传复杂**：Argo Server 的 SSE 格式是 JSON 行 `{"result": {...}}\n`，Cyber Backend 透传时需要保持该格式，并保持 connection 不被中断。Go 的 `http.ResponseWriter` 需要 Flusher 支持。 | 高 | 中 | 先用轮询降级（30s 间隔 `GET /api/v1/workflows/{ns}/{name}`），保证功能闭环后再实现 SSE 透传。 |
| R3 | **Node version 兼容**：Argo UI 的 TypeScript 4.6 + webpack 5 在 Node 22 上可能有构建问题。 | 中 | 中 | P1.3 验证构建。如有问题，使用 `nvm` 切换 Node 20 LTS；或在 Dockerfile 中指定 Node 版本。 |
| R4 | **monaco-editor 构建体积**：monaco-editor (~20MB) 和 xterm 是 Argo UI 最重的依赖。首次加载可能慢（~5-8s）。 | 低 | 高 | webpack 的 `splitChunks` 已配置代码分割。monaco 本身有 `MonacoWebpackPlugin` 实现按需加载。验证构建产物大小，必要时进一步优化。 |
| R5 | **DAG 图字段缺失**：如果我们的透传端点遗漏了某些字段（如 `boundaryID`、`outboundNodes`），DAG 图可能渲染异常（缺失节点、边错误）。 | 高 | 低 | 后端透传端点直接返回 `wfv1.Workflow` 的完整 JSON（不需要手动映射字段）。测试时用真实 workflow 验证 DAG 图的正确性。 |
| R6 | **namespace 管理差异**：Cyber 可能是单 namespace，Argo UI 默认支持多 namespace 切换。 | 中 | 低 | Argo UI 识别 `managedNamespace` 信息，如果设置了自动隐藏 namespace 选择器。我们的 `/api/v1/info` 返回正确的 `managedNamespace` 即可。 |
| R7 | **Canvas iframe 跨域通信**：如果 Argo UI 和 Cyber 画布部署在不同端口，iframe 的 `postMessage` 需要验证 targetOrigin。 | 中 | 低 | Dev 环境用 nginx 统一入口（同一 origin 下无跨域问题）。Prod 环境同域部署。 |

---

## 8. 关键架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    Nginx / Load Balancer                     │
│                    databrew.example.com                       │
├──────────────────────┬──────────────────────────────────────┤
│    /argo-ui/*        │               /*                      │
│    (Argo UI SPA)     │     (Cyber Frontend SPA)              │
│                      │                                       │
│  ┌────────────────┐  │  ┌────────────────────────────────┐  │
│  │ Argo UI Fork   │  │  │ Cyber Frontend (Vite, React 19) │  │
│  │ (Webpack,      │  │  │                                 │  │
│  │  React 18,     │  │  │  /pipeline    → PipelinePage    │  │
│  │  argo-ui)      │  │  │  /assets      → AssetsPage      │  │
│  │                │  │  │  /dashboard   → DashboardPage   │  │
│  │ DAG 图         │  │  │                                 │  │
│  │ 日志查看 (xterm)│  │  │  ← iframe →  Pipeline Canvas   │  │
│  │ 节点面板       │  │  │     (postMessage 通信)          │  │
│  │ YAML 查看      │  │  │                                 │  │
│  │ Submit 页      │◄─┼──┤  └────────────────────────────────┘  │
│  │  (iframe 画布) │  │  │                                       │
│  └───────┬────────┘  │  │                                       │
│          │           │  │                                       │
└──────────┼───────────┴──┼───────────────────────────────────────┘
           │              │
           │ cookie/auth   │ cookie/auth
           ▼              ▼
┌──────────────────────────────────────┐
│        Cyber Backend (Go + Gin)      │
│                                      │
│  /api/v1/workflows/*    ←透传→ Argo Server REST API
│  /api/v1/workflow-events/* ←透传→ Argo Server SSE
│  /api/v1/auth/*         (session mgmt)
│  /api/v1/userinfo       (模拟 Argo userinfo)
│  /api/v1/info           (模拟 Argo info)
└──────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────┐
│         Argo Workflows Server        │
│   (Kubernetes CRD Controller)        │
└──────────────────────────────────────┘
```
