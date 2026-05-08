# Eval Metrics 设计（Phase 1.5）

本文档定义资产评估指标（Eval / Metrics）在 `Databricks4robot` 中的目标形态、数据边界、表结构、事件契约与分阶段落地路径。

目标：在不破坏现有 `asset_tags / asset_algo_latest / asset_events` 主线的前提下，让质量评估结果成为一等公民，支持过滤、聚合、入 ES、入湖分析。

---

## 1. 核心结论

- `asset_algo_latest` 只表达算法运行状态，不承载质量评估指标。
- `asset_eval_results` 保存一次评估运行的完整原始结果（`result_payload`）。
- `asset_metrics` 保存可检索、可聚合的指标投影，不扫 JSONB。
- `metric_registry` 作为可查询指标白名单与语义中心。
- `asset_events` 承载 eval/metric 事件，沿用 Pure CDC + `event_seq` 主线。
- 指标不直接驱动 `lifecycle_state`，必须通过显式规则决策。

---

## 2. 目标与非目标

### 2.1 目标

- 支持对 `video / segment / action / frame / frame_set / annotation / delivery_item` 写入评估结果。
- 支持按指标过滤资产（例如 `good_frames_ratio >= 0.8`）。
- 支持统计与分布分析（P50/P90、分桶、按 eval 版本对比）。
- 支持 ES 检索与 Iceberg 入湖，保持与 Pure CDC 主链路一致。

### 2.2 非目标

- 不引入新的多租户模型。
- 不把所有原始 payload 字段都强行列化为主表列。
- 不让 metric 直接触发生命周期变更（避免隐式业务规则）。

---

## 3. 领域边界

- `asset_algo_latest`：算法执行状态（pending/running/ok/failed/blocked）与执行元信息。
- `asset_eval_results`：评估运行事实（谁在何时用何参数评估了什么，结果原文是什么）。
- `asset_metrics`：可检索指标投影（面向过滤/排序/聚合）。
- `asset_tags`：业务解释结果（如 `quality_level=bad`、`qc.deliverable=false`）。
- `asset_events`：以上状态变化与事实变化的统一事件轨迹。

边界约束：

1. `good_frames_ratio`、`total_good_frames`、`weighted_avg_convex_hull_volume` 等数值不进入 `assets.metadata`。
2. 未注册指标不得进入 `asset_metrics`（但保留在 `asset_eval_results.result_payload`）。
3. `lifecycle_state` 只能由显式 quality rule / QA rule 变更。

---

## 4. 数据模型

## 4.1 `asset_eval_results`（原始事实）

```sql
CREATE TABLE asset_eval_results (
    eval_result_id UUID PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    mcap_file_id UUID NULL,

    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL DEFAULT '',

    eval_name TEXT NOT NULL,
    eval_version TEXT NOT NULL,
    parameter_version TEXT NULL,
    run_id TEXT NULL,
    status TEXT NOT NULL,

    result_payload JSONB NOT NULL DEFAULT '{}',
    output_uri TEXT NULL,
    summary_uri TEXT NULL,

    source_type TEXT NOT NULL,
    source_name TEXT NULL,
    source_version TEXT NULL,

    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

建议索引：

- `(asset_id, target_type, target_id, created_at DESC)`
- `(eval_name, eval_version, parameter_version, created_at DESC)`
- `(run_id)`（where run_id is not null）

## 4.2 `asset_metrics`（可查询投影）

```sql
CREATE TABLE asset_metrics (
    asset_id TEXT NOT NULL REFERENCES assets(asset_id),
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL DEFAULT '',

    metric_key TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    metric_unit TEXT NULL,

    metric_value DOUBLE PRECISION NULL,
    metric_value_int BIGINT NULL,
    metric_value_text TEXT NULL,
    metric_value_bool BOOLEAN NULL,

    eval_name TEXT NOT NULL,
    eval_version TEXT NOT NULL,
    parameter_version TEXT NULL,
    run_id TEXT NULL,

    source_type TEXT NOT NULL,
    source_name TEXT NULL,
    source_version TEXT NULL,
    confidence DOUBLE PRECISION NULL,

    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (
        asset_id,
        target_type,
        target_id,
        metric_key,
        eval_name,
        eval_version
    )
);
```

说明：主键显式包含 `eval_version`，避免不同版本评估相互覆盖。

建议索引：

```sql
CREATE INDEX idx_asset_metrics_key_value
ON asset_metrics (metric_key, metric_value);

CREATE INDEX idx_asset_metrics_asset
ON asset_metrics (asset_id, target_type, target_id);

