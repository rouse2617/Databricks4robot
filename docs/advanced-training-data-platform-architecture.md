# 后训练数据平台高级架构设计

本文描述 `data-platform` 面向机器人/自动驾驶后训练场景的长期演进架构。目标不是替换当前 `PostgreSQL + OpenSearch + Iceberg + Trino` 架构，而是在其基础上增加数据集治理、训练血缘、特征/向量平台、数据质量和任务编排能力。

## 1. 架构目标

后训练数据平台要解决的不只是“存 asset”，而是完整回答以下问题：

- 如何找到适合训练/评测的数据？
- 某次训练到底用了哪些 asset？
- 数据集能否复现？
- 某批 asset 出问题后影响哪些模型？
- 某个算法版本变更后，哪些历史资产需要重算？
- 训练数据的质量分布、场景分布是否符合要求？
- 能否通过文本、图像、视频片段找相似数据？

因此长期架构应覆盖：

```text
在线资产管理
复杂检索
历史事实
训练数据集构建
训练任务追踪
数据质量检查
数据/模型血缘
多模态向量检索
批处理和重算编排
```

## 2. 总体架构

```text
Frontend / SDK / Internal Tools
  -> Backend API
      -> PostgreSQL
      -> OpenSearch / ES
      -> Trino

PostgreSQL
  -> Outbox / CDC
      -> OpenSearch / ES
      -> Iceberg
      -> Feature / Embedding Jobs

Object Storage
  -> raw MCAP
  -> derived assets
  -> thumbnails
  -> algorithm outputs
  -> dataset manifests
  -> training artifacts

Iceberg
  -> Bronze: raw synced facts
  -> Silver: normalized history
  -> Gold: dataset / replay / recompute / quality tables

Spark / Ray / Dagster
  -> dataset build
  -> feature extraction
  -> embedding generation
  -> backfill
  -> recompute
  -> export

Vector Index / Lance / Milvus / Vespa
  -> semantic search
  -> image search
  -> video clip similarity
  -> long-tail scenario mining
```

推荐职责：

| 层 | 组件 | 主要职责 |
|---|---|---|
| 在线业务层 | PostgreSQL | 权威当前态、事务、权限、幂等、状态机 |
| 检索层 | OpenSearch / ES | 模糊查询、全文检索、多字段过滤、facets、资产发现 |
| 湖仓层 | Iceberg | 历史事实、训练数据明细、审计回放、长期分析 |
| SQL 查询层 | Trino | 查询 Iceberg，服务复杂分析、训练数据筛选、报表 |
| 计算层 | Spark / Ray | 批处理、特征抽取、重算、数据导出 |
| 编排层 | Dagster | 调度 dataset build、feature build、CDC 校验、重算任务 |
| 向量检索层 | Lance / Milvus / Vespa | 多模态 embedding 检索、相似片段召回 |
| 质量与血缘层 | Data Quality / Lineage | 数据质量报告、训练血缘、影响分析 |

## 3. 当前架构是否保留

当前架构应该保留：

```text
PostgreSQL + OpenSearch + Iceberg + Trino
```

原因：

- PostgreSQL 适合作为在线权威主库。
- OpenSearch 适合做资产检索和发现。
- Iceberg 适合保存全量历史事实和训练数据明细。
- Trino 适合对 Iceberg 做复杂 SQL 查询。

不建议用单一组件替代全部能力：

| 组件 | 不适合替代什么 |
|---|---|
| PostgreSQL | 不适合承载所有 50 亿级历史明细和训练集 item |
| OpenSearch | 不适合做事务主库、审计事实源、训练快照事实源 |
| Iceberg | 不适合在线事务、毫秒级点查、权限状态更新 |
| Bigtable | 不适合复杂过滤、join、训练分析和历史统计 |

更合理的方式是分层协作。

## 4. PostgreSQL 在线模型

PostgreSQL 只保存在线业务必须稳定读写的数据。

推荐核心表：

```text
mcap_files
assets
asset_tags
asset_algo_latest
asset_events
asset_relations
deliveries
delivery_items
dataset_snapshots
training_runs
idempotency_keys
```

其中：

