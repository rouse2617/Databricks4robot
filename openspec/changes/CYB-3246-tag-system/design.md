# Design — CYB-3246

## Architecture Context
- **Constraints**:
  - Go 1.25+；标签校验在每次资产写入的热路径上（`UpsertTag` / `Create` / batch），必须保持内存 map 查询，不能每次打标签查库。
  - `backend/migrations/` 是 off-limits 区域；`tag_registry` 建表必须走 Atlas 迁移文件 + PR，禁止手工 DDL（AI-RULES §Schema changes）。
  - 现有 `TagRegistry`（`backend/internal/config/tag_registry.go`）已有 `sync.RWMutex` 保护的 `tags`/`sources` map，并有 `ConfigWatcher` 监听 YAML 文件变化触发 `Reload()`。
- **Goals**: 无需改 YAML 即可加标签；受管标签可 UI 化管理并即时生效；对既有校验零回退。
- **Non-Goals**: 不改 `tag_sources[]` 存储位置；不做标签 ACL；不迁移历史标签数据。

## Affected Modules
- `backend/internal/config/tag_registry.go` — Phase 1 放开 `Validate()`；Phase 2 增加 DB-backed 加载与 reload。
- `backend/internal/repository/` + `backend/internal/postgres/` — Phase 2 `tag_registry` 表的 repository。
- `backend/internal/handlers/admin/tag_registry.go` — Phase 2 CRUD handler（新文件）。
- `backend/routes/routes.go` — Phase 2 挂载 `/api/v1/admin/tag-registry`。
- `Frontend/src/components/assets/` — Task A 标签展示组件（省略/展开/来源 Badge/删除确认）。
- `Frontend/src/pages/SettingsPage.tsx` + `Frontend/src/api/tagRegistry.ts` — Phase 2 管理界面 + 类型化 client。

## Architecture Decisions

### Decision 1: 开放词汇 — 未注册 key 按自由字符串放行
- **Approach**: `Validate(key, value)` 中，`tags[key]` 不存在时不再返回错误，而是按默认 string 规则处理（默认 `max_length = 500`，与现有 `notes` 一致）。已注册 key 保持原有 enum / string 严格校验。
- **Alternative**: (a) 前端预置一批常用 key；(b) 保持拒绝、仅扩 YAML。
- **Rationale**: 用户的核心诉求是「不要每次改配置」。放行未注册 string key 一处改动即彻底解决，且无需 schema/重启。已注册标签仍受控，enum 语义不被破坏。
- **Trade-off**: 失去对「拼写错误的 key」的即时拦截（如 `notesss`）；用默认长度上限 + 后续 Phase 2 注册表治理弥补。
- **Rollback**: 恢复 `Validate()` 中未注册即报错的一行分支即可，无数据影响。

### Decision 2: Phase 2 注册表以 DB 为运行时来源，YAML 降级为 seed
- **Approach**: 新建 `tag_registry` 表持久化 TagDef。启动时若表为空则从 `tag_registry.yaml` seed；随后 `TagRegistry` 从 DB 加载进内存 map。admin CRUD 写 DB 后调用内存 `Reload()`（从 DB），实现无重启热加载。校验热路径仍只读内存 map，不查库。
- **Alternative**: (a) 纯文件 + `ConfigWatcher`（现状，K8s 里改文件需重建镜像/重启）；(b) 存到通用 settings/kv；(c) 存到 ConfigMap 热挂载。
- **Rationale**: DB 是本项目唯一权威数据源，天然多副本一致、可审计；复用既有 repository 模式；内存 map 保住热路径性能。
- **Trade-off**: 引入一次 schema 变更（off-limits，需批准 + 二审）。
- **Risk**: 启动 seed 与并发写的竞态 — 用「表空才 seed」+ 事务/唯一键规避。
- **Rollback**: 将 `TagRegistry` 加载来源切回 YAML（保留 `LoadTagRegistry`），表保留不用即可；迁移文件为纯新增表，不改既有表。

### Decision 3: admin 路由鉴权用 AdminTokenOrAdminRole
- **Approach**: 4 个 tag-registry 端点挂在 `AdminTokenOrAdminRole`（静态 admin token 或 `ADMIN_EMAILS` web session），与 CYB-3229 只读 admin 视图一致。
- **Alternative**: 仅静态 admin token（destructive `admin` 组）。
- **Rationale**: 目标是 Settings 界面里登录的管理员能自助管理标签；纯静态 token 无法支撑 UI 场景。CRUD 均记 `audit.Log`。
- **Trade-off**: 写操作对 admin-role 会话开放，需确保 `ADMIN_EMAILS` 收敛。

## Data Flow (Phase 2 启动与写入)
```
启动:
  tag_registry 表为空? --是--> 从 tag_registry.yaml seed 入表
                       --否--> 跳过
  TagRegistry.LoadFromDB() -> 内存 map

打标签 (热路径):
  UpsertTag -> TagRegistry.Validate(key,value)  [只读内存 map, 不查库]

admin 改标签:
  PATCH /admin/tag-registry/:key -> 写 DB -> TagRegistry.Reload()(从DB) -> 内存 map 刷新
```

## Data Model Changes (Phase 2)
- **Table**: `tag_registry`
- **Change**: 新建表，列约定（最终以迁移文件为准）：
  - `key VARCHAR(64) PRIMARY KEY`
  - `description TEXT`
  - `type VARCHAR(16) NOT NULL`（`enum` / `string`，CHECK 约束）
  - `values JSONB`（enum 允许值）
  - `max_length INT`
  - `propagation VARCHAR(16) NOT NULL DEFAULT 'none'`
  - `created_by VARCHAR(128)`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- **Migration**: `backend/migrations/<ts>_tag_registry.sql`（手写 SQL，`make db-migrate-hash` 重算 `atlas.sum`，`atlas migrate validate` 通过）。**off-limits，实现前单独取得用户批准。**

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 开放词汇导致 key 拼写发散 | 检索/聚合噪声 | 默认长度上限 + Phase 2 注册表治理受管 key；facet 只对受管 key 聚合 |
| 迁移未先于镜像应用 | 部署后 500 | 遵循 §Migration check：先 `apply-migration-dev.sh` 再部署后端 |
| DB seed 与并发写竞态 | 重复/丢 seed | 「表空才 seed」+ 主键冲突忽略；启动串行执行 |
| admin-role 写权限过宽 | 误改注册表 | 审计日志 + `ADMIN_EMAILS` 收敛；破坏性删除加确认 |
