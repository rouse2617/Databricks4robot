# Lakehouse 查询与 Asset Tag 过滤设计

本文说明当前项目中 `Postgres`、`Iceberg`、`Trino`、`Spark/Dagster` 的查询边界，以及当算法产生大量 tag 时，资产管理页应该如何设计过滤能力。

## 1. 总体结论

当前推荐边界：

```text
资产管理列表页 / 在线管理:
Frontend -> Backend -> Postgres

湖仓分析 / 历史扫描 / 训练集 / 回放:
Frontend -> Backend -> Trino -> Iceberg

批量同步 / 回填 / 重算 / 生成快照:
Dagster/Spark -> Iceberg
```

一句话：

- `Postgres` 负责当前态和在线交互。
- `Trino + Iceberg` 负责历史、大范围扫描和聚合分析。
- `Spark/Dagster` 负责批处理、回填、重算、训练集导出。

当前本地 MVP 中，`Postgres -> Iceberg` 仍然是 Spark 手动批同步。生产形态应改为：

```text
Backend API -> Postgres
Postgres WAL -> CDC Engine(RisingWave/Debezium) -> Iceberg
Backend Lakehouse API -> Trino -> Iceberg
Dagster/Spark -> backfill / recompute / export
```

## 2. 为什么资产管理页优先走 Postgres

资产管理页通常是在线交互查询：

- 用户筛选状态、owner、reviewer。
- 用户筛选 env、task、type。
- 用户筛选 tag，例如 `quality=good`。
- 用户筛选算法状态，例如 `hand_tracking@1.2.0=failed`。
- 用户按更新时间排序、分页。
- 用户点开单个 asset 详情。

这些查询要求：

- 低延迟，通常几十毫秒到几百毫秒。
- 稳定分页。
- 权限、租户、业务规则容易加。
- 和当前业务写入保持强一致。

所以它更适合：

```text
Frontend -> Backend -> Postgres
```

而不适合：

```text
Frontend -> Backend -> Trino -> Iceberg
```

原因是 Trino 更适合批量扫描和聚合，不适合作为每次用户点击筛选时的在线 serving 数据库。

## 3. Trino + Iceberg 应该支撑哪些功能

`Trino + Iceberg` 不应该替代资产列表页的日常筛选。它更适合以下功能。

### 3.1 数据集 / 训练集构建

例如：

```text
找出所有 hand_tracking@1.2.0 = ok
并且 quality in good/excellent
并且 task = pick_item
并且 created_at 在最近 3 个月
```

输出一个固定训练快照：

```text
dataset_snapshot_id = train_hand_tracking_2026_04_v1
```

对应 Iceberg Gold 表：

```text
gold_dataset_snapshots
gold_dataset_snapshot_items
gold_training_manifests
```

### 3.2 历史重算候选

算法版本升级后，例如：

```text
sam2@1.0.0 -> sam2@1.1.0
```

需要查询：

```text
哪些历史 asset 已经跑过 sam2@1.0.0？
哪些 asset 的输入条件满足新版本重算？
哪些 asset 已经交付过，需要保留旧结果并产生新版本结果？
```

这类查询可能扫描数千万甚至上亿资产，更适合 Trino 查询 Iceberg。

### 3.3 质量分布和统计分析

例如：

```text
过去 30 天不同 quality 的分布
不同 env 下 unusable 比例
不同 task 的算法失败率
每个月新增多少 MCAP segment
各团队贡献了多少数据
```

这类是聚合分析，不应该通过资产列表接口实时计算。

### 3.4 客户交付回放和审计

例如：

```text
客户 A 在 2026-04 交付过哪些数据？
这些 asset 当时来自哪些 MCAP？
当时算法结果是什么？
是否能完整重建 delivery manifest？
```

这类查询需要 join：

```text
deliveries
delivery_items
assets
asset_algo_events
asset_mutation_outbox
```

历史越久，越适合 Iceberg。

