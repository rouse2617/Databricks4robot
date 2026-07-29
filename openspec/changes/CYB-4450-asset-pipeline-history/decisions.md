# Decisions — CYB-4450

## 2026-07-29 — 复用 CYB-4297 endpoint，不写新 API

- **Context**: 调研后发现 `GET /api/v1/assets/:id/runs` 已经在 CYB-4297 落地，调用 `pipelineHandler.ListRunsByAsset` 反查 `pipeline_runs.asset_ids`（GIN index）；既有的 `RunsTab` + `listPipelineRunsByAsset` 已经覆盖用户要的「资产 → 运行记录」场景，只是 tab label 叫 `运行历史`。
- **Decision**: 不新写 handler、不动 OpenSpec、不动 api-guide；只删「算法处理」tab + 改 `运行历史` → `运行记录`。
- **Rationale**: 唯一的真实变化是：「算法」概念退役。原来由 RunsTab 提供的 UI 已经满足需求——流水线、状态、批次、目标、进度、优先级、开始/结束时间。没有功能性 gap。
- **How to apply**: PR 的 diff 几乎只是前端树的删减。

## 2026-07-29 — 第一版不做 step 展开和镜像

- **Context**: 用户提到「被哪些 step 处理过 / 被哪些镜像处理过」。镜像列不在 `pipeline_runs`（只有 jsonb `pipeline_json`），step 粒度要 JOIN `pipeline_run_nodes` + `pipeline_run_asset_nodes`。
- **Decision**: 第一版只做 run 级，step 和镜像另开 follow-up。
- **Rationale**: Run 级是大多数用户用例的最小必要信息；step 展开 + 镜像可以作为 V2 单独跟踪。
- **How to apply**: 不动后端 SQL，UI 列只展示 Run 概览。

## 2026-07-29 — 保留后端 `startAlgo` / `listAlgoEvents` handler

- **Context**: 这些 handler 还在 SDK / 内部脚本路径上用到。删除会断连接。
- **Decision**: 后端 handler 不动，只删前端 AlgoTab 引用。
- **Rationale**: 避免不必要的破坏面。
- **How to apply**: 这次改动不下任何 `backend/internal/handlers/asset/algo_handler.go`。

## 2026-07-29 — V2 follow-ups（不在此 PR 范围）

- 资产详情运行记录按模板折叠（一个模板一段，可展开看每次 run）
- step 粒度（用 `pipeline_run_asset_nodes` JOIN `pipeline_run_nodes`）
- 镜像信息（可能从 `pipeline_json` JSONB 解析，或单独建表）

