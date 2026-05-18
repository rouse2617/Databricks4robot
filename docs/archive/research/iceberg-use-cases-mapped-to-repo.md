# Iceberg 常见使用场景与 cyber-databrew 目标表映射

## 1. 目的

这份文档回答两个问题：

1. `Apache Iceberg` 在真实工程里最常见的使用场景是什么
2. 这些场景在当前 `cyber-databrew` 仓库里，应该落到哪些目标表

本文不讨论在线主库设计。默认前提是：

- 在线业务当前态在 `Postgres`
- `RisingWave` 承担 `Postgres -> Iceberg` 的实时传输
- `Iceberg` 负责历史、分析、训练集、检索投影、重算
- 原始 `MCAP` 二进制文件放对象存储，不放进 `Iceberg`

---

## 2. Iceberg 在网上最常见的 6 类使用场景

### 2.1 多引擎共享同一份大表

这是最常见的主场景。

同一份数据被：

- `Spark`
- `Trino`
- `Flink`
- `Snowflake`
- `Athena`
- `BigQuery/BigLake`

等引擎共享读取，不需要把数据复制很多份。

对你来说，这对应的是：

- 资产历史层
- 训练集层
- 搜索投影源表
- 运营分析表

### 2.2 CDC / Streaming 持续入湖

第二常见场景是：

`Postgres/Kafka/Event Stream -> 实时引擎 -> Iceberg`

这时 `Iceberg` 不是 OLTP 库，而是：

- 持续接收增量事实
- 保留 snapshot
- 给后续分析、训练和审计使用

对你来说，这对应：

- `assets` 变更
- `algo` 状态事件
- `delivery` 事件
- `mcap_files` 状态变化

### 2.3 可变分析表

很多团队用 `Iceberg` 做“可更新的数据集市”：

- `MERGE INTO`
- `UPDATE`
- `DELETE`
- `UPSERT`

但注意，这不是 OLTP 行更新，而是分析表级别的文件重写语义。

对你来说，最贴近的是：

- 资产 current-state 宽表投影
- 算法 latest-state 宽表投影
- 搜索索引源表

### 2.4 Time Travel / 审计 / 回滚 / 复现

这是 `Iceberg` 的核心强项之一。

每次写入形成 snapshot，因此非常适合：

- 回看某一时刻资产长什么样
- 重放一次批量修正前后的差异
- 冻结训练集
- 为大规模重算保留输入基线

对你来说，这直接对应：

- `assets` 历史版本
- `dataset snapshot`
- `training manifest`

### 2.5 Schema Evolution / Partition Evolution

这是很多团队采用 `Iceberg` 的重要原因。

适用于：

- 表结构经常增长
- 字段会改名或增加
- 分区策略会调整
- 不希望每次变更都重写整表

对你来说，很贴近：

- `tag`
- `algo result`
- `files`
- `lifecycle fields`

这些字段都还在快速演进。

### 2.6 训练集、特征集、导出集物化

这对你这种 `MCAP -> segment asset -> algo outputs -> dataset export` 的平台尤其重要。

`Iceberg` 在这里最常见的角色是：

- 训练样本明细表
- 数据集快照表
- 特征表
- 导出 manifest 的上游表

它不是存原始视频流，而是存：

- 标注
- 算法特征
- embedding
- 文件 URI
- 时间段定位信息

---

## 3. 当前仓库里已经有什么

当前仓库已经有清晰的业务骨架：

- `backend/internal/models/asset.go`
  - `Asset` 是 segment 级对象
  - `Files`、`AlgoResults`、`Tags`、`LifecycleMeta` 已经是湖仓友好的属性集合
- `docs/sql.md`
  - 已经把 `assets` 拆成 `cf:meta / cf:algo / cf:tag / cf:files`
- `docs/review/algo-lifecycle-and-data-model.md`
  - 已经定义好算法生命周期事件
- `docs/archive/research/postgres-risingwave-iceberg-architecture.md`
  - 已经确定当前阶段推荐链路是 `Postgres + RisingWave + Iceberg`

所以接下来最合理的做法，不是重新设计一套实体，而是把现有在线模型投影成一组 Iceberg 目标表。

---

## 4. 推荐的 6 张核心 Iceberg 目标表

