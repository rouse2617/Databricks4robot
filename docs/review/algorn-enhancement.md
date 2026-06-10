# AlgoRun 与资产平台深化集成设计

> 版本：2026-05-28
> 状态：设计评审稿
> 范围：AlgoRun 训练产出自动资产化、输入血缘自动记录、Pipeline 训练节点、训练执行追踪
> 约束：Go 1.25 + Gin + PostgreSQL + Elasticsearch + BigQuery/BigLake Iceberg；后端分层保持 `handler -> usecase -> repository`

---

## 1. 背景与目标

cyber-databrew 已具备三块基础能力：

| 能力 | 当前状态 | 关键实现 |
|------|----------|----------|
| AlgoRun | 已上线为独立算法执行记录 | `algo_runs`、`POST /api/v1/algo-runs`、`/start`、`/finish`、`/affected-assets` |
| Pipeline + Argo | 已完成可视化设计器、模板、部署、Workflow 监控 | `pipeline_templates`、`pipeline_deployments`、transpiler、Argo Workflow |
| 资产版本与血缘 | 已上线 logical asset + revision + recursive CTE lineage | `logical_assets`、`assets.revision/is_current`、`asset_relations` |

当前缺口是 **训练执行没有进入资产闭环**：

1. AlgoRun 的输入依赖主要靠 `input_filter` / `input_asset_ids` 手动传入，平台不区分训练集、评估集、配置资产等输入角色。
2. 训练产出的模型文件、数据集文件只存在于对象存储或外部训练系统，默认不会注册成 `assets` 新版本。
3. 即使训练产物被手动注册，也不会自动写入“训练数据 -> 模型产出”的 `asset_relations` 血缘边。
4. Pipeline 只能编排通用容器节点，还没有一等“训练节点”语义，无法把 Argo 执行、AlgoRun、资产输入、产出注册串起来。

本设计目标：

```text
训练启动时：输入资产角色化记录
训练完成时：产出自动注册为资产版本
注册成功后：输入资产 -> 产出资产 自动建血缘边
Pipeline 中：训练节点可选择 AlgoRun 配置、输入资产并等待训练完成
运行后：可查看历史、重跑、对比 run 与模型版本
```

---

## 2. 现状分析

### 2.1 当前 AlgoRun 架构

现有 AlgoRun 是一个 **执行事件实体**，不直接拥有资产版本语义。

```text
HTTP API
  |
  v
backend/internal/handlers/algorun
  |
  v
backend/internal/usecase/algorun
  |
  v
backend/internal/repository.AlgoRunRepository
  |
  v
PostgreSQL algo_runs
```

核心字段摘要：

| 字段 | 用途 |
|------|------|
| `run_id` | 16 位运行 ID，AlgoRun 主键 |
| `algo_name` / `algo_version` / `algo_kind` | 算法身份 |
| `status` | `pending` / `running` / `ok` / `failed` / `cancelled` |
| `input_filter` | 输入资产筛选条件快照 |
| `input_asset_ids` | 小批量输入资产 ID 快照 |
| `params` | 训练或算法参数 |
| `pipeline_name` / `pipeline_version` | 可选编排来源 |
| `outputs` | 运行结束后的非结构化输出摘要 |
| `external_runtime` / `external_url` | 外部执行平台锚点 |

现有执行流程：

```text
Worker / Pipeline / 外部训练系统
  |
  | POST /api/v1/algo-runs
  v
algo_runs: pending
  |
  | POST /api/v1/algo-runs/{run_id}/start
  v
algo_runs: running
  |
  | 训练/处理文件，产物写对象存储
  v
  | POST /api/v1/algo-runs/{run_id}/finish
  v
algo_runs: ok / failed
```

### 2.2 当前资产与版本模型

资产当前模型以 `assets` 为物理 revision 行，以 `logical_assets` 管理一组版本。

| 表 | 语义 |
|----|------|
| `logical_assets` | 资产家族，字段包括 `logical_asset_id`、`asset_type`、`current_revision`、`total_revisions` |
| `assets` | 每个 revision 一行，字段包括 `asset_id`、`logical_asset_id`、`revision`、`is_current`、`storage_uri`、`metadata` |
| `asset_relations` | 复杂血缘边，当前主键 `(parent_asset_id, child_asset_id, relation_type)` |

资产版本策略已有明确约束：

| 场景 | 应写入 |
|------|--------|
| 同一个业务对象的新版本 | 新 `assets` revision，复用同一 `logical_asset_id` |
| 新模型、新数据集、新业务身份 | 新 `logical_assets` + revision 1 |
| revision 晋升 | 旧 revision `is_current=false`，新 revision `is_current=true` |

### 2.3 当前脱节点

| 脱节点 | 当前表现 | 后果 |
|--------|----------|------|
| 输入靠手动传 | `input_asset_ids` 只是 run 字段，不区分训练/评估/配置角色 | 无法回答“哪个模型由哪些训练集训练、由哪个验证集评估” |
| 产出不注册 | `finish.outputs` 可记录 URI，但不会创建 `assets` | 模型无法被搜索、授权、交付、版本比较 |
| 无训练血缘 | `asset_relations` 不由 AlgoRun 自动写入 | 资产详情 lineage 看不到训练链路 |
| Pipeline 无训练语义 | Pipeline 节点只是容器组件 | 无法声明“这个节点会创建 AlgoRun 并产出模型资产” |
| 执行追踪分散 | `algo_runs` 有 run 级摘要，但缺训练产物、对比、重跑入口 | MLOps 运维能力不足 |

