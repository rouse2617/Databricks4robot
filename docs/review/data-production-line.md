# 数据生产全链路编排设计

> 版本：2026-05-28
> 状态：方案设计
> 范围：在现有 Pipeline 可视化设计器、Argo 集成、资产版本管理、资产血缘基础上，扩展为对标 DataClaw 的数据生产链路。

---

## 1. 背景与目标

cyber-databrew 当前已经具备资产平台和单次 Pipeline 执行能力：

- Pipeline 可视化设计器：Frontend 画布定义 DAG，后端 Transpiler 生成 Argo Workflow。
- 资产版本管理：`logical_assets` 协调逻辑资产族，`assets.revision` 与 `is_current` 标识当前版本。
- 资产血缘查询：`asset_relations` 记录父子/派生关系，递归 CTE 支持上下游追溯。
- Pipeline 资产联动：运行前可选择资产，容器产出可通过 `POST /api/v1/pipeline-assets` 注册为资产并记录血缘。

当前缺口是：Pipeline 更像“单段处理 workflow”，还不是从采集、格式转换、清洗、标注、挖掘到训练集生成的完整数据生产链路。本文设计一个“数据生产全链路编排”层，在不推翻现有 handler -> usecase -> repository 架构的前提下，引入多阶段模板、阶段间资产传递、质量门禁和产线级监控。

### 1.1 设计目标

1. 支持多阶段串联：采集 -> 格式转换 -> 清洗 -> 标注 -> 挖掘 -> 训练集生成。
2. 每个阶段可复用现有 Pipeline Template，也可落到一个 Argo Step/Template。
3. 阶段输入输出全部资产化：读取当前资产版本，产出自动注册为新版本或新逻辑资产。
4. 血缘自动闭环：输入资产 -> 阶段运行 -> 输出资产，支持跨阶段递归追溯。
5. 质量检查内建：每个阶段可配置质量规则、阻断策略、告警策略。
6. 兼容现状：MVP 优先复用 `pipeline_templates`、`pipeline_deployments`、`asset_events`、`asset_relations`。

### 1.2 非目标

- 不在本阶段实现完整数据计划系统、跨地域调度器或 FinOps。
- 不替代 BigLake Iceberg / BigQuery 的湖仓建模，只设计产线与资产系统的控制面。
- 不把标注平台、训练平台全部内置到 databrew；它们以组件或外部服务回调方式接入。

---

## 2. 总体架构

```text
┌──────────────────────────────────────────────────────────────────┐
│                         Frontend                                  │
│  产线模板编排 / 阶段配置 / 质量规则 / 运行监控 / 资产血缘展示        │
└──────────────────────────────┬───────────────────────────────────┘
                               │ REST
┌──────────────────────────────▼───────────────────────────────────┐
│                         Gin API                                   │
│  handler: production_line / pipeline / assets / workflows          │
│  usecase: template validation, run orchestration, quality gates     │
│  repo: templates, runs, stage_runs, asset events, relations         │
└──────────────┬───────────────────────┬────────────────────────────┘
               │                       │
               │ submit/watch           │ read/write metadata
┌──────────────▼──────────────┐ ┌──────▼────────────────────────────┐
│        Argo Workflows        │ │          Asset Platform            │
│  multi-step DAG Workflow     │ │ logical_assets / assets revisions  │
│  per-stage pod templates     │ │ asset_relations / asset_events     │
│  retry / condition / outputs │ │ ES index / BigLake storage URI      │
└──────────────┬──────────────┘ └──────┬────────────────────────────┘
               │                       │
               │ compute               │ data files / tables
┌──────────────▼───────────────────────▼────────────────────────────┐
│                Storage / Lakehouse / External Services             │
│  object storage, BigLake Iceberg, Elasticsearch, labeling service,  │
│  mining service, training data builder, model training platform     │
└───────────────────────────────────────────────────────────────────┘
```

核心新增层是 **Production Line Template**：

- Pipeline Template：一个阶段内部的处理 DAG，当前系统已支持。
- Production Line Template：多个阶段的编排模板，负责阶段顺序、条件分支、资产传递、质量门禁和重试策略。

---

## 3. 业务场景

### 3.1 场景 A：路采数据入库链路

目标：将车端或采集系统产生的原始路采数据纳入资产平台，并产出可检索、可追溯、可用于后续标注/挖掘的数据资产。

```text
原始路采包
  -> 采集接入
  -> 格式转换（bag/mcap/log -> 标准分片 + 元数据）
  -> 清洗（坏包剔除、时间戳校验、重复片段去重）
  -> 索引入湖（BigLake Iceberg + ES）
  -> 资产注册（logical asset revision 1）
```

典型阶段：

| 阶段 | 输入 | 输出 | 关键质量检查 |
|------|------|------|--------------|
| ingest | 采集任务 ID、对象存储前缀 | raw_recording_asset | 文件完整性、manifest 校验、大小阈值 |
| convert | raw_recording_asset | standardized_segment_asset | schema 校验、时间戳单调性、传感器通道齐全 |
| clean | standardized_segment_asset | cleaned_segment_asset | 空值率、坏帧率、重复率、GPS/时间漂移 |
| index | cleaned_segment_asset | searchable_asset_revision | ES 文档数、Iceberg 分区写入数 |

产线价值：原始数据进入平台后立即形成“可查、可回放、可追溯”的资产，而不是散落在对象存储路径中。

### 3.2 场景 B：标注数据回流链路

目标：标注平台完成任务后，将标注结果回流为资产新版本，并把标注结果与原始数据建立血缘。

```text
待标注片段资产
  -> 标注任务创建
  -> 外部标注平台执行
  -> 标注结果回收
  -> 标注格式校验
  -> 标注资产注册
  -> 原始片段 -> 标注资产 血缘记录
```

