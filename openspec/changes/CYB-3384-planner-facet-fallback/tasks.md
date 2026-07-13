# Tasks — CYB-3384

## 数据结构 + cache
- [ ] `internal/queryplan/sync_health.go` — `type SyncHealth struct { PGESGap int64; UpdatedAt time.Time }`
- [ ] `internal/queryplan/sync_health.go` — `type SyncHealthCache struct` with 30s TTL,后台 goroutine refresh via `SyncProgressFn(ctx) (SyncProgress, error)`
- [ ] `cmd/server/optional.go` wire cache启动 + shutdown

## Planner
- [ ] `Plan` 增字段 `UsePGFacets bool`
- [ ] `PGBridgePlanner` 增 `SyncHealth SyncHealthProvider` 依赖
- [ ] `Plan()` 内 useFacets 分支:gap==0 → UseESFacets; gap>0 → UsePGFacets(优先直接列 field);env override 支持
- [ ] `planner_test.go` — table-driven 3 组:gap==0 走 ES / gap>0 走 PG / env=es 强制 ES

## PG Facet Executor(4 字段)
- [ ] `internal/queryexec/postgres/facet.go` — `FacetExecutor.Execute(ctx, compiled) ([]FacetBucket, error)`
- [ ] 支持 `asset_type`, `lifecycle_state`, `owner`, `env`(metadata->>'env')
- [ ] 复用 `buildListParams` 拿 WHERE + args,追加 `GROUP BY` + `ORDER BY count DESC LIMIT size`
- [ ] `facet_test.go` — table-driven 4 field + empty result + unsupported field 分支

## Handler dispatch
- [ ] `handlers/query/handler.go` `executeCompiledRun` 分支:
  - `plan.UsePGFacets` → 调 PG FacetExecutor → 塞回 `compiled.Facets`
  - 现有 `plan.UseESFacets` 路径不变
- [ ] `handler_test.go` — 补 3 用例:UseESFacets / UsePGFacets / fallback partial buckets

## Env override + metrics
- [ ] `config.Config` 增 `FacetEngine string`("auto|es|pg";default "auto")
- [ ] `metrics/backend.go` 增 `FacetEngineSelected` counter with `engine=pg|es` label

## Tier L + PR
- [ ] worktree 内 `go test ./backend/... -count=1` 全绿
- [ ] `go build ./...` 通过
- [ ] biome / prettier 若涉及前端(本改动仅后端)不需要
- [ ] 推 branch `fix/CYB-3384-planner-facet-fallback`(worktree 已在此 branch,rename 一次)
- [ ] PR → base dev

## dev 验证
- [ ] 部署后 curl `/api/v1/queries/run` with facets → `debug_plan.steps` 含 `{"engine":"postgres","mode":"facet"}`
- [ ] Facet counts == PG list counts(asset_type / lifecycle / owner / env)
- [ ] `/api/v1/metrics` `facet_engine_selected{engine="pg"}` 递增
