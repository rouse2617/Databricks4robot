# 资产模型扩展设计

本文设计 cyber-databrew 的资产模型扩展方案，目标是从当前以 MCAP / segment 为中心的通用资产登记，演进到可承载 Dataset、AnnotationResult、ML Model、EvaluationReport 等 Data+AI 资产的统一目录、搜索、版本、血缘和治理模型。

## 1. 当前资产模型分析

### 1.1 当前数据表

当前资产主链路由 `assets`、`logical_assets`、`asset_relations` 组成。

| 表 | 职责 | 关键字段 |
|---|---|---|
| `assets` | 资产 revision/current row 的事实表；承载生命周期、时间窗、存储位置、标签投影、低频 metadata | `asset_id`, `mcap_file_id`, `asset_type`, `lifecycle_state`, `storage_uri`, `metadata`, `logical_asset_id`, `revision`, `is_current`, `version` |
| `logical_assets` | 同一逻辑资产的版本族协调表 | `logical_asset_id`, `asset_type`, `display_name`, `description`, `current_revision`, `total_revisions`, `metadata` |
| `asset_relations` | 多父、多子、复杂血缘边 | `parent_asset_id`, `child_asset_id`, `relation_type`, `method`, `algo_name`, `algo_version`, `run_id` |

`assets` 现有字段可分为几组：

| 类别 | 字段 |
|---|---|
| 身份 | `asset_id`, `logical_asset_id`, `revision`, `is_current`, `version` |
| 物理来源 | `mcap_file_id`, `storage_uri`, `files`, `algo_inputs_uris`, `annot_inputs_uris` |
| 时间窗 | `start_timestamp_ns`, `end_timestamp_ns`, `duration_ms`, `segment_locator` |
| 类型与状态 | `asset_type`, `lifecycle_state`, legacy `status` |
| 层级 | `asset_level`, `parent_asset_id`, `root_asset_id`, `segment_index`, `parent_start_offset_ms`, `parent_end_offset_ms` |
| 治理 | `owner`, `reviewer`, `retention_tier`, `expire_at`, `tenant_id`, `project_id` |
| 扩展 | `metadata JSONB`, `files JSONB`, legacy `cf_*` |

注意：当前 `assets` 表本身没有 `name` 字段；可展示名称位于 `logical_assets.display_name`。如果要让所有资产都有稳定名称，应优先使用 `logical_assets.display_name`，不要再向 `assets` 增加重复的 `name` 列，除非有 revision 级名称差异的明确需求。

### 1.2 当前 asset_type

现有写入校验中可识别的资产类型：

| `asset_type` | 当前语义 | 父子约束 |
|---|---|---|
| `raw_mcap` | 原始 MCAP 对应的资产登记 | root，无需 parent |
| `segment` | 从 raw MCAP 切出的时间段 | parent 必须是 `raw_mcap` |
| `clip` | 从 segment 切出的片段 | parent 必须是 `segment` |
| `frame` | 从 segment 采样的帧集合/帧资产 | parent 必须是 `segment` |
| `task` | 标注/处理任务资产 | parent 可为 `segment` 或 `task` |
| `action` | segment/task 上的动作类子资产 | parent 可为 `segment` 或 `task` |
| `derived_asset` | 聚合、融合、跨 MCAP 派生产物 | 可无 `parent_asset_id`，多父关系走 `asset_relations` |

`asset_relations.relation_type` 当前受 CHECK 约束限制为：

```text
split_from, derived_from, contains, sampled_from, merged_from, revision_of
```

### 1.3 当前 ES 索引

当前 `assets` ES index 采用一份统一 mapping：

| 字段 | 类型 | 说明 |
|---|---|---|
| `asset_id`, `mcap_file_id`, `asset_type`, `lifecycle_state`, `owner`, `reviewer` | `keyword` | 精确过滤与聚合 |
| `metadata` | `flattened` | 低频扩展字段，支持 `metadata.<key>` 过滤 |
| `tags_flat` | `flattened` | tag 快速过滤 |
| `tags`, `algos`, `actions` | `nested` | 多值结构化查询 |
| `mcap` | object | MCAP 侧字段投影 |
| `start_timestamp_ns`, `end_timestamp_ns`, `duration_ms`, `created_at`, `updated_at` | numeric/date | 时间与排序 |