---

## 3. 总体设计

### 3.1 核心决策

| 决策 | 结论 | 理由 |
|------|------|------|
| AlgoRun 是否升级为资产 | 否 | AlgoRun 是执行事件，不是可交付资产；继续作为 run 锚点 |
| 模型/数据集产出是否资产化 | 是 | 产物需要版本、权限、搜索、血缘、交付 |
| 训练产出写新 revision 还是新 logical asset | 由 `logical_asset_id` 或 `output_identity` 决定 | 同一模型名/任务的新 checkpoint 应进入同一家族；新模型身份新建 logical asset |
| 血缘边是否写 `asset_relations` | 是 | 资产 lineage 的权威查询已基于 `asset_relations` 递归 CTE |
| 是否新增 `trained_from` / `evaluated_on` relation_type | 是 | 用 `derived_from` 无法表达 ML 输入角色；需扩展 DB CHECK |
| 输出注册是否复用 pipeline output registration | 逻辑复用，接口独立 | pipeline output 当前偏通用节点产物；训练产物需要模型指标、框架、参数等 ML 元信息 |
| 运行追踪存 PG 还是 ES | PG 为事实源，ES 做搜索投影 | 状态机、重跑、审计需强一致；ES 适合列表筛选与全文检索 |

### 3.2 目标架构

```text
Pipeline TrainingStep / 外部训练 Worker
        |
        | 1. create/start AlgoRun with typed inputs
        v
  AlgoRun Usecase
        |
        | 2. training runs in Argo / external runtime
        v
  Object Storage / BigLake / Model Registry path
        |
        | 3. finish callback with output artifacts
        v
  AlgoRun Integration Usecase
        |
        +--> register model/dataset as asset revision
        |
        +--> write asset_relations:
        |      training_dataset --trained_from--> model
        |      eval_dataset     --evaluated_on--> model
        |      config_asset     --configured_by--> model
        |
        +--> append asset_events / algo_run events
        |
        v
  Asset search / lineage / model history / run comparison
```

### 3.3 建议新增后端模块

保持 `handler -> usecase -> repository` 分层，新增 ML 训练集成薄层：

```text
backend/internal/handlers/algorun_artifact/
backend/internal/usecase/algorunartifact/
backend/internal/repository/algorun_artifact_repository.go
backend/internal/postgres/algo_run_artifacts.go
```

职责边界：

| 模块 | 职责 |
|------|------|
| `algorun` | run 生命周期：create/start/finish/cancel/list |
| `algorunartifact` | run 完成后的产物注册、输入血缘、重跑、对比 |
| `asset` usecase | 资产创建、版本晋升、logical asset 约束 |
| `pipeline` usecase | Pipeline 模板部署、Argo Workflow 提交、TrainingStep 转译 |

---

## 4. 训练产出自动注册为资产

### 4.1 注册时机

注册时机选择：**训练完成后回调，同步校验，事务内注册资产与血缘**。

推荐流程：

```text
Worker
  |
  | POST /api/v1/algo-runs/{run_id}/finish
  | body.status=ok
  | body.outputs.artifacts=[...]
  v
AlgoRun Usecase
  |
  | update algo_runs terminal state
  v
AlgoRunArtifact Usecase
  |
  | for each output artifact:
  |   1. classify asset type
  |   2. resolve logical asset family
  |   3. create new asset revision
  |   4. record run-output mapping
  |   5. create input lineage edges
  v
PostgreSQL transaction
```

失败处理：

| 场景 | 行为 |
|------|------|
| 训练失败 `status=failed` | 不注册产物，只保留 `algo_runs.outputs` 和错误信息 |
| 训练成功但产物注册失败 | `algo_runs.status` 保持 `ok`，新增 `artifact_registration_status=failed` 或记录 `algo_run_artifacts.status=failed`；API 返回 207 不推荐，建议单独 `POST /register-outputs` 可重试 |
| 重复回调 | 按 `(run_id, artifact_name)` 或 `idempotency_key` 幂等返回既有资产 |

为降低对现有 `/finish` 的破坏，MVP 建议采用 **两步式**：

1. `/finish` 只结束 run。
2. `/algo-runs/{run_id}/outputs:register` 注册产物与血缘。

后续 SDK 可封装为一个 `finish_and_register()`。

### 4.2 资产类型定义

建议扩展 `assets.asset_type` 使用以下 ML 资产类型：

| asset_type | 语义 | 典型 storage_uri | 关键 metadata |
|------------|------|------------------|---------------|
| `ml_model` | 模型 artifact、checkpoint、serving bundle | `gs://.../models/{model}/v{n}/` | framework、model_format、metrics、hyperparameters |
| `dataset` | 训练/评估数据集快照或 manifest | `gs://.../datasets/{name}/snapshots/{id}/manifest.json` | dataset_role、item_count、schema、query_spec |
| `feature_set` | 特征集快照，后续可选 | `gs://.../features/{name}/{version}/` | feature_schema、source_assets |
| `training_report` | 可审计训练报告，后续可选 | `gs://.../runs/{run_id}/report.html` | metric_summary、confusion_matrix_uri |