- `assets` 保存资产当前态。
- `asset_tags` 保存 tag 当前态投影。
- `asset_algo_latest` 保存算法最新状态投影。
- `asset_events` 保存统一业务事件，作为审计和同步源。
- `asset_relations` 保存复杂资产血缘。
- `dataset_snapshots` 保存数据集快照元信息。
- `training_runs` 保存训练任务和数据集使用记录。

注意：

```text
PostgreSQL 不存大规模 dataset_snapshot_items 明细。
PostgreSQL 不存全量训练历史事实。
PostgreSQL 不作为多模态向量检索主引擎。
```

大规模明细应进入 Iceberg 或对象存储 manifest。

## 5. 数据集治理

后训练场景中，数据集不是一次临时查询，而是需要版本化和可复现的资产。

建议抽象：

```text
dataset_definitions
dataset_snapshots
dataset_snapshot_items
dataset_quality_reports
dataset_diff_reports
```

当前阶段可以先保留：

```text
dataset_snapshots
```

长期可以演进为：

| 表 / 数据集 | 存储位置 | 说明 |
|---|---|---|
| `dataset_definitions` | PostgreSQL | 数据集定义，如名称、用途、owner、基础筛选条件 |
| `dataset_snapshots` | PostgreSQL | 某次固定下来的数据集快照元信息 |
| `dataset_snapshot_items` | Iceberg | 快照中的 asset 明细，可能百万/千万/亿级 |
| `dataset_quality_reports` | PostgreSQL + Iceberg | 数据集质量报告摘要和详细统计 |
| `dataset_diff_reports` | Iceberg | 不同快照之间新增/删除/变化的 asset |

典型流程：

```text
1. 用户通过 ES / PG / Trino 筛选候选资产
2. 后端创建 dataset_snapshot 记录
3. Spark / Trino 生成 snapshot item 明细
4. 明细写入 Iceberg 或 manifest parquet
5. PostgreSQL 保存 manifest_uri、item_count、query_spec、source_query_hash
6. 训练任务绑定 snapshot_id
```

这样可以回答：

- 这个数据集快照包含哪些 asset？
- v1 到 v2 增加了哪些场景？
- 训练效果变化是否由数据集变化引起？
- 某批有问题的数据影响了哪些数据集？

## 6. 训练任务追踪

训练任务不仅要记录模型名称，还要记录数据、代码、配置和产物。

`training_runs` 应覆盖：

```text
training_run_id
dataset_id
snapshot_id
model_name
model_version
algo_name
code_version
config_uri
data_manifest_uri
metrics
artifact_uri
status
started_at
finished_at
```

这些字段用于复现训练：

| 字段 | 作用 |
|---|---|
| `snapshot_id` | 固定当时使用的数据集版本 |
| `data_manifest_uri` | 指向实际训练读取的数据清单 |
| `code_version` | 记录训练代码版本，如 git commit 或 image tag |
| `config_uri` | 记录训练参数配置 |
| `artifact_uri` | 记录模型产物地址 |
| `metrics` | 保存训练指标摘要 |

训练链路：

```text
dataset_snapshot
  -> training_run
  -> training artifact
  -> evaluation result
  -> model registry
```

后续可扩展：

```text
model_versions
evaluation_runs
model_dataset_lineage
```

## 7. 数据血缘

机器人数据平台里的 asset 可能多级派生：

```text
raw MCAP
  -> segment
    -> clip
      -> frame_set
        -> feature / embedding
```

因此需要同时记录：

- 文件到 asset 的关系。
- asset 到子 asset 的关系。
- asset 到算法产物的关系。
- asset 到数据集快照的关系。
- 数据集快照到训练任务的关系。
- 训练任务到模型产物的关系。

当前可以用：

```text
assets.parent_asset_id
assets.root_asset_id
asset_relations
asset_events
dataset_snapshots
training_runs
```

长期可以增加：

```text
asset_lineage
model_dataset_lineage
feature_lineage
```

血缘能力：

- 某个子 clip 来自哪个原始 MCAP？
- 某个算法产物来自哪些 asset？
- 某次训练用了哪些 asset？
- 某个 asset 被删除后影响哪些数据集和模型？
- 某个算法版本升级后哪些资产和训练集要重算？

## 8. Feature Store 与 Embedding Store

后训练不只是管理 asset，还要管理特征。

推荐增加：

