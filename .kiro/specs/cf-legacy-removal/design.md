# Design — cf_* Legacy Layer Removal

> **Spec ID**: P2-9
> **设计依据**: #[[file:.kiro/specs/cf-legacy-removal/requirements.md]]

## 当前状态分析

### 数据库层

| 表 | cf_* 列 | 当前状态 |
|---|---------|----------|
| `assets` | `cf_meta`, `cf_algo`, `cf_tag`, `cf_files` | 已有 typed columns，cf_* 作为遗留 |
| `mcap_files` | `cf_meta`, `cf_process` | 已有 typed columns，cf_* 作为遗留 |
| `deliveries` | `cf_meta` | 已有 typed columns，cf_* 作为遗留 |

### 索引层

**GIN 索引** (pg-phase0.sql):
```sql
CREATE INDEX idx_assets_cf_meta_gin ON assets USING GIN (cf_meta);
CREATE INDEX idx_assets_cf_algo_gin ON assets USING GIN (cf_algo);
CREATE INDEX idx_assets_cf_tag_gin  ON assets USING GIN (cf_tag);
CREATE INDEX idx_assets_cf_files_gin ON assets USING GIN (cf_files);
CREATE INDEX idx_assets_algo_statuses_gin_active ON assets USING GIN (asset_algo_statuses(cf_algo));
```

**表达式索引** (migrations/004):
```sql
-- cf_meta 表达式索引
idx_assets_cf_meta_owner_active
idx_assets_cf_meta_reviewer_active
idx_assets_cf_meta_env_active
idx_assets_cf_meta_task_active
idx_assets_cf_meta_duration_num_active
idx_assets_cf_meta_delivery_count_active

-- cf_tag 表达式索引
idx_assets_cf_tag_priority_active
idx_assets_cf_tag_quality_active
idx_assets_cf_tag_scene_active
idx_assets_cf_tag_task_active
idx_assets_cf_tag_batch_active
```

### 代码层

**已清理**:
- `filter/types.go`: `FieldAliasMap` 已清空
- `repository/asset_repository.go`: `MergeCfAlgo` 已从 interface 移除
- `postgres/repos.go`: `MergeCfAlgo` 方法不再写入 `cf_algo`/`cf_files`

**待清理**:
- `migrations/002_seed.sql`: INSERT 语句使用 `cf_*` 列
- `migrations/004_asset_query_indexes.sql`: `cf_*` 表达式索引
- `schemas/pg-phase0.sql`: `cf_*` 列定义

## 执行设计

### Step 1: 验证无运行时依赖

1. 确认 `postgres/repos.go` 不读写 `cf_*` 列
2. 确认 `searchindex/builder.go` 不依赖 `cf_*`
3. 确认 `outbox/` 不依赖 `cf_*`
4. 检查运维脚本 (`scripts/`) 不依赖 `cf_*`

### Step 2: 更新 Seed 数据

**目标**: `migrations/002_seed.sql` 改用 typed columns + projection tables

**变更**:
- `mcap_files`: 
  - 移除 `cf_meta`, `cf_process` 列
  - 使用 `metadata`, `process_state` 列
- `assets`:
  - 移除 `cf_meta`, `cf_algo`, `cf_tag`, `cf_files` 列
  - 使用 typed columns (`owner`, `reviewer`, `retention_tier`, etc.)
  - 算法状态写入 `asset_algo_latest` 表
  - 标签写入 `asset_tags` 表
- `deliveries`:
  - 移除 `cf_meta` 列
  - 使用 `metadata` 列

### Step 3: 创建 Migration 删除列和索引

**文件**: `migrations/013_drop_cf_legacy_columns.sql`

```sql
-- 013_drop_cf_legacy_columns.sql
-- Remove cf_* legacy columns and indexes

-- Drop expression indexes on cf_meta
DROP INDEX IF EXISTS idx_assets_cf_meta_owner_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_reviewer_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_env_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_task_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_duration_num_active;
DROP INDEX IF EXISTS idx_assets_cf_meta_delivery_count_active;

-- Drop expression indexes on cf_tag
DROP INDEX IF EXISTS idx_assets_cf_tag_priority_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_quality_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_scene_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_task_active;
DROP INDEX IF EXISTS idx_assets_cf_tag_batch_active;

-- Drop GIN indexes
DROP INDEX IF EXISTS idx_assets_cf_meta_gin;
DROP INDEX IF EXISTS idx_assets_cf_algo_gin;
DROP INDEX IF EXISTS idx_assets_cf_tag_gin;
DROP INDEX IF EXISTS idx_assets_cf_files_gin;
DROP INDEX IF EXISTS idx_assets_algo_statuses_gin_active;

-- Drop cf_* columns
ALTER TABLE assets DROP COLUMN IF EXISTS cf_meta;
ALTER TABLE assets DROP COLUMN IF EXISTS cf_algo;
ALTER TABLE assets DROP COLUMN IF EXISTS cf_tag;
ALTER TABLE assets DROP COLUMN IF EXISTS cf_files;

ALTER TABLE mcap_files DROP COLUMN IF EXISTS cf_meta;
ALTER TABLE mcap_files DROP COLUMN IF EXISTS cf_process;

ALTER TABLE deliveries DROP COLUMN IF EXISTS cf_meta;
```

### Step 4: 更新 Schema 文档

**文件更新**:
1. `schemas/pg-phase0.sql`: 删除 `cf_*` 列定义和相关索引
2. `docs/review/sql.md`: 更新字段表，标注 `cf_*` 已删除
3. `backend/README.md`: 更新架构说明，移除 `cf_*` 相关内容

## 回滚策略

如需回滚：
1. `ALTER TABLE` 重新添加 `cf_*` 列 (JSONB DEFAULT '{}')
2. 重新创建 GIN 索引
3. 从投影表 backfill 到 `cf_*` 列

## 测试计划

1. **单元测试**: 确保现有测试不依赖 `cf_*`
2. **集成测试**: 运行 `make test-integration`
3. **端到端测试**: 
   - 创建 asset → 验证 projection tables 正确
   - 搜索 asset → 验证 ES 返回正确
   - 运行 outbox worker → 验证事件投递正确

## Bigtable 说明

`internal/bigtable/` 包中的 `cf_*` 引用保留不修改：
- Bigtable 已标记为 DEPRECATED
- 启动时 `STORAGE_BACKEND=bigtable` 会 fail fast
- 代码保留作为历史参考

## 时间线

| 步骤 | 估时 | 说明 |
|------|------|------|
| Step 1: 验证 | 0.5h | grep + 代码审查 |
| Step 2: Seed 更新 | 1h | 修改 + 测试 |
| Step 3: Migration | 0.5h | 创建 + 测试 |
| Step 4: 文档更新 | 0.5h | schema + README |
| **总计** | **2.5h** | |