MVP 只要求 `ml_model` 与 `dataset`。

### 4.3 版本策略

产物注册请求必须携带足够信息决定是“新 revision”还是“新 logical asset”。

| 请求字段 | 行为 |
|----------|------|
| `logical_asset_id` 非空 | 在该 logical asset 下创建新 revision；校验 `asset_type` 一致 |
| `model_key` / `dataset_key` 命中唯一 logical asset | 创建新 revision |
| `create_logical_asset=true` | 新建 logical asset，创建 revision 1 |
| 都未提供且无法匹配 | 400，要求调用方显式确认资产身份 |

推荐模型身份键：

```text
tenant_id + project_id + asset_type + output_identity.name + output_identity.task + output_identity.target
```

示例：

```json
{
  "output_identity": {
    "name": "risk_detector",
    "task": "classification",
    "target": "driving_event",
    "stage": "candidate"
  }
}
```

版本号策略：

| 字段 | 来源 |
|------|------|
| `assets.revision` | 平台自增，权威版本号 |
| `metadata.model_version` | 训练系统或模型语义版本，可与 revision 不同 |
| `logical_assets.current_revision` | 平台当前版本 |
| `is_current` | 新注册成功后默认 true；可用 `publish_mode=draft` 让候选版本不自动切 current，需后续扩展 |

### 4.4 元信息提取

训练产物注册时，平台不解析大型模型文件内容；由 Worker/SDK 上报结构化 manifest，平台做 schema 校验与归档。

`ml_model` 元信息：

| 字段 | 示例 | 说明 |
|------|------|------|
| `framework` | `pytorch` / `tensorflow` / `xgboost` / `sklearn` | 训练框架 |
| `framework_version` | `2.7.0` | 框架版本 |
| `model_format` | `torchscript` / `onnx` / `saved_model` / `pickle` | 服务格式 |
| `artifact_digest` | `sha256:...` | artifact 内容摘要 |
| `metrics` | `{"auc":0.91,"f1":0.84}` | 评估指标 |
| `hyperparameters` | `{"lr":0.001,"batch_size":64}` | 训练参数 |
| `training_params_ref` | `gs://.../params.json` | 大参数文件引用 |
| `code_commit` | `git:abc123` | 代码版本 |
| `image_digest` | `sha256:...` | 镜像版本 |
| `run_id` | `R001abc123def456` | AlgoRun 锚点 |

`dataset` 元信息：

| 字段 | 示例 | 说明 |
|------|------|------|
| `dataset_role` | `training` / `evaluation` / `validation` | 数据集用途 |
| `manifest_uri` | `gs://.../manifest.json` | 样本 manifest |
| `query_spec` | `{...}` | 生成该快照的查询 |
| `item_count` | `123456` | 样本数量 |
| `source_query_hash` | `sha256:...` | 查询快照摘要 |
| `schema_uri` | `gs://.../schema.json` | 数据 schema |

### 4.5 API 设计

#### 4.5.1 注册训练产出

```http
POST /api/v1/algo-runs/{run_id}/outputs:register
X-Databrew-Token: <token>
Idempotency-Key: <uuid>
Content-Type: application/json
```

请求：

```json
{
  "outputs": [
    {
      "name": "risk-detector-model",
      "asset_type": "ml_model",
      "storage_uri": "gs://databrew/models/risk-detector/run-R001/",
      "files": {
        "model": "gs://databrew/models/risk-detector/run-R001/model.onnx",
        "metrics": "gs://databrew/models/risk-detector/run-R001/metrics.json"
      },
      "logical_asset_id": "Ab12Cd34",
      "create_logical_asset": false,
      "output_identity": {
        "name": "risk_detector",
        "task": "classification",
        "target": "driving_event"
      },
      "metadata": {
        "framework": "pytorch",
        "framework_version": "2.7.0",
        "model_format": "onnx",
        "artifact_digest": "sha256:6a7...",
        "metrics": {
          "auc": 0.913,
          "f1": 0.842,
          "loss": 0.218
        },
        "hyperparameters": {
          "lr": 0.001,
          "batch_size": 64,
          "epochs": 20
        }
      }
    }
  ],
  "input_lineage": [
    {
      "input_asset_id": "TrnA0012",
      "relation_type": "trained_from",
      "role": "training_dataset"
    },
    {
      "input_asset_id": "Eval0099",
      "relation_type": "evaluated_on",
      "role": "evaluation_dataset"
    }
  ]
}
```

响应：

```json
{
  "run_id": "R001abc123def456",
  "registered_assets": [
    {
      "artifact_name": "risk-detector-model",
      "asset_id": "MdL9Xa21",
      "logical_asset_id": "Ab12Cd34",
      "revision": 7,
      "asset_type": "ml_model",
      "is_current": true,
      "storage_uri": "gs://databrew/models/risk-detector/run-R001/"
    }
  ],
  "relations_created": 2
}
```

校验规则：

