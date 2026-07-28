# CYB-4338 decisions

## 2026-07-28 — 跳过合并前 dev 部署 + MCP 验证(前端已本地 MCP,后端顺延 GHA)

- **Context**: 本 PR 含 `Frontend/` + `backend/` 运行时改动,按 deploy-before-commit
  应先部署 backend dev + frontend dev 再由 Agent 跑 MCP 验收。用户明确指示「完成了?
  你来提交 pr 吧」,即跳过合并前部署直接开 PR。
- **Decision**: 跳过合并前部署。前端 ECharts 重构/配额池移除已在**本地 vite(连 Cloud Run
  dev 后端)+ Chrome DevTools MCP** 验证(canvas 渲染、KPI+Divider、配额池消失、无 console
  error)。后端 10 桶属 API 契约变更,合并到 dev 后由 `deploy-dev.yml` 自动部署,届时
  10 桶渲染 + 契约 smoke 顺延补验。
- **Alternatives**:
  - (A) 合并前部署 backend dev(`local-build-deploy.sh`)+ frontend dev(`wrangler`)再 MCP
    —— 完全合规,但用户要求直接提。
  - (B) 只本地不部署 —— 后端 10 桶本地前端看不到(本地连的是 dev 旧 5 桶后端),已如实告知用户。
- **Rationale**:
  - Rule precedence 优先级 1(用户当前消息显式指示)> 优先级 4(deploy-before-commit)。
    用户是平台 owner,已知情裁定,且明确表示 MCP 自己复核。
  - 后端改动是**只读 SQL 分桶 + 契约**,`go test ./...` 全绿、契约同步齐,回归风险低。
  - dev 为自动部署环境,合并即 GHA build+deploy;PR 注明 MCP 顺延,可追溯。

## 2026-07-28 — P50/P90 桶高亮延后

- **Decision**: 分类轴无法精确画竖线;折中"高亮落入桶"作为 follow-up,不阻塞本 PR。
- **Rationale**: 用户两次要求直接提 PR;该项是我自标的"可选进阶亮点",非方案A核心。已在
  proposal/tasks Non-goals + PR 描述明示,owner 要则一句话补。
