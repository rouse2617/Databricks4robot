# 02 · 架构全景

## 1. 目标架构(对齐"三级能力"参考图)

用户输入的目标架构分为三层,`data4cyber` 的实现与之一一对应:

| 参考图层 | 参考图能力 | `data4cyber` 实现 |
| --- | --- | --- |
| **最终目标** | 资产、数据统一视图 | Bigtable 宽表 + Web UI 统一入口 |
| | 集群管理、资产数据流转 | 独立的 `cluster-service` / `transfer-service` |
| **核心能力** | 文件级记录 | `cf:file` 列族 |
| | 多版本信息 | ⭐ **Bigtable cell-level 原生多版本** |
| | 资产数据强一致性 | ⭐ **Bigtable 单行原子性** |
| | 数据追溯 | `cf:lineage` + `cf:event` |
| | 操作审计 | `cf:event` + Cloud Audit Logs + BigQuery 审计表 |
| | 统一元数据底座 | ⭐ **Bigtable 宽表 = 底座** |
| **三级能力(资产平台)** | 元数据统一底座 | Bigtable 主表 |
| | 元数据注册机制 | Schema Registry(Git `schemas/fields.yaml` + API) |
| | 数据链路 | Dagster Assets + Pub/Sub(MCE / MCL 事件) |
| **三级能力(数据管理)** | 集群 / 预览 / 配置 | GKE + Foxglove 嵌入 + Secret Manager |
| **三级能力(集群管理)** | 任务列表 / 优先级 / 审计 | Dagster UI + RBAC + Cloud Audit Logs |
| **三级能力(存储平台)** | 元数据外置 / SDK 适配 | Bigtable 外置 / `grace-sdk` |
| | 多协议互通 | REST(FastAPI)+ gRPC + GraphQL(Phase 2) |
| | 实时同步机制 | Pub/Sub + Dataflow CDC |
| | 统一命名空间 | URN 方案(`urn:grace:asset:<uuid>`) |
| | 归档 / 回档 / 云间同步 | GCS Lifecycle + Storage Transfer Service + Nearline / Coldline |

---

## 2. 总体架构图

> **两张图各自承担不同相位**:
> - **本节主图 = 目标态(Phase 1+)**:Bigtable + CDC + MCE/MCL,对应 [`diagrams/architecture.mmd`](../diagrams/architecture.mmd)
> - **Phase 0 运行态**:PG 宽表 + Outbox + MCL,见 [`diagrams/phase0-runtime.mmd`](../diagrams/phase0-runtime.mmd) 以及 [08-roadmap §Phase 0](08-roadmap.md)
>
> 写入契约(唯一权威写入者、MCL 广播、URN↔UUID 边界)以 [ADR-007](adr/ADR-007-write-path-and-event-contract.md) 为准。

```
┌─────────────────────────────────────────────────────────────────────┐
│                       data4cyber Web UI (React)                     │
│  自研:搜索 / 详情 / Lineage / QA 队列 / Dashboard                  │
│  嵌入:Foxglove / Dagster / Label Studio / Superset(iframe)        │
└───────────────────────────────┬─────────────────────────────────────┘
                                │   REST / GraphQL(浏览器唯一入口)
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│    data4cyber-api (BFF) — 鉴权 / 聚合 / URN↔UUID 转换              │
└───────────────────────────────┬─────────────────────────────────────┘
                                │   gRPC(内部 UUID)
     ┌──────────────────────────┤
     │                          │     ┌─────────────────────────────┐
     │                          │     │  grace-sdk (Python) / CLI   │
     │                          │     │  Ray worker — 直连          │
     │                          │     └──────────────┬──────────────┘
     │                          │                    │
┌────▼────────────┐   ┌─────────▼─────────┐   ┌──────▼──────────────┐
│ asset-service   │   │   mcap-gateway    │   │ cluster / transfer  │
│ (Go, 唯一写入者)│   │   (Go, streaming) │   │ service             │
│ • 状态机 + 乐观锁│  │ • Range GET        │   │ • 集群纳管          │
│ • URN 生成      │   │ • 多 part upload   │   │ • 配额 / 流控       │
│ • Outbox relay  │   │ • Summary 提取     │   │ • 归档 / 回档任务   │
└────┬────────────┘   └─────────┬─────────┘   └─────────────────────┘
     │    写+读                  │   流式 I/O
     │                           │
┌────▼───────────────────────────▼──────────────────────────────────────┐
│   ⭐ Cloud Bigtable — 统一元数据底座                                  │
│                                                                       │
│   主表 assets:row key = v0/v1 含 tenant/project 打散前缀 + asset_uuid│
│   9 个列族:                                                          │
│     cf:core    cf:time    cf:tag     cf:file                          │
│     cf:qa      cf:event   cf:lineage cf:algo   cf:emb                 │
│                                                                       │
│   二级索引:                                                          │
│     asset_locator       (asset_uuid → main_row_key,点查入口)         │
│     assets_by_project   (按 project 扫最近,32 路并行 prefix scan)    │
│                                                                       │
│   内置能力:多版本 / 单行原子 / 稀疏列 / 10ms 点查 / 横向扩展         │
│                                                                       │
│   Phase 0 替代品:Cloud SQL Postgres(8 JSONB 伪列族 + asset_events) │
└────┬──────────────────────────────────────────────────────────────────┘
     │
     │  Outbox → Pub/Sub(grace-mcl) 权威事件流
     │  (Phase 1+ 另有 grace-mce 作异步/批量入口)
     │
     ├──▶ OpenSearch          (Tag / 全文检索,分面过滤)
     ├──▶ Vertex AI Vector    (多模态 / embedding 检索,Phase 2)
     ├──▶ BigQuery            (OLAP / 报表 / 审计仓)
     └──▶ Dagster Assets      (数据链路编排:触发 sam2_v3 等算法)
                 │
                 └──▶ Ray on GKE — 统一算力层(PipesRayJobClient)

─────────────────────────────────────────────────────────────────────
物理存储:
  GCS:
    • grace-raw-mcap/       原始 MCAP(客户回传)
    • grace-derived/        派生产物(sam2_v3 等输出)
    • grace-annotation/     标注文件
  Lifecycle:
    30 天 → Nearline,90 天 → Coldline,一年 → Archive
─────────────────────────────────────────────────────────────────────
运行时:
  GKE Autopilot(集群)+ Artifact Registry(镜像)+ Secret Manager
  Cloud Monitoring / Logging / Trace(可观测)
  IAM + Workload Identity(身份)
```