| 条件 | 返回 |
|------|------|
| run 不存在 | 404 |
| run 非 `ok` | 409，不允许为 failed/cancelled run 注册成功产物 |
| `asset_type` 不是 `ml_model` / `dataset` | 400 |
| `logical_asset_id` 存在但类型不一致 | 409 |
| `input_asset_id` 不存在 | 400 |
| 重复 `(run_id, artifact_name)` | 200 返回既有注册结果 |

#### 4.5.2 查询 run 产物

```http
GET /api/v1/algo-runs/{run_id}/outputs
```

响应：

```json
{
  "run_id": "R001abc123def456",
  "items": [
    {
      "artifact_name": "risk-detector-model",
      "asset_id": "MdL9Xa21",
      "logical_asset_id": "Ab12Cd34",
      "revision": 7,
      "asset_type": "ml_model",
      "storage_uri": "gs://databrew/models/risk-detector/run-R001/",
      "registered_at": "2026-05-28T10:00:00Z"
    }
  ]
}
```

### 4.6 后端变更

Handler：

```text
backend/internal/handlers/algorun_artifact/handler.go
  POST /api/v1/algo-runs/:run_id/outputs:register
  GET  /api/v1/algo-runs/:run_id/outputs
```

Usecase：

```go
type RegisterOutputsInput struct {
    RunID        string
    Outputs      []OutputArtifactInput
    InputLineage []InputLineageInput
}

type OutputArtifactInput struct {
    Name               string
    AssetType          string
    StorageURI         string
    Files              map[string]string
    LogicalAssetID     string
    CreateLogicalAsset bool
    OutputIdentity     map[string]string
    Metadata           map[string]any
}

type InputLineageInput struct {
    InputAssetID  string
    RelationType  string
    Role          string
    Metadata      map[string]any
}
```

Repository：

```go
type AlgoRunArtifactRepository interface {
    InsertRegisteredOutput(ctx context.Context, row *AlgoRunOutput) error
    ListRegisteredOutputs(ctx context.Context, runID string) ([]*AlgoRunOutput, error)
    FindByRunAndName(ctx context.Context, runID, name string) (*AlgoRunOutput, error)
}
```

事务边界：

```text
BEGIN
  SELECT algo_runs FOR UPDATE
  resolve logical asset
  insert assets row
  update logical_assets revision/current
  insert algo_run_outputs row
  insert asset_relations rows
  append asset_events
COMMIT
```

### 4.7 建议新增表：`algo_run_outputs`

虽然可以只靠 `asset_relations.run_id` 反查产物，但建议增加 run-output 映射表，便于幂等、列表、对比。

```sql
CREATE TABLE algo_run_outputs (
    run_id              TEXT        NOT NULL REFERENCES algo_runs(run_id) ON DELETE CASCADE,
    artifact_name       TEXT        NOT NULL,
    artifact_role       TEXT        NOT NULL DEFAULT 'primary',
    asset_id            TEXT        NOT NULL REFERENCES assets(asset_id),
    logical_asset_id    TEXT        REFERENCES logical_assets(logical_asset_id),
    revision            BIGINT,
    asset_type          TEXT        NOT NULL,
    storage_uri         TEXT,
    metadata            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    registration_status TEXT        NOT NULL DEFAULT 'registered',
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, artifact_name)
);

CREATE INDEX idx_algo_run_outputs_asset
    ON algo_run_outputs(asset_id);

CREATE INDEX idx_algo_run_outputs_logical
    ON algo_run_outputs(logical_asset_id, revision DESC);
```

---

## 5. 输入血缘自动记录

### 5.1 记录时机

输入血缘由两个阶段完成：

| 阶段 | 写入内容 | 原因 |
|------|----------|------|
| AlgoRun 启动时 | 在 run 上保存 typed inputs | 此时产出 asset 尚不存在，不能写 `asset_relations` |
| 产出注册时 | 写 `asset_relations(parent=input, child=output)` | 产出 asset_id 已确定，可形成资产血缘 |

MVP 不建议在启动时创建“悬空血缘边”，因为 `asset_relations.child_asset_id` 有 FK 约束。

### 5.2 输入声明结构

现有 `input_asset_ids` 是字符串数组，建议新增结构化输入字段。为减少迁移风险，可先放入 `algo_runs.params.training_inputs` 或新增 `algo_run_inputs` 表；长期建议独立表。

推荐表：

```sql
CREATE TABLE algo_run_inputs (
    run_id          TEXT        NOT NULL REFERENCES algo_runs(run_id) ON DELETE CASCADE,
    input_asset_id  TEXT        NOT NULL REFERENCES assets(asset_id),
    input_role      TEXT        NOT NULL,
    relation_type   TEXT        NOT NULL,
    required        BOOLEAN     NOT NULL DEFAULT TRUE,
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, input_asset_id, input_role)
);

CREATE INDEX idx_algo_run_inputs_asset
    ON algo_run_inputs(input_asset_id);
```

输入角色与 relation type：

| input_role | relation_type | 语义 |
|------------|---------------|------|
| `training_dataset` | `trained_from` | 模型由该训练数据训练得到 |
| `evaluation_dataset` | `evaluated_on` | 模型由该评估数据评估 |
| `validation_dataset` | `validated_on` | 模型由该验证集调参/验证 |
| `config` | `configured_by` | 训练配置、特征配置、prompt 配置 |
| `base_model` | `fine_tuned_from` | Fine-tune 基座模型 |
| `feature_set` | `features_from` | 模型使用该特征集训练 |

