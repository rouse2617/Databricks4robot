# 06 · DataHub / OpenMetadata 偷师清单

## 1. 策略总览

> **不 fork,只偷师**。这两家是业界最好的"元数据系统"开源实现,**设计模式直接抄**,代码按需参考。

| 场景 | 选择 |
| --- | --- |
| 全量 fork 作为主平台 | ❌ 数据模型错位,维护噩梦(详见 [ADR-004](adr/ADR-004-no-fork-datahub-openmetadata.md)) |
| 抄核心设计模式 | ✅ **强烈推荐**,本文重点 |
| 直接用某些组件(如 ingestion framework) | ✅ 择优 |
| Phase 2+ 作为 sidecar 集成 | ⭐ 如果生态到了再看 |

---

## 2. **必抄的 10 个核心模式**

### ⭐⭐⭐⭐⭐ 模式 1:**URN 统一标识**(来源:DataHub)

**模式**:所有 entity 用 URN 表达。
**data4cyber 落地**:

```
urn:grace:asset:<uuid>
urn:grace:mcap:gs://grace-raw-mcap/.../abc.mcap
urn:grace:segment:<asset_id>:<sub_segment_id>
urn:grace:tag:Scene.urban
urn:grace:user:alice@company.com
urn:grace:algo:sam2@v3.1.2
urn:grace:dataset:train_2026q1
urn:grace:run:dagster-run-abc123
```

**好处**:全局可解析、可审计、可拼 URL。
**参考**:[ADR-006](adr/ADR-006-urn-identity.md)

---

### ⭐⭐⭐⭐⭐ 模式 2:**Aspect-Oriented Entity**(来源:DataHub)

**模式**:一个 entity = 多个独立 aspect,各自版本化。
**data4cyber 落地**:Bigtable **9 个列族**就是 9 个 aspect(Phase 0 在 PG 上用 8 JSONB + `asset_events` 表模拟),详见 [03-data-model-wide-table.md](03-data-model-wide-table.md)。

```
Asset (entity)
  ├── cf:core      (aspect 1)
  ├── cf:time      (aspect 2)
  ├── cf:tag       (aspect 3)
  ├── cf:file      (aspect 4)
  ├── cf:qa        (aspect 5)
  ├── cf:event     (aspect 6)
  ├── cf:lineage   (aspect 7)
  ├── cf:algo      (aspect 8)
  └── cf:emb       (aspect 9)
```

**好处**:加新 aspect 不动旧代码;每个 aspect 独立 CDC 到索引。

---

### ⭐⭐⭐⭐⭐ 模式 3:**MCE / MCL 事件流**(来源:DataHub)

**模式**:
- **MCE**(Metadata Change Event):上游应用"我要改 X"的**意图**
- **MCL**(Metadata Change Log):后端落库成功后广播"X 真的改了"

**data4cyber 落地(见 [ADR-007](adr/ADR-007-write-path-and-event-contract.md) 权威契约)**:

**Phase 0(冻结)**:简化到只跑 MCL
```
SDK / BFF / worker ──gRPC──▶ asset-service ──1. sync write──▶ PG
                                             │
                                             └──2. Outbox relay──▶ Pub/Sub(grace-mcl)
                                                                         │
                                          ┌──────────────────────────────┼───────────────┐
                                          ▼                              ▼               ▼
                                    OpenSearch                     BigQuery        Dagster Sensor
```

**Phase 1+(扩展)**:引入 `grace-mce` 做异步/批量入口
```
批量 ingest / 外部系统 ──MCE──▶ Pub/Sub(grace-mce)──▶ asset-service ──▶ Bigtable
                                                              │
                                                              └──MCL──▶ grace-mcl ──▶ 订阅者
```

**好处**:上下游解耦、审计、回放、增量订阅。

**落地位置**:
- `grace-mcl` topic:**Phase 0 起**启用,服务端成功广播
- `grace-mce` topic:**Phase 1** 启用,仅用于异步/批量意图
- 同步写入(SDK/UI)**不经过 MCE**,直接 gRPC 到 asset-service

---

### ⭐⭐⭐⭐ 模式 4:**Ingestion Framework**(来源:OpenMetadata,**可直接用**)

**模式**:Source → Processor → Sink 三段式插件。

**data4cyber 落地**:MCAP ingestion pipeline 用这个骨架:

