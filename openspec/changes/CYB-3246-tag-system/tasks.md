# CYB-3246 Tasks

## Setup
- [x] Linear CYB-3246（DataBrew project）
- [x] Branch `feat/CYB-3246-tag-system` from `origin/dev`
- [x] OpenSpec proposal / design / spec delta / tasks
- [x] **OpenSpec checkpoint** — 用户 2026-07-10 确认「ok」，见 decisions.md

---

## Phase 1 — 开放词汇 + 标签展示（先交付 PR）

### 后端：开放词汇校验（对齐 Scenario「未注册 string key 被接受」/「已注册 enum 非法值仍被拒绝」/「未注册 key 超默认长度被拒绝」）
- [x] [backend] `config/tag_registry.go` `Validate()`：未注册 key 走默认 string 分支，不再直接报错
- [x] [backend] `DefaultUnregisteredTagMaxLength = 500` 常量 + 注释（与 `notes` 对齐，len 字节）
- [x] [backend] `config/tag_registry_test.go`：未注册放行 / 未注册超长拒绝 / 已注册 enum 非法仍拒绝 + 更新 Property24
- [x] [backend] `ShouldPropagate` 未注册 key 返回 false 断言（`TestShouldPropagate_UnregisteredKeyNeverPropagates`）

### 前端：标签展示优化（Task A，对齐 Scenario「超长标签值可展开」/「删除标签需确认」）
- [x] [Frontend] 定位标签组件：`components/asset-detail/TagsTab.tsx`
- [x] [Frontend] 标签值超长（>40 字）省略 + 点击 Popover 查看全文（`TagValueText`，可复制）
- [x] [Frontend] 来源信息改为独立 `Tag` Badge，与值分离（不再混排 `[system]`）
- [x] [Frontend] 删除标签 `Popconfirm` 确认（okText/danger）
- [x] [Frontend] chip 容器边框 + 布局对齐，`flexWrap` 窄屏不溢出
- [x] [Frontend] **修复现有 bug**：`handleAdd` 把 `localTags`(对象) 当数组 `[...prev]` + 未定义 `LocalTag` → 点添加崩溃；改为对象/数组各自更新 + 快照回滚
- [x] [Frontend] 开放词汇入口：key 选择器 `Select`→`AutoComplete`，可输入自定义 key（注册 key 作建议）

### Phase 1 API contract sync（校验行为变更，无新端点）
- [x] [docs] `api/openapi.yaml`：`TagUpsertRequest` 修正字段名 `tag_key/tag_value`→`key/value`（修 drift）+ 开放词汇说明 + POST /tags 加 422
- [x] [docs] `docs/review/api-guide.md`：Tags 校验规则补开放词汇；curl #6 未注册 key；422 行改写
- [x] [test] `scripts/smoke-tag-openvocab-dev.sh`：未注册 200+持久化 / enum 非法 422 / enum 合法 200 防回归

### Phase 1 验证 + 部署
- [x] [backend] Tier M：`gofmt` clean + `go vet ./internal/config/...` + `go test ./internal/config/... ./internal/usecase/asset/...` 通过
- [x] [Frontend] Tier L：Biome check clean（TagsTab）+ `tsc` TagsTab 无错 + `npm run build` 通过（组件测试受预存缺 `@testing-library/dom` 阻塞，见 decisions）
- [ ] deploy dev（backend Cloud Run dev + frontend dev）
- [ ] **Chrome DevTools MCP**（diff 触及 `Frontend/`）：资产详情页打未注册标签→展开→删除确认全流程
- [ ] 针对性 deploy 验证：`bash scripts/smoke-tag-openvocab-dev.sh`（dev）
- [ ] PR（填 `.github/pull_request_template.md`）→ merge → deploy-verify

---

## Phase 2 — 注册表 DB 化 + admin CRUD + Settings 管理界面（独立 PR）

### 前置门禁
- [x] **off-limits 批准**：用户「合并吧，并且做 Phase 2」批准新建 `tag_registry` 表（见 decisions.md 2026-07-10）；PR 需二审。