### 1.4 核心限制

| 限制 | 影响 |
|---|---|
| `asset_type` 规则硬编码 | 新增类型需要改 Go validator、OpenAPI、Query registry、ES builder、测试，多处同步成本高 |
| 类型专属 metadata 无 schema | `metadata JSONB` 可放任意字段，但缺少必填、枚举、范围、引用完整性和版本演进能力 |
| Query/ES 字段注册偏通用 | 新类型字段可进入 `metadata`，但排序、聚合、范围查询、字段提示和 API 契约不稳定 |
| 关系类型 CHECK 过窄 | AI 资产链路中的训练、评测、标注、快照关系无法表达，必须扩展 relation_type |
| `mcap_file_id` 仍嵌入资产核心模型 | 对 Dataset、Model、Report 等非 MCAP 资产不自然；当前只有 `derived_asset` 允许 NULL |
| `assets` 缺少类型 schema 版本 | 无法表达 `ml_model.metadata_schema_version=v1` 这类向后兼容契约 |
| 搜索文档无类型专属命名空间 | 所有类型字段混在 `metadata`，会造成字段冲突，例如 `version` 是模型框架版本还是资产 row version |

## 2. 新增资产类型设计

推荐新增 4 个标准类型：

| `asset_type` | 语义 | 是否需要 `mcap_file_id` | 推荐 lifecycle |
|---|---|---:|---|
| `dataset` | 可训练、评测或分析的数据集快照 | 否 | `created -> processing -> ready -> archived` |
| `annotation_result` | 标注产物或标注集合 | 否 | `created -> processing -> ready/rejected -> archived` |
| `ml_model` | 训练完成的模型 artifact | 否 | `created -> processing -> ready -> delivered/archived` |
| `evaluation_report` | 模型在数据集上的评测报告 | 否 | `created -> ready -> archived` |

为避免字段冲突，类型专属字段统一放在：

```json
{
  "metadata": {
    "schema_version": "asset.model.ml_model.v1",
    "type_metadata": {
      "...": "..."
    }
  }
}
```

ES 投影时同时保留 `metadata` flattened，并将高频字段投影到类型命名空间：

```text
ml_model.*
dataset.*
annotation_result.*
evaluation_report.*
```

### 2.1 ML Model

#### 专属字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `framework` | enum | 是 | `pytorch`, `tensorflow`, `onnx`, `sklearn`, `xgboost`, `other` |
| `framework_version` | string | 是 | 训练/导出框架版本，例如 `2.3.1` |
| `architecture` | string | 是 | 模型架构，例如 `resnet50`, `pointpillar`, `transformer_encoder` |
| `input_schema` | object | 是 | 输入 tensor/table schema |
| `output_schema` | object | 是 | 输出 schema |
| `metrics` | object | 否 | `accuracy`, `precision`, `recall`, `f1`, `auc`, `map`, `loss` 等 |
| `training_runtime` | object | 否 | 训练 runtime，例如 CUDA、Python、镜像、commit |
| `quantization` | enum/object | 否 | `none`, `fp16`, `int8`, `int4`, `dynamic`, `static`；可带校准集 |
| `artifact_uri` | string | 是 | 模型 artifact 地址；可与 `storage_uri` 相同 |
| `model_size_bytes` | integer | 否 | artifact 大小 |
| `license` | string | 否 | 模型许可 |

#### 校验规则

| 规则 | 说明 |
|---|---|
| `framework` 必须在枚举内 | 未知框架写 `other`，并要求 `framework_name` |
| `framework_version` 必须非空 | 建议 SemVer；不强制，因为 PyTorch nightly/ONNX opset 可能非标准 SemVer |
| `input_schema` / `output_schema` 必须包含 `format` 与 `fields` 或 `tensors` | 便于自动生成推理 API 文档 |
| `metrics.*` 必须是 number | `accuracy/precision/recall/f1/auc/map` 范围 `[0,1]`；`loss` 范围 `[0,+inf)` |
| `quantization.mode` 必须在枚举内 | `int8/int4/static` 建议要求 `calibration_dataset_asset_id` |
| `artifact_uri` 必须是 `gs://`, `s3://`, `bq://`, `file://` 或平台允许的 URI scheme | 写入时做 scheme 白名单 |