下面这 6 张表，是最值得第一批落下来的。

---

### 4.1 `bronze_asset_mutations`

**对应使用场景：**

- CDC / Streaming 入湖
- 审计
- 历史回放
- 大规模修正重放

**粒度：**

- 1 行 = 1 次 asset 业务变更

**来源：**

- `Postgres.asset_mutation_outbox`

**建议字段：**

- `mutation_id`
- `asset_id`
- `segment_locator`
- `mcap_file_id`
- `op_type`
- `payload_json`
- `actor`
- `event_time`
- `source_tx_id`

**为什么它重要：**

- 它是 `assets current row` 之外的“语义真相”
- 适合 reconstruct 历史
- 适合给后续 `silver_assets_history` 喂数据

---

### 4.2 `bronze_asset_algo_events`

**对应使用场景：**

- CDC / Streaming 入湖
- 算法运行审计
- 失败分析
- 运行耗时分析

**粒度：**

- 1 行 = 1 次算法状态跃迁

**来源：**

- 当前仓库的 `asset_algo_events`

**建议字段：**

- `event_id`
- `asset_id`
- `algo_key`
- `prev_status`
- `new_status`
- `run_id`
- `reason`
- `created_at`

**它解决什么问题：**

- 分析哪个算法最常失败
- 统计某版本算法运行耗时
- 给 `silver_asset_algo_runs` 提供输入

---

### 4.3 `silver_assets_current`

**对应使用场景：**

- 多引擎共享 current-state 宽表
- 可变分析表
- 资产发现工作台的主投影源

**粒度：**

- 1 行 = 1 个当前有效 asset

**来源：**

- `Postgres.assets` CDC
- `RisingWave upsert sink`

**建议字段：**

- `asset_id`
- `segment_locator`
- `mcap_file_id`
- `start_timestamp_ns`
- `end_timestamp_ns`
- `duration_sec`
- `status`
- `owner`
- `reviewer`
- `seg_type`
- `env`
- `task`
- `delivery_count`
- `last_delivered_at`
- `retention_tier`
- `total_size_bytes`
- `tags_json`
- `files_json`
- `algo_summary_json`
- `updated_at`

**它解决什么问题：**

- 让 `Spark / Trino / 搜索投影任务` 共享同一份资产当前态
- 避免直接读线上 `Postgres`

---

### 4.4 `silver_assets_history`

**对应使用场景：**

- Time Travel
- 审计
- 回滚
- 训练复现

**粒度：**

- 1 行 = 1 个 asset 的 1 个历史版本

**来源：**

- `bronze_asset_mutations`

**建议字段：**

- `asset_id`
- `segment_locator`
- `version_no`
- `change_type`
- `valid_from`
- `valid_to`
- `state_json`
- `source_mutation_id`

**它解决什么问题：**

- “这个 asset 在某天某时是什么状态”
- “这次批量改 tag 之前和之后差了什么”
- “当时训练用的是哪一版资产状态”

---

### 4.5 `gold_asset_search_docs`

**对应使用场景：**

- 搜索投影
- 检索 serving
- Facet / Filter / 排序

**粒度：**

- 1 行 = 1 个可检索 asset 文档

**来源：**

- `silver_assets_current`
- 以及未来的 `silver_asset_tags / silver_asset_files / silver_asset_algo_latest`

**建议字段：**

- `asset_id`
- `segment_locator`
- `search_text`
- `title`
- `owner`
- `reviewer`
- `status`
- `env`
- `task`
- `seg_type`
- `tag_json`
- `algo_status_summary_json`
- `thumbnail_uri`
- `preview_manifest_uri`
- `raw_mcap_uri`
- `updated_at`

**它解决什么问题：**

- 不让前端检索直接扫在线表
- 让 `Elasticsearch` 有一张干净、稳定、扁平的上游文档表

---

### 4.6 `gold_dataset_snapshot_items`

**对应使用场景：**

- 训练集物化
- 数据集冻结
- 导出 manifest
- 训练复现

**粒度：**

- 1 行 = 1 个数据集快照里的 1 个样本

**来源：**

- `silver_assets_current`
- `silver_assets_history`
- 导出 job 的选择结果

**建议字段：**

