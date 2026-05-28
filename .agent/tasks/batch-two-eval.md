# 任务：批次二评估 + 启动

## 背景
批次一（资产模型 Phase 1 + 统一检索 P1）已完成部署验证。批次二范围：
1. **AlgoRun MVP Phase 1** — 训练产出自动资产化 + 输入血缘
2. **数据产线 MVP** — 多阶段产线编排（采集→转换→清洗→注册）
3. **资产模型 Phase 2** — ML Model + EvaluationReport 类型注册

用户要求先评估当前状态，确认启动路线。

## 任务步骤

### 1. 评估当前代码状态
- 读 `docs/review/algorn-enhancement.md` 的 Phase 1 MVP 范围
- 读 `docs/review/data-production-line.md` 的 §12 MVP 实施建议
- 读 `docs/review/asset-model-expansion.md` 的 Phase 2 范围
- 读 `docs/review/use-cases.md` 查看批次二相关的 Phase 2 用例
- 检查当前 `backend/routes/routes.go` 中 algo-runs 和 production-line 相关的路由注册状态
- 检查现有 migration 文件编号（看最新编号，确定新 migration 从哪开始）

### 2. 评估已有实现与缺口
- AlgoRun 当前已有哪些：POST/GET algo-runs、/start、/finish、/affected-assets
- Pipeline 当前已有哪些：模板、部署、Workflow 监控、资产联动
- 需要新增什么表和接口

### 3. 输出路线建议
在 `docs/review/batch-two-roadmap.md` 中写：
- 批次二范围概览（三个子项）
- 各子项依赖关系（先做 AlgoRun 还是先做产线？可并行？）
- 建议实施顺序
- 评估的工作量（人日估算）
- 风险点（如 ES 不可用对搜索的影响已暴露）
- 建议的 worktree 拆分方案

## 参考文档
- `docs/review/algorn-enhancement.md`
- `docs/review/data-production-line.md`
- `docs/review/asset-model-expansion.md`
- `docs/review/use-cases.md`
- `backend/routes/routes.go`
- `backend/migrations/`（最新 migration 编号）

## 输出
写入 `docs/review/batch-two-roadmap.md`