#### ES mapping

```json
{
  "ml_model": {
    "properties": {
      "framework": { "type": "keyword" },
      "framework_version": { "type": "keyword" },
      "architecture": { "type": "keyword", "fields": { "text": { "type": "text" } } },
      "quantization_mode": { "type": "keyword" },
      "artifact_uri": { "type": "keyword" },
      "model_size_bytes": { "type": "long" },
      "metrics": { "type": "flattened" },
      "accuracy": { "type": "double" },
      "precision": { "type": "double" },
      "recall": { "type": "double" },
      "f1": { "type": "double" },
      "auc": { "type": "double" },
      "map": { "type": "double" },
      "loss": { "type": "double" }
    }
  }
}
```

### 2.2 Dataset

#### 专属字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `data_format` | enum | 是 | `parquet`, `csv`, `jsonl`, `image`, `video`, `mcap`, `lidar`, `iceberg`, `delta`, `other` |
| `record_count` | integer | 否 | 记录数；图像/点云可为样本数 |
| `size_bytes` | integer | 否 | 数据总大小 |
| `manifest_uri` | string | 是 | 数据集 manifest 或 Iceberg table URI |
| `schema_uri` | string | 否 | 外部 schema 文件 |
| `is_annotated` | boolean | 是 | 是否已有标注 |
| `annotation_type` | enum | 条件必填 | `classification`, `detection_2d`, `detection_3d`, `segmentation`, `tracking`, `caption`, `qa`, `other` |
| `time_range` | object | 否 | `start_time`, `end_time`，ISO8601 |
| `source` | object | 是 | `source_type`, `source_system`, `source_uri`, `collector`, `license` |
| `partitioning` | object | 否 | 分区字段、日期粒度、Iceberg snapshot |
| `quality` | object | 否 | 缺失率、重复率、抽检分数 |

#### 校验规则

| 规则 | 说明 |
|---|---|
| `data_format` 必须在枚举内 | 自定义格式用 `other` + `format_name` |
| `record_count >= 0`, `size_bytes >= 0` | 未计算可缺省，不允许负数 |
| `is_annotated=true` 时 `annotation_type` 必填 | 如果有多个标注类型，使用数组 `annotation_types` |
| `time_range.start_time <= time_range.end_time` | 时间必须为 RFC3339 |
| `manifest_uri` 必填 | 对 BigLake Iceberg 推荐 `bq://project.dataset.table@snapshot` 或 `gs://.../metadata.json` |
| `source.source_type` 必须在枚举内 | `raw_mcap`, `warehouse`, `upload`, `external`, `synthetic`, `generated` |

#### ES mapping

```json
{
  "dataset": {
    "properties": {
      "data_format": { "type": "keyword" },
      "record_count": { "type": "long" },
      "size_bytes": { "type": "long" },
      "manifest_uri": { "type": "keyword" },
      "is_annotated": { "type": "boolean" },
      "annotation_type": { "type": "keyword" },
      "time_start": { "type": "date" },
      "time_end": { "type": "date" },
      "source_type": { "type": "keyword" },
      "source_system": { "type": "keyword" },
      "source_uri": { "type": "keyword" },
      "quality": { "type": "flattened" },
      "partitioning": { "type": "flattened" }
    }
  }
}
```

### 2.3 AnnotationResult

#### 专属字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `tool` | string | 是 | 标注工具，例如 `label-studio`, `cvatex`, `custom` |
| `tool_version` | string | 否 | 工具版本 |
| `annotation_schema_version` | string | 是 | 标注 schema 版本 |
| `annotators` | array[string] | 否 | 标注者 user id / email |
| `reviewers` | array[string] | 否 | 复核者 |
| `quality_score` | number | 否 | 质量分数 `[0,1]` |
| `coverage` | number | 是 | 覆盖率 `[0,1]` |
| `label_count` | integer | 否 | 标注对象数量 |
| `artifact_uri` | string | 是 | 标注结果文件或表 URI |
| `annotation_type` | enum | 是 | 同 Dataset `annotation_type` |