- `dataset_snapshot_id`
- `asset_id`
- `segment_locator`
- `split`
- `label_version`
- `sample_weight`
- `raw_mcap_uri`
- `start_timestamp_ns`
- `end_timestamp_ns`
- `feature_refs_json`
- `manifest_row_num`

**它解决什么问题：**

- 精确记录一个训练集里到底包含哪些 segment
- 可以导出给 `Spark / Ray / 训练框架`
- 可以复现一次训练输入

---

## 5. 这 6 张表如何对应 6 类常见场景

| 常见场景 | 推荐表 |
| --- | --- |
| 多引擎共享大表 | `silver_assets_current`, `gold_dataset_snapshot_items` |
| CDC / Streaming 入湖 | `bronze_asset_mutations`, `bronze_asset_algo_events` |
| 可变分析表 | `silver_assets_current` |
| Time Travel / 审计 / 回滚 | `silver_assets_history` |
| Schema / Partition Evolution | `silver_assets_current`, `silver_assets_history` |
| 训练集 / 特征集 / 导出集物化 | `gold_dataset_snapshot_items` |

---

## 6. 这 6 张表之外，强烈建议补的伴随表

为了把核心链路讲清楚，前面只列了 6 张主表。但真正落地时，建议至少再补下面几张：

- `silver_asset_algo_latest`
  - 保存每个 `asset_id + algo_key` 的当前算法状态
- `silver_asset_files`
  - 保存 `raw_mcap`、算法产物、预览文件、缩略图
- `silver_asset_tags`
  - 把 tag 从 JSON 宽字段中拆出来，支持统计和治理
- `gold_dataset_snapshots`
  - 保存 snapshot 头信息，如 `selection_spec_json`、`created_by`、`source_snapshot_refs`

其中最关键的是：

- `gold_dataset_snapshot_items` 负责样本明细
- `gold_dataset_snapshots` 负责快照元数据

这两张表通常要成对出现。

---

## 7. 对当前仓库的直接映射

| 当前仓库对象 | Iceberg 目标表 |
| --- | --- |
| `assets` 当前态 | `silver_assets_current` |
| `asset_mutation_outbox` | `bronze_asset_mutations` |
| `asset_algo_events` | `bronze_asset_algo_events` |
| `Asset.AlgoResults` | `silver_asset_algo_latest` |
| `Asset.Files` | `silver_asset_files` |
| `Asset.Tags` | `silver_asset_tags` |
| `Assets Discovery` | `gold_asset_search_docs` |
| `训练集导出` | `gold_dataset_snapshot_items` |

---

## 8. 不应该让 Iceberg 承担的事情

对这个仓库，下面这些事情不该直接落在 `Iceberg`：

- 前端按钮点击后的同步事务写入
- 高频单 asset patch
- 幂等控制
- 权限和工作流控制面
- 原始 `MCAP` 二进制存储

这些职责应分别落在：

- `Postgres`
- `Backend`
- `Object Storage`

---

## 9. 一句话结论

`Iceberg` 在你这个项目里最合理的定位不是“主数据库”，而是：

**把在线资产系统沉淀成可共享、可回放、可导出、可训练、可复现的数据层。**

第一批最值得落下来的 6 张核心表是：

1. `bronze_asset_mutations`
2. `bronze_asset_algo_events`
3. `silver_assets_current`
4. `silver_assets_history`
5. `gold_asset_search_docs`
6. `gold_dataset_snapshot_items`

这 6 张表已经能覆盖 `Iceberg` 在真实工程里最常见、也最适合你当前仓库的使用场景。

---

## 10. 参考资料

- Iceberg multi-engine support  
  https://iceberg.apache.org/multi-engine-support/
- Iceberg evolution  
  https://iceberg.apache.org/docs/latest/docs/evolution/
- Iceberg Spark writes  
  https://iceberg.apache.org/docs/latest/docs/spark-writes/
- Iceberg Spark structured streaming  
  https://iceberg.apache.org/docs/latest/docs/spark-structured-streaming/
- Iceberg maintenance  
  https://iceberg.apache.org/docs/1.10.1/docs/maintenance/
- Snowflake Iceberg tables  
  https://docs.snowflake.com/en/user-guide/tables-iceberg
- Athena with Iceberg  
  https://docs.aws.amazon.com/athena/latest/ug/querying-iceberg.html