---

## 3. 分层职责

### Layer 1 · 用户接入层

| 组件 | 技术 | 职责 | 谁用 |
| --- | --- | --- | --- |
| **Web UI** | React + TS + Ant Design | 搜索 / 详情 / Lineage / Dashboard | 所有人 |
| **grace-sdk** | Python | 资产 CRUD / search / stream read | 算法用户 |
| **CLI** | Python(typer) | 运维 / 手动 ingest / 诊断 | Ops |

### Layer 2 · 服务层

| 组件 | 技术 | 职责 |
| --- | --- | --- |
| **asset-service** | Go(Gin / Echo) | 资产 CRUD、状态机、权限、审计、URN 生成 |
| **mcap-gateway** | Go | MCAP 流式读/写、Range GET、Multipart Upload、Summary 提取 |
| **search-service** | Go / Python | 代理 OpenSearch / Vector 查询,回流 asset_id 到 asset-service |
| **cluster-service** | Go | 多集群纳管、配额、资源信息 |
| **transfer-service** | Go | 归档 / 回档 / 云间同步任务 |

### Layer 3 · 存储层

| 组件 | GCP 产品 | 职责 |
| --- | --- | --- |
| **元数据底座** | Bigtable(Phase 1+)/ Cloud SQL PG(Phase 0) | 宽表主数据 |
| **文件存储** | GCS | MCAP 原始 + 派生 + 标注 |
| **搜索索引** | OpenSearch on GKE(或 Elasticsearch Serverless) | Tag / 全文 |
| **向量索引** | Vertex AI Vector Search | 多模态(Phase 2) |
| **OLAP / 审计** | BigQuery(含 External Table 读 Bigtable) | 分析、审计、报表 |
| **事件总线** | Pub/Sub | MCE / MCL / 生命周期事件 |
| **编排** | Dagster on GKE | Asset graph、job 调度、sensor |
| **算力** | Ray on GKE | 分布式算法执行 |

### Layer 4 · 基础设施

| 组件 | GCP 产品 |
| --- | --- |
| 容器 | GKE Autopilot |
| 镜像 | Artifact Registry |
| 密钥 | Secret Manager |
| 身份 | IAM + Workload Identity |
| 可观测 | Cloud Logging / Monitoring / Trace |
| IaC | Terraform |

---

## 4. 关键数据流

### 4.1 MCAP 接入流(上行)

```
用户 / 机器人
    │
    │  1. multipart upload(大文件断点续传)
    ▼
mcap-gateway ──▶ GCS(grace-raw-mcap/)
    │
    │  2. 发 "mcap.uploaded" 事件
    ▼
Pub/Sub ──▶ Dagster Sensor ──▶ 触发 ingestion Asset
                                    │
                                    │  3. 跑 indexer-worker:
                                    │     • 读 MCAP footer / summary(Range GET,不下载全文件)
                                    │     • 提取 topic 列表、时间范围
                                    │     • 生成初始 asset(pending 状态)
                                    ▼
                              asset-service ──▶ Bigtable(写 cf:core + cf:time + cf:file)
                                    │
                                    └──▶ MCL 事件 → CDC → OpenSearch
```

### 4.2 QA / Tag 流(人工,写路径见 [ADR-007](adr/ADR-007-write-path-and-event-contract.md))

```
Web UI ──▶ BFF (data4cyber-api) ──gRPC──▶ asset-service ──▶ Bigtable
  │          │  URN→UUID 转换                   │
  │          │  鉴权 / 聚合                     │  CheckAndMutate(乐观锁)
  │          │                                  │  写 cf:qa + cf:tag + cf:event
  │                                             │
  │   状态:approved                     Outbox → MCL 事件
  │                                             │
  ▼                                             ▼
Foxglove(嵌入预览)                  Pub/Sub(grace-mcl) ──▶ Dagster Sensor
                                                              │
                                                              └─▶ 触发下游算法(如 sam2_v3)
```