### 3.5 算法 / Tag 生命周期追踪

后续补充事件表后，可以回答：

```text
quality=excellent 是哪个算法加的？
什么时候加的？
哪个 run_id 加的？
后来有没有被人工修改过？
```

这需要事件历史，而不是只看当前 `assets.cf_tag`。

建议新增：

```text
asset_mutation_outbox
asset_tag_events
```

入湖后形成：

```text
bronze_asset_mutations
bronze_asset_tag_events
```

## 4. 算法产生很多 tag 时，资产过滤是否适合 Postgres

结论：适合，但不要长期只依赖 `assets.cf_tag` JSONB 直接过滤。

当前模型中：

```text
assets.cf_tag  JSONB
assets.cf_algo JSONB
```

适合早期快速迭代，因为算法 tag 和算法状态变化快，不需要频繁改表结构。

但当 tag 很多、过滤很频繁、资产量上到百万级以上时，建议增加 Postgres projection 表。

## 5. 推荐的 Postgres 在线过滤模型

### 5.1 当前态主表

保留：

```sql
assets (
  asset_id,
  mcap_file_id,
  status,
  is_deleted,
  segment_locator,
  cf_meta,
  cf_algo,
  cf_tag,
  cf_files,
  created_at,
  updated_at,
  version
)
```

职责：

- 单 asset 当前态。
- 详情页。
- 写入时的一致性源。
- 少量灵活字段存储。

### 5.2 Tag 投影表

建议新增：

```sql
asset_tags (
  asset_id UUID NOT NULL,
  tag_key TEXT NOT NULL,
  tag_value TEXT NOT NULL,
  source TEXT,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (asset_id, tag_key)
);
```

推荐索引：

```sql
CREATE INDEX idx_asset_tags_key_value_asset
ON asset_tags (tag_key, tag_value, asset_id);

CREATE INDEX idx_asset_tags_key_updated
ON asset_tags (tag_key, updated_at DESC);
```

资产列表页过滤：

```sql
SELECT a.*
FROM assets a
JOIN asset_tags t
  ON t.asset_id = a.asset_id
WHERE t.tag_key = 'quality'
  AND t.tag_value = 'good'
ORDER BY a.updated_at DESC
LIMIT 50;
```

### 5.3 算法状态投影表

建议新增：

```sql
asset_algo_latest (
  asset_id UUID NOT NULL,
  algo_key TEXT NOT NULL,
  status TEXT NOT NULL,
  method TEXT,
  run_id TEXT,
  output_uri TEXT,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (asset_id, algo_key)
);
```

推荐索引：

```sql
CREATE INDEX idx_asset_algo_latest_key_status_asset
ON asset_algo_latest (algo_key, status, asset_id);

CREATE INDEX idx_asset_algo_latest_run_id
ON asset_algo_latest (run_id)
WHERE run_id IS NOT NULL;
```

资产列表页过滤：

```sql
SELECT a.*
FROM assets a
JOIN asset_algo_latest algo
  ON algo.asset_id = a.asset_id
WHERE algo.algo_key = 'hand_tracking@1.2.0'
  AND algo.status = 'failed'
ORDER BY a.updated_at DESC
LIMIT 50;
```

### 5.4 组合过滤示例

例如页面筛选：

```text
status = approved
env = warehouse
quality = good
hand_tracking@1.2.0 = ok
created_at 最近 30 天
```

Postgres 查询可以是：

```sql
SELECT a.*
FROM assets a
JOIN asset_tags quality
  ON quality.asset_id = a.asset_id
 AND quality.tag_key = 'quality'
JOIN asset_algo_latest hand_tracking
  ON hand_tracking.asset_id = a.asset_id
 AND hand_tracking.algo_key = 'hand_tracking@1.2.0'
WHERE a.status = 'approved'
  AND a.cf_meta ->> 'env' = 'warehouse'
  AND quality.tag_value = 'good'
  AND hand_tracking.status = 'ok'
  AND a.created_at >= now() - interval '30 days'
ORDER BY a.updated_at DESC
LIMIT 50;
```