### 5.3 `asset_relations` 变更

当前 `asset_relations.relation_type` CHECK 仅允许：

```text
split_from, derived_from, contains, sampled_from, merged_from, revision_of
```

MVP 需要扩展为：

```sql
ALTER TABLE asset_relations
  DROP CONSTRAINT IF EXISTS chk_relation_type,
  ADD CONSTRAINT chk_relation_type CHECK (
    relation_type IN (
      'split_from',
      'derived_from',
      'contains',
      'sampled_from',
      'merged_from',
      'revision_of',
      'trained_from',
      'evaluated_on',
      'validated_on',
      'configured_by',
      'fine_tuned_from',
      'features_from'
    )
  );
```

同时建议给 `asset_relations` 增加 `metadata JSONB`，否则训练参数、run input role、metric 摘要只能塞进 `method` 或事件 payload，不利于查询。

```sql
ALTER TABLE asset_relations
  ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_asset_relations_run
  ON asset_relations(run_id);

CREATE INDEX IF NOT EXISTS idx_asset_relations_metadata_gin
  ON asset_relations USING GIN(metadata);
```

写入示例：

```sql
INSERT INTO asset_relations (
    parent_asset_id,
    child_asset_id,
    relation_type,
    method,
    algo_name,
    algo_version,
    run_id,
    metadata
) VALUES (
    'TrnA0012',
    'MdL9Xa21',
    'trained_from',
    'algorun',
    'risk_detector_train',
    '1.4.0',
    'R001abc123def456',
    '{
      "input_role": "training_dataset",
      "epochs": 20,
      "sample_count": 123456,
      "metrics": {"auc": 0.913}
    }'::jsonb
);
```

### 5.4 血缘查询效果

注册前：

```text
training dataset asset -- no edge --> model file in GCS
```

注册后：

```text
TrnA0012(dataset rev 3)
  -- trained_from / run_id=R001abc123def456 -->
MdL9Xa21(ml_model rev 7)

Eval0099(dataset rev 5)
  -- evaluated_on / run_id=R001abc123def456 -->
MdL9Xa21(ml_model rev 7)
```

资产详情页 lineage 递归 CTE 可直接展示：

```text
上游
  TrnA0012  dataset  rev=3  relation=trained_from
  Eval0099  dataset  rev=5  relation=evaluated_on
当前
  MdL9Xa21  ml_model rev=7  run=R001abc123def456
下游
  ServingBundle / Report / Delivery ...
```

---

## 6. Pipeline 训练节点集成

### 6.1 产品语义

在 Pipeline 设计器新增节点类型：`training`。

用户配置：

1. 选择 AlgoRun 配置：算法名、版本、镜像、命令、默认参数、资源需求。
2. 选择输入资产：训练集、评估集、配置资产、基座模型等。
3. 配置输出：模型资产 identity、是否创建新 logical asset、输出路径模板。
4. 部署后由 Argo 执行训练容器，容器通过平台 API 创建并更新 AlgoRun。
5. 节点完成后自动注册模型/数据集资产，并写入血缘。

### 6.2 Pipeline JSON schema 扩展

当前 transpiler 节点核心结构是：

```json
{
  "id": "train-risk-model",
  "component": {
    "name": "train",
    "image": "gcr.io/.../trainer:1.4.0",
    "command": ["python", "train.py"],
    "args": []
  },
  "inputs": [],
  "outputs": []
}
```

建议在节点上增加 `type` 与 `training` 扩展块；旧节点默认 `type=container`。

```json
{
  "id": "train-risk-model",
  "type": "training",
  "component": {
    "name": "risk-detector-train",
    "image": "us-docker.pkg.dev/databrew/trainers/risk-detector:1.4.0",
    "command": ["python", "-m", "trainer.main"],
    "resources": {
      "cpu": "4",
      "memory": "16Gi"
    }
  },
  "training": {
    "algo": {
      "name": "risk_detector_train",
      "version": "1.4.0",
      "kind": "processing"
    },
    "triggered_by": "pipeline",
    "params": {
      "lr": 0.001,
      "batch_size": 64,
      "epochs": 20
    },
    "inputs": [
      {
        "name": "train",
        "asset_id": "TrnA0012",
        "role": "training_dataset",
        "relation_type": "trained_from",
        "mount_env": "TRAIN_MANIFEST_URI"
      },
      {
        "name": "eval",
        "asset_id": "Eval0099",
        "role": "evaluation_dataset",
        "relation_type": "evaluated_on",
        "mount_env": "EVAL_MANIFEST_URI"
      }
    ],
    "outputs": [
      {
        "name": "model",
        "asset_type": "ml_model",
        "logical_asset_id": "Ab12Cd34",
        "storage_uri_template": "gs://databrew/models/risk-detector/{{run_id}}/",
        "register": true,
        "output_identity": {
          "name": "risk_detector",
          "task": "classification",
          "target": "driving_event"
        }
      }
    ],
    "wait": {
      "mode": "container_exit",
      "timeout_seconds": 21600
    }
  }
}
```

### 6.3 TrainingStep 执行方式

MVP 推荐 **SDK/sidecar-less 模式**：训练容器直接调用 Databrew API。

