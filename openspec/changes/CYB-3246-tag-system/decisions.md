# Decisions — CYB-3246

## 2026-07-10 — OpenSpec 获批，开始 Phase 1
- **Context**: OpenSpec（proposal/design/spec delta/tasks）写完并过质量自检，用户在 checkpoint 回复「ok」确认。
- **Decision**: 进入 Phase 1 实现（后端开放词汇校验 + 前端标签展示优化）。Phase 2（`tag_registry` 表 + admin CRUD + Settings UI）作为独立 PR，动 off-limits `backend/migrations/` 前再单独取得批准。
- **Alternatives**: 一次性做完两个 Phase（PR 过大、前端优化被后端阻塞）。
- **Rationale**: 路线 C（用户选定）—— 先快速止血，大改造走独立 PR。

## 2026-07-10 — 开放词汇默认长度取 500
- **Context**: 未注册 key 放行后需要一个长度上限防止无界值。
- **Decision**: `DefaultUnregisteredTagMaxLength = 500`，用 `len(value)`（字节）计量。
- **Alternatives**: 无上限；按 rune 计数。
- **Rationale**: 与已注册 `notes` 标签的 `max_length: 500` 对齐；`len()` 字节计量与现有 string 分支校验完全一致，避免行为分叉。

## 2026-07-10 — 下游无需改动
- **Context**: 担心未注册 key 落库/入 ES 时 `tag_type` 缺失。
- **Decision**: 无需改动 —— `tagTypeFor()` 对未注册 key 已返回 `"string"`。
- **Rationale**: Phase 1 后端改动收敛到 `Validate()` 单函数，风险最小。
