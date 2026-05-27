# Pipeline 上线评估（一页纸 + 修复计划）

> **环境**: `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app`  
> **版本**: `v3caf41e (feat/pipeline-integration)`  
> **评估日期**: 2026-05-27  
> **读者**: PM / 技术负责人 / 发布决策

**配套文档**（细节勿重复读本页）：

| 文档 | 用途 |
|------|------|
| [pipeline-qa-report.md](./pipeline-qa-report.md) | 全功能测试矩阵、用户旅程、缺陷 ID |
| [pipeline-frontend-guide.md](./pipeline-frontend-guide.md) | 前端 JSON 契约、Phase A/B/C 实施步骤 |
| [pipeline-requirements.md](./pipeline-requirements.md) | 产品需求真源 |
| [CYB-1254 design](../openspec/changes/CYB-1254-pipeline-integration/design.md) | 集成架构 |

---

## 1. 能否上线？（决策摘要）

| 场景 | 结论 | 一句话 |
|------|------|--------|
| **内部演示 / MVP** | ⚠️ **可以，有前提** | 走「保存模板 → 部署 Tab 运行 → 流水线运行看状态」；勿演示画布直部署 |
| **对外产品 / 生产** | ❌ **不可以** | P0 未修 + 排障无日志 + 模板无法回画布编辑 |

**架构判断**：UI DAG → Pipeline JSON → Transpiler → Argo → K8s **方向正确**；当前瓶颈是 **前后端契约** 与 **产品闭环**，不是换技术栈。

**成熟度对比**：**流水线运行**（`/workflows`）> **流水线**（`/pipeline`）。监控侧比编辑侧更接近可交付。

---

## 2. 主路径是否走得通？

```text
✅ 推荐路径（演示/内测用这条）
  注册组件 → 画布编排 → 保存模板 → 部署 Tab「运行」→ 流水线运行 看 DAG/状态

❌ 常见预期路径（当前会翻车）
  画布编排 → 顶部「部署」→ 500（PG FK: template_id 空串）

⚠️ 排障路径（半成品）
  流水线运行 → Failed 详情 → 看节点状态 ✅ → 看 Pod 日志 ❌（API 有，UI 无）
```

| 旅程 | 状态 | 说明 |
|------|------|------|
| A：编排并跑通 | ⚠️ | 模板部署 + Argo 可见；缺日志、边契约未闭合 |
| B：画完立刻部署 | ❌ | **B1** 阻塞 |
| C：失败排障 | ❌ | **U5** 阻塞 |

---

## 3. P0 修复计划（发布前必做）

> 以下为**计划说明**，实施时对照 [pipeline-frontend-guide.md](./pipeline-frontend-guide.md) 与 QA 报告 §3，**不在本文档内改代码**。

### B1 — 画布直部署 PG FK 失败

| 项 | 内容 |
|----|------|
| **现象** | `POST /api/v1/deploy` 返回 500：`pipeline_deployments_template_id_fkey` |
| **根因** | 未关联模板时 `template_id` 以 `""` 写入，违反 FK；应为 SQL `NULL` |
| **改哪里** | `backend/internal/postgres/pipeline_repo.go`（`PipelineDeploymentRepo.Save`）；可选 `usecase/pipeline/usecase.go` 部署前不写空串 |
| **验收** | 画布 1+ 节点 → 顶部部署 → 200 + `workflowName`；`pipeline_deployments.template_id` 为 NULL |
| **产品备选** | 部署前强制「先保存模板」+ UI 文案（与 DB 修复二选一或叠加） |

### B2 — 导出 JSON 不符合 transpiler 契约

| 项 | 内容 |
|----|------|
| **现象** | 节点无 `inputs`/`outputs`；边为 `step-1` → `step-2`，非 `step-1.output` → `step-2.input` |
| **根因** | `PipelinePage` 的 `buildPipelineJSON` 与 React Flow 模型未映射到 transpiler |
| **改哪里** | 新增 `Frontend/src/lib/pipelineContract.ts`（`toTranspilerPipeline` / `fromTranspilerPipeline`）；`PipelinePage` 保存/导出/部署/导入统一走该模块 |
| **验收** | 导出 JSON 与 `backend/internal/transpiler/transpiler_test.go` golden 一致；模板部署后 Argo DAG 边与依赖正确 |
| **参考** | [pipeline-frontend-guide.md](./pipeline-frontend-guide.md) **Phase A** |

**P0 工时粗估**：后端 0.5d + 前端契约 1d + 联调回归 0.5d。

---

## 4. P1 修复计划（对外发布前强烈建议）

| ID | 问题 | 建议改动 | 验收 |
|----|------|----------|------|
| **U1** | 三处都叫「运行记录」 | 侧栏保持「算法运行记录」；部署 Tab →「部署记录」；`/workflows` 页 H2 →「流水线运行」 | 用户能区分三类列表 |
| **U2** | 模板无法加载回画布 | 部署 Tab 模板卡片「编辑」→ `GET /api/v1/pipelines/:id` + `fromTranspilerPipeline` | 改模板后保存再部署 |
| **U3** | 部署历史无跳转 Workflow | 运行历史卡片加「查看」→ `/workflows/{workflowName}` | 从部署 Tab 一步进 Argo 详情 |
| **U4** | 画布部署无选 asset | 复用模板部署的资产 Modal；`POST /deploy` 带 `asset_ids` | 与模板路径行为一致 |
| **U5** | Workflow 详情无日志 | `workflowApi` + 详情侧栏 Tab 调 `GET .../logs?nodeId=` | Failed 节点可看日志 |
| **U6** | 清空无确认 | `Modal.confirm` | 误点不丢图 |
| **U7** | Workflow 列表无分页/筛选 | Table 分页 + status 筛选 | workflow 多时可用 |

