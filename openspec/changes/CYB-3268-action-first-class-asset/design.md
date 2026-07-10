# Design — CYB-3268

## Architecture Context

- **Constraints**:
  - Go 1.25+ / Postgres 17 / Elasticsearch 8.x(flattened mapping 在用)
  - `assets.lifecycle_state` 列 NOT NULL + CHECK(8 个值),不允许 NULL
  - `action_label_registry` 已是受控词表源,`Validate(primary, labels) error` 接口现成
  - CYB-3267 已修 `duration_sec=0`(`usecase/asset/usecase.go:907` 直接落 `DurationMs`),新 action 不会再显示 0s
  - validator 不做时间窗校验(`asset_validator.go:195` 注释 "time-window check deferred to P1.5"),3268 自己在 usecase 加
- **Goals**:
  - 新 action 一律走一等资产路径,`actions` 表只读(老数据查询)
  - 用户能按 `asset_type=action`、`primary_label`、`description` 发现 action
  - action 在 lineage 图里可被看到(L2/L3 边)
  - 零 DB migration(action 完全 fit `assets` 表已有 schema)
- **Non-Goals**:
  - 不下线 `actions` 表(老数据保留,迁移工具后续 issue 处理)
  - 不加 `algo_run → action` 血缘边
  - 不放宽 `lifecycle_state` NOT NULL / CHECK(走 plan A hard-code 'ready')
  - 不动 validator(`checkAction` 已覆盖 L2/L3,验收 #1 不需改)

## Affected Modules

- `backend/routes/routes.go:387-390` — 改 `POST /assets/:id/actions` 路由
- `backend/internal/handlers/asset/handler.go:746-786` — `createChildAsset` 加 label 校验 + ≥1ms 时长校验
- `backend/internal/usecase/asset/usecase.go:760-768, 914` — `CreateChildAssetInput` 加 `PrimaryLabel`/`Labels` 字段,usecase 对 `asset_type='action'` 硬写 `LifecycleState="ready"`
- `backend/internal/searchindex/builder.go:42-75, 206-244` — `lifecycle_state` skip on action;删除 `actions[]` nested 投影
- `docs/agents/knowledge/action-first-class.md` — 新建迁移规则说明
- 不动:`backend/internal/deliveryrules/asset_validator.go`(已就绪)、`backend/internal/config/action_label_registry.go`(已有)、`backend/internal/elasticsearch/init-index.sh`(mapping 不变)

## Architecture Decisions

### Decision 1: `/assets/:id/actions` 4 个方法统一走 first-class asset 路径
- **Approach**: `POST/GET/PATCH/DELETE /assets/:id/actions` 全部改挂 `assetHandler.*`(`CreateAction` 已存在 + 新增 `ListActions` / `UpdateAction` / `DeleteAction`),读写 `assets` 表。老 `actionHandler` 的 4 个路由 wiring 全部切走,handler / usecase / repo 代码**保留**(不删),留给后续 backfill issue 复用,避免 3268 范围爆炸。
- **Alternative**:
  - A(已否决):只切 POST,GET/PATCH/DELETE 保留老路径 → 同 URL 不同行为,客户端必踩坑。
  - B:全部切 + 删除老 handler 代码 → 3268 范围过大(需要同步处理 backfill 脚本 `/tmp/migrate_grace_to_databrew.py` 和老数据迁移)。
- **Rationale**: 用户拍板"同 URL 必须行为统一"。方案 C 是最小改动满足"统一"诉求;老代码不删避免 backfill 脚本(单独 issue)还要重新实现 action 表读写。
- **Trade-off**: 老 `actionHandler` 代码闲置,但不删,所以不算 dead code 风险。
- **Risk**: 老 API 调用方如果依赖 200 OK + 老 shape 响应,3268 切走后响应 shape 变化(action 表行 → assets 表行) → **breaking change**。在 PR 描述里显式标注 BREAKING;`docs/review/api-guide.md` `actions` 段加醒目的 v2 行为变更横幅。
- **Rollback**: `routes.go` 4 个 wiring 改回 `actionHandler.*`(代码还在),功能回到 3267 状态。但 3268 期间新建的 action 资产(在 `assets` 表)不会被老 GET 看到 —— 这是数据迁移期固有问题,不是 3268 bug。

### Decision 2: `lifecycle_state` 走 Plan A(hard-code 'ready')
- **Approach**: `CreateChildAsset` usecase 里 `if assetType == "action" { a.LifecycleState = "ready" }`;ES doc builder 写 `lifecycle_state` 时 `if a.AssetType != "action"` 才写。
- **Alternative**:
  - Plan B:放宽 NOT NULL + CHECK,落 NULL。
  - Plan C:加 `'n/a'` 到 CHECK,落 `'n/a'`。
- **Rationale**: Plan A 零 migration,语义代价最小(用一个不准的 enum 值换 0 schema change);ES builder 那侧加 1 行 `if` 让 facet 干净。
- **Trade-off**: `lifecycle_state='ready'` 在 PG 是个"谎言";action 从来没走过 `created → processing → ready` 的状态机。任何依赖 `lifecycle_state` 推导行为(比如"ready 才能交付")的逻辑**不该作用到 action 上**,需要 reviewer 验证现有逻辑是否过滤 `asset_type='action'`。
- **Risk**: 现有某个地方 `WHERE lifecycle_state = 'ready'` 没排除 action,会把 action 错算成"可交付资产"。部署前在 PR 描述里要求 reviewer grep 一遍。
- **Rollback**: 一行 `a.LifecycleState = ""`(让 DB DEFAULT 'created' 接管),ES builder 那侧去掉 `if` 守卫即可回滚到 CYB-3267 状态。

### Decision 3: 不加 `algo_run → action` 血缘边
- **Approach**: `metadata.source_run_id` / `metadata.source_name` 继续走 metadata,lineage 图保持 L2/L3(`segment → action` / `task → action`),不在 `asset_relations` 写 `algo_run → action`。
- **Alternative**: usecase 检测 `metadata.source_run_id` 非空时写 `asset_relations` 行(`relation_type='produced_by'`)。
- **Rationale**: 用户拍板。当前业务没有"按 algo run 查它产生的 action"的查询需求;action 数远多于 algo_run,加这条边会让 lineage 图膨胀。前端如果真要 join,在 `GetAssetLineage` 里再 join `algo_runs` 表。
- **Trade-off**: 之后真要做"按 algo run 下钻"需要改路径(写边 OR join 查询),不是 3268 关切的。
- **Risk**: 低,纯后续决策。

### Decision 4: 校验放 `createChildAsset` usecase,不放 validator
- **Approach**: label 校验 + ≥1ms 时长约束加在 `usecase/asset/usecase.go:CreateChildAsset` 里(`if assetType == "action"` 分支),不动 `deliveryrules/asset_validator.go`。
- **Alternative**: 在 `asset_validator.go` 的 `checkAction` 里加。
- **Rationale**: validator 当前设计是"层级 / 存在性",不放内容语义;label 校验是 action-specific 内容规则;时长约束是 P1.5 deferred,validator 注释明确写了"不归我管"。两层职责清楚。
- **Trade-off**: 校验逻辑分两处,但每处职责单一。
- **Risk**: 后续如果有别的 child-asset 类型也需要类似校验,可能需要在 usecase 里重复判断。短期可接受,长期可以把 action-specific 校验抽 `ActionValidator`。

### Decision 5: ES 不 reindex mapping,只删 `actions[]` nested 数据
- **Approach**: ES `init-index.sh` 的 mapping(`metadata: flattened`)不变;3268 改 `builder.go` 删 `actions[]` nested 投影后,通过 `_update_by_query` 或全量 rebuild 把现有 `actions[]` 数组从老 doc 里清掉。
- **Alternative**: 重灌 mapping(改 `metadata` 子字段),代价巨大(ES 是 flattened,本来就不需要 reindex mapping)。
- **Rationale**: `metadata` 字段 mapping 没变,query IR 也没变(`metadata.X: ilike/eq` 已经走 flattened 路径),3268 唯一的 mapping-shape 改动是删 `actions[]` 数组 —— 这是文档层,不是 mapping 层,`_update_by_query` 即可。
- **Trade-off**: 删 `actions[]` 是单向操作(老数据从此不再有 ES 投影),但 `actions` 表本身保留,审计可查 PG。
- **Rollback**: 恢复 `actions[]` 嵌套投影,再 `_update_by_query` 重建数组。

## Data Flow

```
Client
  │  POST /assets/<seg_id>/actions
  │  body: createChildAssetRequest{start_ns, end_ns, metadata:{primary_label, labels, description, source_*}}
  ▼
routes.go:387  ──(统一改挂)──▶  assetHandler.CreateAction
                                                  │
  GET /assets/<seg_id>/actions                   ▼
routes.go:388  ──(统一改挂)──▶  assetHandler.ListActions
                                                  │
  PATCH /assets/<seg_id>/actions/<aid>           ▼
routes.go:389  ──(统一改挂)──▶  assetHandler.UpdateAction
                                                  │
  DELETE /assets/<seg_id>/actions/<aid>          ▼
routes.go:390  ──(统一改挂)──▶  assetHandler.DeleteAction
  │
  ▼ (4 个方法都进 assetUC)
  assetUC:
    - CreateChildAsset(CreateChildAssetInput{...})
        ├─ bind + struct validate (handler 层)
        ├─ label 校验 (usecase 层, assetType=="action" 分支)
        │     └─ action_label_registry.Validate(metadata.primary_label, metadata.labels)
        │        └─ 422 INVALID_ACTION on miss
        ├─ ≥1ms 时长校验 (usecase 层, 同分支)
        ├─ validator.checkAction(parent) → OK if parent∈{segment, task}
        │   (deliveryrules/asset_validator.go:189 已有,不动)
        ├─ if assetType=="action": a.LifecycleState = "ready"
        ├─ prepAssetForWrite
        ├─ repo.Create(asset) → INSERT INTO assets (... asset_type='action', ...)
        ├─ 写 ln(parent_asset_id=<seg_id>, child_asset_id=<new_id>) 边
        └─ outbox: asset_created 事件

    - ListActionsByParent(parent_id, limit, offset)
        └─ SELECT * FROM assets WHERE asset_type='action' AND parent_asset_id=<parent_id> AND is_deleted=false
           ORDER BY start_timestamp_ns LIMIT ... OFFSET ...
           (走 repository.AssetRepository 新增的 ListByParentAndType 方法)

    - UpdateActionAsset(parent_id, aid, UpdateInput{metadata:...})
        ├─ 校验 aid 存在 + asset_type='action' + parent_asset_id=<parent_id>
        ├─ repo.Update(a) → UPDATE assets SET metadata=..., version+=1, updated_at=now() WHERE asset_id=<aid>
        └─ outbox: asset_updated 事件

    - SoftDeleteActionAsset(parent_id, aid)
        ├─ 校验同上
        ├─ repo.SoftDelete(aid) → UPDATE assets SET is_deleted=true, version+=1 WHERE asset_id=<aid>
        └─ outbox: asset_deleted 事件

  ▼ (后续 outbox → ES)
es_subscriber.handleData
  │
  ▼
searchindex.Builder.Build(ctx, assetID)
  │
  ├─ lifecycle_state: SKIP on action
  ├─ actions[] nested: SKIP (全部删除)
  ├─ metadata: 整 map → ES flattened
  ├─ parent_asset_id / root_asset_id: 写
  └─ BulkIndex → ES

ES doc (asset_type='action'):
  {
    asset_id, asset_type: 'action',
    parent_asset_id, root_asset_id,
    metadata: { description, primary_label, labels, source_*, content_zh },
    // NOTE: lifecycle_state 不写
    tags, algos, lineage_upstream/downstream_ids, ...
  }

Client query:
  GET /assets/<seg_id>/actions
    → assetUC.ListActionsByParent(<seg_id>, ...) → assets 表 SELECT
    → 响应: [{asset_id, asset_type='action', parent_asset_id, metadata:..., start_timestamp_ns, end_timestamp_ns, ...}, ...]
    (注意:不是 /search/assets 路径,直接查 PG,不做 ES 检索)

  GET /search/assets?where=asset_type:eq:action
    → ES term query on asset_type='action' → 命中所有新 action ✓
  GET /search/assets?where=metadata.description:ilike:拿手机
    → ES wildcard on metadata.description (flattened) → 命中 ✓
  GET /search/assets?where=metadata.primary_label:eq:pickup
    → ES term on metadata.primary_label (flattened keyword) → 命中 ✓
```

## Data Model Changes

- **Table**: 无
- **Change**: 无 migration(`assets` 表 schema 不动,action 行复用同一张表)
- **Migration**: 无

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| `lifecycle_state='ready'` 在 PG 里是语义谎言 | 现有"lifecycle=ready 才可交付"之类的逻辑可能误算 action | PR 描述要求 reviewer grep `WHERE lifecycle_state` 看是否过滤 `asset_type='action'`;若发现,加 `AND asset_type != 'action'` 守卫 |
| ~~`POST` 走一等资产、`GET` 走老 `actions` 表 —— 同路径分裂~~(已推翻) | (无,3268 v2 改成 4 方法统一) | — |
| **BREAKING:老 API 调用方依赖 200 + 老 shape(action 表行),3268 切走后响应 shape 变(assets 表行)** | grace 迁移工具、其他内部脚本可能断 | `docs/review/api-guide.md` `actions` 段加 v2 行为变更横幅;`CHANGELOG` 标 BREAKING;PR 描述 mention grace 迁移工具 owner(虽然迁移脚本 3268 不改,但读 shape 变了) |
| 老 `actions` 表数据不在新 GET 里 | 老用户看不到迁移期数据 | 文档说明:3268 后 GET 只看一等资产;老数据等 backfill issue(单独)回填。`actions` 表保留供 backfill 脚本内部读,不对外 API 暴露 |
| 删除 ES `actions[]` nested 投影 → 老 doc 里这部分字段没了 | 现有"按 nested actions 搜"会失效 | ES `_update_by_query` 清掉所有 doc 里的 `actions` 字段;确认没有外部查询依赖 `actions.*` 路径(nestedPath 只对 `actions.X` 起作用,3268 改完保留 `actions` 表 PG 实体,所以 ES 这块丢的是 nested 投影,不是数据) |
| `createChildAsset` 内加 if 分支,后续 child 类型可能复制 | 校验逻辑分散 | 决策 4 的 trade-off,接受;后续可抽 `ActionValidator` |
| validator `checkAction` 注释 "L2+L3+L5" typo | 阅读者误解 | 3268 PR 顺手改注释(1 行),不影响行为 |
| ES doc 重建窗口期(改 builder.go → deploy → ES 同步)内,搜索结果可能短暂出现 action 在 `actions[]` nested 里、又在顶层 `asset_type='action'` 里 | 搜索结果短期重复 | 3268 deploy 顺序:① backend deploy(新 builder),等 5 分钟让 outbox 跑完;② 跑 `_update_by_query { "script": ... unset actions }` 清老 nested;③ 验证搜索只命中顶层 |
