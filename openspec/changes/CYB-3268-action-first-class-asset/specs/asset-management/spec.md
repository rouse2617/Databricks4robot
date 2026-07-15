## ADDED Requirements

### Requirement: Action 一等资产创建
The system SHALL let clients create an `action` first-class asset under a segment or task parent via `POST /assets/:id/actions`, persisting the asset to the `assets` table (not the legacy `actions` table). The request body SHALL use `createChildAssetRequest` shape (`start_timestamp_ns`, `end_timestamp_ns`, `split_method`, `split_run_id`, `metadata`). The created asset SHALL have `asset_type='action'`, `parent_asset_id=<parent>`, and `lifecycle_state='ready'` (set unconditionally for action).

**Priority**: P0 (Critical)
**Rationale**: Grace 迁移的 action 现在通过独立 `actions` 表读取,无法被资产搜索 / facet / lineage 统一处理;新 action 必须走 `assets` 表一等资产路径,才能让用户主动发现。

#### Scenario: 在 segment 下创建 action 成功
- **Given** 一个 `asset_type='segment'` 的资产 `<seg_id>` 已存在,且 `metadata.primary_label='pickup'`、`metadata.labels=['hand','grasp']` 在 `action_label_registry` 注册
- **When** 客户端发 `POST /assets/<seg_id>/actions` body `createChildAssetRequest{start_timestamp_ns: 100, end_timestamp_ns: 200, metadata: {primary_label: 'pickup', labels: ['hand','grasp'], description: '拿手机'}}`
- **Then** 响应 201 + 资产对象 `{asset_id: '<new>', asset_type: 'action', parent_asset_id: '<seg_id>', lifecycle_state: 'ready', duration_ms: 0}`
- **And** `assets` 表新增一行 `asset_type='action'`,`metadata` JSONB 含 `primary_label='pickup'` / `labels=['hand','grasp']` / `description='拿手机'`
- **And** `actions` 表**不**新增任何行
- **And** `ln(parent_asset_id=<seg_id>, child_asset_id=<new>, relation_type='derived_from')` 边被写入
- **And** outbox 收到一条 `asset_created` 事件,ES 同步后该 action 可被 `asset_type=eq:action` 搜索命中

#### Scenario: 在 task 下创建 action 成功
- **Given** 一个 `asset_type='task'` 的资产 `<task_id>` 已存在
- **When** 客户端发 `POST /assets/<task_id>/actions` body 同上合法内容
- **Then** 响应 201 + 资产对象,`parent_asset_id='<task_id>'`,`root_asset_id` 指向根 segment / raw_mcap

#### Scenario: 父资产类型非法(action 挂到非 segment/task)
- **Given** 一个 `asset_type='frame'` 的资产 `<frame_id>` 已存在
- **When** 客户端发 `POST /assets/<frame_id>/actions`
- **Then** 响应 422 `HIERARCHY_VIOLATION`,`invariant='L2'`,`expected='action parent must be segment (L2) or task (L3)'`

#### Scenario: 父资产不存在
- **Given** `<parent_id>` 在 `assets` 表里查不到
- **When** 客户端发 `POST /assets/<parent_id>/actions`
- **Then** 响应 422 `HIERARCHY_VIOLATION`,`expected='parent asset <parent_id> not found'`

### Requirement: Action 不参与 lifecycle_state
The system SHALL set `lifecycle_state='ready'` for every newly-created `asset_type='action'` row regardless of request input, and SHALL NOT include `lifecycle_state` in the Elasticsearch document for `asset_type='action'` rows.

**Priority**: P0 (Critical)
**Rationale**: action 是只读标注,没有 `created → processing → ready → delivered` 的状态机;但 DB `assets.lifecycle_state` 是 NOT NULL + CHECK(8 个值),不允许 NULL,3268 走 plan A:PG 兜底写 'ready',ES facet 侧不暴露。

#### Scenario: action 行 lifecycle_state 在 PG 是 'ready'
- **Given** 任何合法的 action 创建请求
- **When** action 写入 `assets` 表
- **Then** 该行 `lifecycle_state='ready'`(即使请求 body 没带 `lifecycle_state` 字段,或带了别的值)

#### Scenario: action ES doc 不含 lifecycle_state
- **Given** `assets` 表里有一行 `asset_type='action'`、`lifecycle_state='ready'` 的资产
- **When** ES subscriber 把该资产同步到 ES
- **Then** ES `_source` 里**不包含** `lifecycle_state` 字段
- **And** `asset_type_agg` facet 仍然包含 `action` bucket
- **And** `lifecycle_state` facet **不**出现 `ready` bucket 来自这条 action(其他类型的 ready bucket 不受影响)