典型阶段：

| 阶段 | 输入 | 输出 | 关键质量检查 |
|------|------|------|--------------|
| task_create | cleaned_segment_asset | labeling_task_ref | 任务去重、样本数、标签体系版本 |
| wait_labeling | labeling_task_ref | annotation_package | 任务状态、标注员/审核状态 |
| validate_annotation | annotation_package | validated_annotation_asset | schema、类别覆盖、坐标合法性、空标注率 |
| register_revision | validated_annotation_asset + source asset | labeled_dataset_revision | 版本递增、血缘完整性 |

产线价值：标注数据不再只是外部系统的附件，而是资产平台内可版本化、可回溯、可参与训练集生成的正式资产。

### 3.3 场景 C：训练集生成链路

目标：基于筛选条件、挖掘结果和标注结果生成可复现实验的训练集资产。

```text
资产筛选条件 / 场景挖掘结果 / 标注资产
  -> 样本召回
  -> 样本去重与均衡
  -> 标签对齐
  -> 训练集切分
  -> manifest 生成
  -> 训练集资产版本注册
```

典型阶段：

| 阶段 | 输入 | 输出 | 关键质量检查 |
|------|------|------|--------------|
| retrieve | 查询条件、标签条件、场景条件 | candidate_asset_set | 召回数量、场景覆盖、权限过滤 |
| mine | candidate_asset_set | mined_asset_set | 难例命中率、相似度阈值、重复率 |
| balance | mined_asset_set | balanced_asset_set | 类别分布、地域/天气/时间段分布 |
| split | balanced_asset_set | train_val_test_manifest | 泄漏检查、比例校验、随机种子记录 |
| register_dataset | manifest + source assets | training_dataset_revision | manifest 可读、文件存在、血缘边数量 |

产线价值：训练集生成过程可复现，训练集资产知道自己来自哪些原始片段、标注结果和挖掘规则。

### 3.4 场景 D：算法挖掘回流链路

目标：模型或规则挖掘发现有价值片段，回流资产平台并触发标注或训练集更新。

```text
资产集合
  -> 算法批处理
  -> 场景/事件挖掘
  -> 结果去重与置信度过滤
  -> 生成候选样本集
  -> 触发标注或训练集产线
```

关键点：挖掘结果本身是资产，后续标注和训练集链路都应引用它，而不是复制一份不可追溯的 ID 列表。

---

## 4. 核心领域模型

建议新增 4 类控制面对象。MVP 可先使用 JSONB 字段保存模板和运行快照，后续再做更细粒度表结构。

```text
production_line_templates
  id, name, version, description, template_yaml/json,
  status, created_by, created_at, updated_at

production_line_runs
  id, template_id, template_version, name, status,
  trigger_type, input_asset_ids, params_json,
  argo_workflow_name, started_at, finished_at, created_by

production_line_stage_runs
  id, line_run_id, stage_id, stage_name, status,
  pipeline_deployment_id, argo_node_id,
  input_asset_ids, output_asset_ids, quality_result_json,
  started_at, finished_at, error_message

quality_check_results
  id, line_run_id, stage_run_id, check_name, status,
  metric_name, actual_value, threshold, severity,
  details_json, created_at
```

### 4.1 与现有表的关系

```text
production_line_runs 1 ── N production_line_stage_runs
production_line_stage_runs N ── 1 pipeline_deployments       (阶段复用现有 pipeline run)
production_line_stage_runs N ── N assets                     (输入/输出资产)
assets N ── N assets via asset_relations                     (输入 -> 输出血缘)
assets 1 ── N asset_events                                   (pipeline_processing / pipeline_output / quality_check)
logical_assets 1 ── N assets                                 (revision / is_current)
```

MVP 不要求每个阶段都必须对应已有 `pipeline_templates`。对于简单质量检查、外部回调等待、资产注册等阶段，可以由平台提供 system component 或内置 Argo template。

---

## 5. Pipeline 模板系统

### 5.1 模板分层

| 层级 | 作用 | 对应现状 |
|------|------|----------|
| Component | 一个容器镜像或平台内置动作 | `pipeline_components` |
| Pipeline Template | 一个阶段内部 DAG | `pipeline_templates.pipeline` |
| Production Line Template | 多阶段业务链路 | 新增 |

Production Line Template 只关心“阶段之间怎么流转”，每个阶段内部继续由现有 Pipeline Template 处理复杂 DAG。

### 5.2 阶段类型

| 类型 | 含义 | 示例 |
|------|------|------|
| `pipeline` | 调用现有 pipeline template | 格式转换、清洗、挖掘 |
| `quality_gate` | 数据质量检查 | 空值率、schema、去重 |
| `external_task` | 外部系统任务 | 标注平台任务创建/等待 |
| `asset_register` | 注册或晋升资产版本 | 训练集 manifest 注册 |
| `branch` | 条件路由 | 低质量数据进入修复链路 |
| `manual_approval` | 人工确认门禁 | 发布训练集前确认 |

### 5.3 阶段间数据传递

阶段之间不直接传本地文件路径，而传 **资产引用** 和 **输出端口名**：

```text
stageA.outputs.cleaned_segments.asset_ids
stageB.inputs.source_assets <- stageA.outputs.cleaned_segments
```

运行时约定：

1. 每个阶段启动前，编排器解析输入资产 ID。
2. 资产平台读取当前 revision、storage URI、metadata、tags、schema。
3. 这些信息作为 workflow 参数、环境变量或挂载文件传入容器。
4. 阶段产出通过统一回调注册资产。
5. 编排器把输出资产 ID 写入 stage run，供下游阶段引用。

