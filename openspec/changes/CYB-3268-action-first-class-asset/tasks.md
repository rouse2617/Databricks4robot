# Tasks — CYB-3268

## Context files
- `backend/routes/routes.go:385-390` — `/assets/:id/actions` 4 个方法路由挂载(全切)
- `backend/internal/handlers/asset/handler.go:736-798` — `createChildAssetRequest` / `createChildAsset` / `CreateAction`(POST 已就绪,3268 加校验)
- `backend/internal/handlers/action/handler.go:42,105,189,244` — 老 `Create`/`List`/`Patch`/`Delete`(退役,代码保留)
- `backend/internal/usecase/asset/usecase.go:760-768, 880-924, 1006-1050` — `CreateChildAssetInput` / `Create` / `CreateChildAsset`
- `backend/internal/usecase/action/usecase.go` — 老 actionUC(退役后保留供 backfill)
- `backend/internal/deliveryrules/asset_validator.go:189-204` — `checkAction` (已就绪,不动)
- `backend/internal/config/action_label_registry.go` — 受控词表
- `backend/internal/repository/asset_repository.go` — 加 `ListByParentAndType` / `UpdateWithParentCheck` / `SoftDeleteWithParentCheck` 3 个方法(查 / 改 / 删 + 跨 parent 校验)
- `backend/internal/searchindex/builder.go:42-244` — ES doc builder (lifecycle_state skip + actions[] 删除)
- `backend/deploy/local/elasticsearch/init-index.sh:55-69` — ES mapping(确认 metadata=flattened 不变)
- `backend/internal/elasticsearch/query_ir.go:95-174, client.go:404-449, 670-700` — query IR / 字段路径 / buildScalarClause
- `backend/migrations/archive/016_add_actions.sql` — 老 actions 表 schema(字段映射参考)
- `docs/agents/deploy-before-commit.md`, `docs/agents/deploy-verification.md` — deploy / verify 流程

## Implementation

### Phase A: 路由 + handler + usecase 4 个方法统一
- [ ] [backend] `routes/routes.go:387-390` — `POST/GET/PATCH/DELETE /assets/:id/actions` 全部改挂 `h.asset.{CreateAction,ListActions,UpdateAction,DeleteAction}`(原 4 个 `actionHandler.*` 切走)
- [ ] [backend] `handlers/asset/handler.go` — 新增 `ListActions(c)`:`SELECT` 风格,分页参数 `limit/offset`(默认 50/0),返回 `[]Asset`(响应 shape 与 `GET /search/assets` 命中行一致)
- [ ] [backend] `handlers/asset/handler.go` — 新增 `UpdateAction(c)`:路径参数 `:aid`,body 接 `UpdateInput` 风格的 metadata/files 等;校验 `aid` 存在 + `asset_type='action'` + `parent_asset_id` 匹配 URL `:id`,否则 404
- [ ] [backend] `handlers/asset/handler.go` — 新增 `DeleteAction(c)`:路径参数 `:aid`,软删,跨 parent 校验同上,404 on miss
- [ ] [backend] `handlers/asset/handler.go:760` — `createChildAsset("action")` 在调 usecase 前加 label 校验 + ≥1ms 时长校验(assetType == "action" 分支)
- [ ] [backend] `usecase/asset/usecase.go` — `CreateChildAssetInput` 加 `PrimaryLabel` + `Labels` 字段;`CreateChildAsset` 内 `if assetType == "action" { a.LifecycleState = "ready" }`
- [ ] [backend] `usecase/asset/usecase.go` — `CreateChildAsset` 内 action 类型加 ≥1ms 时长校验(返回 `INVALID_ARGUMENT`)
- [ ] [backend] `usecase/asset/usecase.go` — 新增 `ListActionsByParent(ctx, parentID, limit, offset)`:走 assetRepo
- [ ] [backend] `usecase/asset/usecase.go` — 新增 `UpdateActionAsset(ctx, parentID, aid, UpdateInput)`:走 assetRepo.UpdateWithParentCheck
- [ ] [backend] `usecase/asset/usecase.go` — 新增 `SoftDeleteActionAsset(ctx, parentID, aid)`:走 assetRepo.SoftDeleteWithParentCheck
- [ ] [backend] `repository/asset_repository.go` — 新增 `ListByParentAndType(ctx, parentID, assetType, limit, offset)`:`SELECT ... WHERE parent_asset_id=$1 AND asset_type=$2 AND is_deleted=false ORDER BY start_timestamp_ns LIMIT $3 OFFSET $4`
- [ ] [backend] `repository/asset_repository.go` — 新增 `UpdateWithParentCheck(ctx, aid, parentID, updates)`:UPDATE 用 `WHERE asset_id=$1 AND parent_asset_id=$2 AND asset_type='action' AND is_deleted=false`,affected rows=0 → 返回 ErrNotFound
- [ ] [backend] `repository/asset_repository.go` — 新增 `SoftDeleteWithParentCheck(ctx, aid, parentID)`:同上模式
- [ ] [backend] `cmd/server/core.go:97` — `actionLabelReg` 注入确认(若 assetUC 复用同一个,无需改;若没有,从 actionUC 抽出来共享注入)