CREATE INDEX idx_asset_metrics_eval
ON asset_metrics (eval_name, eval_version, parameter_version);
```

## 4.3 `metric_registry`（注册中心）

最小落地可以先用 YAML（`backend/config/metric_registry.yaml`），后续可映射到配置表。

字段建议：

- `metric_key`
- `display_name`
- `metric_type`（`count|ratio|score|gauge|duration|size|histogram|bool|string`）
- `metric_unit`
- `target_type`
- `higher_is_better`
- `default_aggregation`
- `queryable`
- `description`

强约束：

- `queryable=true` 的 key 才允许写入 `asset_metrics` 与 ES `metrics_flat/nested`。
- 未注册 key 只保留在 `asset_eval_results.result_payload`，并打观测指标。

---

## 5. 写入流程

`POST /api/v1/assets/{asset_id}/eval-results` 的事务语义：

1. 写 `asset_eval_results`（完整 payload）。
2. 读取 `metric_registry`，做白名单展开写 `asset_metrics`。
3. 可选：根据 quality rule 写 `asset_tags` 或 `qa_issue`。
4. 追加 `asset_events`（`eval_result_reported`、`metric_upserted` 等）。
5. COMMIT 后由 CDC consumers 同步到 ES / Lakehouse。

注意：

- 步骤 1-4 必须同事务。
- metric 自身不直接修改 `assets.lifecycle_state`。

---

## 6. 事件契约（新增）

建议新增事件类型：

- `eval_started`
- `eval_finished`
- `eval_failed`
- `eval_result_reported`
- `metric_upserted`
- `metric_deleted`
- `metric_rule_triggered`

payload 关键字段：

- `asset_id`
- `target_type`
- `target_id`
- `eval_name`
- `eval_version`
- `parameter_version`
- `run_id`
- `metric_keys`（批量 upsert 时）

---

## 7. 读路径设计

- 资产详情：`assets + asset_tags + asset_algo_latest + asset_metrics(latest)`
- 资产筛选（PG fallback）：支持 `metric.<key>.<op>`（如 `metric.good_frames_ratio.gte=0.8`）
- ES 检索（2.0）：`metrics_flat`（高频简单过滤）+ `metrics` nested（复杂组合过滤）
- 湖仓分析（2.0+）：`silver.asset_eval_results` + `gold.asset_metric_latest`

---

## 8. API 设计（新增 K 组）

- `K1` `POST /api/v1/assets/{asset_id}/eval-results` 🟢
- `K2` `GET /api/v1/assets/{asset_id}/eval-results` 🟢
- `K3` `GET /api/v1/assets/{asset_id}/metrics` 🟢
- `K4` ~~`POST /api/v1/eval-results:batchReport`~~ **未实现**（无独立 batch 路由）：批量场景请**循环调用 K1** 或走离线作业；若未来提供 batch，将单独发版并更新 `api/openapi.yaml`
- `K5` `GET /api/v1/metrics/registry` 🟢
- `K6` `POST /api/v1/metrics:search` 🟢

---

## 9. 分阶段落地

## 9.1 Phase 1.5（MVP）

1. 新增 `metric_registry`（先 YAML）。
2. 新增 PG 表：`asset_eval_results`、`asset_metrics`。
3. 新增 K1/K2/K3 核心 API。
4. 实现 registry 白名单展开（未注册 key 不进投影）。
5. 增加 `eval_result_reported` / `metric_upserted` 事件。
6. 前端资产详情页展示 metrics latest。

## 9.2 Phase 2

1. ES 文档增加 `metrics_flat + metrics nested`。
2. CDC consumer 同步 metric 事件到 ES。
3. Iceberg 增加 `silver.asset_eval_results`、`gold.asset_metric_latest`。
4. Trino 提供质量分布分析查询。

---

## 10. 可观测性

新增指标：

- eval 写入成功率
- eval 上报延迟
- 未注册 metric key 数量
- metrics 展开失败数量
- `metric_upserted` CDC lag
- ES metrics 同步延迟
- Iceberg `asset_metric_latest` 入湖延迟

---

## 11. 关键风险

- **R-eval-1**：metric key 无约束扩张导致 ES mapping 爆炸。  
  对策：registry 白名单 + 未注册 key 仅入 `result_payload`。
- **R-eval-2**：不同 eval 版本互相覆盖。  
  对策：`asset_metrics` 主键纳入 `eval_version`。
- **R-eval-3**：指标直接绑定生命周期导致隐式规则失控。  
  对策：指标与状态机解耦，必须通过显式 quality/QA 规则落地。

