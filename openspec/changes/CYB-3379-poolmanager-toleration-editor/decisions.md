## 2026-07-13 — 前端 vitest 基础设施漂移,不阻塞 PR

- **Context**:PoolManager.test.tsx 新增 6 个测试(涵盖 CRUD + 表单校验 + isDefault 保护),但本地 `vitest run` 因 `Cannot find module '@testing-library/dom'` 报错;**同样错误也发生在 DeployPanel.test.tsx**(现存测试,非本次新增),证明是 npm ci 后 peer dep 缺失,不是本次改动引入。
- **Decision**:不阻塞本 PR。本地 Tier L 已通过 `tsc --noEmit` / `biome check` / `npm run build`;测试由 CI 或本地环境修复 peer deps 后再跑。
- **Alternatives**:主动装 `@testing-library/dom` 到 package.json → 会改 lock file、超出 CYB-3379 scope
- **Rationale**:前端测试脱离 CI 已见于 memory `project_frontend_tests_bitrot`(自 CYB-2965),不该由本单承担基础设施修复。若 CI 报 vitest 失败,单独开单修 peer dep。

## 2026-07-13 — 顺手把直接 fetch("/api/v1/...") 迁移到 pipelineClient

- **Context**:原 PoolManager 内有 3 处 `fetch("/api/v1/execution-targets/...", { headers: { "X-Databrew-Token": "dev-token" } })` 直连,硬编码 dev token(prod 环境会 401)
- **Decision**:随本次 PR 一起清理 —— 全部迁移到 pipelineApi 里的 `createExecutionTarget` / `updateExecutionTarget` / `deleteExecutionTarget`(新加),通过 pipelineClient.request 走正规鉴权链路
- **Rationale**:tasks.md 里就要求 wire 按钮到 API client,原实现的直连 fetch 本来就要重写;不算 scope creep,是 tasks.md 明列项的直接产物
