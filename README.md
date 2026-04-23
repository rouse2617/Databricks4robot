# data4cyber · MCAP 资产数据平台

> 面向机器人 / 自动驾驶 / CV 算法场景的**统一资产数据平台**。
> 核心定位:**以 MCAP 为原生数据格式,以"视频片段 (segment)"为最小资产单元**,
> 用一张 **Bigtable 宽表** 作为统一元数据底座,屏蔽底层存储/算力,
> 让算法用户"开箱即用",所有交互只通过 SDK / Web UI。

---

## TL;DR(1 分钟读懂)

| 维度 | 结论 |
| --- | --- |
| **数据源** | 单一:用户回传的 MCAP 文件(GCS) |
| **资产单元** | 一段**有效视频片段**(t_start, t_end within MCAP),主键 `asset_id` |
| **存储底座** | **Bigtable 宽表,9 个 column family**(Phase 0 = PG 上 8 JSONB + `asset_events` 表模拟) |
| **编排** | Dagster Assets + Ray(Pipes 协议) |
| **检索** | OpenSearch(Tag/全文) + Vertex AI Vector Search(多模态) + BigQuery(OLAP) |
| **UI** | 自研 React 核心页 + 嵌入 Foxglove / Dagster / Label Studio / Superset |
| **云** | GCP 全家桶(GKE / Bigtable / GCS / Pub/Sub / BigQuery / Vertex AI) |
| **不造的轮子** | DataHub / OpenMetadata(只偷设计模式,不 fork) |

---

## 目录导航

### 📘 核心文档(按顺序读)

| 文档 | 内容 |
| --- | --- |
| [01-overview.md](docs/01-overview.md) | **愿景、场景、核心概念、非目标** |
| [02-architecture.md](docs/02-architecture.md) | **架构全景**(对齐三级能力图)+ Mermaid |
| [03-data-model-wide-table.md](docs/03-data-model-wide-table.md) | **宽表设计**(asset_id + 9 列族) |
| [04-mcap-and-segment.md](docs/04-mcap-and-segment.md) | **MCAP 原生 + Segment 模型** |
| [05-ui-strategy.md](docs/05-ui-strategy.md) | **Web UI 方案**(借鉴 + 嵌入) |
| [06-borrowed-patterns.md](docs/06-borrowed-patterns.md) | **DataHub / OpenMetadata 偷师清单** |
| [07-tech-stack-gcp.md](docs/07-tech-stack-gcp.md) | **GCP 产品选型映射** |
| [08-roadmap.md](docs/08-roadmap.md) | **Phase 0 / 1 / 2 演进路线** |
| [09-asset-contract.md](docs/09-asset-contract.md) | **Asset 数据契约**(Asset=Segment,QA 可消费规则,API 契约) |

### 📜 架构决策记录(ADR)

| ADR | 决策 |
| --- | --- |
| [ADR-001](docs/adr/ADR-001-wide-table-with-bigtable.md) | 用 Bigtable 宽表作为统一元数据底座 |
| [ADR-002](docs/adr/ADR-002-segment-centric-asset.md) | 资产最小单元是 segment 而非整 MCAP |
| [ADR-003](docs/adr/ADR-003-mcap-as-primary-format.md) | MCAP 作为唯一原生数据格式 |
| [ADR-004](docs/adr/ADR-004-no-fork-datahub-openmetadata.md) | 不 fork DataHub / OpenMetadata |
| [ADR-005](docs/adr/ADR-005-ui-embed-strategy.md) | UI 用自研 + iframe 嵌入策略 |
| [ADR-006](docs/adr/ADR-006-urn-identity.md) | 全平台统一 URN 标识(**外部** API) |
| [ADR-007](docs/adr/ADR-007-write-path-and-event-contract.md) | **写入路径 & 事件契约**(权威)|
| [ADR-008](docs/adr/ADR-008-no-databricks-as-core-platform.md) | 不把 Databricks 作为核心平台(含逐项更优解扫描) |
| [ADR-009](docs/adr/ADR-009-phase1-migration-plan.md) | **Phase 0 → Phase 1 迁移计划**(PG → Bigtable + 索引 5 阶段 runbook) |