Argo 节点注入环境变量：

| env | 值 |
|-----|----|
| `DATABREW_API_BASE` | 后端地址 |
| `DATABREW_TOKEN` | K8s Secret 引用 |
| `PIPELINE_DEPLOYMENT_ID` | 当前 deployment ID |
| `PIPELINE_NODE_ID` | 当前 node ID |
| `ALGO_NAME` / `ALGO_VERSION` | training.algo |
| `TRAINING_INPUTS_JSON` | training.inputs |
| `TRAINING_OUTPUTS_JSON` | training.outputs |
| `TRAINING_PARAMS_JSON` | training.params |

容器入口伪代码：

```python
run = databrew.algo_runs.create(
    algo_name=os.environ["ALGO_NAME"],
    algo_version=os.environ["ALGO_VERSION"],
    algo_kind="processing",
    triggered_by=f"pipeline:{deployment_id}:{node_id}",
    input_asset_ids=[i["asset_id"] for i in inputs],
    params=params,
    pipeline_name=pipeline_name,
    pipeline_version=pipeline_version,
)
databrew.algo_runs.start(run.run_id)

train_result = train(...)

databrew.algo_runs.finish(run.run_id, status="ok", outputs=train_result.summary)
databrew.algo_runs.register_outputs(
    run.run_id,
    outputs=resolved_outputs,
    input_lineage=inputs,
)
```

### 6.4 Transpiler 变更

后端 `transpiler.Node` 增加：

```go
type Node struct {
    ID        string        `json:"id" yaml:"id"`
    Type      string        `json:"type,omitempty" yaml:"type,omitempty"`
    Component Component     `json:"component" yaml:"component"`
    Training  *TrainingStep `json:"training,omitempty" yaml:"training,omitempty"`
    // existing fields...
}

type TrainingStep struct {
    Algo   TrainingAlgo     `json:"algo" yaml:"algo"`
    Params map[string]any   `json:"params,omitempty" yaml:"params,omitempty"`
    Inputs []TrainingInput  `json:"inputs,omitempty" yaml:"inputs,omitempty"`
    Outputs []TrainingOutput `json:"outputs,omitempty" yaml:"outputs,omitempty"`
    Wait TrainingWait       `json:"wait,omitempty" yaml:"wait,omitempty"`
}
```

转译规则：

| 节点类型 | 行为 |
|----------|------|
| 空或 `container` | 保持现有转译逻辑 |
| `training` | 使用现有 container template，但额外注入训练 env、参数、输出路径 |
| `training.wait.mode=container_exit` | Argo 节点成功退出即视为训练完成；容器负责 finish/register |
| `training.wait.mode=api_poll` | 后续扩展：wrapper 容器触发外部训练系统后轮询 AlgoRun 状态 |

### 6.5 UI 变更

Pipeline 设计器新增组件分类：

```text
组件面板
  数据处理
  训练
    - 通用训练节点
    - PyTorch 训练
    - XGBoost 训练
  评估
  导出
```

训练节点属性面板：

| 区域 | 控件 |
|------|------|
| 算法 | AlgoRun 配置下拉、版本选择 |
| 输入资产 | Asset picker，按 role 添加多行 |
| 参数 | JSON editor + 常用参数表单 |
| 输出资产 | 模型名称、logical asset 选择/新建、storage URI 模板 |
| 资源 | CPU、Memory、GPU、timeout |

---

## 7. AlgoRun 状态追踪、历史、重跑、对比

### 7.1 当前能力与增强方向

当前 `algo_runs` 已能支持基础历史列表：

```http
GET /api/v1/algo-runs?algo_name=&status=&started_after=&page=&page_size=
GET /api/v1/algo-runs/{run_id}
GET /api/v1/algo-runs/{run_id}/affected-assets
```

需要增强：

| 能力 | 设计 |
|------|------|
| 查看历史执行 | 扩展 list filter：tenant、project、pipeline、model logical asset、date range |
| 重跑 | 根据原 run 的 inputs/params/code/image 创建新 pending run |
| 对比 | 对比两个 run 的参数、指标、输入资产、输出模型 revision |
| 搜索 | ES 投影 run 摘要和关键 metadata |

### 7.2 PG schema 建议

现有 `algo_runs` 保持事实源，新增两张辅助表：

```sql
CREATE TABLE algo_run_inputs (
    run_id          TEXT        NOT NULL REFERENCES algo_runs(run_id) ON DELETE CASCADE,
    input_asset_id  TEXT        NOT NULL REFERENCES assets(asset_id),
    input_role      TEXT        NOT NULL,
    relation_type   TEXT        NOT NULL,
    required        BOOLEAN     NOT NULL DEFAULT TRUE,
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, input_asset_id, input_role)
);

CREATE TABLE algo_run_outputs (
    run_id              TEXT        NOT NULL REFERENCES algo_runs(run_id) ON DELETE CASCADE,
    artifact_name       TEXT        NOT NULL,
    artifact_role       TEXT        NOT NULL DEFAULT 'primary',
    asset_id            TEXT        NOT NULL REFERENCES assets(asset_id),
    logical_asset_id    TEXT        REFERENCES logical_assets(logical_asset_id),
    revision            BIGINT,
    asset_type          TEXT        NOT NULL,
    storage_uri         TEXT,
    metadata            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    registration_status TEXT        NOT NULL DEFAULT 'registered',
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, artifact_name)
);
```