#### 校验规则

| 规则 | 说明 |
|---|---|
| `tool`、`annotation_schema_version`、`artifact_uri` 必填 | 保障可复现 |
| `coverage` 必填且范围 `[0,1]` | 标注结果必须说明覆盖程度 |
| `quality_score` 范围 `[0,1]` | 可为空，表示未质检 |
| `annotators` 长度建议 <= 500 | 大规模众包人员应存外部明细表/manifest |
| `annotation_type` 必须与源 Dataset 的允许标注类型兼容 | 通过 relation 校验或异步审计实现 |

#### ES mapping

```json
{
  "annotation_result": {
    "properties": {
      "tool": { "type": "keyword" },
      "tool_version": { "type": "keyword" },
      "annotation_schema_version": { "type": "keyword" },
      "annotation_type": { "type": "keyword" },
      "annotators": { "type": "keyword" },
      "reviewers": { "type": "keyword" },
      "quality_score": { "type": "double" },
      "coverage": { "type": "double" },
      "label_count": { "type": "long" },
      "artifact_uri": { "type": "keyword" }
    }
  }
}
```

### 2.4 EvaluationReport

#### 专属字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `model_asset_id` | string | 是 | 被评测模型 |
| `dataset_asset_id` | string | 是 | 评测数据集 |
| `metrics` | object | 是 | 指标详情 |
| `metric_groups` | array[object] | 否 | 按类别、场景、slice 拆分指标 |
| `evaluated_at` | datetime | 是 | 评测时间 |
| `evaluation_tool` | string | 是 | 评测工具 |
| `evaluation_tool_version` | string | 否 | 工具版本 |
| `report_uri` | string | 是 | 报告 artifact URI |
| `run_id` | string | 否 | 关联 algo/workflow run |
| `threshold_result` | enum | 否 | `pass`, `warning`, `fail`, `not_applicable` |

#### 校验规则

| 规则 | 说明 |
|---|---|
| `model_asset_id` 必须引用 `asset_type=ml_model` | 写入时可同步校验 |
| `dataset_asset_id` 必须引用 `asset_type=dataset` 或 `annotation_result` | 取决于评测是否需要标注 |
| `metrics` 必须非空且 value 为 number/object | 支持 scalar metric 和分组 metric |
| 常见比例指标范围 `[0,1]` | `accuracy`, `precision`, `recall`, `f1`, `auc`, `map` |
| `evaluated_at` 必须 RFC3339 | 不允许未来时间超过可配置容忍窗口，例如 10 分钟 |
| `report_uri` 必填 | 详情大对象不直接塞入 `metadata` |

#### ES mapping

```json
{
  "evaluation_report": {
    "properties": {
      "model_asset_id": { "type": "keyword" },
      "dataset_asset_id": { "type": "keyword" },
      "evaluation_tool": { "type": "keyword" },
      "evaluation_tool_version": { "type": "keyword" },
      "evaluated_at": { "type": "date" },
      "report_uri": { "type": "keyword" },
      "run_id": { "type": "keyword" },
      "threshold_result": { "type": "keyword" },
      "metrics": { "type": "flattened" },
      "accuracy": { "type": "double" },
      "precision": { "type": "double" },
      "recall": { "type": "double" },
      "f1": { "type": "double" },
      "auc": { "type": "double" },
      "map": { "type": "double" },
      "loss": { "type": "double" }
    }
  }
}
```

## 3. 类型间关系建模

### 3.1 推荐新增 relation_type

现有关系类型不足以表达数据、标注、训练、评测链路。建议扩展 `asset_relations.relation_type`：

