# 01 · 愿景、场景与核心概念

> **阅读指南**:
> - 写路径 / 事件契约 / BFF / URN↔UUID 边界 以 [ADR-007](adr/ADR-007-write-path-and-event-contract.md) 为准
> - 主架构图 = 目标态(Phase 1+)。**Phase 0 运行态**见 [`diagrams/phase0-runtime.mmd`](../diagrams/phase0-runtime.mmd) 及 [08-roadmap](08-roadmap.md)
> - 列族口径:**Bigtable 9 CF(目标态)**;Phase 0 = PG 上 8 个 JSONB 伪列族 + `asset_events` 独立表

## 1. 我们要解决什么问题

### 1.1 当前现状(痛点)

基于对 `cyber-grace` 现有系统的分析,识别出以下核心痛点:

1. **资产定义模糊** —— 所有数据都在一个 Postgres 库里,"视频"、"处理状态"、"派生产物" 杂糅在 `grace_videos` / `grace_video_steps` 表。
2. **算法用户手工鲁代码** —— 上游产物是否就绪、下游是否可以开始,全靠用户自己写脚本轮询数据库。
3. **缺少触发机制** —— 没有"上游完成 → 自动触发下游"的事件驱动。
4. **Video 是"上帝对象"** —— `videos/handlers.go` 集中了几乎所有业务逻辑,耦合严重。
5. **无统一检索能力** —— 查一个视频的派生产物、标签、QA 状态,要跨多张表 JOIN。
6. **无多模态检索** —— 想"找相似片段"做不了。
7. **规模扩展受限** —— 规划规模:**每用户几十万条视频,合计数百万到亿级 segment**。

### 1.2 目标

建设一个**独立的资产数据平台** `data4cyber`,达成:

- ✅ **算法用户零感知底层** —— 通过 SDK `asset.get(id)` 一行代码拿到所有他需要的东西
- ✅ **资产即一等公民** —— 明确定义 `asset`、`segment`、`file` 概念
- ✅ **统一元数据底座** —— 一张宽表,`asset_id` 主键,所有 aspect 汇聚
- ✅ **事件驱动的数据链路** —— 上游产物就绪自动触发下游
- ✅ **多维检索** —— Tag 过滤、全文、向量(多模态)
- ✅ **强追溯 / 审计** —— 每次变更有迹可循
- ✅ **面向 MCAP 原生** —— 大文件 streaming、summary 索引、range read

---

## 2. 使用场景

### 2.1 典型角色

| 角色 | 主要动作 | 主要页面 |
| --- | --- | --- |
| **数据工程师 / Ops** | 接入 MCAP、运维 pipeline | Ingestion、Dagster UI |
| **QA 标注员** | 审核 segment、打 tag、写备注 | 标注工作台(Label Studio) |
| **算法用户** | 检索数据、训练模型、跑 inference | SDK + 搜索 UI |
| **产品经理** | 看数据量、质量、流转状态 | Dashboard |
| **客户**(外部) | 接收交付物 | 交付交付 API / 下载 |

### 2.2 核心用户旅程

**旅程 A — 算法用户要做新算法 v4**

```
1. 打开搜索页 → 输入 "Scene.urban AND Weather.rainy AND duration > 30s"
2. 拿到 10,000 个 asset_id 列表
3. 打开 IDE,写:
     for asset in grace.search(tags={"Scene":"urban", "Weather":"rainy"}):
         original = asset.files["raw_mcap"]       # streaming 读,不下载
         segments = asset.segments                # t_start, t_end
         # ... 跑 v4 算法
         asset.files.create("sam2_v4", uri=output_uri, version="4.0")
4. 下游看到 v4 产物就绪(事件),自动触发评估 job
```

**旅程 B — QA 标注员审核新回传**

```
1. 收到通知 "有 50 个新 asset 待 QA"
2. 打开资产管理 UI → QA 队列
3. 点击第一个 → 右侧嵌入 Foxglove 播放 MCAP 片段
4. 确认有效 → 点击 "Approve",打 tag "Scene.urban" "Vehicle.truck"
5. 状态变 approved → 自动触发自动标注算法
```

**旅程 C — 产品经理看周报**

```
1. 打开 Dashboard → "本周新接入 3,000 asset,已 QA 2,500,approved 率 84%"
2. 点击 "sam2_v3 算法" → 运行趋势图,失败率 0.2%,平均耗时 42s
3. 点击 "交付" → 交付给客户 X 的 dataset_train_2026q1 有 180k 条 asset
```

---

## 3. 核心概念(术语表)

### 3.1 三层实体模型

```
MCAP File(物理文件)
  │
  │  一个 MCAP 文件 → N 个 segment
  ▼
Asset(= Segment,视频有效片段,平台的一等公民)
  │
  │  一个 asset 关联 N 个文件
  ▼
File(原始 MCAP 引用 / 派生产物 / 标注 / embedding...)
```

### 3.2 术语定义