### 5.4 条件分支

条件分支基于阶段状态、质量指标、资产 metadata 或运行参数判断：

```yaml
when:
  expression: "stages.clean.quality.bad_frame_rate <= 0.02 && stages.clean.outputs.cleaned.count > 0"
```

MVP 建议支持简单表达式：

- `params.xxx`
- `stages.<stage_id>.status`
- `stages.<stage_id>.quality.<metric>`
- `stages.<stage_id>.outputs.<port>.count`
- 比较运算：`== != > >= < <=`
- 逻辑运算：`&& ||`

复杂表达式后续再引入 CEL 或 OPA，避免第一期过度设计。

### 5.5 重试策略

重试策略分三层：

| 层级 | 配置项 | 行为 |
|------|--------|------|
| 产线级 | `defaults.retry` | 所有阶段默认继承 |
| 阶段级 | `stages[].retry` | 覆盖默认策略 |
| Argo Step 级 | `retryStrategy` | 最终落到 Argo Workflow |

建议保留两类失败：

- `transient`：镜像拉取、网络、外部服务 5xx，可重试。
- `business`：质量门禁失败、schema 不兼容、权限不足，不自动重试。

### 5.6 模板 YAML Schema 示例

```yaml
apiVersion: databrew.cyberorigin.io/v1alpha1
kind: ProductionLineTemplate
metadata:
  name: road-collection-ingestion
  version: "1.0.0"
  labels:
    domain: autonomous-driving
    dataClass: road-collection
spec:
  description: "路采数据从原始包到可检索清洗片段的生产链路"
  owner: data-platform
  parallelism: 4
  timeout: 12h

  inputs:
    assets:
      - name: raw_packages
        assetType: raw_recording
        required: true
        allowMultiple: true
        revisionPolicy: current
    params:
      - name: region
        type: string
        required: true
      - name: force_rebuild
        type: boolean
        default: false

  defaults:
    retry:
      limit: 2
      backoff:
        duration: 30s
        factor: 2
        maxDuration: 5m
      retryOn:
        - transient
    quality:
      onFailure: block
    assetOutput:
      register: true
      lineageRelationType: pipeline_output

  stages:
    - id: convert
      name: "格式转换"
      type: pipeline
      templateRef:
        name: mcap-convert
        version: ">=1.2.0"
      inputs:
        source_assets: "{{ inputs.assets.raw_packages }}"
        region: "{{ inputs.params.region }}"
      outputs:
        standardized_segments:
          assetType: standardized_segment
          logicalAssetStrategy: create
          metadata:
            stage: convert
      retry:
        limit: 3
      timeout: 3h

    - id: clean
      name: "清洗"
      type: pipeline
      dependsOn: [convert]
      templateRef:
        name: segment-cleaner
        version: "2.0.0"
      inputs:
        source_assets: "{{ stages.convert.outputs.standardized_segments }}"
      outputs:
        cleaned_segments:
          assetType: cleaned_segment
          logicalAssetStrategy: revise_source
          revisionOf: "{{ stages.convert.outputs.standardized_segments }}"

    - id: clean_quality
      name: "清洗质量门禁"
      type: quality_gate
      dependsOn: [clean]
      inputs:
        source_assets: "{{ stages.clean.outputs.cleaned_segments }}"
      checks:
        - name: schema_valid
          kind: schema
          severity: critical
          config:
            schemaRef: segment_schema_v3
        - name: bad_frame_rate
          kind: metric_threshold
          severity: critical
          config:
            metric: bad_frame_rate
            op: <=
            value: 0.02
        - name: duplicate_rate
          kind: metric_threshold
          severity: warning
          config:
            metric: duplicate_rate
            op: <=
            value: 0.05
      onFailure:
        critical: block
        warning: continue

    - id: index
      name: "索引入湖"
      type: pipeline
      dependsOn: [clean_quality]
      when:
        expression: "stages.clean_quality.status == 'succeeded'"
      templateRef:
        name: lakehouse-indexer
        version: "1.1.0"
      inputs:
        source_assets: "{{ stages.clean.outputs.cleaned_segments }}"
      outputs:
        searchable_segments:
          assetType: searchable_segment
          logicalAssetStrategy: revise_source
          revisionOf: "{{ stages.clean.outputs.cleaned_segments }}"

    - id: notify_low_quality
      name: "低质量告警"
      type: external_task
      dependsOn: [clean_quality]
      when:
        expression: "stages.clean_quality.status == 'failed'"
      action:
        provider: lark
        operation: send_message
        params:
          channel: data-quality-alerts
          template: data_quality_failed

  finalOutputs:
    - name: searchable_segments
      from: "{{ stages.index.outputs.searchable_segments }}"
      publish: true
```

### 5.7 模板校验规则

保存模板时必须校验：

1. `stages[].id` 唯一，且只包含 `[a-z0-9_-]`。
2. `dependsOn` 引用存在，并且整体无环。
3. `templateRef` 指向存在的 Pipeline Template 或合法版本范围。
4. 输入引用合法：`inputs`、`params`、`stages.<id>.outputs.<port>` 存在。
5. 输出资产类型在资产模型允许范围内。
6. `revisionOf` 只能引用同一阶段输入或上游输出，避免跨无关逻辑资产误晋升。
7. 质量检查的 `onFailure` 只能是 `block`、`continue`、`branch`。

---

## 6. 资产联动设计

### 6.1 运行时资产读取

启动产线时，用户传入逻辑资产或具体资产 revision：

- 传 `asset_id`：精确使用该资产行。
- 传 `logical_asset_id` + `revisionPolicy=current`：运行开始时解析为当前 `asset_id`，并在 run snapshot 固化。
- 传筛选条件：先走资产搜索，固化为资产 ID 列表，避免运行中集合漂移。

