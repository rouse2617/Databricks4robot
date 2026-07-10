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
- [ ] **off-limits 批准**：`backend/migrations/` 新建 `tag_registry` 表需用户明确批准 + 二审，记入 `decisions.md`

### 后端：迁移 + repository（对齐 Scenario「新注册的 enum 标签即时生效」）
- [ ] [backend] `backend/migrations/<ts>_tag_registry.sql` 手写建表（含 CHECK: type∈{enum,string}）
- [ ] [backend] `make db-migrate-hash` 重算 `atlas.sum`
- [ ] [backend] `atlas migrate validate --env migrate`（fresh PG17 replay）通过
- [ ] [backend] `TagRegistryRepo`（list/get/create/update/delete）+ Go struct
- [ ] [backend] 启动 seed：表空则从 `tag_registry.yaml` 灌入；`TagRegistry` 从 DB 加载进内存 map
- [ ] [backend] admin 写入后 `Reload()`（从 DB 刷新内存 map），热路径仍只读 map

### 后端：admin CRUD handler + 路由（对齐 Scenario「非管理员无权」/「重复 key 被拒绝」）
- [ ] [backend] `handlers/admin/tag_registry.go`：List/Create/Update/Delete，重复 key 返回冲突，删除记 `audit.Log`
- [ ] [backend] `routes.go`：4 端点挂 `AdminTokenOrAdminRole`
- [ ] [backend] handler 单测：非 admin 拒绝、重复 key 冲突、create→list 可见

### Phase 2 API contract sync（新端点，必做 rows 1/2/5/7；SDK 见 decisions）
- [ ] [docs] `api/openapi.yaml`：4 端点 paths + schema + 错误信封
- [ ] [docs] `docs/review/api-guide.md`：4 端点 curl（含 `X-Databrew-Token`/admin 会话，happy + ≥1 error）
- [ ] [test] `scripts/smoke-tag-registry-dev.sh`：create→list→update→delete + 非 admin 401
- [ ] [spec] 本 change spec delta 已含注册管理 Requirement（对齐）

### 前端：Settings 标签管理界面
- [ ] [Frontend] `src/api/tagRegistry.ts`：类型化 client（与 OpenAPI 对齐）
- [ ] [Frontend] `SettingsPage.tsx` 新增「标签管理」Tab：列表 + 新增/编辑/删除表单
- [ ] [Frontend] enum 值编辑器；删除确认；错误提示（重复 key/权限）

### Phase 2 验证 + 部署
- [ ] [backend] Tier L：`go test ./...`
- [ ] [Frontend] Tier L：`npm run lint && npm run build`
- [ ] [backend] `bash scripts/apply-migration-dev.sh "$(pwd)/backend/migrations/<ts>_tag_registry.sql"` **先于** 后端部署
- [ ] deploy dev（migrate → backend → frontend）
- [ ] **Chrome DevTools MCP**：Settings 注册 `severity` → 回资产页用 `severity=high` 成功、`fatal` 被拒
- [ ] 针对性 deploy 验证：admin `curl` create/list/delete；非 admin 401
- [ ] PR → merge → deploy-verify

---

## 收尾
- [ ] Linear CYB-3246 → Done，附 commit hash
- [ ] `decisions.md`：记录 off-limits 迁移批准、SDK out-of-scope、开放词汇默认长度取值