```text
feature_sets
feature_jobs
asset_features
asset_embeddings
```

职责：

| 概念 | 说明 |
|---|---|
| `feature_sets` | 定义一组特征，如场景特征、视觉特征、文本特征 |
| `feature_jobs` | 记录特征抽取任务 |
| `asset_features` | 保存结构化特征元信息，明细可入 Iceberg |
| `asset_embeddings` | 保存 embedding 元信息，向量本体进入 Lance/Milvus/Vespa |

存储建议：

```text
PostgreSQL:
  feature job 状态、feature set 元信息、索引引用

Iceberg:
  大规模 feature 明细、embedding 元数据、历史版本

Vector Index:
  embedding 向量检索

Object Storage:
  特征文件、embedding 文件、模型输出文件
```

典型能力：

- 文本描述召回相关片段。
- 图片找相似图片。
- 视频 clip 找相似 clip。
- 挖掘长尾失败场景。
- 查找某类动作/场景的相似训练样本。

## 9. 多模态检索

OpenSearch 适合关键词和结构化条件，向量引擎适合语义相似。

推荐混合检索：

```text
query text / image / video clip
  -> embedding model
  -> vector search
  -> candidate asset_ids
  -> OpenSearch filter / facets
  -> PostgreSQL current state
  -> final result
```

或者：

```text
OpenSearch keyword recall
  + Vector similarity recall
  + PostgreSQL permission/current-state filter
  -> rerank
  -> result
```

组件选择：

| 组件 | 适合场景 |
|---|---|
| Lance / LanceDB | 本地湖仓式向量数据、和对象存储/文件数据结合 |
| Milvus | 大规模向量检索服务 |
| Vespa | 搜索 + 向量 + 排序一体化 |
| pgvector | 小规模、简单向量能力，适合早期验证 |
| OpenSearch Vector | 搜索和向量结合，但复杂多模态场景可能需要专门向量引擎 |

建议：

```text
早期验证: pgvector / OpenSearch vector / LanceDB
中长期: Lance 或 Milvus
搜索排序一体化: Vespa
```

## 10. Iceberg 分层设计

Iceberg 是后训练历史事实层。

### Bronze

从 PostgreSQL 同步过来的原始事实：

```text
bronze_mcap_files
bronze_assets
bronze_asset_tags
bronze_asset_algo_latest
bronze_asset_events
bronze_asset_relations
bronze_deliveries
bronze_delivery_items
bronze_dataset_snapshots
bronze_training_runs
```

### Silver

清洗、展开、标准化后的明细：

```text
silver_assets_current
silver_asset_lineage
silver_asset_tag_history
silver_asset_algo_runs
silver_delivery_items
silver_training_dataset_usage
silver_feature_jobs
silver_asset_embeddings
```

### Gold

面向训练和分析的结果表：

```text
gold_dataset_snapshot_items
gold_dataset_quality_reports
gold_dataset_diff_reports
gold_recompute_candidates
gold_customer_delivery_replay
gold_quality_distribution
gold_model_dataset_lineage
```

原则：

```text
PostgreSQL 保存在线当前态和元信息。
Iceberg 保存大规模历史事实和训练明细。
Trino 查询 Iceberg。
Spark/Ray 生成训练需要的数据文件。
```

## 11. 数据质量

后训练之前必须知道数据是否可用。

建议质量维度：

```text
数据完整性
文件可访问性
时间戳合法性
传感器字段完整性
场景分布
质量等级分布
标注覆盖率
算法处理成功率
重复数据比例
异常片段比例
训练集泄漏检查
```

可引入：

```text
Great Expectations
Deequ
自定义 Spark quality jobs
```

质量结果：

```text
PostgreSQL:
  保存质量报告摘要和状态

Iceberg:
  保存详细统计、异常 asset 明细、历史趋势
```

数据集构建时建议流程：

```text
build dataset snapshot
  -> run quality checks
  -> generate quality report
  -> approve / reject snapshot
  -> allow training
```

## 12. 编排与生产化

后训练平台需要大量异步任务：

- PG 同步 ES。
- PG 同步 Iceberg。
- asset 切分。
- 算法重算。
- 特征抽取。
- embedding 生成。
- 数据集快照构建。
- 数据质量检查。
- 训练数据导出。
- Iceberg compaction。