可选增强 `algo_runs` 字段：

```sql
ALTER TABLE algo_runs
  ADD COLUMN IF NOT EXISTS parent_run_id TEXT REFERENCES algo_runs(run_id),
  ADD COLUMN IF NOT EXISTS rerun_reason TEXT,
  ADD COLUMN IF NOT EXISTS artifact_registration_status TEXT NOT NULL DEFAULT 'none';
```

状态流：

```text
pending
  |
  v
running
  |
  +--> ok ---------+
  |                |
  |                v
  |          outputs registered
  |
  +--> failed
  |
  +--> cancelled

rerun: 原 run 不改，创建新 run，new.parent_run_id = old.run_id
```

### 7.3 ES 索引设计

PG 仍为权威；ES 仅作为查询加速与列表搜索。

索引：`databrew-algo-runs-v1`

```json
{
  "run_id": "R001abc123def456",
  "algo_name": "risk_detector_train",
  "algo_version": "1.4.0",
  "status": "ok",
  "triggered_by": "pipeline:dep-123:train-risk-model",
  "tenant_id": "t1",
  "project_id": "p1",
  "started_at": "2026-05-28T09:00:00Z",
  "finished_at": "2026-05-28T10:00:00Z",
  "duration_ms": 3600000,
  "params": {
    "lr": 0.001,
    "batch_size": 64
  },
  "metrics": {
    "auc": 0.913,
    "f1": 0.842
  },
  "input_assets": [
    {"asset_id": "TrnA0012", "role": "training_dataset"}
  ],
  "output_assets": [
    {
      "asset_id": "MdL9Xa21",
      "logical_asset_id": "Ab12Cd34",
      "revision": 7,
      "asset_type": "ml_model"
    }
  ],
  "external_runtime": "argo",
  "external_url": "https://argo/.../workflow/..."
}
```

### 7.4 API 设计

#### 历史列表

```http
GET /api/v1/algo-runs?algo_name=risk_detector_train&status=ok&logical_asset_id=Ab12Cd34&page=1&page_size=50
```

响应沿用现有 envelope：

```json
{
  "items": [],
  "total": 123,
  "page": 1,
  "page_size": 50
}
```

#### 重跑

```http
POST /api/v1/algo-runs/{run_id}:rerun
Content-Type: application/json
```

请求：

```json
{
  "triggered_by": "manual:rick",
  "override_params": {
    "epochs": 30
  },
  "input_mode": "same_inputs",
  "reason": "increase epochs after underfit"
}
```

响应：

```json
{
  "run_id": "R009newRunId123",
  "parent_run_id": "R001abc123def456",
  "status": "pending"
}
```

`input_mode`：

| 值 | 行为 |
|----|------|
| `same_inputs` | 复制 `algo_run_inputs`，推荐默认 |
| `failed_assets_only` | 只复制原 run 失败资产，需要 `affected-assets` 支持状态过滤 |
| `same_filter` | 复制 `input_filter`，重新解析资产 |

#### 对比

```http
GET /api/v1/algo-runs:compare?left=R001abc123def456&right=R009newRunId123
```

响应：

```json
{
  "left": {"run_id": "R001abc123def456", "status": "ok"},
  "right": {"run_id": "R009newRunId123", "status": "ok"},
  "params_diff": [
    {"key": "epochs", "left": 20, "right": 30}
  ],
  "metrics_diff": [
    {"key": "auc", "left": 0.913, "right": 0.921, "delta": 0.008}
  ],
  "inputs": {
    "same_count": 2,
    "left_only": [],
    "right_only": []
  },
  "outputs": {
    "left_assets": [{"asset_id": "MdL9Xa21", "revision": 7}],
    "right_assets": [{"asset_id": "Qw83Lp02", "revision": 8}]
  }
}
```

---

## 8. 数据一致性与幂等

### 8.1 幂等键

| 操作 | 幂等键 |
|------|--------|
| 创建 run | `run_id` 或 `Idempotency-Key` |
| 注册产物 | `(run_id, artifact_name)` |
| 写输入 | `(run_id, input_asset_id, input_role)` |
| 写血缘 | `(parent_asset_id, child_asset_id, relation_type)` |

### 8.2 事务与并发

产出注册必须串行化同一 logical asset 的 revision 分配：

```text
BEGIN
  SELECT logical_assets WHERE logical_asset_id=$1 FOR UPDATE
  max_revision + 1
  clear previous current
  insert new assets revision
  bump logical_assets
COMMIT
```

并发冲突返回 409 或内部重试一次；不允许出现两个 current revision。

### 8.3 事件

建议追加 `asset_events`：

| event_type | aggregate_type | asset_id | payload |
|------------|----------------|----------|---------|
| `algo_output_registered` | `asset` | 输出 asset_id | run_id、artifact_name、logical_asset_id、revision |
| `algo_lineage_recorded` | `asset` | 输出 asset_id | input_asset_ids、relation_types |
| `training_run_rerun_created` | `algo_run` | new run_id | parent_run_id、reason |

---

## 9. 实施路径