### 🗂 Schema / 示例

| 文件 | 内容 |
| --- | --- |
| [schemas/fields.yaml](schemas/fields.yaml) | Classification / Tag / Glossary 三层字段注册表 |
| [schemas/column-families.yaml](schemas/column-families.yaml) | Bigtable 9 个列族定义(目标态) |
| [schemas/pg-phase0.sql](schemas/pg-phase0.sql) | Phase 0 Postgres DDL(8 JSONB + `asset_events` 表) |
| [diagrams/architecture.mmd](diagrams/architecture.mmd) | **目标态**架构图(Bigtable + CDC) |
| [diagrams/phase0-runtime.mmd](diagrams/phase0-runtime.mmd) | **Phase 0 运行态**架构图(PG + Outbox + MCL) |

---

## 项目状态

- **阶段**:设计收敛 → 待 Phase 0 开工
- **负责**:TBD
- **最后更新**:2026-04-22

---

## 一图总览

```
┌─────────────────────────────────────────────────────────────────┐
│                    data4cyber Web UI (React)                    │
│        自研核心页 + iframe(Foxglove/Dagster/LabelStudio)        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────┴─────────────────────────────────┐
│          grace-sdk (Python,算法用户只认这个)                   │
└───────────────────────────────┬─────────────────────────────────┘
                                │
      ┌─────────────────────────┼─────────────────────────┐
      │                         │                         │
┌─────▼──────────┐   ┌──────────▼────────┐   ┌────────────▼────────┐
│ asset-service  │   │  mcap-gateway     │   │ cluster/transfer    │
│ (Go)           │   │  (Go, streaming)  │   │ service (独立)      │
└─────┬──────────┘   └──────────┬────────┘   └─────────────────────┘
      │                         │
      │                         │
┌─────▼─────────────────────────▼─────────────────────────────────┐
│    ⭐ Cloud Bigtable(asset_id 主键 + 9 列族宽表,目标态)        │
│       cf:core | cf:time | cf:tag | cf:file |                    │
│       cf:qa   | cf:event| cf:lineage | cf:algo | cf:emb         │
│    Phase 0:Cloud SQL Postgres(8 JSONB 伪列族 + asset_events)   │
└─────┬───────────────────────────────────────────────────────────┘
      │  CDC (Pub/Sub + Dataflow)
      │
      ├─▶ OpenSearch      (Tag / 全文)
      ├─▶ Vertex AI VS    (多模态 / 向量)
      ├─▶ BigQuery        (OLAP / 审计)
      └─▶ Dagster Assets  (编排)
              │
              └─▶ Ray on GKE (算力)

─────────────────────────────────────────────────────────────────
GCS:  grace-raw-mcap / grace-derived / grace-annotation
      生命周期 → Nearline → Coldline(归档)
```

---

## 推荐阅读路径

- **产品 / 管理**: [01-overview](docs/01-overview.md) → [08-roadmap](docs/08-roadmap.md)
- **架构师**: [02-architecture](docs/02-architecture.md) → [03-data-model-wide-table](docs/03-data-model-wide-table.md) → [ADR 全部](docs/adr/)
- **后端工程师**: [03](docs/03-data-model-wide-table.md) → [04](docs/04-mcap-and-segment.md) → [schemas/](schemas/)
- **前端工程师**: [05-ui-strategy](docs/05-ui-strategy.md) → [06-borrowed-patterns](docs/06-borrowed-patterns.md)
- **算法用户**: [01-overview](docs/01-overview.md)(了解 SDK 能做什么)
- **算法 / QA 契约**: [09-asset-contract](docs/09-asset-contract.md) → [ADR-007](docs/adr/ADR-007-write-path-and-event-contract.md)

---

## License / 内部可见性

- 内部项目,暂未开源。
