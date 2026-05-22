# 算法 Run 可视化对接：MLflow Tracking

| 字段 | 值 |
|------|----|
| 状态 | Proposal（P2 路线图，未排期） |
| 关联 | `algo-runs.md` / `../schema.md` §5 algo_runs / `openlineage-integration.md` |
| 决策来源 | 待评审（rev.13+ 候选） |

---

## 0. TL;DR

**问题**：你们已有完整的 `algo_runs` 数据模型（status / params / metrics / 关联 asset），但**没有可视化前端给算法工程师用**。算法同学想看「我那次跑了啥参数 / 指标曲线 / 失败原因 / 比对两次跑的差异」，得自己写 SQL。

**方案**：拉 [MLflow Tracking](https://mlflow.org/) 当**算法工程师专用面板**，**不取代 PG 事实源**：

- PG `algo_runs` + 关联表 = **事实源**（合规、强约束、多租户）
- MLflow UI = **展示层**（算法工程师习惯的工具，零学习成本）
- 通过 outbox emitter 把 PG 事件转 MLflow REST API 调用

**效果**：
- 算法工程师不用学新工具，沿用 MLflow 习惯（`mlflow ui` 那一套）
- 比对两次 run 参数/指标差异、画 metric 曲线、看失败堆栈 —— 都不用工程团队造轮子
- 部署成本极低（docker 一行起）
- 跟 [`openlineage-integration.md`](openlineage-integration.md) 平行：**Marquez 看血缘 + MLflow 看 run 详情**，分工清晰

---

## 1. 现状问题

### 1.1 算法工程师常问

| 问题 | 当前怎么搞 |
|---|---|
| 「3 个月前 R789 那次 hand_track@2.0 用的什么参数？」 | 写 SQL 查 `algo_runs.run_inputs` JSONB |
| 「比较 R789 vs R790 两次跑的指标差异」 | 拉 `asset_eval_results` 两行手动对比 |
| 「画 hand_track 这个算法过去 100 次跑的 mAP 曲线」 | SQL + 自己画图 |
| 「这次跑失败了，stack trace 在哪？」 | 看 `algo_runs.error_message`，没有上下文 |
| 「这次跑了多少 GPU 时长？花了多少钱？」 | `cost_usd` 字段单值，没分时段曲线 |

### 1.2 核心 gap

**事实源（PG）有数据，但没有算法工程师的母语界面**。MLflow UI 在 ML 圈是事实标准，多数算法同学已经熟，迁过来零成本。

---

## 2. MLflow Tracking 是什么

### 2.1 身份

[MLflow](https://mlflow.org/) 是 Databricks 开源的**机器学习全生命周期平台**（Apache 2.0），4 个子模块：

| 模块 | 干啥 | 你们用得上吗 |
|---|---|---|
| **MLflow Tracking** | 记录每次实验 run（params / metrics / artifacts / tags） | ✅ 直接用 |
| **MLflow Models** | 模型文件 + signature + flavor 标准化打包 | 🟡 看是否需要追踪算法权重文件 |
| **MLflow Model Registry** | 模型版本管理 + Stage（Staging/Production）| 🔴 你们 `assets` 已经做了类似事 |
| **MLflow Projects** | 可复现的算法运行打包 | 🔴 你们 algo_sdk 已有自己的封装 |

**只接 Tracking 子模块**，其他三个不需要。

### 2.2 核心数据模型

```
Experiment（实验，类似 algo_name）
  └─ Run（一次跑的实例）
      ├─ Params      key-value，输入参数
      ├─ Metrics     time-series 数值，训练/评估指标
      ├─ Tags        任意 key-value 标签
      ├─ Artifacts   输出文件（S3/GCS URI）
      └─ Source      代码版本（git commit / 仓库 URI）
```

### 2.3 字段对照（你们 algo_runs ↔ MLflow Run）

| 你们字段 | MLflow 字段 | 备注 |
|---|---|---|
| `algo_runs.run_id` | `run_id`（UUID）| 1:1，但 MLflow 默认生成 UUID，需要 mapping |
| `algo_runs.algo_name` | `experiment_name` | MLflow 用 experiment 概念 group runs |
| `algo_runs.algo_version` | `tags.mlflow.source.git_commit` 或自定义 tag | |
| `algo_runs.started_at` / `completed_at` | `start_time` / `end_time` | 1:1 |
| `algo_runs.status` (running/completed/failed) | `status` (RUNNING / FINISHED / FAILED / KILLED) | 状态机映射 |
| `algo_runs.run_inputs` JSONB | `params`（每个 KV 一行）| 拆开投递 |
| `algo_runs.error_message` | `tags.mlflow.runFailureReason` | |
| `algo_runs.triggered_by` | `tags.mlflow.user` | |
| `algo_runs.cost_usd` / `gpu_seconds` | 自定义 metrics（`cost_usd` / `gpu_hours`）| 数值用 metric 而非 param |
| `asset_metrics`（关联 run 的）| `metrics`（time-series）| 投递时按 run_id 聚合 |
| `asset_eval_results`（关联 run 的）| `metrics`（标量）+ `tags.eval.*` | 数值进 metrics，标签信息进 tags |
| `assets.storage_uri`（run 产出的）| `artifacts`（URI 列表）| |
| `algo_runs.parent_run_id`（如有 pipeline 嵌套） | `tags.mlflow.parentRunId` | |

### 2.4 你们 vs MLflow 的关键差异

不是简单的「重新发明」，**有合理差异**：

| 差异点 | MLflow | 你们 | 为什么 |
|---|---|---|---|
| **Artifact 一等公民** | artifact 是 run 附属，URI 字符串 | `assets` 是核心实体，`algo_runs` 通过 `derived_from` 边关联 | artifact 有强业务含义和血缘，需要独立建模 |
| **多版本资产** | 没有 logical / revision | `logical_asset_id` + `revision` 严格线性 | 合规约束（客户引用旧版不被覆盖）|
| **血缘** | 弱（只到 input run）| 强（`asset_relations` 6 种边）| 不只追溯算法 run，还追溯 asset 派生关系 |
| **审计** | 弱（事件不分级）| 三段式 events（actor + system_metadata + payload）| 合规需求 + outbox CDC 投影 |
| **多租户** | 弱（per-experiment ACL）| `tenant_id` / `project_id` 普遍存在 | 客户隔离 |

→ MLflow **不能取代你们的 PG**，但完全可以**当展示层**。

---

## 3. 整合架构

### 3.1 数据流

```
┌──────────────────────────────────────┐
│ AssetWriter（你们已有）              │
│ ├─ INSERT algo_runs                  │
│ ├─ INSERT asset_metrics              │
│ ├─ INSERT asset_eval_results         │
│ └─ INSERT asset_events               │
└──────────┬──────────────────────────┘
           │ 同事务 outbox
           ▼
┌──────────────────────────────────────┐
│ Pub/Sub Topic: algo_run_stream       │
└──────────┬──────────────────────────┘
           │ subscribe
           ▼
┌──────────────────────────────────────┐
│ MLflow Emitter Service（新）         │ ★ 唯一新写代码
│ - 订阅 algo_run_started/completed    │
│ - 订阅 asset_metrics 写入             │
│ - 调 MLflow REST API:                │
│   - POST /runs/create                 │
│   - POST /runs/log-parameter          │
│   - POST /runs/log-metric             │
│   - POST /runs/update (set status)   │
└──────────┬──────────────────────────┘
           │ MLflow REST API
           ▼
┌──────────────────────────────────────┐
│ MLflow Tracking Server（开源现成）   │
│ ├─ Backend: Postgres / SQLite        │
│ ├─ Artifact store: GCS / S3 (URI)    │
│ └─ UI: http://mlflow/...              │
└──────────────────────────────────────┘
           ▲
           │ 浏览
           │
       算法工程师 / 数据科学家
```

### 3.2 关键设计原则

| 原则 | 说明 |
|---|---|
| **事实源在 PG**，MLflow 只投影 | MLflow 挂了不影响业务 |
| **异步投递** | 不在主事务里调 MLflow API |
| **at-least-once + 幂等** | MLflow `run_id` 用你们的 `algo_runs.run_id`（覆盖默认 UUID）保证幂等 |
| **失败可重放** | Pub/Sub 消息保留，MLflow 重启可重建 |
| **artifact 不真传** | 只传 URI（`gs://...`），文件本身不动 |

### 3.3 状态机映射

| 你们 `algo_runs.status` | MLflow `Run.status` |
|---|---|
| `pending` | `SCHEDULED` |
| `running` | `RUNNING` |
| `completed` | `FINISHED` |
| `failed` | `FAILED` |
| `cancelled` | `KILLED` |

---

## 4. 落地路线图

### Phase 1：POC（1 周）

| Day | 任务 |
|---|---|
| 1 | docker run 起 MLflow Tracking Server（SQLite backend，GCS artifact store） |
| 2-3 | 写 emitter：订阅 `algo_run_started/completed`，调 MLflow REST 创建 run + log params |
| 4 | 跑一次真实 algo run，UI 验证看到 run + params + metrics |
| 5 | 补 status 切换 + error message |

**交付**：MLflow UI 能看到 1-2 次 run 的完整记录（params、metrics、artifact URI）。

### Phase 2：metric 曲线 + 完整字段（2 周）

| 周 | 任务 |
|---|---|
| 1 | 投递 `asset_metrics` 时间序列（按 step 投 metric）|
| 1 | 投递 `asset_eval_results` 标量（mAP / accuracy 等）|
| 2 | 投递 `algo_runs.cost_usd` / `gpu_seconds` 当 metric |
| 2 | git_commit / triggered_by / parent_run_id 等 tag 完整 |

**交付**：算法工程师可以画曲线、比对 run、看堆栈。

### Phase 3：生产部署（1 个月）

| 周 | 任务 |
|---|---|
| 1 | Helm chart 部署 MLflow 到 K8s（PG backend）|
| 2 | emitter 接现网 Pub/Sub |
| 3 | Cloud Monitoring 告警 |
| 4 | 灰度 → 全量 |

**交付**：prod MLflow UI，覆盖 100% algo_runs。

### Phase 4：可选扩展

| 项 | 说明 |
|---|---|
| **MLflow Models 接入** | 如果要追踪算法权重文件版本 |
| **A/B compare UI** | MLflow 自带 run 对比，无需开发 |
| **AutoML / hyperparameter sweep 集成** | 跟 Optuna/Ray Tune 连动 |
| **MLflow + Marquez 双向链接** | UI 上互跳（同 run_id）|

---

## 5. 解决的问题 / 带来的效果

### 5.1 Before → After

| 问题 | Before | After |
|---|---|---|
| 「R789 用了什么参数？」 | SQL | UI 点开 run 看 params 表 |
| 「比对两次 run」 | 自己 diff | UI 选两个 run 自动 side-by-side |
| 「画 mAP 曲线」 | 自己写 plot | UI 选 metric 自动画 |
| 「看失败堆栈」 | 看 error_message 单字段 | UI 完整 log + tags |
| 「找 cost_usd 最高的 10 次跑」 | SQL | UI sort by metric |

### 5.2 量化效果

| 指标 | 估计 |
|---|---|
| 算法工程师查 run 信息时间 | ~5 分钟 SQL → ~5 秒 UI |
| 自研算法面板节省 | 2-3 工程师月 |
| 算法工程师上手成本 | 0（已习惯 MLflow）|
| 部署成本 | 1 vCPU + 2 GB RAM（中小规模） |

### 5.3 不解决的问题

| 不解决 | 原因 |
|---|---|
| 业务血缘可视化 | 那是 Marquez 的事（OpenLineage） |
| 多版本资产管理 | 那是你们 PG 的事 |
| 合规审计 | MLflow 不做 append-only event log |
| 行级权限 | MLflow ACL 弱，敏感数据走 PG |

---

## 6. 双前端分工：MLflow vs Marquez

跟 [`openlineage-integration.md`](openlineage-integration.md) 平行存在，**不冲突**：

| 维度 | Marquez | MLflow |
|---|---|---|
| **看什么** | 血缘 DAG（asset → algo → asset）| algo run 详情（params / metrics / artifacts）|
| **谁用** | PM / 客户 / 数据治理 / 工程师 | 算法工程师 / 数据科学家 |
| **核心实体** | Dataset + Job | Run |
| **协议** | OpenLineage 标准 | MLflow REST（事实标准）|
| **重叠** | 都关注 run，但 Marquez 偏「连接」，MLflow 偏「细节」 | |

理想流：

```
1. 算法工程师在 MLflow UI 看到 R789 这次跑的 metric 曲线
2. 想知道 R789 影响了哪些 asset，点进 asset 链接（自定义 tag 跳 Marquez）
3. Marquez 上看血缘图，发现 R789 产出的 clipX 还被下游 R815 处理过
4. 跳回 MLflow 查 R815 的细节
```

---

## 7. 风险 / 决策点

| 风险 | 缓解 |
|---|---|
| MLflow run_id UUID vs 你们 8 位 ID | emitter 用 `tags.external_run_id` 存你们 ID 反查 |
| 大量 metric 时间点投递压力 | emitter 批量 + 限速；高频 metric 降采样 |
| MLflow 没有删除事件 | 业务上 algo_runs 也很少删，可接受残留 |
| 多租户隔离 | 用 MLflow experiment per-tenant 或 tags.tenant_id |
| backend 选 PG 还是 SQLite | prod 必 PG（MLflow 推荐）|

**需要业务决策**：

1. **算法工程师真用 MLflow 吗？** 如果你们已有别的工具（wandb / 自研），可能不需要
2. **MLflow 部署在哪？** 跟 Marquez 同集群还是分开
3. **是否启用 MLflow Models 模块？** 涉及算法权重文件的版本管理是否走 MLflow

---

## 8. 参考资料

- [MLflow 官网](https://mlflow.org/)
- [MLflow Tracking 文档](https://mlflow.org/docs/latest/tracking.html)
- [MLflow REST API](https://mlflow.org/docs/latest/rest-api.html)
- [MLflow Helm chart（社区维护）](https://github.com/community-charts/helm-charts/tree/main/charts/mlflow)
- [MLflow vs DVC vs wandb 对比（社区文章）](https://neptune.ai/blog/best-ml-experiment-tracking-tools)
- [Databricks MLflow architecture](https://databricks.com/blog/2018/06/05/introducing-mlflow-an-open-source-machine-learning-platform.html)

---

## 9. 一句话总结

> **MLflow Tracking 是 ML 圈的「Run 详情面板」事实标准**。你们 `algo_runs` 数据模型跟它 90% 重合，**别取代 PG，但拉它当算法工程师的展示层**。加一个 outbox emitter，1 周 POC，2 个月生产，节省 2-3 工程师月。**配合 Marquez 双前端**：业务血缘看 Marquez，算法 run 详情看 MLflow，分工清晰。