### Phase 0：设计评审与约束确认

目标：冻结术语和最小 schema。

交付：

| 项 | 内容 |
|----|------|
| relation type | 确认 `trained_from`、`evaluated_on`、`configured_by` 等白名单 |
| asset type | 确认 `ml_model`、`dataset` 是否进入正式资产类型 |
| 版本策略 | 确认默认是否自动设为 current |
| API 策略 | 确认两步式 `/finish` + `/outputs:register` |

### Phase 1：MVP，产出注册 + 输入血缘

目标：不改 Pipeline UI，也能让训练系统通过 API 接入资产闭环。

后端：

1. 新增 migration：
   - `algo_run_inputs`
   - `algo_run_outputs`
   - 扩展 `asset_relations.relation_type` CHECK
   - `asset_relations.metadata JSONB`
2. 新增 handler/usecase/repository：
   - `POST /api/v1/algo-runs/{run_id}/outputs:register`
   - `GET /api/v1/algo-runs/{run_id}/outputs`
3. 复用资产版本逻辑创建 `ml_model` / `dataset` revision。
4. 注册成功时写 `asset_relations`。
5. API contract sync：OpenAPI、api-guide、SDK、smoke。

验收：

```text
Given 一个 ok 状态 AlgoRun
And 输入资产 TrnA0012 / Eval0099 存在
When 调用 outputs:register 注册 ml_model
Then 创建新 assets row
And logical_assets revision +1
And asset_relations 包含 trained_from / evaluated_on
And GET asset lineage 能看到训练输入
And GET /algo-runs/{run_id}/outputs 返回输出资产
```

### Phase 2：AlgoRun 执行追踪增强

目标：支持历史、重跑、对比。

交付：

1. `POST /api/v1/algo-runs/{run_id}:rerun`
2. `GET /api/v1/algo-runs:compare`
3. 列表增加 `logical_asset_id`、`parent_run_id`、`pipeline_name` 过滤。
4. ES `algo_run` 投影增强。
5. 前端 AlgoRunsPage 增加输出资产、指标摘要、重跑按钮、对比入口。

### Phase 3：Pipeline 训练节点集成

目标：训练可在设计器中一等编排。

后端：

1. `transpiler.Node` 增加 `type` 和 `training` schema。
2. TrainingStep 转译注入 Databrew API env 与训练输入/输出 JSON。
3. 部署前校验输入资产存在、logical asset 类型匹配。
4. Workflow detail 展示关联 `run_id`。

前端：

1. 组件 registry 增加 `training` 类别。
2. 节点属性面板支持 AlgoRun 配置、输入角色、输出资产。
3. 部署记录和 Workflow 详情链接 AlgoRun 详情。

### Phase 4：高级 MLOps 能力

目标：模型治理闭环。

候选：

| 能力 | 说明 |
|------|------|
| 模型候选/发布状态 | `candidate` -> `approved` -> `serving` |
| 模型评估报告资产 | confusion matrix、ROC、报告 HTML 资产化 |
| MLflow / Vertex / Argo 深链 | `external_runtime` / `external_url` 标准化 |
| OpenLineage 输出 | 将训练血缘投递 Marquez |
| BigLake 训练集快照 | dataset manifest 与 Iceberg snapshot 对齐 |

---

## 10. 风险与取舍

| 风险 | 影响 | 缓解 |
|------|------|------|
| relation_type 白名单扩展影响现有约束 | migration 风险 | 单独 migration，先 dev 验证 lineage 查询 |
| 训练产物注册与 finish 强耦合 | finish 成功但注册失败难处理 | MVP 使用独立 `outputs:register`，可重试 |
| logical asset 自动匹配误判 | 错把新模型写成旧模型 revision | 默认要求显式 `logical_asset_id` 或 `create_logical_asset=true` |
| metadata 过大 | PG row 膨胀 | 大对象放 GCS，只存 URI 和摘要 |
| Pipeline training 节点过早复杂化 | UI/Transpiler 改动大 | 放到 Phase 3，MVP 先 API 接入 |
| ES 与 PG 不一致 | 搜索结果延迟 | PG 为事实源，详情页总是回源 PG |

---

## 11. 推荐 MVP 范围

本次最小可交付建议只做：

1. `algo_run_inputs` / `algo_run_outputs` 表。
2. 扩展 `asset_relations` 支持 `trained_from`、`evaluated_on`、`configured_by`。
3. `POST /api/v1/algo-runs/{run_id}/outputs:register`。
4. 注册 `ml_model` / `dataset` 资产 revision。
5. 自动写输入到输出的血缘边。
6. `GET /api/v1/algo-runs/{run_id}/outputs`。

暂缓：

| 暂缓项 | 原因 |
|--------|------|
| Pipeline 训练节点 UI | 涉及前端画布、schema、transpiler、Argo 注入，改动大 |
| run 对比 | 依赖指标规范和输出表稳定 |
| ES 投影 | PG 查询足够支撑 MVP |
| 模型发布治理 | 需要产品状态机单独评审 |

MVP 完成后，平台即可回答：

```text
这个模型资产是哪次 AlgoRun 产出的？
这次训练用了哪些训练集和评估集？
这些输入数据后续影响了哪些模型版本？
某个 run 产出了哪些资产 revision？
```