| relation_type | parent -> child | 语义 |
|---|---|---|
| `annotated_from` | Dataset -> AnnotationResult | 标注结果来自某数据集 |
| `materialized_from` | Dataset/AnnotationResult -> Dataset | 训练集、验证集、测试集快照由上游数据/标注物化而来 |
| `trained_from` | Dataset/AnnotationResult -> ML Model | 模型由训练数据或标注集训练得到 |
| `validated_on` | Dataset/AnnotationResult -> ML Model | 模型训练/调参时使用的验证集 |
| `tested_on` | Dataset/AnnotationResult -> EvaluationReport | 报告基于该评测集产生 |
| `evaluates` | ML Model -> EvaluationReport | 报告评测该模型 |
| `compares_to` | ML Model -> EvaluationReport | 报告中作为 baseline/对照模型 |
| `calibrated_from` | Dataset -> ML Model | 量化/校准数据集 |
| `generated_by` | ML Model -> Dataset/AnnotationResult | 模型生成的合成数据或伪标签 |

保留现有 `derived_from`, `merged_from`, `contains`, `sampled_from`, `split_from`, `revision_of`，用于通用派生、集合成员、切分采样和版本关系。

### 3.2 关系图

```text
 raw_mcap
    |
    | split_from
    v
 segment / clip / frame
    |
    | materialized_from
    v
 Dataset(raw/snapshot)
    |
    | annotated_from
    v
 AnnotationResult
    |
    | materialized_from
    v
 Dataset(train/val/test)
    |
    | trained_from
    v
 ML Model
    | \
    |  \ calibrated_from
    |   \
    |    Dataset(calibration)
    |
    | evaluates
    v
 EvaluationReport
    ^
    |
    | tested_on
 Dataset(test or benchmark)
```

`asset_relations` 的方向保持 `parent_asset_id` 为上游、`child_asset_id` 为下游。查询上游血缘时从 child 反查 parent；查询下游影响面时从 parent 查 child。

### 3.3 关系约束

| relation_type | parent type | child type | 强度 |
|---|---|---|---|
| `annotated_from` | `dataset` | `annotation_result` | 强校验 |
| `materialized_from` | `dataset`, `annotation_result`, `raw_mcap`, `segment`, `clip`, `frame`, `derived_asset` | `dataset` | 强校验 |
| `trained_from` | `dataset`, `annotation_result` | `ml_model` | 强校验 |
| `validated_on` | `dataset`, `annotation_result` | `ml_model` | 建议校验 |
| `tested_on` | `dataset`, `annotation_result` | `evaluation_report` | 强校验 |
| `evaluates` | `ml_model` | `evaluation_report` | 强校验 |
| `compares_to` | `ml_model` | `evaluation_report` | 建议校验 |
| `generated_by` | `ml_model` | `dataset`, `annotation_result` | 建议校验 |

不建议把所有关系强行塞进 `parent_asset_id`。`parent_asset_id` 继续服务简单树状层级，AI 资产链路统一用 `asset_relations`，否则训练集多源、模型多数据集、报告多模型的场景会被单父字段限制。

## 4. 扩展性设计

### 4.1 方案 A：通用属性 + `metadata JSONB`

| 维度 | 评价 |
|---|---|
| 优点 | 改表少；新增类型快；适合早期探索；当前 `assets.metadata` 已存在 |
| 缺点 | 类型字段无强 schema；索引、排序、聚合能力弱；字段冲突风险高；API 文档难稳定 |
| 适用 | 低频字段、长尾类型、PoC 类型 |

### 4.2 方案 B：每种类型独立表

示例：

```sql
CREATE TABLE ml_model_assets (
  asset_id TEXT PRIMARY KEY REFERENCES assets(asset_id),
  framework TEXT NOT NULL,
  framework_version TEXT NOT NULL,
  architecture TEXT NOT NULL,
  input_schema JSONB NOT NULL,
  output_schema JSONB NOT NULL,
  metrics JSONB NOT NULL DEFAULT '{}',
  quantization JSONB NOT NULL DEFAULT '{}',
  artifact_uri TEXT NOT NULL
);
```

| 维度 | 评价 |
|---|---|
| 优点 | DB 约束强；SQL 查询和排序清晰；字段类型稳定；适合高频核心类型 |
| 缺点 | 每加类型都要 migration/repo/model/API/ES 同步；早期迭代慢；跨类型搜索复杂 |
| 适用 | 稳定、高价值、高查询频率的类型，例如 `dataset`, `ml_model` |

### 4.3 方案 C：混合方案

