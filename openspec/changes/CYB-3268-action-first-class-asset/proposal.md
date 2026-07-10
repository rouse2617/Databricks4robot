# Proposal — CYB-3268

## Why

databrew 的 `actions` 表是一个**标注层**(annotation overlay):action 不进 lifecycle、不进 deliveries、不进资产列表 API。Grace 迁移过来后用户开始主动发现 action(按 `primary_label=pickup` 搜、按 description 搜、lineage 下钻)—— 这套 overlay 抽象装不下了,要让 action 变成一等资产(`asset_type='action'`),和 segment/task 一样可被 facet、可被检索、可被血缘图看到。

CYB-3267 修了 overlay 层的两个 bug(scan 漏 `task_id` + `duration_sec=0`),3268 在此基础上**把新写入路径从 overlay 切到 first-class**,老数据保留。

## What Changes

### New Capabilities
- **asset-management：Action 一等资产创建** — `POST /assets/:id/actions` 走 `assetHandler.CreateAction`(已在 `handler.go:792` 写好,挂在 `createChildAsset("action")`),新 action 落到 `assets` 表(非 `actions` 表),body 用 `createChildAssetRequest`,Grace 富字段进 `metadata.*`,`primary_label`/`labels[]` 走 `action_label_registry` 受控词表校验。

### Modified Capabilities
- **asset-management：Asset CRUD** — `/assets/:id/actions` 4 个方法(`POST` / `GET` / `PATCH /:aid` / `DELETE /:aid`)**统一**路由到 `assetHandler` 的 first-class asset 实现(写 `assets` 表);老 `actionHandler` 整组 handler 退役(routes 上 4 个 wiring 切走,**代码保留不删**,供后续 backfill issue 复用);老 `actions` 表数据由单独的 backfill issue 处理,不在 3268 范围内。
- **asset-management：Asset CRUD** — `lifecycle_state` 对 `asset_type='action'` 行 hard-code 落 `'ready'`(DB NOT NULL + CHECK 限制),ES doc builder 对 action 类型**不写** `lifecycle_state` 字段,避免 facet 出现 `lifecycle: ready` 这种无意义 bucket。
- **search：Action 检索与 facet** — ES doc builder 对 action 资产不再写 `actions[]` nested 投影(转写 `assets` 顶层 doc 的 `metadata.*`);`asset_type=eq:action`、`metadata.primary_label=eq:pickup`、`metadata.description=ilike:拿手机` 三类查询直接命中 ES 现有 flattened metadata 字段,无需 ES reindex(映射 `metadata: flattened` 已在 init-index.sh 里)。

## Impact

- **Affected code**:
  - `backend/routes/routes.go:387-390` — 4 个方法全改挂 `h.asset.*`(原本挂的是 `actionHandler.*`);老 `actionHandler` 在 `core.go:97` 的注入保留(backfill 阶段可能复用),但 routes 上的 4 个 wiring 全部切走
  - `backend/internal/handlers/asset/handler.go` — 新增 `ListActions` / `UpdateAction` / `DeleteAction` 3 个 handler(查 / 改 / 删 `assets` 表 `asset_type='action'` 行);`CreateAction` 已存在(792),只需在 `createChildAsset` 内做 label 校验 + ≥1ms 时长校验
  - `backend/internal/handlers/action/handler.go` — 4 个 handler 退役(routes wiring 切走,**代码保留不删**供 backfill 复用;`internal/usecase/action/`、`internal/postgres` 的 action 表读写代码一并保留,不删)
  - `backend/internal/usecase/asset/usecase.go` — `CreateChildAsset` 对 `asset_type='action'` 时硬写 `LifecycleState="ready"`;新增 `ListActionsByParent` / `UpdateActionAsset` / `SoftDeleteActionAsset` 3 个方法
  - `backend/internal/searchindex/builder.go` — line 52 `lifecycle_state` 写入对 action 类型 skip;line 206-244 `actions[]` nested 投影删除(action 已是一等资产,自顶 doc 即可)
  - `backend/internal/outbox/es_subscriber.go` — 无改动(继续通过 `Builder.Build` 走同一管道)