### Phase B: ES doc builder
- [ ] [backend] `searchindex/builder.go:42-75` — `lifecycle_state` 写入对 `a.AssetType == "action"` skip(`if a.AssetType != "action" { doc["lifecycle_state"] = lifecycleState }` 改写逻辑)
- [ ] [backend] `searchindex/builder.go:206-244` — 删 `actions[]` nested 投影(`if b.Actions != nil` 整块);Builder.Actions 字段保留(暂时不用,后面清理)
- [ ] [backend] `outbox/es_subscriber.go` — 确认无引用 Builder.Actions(若有,跟着删)

### Phase C: 测试
- [ ] [backend] `handlers/asset/handler_test.go` — 加 `TestCreateAction_AsFirstClassAsset`:mock assetUC,验 `asset_type='action'`、`lifecycle_state='ready'`、`parent_asset_id` 设对;422 路径覆盖 unregistered label + duration<1ms + hierarchy violation
- [ ] [backend] `handlers/asset/handler_test.go` — 加 `TestListActions_ReturnsFromAssetsTable`:mock repo,验查询 `assets` 表(非 `actions` 表)、shape 正确
- [ ] [backend] `handlers/asset/handler_test.go` — 加 `TestUpdateAction_RejectsCrossParent`:mock repo,`aid` 存在但 parent 不匹配 → 404
- [ ] [backend] `handlers/asset/handler_test.go` — 加 `TestDeleteAction_RejectsCrossParent`:同上模式
- [ ] [backend] `usecase/asset/usecase_test.go` — `TestCreateChildAsset_ActionHardCodesLifecycleReady`:验 action 走的 input 即使没 LifecycleState 也会被写 'ready';`TestCreateChildAsset_ActionDurationFloor`:duration < 1ms → `INVALID_ARGUMENT`
- [ ] [backend] `repository/asset_repository_test.go` — `TestUpdateWithParentCheck_AffectsZeroRowsOnMismatch`:`aid` 存在但 parent 不同 → 0 rows affected → ErrNotFound
- [ ] [backend] `searchindex/builder_test.go` — `TestBuild_ActionAssetOmitsLifecycleState`:action 资产 doc 不含 `lifecycle_state` 字段;`TestBuild_ActionAssetNoActionsNested`:doc 不含 `actions` 字段
- [ ] [backend] `config/action_label_registry_test.go` 或 `usecase/asset/*` — `TestCreateChildAsset_ActionLabelRegistryRejectsUnknown`:未注册 label → 422

### Phase D: 文档
- [ ] [docs] `docs/agents/knowledge/action-first-class.md` — 新建:
  - grace action → `asset_type='action'` 字段映射表(`primary_label` / `labels` / `description` / `source_*` 进 `metadata.*`)
  - **v2 行为变更横幅**:3268 后 4 个方法统一读写 `assets` 表,响应 shape 是 asset 行;老 `actions` 表不再通过 API 暴露
  - 老 `actions` 表数据查询指南(backfill 之前只能查 PG;运维 SQL 模板)
- [ ] [docs] `docs/review/api-guide.md` — `actions` 段加 v2 横幅 + 字段映射 + 错误码(`INVALID_ACTION` / `HIERARCHY_VIOLATION` / `NOT_FOUND` / `INVALID_ARGUMENT`)
- [ ] [docs] `CHANGELOG.md`(或类似)— 标 **BREAKING**: `/assets/:id/actions` 4 方法响应 shape 变化(action 表行 → assets 表行)

### Phase E: Frontend adapter (option B — no shape regression)
- [ ] [frontend] `Frontend/src/api/actions.ts` — `list()`: map returned first-class asset rows → `Action` view model (`asset_id→action_id`, `metadata.primary_label→primary_label`, `metadata.labels→labels`, `metadata.description→description`, `metadata.source_type/source_name→…`, `start_timestamp_ns→start_ns`, `end_timestamp_ns→end_ns`); envelope stays `{items, asset_id, total}`; client-side `label` filter (backend GET is limit/offset only)
- [ ] [frontend] `Frontend/src/api/actions.ts` — `create()`: map `ActionCreateInput` → `createChildAssetRequest{start_timestamp_ns, end_timestamp_ns, metadata:{primary_label, labels, description, source_type, source_name}}`; map 201 response back → `Action`
- [ ] [frontend] adapter is **defensive dual-shape** (`a.primary_label ?? a.metadata?.primary_label`, `a.start_ns ?? a.start_timestamp_ns`, …) so a FE/BE deploy-order gap does not break the tab
- [ ] [frontend] `ActionsTimelineTab.tsx` unchanged (verify 8 read-sites + create modal all go through the adapter); only endpoint consumer (PreviewPage.tsx is a false match)

