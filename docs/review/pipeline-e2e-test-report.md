# Pipeline 全功能 E2E 测试报告

> **环境**: https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app
> **对比环境**（曾可用 pipeline/deployments API）: `cyber-databrew-frontend-dev-234851712830.us-central1.run.app`
> **时间**: 2026-05-28
> **方式**: Chrome DevTools MCP + Network 抓包 + 代码审查
> **结论**: **前端 UI/契约层大幅改进，但当前 dev 后端未注册 pipeline/workflow 路由，全链路无法在本 URL 跑通。**

---

## 问题汇总

按优先级汇总历次巡检（含 UI/UX、产品、数据、环境）。修复状态以 **2026-05-28** 代码与 `wtttm6suaq` dev 实测为准。

### P0 — 阻塞（不修无法宣称 E2E 通过）

| ID | 问题 | 现象 | 建议修复 |
|----|------|------|----------|
| **P0-1** | **Backend 未注册 pipeline/workflow 路由** | `GET/POST /pipelines`、`GET /deployments`、`GET /workflows` 均 **404**（`404 page not found`）；仅 `GET /components` 为 200 | 查 backend-dev 日志是否 `k8s client unavailable`；确保 `pipelineHandler` / `workflowHandler` 非 nil；PG migration `039` 已应用；frontend 反代到正确 backend revision |
| **P0-2** | **全链路无法验收** | 保存、画布部署、部署 Tab 列表、流水线运行页均无数据或提交失败 | P0-1 修复后按 [§5 推荐验收流程](#5-推荐验收流程后端就绪后) 重跑 |

### P1 — 功能 / 数据（环境通后仍影响正确性）

| ID | 问题 | 现象 | 建议修复 |
|----|------|------|----------|
| **P1-1** | **组件重复（Pass Through ×2）** | API 返回 `0a7df9f5-...` 与 `sys-pass-through` 两条；侧栏/注册表各显示一次 | Seed 幂等合并；或 UI 对 `source=system` 按 name 去重 |
| **P1-2** | **镜像展示 `image:tag` 重复拼接** | 侧栏/画布显示 `busybox:latest:latest`、`python:3.12-slim:3.12-slim` | 修 `apiToRegistered`：`image` 已含 tag 时不再拼 `:${tag}`；展示层统一 `formatImage(image, tag)` |
| **P1-3** | **部分组件 ports 为空** | `Python Script` 等 `inputPorts`/`outputPorts` 为 `[]` | DB/seed 保证每组件至少 `input` + `output`；创建组件表单默认值必填 |
| **P1-4** | **画布手工连线未回归** | 导入/导出 JSON 契约正确（`step-1.output → step-2.input`）；鼠标拖 handle 连线是否生成 port 边未在 UI 自动化中验证 | 手工拖线 + 导出 JSON 断言；空态文案引导连线 |
| **P1-5** | **历史：画布直 deploy 写空 template_id** | 旧逻辑 `POST /deploy` 且 `template_id=""` 触发 PG FK（`pipeline_deployments_template_id_fkey`） | 代码已改为 **先 `savePipeline` 再 `deployTemplate(saved.id)`**；需在 API 200 环境复测确认 |

### P2 — UI / UX（不挡开发，影响体验与观感）

| ID | 问题 | 现象 | 建议修复 |
|----|------|------|----------|
| **P2-1** | **拖入画布节点显得过大** | 侧栏卡片紧凑，画布节点明显更大、易重叠 | `.pipeline-node { min-width: 180px }` 接近侧栏宽 210px；节点为双行结构（header+body）+ 左右 Handle；长镜像 `word-break: break-all` 撑高。建议 `min-width` 降至 ~140px、镜像 `ellipsis`、与 palette 尺寸 token 对齐 |
| **P2-2** | **空组件库文案过时** | 0 个组件时提示「请在 **Registry** 中添加」 | 改为「请在 **组件** Tab 新建」 |
| **P2-3** | **画布空态无引导** | 仅点阵网格，新用户不知拖组件、连线 | 增加简短 onboarding：「从左侧拖入组件 → 连接圆点 → 保存/部署」 |
| **P2-4** | **部署 Tab 内标题仍为「运行记录」** | 与侧栏「流水线运行」并存，语义略混 | 部署 Tab 内改为「已保存模板 / 部署历史」 |
| **P2-5** | **API 失败时前端静默** | `listPipelines` / `listDeployments` 失败时列表为 0，无明确「后端未启用流水线」提示 | DeployPanel / PipelinePage 对 404 显示 `Alert` + 文档链接 |
| **P2-6** | **a11y** | Chrome Issues：部分表单缺 `id`/`name`、label 未关联（节点配置、新建组件） | 为 Input 补 `id` + `htmlFor` |

### 已修复 / 已改进（本次相对早期 dev）

| ID | 项 | 说明 |
|----|-----|------|
| ✅ | 侧栏「运行记录」重复 | 已分为 **流水线** / **流水线运行** |
| ✅ | 概览 → 流水线 URL 变内容不变 | `App.tsx` `Suspense` + `key={location.pathname}` 后复测通过 |
| ✅ | Tab 命名 | **画布 / 组件 / 部署** 分段 |
| ✅ | JSON 契约 | `toTranspilerPipeline` 导出含 `inputs`/`outputs` 与 port 边 |
| ✅ | 清空画布 | `Modal.confirm` 二次确认 |
| ✅ | 部署弹窗 | 节点/连线统计 +「高级：绑定资产」折叠 |
| ✅ | 组件源 | 从 localStorage 改为 **GET /api/v1/components** |
| ✅ | 部署逻辑 | **先存模板再 deployTemplate**（避免空 template_id） |

### 问题与测试矩阵对照

| 问题 ID | 测试项 |
|---------|--------|
| P0-1 | A9、A10、C1–C5、D4–D5 |
| P1-1、P1-2 | A2、B4 |
| P1-4 | A12 |
| P2-1 | A3（拖拽观感） |
| P2-5 | C1、C2、D4 |

---

## 1. 环境阻塞（必须先修）

与 [P0-1](#p0--阻塞不修无法宣称-e2e-通过) 相同，细节如下。

| API | 状态 | 影响 |
|-----|------|------|
| `GET/POST /api/v1/components` | **200** | 组件注册表可用 |
| `GET /api/v1/pipelines` | **404** | 无法列/存模板 |
| `GET /api/v1/deployments` | **404** | 部署 Tab 永远空 |
| `POST /api/v1/deploy/template/:id` | **404**（推断） | 无法部署 |
| `GET /api/v1/workflows` | **404** | 流水线运行页空 |

响应体为 Go 默认 `404 page not found`，说明 **路由未注册**（常见于 backend 启动时 K8s/Argo client 未就绪 → `pipelineHandler` / `workflowHandler` 为 nil）。

**修复方向**（运维/后端）：

1. 确认 Cloud Run **backend-dev** 与 frontend 指向同一 revision，且日志无 `k8s client unavailable`。
2. 确认 `backend/routes/routes.go` 中 `pipelineHandler`、`workflowHandler` 非 nil。
3. 确认 migration `039_pipeline_tables.sql` 已应用在 dev PG。
4. 复测：浏览器 Network 中 `GET /api/v1/pipelines` 应为 **200**。

在此之前，**产品 E2E「设计 → 保存 → 部署 → 看运行」无法在本环境验收通过**。

---

## 2. 测试矩阵与结果

### A. `/pipeline` — 画布

| # | 功能 | 结果 | 备注 |
|---|------|------|------|
| A1 | 页面加载、侧栏高亮 | ✅ | 画布/组件/部署 分段清晰 |
| A2 | 组件库从 API 加载 | ✅ | 3 个组件（见 P1-1、P1-2） |
| A3 | 拖拽组件到画布 | ✅ | 节点偏大，见 P2-1 |
| A4 | 节点选中 + 右侧配置 | ✅ | 名称/镜像/命令/资源 |
| A5 | 导入 JSON（两节点+边） | ✅ | 名称变为 `e2e-test-chain` |
| A6 | 导出 JSON 契约 | ✅ | 含 `inputs`/`outputs`，边为 `step-1.output → step-2.input` |
| A7 | 部署弹窗（节点/连线统计） | ✅ | 显示 2 节点 · 1 连线 |
| A8 | 部署弹窗「绑定资产」折叠 | ✅ | 高级可选，产品合理 |
| A9 | 画布「部署」提交 | ❌ | `POST /pipelines` → **404**（P0-1） |
| A10 | 「保存」 | ❌ | 同上 |
| A11 | 「清空」确认 | ✅ | `Modal.confirm` |
| A12 | 画布连线（鼠标拖 handle） | ⚠️ | 未自动化；契约 OK，见 P1-4 |

### B. `/pipeline` — 组件

| # | 功能 | 结果 | 备注 |
|---|------|------|------|
| B1 | 列表展示 | ✅ | |
| B2 | 新建组件表单 | ✅ | UI 正常 |
| B3 | 新建持久化到 API | ⚠️ | 未单独 POST 验证 |
| B4 | 镜像展示 | ❌ UX | P1-2 |

### C. `/pipeline` — 部署记录

| # | 功能 | 结果 | 备注 |
|---|------|------|------|
| C1 | 已保存模板列表 | ❌ | API 404 → 显示 0（P0-1、P2-5） |
| C2 | 运行历史 | ❌ | API 404 → 显示 0 |
| C3 | 模板「运行」+ 选资产 | ⬜ | 阻塞 |
| C4 | 模板「编辑」加载画布 | ⬜ | 阻塞 |
| C5 | 删除模板/部署 | ⬜ | 阻塞 |

### D. `/workflows` — 流水线运行

| # | 功能 | 结果 | 备注 |
|---|------|------|------|
| D1 | 列表页标题 | ✅ | 「流水线运行」 |
| D2 | 状态筛选 | ✅ UI | 无数据时可交互 |
| D3 | 刷新 | ✅ | |
| D4 | 列表数据 | ❌ | `GET /workflows` 404（P0-1） |
| D5 | 进入详情 / DAG / 日志 | ⬜ | 阻塞 |

### E. 导航与全局 UX

| # | 功能 | 结果 | 备注 |
|---|------|------|------|
| E1 | 概览 → 流水线 | ✅ | |
| E2 | 流水线 vs 流水线运行 | ✅ | |
| E3 | Console 报错 | ✅ | 无 JS error |

---

## 3. 与上次对比（改进项）

| 项 | 上次 | 本次 |
|----|------|------|
| Tab 结构 | 流水线设计/自定义组件/运行记录 | **画布/组件/部署** |
| 部署 | 直接 POST deploy，FK 失败 | **先 save 再 deployTemplate**（P1-5） |
| JSON 契约 | 无 inputs/outputs | **toTranspilerPipeline** |
| 清空 | 无确认 | **Modal.confirm** |
| 部署弹窗 | 无选资产 | **折叠「绑定资产」** |
| 组件源 | localStorage | **GET /components** |
| 侧栏 | 两个「运行记录」 | **流水线 / 流水线运行** |
| 路由 | URL 变内容不变 | **已修复**（见问题汇总 ✅） |

---

## 4. 问题细节说明（与代码位置）

### 画布节点偏大（P2-1）

- **CSS**: `Frontend/src/styles/pipeline.css` — `.pipeline-node { min-width: 180px; }`，侧栏仅 210px 宽。
- **结构**: `PipelineNode.tsx` 为 header + body 双行，侧栏 `.palette-item` 为单行紧凑卡片。
- **长镜像**: `.node-info { word-break: break-all; }` 使 `busybox:latest:latest` 折行拉高。

### 镜像重复 tag（P1-2）

- **代码**: `PipelinePage.tsx` — `apiToRegistered()` 在 `api.tag` 存在时拼 `` `${api.image}:${api.tag}` ``，若 DB 中 `image` 已含 `:latest` 则重复。

### 组件重复（P1-1）

- **数据**: `GET /components` 同时返回用户 seed 与 `sys-pass-through` 两条 Pass Through。

### 契约与连线（P1-4）

- **代码**: `Frontend/src/lib/pipelineContract.ts`（或同目录）— `toTranspilerPipeline` / `fromTranspilerPipeline`。
- **验证**: 导入 golden JSON 后导出，edges 为 `step-1.output → step-2.input` ✅。

---

## 5. 推荐验收流程（后端就绪后）

按顺序跑一遍即可证明「全链路 OK」：

```text
1. 组件 Tab：确认 Pass Through 唯一；新建一个 custom 组件
2. 画布：拖 2 个组件 → 手工连线 → 导出 JSON 确认 edges（含 port）
3. 保存：POST /pipelines 201
4. 部署：弹窗部署（可选绑 1 个 asset）→ POST deploy/template 201
5. 部署 Tab：运行历史出现新记录，状态 Pending/Running
6. 流水线运行：/workflows 出现同名 workflow，状态 Succeeded
7. 详情：点「查看」→ DAG → 某节点看日志
8. 回归：模板「编辑」回画布 → 再部署
```

**通过标准**：无 404/5xx；至少 1 次 workflow `Succeeded`；UI 每步有成功 toast 或可见状态变化；画布节点尺寸可接受（P2-1 可选优化）。

---

## 6. 产品层面结论

| 维度 | 评价 |
|------|------|
| **信息架构** | OK — 设计器 / 运行监控分离清楚 |
| **核心契约** | OK — 导出 JSON 已对齐 transpiler |
| **部署体验** | 设计 OK — 先存模板再部署、可选资产；**环境未通** |
| **可运维性** | 待验证 — 依赖 workflows API + K8s |
| **当前能否交付** | **仅适合 demo 画布+组件**；**不能宣称 E2E 已通** |

---

## 7. 建议你本地/CI 加的自动化

| 层级 | 内容 |
|------|------|
| 单元 | `pipelineContract.test.ts` golden JSON |
| API smoke | `POST /pipelines` → `POST /deploy/template/:id` → `GET /deployments` |
| UI E2E | Playwright：导入 golden → 部署 → `/workflows` Succeeded（需 dev K8s） |
| 视觉 | 画布节点宽度 ≤ 侧栏卡片视觉宽度（可选 snapshot） |

---

## 附录 A：导出 JSON 片段（契约正确）

```json
"edges": [
  { "source": "step-1.output", "target": "step-2.input" }
],
"nodes": [
  { "id": "step-1", "inputs": [...], "outputs": [...], "component": {...} },
  { "id": "step-2", "inputs": [...], "outputs": [...], "component": {...} }
]
```

---

## 附录 B：相关文档

- [pipeline-frontend-guide.md](./pipeline-frontend-guide.md) — 前端契约与分期改造
- [pipeline-requirements.md](./pipeline-requirements.md) — 产品需求与能力矩阵