运行快照必须记录：

```json
{
  "input_asset_ids": ["asset_001", "asset_002"],
  "resolved_at": "2026-05-28T10:00:00Z",
  "revision_policy": "current",
  "asset_snapshots": [
    {
      "asset_id": "asset_001",
      "logical_asset_id": "logical_001",
      "revision": 3,
      "is_current_at_start": true,
      "storage_uri": "gs://bucket/path",
      "asset_type": "cleaned_segment"
    }
  ]
}
```

### 6.2 阶段输入注入

沿用现有 Pipeline 部署的资产 env 注入能力，并扩展为阶段级 manifest 文件：

```text
ASSET_IDS=asset_001,asset_002
ASSET_MANIFEST_PATH=/databrew/input/assets.json
DATABREW_RUN_ID=plrun_123
DATABREW_STAGE_ID=clean
DATABREW_OUTPUT_CONTRACT_PATH=/databrew/output/outputs.json
```

`assets.json` 示例：

```json
{
  "assets": [
    {
      "id": "asset_001",
      "logical_asset_id": "logical_001",
      "revision": 3,
      "storage_uri": "gs://databrew/raw/001",
      "asset_type": "raw_recording",
      "metadata": {
        "region": "shanghai",
        "vehicle_id": "veh_001"
      }
    }
  ]
}
```

### 6.3 阶段输出注册

容器或平台内置步骤产出 `outputs.json`，由 sidecar 或阶段收尾步骤调用统一 API 注册：

```json
{
  "outputs": [
    {
      "port": "cleaned_segments",
      "asset_type": "cleaned_segment",
      "name": "cleaned-shanghai-20260528",
      "storage_uri": "gs://databrew/cleaned/2026/05/28/run-123/",
      "logical_asset_strategy": "revise_source",
      "revision_of": "asset_001",
      "metadata": {
        "record_count": 123456,
        "bad_frame_rate": 0.008,
        "schema_version": "segment_schema_v3"
      },
      "tags": ["stage:clean", "quality:passed"]
    }
  ]
}
```

注册规则：

| 策略 | 行为 | 适用场景 |
|------|------|----------|
| `create` | 创建新的 `logical_assets` + revision 1 | 新训练集、新挖掘结果集 |
| `revise_source` | 基于源资产逻辑资产族创建新 revision，更新 `is_current` | 格式转换/清洗后替代上一版本 |
| `derive` | 创建新逻辑资产，但与输入建立派生血缘 | 标注结果、训练集 manifest |
| `attach` | 不创建新资产，只向已有资产追加事件/质量报告 | 质量检查、索引状态 |

### 6.4 血缘写入

每个输出资产注册成功后，写入：

1. `asset_events`：
   - `event_type = "pipeline_output"` 或 `"production_line_output"`
   - payload 包含 `line_run_id`、`stage_id`、`pipeline_deployment_id`、`workflow_name`、`template_version`、`image_digest`。
2. `asset_relations`：
   - `parent_asset_id = input_asset_id`
   - `child_asset_id = output_asset_id`
   - `relation_type = "pipeline_output"` 或更细粒度：`converted_from`、`cleaned_from`、`annotated_from`、`dataset_from`。
   - `method = "production_line"`
   - `run_id = line_run_id` 或写入 metadata。
3. 搜索索引：
   - 输出资产入 ES，带上 `logical_asset_id`、`revision`、`is_current`、`lineage_summary`。

血缘示意：

```text
raw_recording@rev1
       │ converted_from
       ▼
standardized_segment@rev1
       │ cleaned_from
       ▼
cleaned_segment@rev2
       ├──────────── annotated_from ────────────► annotation_asset@rev1
       └──────────── dataset_from ──────────────► training_dataset@rev1
```

---

## 7. API 契约设计

以下为新增产线控制面 API。命名保持 `/api/v1`，响应字段使用现有项目前端习惯的 camelCase；底层模型可继续使用 Go struct tag 映射。

### 7.1 保存产线模板

```http
POST /api/v1/production-lines/templates
Content-Type: application/json
X-Databrew-Token: <token>
```

请求：

```json
{
  "name": "road-collection-ingestion",
  "description": "路采数据入库链路",
  "template": {
    "apiVersion": "databrew.cyberorigin.io/v1alpha1",
    "kind": "ProductionLineTemplate",
    "metadata": {
      "name": "road-collection-ingestion",
      "version": "1.0.0"
    },
    "spec": {}
  }
}
```

响应：

```json
{
  "id": "plt_01j...",
  "name": "road-collection-ingestion",
  "version": 1,
  "status": "draft",
  "stageCount": 5,
  "createdAt": "2026-05-28T10:00:00Z"
}
```

校验失败：

```json
{
  "error": "invalid_template",
  "message": "stage clean depends on unknown stage convert_v2",
  "details": [
    {
      "path": "spec.stages[1].dependsOn[0]",
      "reason": "unknown_stage"
    }
  ]
}
```

### 7.2 产线模板列表与详情

```http
GET /api/v1/production-lines/templates?status=published&q=road&page=1&pageSize=20
GET /api/v1/production-lines/templates/{id}
GET /api/v1/production-lines/templates/{id}/versions
POST /api/v1/production-lines/templates/{id}/publish
```

发布策略：

- `draft` 可编辑。
- `published` 不可原地修改，再保存生成新版本。
- 运行只能使用 `published`，MVP 可允许 admin 运行 draft。

### 7.3 启动产线运行

```http
POST /api/v1/production-lines/runs
Content-Type: application/json
Idempotency-Key: <uuid>
```

请求：