推荐采用混合方案：

1. `assets` 保留通用字段。
2. `assets.metadata.type_metadata` 存类型专属 payload。
3. 新增 `asset_type_schemas` 注册表，记录 JSON Schema、ES 投影、API 能力。
4. 对高频字段做 ES 投影。
5. 对稳定且高频查询的类型，二期增加专属投影表。

建议表：

```sql
CREATE TABLE asset_type_schemas (
  asset_type TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  json_schema JSONB NOT NULL,
  es_mapping JSONB NOT NULL DEFAULT '{}',
  query_fields JSONB NOT NULL DEFAULT '[]',
  lifecycle_policy JSONB NOT NULL DEFAULT '{}',
  relation_policy JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (asset_type, schema_version)
);
```

推荐写入结构：

```json
{
  "asset_type": "ml_model",
  "storage_uri": "gs://models/resnet50/model.onnx",
  "metadata": {
    "schema_version": "asset.model.ml_model.v1",
    "type_metadata": {
      "framework": "onnx",
      "framework_version": "1.16.0",
      "architecture": "resnet50",
      "input_schema": {"format": "tensor", "tensors": []},
      "output_schema": {"format": "tensor", "tensors": []},
      "metrics": {"accuracy": 0.93},
      "quantization": {"mode": "int8"},
      "artifact_uri": "gs://models/resnet50/model.onnx"
    }
  }
}
```

### 4.4 推荐决策

推荐：**近期用 JSONB + schema registry + ES typed projection；中期为 `dataset` 和 `ml_model` 增加专属投影表。**

理由：

| 决策点 | 结论 |
|---|---|
| 新类型会快速变化 | 先用 JSON Schema 管控 payload，避免频繁 migration |
| 搜索是主要入口 | ES typed projection 能满足筛选、聚合、排序和展示 |
| PG 仍是真相源 | `assets.metadata` 保留完整原始 payload |
| Dataset/Model 会成为核心资产 | 当字段稳定后再建 `dataset_assets`, `ml_model_assets`，提升 SQL 分析与完整性 |
| 对标 yx 资产模型 3.0 | 模型应从元信息管理扩展到非结构化数据、标注、模型、报告的链路治理；混合方案更利于渐进演进 |

## 5. API 变更

### 5.1 资产创建

保留通用入口：

```http
POST /api/v1/assets
```

请求体增加明确的类型 payload 约定：

```json
{
  "asset_type": "dataset",
  "logical_asset": {
    "display_name": "warehouse-camera-train-v1",
    "description": "training snapshot for warehouse camera detection"
  },
  "storage_uri": "bq://project.dataset.table@snapshot-123",
  "metadata": {
    "schema_version": "asset.model.dataset.v1",
    "type_metadata": {
      "data_format": "iceberg",
      "record_count": 1200000,
      "size_bytes": 9876543210,
      "manifest_uri": "gs://bucket/table/metadata/v1.metadata.json",
      "is_annotated": true,
      "annotation_type": "detection_2d",
      "source": {
        "source_type": "warehouse",
        "source_system": "bigquery"
      }
    }
  }
}
```

不建议一开始为每种类型实现完整独立 CRUD，例如：

```text
POST /api/v1/models
POST /api/v1/datasets
POST /api/v1/evaluation-reports
```

原因：会复制 handler/usecase/repository 逻辑，也容易破坏统一版本、血缘、标签、事件、搜索链路。

推荐折中：

| API | 用途 |
|---|---|
| `POST /api/v1/assets` | 唯一通用创建入口 |
| `GET /api/v1/assets/{id}` | 返回通用字段 + `type_metadata` |
| `PATCH /api/v1/assets/{id}` | 通用更新，禁止改 `asset_type` |
| `POST /api/v1/assets/{id}/relations` | 增加血缘边 |
| `GET /api/v1/assets/{id}/lineage` | 查看上下游链路 |
| `GET /api/v1/asset-types` | 列出支持类型、schema、字段能力 |
| `GET /api/v1/asset-types/{asset_type}/schema` | 返回 JSON Schema 和字段能力 |

可选增加类型语义别名端点，但内部仍调用通用 usecase：