```python
from grace.ingestion import Source, Processor, Sink, Workflow

class McapSource(Source[McapCandidate]):
    """扫 GCS 找新 MCAP"""
    def _iter(self):
        for blob in list_new_blobs(self.last_ts):
            yield McapCandidate(uri=blob.uri)

class McapSummaryProcessor(Processor):
    """Range GET 解 MCAP footer,提取 topic / time"""
    def run(self, candidate):
        summary = extract_summary(candidate.uri)
        return EnrichedCandidate(candidate, summary)

class AutoTagProcessor(Processor):
    """跑自动分类器,打初始 tag"""
    ...

class AssetServiceSink(Sink[EnrichedCandidate]):
    """调 asset-service gRPC → 写 PG/Bigtable → Outbox 发 MCL
    注意:Sink 不直接写底座、不直接发事件,统一走 asset-service(见 ADR-007)。"""
    ...

Workflow([McapSource(), McapSummaryProcessor(), AutoTagProcessor(), AssetServiceSink()]).run()
```

> 直接抄 OpenMetadata 的 `openmetadata-ingestion/src/metadata/workflow/*.py`,<500 行,核心部分。

---

### ⭐⭐⭐⭐⭐ 模式 5:**Stateful Ingestion**(来源:OpenMetadata)

**模式**:每次 ingestion 记录 checkpoint,下次从增量开始。

```python
class StatefulSource(Source):
    def prepare(self):
        self.last_ts = state_store.get("last_ingest_ts", default=0)
    
    def _iter(self):
        for item in self.scan_since(self.last_ts):
            yield item
        state_store.put("last_ingest_ts", self.now_ts)
```

**data4cyber 落地**:扫 GCS 只处理**上次以来的新对象**,避免全量扫。

---

### ⭐⭐⭐⭐⭐ 模式 6:**Classification + Tag + Glossary 三层体系**(来源:OpenMetadata)

**模式**:

| 层 | 作用 |
| --- | --- |
| **Classification** | 一级分类(Scene / Weather / Quality...) |
| **Tag** | 具体标签(Scene.urban / Weather.rainy...) |
| **Glossary Term** | 业务术语 + 正式定义(有效 segment = ...) |

**data4cyber 落地**:见 [schemas/fields.yaml](../schemas/fields.yaml)。

```yaml
classifications:
  Scene:
    description: 场景类型
    tags:
      - name: urban
        description: 城市场景
      - name: highway
      - name: rural

glossary:
  - term: valid_segment
    definition: 经过 QA approve 且非空的视频片段
    synonyms: [有效片段, effective segment]
```

---

### ⭐⭐⭐⭐ 模式 7:**Lineage Graph**(来源:两家都有)

**模式**:Edge-based lineage,每条边带 operation。

```
(upstream_urn, downstream_urn, operation, metadata, created_at)
```

**data4cyber 落地**:

```
urn:grace:mcap:abc.mcap
  ──(SEGMENT_EXTRACT, by=human:bob)──▶ urn:grace:asset:uuid1

urn:grace:asset:uuid1
  ──(DERIVE, algo=sam2@v3.1.2, run=run-abc)──▶ urn:grace:mcap:abc_mask.mcap

urn:grace:asset:[uuid1, uuid2, ...]
  ──(BUNDLE)──▶ urn:grace:dataset:train_2026q1
```

持久化:Bigtable `cf:lineage` + BigQuery lineage_edges 表(用于 OLAP)。

---

### ⭐⭐⭐⭐ 模式 8:**Data Quality TestSuite**(来源:OpenMetadata,**可直接 pip install**)

**模式**:TestSuite → TestCase → TestDefinition 三层抽象。

```python
from grace.quality import TestCase, TestSuite

suite = TestSuite("asset_integrity")

suite.add(TestCase(
    name="duration_positive",
    assertion="cf_time.duration_ns > 0",
    severity="error"
))

suite.add(TestCase(
    name="tag_scene_required",
    assertion="cf_tag has key starting with 'Scene.'",
    severity="warning"
))

suite.run(on=asset_batch)
```

直接:`pip install openmetadata-ingestion[data-quality]`,白嫖框架。

---

### ⭐⭐⭐⭐ 模式 9:**Activity Feed**(来源:OpenMetadata)

