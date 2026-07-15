# CYB-3300 — prod 迁移安全（无回滚）设计

> 本文件是**设计草案 + checkpoint**：给出选项与推荐，待选型后再拆分实现 PR。不含实现（触碰 migrations off-limits + 部署基建，需 team 决策 + 二审）。

## 问题（部署审计 P0-1）
- `deploy-prod.yml:53-55`：`deploy-backend needs deploy-migrate` → **迁移先于后端部署执行**。
- `backend/atlas/Dockerfile.migrate`：`atlas migrate apply`，**forward-only，无 down 迁移**。
- `deploy-prod.yml:131-145`：`gcloud run deploy` 直接路由到新 revision，**无 health/smoke gate**。

后果：一个不向后兼容的迁移，会让**仍在服务的旧 revision** 立刻打在已变更的 schema 上（migrate 与新代码之间的窗口），且**无脚本化回滚**，只能手动 DB 手术。

## 关键约束
- Atlas 迁移跨文件**非事务**、down 支持有限 → 「down 迁移回滚」不可靠，不作为主方案。
- 生产是 CloudSQL Postgres，有 PITR/备份能力（需确认已开启与保留期）。
- dev 已有迁移门禁（CYB-3299）；本条聚焦 prod 安全。

## 选项
### 方案 1（推荐主线）：expand/contract 迁移纪律 + CI 破坏性检测
- 规范：所有迁移必须**向后兼容**（先 expand：加列/加表/加可空；contract：删列/改类型 拆到「代码已上线后」的后续迁移）。使**新旧代码都能在当前 schema 运行** → migrate-before-deploy 的窗口不再危险，天然「无需回滚」。
- 落地：`atlas migrate lint` 的 destructive-change 检测在 **PR→main** 设为**必需**（现有 db-migrate-lint 已有 Pro lint，改为对破坏性变更 fail；非 Pro 也可用内置分析器）。破坏性迁移需显式标注/审批。
- 成本：低—中；主要是流程 + 一个 CI 门禁。收益最大、最治本。

### 方案 2（backstop）：迁移前自动快照/PITR 校验
- migrate Job 执行前，确认 CloudSQL 自动备份/PITR 可用（或触发一次按需备份），记录可恢复点；失败即 PITR。
- 成本：中；需 gcloud sql 权限接入 CI。作为「真出事」的兜底，不替代方案 1。

### 方案 3（补充）：prod 流量切换前 smoke gate
- 新 revision 部署后、`update-traffic` 前，跑健康冒烟（/healthz + 关键只读端点）；失败则不切流量（对齐 dev 已有的「migrate 成功才切流」做法）。
- 成本：低;直接降低「坏部署切到 prod」风险。

### 方案 4（不推荐）：down 迁移
- Atlas 跨文件非事务 + down 支持弱 → 回滚本身可能半途失败，风险高。仅在个别可逆迁移手写，不作为通用机制。

## 推荐组合
**方案 1（expand/contract + CI 破坏性门禁）为主** + **方案 3（smoke gate）** 立即降险 + **方案 2（PITR 校验）** 作兜底。分三个独立 PR，按 1→3→2 顺序。

## Checkpoint（待你选型）
1. 主线是否采用 expand/contract + 破坏性迁移 CI 门禁？
2. 是否要我先做**方案 3（prod smoke gate）**——最小、最快、纯部署 yaml，能立即降险（仍需二审，因改 deploy-prod）。
3. 方案 2 是否有 CloudSQL 备份/PITR 权限可接入 CI？

## Impact（实现阶段，非本文件）
- 方案 1：`.github/workflows/db-migrate-lint.yml`（门禁）+ 迁移规范文档。⚠️ 关联 migrations 语义。
- 方案 3：`.github/workflows/deploy-prod.yml`（加 smoke 步骤）。⚠️ 改 prod 部署,二审。
- 方案 2：deploy-prod + gcloud sql。⚠️ 权限 + prod。