```json
{
  "templateId": "plt_01j...",
  "name": "road-ingestion-20260528",
  "inputAssets": {
    "raw_packages": [
      { "assetId": "asset_001" },
      { "logicalAssetId": "logical_002", "revisionPolicy": "current" }
    ]
  },
  "params": {
    "region": "shanghai",
    "force_rebuild": false
  },
  "dryRun": false
}
```

响应：

```json
{
  "id": "plrun_01j...",
  "templateId": "plt_01j...",
  "templateVersion": 3,
  "name": "road-ingestion-20260528",
  "status": "pending",
  "argoWorkflowName": "pl-road-ingestion-01j...",
  "inputAssetIds": ["asset_001", "asset_002"],
  "createdAt": "2026-05-28T10:05:00Z"
}
```

`dryRun=true` 时不提交 Argo，只返回渲染计划：

```json
{
  "valid": true,
  "resolvedInputAssetIds": ["asset_001"],
  "stages": [
    { "id": "convert", "willRun": true, "reason": "dependency satisfied" },
    { "id": "index", "willRun": true, "reason": "condition pending runtime metrics" }
  ],
  "warnings": [
    { "code": "non_current_asset", "message": "asset_001 is not current revision" }
  ]
}
```

### 7.4 查询运行与阶段状态

```http
GET /api/v1/production-lines/runs?status=running&page=1&pageSize=20
GET /api/v1/production-lines/runs/{id}
GET /api/v1/production-lines/runs/{id}/stages
GET /api/v1/production-lines/runs/{id}/quality-results
GET /api/v1/production-lines/runs/{id}/outputs
```

运行详情响应：

```json
{
  "id": "plrun_01j...",
  "status": "running",
  "progress": {
    "totalStages": 5,
    "succeeded": 2,
    "running": 1,
    "failed": 0,
    "skipped": 0
  },
  "argoWorkflowName": "pl-road-ingestion-01j...",
  "inputAssetIds": ["asset_001"],
  "outputAssetIds": ["asset_100", "asset_101"],
  "stages": [
    {
      "id": "convert",
      "status": "succeeded",
      "pipelineDeploymentId": "dep_001",
      "inputAssetIds": ["asset_001"],
      "outputAssetIds": ["asset_100"],
      "startedAt": "2026-05-28T10:06:00Z",
      "finishedAt": "2026-05-28T10:20:00Z"
    }
  ]
}
```

### 7.5 操作运行

```http
POST /api/v1/production-lines/runs/{id}/retry
POST /api/v1/production-lines/runs/{id}/cancel
POST /api/v1/production-lines/runs/{id}/resume
POST /api/v1/production-lines/runs/{id}/stages/{stageId}/retry
POST /api/v1/production-lines/runs/{id}/stages/{stageId}/approve
```

重试语义：

- `run retry`：从失败阶段继续，默认复用已成功阶段输出。
- `stage retry`：只重跑指定失败阶段及其下游。
- `force=true`：允许重跑已成功阶段，并产生新输出资产版本。

### 7.6 阶段输出注册 API

现有 `POST /api/v1/pipeline-assets` 可继续保留。产线建议新增更明确的阶段输出 API，内部复用现有注册逻辑：

```http
POST /api/v1/production-lines/runs/{id}/stages/{stageId}/outputs
```

请求：

```json
{
  "deploymentId": "dep_001",
  "nodeId": "clean",
  "inputAssetIds": ["asset_001"],
  "outputs": [
    {
      "port": "cleaned_segments",
      "assetType": "cleaned_segment",
      "name": "cleaned asset",
      "storageUri": "gs://databrew/cleaned/run-123/",
      "logicalAssetStrategy": "revise_source",
      "revisionOf": "asset_001",
      "metadata": {
        "recordCount": 123456,
        "badFrameRate": 0.008
      }
    }
  ]
}
```

响应：

```json
{
  "registered": [
    {
      "port": "cleaned_segments",
      "assetId": "asset_100",
      "logicalAssetId": "logical_001",
      "revision": 4,
      "isCurrent": true
    }
  ]
}
```

---

## 8. Argo 集成深化

### 8.1 当前模式

当前模式是：Pipeline JSON -> Transpiler -> 一个 Argo Workflow，其中 `entrypoint: dag`，每个画布节点对应 Argo DAG task 和 container template。

新产线不应把所有逻辑压进一个巨大容器，也不应在后端同步串行调用多个 workflow。推荐模式：

```text
Production Line Template
  -> render Argo Workflow
     -> stage convert: templateRef 或 inline container/DAG
     -> stage clean: templateRef 或 inline container/DAG
     -> stage quality: quality gate container
     -> stage index: templateRef 或 inline container/DAG
```

### 8.2 Multi-step DAG 设计

每个产线阶段对应一个 Argo DAG task。阶段内部有两种落地方式：

1. **嵌入式**：将 Pipeline Template 转成 Argo template，并作为子 DAG template 引用。
2. **提交式**：产线 workflow 的 stage task 调用 databrew API 创建现有 pipeline deployment，等待其 Argo Workflow 完成。

推荐 MVP 使用嵌入式，原因：

- 一个 Argo Workflow 展示完整产线 DAG，排障路径更短。
- 重试、条件、依赖由 Argo 原生处理。
- 不需要后端常驻 worker 串联多个 workflow。

提交式可作为后续兼容路径，用于复用已经部署成熟的 Pipeline Template 或跨集群执行。

### 8.3 WorkflowTemplate 示例

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: production-line-road-ingestion
  labels:
    databrew.io/template-type: production-line