**模式**:每个 entity 有消息流(谁什么时候做了什么),@ 提及 + 评论。

**data4cyber 落地**:
- 底层:`cf:event` + `asset_events` 表
- UI:资产详情页 Activity Tab(直接抄 OpenMetadata 的 UI 风格)

---

### ⭐⭐⭐ 模式 10:**Actions / Event Subscription**(来源:DataHub Actions)

**模式**:订阅 MCL,满足 filter 就触发 action。

```yaml
# subscriptions/auto_trigger_sam2.yaml
name: auto_trigger_sam2_on_qa_approved
trigger:
  types: [asset.qa.state_changed]
  filter:
    new_state: approved
action:
  type: dagster_launch
  config:
    job: sam2_v3_process
    asset_key: sam2_output
    partition: "{{ asset.id }}"
```

**data4cyber 落地**:`subscription-service` 订阅 `grace-mcl` topic,按规则分发。

---

## 3. **值得参考但 Phase 0 不做的模式**

| 模式 | 来源 | 何时做 |
| --- | --- | --- |
| Domain / DataProduct(Data Mesh) | 两家 | Phase 2,多租户隔离时 |
| RBAC / Policy 引擎(SpEL) | OpenMetadata | Phase 2,多团队时 |
| PII 自动识别 | 两家插件 | Phase 2,合规要求提升时 |
| Data Contract | DataHub | Phase 3,外部交付契约化时 |
| MLModel + MLFeature entity | DataHub | 如果要做 Feature Store |
| Business Glossary 富文本编辑 | OpenMetadata | Phase 2 |

---

## 4. **明确不抄的模式**

| 模式 | 为什么不抄 |
| --- | --- |
| 50+ 数据源 connector | 我们只有 MCAP+GCS |
| SQL 血缘自动解析 | 我们不是 SQL 场景 |
| 列级血缘 | MCAP 无列概念 |
| Neo4j 图数据库 | Phase 0 PG 表 `lineage_edges` 够 |
| Kafka 事件总线 | GCP 原生用 Pub/Sub |
| 全套 React UI | 抄设计,不 fork 代码 |

---

## 5. 源码阅读推荐路径(2 周)

### Week 1:OpenMetadata(Python / REST,门槛低)

| 路径 | 学什么 |
| --- | --- |
| `openmetadata-spec/src/main/resources/json/schema/entity/data/*.json` | Schema-first entity 设计 |
| `ingestion/src/metadata/workflow/*.py` | Ingestion 框架(**可直接抄**) |
| `ingestion/src/metadata/data_quality/` | DQ 框架(**可直接用**) |
| `openmetadata-service/src/main/java/org/openmetadata/service/resources/DatasetResource.java` | REST API 设计 |

### Week 2:DataHub(概念深,值得看高阶)

| 路径 | 学什么 |
| --- | --- |
| `metadata-models/src/main/pegasus/com/linkedin/dataset/*.pdl` | Aspect-oriented entity |
| `metadata-events/mxe-schemas/` | MCE / MCL 事件定义 |
| `metadata-ingestion/src/datahub/ingestion/source/` | 多源抽象 |
| `datahub-actions/src/datahub_actions/` | Actions 框架 |

---

## 6. 具体"偷师产出物"清单

完成这些偷师,你会产出如下文件(Phase 0):

| 产出 | 来源 |
| --- | --- |
| `schemas/fields.yaml`(Classifications + Tags + Glossary) | OpenMetadata 模式 |
| `schemas/urn-spec.md`(URN 规范) | DataHub 模式 |
| `schemas/mce-mcl-schema.proto`(事件 schema) | DataHub 模式 |
| `services/ingestion-framework/`(Source/Processor/Sink) | OpenMetadata 代码 |
| `services/quality-runner/`(DQ 测试引擎) | OpenMetadata 代码 |
| `ui/components/ActivityFeed/`(活动流组件) | OpenMetadata UI |
| `ui/components/LineageGraph/`(血缘图,用 React Flow) | DataHub UI 交互 |
| `ui/pages/SearchPage.tsx`(分面搜索) | DataHub UI |
| `services/subscription-service/`(事件订阅) | DataHub Actions |

---

## 7. 一句话总结