| 端点 | 定位 |
|---|---|
| `POST /api/v1/ml-models` | 只做 `asset_type=ml_model` 的请求体验优化 |
| `POST /api/v1/datasets` | 数据平台用户友好的薄封装 |
| `POST /api/v1/evaluation-reports` | 自动创建 `evaluates/tested_on` 关系 |

这些别名端点不是独立资源模型，不应绕过 `assets`。

### 5.2 搜索返回

通用 Query API 继续作为主入口：

```http
POST /api/v1/queries/run
```

返回每条资产时增加：

```json
{
  "asset_id": "abcd1234",
  "asset_type": "ml_model",
  "logical_asset_id": "lm000001",
  "display_name": "resnet50-int8",
  "storage_uri": "gs://models/resnet50/model.onnx",
  "type_metadata": {
    "framework": "onnx",
    "architecture": "resnet50",
    "metrics": {"accuracy": 0.93}
  },
  "type_summary": {
    "framework": "onnx",
    "architecture": "resnet50",
    "accuracy": 0.93,
    "quantization": "int8"
  }
}
```

`type_metadata` 是完整 payload；`type_summary` 是 UI 列表和搜索结果卡片需要的稳定摘要，字段由 `asset_type_schemas.query_fields` 或代码 registry 决定。

### 5.3 类型专属校验

新增 `AssetTypeRegistry`，在 handler/usecase 层执行：

```text
CreateAsset
  -> normalize common fields
  -> load asset_type schema
  -> validate metadata.type_metadata by JSON Schema
  -> validate relation policy if request includes relations
  -> persist assets row
  -> persist asset_relations rows
  -> append asset_created event
  -> project ES document
```

校验错误建议：

| 状态码 | code | 场景 |
|---:|---|---|
| 400 | `INVALID_ARGUMENT` | JSON 结构错误、非法 URI、非法时间 |
| 422 | `UNSUPPORTED_ASSET_TYPE` | 未注册类型 |
| 422 | `ASSET_TYPE_SCHEMA_VIOLATION` | 类型专属字段不满足 schema |
| 422 | `INVALID_ASSET_RELATION` | 关系类型和资产类型不匹配 |
| 409 | `DUPLICATE_ASSET_ID` | 指定 asset_id 冲突 |

### 5.4 类型专属查询与排序

Query IR 字段建议使用命名空间：

| 字段 | 示例 |
|---|---|
| `ml_model.framework` | `{"field":"ml_model.framework","op":"eq","value":"onnx"}` |
| `ml_model.metrics.accuracy` | `{"field":"ml_model.metrics.accuracy","op":"gte","value":0.9}` |
| `dataset.data_format` | `{"field":"dataset.data_format","op":"eq","value":"parquet"}` |
| `dataset.record_count` | 排序 `desc` |
| `annotation_result.quality_score` | 范围过滤 |
| `evaluation_report.evaluated_at` | 时间排序 |

字段能力由 registry 控制：

```yaml
resources:
  assets:
    fields:
      - field: ml_model.framework
        filter_engines: [postgres, elasticsearch]
        facet_engines: [elasticsearch]
      - field: ml_model.metrics.accuracy
        filter_engines: [elasticsearch]
        sort_engines: [elasticsearch]
      - field: dataset.record_count
        filter_engines: [elasticsearch]
        sort_engines: [elasticsearch]
```

PG 初期可通过 `metadata #>> '{type_metadata,framework}'` 支持少量字段；高频排序和聚合优先走 ES。等字段稳定后，把 `dataset.record_count`、`dataset.size_bytes`、`ml_model.framework`、`ml_model.architecture` 等落到专属投影表。

## 6. 分阶段实施建议

### Phase 0：设计与注册基础

目标：不引入大规模 runtime 改动，先定契约。

| 工作 | 说明 |
|---|---|
| 定义 `asset_type_schemas` 设计 | JSON Schema、ES projection、query fields、relation policy |
| 扩展 relation_type 列表 | 增加 AI 资产链路关系 |
| 确定 metadata 结构 | `metadata.schema_version` + `metadata.type_metadata` |
| 补充 API/OpenAPI 契约 | 通用创建、查询、schema discovery |