### 后端：迁移 + repository（对齐 Scenario「新注册的 enum 标签即时生效」）
- [x] [backend] `migrations/20260710120219_add_tag_registry.sql` plain CREATE（CHECK: type∈{enum,string}, propagation∈{none,descendants}）
- [x] [backend] `make db-migrate-hash` 重算 `atlas.sum`（含新迁移哈希）
- [x] [backend] `atlas migrate validate --env migrate`（fresh PG17 docker replay）EXIT=0
- [x] [backend] `postgres.TagRegistryRepo`（Count/List/Get/Create/Update/Delete）+ `models.TagRegistryEntry` + `repository.TagRegistryRepository` 接口 + `ErrDuplicateTagKey`
- [x] [backend] 启动 seed+load：`adminH.SeedAndLoadTagRegistry`（表空从 YAML 灌入 → 从 DB 加载进内存 map），wired in `cmd/server/server.go`
- [x] [backend] `config.TagRegistry.ReplaceTags` 原子换 map；admin 写入后 `refreshRegistry` 从 DB 刷新，热路径仍只读 map

### 后端：admin CRUD handler + 路由（对齐 Scenario「非管理员无权」/「重复 key 被拒绝」）
- [x] [backend] `handlers/admin/tag_registry.go`：List/Create/Update/Delete，重复 key→409，缺失→404，校验→422，写入记 `audit.Log`
- [x] [backend] `routes.go`：4 端点挂 `AdminTokenOrAdminRole`（guard `tagRegistryHandler != nil`）
- [x] [backend] handler 单测：create→refresh→list、重复 409、校验 422、update 404、delete 后回到开放词汇（全绿）+ 修 `routes_test.go` 9 处 RegisterAll 调用

### Phase 2 API contract sync（新端点，必做 rows 1/2/5/7；SDK 见 decisions）
- [x] [docs] `api/openapi.yaml`：4 端点 paths + `TagRegistryEntry`/`TagDefRequest` schema + 401/409/422/404 错误信封（YAML 校验通过）
- [x] [docs] `docs/review/api-guide.md`：§2.4.1 受管标签注册表 CRUD（curl happy + 错误路径表）
- [x] [test] `scripts/smoke-tag-registry-dev.sh`：create 201→list→update 200→dup 409→bogus 401→delete 200
- [x] [spec] 本 change spec delta 已含注册管理 Requirement（对齐）

### 前端：Settings 标签管理界面
- [x] [Frontend] `src/api/tagRegistry.ts`：`tagRegistryAdminApi`（list/create/update/remove，`skipAuthRedirect`）+ 类型
- [x] [Frontend] `SettingsPage.tsx` 新增「标签管理」Card → `TagRegistryManager` 组件（Table + 新增/编辑/删除 Modal 表单）
- [x] [Frontend] enum 值 tags 编辑器；删除 Popconfirm；重复 key/权限错误提示；非 admin(401) 显示「需要管理员权限」而非登出

### Phase 2 验证 + 部署
- [x] [backend] Tier L：`go build ./...` + `go vet ./...` + `go test ./internal/handlers/admin/... ./internal/config/... ./routes/...` 通过
- [x] [Frontend] Tier L：Biome check clean + `tsc` 新文件无错 + `npm run build` 通过
- [ ] [backend] migrate 由 GHA `deploy-dev.yml` 的 Atlas migrate Job 应用（push-to-dev 触发；本仓库无本地 apply 脚本）
- [ ] deploy dev（GHA：migrate → backend → frontend，merge 触发）
- [ ] **Chrome DevTools MCP**（post-merge）：Settings 注册 `severity` → 资产页 `severity=high` 成功、`fatal` 被拒
- [ ] 针对性 deploy 验证（post-merge）：`bash scripts/smoke-tag-registry-dev.sh`
- [ ] PR → 二审 → merge → deploy-verify

### Phase 1 部署验证（已完成，2026-07-10）
- [x] deploy dev（GHA deploy-dev exit 0；frontend `dev#87af5621`）
- [x] **Chrome DevTools MCP**：资产 XlPAwMz0 → AutoComplete 加自定义 key `ui_custom_tag` 成功（handleAdd 崩溃已修）、来源 Badge、删除确认、控制台无新错误
- [x] 针对性 deploy 验证：`smoke-tag-openvocab-dev.sh` 全绿（未注册 200 / enum 非法 422 / enum 合法 200）

---

## 收尾
- [ ] Linear CYB-3246 → Done，附 commit hash
- [ ] `decisions.md`：记录 off-limits 迁移批准、SDK out-of-scope、开放词汇默认长度取值
