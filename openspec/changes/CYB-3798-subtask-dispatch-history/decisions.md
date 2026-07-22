# CYB-3798 Decisions

## 2026-07-22 — SDK 覆盖暂缓（API contract sync row 3/4 waived）

- **Context**: 契约同步规则默认要求新增/变更公共 REST 面同步 SDK client（rows 3、4）。本次两个接口在 `/api/v1/backfill`，走 `X-Databrew-Token` admin 面，前端消费，非对外 SDK 面。
- **Decision**: 本 PR 不加 SDK client 方法；仍完成 rows 1（OpenAPI）、2（api-guide）、5（smoke）、7（spec delta）。
- **Alternatives**: 一并加 `sdk/.../backfill.py` —— 与本需求（UI 下钻）无关，扩大范围，违背最小化。
- **Rationale**: backfill 面当前 SDK 未覆盖；单独为这两个只读接口开 SDK 面没有真实消费方。若后续有 SDK 需求再单开 issue。

## 2026-07-22 — 通用 `createdBy` 过滤 vs 专用订阅任务接口

- **Context**: "查订阅任务的历史批次" 可做成 `GET /subscription-tasks/:id/batches`（语义专用）或 `GET /backfill?createdBy=`（通用过滤）。
- **Decision**: 通用 `createdBy` 过滤。
- **Alternatives**: 专用接口 —— 需 subtask handler 依赖 backfill usecase，跨用例耦合，代码更多。
- **Rationale**: 最小改动、可复用；`"subscription-task:<id>"` 拼接约定封装在前端一个函数内，泄漏面可控。