如果 `env`, `task`, `type` 也是高频过滤字段，建议不要一直放在 `cf_meta` 里做 JSONB 表达式过滤，而是提升为投影列或生成列。

## 6. JSONB 直接过滤 vs 投影表

| 方式 | 适合阶段 | 优点 | 问题 |
|------|----------|------|------|
| `assets.cf_tag` JSONB 直接过滤 | MVP / 低频过滤 | 快速灵活，不用改 schema | 索引和统计信息弱，复杂组合过滤不好优化 |
| JSONB GIN / 表达式索引 | 过渡阶段 | 改动较小，可优化常见 tag | tag 很多时索引治理复杂 |
| `asset_tags` 投影表 | 推荐正式在线过滤 | 可控索引，JOIN 清晰，分页稳定 | 写入时需要维护投影 |
| `asset_algo_latest` 投影表 | 推荐正式在线过滤 | 算法状态过滤高效 | 需要和算法状态机保持一致 |
| Iceberg/Trino 查询 | 分析和历史扫描 | 适合大范围扫描和聚合 | 不适合在线分页筛选 |

## 7. 写入时如何维护投影表

后端写入 asset 或更新 tag 时：

```text
1. 更新 assets.cf_tag / assets.cf_algo
2. 同事务 upsert asset_tags / asset_algo_latest
3. 写 asset_algo_events 或 asset_mutation_outbox
```

示意：

```sql
INSERT INTO asset_tags (asset_id, tag_key, tag_value, source, updated_at)
VALUES ($1, 'quality', 'good', 'algorithm', now())
ON CONFLICT (asset_id, tag_key)
DO UPDATE SET
  tag_value = EXCLUDED.tag_value,
  source = EXCLUDED.source,
  updated_at = EXCLUDED.updated_at;
```

这样在线列表页可以高效查投影表，Iceberg 侧也可以通过 CDC 获得更结构化的数据。

## 8. 什么时候从 Postgres 切到 Trino

建议规则：

| 场景 | 推荐查询层 |
|------|------------|
| 资产列表分页 | Postgres |
| 单 asset 详情 | Postgres |
| 按 status/owner/env/tag/algo 筛选当前资产 | Postgres + projection 表 |
| 近实时运营小报表 | Postgres 或 Postgres materialized view |
| 全历史质量分布 | Trino + Iceberg |
| 训练集快照构建 | Trino + Iceberg，结果写 Gold 表 |
| 算法版本变更后的全量重算候选 | Trino + Iceberg |
| 客户历史交付回放 | Trino + Iceberg |
| 超大规模 backfill / 重算 | Spark/Dagster |

## 9. 当前项目下一步建议

建议下一阶段按这个顺序做：

1. 在 Postgres 中增加 `asset_tags` 和 `asset_algo_latest` projection 表。
2. 后端写入/更新 asset 时同步维护 projection 表。
3. 资产管理页过滤改为优先查 projection 表，而不是直接扫 JSONB。
4. 增加 `asset_mutation_outbox` 或 `asset_tag_events`，记录 tag 是谁、何时、由哪个算法或用户修改的。
5. CDC 入湖后，在 Iceberg 中形成：

```text
bronze_asset_tag_events
silver_asset_tags
silver_asset_algo_latest
gold_dataset_snapshot_items
```

6. Trino 继续负责湖仓分析页和训练/审计类查询。

## 10. 关键结论

对于“算法很多 tag，资产管理要过滤多个字段”这个问题：

```text
资产管理当前态过滤：Postgres 合适
但正式阶段应使用 asset_tags / asset_algo_latest projection 表
不要长期只依赖 JSONB 直接过滤

历史分析、训练集、重算、交付回放：Trino + Iceberg 合适
Spark/Dagster：负责批处理和回填，不负责页面实时筛选
```