spec:
  entrypoint: production-line-dag
  serviceAccountName: argo-workflow
  parallelism: 4
  ttlStrategy:
    secondsAfterCompletion: 86400
  arguments:
    parameters:
      - name: line_run_id
      - name: input_asset_ids
      - name: params_json

  templates:
    - name: production-line-dag
      dag:
        tasks:
          - name: resolve-inputs
            template: resolve-assets
            arguments:
              parameters:
                - name: line_run_id
                  value: "{{workflow.parameters.line_run_id}}"
                - name: input_asset_ids
                  value: "{{workflow.parameters.input_asset_ids}}"

          - name: convert
            dependencies: [resolve-inputs]
            template: stage-convert
            arguments:
              parameters:
                - name: input_manifest
                  value: "{{tasks.resolve-inputs.outputs.parameters.asset_manifest}}"

          - name: clean
            dependencies: [convert]
            template: stage-clean
            arguments:
              parameters:
                - name: input_manifest
                  value: "{{tasks.convert.outputs.parameters.output_manifest}}"

          - name: clean-quality
            dependencies: [clean]
            template: quality-gate
            arguments:
              parameters:
                - name: input_manifest
                  value: "{{tasks.clean.outputs.parameters.output_manifest}}"
                - name: checks_json
                  value: |
                    [
                      {"name":"bad_frame_rate","op":"<=","value":0.02},
                      {"name":"duplicate_rate","op":"<=","value":0.05}
                    ]

          - name: index
            dependencies: [clean-quality]
            when: "{{tasks.clean-quality.outputs.parameters.status}} == succeeded"
            template: stage-index
            arguments:
              parameters:
                - name: input_manifest
                  value: "{{tasks.clean.outputs.parameters.output_manifest}}"

          - name: notify-low-quality
            dependencies: [clean-quality]
            when: "{{tasks.clean-quality.outputs.parameters.status}} == failed"
            template: notify

    - name: resolve-assets
      inputs:
        parameters:
          - name: line_run_id
          - name: input_asset_ids
      outputs:
        parameters:
          - name: asset_manifest
            valueFrom:
              path: /tmp/assets.json
      container:
        image: databrew/asset-resolver:1.0.0
        env:
          - name: DATABREW_API_BASE
            value: "https://cyber-databrew-backend-dev.example.com/api/v1"
        command: ["resolver"]
        args:
          - "--asset-ids={{inputs.parameters.input_asset_ids}}"
          - "--output=/tmp/assets.json"

    - name: stage-convert
      inputs:
        parameters:
          - name: input_manifest
      outputs:
        parameters:
          - name: output_manifest
            valueFrom:
              path: /tmp/output-assets.json
      container:
        image: registry.example.com/databrew/mcap-convert:1.2.3
        command: ["convert"]
        args:
          - "--input-manifest={{inputs.parameters.input_manifest}}"
          - "--output-contract=/tmp/outputs.json"

    - name: stage-clean
      inputs:
        parameters:
          - name: input_manifest
      outputs:
        parameters:
          - name: output_manifest
            valueFrom:
              path: /tmp/output-assets.json
      retryStrategy:
        limit: 2
        backoff:
          duration: "30s"
          factor: 2
          maxDuration: "5m"
      container:
        image: registry.example.com/databrew/segment-cleaner:2.0.0
        command: ["clean"]
        args:
          - "--input-manifest={{inputs.parameters.input_manifest}}"
          - "--output-contract=/tmp/outputs.json"

    - name: quality-gate
      inputs:
        parameters:
          - name: input_manifest
          - name: checks_json
      outputs:
        parameters:
          - name: status
            valueFrom:
              path: /tmp/status.txt
          - name: quality_result
            valueFrom:
              path: /tmp/quality-result.json
      container:
        image: databrew/quality-gate:1.0.0
        command: ["quality-gate"]
        args:
          - "--input-manifest={{inputs.parameters.input_manifest}}"
          - "--checks={{inputs.parameters.checks_json}}"
          - "--status-output=/tmp/status.txt"
          - "--result-output=/tmp/quality-result.json"

    - name: stage-index
      inputs:
        parameters:
          - name: input_manifest
      container:
        image: registry.example.com/databrew/lakehouse-indexer:1.1.0
        command: ["index"]
        args:
          - "--input-manifest={{inputs.parameters.input_manifest}}"

    - name: notify
      container:
        image: databrew/notifier:1.0.0
        command: ["notify"]
        args:
          - "--channel=data-quality-alerts"
```

### 8.4 阶段状态同步

后端 `production_line` usecase 需要定期或按查询同步 Argo 节点状态：

```text
Argo Workflow.status.nodes
  -> production_line_stage_runs.status
  -> production_line_runs.progress
  -> Frontend 运行详情
