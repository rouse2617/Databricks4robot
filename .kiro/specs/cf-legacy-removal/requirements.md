# Requirements — cf_* Legacy Layer Removal

> **Spec ID**: P2-9
> **Priority**: P2
> **Estimated Effort**: Large (L)

## 背景

`cf_meta / cf_algo / cf_tag / cf_files / cf_process` 是 Phase 0 遗留的 JSONB 列，用于存储非结构化的元数据、算法结果、标签和文件引用。Phase 1 已完成：

- **投影表落地**：`asset_tags` / `asset_algo_latest` 已成为 tag/algo 当前态的权威来源
- **ES 索引切到投影表**：outbox-worker 从 `asset_tags` 读取 tag 数据
- **API 层双写**：写入时同时更新投影表 + `cf_*` 列（仅 `MergeCfAlgo` 仍保留）
- **Filter 别名移除**：`FieldAliasMap` 已清空，`tag.`/`file.`/`algo.` 别名不再使用

## 问题

`cf_*` 列和相关索引仍在数据库中占用存储，且：
1. Seed 数据仍依赖 `cf_*` 列
2. `migrations/004_asset_query_indexes.sql` 中有多个 `cf_*` 表达式索引
3. `pg-phase0.sql` 和 `sql.md` 仍记录 `cf_*` 列定义
4. Bigtable repo 代码（已 DEPRECATED）仍有 `cf_*` 引用

## 目标

彻底移除 `cf_*` 遗留层：

1. **filter / API / SDK 不再接受 `cf_*` 别名** — 已完成，`FieldAliasMap` 为空
2. **seed / mock / 脚本 / 旧测试改用 typed columns + projection tables**
3. **确认前端、运维脚本、reindex / backfill 不再读 `cf_*`**
4. **出 migration 删除 `cf_*` 列与相关 GIN/表达式索引**

## 影响范围

### 数据库表
- `assets`: `cf_meta`, `cf_algo`, `cf_tag`, `cf_files`
- `mcap_files`: `cf_meta`, `cf_process`
- `deliveries`: `cf_meta`

### 索引
- `idx_assets_cf_meta_gin`
- `idx_assets_cf_algo_gin`
- `idx_assets_cf_tag_gin`
- `idx_assets_cf_files_gin`
- `idx_assets_cf_meta_*` 表达式索引 (migrations/004)
- `idx_assets_cf_tag_*` 表达式索引 (migrations/004)
- `idx_assets_algo_statuses_gin_active` (依赖 `cf_algo`)

### 代码
- `migrations/002_seed.sql`: 需改用 typed columns + projection tables
- `migrations/004_asset_query_indexes.sql`: 删除 `cf_*` 相关索引
- `schemas/pg-phase0.sql`: 删除 `cf_*` 列定义
- `internal/bigtable/`: DEPRECATED，保留但不修改

## 验收标准

1. 所有 seed 数据使用 typed columns + projection tables
2. 新 migration 文件删除 `cf_*` 列和索引
3. `pg-phase0.sql` 不再包含 `cf_*` 列
4. `go build ./...` + `go test ./...` 全部通过
5. Bigtable 代码保留作为历史参考（标记 DEPRECATED）

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Bigtable 用户依赖 `cf_*` | 低 | Bigtable 已 DEPRECATED，启动时 fail fast |
| 回滚需要恢复 `cf_*` | 中 | migration 使用 `IF EXISTS`，保留 schema 文档 |
| 外部脚本依赖 `cf_*` | 低 | 先运行 grep 确认无外部依赖 |

## 相关文档

- `docs/review/sql.md` — Schema companion
- `docs/review/next-steps-tasks.md` — P2-9 任务定义
- `backend/README.md` — 架构说明
