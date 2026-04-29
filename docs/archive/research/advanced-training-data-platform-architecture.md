# 后训练数据平台高级架构设计

本文描述 `data-platform` 面向机器人/自动驾驶后训练场景的长期演进架构。目标不是替换当前 `PostgreSQL + Elasticsearch + Iceberg + Trino` 架构，而是在其基础上增加数据集治理、训练样本层、训练血缘、特征/向量平台、数据质量、任务编排，以及对标 LAS 的 `Daft + Lance` 多模态湖计算/湖存储能力。

参考业界实践：字节跳动在 EB 级 Iceberg 机器学习样本湖中，把 Iceberg 用作训练样本和特征工程底座，重点解决海量样本存储、特征调研、特征回填、版本管理、高吞吐读取和存储成本问题。这对本项目后续机器人数据后训练架构有直接参考价值。参考：[字节跳动 EB 级 Iceberg 数据湖的机器学习应用与优化](https://developer.volcengine.com/articles/7317095622338117641)。

同时，对标火山 LAS 的多模态数据湖方向时，需要把 `Daft + Lance` 作为后续核心增强层：Daft 负责多模态 DataFrame 处理、Ray 分布式执行、CPU/GPU 异构算子调度；Lance 负责图片、视频、音频、点云、embedding、tensor 等多模态样本的列式存储、高性能随机读取、版本化和向量检索能力。它们不是 PostgreSQL / Iceberg / Elasticsearch 的替代品，而是补齐 AI 多模态处理和训练读取性能的专用层。

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
训练样本表
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
      -> Elasticsearch
      -> Trino

PostgreSQL
  -> Outbox / CDC
      -> Elasticsearch
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
  -> Gold: dataset / training samples / replay / recompute / quality tables

Spark / Ray / Dagster
  -> dataset build
  -> feature extraction
  -> embedding generation
  -> backfill
  -> recompute
  -> export

Daft + Ray
  -> multimodal dataframe processing
  -> image / video / audio / point cloud operators
  -> CPU/GPU heterogeneous execution
  -> model-assisted data cleaning

Lance / Vector Index / Milvus / Vespa
  -> multimodal sample storage
  -> tensor / embedding storage
  -> high-performance random reads
  -> semantic search
  -> image search
  -> video clip similarity
  -> long-tail scenario mining
```

推荐职责：

| 层 | 组件 | 主要职责 |
|---|---|---|
| 在线业务层 | PostgreSQL | 权威当前态、事务、权限、幂等、状态机 |
| 检索层 | Elasticsearch | 模糊查询、全文检索、多字段过滤、facets、资产发现 |
| 湖仓层 | Iceberg | 历史事实、训练数据明细、审计回放、长期分析 |
| SQL 查询层 | Trino | 查询 Iceberg，服务复杂分析、训练数据筛选、报表 |
| 计算层 | Spark / Ray | 批处理、特征抽取、重算、数据导出 |
| 多模态湖计算层 | Daft + Ray | 图片/视频/音频/点云处理、DataFrame 化数据清洗、CPU/GPU 异构调度 |
| 多模态湖存储层 | Lance | 多模态样本、embedding、tensor、向量索引、高性能随机读取 |
| 编排层 | Dagster | 调度 dataset build、feature build、CDC 校验、重算任务 |
| 向量检索层 | Lance / Milvus / Vespa | 多模态 embedding 检索、相似片段召回 |
| 质量与血缘层 | Data Quality / Lineage | 数据质量报告、训练血缘、影响分析 |

## 3. 当前架构是否保留

当前架构应该保留：

```text
PostgreSQL + Elasticsearch + Iceberg + Trino
```

原因：

- PostgreSQL 适合作为在线权威主库。
- Elasticsearch 适合做资产检索和发现。
- Iceberg 适合保存全量历史事实和训练数据明细。
- Trino 适合对 Iceberg 做复杂 SQL 查询。

不建议用单一组件替代全部能力：

| 组件 | 不适合替代什么 |
|---|---|
| PostgreSQL | 不适合承载所有 50 亿级历史明细和训练集 item |
| Elasticsearch | 不适合做事务主库、审计事实源、训练快照事实源 |
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

## 5.1 Training Sample Layer

后训练场景里，训练任务不应该直接读取 PostgreSQL，也不应该通过 Backend API 一条条拉取 asset。训练真正读取的应该是 Iceberg Gold 层训练样本表，或者由 Iceberg/Spark/Ray 导出的 Parquet、Arrow、TFRecord 等样本文件。

建议增加一层：

```text
Training Sample Layer
```

它位于 Iceberg Gold 层，职责是把 asset、tag、算法结果、feature、label 拼成训练框架可直接消费的样本。

推荐表：

```text
gold_dataset_snapshot_items
gold_training_samples
gold_feature_samples
gold_eval_samples
gold_feature_branch_samples
```

职责说明：

| 表 | 职责 |
|---|---|
| `gold_dataset_snapshot_items` | 保存某个 dataset snapshot 包含的 asset 明细 |
| `gold_training_samples` | 训练主样本表，训练任务优先读取 |
| `gold_feature_samples` | 特征工程后的样本表，包含可复用特征列 |
| `gold_eval_samples` | 评测样本表，和训练样本分开管理 |
| `gold_feature_branch_samples` | 特征调研/实验分支样本表 |

训练样本层的关键字段：

```text
sample_id
asset_id
dataset_id
snapshot_id
feature_set_id
feature_set_version
sample_branch
label_version
algo_versions
storage_uri
sample_payload
created_at
```

原则：

```text
PostgreSQL:
  保存 dataset_snapshot、training_run、training_sample_export 元信息

Iceberg:
  保存训练样本行级明细、特征列、标签、历史版本

Object Storage:
  保存导出的 parquet / arrow / tfrecord / manifest

Spark / Ray:
  负责构建训练样本和导出文件
```

这样可以支持：

- 同一个 dataset snapshot 生成不同训练样本格式。
- 同一批 asset 拼接不同版本特征。
- 新特征先进入实验分支，验证后再合并为正式特征版本。
- 训练任务通过 `manifest_uri` 或 Iceberg table 复现数据。

## 5.2 特征调研、回填与分支

字节 Iceberg 机器学习样本实践里，一个重要思想是：特征调研不应该复制全量样本，而应该通过湖仓表、更新文件、分支或实验表复用主干样本数据。

本项目可以先不依赖 Iceberg 原生 branch，而采用更简单的业务分支设计：

```text
feature_set_version
sample_branch
experiment_id
feature_job_id
```

典型流程：

```text
1. 主干样本表 gold_training_samples_main 已存在
2. 算法同学提出新特征 feature_x
3. 创建 feature_job，job_type = branch_experiment
4. Spark/Ray 回填 feature_x 到 gold_feature_branch_samples
5. 使用实验分支样本训练
6. 指标通过后，将 feature_set_version 升级为正式版本
7. 后续训练读取新的 feature_set_version
```

这种方式可以避免：

- 每次特征调研复制全量训练样本。
- 新特征污染主干样本。
- 多个算法团队互相覆盖实验结果。
- 训练数据版本不可复现。

长期如果 Iceberg 分支能力、Nessie catalog 或内部数据分支能力成熟，可以把 `sample_branch` 映射到真实 Iceberg branch。

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

特征工程层需要支持多版本：

```text
feature_name
feature_set_id
feature_set_version
feature_job_id
sample_branch
source_algo_name
source_algo_version
```

这样同一个 asset 可以存在多套特征结果：

```text
asset_id | feature_set | version | branch      | value
a1       | scene_feat  | v1      | main        | ...
a1       | scene_feat  | v2      | exp-rain-v2 | ...
```

当前态和任务状态放 PostgreSQL，大规模特征值和训练样本行进入 Iceberg。向量本体进入 Lance/Milvus/Vespa，PostgreSQL 只保存索引引用和任务状态。

## 9. 多模态检索

Elasticsearch 适合关键词和结构化条件，向量引擎适合语义相似。

推荐混合检索：

```text
query text / image / video clip
  -> embedding model
  -> vector search
  -> candidate asset_ids
  -> Elasticsearch filter / facets
  -> PostgreSQL current state
  -> final result
```

或者：

```text
Elasticsearch keyword recall
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
| Elasticsearch Vector | 搜索和向量结合，但复杂多模态场景可能需要专门向量引擎 |

建议：

```text
早期验证: pgvector / Elasticsearch vector / LanceDB
中长期: Lance 或 Milvus
搜索排序一体化: Vespa
```

## 10. Iceberg 分层设计

Iceberg 是后训练历史事实层，也是训练样本层和特征工程层。

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
bronze_feature_sets
bronze_feature_jobs
bronze_training_sample_exports
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
silver_feature_samples
silver_training_sample_exports
```

### Gold

面向训练和分析的结果表：

```text
gold_dataset_snapshot_items
gold_dataset_quality_reports
gold_dataset_diff_reports
gold_training_samples
gold_feature_samples
gold_eval_samples
gold_feature_branch_samples
gold_recompute_candidates
gold_customer_delivery_replay
gold_quality_distribution
gold_model_dataset_lineage
```

原则：

```text
PostgreSQL 保存在线当前态和元信息。
Iceberg 保存大规模历史事实、训练明细、特征样本和训练样本表。
Trino 查询 Iceberg。
Spark/Ray 生成训练需要的数据文件。
```

训练读取优化建议：

```text
1. 训练任务读取 Iceberg Gold 表或导出的 Parquet / Arrow / TFRecord。
2. 不通过 Backend API 单条读取 asset。
3. 大规模训练样本按 snapshot_id、feature_set_version、date、scenario_type 等分区。
4. 定期做 Iceberg compaction，避免小文件影响训练吞吐。
5. 对高频训练样本保留 manifest_uri，便于训练框架直接读取。
6. 后续可探索 Arrow 向量化读取，减少训练数据读取瓶颈。
```

算法多版本和特征多版本建议统一落在 Iceberg 业务字段上：

```text
algo_name
algo_version
feature_set_id
feature_set_version
run_id
snapshot_id
sample_branch
```

Iceberg 的表级 snapshot 用于数据湖时间旅行；业务字段用于算法/特征/样本版本管理。两者不要混淆。

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
elasticsearch indexing job
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
  -> Elasticsearch
  -> downstream jobs
```

当前 MVP 可以先用：

```text
PostgreSQL -> batch / worker -> Iceberg
PostgreSQL -> outbox worker -> Elasticsearch
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

### Phase 2: Elasticsearch 资产检索

目标：支持数据发现。

新增：

```text
asset_search_docs
PostgreSQL -> Elasticsearch indexing worker
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

### Phase 4.5: 训练样本层和特征调研

目标：让 Iceberg 不只是历史分析层，而是训练样本和特征工程底座。

新增：

```text
training_sample_exports
feature_sets
feature_jobs
gold_training_samples
gold_feature_samples
gold_feature_branch_samples
```

能力：

- 将 asset、tag、algo result、feature、label 拼成训练样本。
- 支持特征回填和实验分支。
- 支持同一份 dataset snapshot 生成不同训练格式。
- 支持导出 Parquet / Arrow / TFRecord。
- 支持训练任务通过 manifest_uri 复现读取数据。

### Phase 4.6: Daft + Lance 多模态湖层

目标：对标 LAS 的多模态数据湖能力，补齐非结构化数据处理、高性能随机读取、向量/张量存储和训练侧读取性能。

新增：

```text
Daft + Ray processing jobs
Lance datasets
lance_dataset_refs
multimodal_operator_runs
gold_multimodal_samples
gold_lance_sample_refs
```

能力：

- 用 Daft 统一处理结构化元数据和图片、视频、音频、点云等多模态样本。
- 支持视频抽帧、图片质量检测、OCR、ASR、embedding 生成、模型辅助清洗等 AI 算子。
- 用 Ray 扩展到分布式执行，并支持 CPU/GPU 异构调度。
- 用 Lance 存储训练真正读取的多模态样本、embedding、tensor 和向量索引。
- Iceberg 继续保存训练样本索引、版本、血缘和审计事实，Lance 保存高性能样本物理数据。
- Dataset snapshot 可以同时产出 `manifest_uri`、`iceberg_table_ref` 和 `lance_dataset_uri`，服务不同训练/分析场景。

推荐边界：

```text
PostgreSQL:
  dataset snapshot、训练任务、权限、状态

Iceberg:
  历史事实、训练样本索引、样本版本、审计、重算候选

Daft:
  多模态样本处理、AI 算子执行、CPU/GPU 混合任务

Lance:
  图片/视频帧/音频片段/点云/embedding/tensor 的物理存储和随机读取

Elasticsearch:
  在线搜索、facets、资产发现
```

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

Elasticsearch:
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
Elasticsearch 管发现，
Iceberg 管历史事实，
Trino 管分析查询，
Spark/Ray 管计算，
Dagster 管流程，
Vector Index 管相似检索，
Lineage/Quality 管后训练可信度。
```

这套架构适合从当前 MVP 平滑演进到面向后训练的机器人数据平台。