**P1 工时粗估**：3–5d（可与 P0 并行部分 U1/U3/U6）。

---

## 5. P2（可排期，不挡内测）

| ID | 摘要 |
|----|------|
| U8 | 空画布 onboarding |
| U9 | 组件库文案「Registry」→「组件 Tab」 |
| U10 | 连线 handle / 成功 toast |
| U11 | 导出 JSON 改 Drawer |
| U12 | 组件接 `/api/v1/components`，去 localStorage |

详见 [pipeline-qa-report.md](./pipeline-qa-report.md) §3 P2。

---

## 6. 发布门槛 Checklist

复制到 PR / Linear 发布项：

```markdown
### P0（阻塞）
- [ ] B1 画布直部署成功或产品明确禁止并引导先保存
- [ ] B2 导出/保存 JSON 通过 transpiler 契约（含 port 边）

### P1（对外强烈建议）
- [ ] U1 命名统一（三处「运行记录」）
- [ ] U2 模板加载到画布
- [ ] U3 部署记录 → Workflow 详情链接
- [ ] U5 Workflow 节点日志 UI
- [ ] U6 清空确认

### 回归
- [ ] 模板部署 → `/workflows` 出现 Running/Succeeded
- [ ] 空画布部署按钮 disabled
- [ ] 概览 ↔ `/pipeline` ↔ `/workflows` 路由
- [ ] dev 部署后按 docs/agents/deploy-verification.md 抽测
```

---

## 7. 信息架构（产品命名，建议一次定稿）

```text
侧栏「算法运行记录」     → /algo-runs          （算法任务，与 Pipeline 无关）
侧栏「流水线」           → /pipeline
  ├─ 画布               — 设计 DAG
  ├─ 组件               — 镜像注册表
  └─ 部署记录（原部署 Tab 标题）— 我的模板 + 我触发的 deployment

侧栏「流水线运行」       → /workflows          （集群 Argo Workflow，运维/排障）
```

避免再出现页内 H2「运行记录」与侧栏「流水线运行」不一致。

---

## 8. 与需求对齐（摘要）

| 能力 | 需求 | 当前 |
|------|------|------|
| F2 设计器 | 拖拽/连线/保存 | 拖拽保存 ✅；契约 ❌ |
| F3 执行 | Transpile + Argo | 模板路径 ✅；画布直部署 ❌ |
| F4 资产 | 运行前选 asset | 仅模板 ✅ |
| F5 监控 | 列表 + DAG + 日志 | 列表/DAG ✅；日志 UI ❌ |
| F7 模板 | CRUD + 编辑 | 列表/保存 ✅；加载编辑 ❌ |

---

## 9. 建议排期（仅供参考）

| 阶段 | 内容 | 目标 |
|------|------|------|
| **Sprint 1** | B1 + B2 + U1 + U6 | 内测可演示两条部署路径 |
| **Sprint 2** | U2 + U3 + U5 | 编辑闭环 + 排障闭环 |
| **Sprint 3** | U4 + U7 + P2 | 对外 Beta |

---

## 10. 测试环境备忘

- 登录：dev 需 `dev-token`（见 deploy-verification）
- 已观测 API：`POST /deploy` ❌ · `POST /deploy/template/:id` ✅ · `GET /workflows` ✅
- 全量用例表：[pipeline-qa-report.md](./pipeline-qa-report.md) §2

---

---

## 11. Dev 复验（2026-05-27 部署后）

**环境**: https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app  
**方式**: Chrome DevTools MCP + API 抽查

| 项 | 结果 | 说明 |
|----|------|------|
| B1 画布直部署 | ✅ | `POST /pipelines` → `POST /deploy/template/:id`；成功 `my-pipeline-efdd1e`，无 FK |
| B2 导出契约 | ✅ | 导出 JSON 含 `inputs`/`outputs`；库内模板边为 `step-1.output` → `step-2.input` |
| U1 命名 | ✅ | 部署 Tab「部署记录」；`/workflows` H2「流水线运行」 |
| U2 模板编辑 | ⚠️ | 有「编辑」按钮；**同页 `/pipeline` 点编辑不灌画布**（`sessionStorage` 仅在 mount 读）；刷新或从别页进入可加载 |
| U3 部署→Workflow | ✅ | 「查看」→ `/workflows/{workflowName}` |
| U6 清空确认 | ⚠️ | 源码有 `Modal.confirm`；自动化未捕获确认框，画布未清空（建议人工再点一次） |
| U5 日志 | ❌ | 详情页无日志 UI；`GET /api/v1/workflows/:name/logs` **404**（路由未注册到 dev backend） |
| U7 分页 | ❌ | 列表仍 `pagination={false}`，全量展示 |

**结论更新**：**内部 MVP / 演示可过**（推荐路径 + 画布直部署已通）；**对外**仍差 U5（需 backend 注册 logs 路由 + 前端面板）与 U7。

*本文档仅做发布决策与修复排期；实现细节以 QA 报告与前端指南为准。*