### Phase 1：先支持 Dataset + AnnotationResult

优先原因：数据集和标注结果是模型训练/评测的前置条件，也最贴近当前 MCAP/segment/annotation 业务。

| 类型 | 范围 |
|---|---|
| `dataset` | 支持 manifest、format、record_count、size_bytes、annotation 状态、source、time_range |
| `annotation_result` | 支持 tool、schema_version、annotators、quality_score、coverage、artifact_uri |
| relation | `annotated_from`, `materialized_from` |
| ES | 增加 `dataset.*`, `annotation_result.*` projection |
| API | 通用 `POST /assets` + `GET /asset-types/{type}/schema` |

### Phase 2：支持 ML Model + EvaluationReport

| 类型 | 范围 |
|---|---|
| `ml_model` | framework、architecture、input/output schema、metrics、quantization、artifact_uri |
| `evaluation_report` | model/dataset 引用、metrics、evaluated_at、tool、report_uri |
| relation | `trained_from`, `validated_on`, `tested_on`, `evaluates`, `compares_to`, `calibrated_from` |
| API | 创建 EvaluationReport 时自动写 `evaluates` 和 `tested_on` 边 |
| ES | 增加模型指标与报告指标的 typed projection |

### Phase 3：稳定类型投影表

当查询模式稳定后，增加专属表：

| 表 | 目的 |
|---|---|
| `dataset_assets` | 数据集大小、记录数、format、manifest、annotation 状态的 SQL 查询和治理报表 |
| `ml_model_assets` | 模型框架、架构、artifact、核心指标的 SQL 查询 |
| `evaluation_report_assets` | 评测报告核心引用和指标索引 |

这些表是 `assets.metadata` 的投影，不是新的事实源。写入时同事务更新，重建时可从 `assets` 回放。

### Phase 4：高级能力

| 能力 | 说明 |
|---|---|
| Schema migration | 支持 `asset.model.ml_model.v1 -> v2` 的在线迁移和兼容读取 |
| 质量治理 | Dataset/AnnotationResult/EvaluationReport 的质量门禁和 lifecycle 自动流转 |
| Lakehouse 同步 | 将 Dataset manifest、Evaluation metrics 同步到 BigLake Iceberg |
| UI 动态表单 | 前端根据 `asset_type_schemas.json_schema` 生成类型专属创建/编辑表单 |
| OpenLineage 集成 | 将 `trained_from/evaluates/tested_on` 转为 OpenLineage Dataset/Job/Run 事件 |

## 7. 技术决策摘要

| 决策 | 结论 |
|---|---|
| 新类型是否直接加列 | 不直接在 `assets` 加大量列；通用字段保持稳定 |
| 类型字段放哪里 | `metadata.type_metadata` 为事实源，ES typed projection 为搜索源 |
| 是否每类型独立 CRUD | 初期不做完整独立 CRUD；保留统一 `/assets`，可增加薄别名端点 |
| 是否每类型独立表 | 二期对稳定高频类型增加投影表，不作为初期依赖 |
| 关系如何表达 | 复杂 AI 链路统一走 `asset_relations`，扩展 relation_type 和类型约束 |
| 查询怎么做 | Query IR 增加类型命名空间字段，ES 承担聚合/排序/全文，PG 兜底精确读取 |
| 版本管理 | 继续使用 `logical_assets`；`asset_type` 在逻辑资产族内不可变 |

## 8. 推荐落地顺序

1. 扩展 `asset_relations.relation_type`，加入 AI 资产链路关系。
2. 引入 `asset_type_schemas` 或等价的代码 registry，先注册 `dataset`、`annotation_result`。
3. 调整写入校验：从硬编码 switch 演进到 registry-driven validation；保留现有类型规则作为内置 schema。
4. ES mapping 增加 `dataset.*`、`annotation_result.*`；搜索 builder 投影 `metadata.type_metadata` 中的高频字段。
5. 支持 `ml_model`、`evaluation_report`，并自动创建训练/评测关系边。
6. 基于真实查询量决定是否新增 `dataset_assets`、`ml_model_assets` 投影表。