## API contract sync (mandatory — same PR,4 个路由改挂 + 新增 GET/PATCH/DELETE 行为)
See [`docs/agents/AI-RULES.md` § API contract sync](../../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [ ] `api/openapi.yaml` — `POST /assets/{id}/actions` 改 body 为 `createChildAssetRequest`,response 200/201 + 422 错误码;`GET /assets/{id}/actions` 改 response shape 为 `[Asset]`(原 `[Action]`)+ 分页参数;`PATCH /assets/{id}/actions/{aid}` 标 first-class 实现;`DELETE /assets/{id}/actions/{aid}` 标 204/404;全部标 **v2 / breaking**
- [ ] `docs/review/api-guide.md` — actions 段重写为 first-class 语义;加 v2 行为变更横幅
- [~] `sdk/` — **DEFERRED / out of scope（用户 2026-07-10「sdk 的你不用管」）**。`ActionManager` 是泛型 dict passthrough（无 shape 耦合，端点不变），deferral 不引入 SDK 破坏；见 decisions.md。契约同步最小行 1/2/5/7 已满足。
- [ ] `scripts/smoke-asset-actions-dev.sh` — 全路径覆盖:POST 创建 → GET 命中(同一 URL) → PATCH 改 metadata → GET 看新值 → DELETE → GET 不见;error path(unregistered label / duration<1ms / PATCH 跨 parent / PATCH 不存在 aid)
- [ ] `openspec/changes/CYB-3268-*/specs/asset-management/spec.md` — 已写(本文档上方)
- [ ] Frontend 调用:**option B** — `Frontend/src/api/actions.ts` 加 adapter(见 Phase E),action tab 切 first-class shape,`ActionsTimelineTab.tsx` 不动;merge 后无 shape regression

## Local verification (Tier L — cross-module,API 4 路由改挂 + 新增 usecase + ES builder)
- [ ] `cd backend && make fmt && make vet`
- [ ] `cd backend && go build ./...`
- [ ] `cd backend && go test ./internal/handlers/asset/... ./internal/usecase/asset/... ./internal/repository/... ./internal/searchindex/... ./internal/postgres/...`
- [ ] `cd sdk && uv run ruff check src/ && uv run pytest tests/unit/ -q`
- [ ] (若改了 `api/openapi.yaml`)`cd sdk && uv run pytest tests/unit/test_openapi.py -q`
- [ ] `cd Frontend && npm run lint && npm run typecheck && npm run build && npm run test -- --run`

## Deploy verification (before commit — runtime change)
- [ ] `git rev-parse --short HEAD`(P3 检查)
- [ ] `bash scripts/dev-local.sh --deploy --preview-id <id>`(preview pod 验证 + 不影响 dev);或 `gcloud builds submit --config cloudbuild.yaml .` + `bash deploy/cloudrun/backend-dev.sh`
- [ ] `bash scripts/smoke-asset-actions-dev.sh`(部署后立刻跑;P5 — 用已知 bad payload 命中新代码路径)
- [ ] `curl POST/GET/PATCH/DELETE $DATABREW_URL/api/v1/assets/<test_seg>/actions[...]` 4 个方法各跑一遍 happy + 1 error path
- [ ] ES `_update_by_query { script: { source: "ctx._source.remove('actions')" } }`(deploy 后清掉老 doc 里的 actions[] 嵌套数组)
- [ ] ES doc 抽样 `GET assets/_doc/<action_id>` 确认无 `lifecycle_state` 字段
- [ ] [frontend] 部署 frontend dev(`cd Frontend && npx wrangler deploy --env dev`;紧跟 backend 部署缩短 shape 窗口)
- [ ] [frontend] **Chrome DevTools MCP**(diff 含 `Frontend/` — 强制):dev AssetDetailPage 打开某 segment 的 Action 时间轴 → list 渲染、新建 Action happy path、label 过滤;截图存 `openspec/changes/CYB-3268-*/deploy-verify-*.png`;Console 无新增 error

## PR
- [ ] PR 标题:`feat(backend): action first-class asset — unify /assets/:id/actions 4 methods on assets table (cyb-3268)`(**BREAKING**)
- [ ] PR body:`## Linear` 链 CYB-3268;`## OpenSpec` 链 `openspec/changes/CYB-3268-action-first-class-asset/`;`## Agent decisions` 抄 4 条(lifecycle plan A / 不加 algo_run 边 / validator 不动 / 4 方法统一 + 老 handler 保留);`## BREAKING` 标 `/assets/:id/actions` 4 方法响应 shape 变;`## API contract sync` 列勾过的 7 行
- [ ] PR 描述要求 reviewer grep `WHERE lifecycle_state` 看是否过滤 `asset_type='action'`(Plan A 风险)
- [ ] PR 描述 mention grace 迁移工具 owner(虽然迁移脚本 3268 不改,但读 shape 变了,需要 heads-up)
- [ ] PR 描述明确:老 `actionHandler` 退役但不删,代码保留供 backfill 复用