> 约束:浏览器**不直连** asset-service;SDK / CLI / Ray worker 可直连。

### 4.3 算法派生流(Dagster + Ray)

```
Dagster Sensor(监听 cf:qa = approved)
    │
    ▼
@asset sam2_v3_run(partition=batch_id)
    │
    │  PipesRayJobClient.run(entrypoint=run_mock.py ...)
    ▼
Ray Cluster on GKE
    │
    │  并行处理若干 asset:
    │  for asset in batch:
    │      read MCAP via mcap-gateway(stream)
    │      run SAM2 v3 inference
    │      write output → GCS(grace-derived/)
    │      call asset-service API:
    │          assets.files.create(kind="sam2_v3", version="3.1", uri=...)
    ▼
Pipes materialization → Dagster(lineage 记录)
    │
    └─▶ 上游 Bigtable cf:file / cf:algo 被更新
        MCL → 下游评估 asset 触发
```

### 4.4 检索流(下行)

```
用户 / SDK
    │
    │  tag=urban AND weather=rainy AND duration>30s
    ▼
asset-service ──▶ search-service ──▶ OpenSearch
                                        │
                                        │  命中 5,000 个 asset_id
                                        ▼
                       asset-service ──▶ Bigtable multi-get(batch read)
                                        │
                                        ▼
                                     SDK / UI
                                        │
                                        ▼
                                  用户拿到结构化 asset 列表
                                  (含 files / tags / qa / lineage)
```

---

## 5. 关键非功能指标(NFR)

| 指标 | 目标 | 依据 |
| --- | --- | --- |
| **API 可用性** | 99.9%(月) | GKE Autopilot + 多可用区 |
| **点查延迟** | P99 < 50ms | Bigtable SSD 单行读 ~5ms |
| **搜索延迟** | P99 < 500ms | OpenSearch 3-5 节点 |
| **MCAP 流式读启动** | < 1s 出第一字节 | Range GET,无需全文件下载 |
| **数据一致性** | 最终一致(秒级) | Bigtable 写 → CDC → 索引 ≤ 5s |
| **Asset 规模支持** | 1 亿 + | Bigtable 原生支持 |
| **Tag 条目规模** | 50 亿 + | 稀疏列 + OpenSearch |
| **单次 Multi-Get** | 1,000 asset / 100ms | Bigtable batch read |
| **数据持久性** | 11 个 9 | GCS + Bigtable 都达到 |

---

## 6. 设计取舍(Trade-offs)

| 选择 | 替代方案 | 为什么选这个 |
| --- | --- | --- |
| **Bigtable 宽表** | Spanner / Postgres / CockroachDB | 百亿行,稀疏列,高吞吐点查最优;牺牲事务/JOIN(我们不需要) |
| **OpenSearch 索引 Tag** | Postgres GIN / ClickHouse | 海量 Tag 组合过滤 + 全文,OpenSearch 是业界标准 |
| **Pub/Sub + Dataflow CDC** | Kafka + Flink | GCP 原生,运维成本低;性能够用 |
| **Dagster 编排** | Airflow / Prefect | Asset-first,与我们"资产即一等公民"天然契合 |
| **不 fork DataHub/OpenMetadata** | 全自研 / 完全 fork | Fork 维护成本高;全自研浪费;最优解=偷设计 |
| **UI 自研 + iframe** | 完全自研 / 完全外包 | MCAP 播放器自研极贵,Foxglove 现成;核心资产 UI 必须自研可控 |

---

## 7. 演进路径(简版)

详见 [08-roadmap.md](08-roadmap.md)。

```
Phase 0(0-3 月)   PG 宽行模拟 Bigtable → 打通 E2E,能跑通一个 MCAP
Phase 1(3-6 月)   切 Bigtable + OpenSearch → 真正支持百万 asset + 搜索
Phase 2(6 月+)    接 Vertex Vector + BigQuery → 多模态 + 分析
```

---

## 8. 依赖与边界

### 上游(我们不负责但依赖)

- GCP 各产品 SLA / 稳定性
- Dagster OSS 版本兼容
- Foxglove App(公共服务)或 Lichtblick(自托管兜底)

### 下游(我们的消费者)

- `cyber-grace`(业务系统,Phase 1 逐步迁移资产到 data4cyber)
- 算法训练 pipeline(只通过 SDK 访问)
- 客户交付 pipeline(只通过 export API 访问)

### 边界(明确划清)

- 我们**不做** Dagster Pipeline 逻辑定义,算法用户自己写
- 我们**不做**模型 serving / inference,只管数据
- 我们**不做**用户鉴权源,接 SSO(IAM / OIDC)

---

## 9. 下一步

- 宽表详细设计 → [03-data-model-wide-table.md](03-data-model-wide-table.md)
- MCAP 特殊处理 → [04-mcap-and-segment.md](04-mcap-and-segment.md)
- UI 设计 → [05-ui-strategy.md](05-ui-strategy.md)