推荐 Dagster 管理：

```text
assets ingestion job
asset split job
pg_to_iceberg sync job
opensearch indexing job
feature extraction job
embedding build job
dataset snapshot build job
dataset quality check job
training export job
recompute job
iceberg maintenance job
```

更生产化的 CDC 架构：

```text
PostgreSQL
  -> Debezium / RisingWave
  -> Kafka / Pulsar
  -> Iceberg
  -> OpenSearch
  -> downstream jobs
```

当前 MVP 可以先用：

```text
PostgreSQL -> batch / worker -> Iceberg
PostgreSQL -> outbox worker -> OpenSearch
```

## 13. 推荐演进路线

### Phase 0: 当前 MVP

目标：验证基本链路。

```text
PostgreSQL
Iceberg
Trino
Frontend lakehouse page
```

完成：

- PG 数据写入。
- 批同步到 Iceberg。
- Trino 查询 Iceberg。
- 前端展示湖仓查询结果。

### Phase 1: PostgreSQL 在线模型稳定

目标：资产管理、当前态过滤、事件审计稳定。

新增/完善：

```text
asset_tags
asset_algo_latest
asset_events
asset_relations
```

重点：

- 高频过滤字段列化。
- tag/algo 当前态投影。
- 统一事件表。
- 父子资产血缘。

### Phase 2: OpenSearch 资产检索

目标：支持数据发现。

新增：

```text
asset_search_docs
PostgreSQL -> OpenSearch indexing worker
```

能力：

- 模糊查询。
- 全文搜索。
- 多字段过滤。
- facets 聚合。
- 资产发现。

### Phase 3: Iceberg 历史事实生产化

目标：历史分析和训练数据明细进入湖仓。

新增：

```text
bronze / silver / gold Iceberg tables
Trino analytics APIs
Spark / Dagster sync jobs
```

能力：

- 历史查询。
- 审计回放。
- 算法重算候选。
- 客户交付回放。
- 质量统计。

### Phase 4: 训练数据闭环

目标：让训练数据可复现。

新增：

```text
dataset_snapshots
training_runs
gold_dataset_snapshot_items
dataset_quality_reports
```

能力：

- 固化数据集版本。
- 记录训练使用的数据版本。
- 训练数据复现。
- 数据集质量报告。
- 数据集 diff。

### Phase 5: Feature / Embedding 平台

目标：支持多模态检索和长尾挖掘。

新增：

```text
feature_sets
feature_jobs
asset_embeddings
Vector Index / Lance / Milvus / Vespa
```

能力：

- 文本语义检索。
- 图片相似检索。
- 视频片段相似检索。
- 场景 embedding。
- 长尾数据发现。

### Phase 6: 训练血缘和影响分析

目标：形成完整数据治理能力。

新增：

```text
model_dataset_lineage
asset_lineage
feature_lineage
OpenLineage / Marquez
```

能力：

- asset -> dataset -> training run -> model 的全链路追踪。
- 数据删除/召回影响分析。
- 算法版本升级影响分析。
- 模型效果变化归因。

## 14. 最终形态

最终架构不是单一数据库，而是分层数据平台：

```text
PostgreSQL:
  在线主库、事务、权限、当前态、状态机、幂等

OpenSearch / ES:
  资产检索、全文搜索、模糊查询、facets、召回

Iceberg:
  历史事实、训练明细、事件、血缘、质量报告、重算结果

Trino:
  湖仓 SQL 查询、复杂分析、训练数据筛选、报表

Spark / Ray:
  大规模计算、特征抽取、重算、导出

Dagster:
  任务编排、数据集构建、质量检查、同步调度

Lance / Milvus / Vespa:
  多模态向量检索、相似片段检索、长尾发现

Data Quality / Lineage:
  数据质量、训练血缘、影响分析、可复现治理
```

一句话总结：

```text
PostgreSQL 管当前态，
OpenSearch 管发现，
Iceberg 管历史事实，
Trino 管分析查询，
Spark/Ray 管计算，
Dagster 管流程，
Vector Index 管相似检索，
Lineage/Quality 管后训练可信度。
```

这套架构适合从当前 MVP 平滑演进到面向后训练的机器人数据平台。