| 术语 | 定义 | 例子 |
| --- | --- | --- |
| **MCAP File** | 用户回传的物理 `.mcap` 文件,存 GCS | `gs://grace-raw-mcap/2026/04/22/robot42_log.mcap` |
| **Asset** | **平台核心概念**,= 一段有效视频片段,对应 MCAP 中 (t_start, t_end) | `asset_id = a1b2c3d4-...` |
| **Segment(虚拟)** | Asset 指向 MCAP 的时间区间,**不物理切文件** | `(t=12.3s, t=57.8s) in robot42_log.mcap` |
| **Segment(物化)** | 必要时真的切出来独立文件 | `gs://grace-derived/asset-a1b2c3/raw.mcap` |
| **File** | Asset 的关联文件(原始引用 / 派生 / 标注) | `{kind:raw_mcap, uri:...}` |
| **Tag** | 业务标签,可无限扩展 | `Scene.urban`、`Weather.rainy` |
| **Classification** | Tag 的一级分类 | `Scene`、`Weather`、`Quality` |
| **Glossary Term** | 业务术语,有正式定义 | "有效 segment = 经 QA approved" |
| **URN** | 全平台统一标识 | `urn:grace:asset:<uuid>` |
| **Aspect** | Asset 的逻辑切面(core / tags / files / qa / ...) | 映射 Bigtable 列族 |
| **Producer** | 生产者(算法/人工) | `sam2@v3.1`、`human@alice` |

### 3.3 Asset 生命周期

```
pending ──(人工/自动 QA)──▶ in_review ──approve──▶ approved ──▶ active
                                 │
                                 └──reject──▶ rejected ──▶ archived
```

### 3.4 事件语义

平台内部所有变更通过 **MCE(Metadata Change Event)→ MCL(Metadata Change Log)** 协议广播(借鉴 DataHub),详见 [06-borrowed-patterns.md](06-borrowed-patterns.md)。

---

## 4. 规模假设(用于选型)

| 维度 | 当前 | 1 年后 | 3 年后 |
| --- | --- | --- | --- |
| MCAP 文件数 | 10 万 | 100 万 | 1000 万 |
| Asset (segment) 数 | 50 万 | 500 万 | **1 亿+** |
| Tag 维度 | 100 | 1000 | 10000 |
| 每个 asset 平均 tag 数 | 5 | 20 | 50 |
| **Tag 条目总数** | 250 万 | **10 亿** | **50 亿** |
| 派生文件 / asset | 2 | 5 | 10 |
| 单 MCAP 平均大小 | 200MB | 500MB | 1GB |
| **单次搜索延迟目标** | — | **< 500ms** | < 500ms |
| **单点查询延迟目标** | — | **< 50ms** | < 20ms |

> 这些数字决定了:
> - Bigtable(点查 <10ms,百亿行扩展)
> - OpenSearch(分面过滤 <500ms)
> - Vertex AI Vector Search(百亿级向量)
> - 不用 Postgres 作为主存储(百亿行压力大)

---

## 5. 非目标(明确不做)

| # | 不做 | 原因 |
| --- | --- | --- |
| 1 | 不做**通用数据目录**(DataHub/OpenMetadata 那种) | 我们只管自己的 MCAP 资产 |
| 2 | 不做**通用 BI / 自助报表** | Phase 2+ 起 Superset 即可 |
| 3 | 不做**训练 pipeline 编排** | Dagster 负责,我们只提供资产 |
| 4 | 不做**模型服务 / inference**  | 用 Vertex AI / 自研服务 |
| 5 | 不做**自己的 MCAP 播放器** | 嵌入 Foxglove App 或 Lichtblick |
| 6 | 不做**标注工具** | 嵌入 Label Studio / CVAT |
| 7 | 不做**多云 / 跨云** | 只支持 GCP(可 Phase 3+ 再扩) |
| 8 | 不支持**非 MCAP 格式**(Phase 0) | 对 rosbag/parquet 不做一等公民支持 |

---

## 6. 核心设计原则

1. **单一事实来源(Single Source of Truth)** —— Bigtable 宽表是权威,其余都是索引副本
2. **面向 `asset_id` 的点查至上** —— 一切对外 API 从 `asset_id`(UUID / URN)出发;物理实现 Phase 1+ 走 `asset_locator → main_row_key` 一跳,**对 SDK/UI 完全透明**(详见 [03-data-model-wide-table §2](03-data-model-wide-table.md))
3. **事件驱动,最终一致** —— 上游写 Bigtable,MCL 广播,下游异步同步索引
4. **算法用户只认 SDK** —— 不暴露 SQL / 表名 / bucket 路径
5. **开源能借就不造** —— 但**绝不 fork 大型数据平台**(详见 ADR-004)
6. **Phase 0 可跑可查** —— 用 PG JSONB 模拟宽表,无 Bigtable 也能本地跑通
7. **每个 asset 自带审计链** —— `cf:event` + Cloud Audit Logs 双重记录

---

## 7. 后续阅读

- 架构总览 → [02-architecture.md](02-architecture.md)
- 宽表设计 → [03-data-model-wide-table.md](03-data-model-wide-table.md)
- 路线图 → [08-roadmap.md](08-roadmap.md)