- **New APIs**: 无新路径,只把现有 `POST /assets/:id/actions` 重路由
- **Dependencies**: 无新增;`action_label_registry` 已在 `backend/internal/config/` 用着

## Scope

- **In scope**:
  - `/assets/:id/actions` 4 个方法(POST / GET / PATCH / DELETE)统一改挂到 `assetHandler` 的 first-class 实现
  - 老 `actionHandler` 4 个方法退役(`routes.go` 的 4 个 wiring 切走;handler / usecase / repo 代码可保留供 backfill 复用,不删)
  - `createChildAsset("action")` 内加 label 校验(`action_label_registry.Validate`)+ ≥1ms 时长约束
  - `CreateChildAsset` 对 `asset_type="action"` hard-code `lifecycle_state="ready"`
  - 新增 `ListActionsByParent` / `UpdateActionAsset` / `SoftDeleteActionAsset` 3 个 usecase 方法 + 对应 handler
  - ES doc builder:action 类型 skip `lifecycle_state`;删除 `actions[]` nested 投影
  - 1 次 ES `_update_by_query` 清掉老 doc 里的 `actions[]` 数组
  - `docs/agents/knowledge/` 一段:grace action → `asset_type='action'` 的字段映射规则 + 老 `actions` 表数据通过 backfill 脚本回填(`/tmp/migrate_grace_to_databrew.py` 单独 issue,**不在 3268**)
- **Out of scope**:
  - `actions` 表的下线 / 老数据迁移脚本 `/tmp/migrate_grace_to_databrew.py` 改造(单独 issue,3268 不动)
  - `algo_run → action` 血缘边(3268 不写,继续走 `metadata.source_run_id`)
  - ES `lifecycle_state` mapping 调整(3268 不放宽 NOT NULL,action 走 hard-code 'ready')
  - `asset_validator.go` 修改(已覆盖 L2/L3 校验)
  - 前端 action tab 切到新接口(后续 issue,3268 只把后端做对)

## Success Criteria

- [ ] `/assets/<seg_id>/actions` 4 个方法(POST / GET / PATCH / DELETE)全部路由到 `assetHandler.*`,行为统一基于 `assets` 表;老 `actionHandler.*` 路径不再被任何路由挂载
- [ ] `POST /assets/<seg_id>/actions` body 用 `createChildAssetRequest`,返回 201 + asset
- [ ] 创建的 action 资产 `asset_type='action'`、`parent_asset_id=<seg_id>`、`lifecycle_state='ready'`(DB 兜底)、`duration_sec` 正确(>0,CYB-3267 同 fix 已就位)
- [ ] `GET /assets/<seg_id>/actions` 返回**该 seg 下所有** first-class action 资产(`assets` 表 `asset_type='action'` + `parent_asset_id=<seg_id>`),shape 与 `GET /search/assets?where=asset_type:eq:action` 命中行一致
- [ ] `PATCH /assets/<seg_id>/actions/<aid>` 只允许更新 `asset_type='action'` 行;不存在的 `aid` 或 `aid` 不属于该 parent → 404
- [ ] `DELETE /assets/<seg_id>/actions/<aid>` 软删 `assets` 行;同样校验 `aid` 存在 + parent 匹配
- [ ] `GET /search/assets?where=asset_type:eq:action` 命中全部新 action
- [ ] `GET /search/assets?where=metadata.description:ilike:拿手机` 命中相应 action(走 ES flattened metadata)
- [ ] `GET /search/assets?where=metadata.primary_label:eq:pickup` 命中受控 label 的 action
- [ ] `GET /assets/<seg_id>/lineage` 包含新增 action 作为下游子节点(`ln` 边表由 `CreateChildAsset` 自动写)
- [ ] label 注册表校验生效:传未注册的 `primary_label` 或 `labels[]` 返 422 `INVALID_ACTION`
- [ ] ES action doc 不含 `lifecycle_state` 字段(`asset_type='action'` 行)
