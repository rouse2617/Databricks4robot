# Decisions — CYB-3246

## 2026-07-10 — Phase 2 修正：合并语义（修复校验被冲掉的 bug）
- **Context**: Phase 2 合并部署到 dev 后验证发现严重 bug —— admin 的 create/delete 循环触发 `refreshRegistry`，它用 `ReplaceTags(DB内容)` 整体替换内存 map；而 GHA migrate job 与后端启动 seed 有时序竞争导致 DB 未被 YAML 播种，结果内存里的 YAML enum 标签（如 priority）被清空，`priority=urgent` 从 422 退化为 200。
- **Decision**: 改为**合并语义**：`tags`（YAML 基线，永不清空）+ `managed`（DB 覆盖层），有效定义 = 基线被覆盖层叠加。取消 seed（DB 只存 admin 受管标签）；`SetManagedTags` 只换覆盖层。启动/刷新只加载覆盖层，DB 空或未迁移也不影响基线校验，彻底消除竞争与清空。admin LIST 返回基线(只读)+受管(可编辑)合并视图，带 `managed` 标记。
- **Alternatives**: (a) 幂等 re-seed 基线到 DB —— 仍受启动时序影响、且与用户删除语义冲突；(b) 保持整体替换但保证 seed —— 脆弱。
- **Rationale**: 基线永不进 DB、永不被覆盖层清空，是唯一对时序与空表都安全的模型。回滚：`SetManagedTags` 换回 `ReplaceTags` 语义即可，无数据影响。

## 2026-07-10 — 标签管理 UI 迁移到注册中心
- **Context**: 用户「都迁移到注册中心吧」。注册中心（RegistryCenterPage）已有只读 Tag 字典表。
- **Decision**: 将 `TagRegistryManager` 移入注册中心「平台字典」Tab 的 Tags 卡片（替换原只读表），并从 Settings 移除该卡片。组件增强：非管理员 401 时回退公共只读端点仍可查看（隐藏增改删），`managed=false` 的基线标签标「内置（只读）」。
- **Rationale**: 注册中心是各类注册表的统一入口，标签治理归口于此更合理；Settings 聚焦会话/搜索索引运维。

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

## 2026-07-10 — Phase 1 合并 + Phase 2 启动（off-limits 批准）
- **Context**: PR #341（Phase 1）合并到 dev（merge commit 87af5621），GHA deploy-dev 触发。用户指示「合并吧，并且做 Phase 2」。
- **Decision**: 启动 Phase 2。用户明确批准触碰 off-limits `backend/migrations/` 新建 `tag_registry` 表（此前已就「需你明确批准」当面确认，用户回复「做 Phase 2」）。PR 需第二审查者（off-limits 规则）。
- **Alternatives**: 用通用 settings/kv 表存注册表（避免新 migration）——被否，DB 表是本项目权威、可审计的标准做法。
- **Rationale**: 见 design.md Decision 2。迁移为纯新增表，回滚只需切回 YAML 加载。

## 2026-07-10 — Phase 1 部署验证在 merge 后
- **Context**: dev 部署改为 GHA push-to-dev 触发，无本地部署脚本。
- **Decision**: Phase 1 的 Chrome DevTools MCP + smoke 验证在 merge 后于 dev 执行（deploy-before-commit 与 GHA 门禁的冲突按现实调和）。
- **Rationale**: 结构上无法在 merge 前对 dev 部署验证。

## 2026-07-10 — 下游无需改动
- **Context**: 担心未注册 key 落库/入 ES 时 `tag_type` 缺失。
- **Decision**: 无需改动 —— `tagTypeFor()` 对未注册 key 已返回 `"string"`。
- **Rationale**: Phase 1 后端改动收敛到 `Validate()` 单函数，风险最小。
