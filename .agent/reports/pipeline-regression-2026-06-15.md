# Pipeline 回归测试报告 — 2026-06-15（dev 部署后）

> 报告人：AI 执行 Agent（chrome-devtools MCP 实操 + git diff/go test 代码审查）
> 环境：`http://localhost:5176/pipeline`（dev 后端，登录态 admin）
> 触发：dev 最新 10 commit（核心 CYB-2007 批次一致性 + CYB-2097 回填结果上传）部署后回归

---

## 一、回归结论：通过

| 页签 / 视图 | 结果 | 关键证据 |
|------|------|------|
| 设计（画布） | ✅ | 18 组件加载，画布 / 组件面板 / 缩放正常 |
| 流水线（模板列表） | ✅ | 搜索 / 范围 / 排序齐全，`GET /api/v1/pipelines` 200 |
| 执行记录 · 单次执行 | ✅ | 列表渲染，分页「共 9581 条 / 480 页」，`pipeline-runs?view=summary&excludeBatch=true` 200 |
| 执行记录 · 批量任务 | ✅ | 批次列表渲染，`GET /api/v1/backfill` 200 |
| 批次详情 | ✅ | 子任务 / 节点汇总 / 回填结果上传齐全 |
| 组件 | ✅ | 加载正常，零控制台错误 |

网络：所有 pipeline 相关 API 均 200。唯一 401 为登录前 `/auth/me` 预检（随后 login 200），属正常。批次详情 / 组件页零控制台错误。

---

## 二、CYB-2007 批次一致性：端到端验证通过

批次 `ui-100-cyb2007-1781494114854`（`b702e330-...`）详情页：

- 子任务数 = **100**（非 200，无重复）
- 成功 / 失败 = **100 / 0**
- 状态 = **completed**，进度条 **100%**
- 子任务列表分页：「**共 100 条 / 5 页**」← 与逻辑批量大小完全一致
- 子任务列表 API `pipeline-runs?view=summary&batchJobId=...&page=1&pageSize=20` 返回 200

旁证：`ui-100`（100/0/100→completed）、多个 1000 批次（1000/0/1000→completed）计数全部收敛到逻辑总数。

结论：CYB-2007 修复在 UI 端完全生效，此前代码 review 标记的 `ListSummaries` INNER JOIN 查询在正常路径返回 200 且总数一致。

---

## 三、发现的问题

### 问题 1：批量任务列表「批次名称」首列宽度过窄（UI / 低）

- **严重程度**：⚠️ 低（纯样式，不影响功能与数据）
- **位置**：执行记录 → 批量任务 列表，首列「批次名称」
- **现象**：列宽过窄，长批次名（如 `e2e-preview-echo-20260613-...`）逐字符竖排换行（`e / 2 / e / - / p / r ...`），可读性差
- **复现**：
  1. `/pipeline?tab=executions&executionView=batch`
  2. 观察首列长名称批次的换行表现
- **建议修复**：给该列设 `min-width`（如 220px）或 `white-space: nowrap` + `text-overflow: ellipsis`（配合 Tooltip 显示全名）
- **候选文件**：批量任务列表组件（`Frontend/src/components/backfill/` 或 `Frontend/src/pages/` 下批量任务列表 / BatchJobList 相关）

---

## 四、附：本次代码 review 留存的次要风险（非本次回归阻断）

`backend/internal/postgres/pipeline_repo.go` `PipelineRunRepo.ListSummaries` 的 batch 分支使用
`INNER JOIN pipeline_runs pr ON pr.id = bi.pipeline_run_id`。
若 `UpsertBatchSubtaskRun`（best-effort，失败仅 warn + continue）对某 item 失败，该 item 无 `pipeline_run_id`，会被 INNER JOIN 丢弃，导致批次明细列表 + count 少于逻辑总数（job 状态 / completedCount / runsTotal 不受影响，有 `job.TotalCount` 兜底）。
属窄边界（仅 upsert 失败触发）。建议改 LEFT JOIN 并为缺 run 的 item 投影占位行，或在 decisions 显式记录为已知取舍。

---

## 五、全流程 e2e（已落地）

新增 `Frontend/e2e/pipeline-full-flow.spec.ts`，覆盖 CYB-2007 关注链路：
**设计（导入流水线 JSON）→ 保存为模板 → 部署对话框选 ≥2 资产 → 批量下发（POST /backfill）→ 跳转批次详情 → 子任务数收敛到资产数**。

- **mock 用例**（确定性、CI 可跑、不依赖后端）：`route` 拦截全部 `/api/v1/**`，断言
  - 保存 POST body `name == e2e-fullflow-pipeline`；
  - 批量 POST `/backfill` body `templateId + assetIds` 正确；
  - 跳转 `/pipeline/batch/:id`，批次详情渲染「批次 ID / 子任务执行记录」且子任务数 == 资产数。
  - ✅ 实测 `1 passed (4.0s)`。
- **@integration 变体**：真连 dev，校验 `GET /backfill/{id}.totalCount == 资产数` 且 `pipeline-runs?batchJobId.total <= 资产数`；无 `E2E_DATABREW_TOKEN` 自动 skip（✅ 实测 `1 skipped`）。
- 关键踩坑：① antd 图标按钮中文双字自动插空格（"导 入"），选择器用 `/导\s*入/`；② AssetPicker 校验资产会请求 `GET /api/v1/assets/{id}`，未 mock 时打真后端返 401 → `UNAUTHORIZED_EVENT` 登出到登录页 → 已加 `/api/v1/**` 兜底路由拦截。

运行：
```bash
cd Frontend
npx playwright test pipeline-full-flow                          # 仅 mock
E2E_DATABREW_TOKEN=xxx npx playwright test pipeline-full-flow   # 含 integration
```

## 六、下一步

1. 修复问题 1（批量列表首列列宽）。
2. e2e 已就绪，可纳入 CI（mock 用例无需后端）。
