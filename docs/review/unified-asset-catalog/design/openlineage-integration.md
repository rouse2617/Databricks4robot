# 血缘生态对接：OpenLineage / DataHub / Marquez

| 字段 | 值 |
|------|----|
| 状态 | Proposal（P2 路线图，未排期） |
| 关联 | `../schema.md` §11 asset_events / `algo-runs.md` / `asset-versioning.md` §4 / `asset-hierarchy-and-derivatives.md` §3.2 asset_relations |
| 决策来源 | 待评审（rev.13+ 候选） |

---

## 0. TL;DR

**问题**：你们已经有完整的资产 + 血缘 + 多版本数据模型（PG `assets` / `algo_runs` / `asset_relations` / `asset_events`），但**没有可视化前端**，工程师查血缘只能写 SQL；要给客户展示「这个 clip 来自哪、被哪些算法处理过」也得自己撸 UI。

**方案**：拉三个开源组件接进来，**不改你们的事实源**：

1. **[OpenLineage](https://openlineage.io/)** —— 业界血缘事件标准协议（schema + JSON 事件格式）
2. **[DataHub](https://datahubproject.io/) SystemMetadata** —— 你们 `asset_events.system_metadata` 字段已经在抄它，确认对齐即可
3. **[Marquez](https://marquezproject.ai/)** —— OpenLineage 官方后端 + UI，**直接当血缘可视化前端**

**效果**：
- 工程师/PM/客户**不写 SQL** 就能看血缘 DAG（哪个 segment → 哪些算法 → 哪些 clip）
- 不用自己造血缘可视化轮子（自研 UI 至少 2-3 个月）
- 未来对接外部数据治理生态（DataHub 实例 / 数据湖团队 / 客户的元数据系统）有标准协议
- 工程量小：**加一个 outbox + emitter 服务，1-2 周 POC**

---

## 1. 现状问题

### 1.1 痛点列表

| 痛点 | 当前怎么搞 | 问题 |
|------|----------|------|
| 「这个 clipX_v3 是哪个算法跑出来的？」 | 写 SQL 查 `asset_relations` JOIN `algo_runs` | 工程师才会，PM 不会 |
| 「hand_track@2.0 到现在跑出过哪些资产？」 | 写 SQL | 同上 |
| 「这个 segment 上后面被哪些下游产物用过？」 | 递归 SQL（CTE）| 复杂度高 |
| 「给客户演示血缘」 | 没法演示，只有 SQL 结果 | 客户体验差 |
| 「数据治理团队要看全公司元数据」 | 没接口 | 没法接 |
| 「外部数据湖团队想消费血缘事件」 | 没标准格式 | 各自适配地狱 |

### 1.2 核心 gap

**事实源（PG）很完整，但缺三层**：

```
┌──────────────────┐
│ 应用层（缺）     │  ← 缺血缘 UI、外部消费协议
├──────────────────┤
│ 协议层（缺）     │  ← 缺标准化事件输出
├──────────────────┤
│ 事实源（已有）   │  ← assets / asset_relations / algo_runs / asset_events
└──────────────────┘
```

要补的就是中间和上面两层，**别自己造**。

---

## 2. 三个组件分别是什么

### 2.1 OpenLineage（协议层）

**身份**：LF AI & Data 基金会孵化的**血缘事件标准协议**，2021 启动，2024 进入 LF Graduated 阶段。Airflow / dbt / Spark / Flink 等主流编排器都已原生集成。

**核心理念**：用 JSON 事件描述「**一个 job 跑了一次，输入是什么、输出是什么**」。事件发到收集端（如 Marquez），就能拼出完整血缘图。

**核心数据模型**（4 个对象）：

| 对象 | 含义 | 你们对应 |
|---|---|---|
| **Job** | 算法/管道的身份（`namespace + name`，长期不变）| `algo_runs.algo_name` |
| **Run** | Job 的一次执行实例（`runId` 唯一）| `algo_runs.run_id` |
| **Dataset** | 输入/输出数据（`namespace + name`）| `assets.asset_id` 或 `logical_asset_id` |
| **Facet** | 可扩展元数据片段（schema/version/quality...）| `assets.metadata` JSONB / `asset_metrics` |

**事件类型**：`START` / `RUNNING` / `COMPLETE` / `ABORT` / `FAIL`

**最小事件示例**（你们 `hand_track@2.0` 跑一次产出 clipX_v3）：

```json
{
  "eventType": "COMPLETE",
  "eventTime": "2026-05-22T10:00:00Z",
  "producer": "https://github.com/yourcorp/cyber-databrew/v1.0",
  "schemaURL": "https://openlineage.io/spec/2-0-2/OpenLineage.json",

  "run": {
    "runId": "R012",
    "facets": {}
  },
  "job": {
    "namespace": "cyber-databrew.algo",
    "name": "hand_track",
    "facets": {
      "documentation": { "description": "Hand tracking algorithm v2" },
      "ownership": { "owners": [{"name": "rick"}] }
    }
  },
  "inputs": [
    {
      "namespace": "cyber-databrew.assets",
      "name": "segment_A"
    }
  ],
  "outputs": [
    {
      "namespace": "cyber-databrew.assets",
      "name": "clipX_v3",
      "facets": {
        "version": { "datasetVersion": "3" },
        "dataSource": { "uri": "gs://bucket/clips/clipX_v3.mp4" },
        "schema": { ... }
      }
    }
  ]
}
```

**关键 Facet 清单**（直接对应你们字段）：

| Facet | 内容 | 你们映射 |
|---|---|---|
| `dataSource` | 数据源 URI | `assets.storage_uri` |
| `version`（dataset facet）| 数据集版本号 | `assets.revision` ✅ 完美对应 |
| `schema` | 字段结构 | derived_asset 的 parquet/json schema |
| `dataQualityMetrics` | 行数/null 率/统计 | `asset_metrics` |
| `columnLineage` | 字段级血缘 | 你们暂不需要 |
| `parent`（run facet）| 父 run（pipeline 嵌套）| `algo_runs.parent_run_id` |
| `errorMessage` | 失败原因 | `algo_runs.error_message` |
| `ownership` | 所有人 | `assets.owner` |
| `nominalTime` | 业务时间窗 | `assets.start/end_timestamp_ns` |

**协议产生的 effect**：

- **生态兼容**：Airflow/dbt/Spark 任务跟你们自研算法在同一张血缘图里
- **标准化**：未来换可视化前端、加多个消费方都不用改 emitter
- **可演进**：新需求加 facet 不破坏老消费方

---

### 2.2 DataHub SystemMetadata（元数据元数据）

**身份**：[DataHub](https://datahubproject.io/) 是 LinkedIn 开源的元数据平台（Acryl Data 商业化运营）。`SystemMetadata` 是它**给每条元数据变更附带的「审计 + 来源」附加信息**。

**为什么提这个**：你们 `asset_events.system_metadata` JSONB **已经在抄它的设计**（rev.10 决策来源里有提）。这一节是确认对齐 + 给未来对接 DataHub 实例铺路。

**Schema**（DataHub `SystemMetadata.pdl`）：

```json
{
  "lastObserved": 1716364800000,
  "runId": "R789",
  "lastRunId": "R788",
  "registryName": "default",
  "registryVersion": "0.10.5",
  "properties": {
    "pipeline_name": "hand_track_pipeline",
    "pipeline_version": "v1.2",
    "ingestion_source": "batch",
    "actor": "user:rick"
  }
}
```

**字段对照**：

| DataHub 字段 | 你们对应 | 备注 |
|---|---|---|
| `lastObserved` | `asset_events.event_time` | 时间戳 |
| `runId` | `system_metadata->>'run_id'` | ✅ 完全对齐 |
| `lastRunId` | 用 `asset_events` 历史推 | 你们没单独存，可推 |
| `registryName/Version` | 暂无 | 元数据 schema 的版本管理，先不上 |
| `properties` | `system_metadata` JSONB 其他字段 | 自由扩展 ✅ |

**你们的 super-set**：把高频审计字段（`actor` / `request_id` / `idempotency_key` / `caller_ip`）拎到顶层做索引，并加了 `event_payload` 字段级 diff（DataHub 没有，更强）。

**对接 DataHub 实例时**做个映射器即可：

```python
def to_datahub_system_metadata(asset_event):
    return {
        "lastObserved": int(asset_event.event_time.timestamp() * 1000),
        "runId": asset_event.system_metadata.get('run_id'),
        "properties": {
            "actor": asset_event.actor,
            "request_id": asset_event.request_id,
            "pipeline_name": asset_event.system_metadata.get('pipeline_name'),
            "algo_name": asset_event.system_metadata.get('algo_name'),
            "algo_version": asset_event.system_metadata.get('algo_version'),
        }
    }
```

**DataHub 完整 Aspect 模型**也值得了解（`DatasetProperties` / `Ownership` / `SchemaMetadata` / `DataQuality` / `Lineage` / `Versioning`）。你们 41 列 `assets` + 8 张 satellite 表本质上就是把这些 aspect **物化到 PG** 的实现。设计思路一致，只是存储介质不同。

**协议产生的 effect**：

- **未来要接 DataHub 实例零摩擦**：mapper 写一次，事件流就过去了
- **审计语义对齐业界**：未来招的人/外部审计员看 system_metadata 字段不需要重新学

---

### 2.3 Marquez（可视化前端）

**身份**：OpenLineage 项目的官方后端 + Web UI，由 WeWork 开源后捐给 LF AI & Data。**完全免费、完全开源**（Apache 2.0）。

**做什么**：接收 OpenLineage 事件（HTTP POST），自动渲染成血缘 DAG。

**核心功能**：

| 功能 | 描述 | 给谁用 |
|---|---|---|
| **血缘 DAG 可视化** | Job → Dataset → Job → Dataset 的图，可点击展开 | 工程师、PM、客户 |
| **版本时间线** | 每个 dataset 的 version 历史 | 运营查回滚 |
| **Run 历史** | 每个 job 的所有 run（状态/时长/失败原因）| 算法工程师 |
| **数据集详情** | facets 全部展开（schema/source/quality）| 数据治理 |
| **搜索** | 按 namespace + name 查 dataset/job | 所有人 |
| **REST API** | 反向查询血缘（你们前端可调）| 内部 service |

**部署成本极低**：

```bash
# 最简部署：docker-compose 三个服务
git clone https://github.com/MarquezProject/marquez
cd marquez
./docker/up.sh

# UI: http://localhost:3000
# API: http://localhost:5000
```

生产部署：

| 组件 | 推荐方式 |
|---|---|
| `marquez-api` | Helm chart 部署到现有 K8s |
| `marquez-web` | 同上 / 或单独 nginx |
| Postgres backend | 单独一个 PG instance（小规模）/ 你们现有 PG 加 schema |
| 资源占用 | 单实例 1 vCPU + 2 GB RAM 足够（中等规模 1k events/min） |

**部署产生的 effect**：

- **替代自研 UI**：自己造血缘可视化 = UI 工程师 2-3 月起步，造完还要长期维护
- **客户演示直接用**：把 Marquez UI 当 demo，不用截 SQL 结果
- **跨团队共享**：数据治理 / 数据湖 / 算法团队都能进 UI 看，不用都问你
- **审计闭环**：合规审计员看「这个客户文件来自哪个 mcap」一目了然

---

## 3. 整合架构

### 3.1 数据流

```
┌──────────────────────────────────────┐
│ AssetWriter（你们已有）              │
│ ├─ INSERT assets                     │
│ ├─ INSERT asset_relations            │
│ ├─ INSERT algo_runs                  │
│ └─ INSERT asset_events               │
└──────────┬──────────────────────────┘
           │ 同事务（Outbox 模式）
           ▼
┌──────────────────────────────────────┐
│ asset_events（PG 表，事实源）        │
│ - 已有，无需改                        │
└──────────┬──────────────────────────┘
           │ CDC（Debezium / 现网 Pub/Sub Outbox）
           ▼
┌──────────────────────────────────────┐
│ Pub/Sub Topic: asset_events_stream   │
│ - 已有                                │
└──────────┬──────────────────────────┘
           │ subscribe
           ▼
┌──────────────────────────────────────┐
│ OpenLineage Emitter Service（新）    │ ★ 唯一新写代码
│ - 订阅 asset_events                   │
│ - asset_event → OpenLineage RunEvent │
│ - 处理 START / COMPLETE / FAIL       │
│ - 拼 inputs / outputs（查 PG）       │
│ - POST /api/v1/lineage to Marquez    │
└──────────┬──────────────────────────┘
           │ HTTP POST OpenLineage events
           ▼
┌──────────────────────────────────────┐
│ Marquez（开源现成）                  │
│ ├─ marquez-api（事件接收 + 存储）    │
│ ├─ marquez-postgres（独立 backend）  │
│ └─ marquez-web（UI）                 │
└──────────────────────────────────────┘
           ▲
           │ 浏览
           │
       工程师 / PM / 客户 / 审计员
```

### 3.2 关键设计原则

| 原则 | 说明 |
|---|---|
| **事实源在 PG**，Marquez 只做投影 | Marquez 挂了不影响业务，可重建 |
| **异步投递**（Outbox CDC）| 不在 AssetWriter 主事务里 POST Marquez，避免拖慢主流程 |
| **at-least-once + 幂等** | OpenLineage 事件带 `runId` + `eventTime`，Marquez 端去重 |
| **失败可重放** | Pub/Sub 消息保留 7 天，Marquez 重启后从头消费即可重建 |
| **schema agility** | 业务字段塞 OpenLineage `facet`，不破坏标准 schema |

### 3.3 字段映射表（emitter 实现核心）

| 你们字段 / 表 | OpenLineage 字段 | 备注 |
|---|---|---|
| `algo_runs.run_id` | `run.runId` | 1:1 |
| `algo_runs.algo_name` | `job.name` | 1:1 |
| `algo_runs.algo_version` | `job.facets.documentation.description` 或自定义 facet | OL 没原生 algo version |
| `algo_runs.status` | `eventType`（COMPLETE/FAIL/ABORT）| 状态映射 |
| `algo_runs.started_at` / `completed_at` | `eventTime`（每个事件一个）| START 用 started_at，COMPLETE 用 completed_at |
| `algo_runs.error_message` | `run.facets.errorMessage.message` | |
| `assets.asset_id` | `inputs[].name` / `outputs[].name` | namespace 前缀 `cyber-databrew.assets` |
| `assets.logical_asset_id` | dataset 自定义 facet `logicalAssetId` | OL 没原生 logical 概念 |
| `assets.revision` | `outputs[].facets.version.datasetVersion` | |
| `assets.storage_uri` | `outputs[].facets.dataSource.uri` | |
| `assets.start/end_timestamp_ns` | dataset facet `nominalTime`（自定义）| |
| `asset_relations.metadata.run_id` | 边推 input/output 关系时用 | run_id 关联 |
| `asset_events.actor` | `run.facets.processing_engine.name` 或自定义 | |
| `asset_metrics` rows | `outputs[].facets.dataQualityMetrics` | 数值指标 |

**自定义 facet** 注册到 OpenLineage（仅声明，无需改协议）：

```json
{
  "logicalAssetId": {
    "_producer": "https://github.com/yourcorp/cyber-databrew",
    "_schemaURL": "https://yourcorp.com/openlineage-facets/logicalAssetId.json",
    "logicalAssetId": "aaa11111"
  }
}
```

---

## 4. 落地路线图

### Phase 1：POC（1 周，1 工程师）

**目标**：跑通最小链路，验证血缘图能渲染。

| Day | 任务 |
|---|---|
| 1 | docker-compose 起 Marquez 本地实例 |
| 2-3 | 写 emitter 服务（Go/Python），订阅 `algo_run_completed` event，转 OpenLineage COMPLETE 事件 |
| 4 | 跑一次真实 algo run，验证 Marquez UI 出血缘图 |
| 5 | 补 START / FAIL 事件类型 |

**交付**：3 张图（segment_A → hand_track 多版本 → clipX 版本链），UI 能点能看。

### Phase 2：Facet 完善（2 周）

**目标**：把所有有用的元数据塞进 facet，让 UI 看到的资产信息完整。

| 周 | 任务 |
|---|---|
| 1 | `dataSource` / `version` / `nominalTime` facet（用 storage_uri / revision / 时间窗）|
| 1 | `dataQualityMetrics`（关联 `asset_metrics`）|
| 2 | `parent` run facet（如果 algo run 有 pipeline 嵌套）|
| 2 | 自定义 `logicalAssetId` facet |
| 2 | 失败回放测试（Marquez 重启后 Pub/Sub 重消费）|

**交付**：UI 上每个 dataset 点开能看到完整业务信息。

### Phase 3：生产部署（1 个月）

**目标**：Marquez 上 K8s，emitter 长期运行，监控告警。

| 周 | 任务 |
|---|---|
| 1 | Helm chart 部署 marquez-api / web 到 dev 集群 |
| 2 | emitter 接入现网 Pub/Sub 订阅 |
| 2 | Cloud Monitoring 告警（事件延迟 / emitter 失败率）|
| 3 | 灰度 prod（1% asset_events 投递）|
| 4 | 全量切换，老的临时血缘 SQL 工具下线 |

**交付**：prod 可访问 Marquez UI，覆盖 100% 血缘事件。

### Phase 4：可选扩展（rev.13+）

| 项 | 说明 |
|---|---|
| **DataHub 实例对接** | 加一个 DataHub emitter（参考 §2.2 mapper），把元数据推 DataHub |
| **客户血缘 API** | 包装 Marquez REST API 给客户用（带租户隔离）|
| **自定义 UI 嵌入** | iframe 嵌 Marquez 进运营后台 |
| **column lineage** | 给 derived_asset parquet schema 补字段级血缘 |

---

## 5. 解决的问题 / 带来的效果

### 5.1 解决的问题（Before → After）

| 问题 | Before | After |
|---|---|---|
| 「这个 clip 的来源链路」 | 写 SQL JOIN 三张表 | Marquez UI 点开看图 |
| 「hand_track@2.0 跑出过啥」 | 写 SQL | UI 搜 job hand_track，看 run 历史 |
| 「客户演示血缘」 | 截图 SQL 结果（差）| 直接给 Marquez UI 链接 |
| 「数据治理团队要看元数据」 | 没接口 | DataHub 实例对接（Phase 4）|
| 「外部数据湖团队消费血缘」 | 没标准格式 | 订阅 OpenLineage 事件流 |
| 「血缘 UI 自研成本」 | UI 工程师 2-3 月 | 0（用 Marquez） |

### 5.2 量化效果（保守估计）

| 指标 | 估计 |
|---|---|
| 血缘可视化 UI 工程量节省 | **2-3 工程师月** |
| 血缘查询响应时间（人工 SQL → UI 点击）| **~10 分钟 → ~10 秒** |
| 客户血缘演示准备时间 | **~30 分钟（截图 + 拼接）→ ~0**（直接给链接）|
| 跨团队元数据对接成本 | 每个团队各自适配 → 一次接入 OL/DataHub 标准 |
| 合规审计追溯响应 | SQL 截图 → 审计员自己进 UI 查 |

### 5.3 不能解决的问题（诚实声明）

| 不解决 | 原因 |
|---|---|
| **PG 性能问题** | Marquez 是投影，PG 慢它也慢 |
| **业务逻辑问题** | 比如版本切换原子性，那是 PG 层 partial unique 的事 |
| **delete 场景** | OpenLineage 没 delete 语义，dataset 删了 Marquez 残留（可接受）|
| **行级权限** | Marquez 不做细粒度 ACL（如客户只能看自己的血缘）；Phase 4 自研 API 包一层 |
| **强一致性** | Marquez 是异步投影，1-3s lag 必然存在 |

---

## 6. 风险 / 需要决策的问题

| 风险 | 缓解 |
|---|---|
| Marquez 维护活跃度 | LF AI & Data 项目，社区活跃；最差自己 fork（Apache 2.0）|
| 自定义 facet 失控 | 列一份 facet registry doc，统一审 |
| Pub/Sub 消息洪峰打垮 emitter | emitter 加批量 + 限速；最差降级到只投 COMPLETE 事件 |
| Marquez UI 不符合品牌 | 早期不嵌入产品，内部用；后期再决定要不要 iframe / 自研 |
| 投递重复 / 丢失 | Pub/Sub at-least-once + Marquez 端 runId 去重；定期对账（PG count vs Marquez count）|

**需要业务决策**：

1. **是否对外暴露 Marquez UI 给客户？**（涉及租户隔离，影响 Phase 4 工程量）
2. **是否要对接外部 DataHub 实例？**（影响 Phase 4 优先级）
3. **Marquez 部署在哪个 GCP project？**（涉及网络/IAM）
4. **是否启用 OpenLineage column lineage？**（derived_asset parquet schema 才有意义）

---

## 7. 参考资料

- [OpenLineage 官网](https://openlineage.io/)
- [OpenLineage Spec（schema 全集）](https://openlineage.io/spec/2-0-2/OpenLineage.json)
- [Marquez GitHub](https://github.com/MarquezProject/marquez)
- [Marquez 架构文档](https://marquezproject.ai/docs/architecture)
- [DataHub Metadata Model](https://datahubproject.io/docs/metadata-modeling/extending-the-metadata-model/)
- [DataHub SystemMetadata.pdl](https://github.com/datahub-project/datahub/blob/master/metadata-models/src/main/pegasus/com/linkedin/mxe/SystemMetadata.pdl)
- [OpenLineage Airflow integration](https://openlineage.io/docs/integrations/airflow/)
- [OpenLineage Spark integration](https://openlineage.io/docs/integrations/spark/)

---

## 8. 一句话总结

> **不要自己造血缘 UI**。OpenLineage 是协议、Marquez 是 UI、DataHub SystemMetadata 是审计字段命名规范——三者都开源、Apache 2.0、无 vendor lock-in。**加一个 emitter 服务把你们 PG 里已有的事件转成标准格式，1-2 周 POC，2 个月生产，节省 2-3 工程师月**。