```

状态映射：

| Argo phase | stage_run status | 说明 |
|------------|------------------|------|
| `Pending` | `pending` | 等待调度或依赖 |
| `Running` | `running` | 正在执行 |
| `Succeeded` | `succeeded` | 执行成功 |
| `Failed` | `failed` | 执行失败 |
| `Error` | `failed` | 系统错误 |
| `Skipped` | `skipped` | 条件未满足 |
| `Omitted` | `skipped` | DAG when 跳过 |

---

## 9. 数据质量与监控

### 9.1 质量检查点

| 阶段 | 检查项 | 指标示例 | 默认策略 |
|------|--------|----------|----------|
| 采集接入 | 文件完整性 | manifest 缺失数、对象大小、checksum | critical block |
| 格式转换 | schema 与时间轴 | schema_valid、timestamp_monotonic、channel_missing_rate | critical block |
| 清洗 | 数据可用性 | null_rate、bad_frame_rate、duplicate_rate、invalid_bbox_rate | critical block / warning continue |
| 标注 | 标注合法性 | empty_label_rate、category_coverage、geometry_valid_rate | critical block |
| 挖掘 | 挖掘有效性 | confidence_avg、hit_rate、near_duplicate_rate | warning continue |
| 训练集生成 | 数据集分布 | class_distribution, split_leakage_count, manifest_missing_count | critical block |
| 索引入湖 | 可检索性 | iceberg_rows_written、es_docs_indexed、partition_count | critical block |

### 9.2 质量规则模型

```json
{
  "name": "bad_frame_rate",
  "kind": "metric_threshold",
  "severity": "critical",
  "config": {
    "metric": "bad_frame_rate",
    "op": "<=",
    "value": 0.02
  },
  "onFailure": "block"
}
```

规则类型：

| `kind` | 用途 |
|--------|------|
| `schema` | JSON/Parquet/Iceberg schema 校验 |
| `metric_threshold` | 指标阈值 |
| `sql_assertion` | BigQuery SQL 断言 |
| `file_existence` | 文件/manifest 存在性 |
| `dedup` | hash、场景 ID、时间窗口去重 |
| `distribution` | 类别/地域/天气/时间段分布 |
| `custom_container` | 用户自定义质量检查镜像 |

### 9.3 质量结果落库

质量检查结果写入 `quality_check_results`，并同步写资产事件：

```json
{
  "event_type": "quality_check",
  "asset_id": "asset_100",
  "payload": {
    "line_run_id": "plrun_01j...",
    "stage_id": "clean_quality",
    "status": "failed",
    "checks": [
      {
        "name": "bad_frame_rate",
        "severity": "critical",
        "actual": 0.035,
        "threshold": 0.02,
        "status": "failed"
      }
    ]
  }
}
```

### 9.4 异常告警策略

告警分级：

| 级别 | 触发条件 | 行为 |
|------|----------|------|
| P0 | 产线系统性不可用、全部阶段无法启动、资产注册失败大面积发生 | 立即通知平台 on-call，阻断发布 |
| P1 | critical 质量门禁失败、训练集泄漏、血缘写入失败 | 通知产线 owner + 数据平台群，运行标记 failed |
| P2 | warning 质量指标超阈值、单阶段重试后成功、外部服务超时恢复 | 通知产线 owner，运行可继续 |
| P3 | SLA 接近超时、成本异常、队列堆积 | 日报或看板提示 |

告警内容必须包含：

- `line_run_id`
- `template_name` + `template_version`
- `stage_id`
- `input_asset_ids` 或数量
- 失败检查项和实际值
- Argo Workflow URL
- 相关资产详情 URL

### 9.5 监控指标

平台级：

- `production_line_runs_total{template,status}`
- `production_line_stage_duration_seconds{template,stage}`
- `production_line_quality_failures_total{template,stage,check,severity}`
- `production_line_asset_outputs_total{asset_type,stage}`
- `production_line_retry_total{template,stage}`
- `production_line_lineage_write_failures_total`

业务级：

- 每日产出 raw / cleaned / labeled / dataset 资产数。
- 采集到可检索的端到端延迟。
- 标注回流到训练集可用的端到端延迟。
- 训练集类别分布和样本增长趋势。

---

## 10. 技术决策

### 10.1 编排层选型：复用 Argo，而不是自研调度器

决策：产线编排最终渲染为 Argo Workflow / WorkflowTemplate。

理由：

- 当前系统已经有 Argo 客户端、Workflow 列表、详情、日志、停止、重试能力。
- Argo 原生支持 DAG、依赖、条件、重试、超时、参数传递。
- 后端只需要承担模板校验、资产解析、运行记录和状态同步，不需要实现调度内核。

### 10.2 阶段间传递资产引用，而不是传文件路径

决策：阶段输出必须注册为资产，下游通过资产 ID 或 manifest 读取。

理由：

- 资产版本、权限、血缘、搜索都依赖资产 ID。
- 文件路径无法表达 revision、is_current、metadata、标签和质量状态。
- 训练集复现需要资产快照，不应依赖“当前路径内容”。

### 10.3 先 JSONB 快照，后规范化表

决策：MVP 的模板定义、运行输入快照、质量详情允许存 JSONB；关键查询字段单独列化。

理由：

- 产线 schema 会快速演进，过早拆表会拖慢迭代。
- 关键列表查询只需要 `id/name/status/template_id/started_at/finished_at` 等稳定字段。
- 后续当质量规则和阶段检索稳定后，再将高频字段规范化。

### 10.4 质量门禁作为一等阶段

决策：质量检查不是日志附件，而是产线阶段和资产事件。

理由：

- 质量失败会改变产线控制流，必须可阻断、可分支。
- 质量结果需要和输出资产绑定，便于搜索和治理。
- 后续数据集准入、模型训练准入可直接复用质量结果。

### 10.5 产线运行快照固定输入 revision

决策：即使用户选择 `current`，运行开始后也解析为具体 `asset_id` 和 revision，并记录快照。

理由：

- 防止长时间运行期间 `is_current` 被新版本改变导致不可复现。
- 失败重试应默认复用同一批输入，除非用户显式 `refreshInputs=true`。

---

## 11. Handler / Usecase / Repository 切分

### 11.1 Handler

建议新增：

```text
backend/internal/handlers/production_line/
  handler.go
```

职责：

- 解析请求、鉴权上下文、参数校验。
- 调用 usecase。
- 返回统一错误结构。

### 11.2 Usecase

建议新增：

```text
backend/internal/usecase/production_line/
  usecase.go
  validator.go
  renderer.go
  quality.go
