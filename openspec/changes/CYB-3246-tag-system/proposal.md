# Proposal — CYB-3246

## Why

新增标签必须改 `tag_registry.yaml` + 重启后端才能生效（自定义 key 直接被 `INVALID_TAG` 拒绝），运维无法自助；同时资产详情页标签展示被截断、来源信息混乱、无法查看全文或安全删除。标签是全文检索的主要内容载体，这两点直接拖累搜索可用性与日常运维。

## What Changes

### New Capabilities
- **asset-tagging：开放词汇标签** — 未注册的 key 作为自由字符串标签被接受并落库/入索引，用户无需改配置即可为资产打任意语义标签。
- **asset-tagging：标签注册管理** — 管理员可在 UI 界面注册 / 编辑 / 删除受管标签（enum 型及其允许值），变更即时生效，无需改 YAML 或重启后端。

### Modified Capabilities
- **asset-tagging：标签写入校验** — 校验行为从「未注册即拒绝」改为「未注册按自由字符串放行，已注册按定义（enum 值 / string 长度）严格校验」。
- **asset-management：标签展示** — 资产详情页标签项支持超长省略 + 展开查看全文、来源信息以 Badge 呈现、删除前确认。

## Impact
- **Affected code**:
  - `backend/internal/config/tag_registry.go`（Phase 1 校验放开；Phase 2 DB 加载 + reload）
  - `backend/internal/handlers/admin/`（Phase 2 新增 tag-registry CRUD handler）
  - `backend/routes/routes.go`（Phase 2 挂载 admin 路由）
  - `backend/migrations/`（Phase 2 新建 `tag_registry` 表 — **off-limits，需批准**）
  - `Frontend/src/components/assets/`（Task A 标签展示组件）
  - `Frontend/src/pages/SettingsPage.tsx` + `Frontend/src/api/`（Phase 2 标签管理界面 + 类型化 client）
- **New APIs**（Phase 2）:
  - `GET /api/v1/admin/tag-registry`
  - `POST /api/v1/admin/tag-registry`
  - `PATCH /api/v1/admin/tag-registry/:key`
  - `DELETE /api/v1/admin/tag-registry/:key`
- **Dependencies**: 无新增第三方依赖。

## Scope
- **In scope**:
  - Phase 1：后端开放词汇校验 + 前端标签展示优化（Task A）
  - Phase 2：`tag_registry` 表、admin CRUD API、Settings 标签管理界面、启动 YAML→DB seed + 热加载
- **Out of scope**:
  - `tag_sources[]`（来源契约）改为 DB 管理 — 本次仍读 YAML
  - 标签 ACL / writable_by 权限（历史 P1.5，未启用）
  - SDK 覆盖 admin tag-registry（admin 内部面，非公共 SDK 面；见 decisions.md）
  - 标签的批量迁移 / 历史数据回填

## Success Criteria
- [ ] Phase 1：通过 `POST /assets/:id/tags` 或 UI 用任意未注册 string key（如 `description`）打标签返回 200，且能在资产详情页与全文检索中读到。
- [ ] Phase 1：已注册 enum 标签（如 `priority`）写非法值仍返回 422，行为不回退。
- [ ] Phase 1：标签文本超长自动省略、可点击展开全文；删除有确认；来源信息清晰。
- [ ] Phase 2：管理员在 Settings「标签管理」界面新增一个 enum 标签后，无需重启即可在资产打标签时使用其允许值校验。
- [ ] Phase 2：admin CRUD 四个端点 OpenAPI / api-guide / smoke 齐全并通过。

## Goals (SLO)
- **Latency**: 标签校验为内存 map 查询，p99 < 5ms（DB 仅在启动 seed 与 admin 写时访问）。
- **Compatibility**: Phase 1 对现有已注册标签零行为回退；Phase 2 DB 为空时自动从 YAML seed，回滚只需切回 YAML 加载。
- **Quality**: 后端改动包（`config`、`handlers/admin`）单测覆盖新分支；至少 1 条 happy-path + 1 条 error-path smoke。
