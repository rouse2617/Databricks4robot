# Design — CYB-3384

## 三个候选方案

### 方案 A(采用):planner 加软开关 + PG facet executor(部分字段)

- planner 依赖 `SyncHealthProvider`,`gap==0` 走 ES,`gap>0` fallback PG facet
- PG facet 一期只支持直接列 + jsonb path(4 field:asset_type / lifecycle_state / owner / env)
- 其他字段(`mcap.*` / `tag.*`)fallback 时返回空 bucket + warning
- env override `DBK_FACET_ENGINE=auto|es|pg`

**Pro**
- 自适应:同步好时享受 ES 性能;不同步时自动一致
- 保留生产 50M asset 场景 ES 优势
- 无破坏性变更:老部署默认 `auto` 等同现状
- 一期覆盖用户高频使用的 4 field

**Con**
- 引入新 executor + cache + planner 依赖,~500 行新代码
- 未覆盖 field 在 fallback 分支体验降级(空 buckets),需要 UX 明确
- SyncHealthCache 30s TTL 意味着 planner 选路有 30s 内的滞后 —— 但 gap 变化速度不会 30s 反复,可接受

### 方案 B(不采用):完全禁用 engine split,一律 PG

- 直接把 planner 里 `UseESFacets` 永远 false,PG facet 全量支持
- Consistency 完美,zero 复杂度

**为什么放弃**
- Prod 50M asset 场景,`mcap.*` / `tag.*` 涉及 join+GROUP BY,PG 可能秒级,把 asset list UI 拖成不可用
- 抛弃 outbox / ES 投入的整个索引方案,过激

### 方案 C(不采用):post-processing ES bucket 用 PG 重算 count

- 保留 ES facet 主路径
- 对每个 ES 返回的 bucket value,额外用 PG 跑 `COUNT(*) WHERE ... AND field=value`,覆盖 ES count
- 遍历 ES buckets 数量(通常 <20)追加 N 次小 count query

**为什么放弃**
- 无法解决"PG 有但 ES 没有"的 bucket(比如 segment 154 vs 144 —— ES 缺 10 条,post-processing 只能修 ES 已知 bucket,发现不了缺失 value)
- 仍然维持 ES 主链路,drift 依旧影响 total 判断

## 关键决策

| 决策 | 选择 | 原因 |
|---|---|---|
| Cache TTL | 30s | gap 变化速度慢(bulk purge 罕见),30s cadence 平衡 |
| Cache miss 行为 | 保守走 PG facet | fallback 优于错误,首次请求 latency 小损失 |
| 未覆盖 field | 空 bucket + warning | 前端已有"facet 空时不显示"逻辑,不引入新 UI 分支 |
| env override 值 | `auto|es|pg` | 便于诊断:强制 es 复现 bug、强制 pg 观测 fallback |
| PG facet 输出格式 | `[]queryir.FacetBucket` 复用现有 struct | handler dispatch 时不需要转换 |

## SyncHealthCache 生命周期

```
cmd/server/main.go
  ├── build SyncProgressFn (existing)
  ├── build SyncHealthCache(progressFn, refresh=30s)
  ├── ctx, cancel := context.WithCancel(ctx)
  ├── go cache.Run(ctx)           ← 后台 refresh goroutine
  └── planner := NewPGBridgePlanner(useES, cache)  ← wire
```

- 启动时立即拉一次(避免首次请求穿透)
- 每 30s tick refresh(shared HTTP timeout 5s from parent ctx)
- refresh 失败保留上次值 + 计入 metric,不影响 planner 判定
- shutdown cancel ctx,goroutine 5s 内退出

## 影响面清单

**新增**
- `internal/queryplan/sync_health.go` — cache 结构 + Provider interface
- `internal/queryexec/postgres/facet.go` — FacetExecutor

**修改**
- `internal/queryplan/planner.go` — Plan struct + Plan() 分支
- `internal/handlers/query/handler.go` — executeCompiledRun dispatch
- `cmd/server/optional.go` / `cmd/server/main.go` — wire cache
- `internal/config/config.go` — FacetEngine env
- `internal/metrics/backend.go` — FacetEngineSelected counter

**不改**
- outbox / subscriber / builder / SQL schema / migrations
- 前端(设计上前端拿的还是同一份 JSON `{items, total, facets, debug_plan}` 结构)
- ES 侧任何东西

## Rollout & 回滚

- 上线时 `DBK_FACET_ENGINE` 默认未设 = `auto`
- 若 fallback 触发性能问题 → 一行 env `DBK_FACET_ENGINE=es` 强制回 ES 路径(牺牲一致性)
- 若 PG facet 有 bug → 同样 `DBK_FACET_ENGINE=es` 拆盒紧急止血
- 无需 migration/schema rollback