> **DataHub + OpenMetadata 是"设计宝典",不是"你要 fork 的地基"**。
> 花 2 周吸收 10 个核心模式(URN / Aspect / MCE-MCL / Ingestion / DQ / Lineage / Classification-Tag-Glossary / Activity / Actions / Stateful),
> 让 `data4cyber` 设计档次对齐业界一线。
> **别 fork,别集成(Phase 0 / 1),Phase 2+ 可作为 sidecar 再议**。

---

## 8. 参考链接

- DataHub: https://github.com/datahub-project/datahub
- OpenMetadata: https://github.com/open-metadata/OpenMetadata
- ADR-004(为什么不 fork)→ [adr/ADR-004-no-fork-datahub-openmetadata.md](adr/ADR-004-no-fork-datahub-openmetadata.md)

---

## 9. 来自 Databricks 生态的补充借鉴(3 条)

> DataHub/OpenMetadata 解决的是"元数据治理",Databricks 生态解决的是"Lakehouse 工程实践"。
> 虽然我们**不用 Databricks 托管产品**(详见 [ADR-008](adr/ADR-008-no-databricks-as-core-platform.md)),但下列 3 条"思想 / 对应 OSS"值得吸收。

### 9.1 Open Table Format — 用 **Iceberg** 管理派生数据集(来源:Databricks Delta Lake 思想)

**模式**:派生数据(训练集、标注快照、交付包)不要只是"堆在 GCS 目录里",用**开放表格式**承载。

**为什么选 Iceberg 而不是 Delta**:
- 中立:Apache 基金会顶级项目,Snowflake / BigQuery / Trino / Spark / DuckDB / Dremio 全支持
- BigQuery 原生支持 Iceberg 外表,零 ETL
- UC-OSS 直接支持 Iceberg REST Catalog

**data4cyber 落地**:
```
gs://grace-derived/datasets/
  train_2026q1/                  # Iceberg table
    metadata/
    data/part-*.parquet
  eval_set_scene_urban/
    ...
```

**时机**:Phase 2+(Phase 0/1 裸 Parquet / MCAP 够用)

---

### 9.2 Medallion(Bronze / Silver / Gold)命名 — 标注数据成熟度(来源:Databricks Lakehouse)

**模式**:用三层命名表达"数据的加工度"。

**data4cyber 落地**:

| 层 | 对应 | 含义 |
| --- | --- | --- |
| **Bronze** | `grace-raw-mcap` | 原始 MCAP,未 QA |
| **Silver** | `grace-derived`、Bigtable `cf:qa.state = passed` | 切段完成 + QA 通过 |
| **Gold** | `grace-materialized`、标注 v1.0 已完成 | 可对外交付 / 可用于训练 |

**好处**:跨团队沟通有统一语汇("给我一份 Gold 级的 Scene.urban 数据")。

**时机**:Phase 1,随 `cf:qa` 设计一起落

---

### 9.3 自托管实验跟踪 — **Vertex AI Experiments 优先 / MLflow OSS 可选**(来源:Databricks MLflow 思想)

**模式**:算法运行的 params / metrics / artifact / model **必须进实验库**,不许散在本地。

**data4cyber 落地**:
- **首选**:Vertex AI Experiments + Model Registry(GCP 原生,零运维)
- **可选**:MLflow OSS 自托管(GKE + Cloud SQL backend + GCS artifact),适合已有 MLflow 使用习惯的团队
- `grace-sdk` 里对两者都做统一 wrapper,算法用户写一遍:
  ```python
  with grace.run(asset_id=...) as run:
      run.log_params({...}); run.log_metrics({...}); run.log_model(...)
  ```

**时机**:Phase 1(和 SAM2 算法跑通同步)

---

### 不抄的(来自 Databricks)

| 模式 | 不抄原因 |
| --- | --- |
| Delta Lake 托管 | 用 Iceberg 替代 |
| Databricks 托管 Unity Catalog | Phase 2+ 可选 UC-OSS,不上托管 |
| Databricks Workspace / Notebook-first | 和"零代码 SDK"哲学冲突 |
| DLT / Lakeflow 声明式 Pipeline | Dagster `@asset` 已经是 |
| Databricks Vector Search | Vertex AI Vector Search 更 GCP 原生 |
| Databricks SQL / Photon | BigQuery 更原生 |
| Lakebase(serverless Postgres) | 是文档 / 关系型,不是宽列,和 Bigtable 错位 |