### Requirement: Action label 受控词表校验
The system SHALL validate `metadata.primary_label` and `metadata.labels[]` against `backend/internal/config/action_label_registry` on every action creation request. If `primary_label` is non-empty and not in the registered primary label set, or any `labels[]` entry is not in the registered label set, the system SHALL reject with 422 `INVALID_ACTION`.

**Priority**: P0 (Critical)
**Rationale**: action 是用户和算法共用的标注,label 必须受控,否则下游检索 / 聚合会污染(`primary_label=pcikup` typo 这种 case)。

#### Scenario: 注册过的 primary_label 接受
- **Given** `action_label_registry.primary_labels` 含 `pickup`
- **When** 客户端发 `metadata.primary_label='pickup'`
- **Then** 响应 201(action 创建成功)

#### Scenario: 未注册的 primary_label 拒绝
- **Given** `action_label_registry.primary_labels` **不**含 `grab`(假设未注册)
- **When** 客户端发 `metadata.primary_label='grab'`
- **Then** 响应 422 `INVALID_ACTION`,`expected` 字段提示 `action_label_registry: primary_label "grab" not registered`

#### Scenario: 至少一个 labels[] entry 未注册
- **Given** `action_label_registry.labels` 含 `hand` 但不含 `gloves-with-mittens`
- **When** 客户端发 `metadata.labels=['hand','gloves-with-mittens']`
- **Then** 响应 422 `INVALID_ACTION`,`expected` 字段提示未注册的那个 label 名

#### Scenario: primary_label 为空,labels[] 全注册
- **Given** `metadata.primary_label=''`,`metadata.labels=['hand']`
- **When** 客户端发请求
- **Then** 响应 201(primary_label 为空时跳过 primary 校验)

### Requirement: Action 时长下限 ≥1ms
The system SHALL reject any action creation request where `end_timestamp_ns - start_timestamp_ns < 1_000_000` (1ms) with 422 `INVALID_ARGUMENT`.

**Priority**: P1 (High)
**Rationale**: databrew 资产统一规则:`duration_ms` ≥ 1。grace action 范围相对偏移,可能产生 0 时长空段,需要在 usecase 兜底校验,避免资产出现 `duration_sec=0`(CYB-3267 修了 stale 状态,3268 防止新写入再现)。

#### Scenario: 时长 0 ns 拒绝
- **Given** `start_timestamp_ns=100, end_timestamp_ns=100`
- **When** 客户端发请求
- **Then** 响应 422 `INVALID_ARGUMENT`,提示 "duration must be >= 1ms"

#### Scenario: 时长 <1ms 拒绝
- **Given** `start_timestamp_ns=100, end_timestamp_ns=999_999`(999_999 ns = 0.999ms)
- **When** 客户端发请求
- **Then** 响应 422 `INVALID_ARGUMENT`,提示 "duration must be >= 1ms"

#### Scenario: 时长 ≥1ms 接受
- **Given** `start_timestamp_ns=100, end_timestamp_ns=1_000_100`(1ms 整)
- **When** 客户端发请求
- **Then** 响应 201,`duration_ms=1`

### Requirement: Action 可被资产搜索与 facet
The system SHALL surface first-class action assets in `/search/assets` so that clients can:
1. facet by `asset_type='action'` (alongside segment / task / clip / frame)
2. filter by `metadata.primary_label` (term match against flattened metadata)
3. search by `metadata.description` (case-insensitive substring match)

**Priority**: P0 (Critical)
**Rationale**: 这是用户拍板 3268 的核心动因 —— 能主动发现 action。

#### Scenario: 按 asset_type=action 列出所有新 action
- **Given** `assets` 表里有 N 条 `asset_type='action'`(grace 迁移工具创建的)
- **When** 客户端发 `GET /search/assets?where=asset_type:eq:action&limit=50`
- **Then** 响应包含全部 N 条 action(分页按 limit/offset 走)
- **And** `facets.asset_type` 里 `action` bucket 的 count ≥ N

#### Scenario: 按 metadata.primary_label=pickup 过滤
- **Given** `assets` 表里有若干 action,其中 `metadata.primary_label='pickup'` 的有 M 条
- **When** 客户端发 `GET /search/assets?where=metadata.primary_label:eq:pickup`
- **Then** 响应只含那 M 条 action

#### Scenario: 按 metadata.description ilike 中文
- **Given** `assets` 表里有 action,`metadata.description='拿手机'` 的有 K 条
- **When** 客户端发 `GET /search/assets?where=metadata.description:ilike:拿手机`
- **Then** 响应含这 K 条 action(ES wildcard 在 flattened metadata 上 case_insensitive 工作)

#### Scenario: 老 actions 表里的 action 不出现在资产搜索
- **Given** `actions` 表里有老数据(grace 迁移工具走老 API 创建的),它们**不在** `assets` 表里
- **When** 客户端发 `GET /search/assets?where=asset_type:eq:action`
- **Then** 响应**不**含老 `actions` 表里的行(它们没在 `assets` 表里,ES 也没 doc)

