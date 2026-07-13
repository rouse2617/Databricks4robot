# CYB-3384 — planner facet engine 软开关:PG-ES 不同步时走 PG

## Why
Assets 页面用户走查:facet 计数与实际过滤结果长期不一致,例如
- `asset_type = action`,facet 显示 **7**,list 显示 **2**(多 5 条)
- `asset_type = segment`,facet 显示 **144**,list 显示 **154**(少 10 条)

**Root cause**(dev live 复现):同一个 `/api/v1/queries/run` 请求,planner 把执行拆成两个 engine:

```json
"debug_plan": {"steps": [
  {"engine": "postgres",     "mode": "filter"},   // list + total  ← PG 权威
  {"engine": "elasticsearch", "mode": "facet"}    // aggregation buckets  ← ES(带存量 drift)
]}
```

dev `/api/v1/search/sync-progress` 显示 `pg_es_gap = 7`(PG 240 asset vs ES 233 doc)。所有 outbox watermark 都对齐(`seq_lag = 0`, `consumer_lag = 0`),存量 gap 是历史 legacy 沉淀,relay/subscriber 自愈不了。

设计上 planner 用 ES 做 aggregation 是**性能优化**(prod 50M asset 时 PG `GROUP BY` 秒级、ES 毫秒),前提**假设 PG≡ES**。dev 240 asset 场景优势不明显,而不同步造成的 UI 误导则是明确的信任杀手。

## What Changes
planner 增加 **facet engine 软开关**:

- 引入 `SyncHealth` in-memory cache(30s TTL,后台 refresh),从 `SyncProgress` 读 `pg_es_gap`;
- `PGBridgePlanner.Plan` 在 `useFacets` 分支根据 `SyncHealth.PGESGap` 选路:
  - `gap == 0` → `UseESFacets=true`(现状,保留 ES 性能)
  - `gap > 0` → `UsePGFacets=true, UseESFacets=false`(fallback PG facet)
- 新增 `queryexec/postgres.FacetExecutor`,支持一期 4 个 facet field:
  - `asset_type`, `lifecycle_state`, `owner` — 直接列 `GROUP BY assets.<col>`
  - `env` — jsonb 路径 `GROUP BY metadata->>'env'`
- 其他 facet field(`mcap.*`,`tag.*`)在 fallback 分支返回空 buckets + `warnings` 提示"partial fallback"(避免全量 join 复杂度,后续 CYB 扩展);
- handler `executeCompiledRun` 在 `plan.UsePGFacets` 时调用 PG facet executor,把 buckets 塞回 `compiled.Facets`;
- env override:`DBK_FACET_ENGINE=es|pg|auto`(默认 `auto` 用软开关;`es` 强制原路径,`pg` 强制 PG);
- 补充 `sync_health` 指标(`ss_pg_es_gap`,`facet_engine_selected{engine=pg|es}`)。

## Impact
- **面向用户**:dev / 任何 drift 环境下 facet count 与 list count 严格一致,信任问题消失。
- **性能**:
  - Prod `gap==0` 场景走 ES(不变)
  - `gap>0` 场景走 PG:一期支持的 4 个字段用直接列 `GROUP BY` + 索引(`idx_assets_asset_type` / `idx_assets_lifecycle_state`)。 dev 240 asset 亚毫秒,prod 大表看基数,预期几十 ms 内可接受(gap>0 本就是异常状态,slower 也比错)。
- **老部署**:`DBK_FACET_ENGINE` 未设时默认 `auto`,行为等同现状(dev gap>0 触发 fallback,prod 保持)。
- **未覆盖 field**:`mcap.*` / `tag.*` fallback 分支返回空 bucket + warning,前端已有"facet 空时不显示"逻辑,UX 无回归但计数不显示。 CYB-3384 只覆盖用户走查里高频出错的 asset_type / lifecycle / owner / env。

## 验证
- 单测:planner_test 分 gap==0 / gap>0 / env override 三组;`postgres.FacetExecutor` table-driven 4 个 field;handler 集成 mock。
- Tier L:`go test ./backend/... -count=1` 全绿。
- dev regression:部署后重现 CYB-3382 走查,facet count == list count(直接列 4 个 field)。
- 观测:`/api/v1/metrics` 出现 `facet_engine_selected{engine=pg}` 非零(dev),`{engine=es}` 保持 prod baseline。