```

职责：

- 模板语义校验。
- 资产输入解析和快照。
- 渲染 Argo WorkflowTemplate / Workflow。
- 创建 run / stage_run。
- 同步 Argo 状态。
- 调用 asset usecase 完成输出注册和血缘写入。

### 11.3 Repository

建议新增：

```text
backend/internal/repository/production_line_repository.go
backend/internal/postgres/production_line_repo.go
```

接口示例：

```go
type ProductionLineRepository interface {
    SaveTemplate(ctx context.Context, in SaveTemplateInput) (*ProductionLineTemplate, error)
    GetTemplate(ctx context.Context, id string) (*ProductionLineTemplate, error)
    ListTemplates(ctx context.Context, filter TemplateFilter) ([]ProductionLineTemplate, int64, error)
    CreateRun(ctx context.Context, in CreateRunInput) (*ProductionLineRun, error)
    UpdateRunStatus(ctx context.Context, id string, status string, patch RunPatch) error
    CreateStageRuns(ctx context.Context, runID string, stages []StageRun) error
    UpdateStageRun(ctx context.Context, runID, stageID string, patch StageRunPatch) error
    SaveQualityResults(ctx context.Context, results []QualityCheckResult) error
}
```

---

## 12. 分阶段实施建议

### 12.1 MVP：可运行的三阶段产线（约 3-4 周）

目标：从“单段 workflow”升级为“多阶段产线”，先跑通采集/转换/清洗/注册闭环。

范围：

| 模块 | 工作项 | 预估 |
|------|--------|------|
| DB | 新增 `production_line_templates`、`production_line_runs`、`production_line_stage_runs`、`quality_check_results` | 2 人日 |
| Backend | 模板 CRUD、发布、校验、运行启动、状态查询 | 5 人日 |
| Backend | Argo multi-stage renderer，复用现有 transpiler 能力 | 5 人日 |
| Backend | 资产输入解析、输出注册、血缘写入封装 | 4 人日 |
| Backend | 内置 quality gate 基础实现：schema、metric threshold、file existence | 4 人日 |
| Frontend | 产线模板列表/详情/运行表单/运行详情 | 6 人日 |
| Frontend | 阶段 DAG 状态图 + 质量结果展示 | 4 人日 |
| QA/DevOps | dev smoke、示例模板、失败场景验证 | 3 人日 |

MVP 验收：

1. 可保存并发布一个产线模板。
2. 可选择输入资产启动产线。
3. Argo 中出现 multi-step DAG。
4. 至少 3 个阶段串联执行，阶段间通过资产 manifest 传递。
5. 阶段产出自动注册为资产版本。
6. `asset_relations` 可追溯输入 -> 输出。
7. 质量门禁失败时产线阻断并告警。

### 12.2 第二阶段：标注与训练集闭环（约 4-6 周）

范围：

- `external_task` 阶段：支持标注平台任务创建、状态轮询、回调。
- `manual_approval` 阶段：训练集发布前人工确认。
- 训练集 manifest builder：支持 train/val/test 切分、泄漏检查。
- 产线运行重试增强：从失败阶段继续，复用上游输出。
- 产线级 lineage API：一次查询返回完整生产链路。
- 质量规则扩展：SQL assertion、distribution、dedup。

验收：

1. 标注结果可回流为资产并关联原始资产。
2. 训练集资产可复现：manifest、输入资产、切分参数、质量结果完整。
3. 产线运行失败后可从指定阶段重试。

### 12.3 第三阶段：生产化与规模化（约 6-8 周）

范围：

- Backfill 与产线结合：按资产筛选条件批量启动产线。
- 产线版本 diff、回滚、灰度发布。
- SLA、成本、资源使用看板。
- 跨集群/跨地域执行策略。
- 与模型训练平台联动：训练集发布后触发训练任务。
- 数据计划层：周期调度、事件触发、增量物化。

验收：

1. 支持按天/按数据源自动触发产线。
2. 支持历史资产批量重跑并生成新版本。
3. 支持产线版本灰度：小流量资产先跑，新版本质量通过后推广。

---

## 13. 风险与约束

| 风险 | 影响 | 缓解 |
|------|------|------|
| 模板表达式过复杂 | 校验和排障困难 | MVP 限制表达式能力，后续再引入 CEL/OPA |
| 资产注册与 workflow 状态不一致 | 产出丢失或重复注册 | 输出注册 API 使用幂等键：`line_run_id + stage_id + port + output_hash` |
| 重试导致重复资产版本 | 版本污染 | 默认失败阶段重试覆盖未发布输出，成功后才设为 current；或使用 idempotency key |
| 外部标注任务耗时长 | Argo workflow 长时间占用 | 第二阶段支持 suspend/resume 或事件回调恢复 |
| 质量指标口径不统一 | 业务不信任质量结果 | 质量规则模板版本化，指标定义入文档和事件 payload |
| 大批量资产传参过大 | Argo parameter 超限 | 使用 manifest 文件存储到对象存储，只在参数中传 manifest URI |

---

## 14. 推荐落地顺序

1. 定义 `ProductionLineTemplate` YAML schema 和校验器。
2. 新增模板/运行/阶段运行表，先用 JSONB 快照。
3. 实现产线启动：解析输入资产 -> 创建 run -> 渲染 Argo DAG -> 提交 workflow。
4. 实现阶段输出注册 API，内部复用现有 pipeline asset 注册和 `asset_relations` 写入。
5. 实现基础 quality gate，先覆盖 schema、metric threshold、file existence。
6. 前端做运行入口和运行详情，优先展示阶段状态、日志入口、输入输出资产。
7. 用“路采数据入库链路”作为第一条样板产线。
8. 再接“标注回流”和“训练集生成”，补齐 external task 与 manual approval。

MVP 只要能稳定跑通一条真实链路，就已经从“Pipeline 工具”升级为“数据生产链路编排”。后续 DataClaw 式能力应围绕产线模板版本、质量准入、资产血缘和批量回放持续增强。