## MODIFIED Requirements

### Requirement: Asset CRUD
- **Before**: `/assets/:id/actions` 4 个方法(`POST` / `GET` / `PATCH /:aid` / `DELETE /:aid`)全部路由到 `actionHandler.*`,读写独立 `actions` 表;action 不进 `assets` 表、不进 ES 资产 facet、不参与血缘图(`ln` 边表)。
- **After**: `/assets/:id/actions` 4 个方法**统一**路由到 `assetHandler.*`,读写 `assets` 表(`asset_type='action'` 行);`ln` 边表由 `CreateChildAsset` 自动写;action 进 ES 资产 facet、可被 `metadata.*` 查询、可被 `parent_asset_id` 血缘下钻。
- **Reason**: 用户对 action 的"主动发现"需求(3268 Why)无法被 annotation overlay 抽象满足;同时要求"同 URL 行为必须统一",杜绝"POST 写一张表 / GET 读另一张表"的分裂。

#### Scenario: POST /assets/:id/actions 写入 assets 表
- **Given** 任何合法的 action 创建请求
- **When** 客户端发 `POST /assets/<seg_id>/actions`
- **Then** 写入 `assets` 表(非 `actions` 表),`ln` 边表新增一行,outbox 发出 `asset_created` 事件,ES 同步后该 action 可被 `asset_type=eq:action` 搜索命中

#### Scenario: GET /assets/:id/actions 读 assets 表
- **Given** `assets` 表里有 N 条 `asset_type='action'`、`parent_asset_id=<seg_id>` 的行(grace 走新路径创建的)
- **When** 客户端发 `GET /assets/<seg_id>/actions?limit=50&offset=0`
- **Then** 响应**只含**这 N 条行(响应 shape 与 `GET /search/assets?where=asset_type:eq:action` 命中的行一致)
- **And** 响应**不**含老 `actions` 表里的行(3268 后老表不通过 API 暴露,backfill 单独 issue)

#### Scenario: PATCH /assets/:id/actions/:aid 更新 assets 行
- **Given** `<aid>` 是 `assets` 表里 `asset_type='action'`、`parent_asset_id=<seg_id>` 的行
- **When** 客户端发 `PATCH /assets/<seg_id>/actions/<aid>` body `{metadata: {primary_label: 'pickup', ...}, files: {...}}`
- **Then** `assets` 行 `metadata` / `files` 等字段被更新,`version` 自增,`updated_at` 刷新
- **And** outbox 发出 `asset_updated` 事件,ES 同步后该 action doc 反映新 metadata

#### Scenario: PATCH /assets/:id/actions/:aid 跨 parent 校验
- **Given** `<aid>` 存在但 `parent_asset_id` **不**等于 `<seg_id>`(挂在别的 parent 下)
- **When** 客户端发 `PATCH /assets/<seg_id>/actions/<aid>`
- **Then** 响应 404 `NOT_FOUND`(防止跨 segment 改 action)

#### Scenario: PATCH /assets/:id/actions/:aid 不存在的 aid
- **Given** `<aid>` 在 `assets` 表里查不到
- **When** 客户端发 `PATCH /assets/<seg_id>/actions/<aid>`
- **Then** 响应 404 `NOT_FOUND`

#### Scenario: DELETE /assets/:id/actions/:aid 软删 assets 行
- **Given** `<aid>` 是 `asset_type='action'`、`parent_asset_id=<seg_id>` 的行
- **When** 客户端发 `DELETE /assets/<seg_id>/actions/<aid>`
- **Then** `assets` 行 `is_deleted=true`、`version` 自增,outbox 发出 `asset_deleted` 事件,ES 同步后该 doc 被删除
- **And** 再次 GET /assets/<seg_id>/actions 不含该 action

#### Scenario: DELETE /assets/:id/actions/:aid 跨 parent 校验
- **Given** `<aid>` 存在但 `parent_asset_id` **不**等于 `<seg_id>`
- **When** 客户端发 `DELETE /assets/<seg_id>/actions/<aid>`
- **Then** 响应 404 `NOT_FOUND`

## REMOVED Requirements

### Requirement: 独立 actions 表的 REST API 暴露
- **Was**: `/assets/:id/actions` 4 个方法路由到 `actionHandler.*`,读写独立 `actions` 表,响应 shape 是 action 表行(`action_id` / `start_ns` / `primary_label` 等老字段)。
- **Reason**: 3268 统一走 `assets` 表后,老 API 不再被路由挂载。老表(独立 `actions`)只在 backfill 脚本内部读写(单独 issue),不对外暴露;老 handler / usecase / repo 代码保留供 backfill 复用,不在 3268 删。
