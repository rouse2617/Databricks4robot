# ADR-008:不把 Databricks 作为 data4cyber 核心平台(附逐项更优解)

- **Status**: Accepted
- **Date**: 2026-04-22
- **Related**: [ADR-001](ADR-001-wide-table-with-bigtable.md) · [ADR-003](ADR-003-mcap-as-primary-format.md) · [ADR-004](ADR-004-no-fork-datahub-openmetadata.md) · [07-tech-stack-gcp.md](../07-tech-stack-gcp.md) · [08-roadmap.md](../08-roadmap.md)

---

## Context

团队注意到 Databricks 在 2025–2026 的产品能力已经非常全(Lakehouse / Unity Catalog / Lakeflow / MLflow 3 / Mosaic AI / Lakebase / Agent Bricks)。自然的问题是:

> 我们要不要直接把 `data4cyber` 搭在 Databricks 上,或者用它替换掉现有方案里的某些组件?

本 ADR 做两件事:

1. **回答主问题**:核心平台要不要上 Databricks(结论:**不**)
2. **按子问题逐项扫描**:哪些地方 Databricks **确实有参考价值 / 甚至是更优解**,哪些地方坚持当前方案

---

## 2026 Databricks 产品全景(调研结论)

| 产品线 | 定位 | 与我们的相关性 |
| --- | --- | --- |
| **Delta Lake** | Open table format(Parquet + 事务日志) | 中:可作为训练集导出格式(但 Iceberg 更中立) |
| **Unity Catalog(UC)** | 统一数据 & AI 资产治理(表 / 模型 / 函数 / Volume) | 中:**OSS 版(2024-06 Apache 2.0 开源)可单独部署** |
| **Lakeflow**(原 DLT + Lakeflow Connect) | 声明式 Pipeline + 数据库 CDC 镜像 | 低:我们是 MCAP 流水线,不是表 CDC |
| **Databricks SQL / Photon** | Serverless SQL 数仓 | 低:BigQuery 在 GCP 上成本 / 集成更好 |
| **MLflow 3** | 实验跟踪 / 模型注册 / AI-assisted eval | 中:**可自托管 OSS 版**,不用 Databricks 托管 |
| **Mosaic AI(Agent Bricks / Agent Framework)** | Auto-optimized agent、RAG、评估 | 低:我们是 CV/机器人,不是 LLM agent 场景 |
| **Lakebase**(2025-06 发布,2026 GA) | Serverless Postgres(Neon 架构) + UC 治理 | 低:文档/关系型,**非宽表**,与 Bigtable 形态错位 |
| **Databricks Vector Search** | 托管向量检索 | 低:Vertex AI Vector Search 已在计划里 |
| **Databricks Apps** | 在 Databricks 里跑 Web 应用 | 低:我们 UI 是独立 React + GKE |

### Databricks on **GCP** 的已知硬伤(2026-04)

| 项 | 限制 | 对我们的影响 |
| --- | --- | --- |
| Standard compute | **不支持 GPU** | 🔴 SAM2 / embedding 必须 GPU,直接出局 |
| Standard compute | **不支持 Databricks Runtime for ML** | 🔴 GPU 侧的 PyTorch/vLLM 栈拿不到 |
| Serverless compute | **没有到 GCS / BigQuery 的 Private Service Connect** | 🔴 和我们私网 + GCS 低延迟的基本要求冲突 |
| Serverless compute | 只支持 Spark Connect API,无 Spark UI / logs | 🟡 调试体验差 |
| 功能完整度 | Databricks 自家资料承认 "GCP has fewer features than AWS/Azure" | 🔴 踩坑风险 |

> **单这几条就足以否决"把 Databricks 当平台底座"这个选项**,后面的技术性讨论只是为了完整性。

---

## Decision

**不将 Databricks 作为 `data4cyber` 的核心平台**,也**不在 Phase 0/1 引入 Databricks Workspace**。

改为:

1. 核心架构维持当前方案:**GKE(Dagster + Ray + 自研服务)+ Bigtable / GCS / Pub/Sub / Vertex AI / BigQuery**
2. **吸收 4 个可落地的好设计**(见下节"What to Borrow")
3. **Phase 2+ 评估**是否在**窄切面**(训练集导出 / 跨团队数据共享)**接入开源等价物**(Iceberg + UC-OSS / MLflow OSS),而非托管 Databricks

---

## 子问题更优解扫描(逐项对比)

对每一个 `data4cyber` 实际子问题,评估"Databricks 的做法" vs "我们当前方案" vs "是否有第三条更优路径"。

### S1. 核心元数据底座(asset 宽表)

| 方案 | 结论 |
| --- | --- |
| **当前**:Phase 0 = Cloud SQL Postgres(8 JSONB + `asset_events`);Phase 1+ = Bigtable(9 CF) | ✅ 形态吻合:宽列 + 百亿 tag + 多版本 |
| Databricks Lakebase(Neon-style Postgres + UC) | 🔴 文档/关系型,**不是宽列**;且把我们拉进 Databricks 生态 |
| Delta Lake + UC 当"asset 表" | 🔴 Delta 优势在 OLAP / 批读,不是高并发点查;且 asset write 路径会被拖成 Spark job |
| **更优解** | **维持当前方案**。Bigtable 是 GCP 上唯一成熟的"宽列 + 行级事务 + 百亿级规模"产品 |

### S2. 训练集 / 派生数据集的存储格式

| 方案 | 结论 |
| --- | --- |
| **当前**:GCS + 自定义目录结构 + MCAP / Parquet 混杂,版本手工管理 | 🟡 能用但易散乱 |
| Databricks 托管 Delta Lake | 🔴 拉进 Databricks 生态,绑定 |
| **开源 Delta Lake**(GCS 上裸跑) | 🟡 格式 OK,但社区活跃度 / 跨引擎支持略弱于 Iceberg |
| ⭐ **Apache Iceberg + BigLake/BigQuery** | ✅ GCP 原生支持,Spark / Dataproc / BigQuery / Trino / DuckDB 都能读,中立开放 |
| **更优解** | **Iceberg on GCS**(Phase 2+)。BigQuery 已直接支持 Iceberg;Ray / Dagster / Dataproc Serverless 也都能读写 Iceberg |

### S3. 训练集 / 派生数据的目录与治理

| 方案 | 结论 |
| --- | --- |
| **当前**:只在 Bigtable `cf:lineage` 记血缘,派生数据没有统一目录 | 🟡 Phase 0 够用,Phase 2+ 不够 |
| Databricks 托管 Unity Catalog | 🔴 要求数据必须在 Databricks / UC-aware 引擎里,强绑定 |
| ⭐ **Unity Catalog OSS**(2024-06 起 Apache 2.0,LF AI & Data) | ✅ **可独立部署**,支持 Iceberg REST Catalog + Hive Metastore API,Spark / Trino / DuckDB / Snowflake 都能接 |
| **更优解** | **Phase 2+ 引入 UC-OSS 作为"训练集 Iceberg 表目录"**,和 `data4cyber` 自有的 asset 目录**互不替代、用 URN 打通**。主 asset 目录仍是我们自研的(只有我们的 Entity 是 MCAP segment) |

### S4. ML 实验跟踪 / 模型注册

| 方案 | 结论 |
| --- | --- |
| **当前**:无(算法用户各自在本地 / GCS 手工管) | 🔴 痛点 |
| Databricks 托管 MLflow | 🔴 捆绑 Databricks Workspace |
| ⭐ **自托管 MLflow OSS**(GKE + Cloud SQL + GCS artifact) | ✅ 5 分钟起,和 Vertex AI Experiments 二选一 |
| **Vertex AI Experiments / Model Registry** | ✅ GCP 原生,免运维,但灵活度略低 |
| **更优解** | **Vertex AI Experiments 优先**(最省心);若团队已有强 MLflow 习惯,走 **OSS 自托管**。两者都比 Databricks MLflow 简单、便宜 |

### S5. 向量检索 / 多模态

| 方案 | 结论 |
| --- | --- |
| **当前计划**:Vertex AI Vector Search | ✅ 已是最优解 |
| Databricks Vector Search | 🔴 多一层托管栈,GCP 侧无产品优势 |
| **更优解** | **维持 Vertex AI Vector Search**。Databricks 的 Vector Search 在非 AWS/Azure 场景没有必要引入 |

### S6. 数据工程编排 / 声明式 Pipeline

| 方案 | 结论 |
| --- | --- |
| **当前**:Dagster Assets + Ray Pipes | ✅ 对 MCAP + GPU 场景极度契合(ADR-003 里已确认) |
| Databricks Lakeflow / DLT | 🔴 声明式的是"SQL/Python 表到表",不是"MCAP → segment → embedding" |
| **更优解** | **维持 Dagster + Ray**。Lakeflow 的"声明式依赖"思想 Dagster 的 `@asset` 已经实现了 |

### S7. 自助 SQL / 分析

| 方案 | 结论 |
| --- | --- |
| **当前**:BigQuery | ✅ GCP 原生,成本可控 |
| Databricks SQL + Photon | 🔴 在 GCP 上比 BigQuery 多一层成本 + 少一层原生集成 |
| **更优解** | **维持 BigQuery**,需要湖仓查询时用 **BigLake + Iceberg**(不是 Delta) |

### S8. AI 应用开发 / Agent

| 方案 | 结论 |
| --- | --- |
| **当前**:Phase 1+ 再评估 | - |
| Databricks Agent Bricks / Mosaic AI | 🟡 LLM-agent 方向的亮点,但我们是 CV / 机器人 |
| **更优解** | **暂不引入**。未来真需要做 RAG / agent 时,Vertex AI Agent Builder + 自研 LangGraph/Agno 的组合比 Databricks 更 GCP 原生 |

### 汇总

| 场景 | Databricks 更优? |
| --- | --- |
| asset 核心元数据底座 | ❌ |
| 派生/训练集存储格式 | 🟡 思想吸收(Iceberg,非 Delta) |
| 派生/训练集目录治理 | 🟡 思想吸收(UC-OSS,非托管 UC) |
| ML 实验追踪 | 🟡 思想吸收(MLflow OSS / Vertex AI Experiments) |
| 向量检索 | ❌ |
| 编排 | ❌ |
| SQL 分析 | ❌ |
| Agent | ❌ |

**没有一项"必须用 Databricks"**,但有 3 项可以**从 Databricks 生态的 OSS 组件里拿灵感 / 拿软件**。

---

## Alternatives Considered

### A. All-in Databricks(Workspace + UC + Delta + MLflow + DLT)

- 🔴 GCP 版硬伤(无 GPU、无 PSC、功能落后于 AWS/Azure)
- 🔴 MCAP 场景在 Databricks 零生态(仅有社区问答讨论 rosbag 库装不装得上),和 Foxglove / Lichtblick 的 MCAP 亲和度不可同日而语
- 🔴 Zero-code SDK 哲学不兼容:Databricks 强推 Notebook / Workspace UI
- 🔴 DBU 计费 + GCP infra 计费叠加,成本不可预测(小规模 POC 2–3 倍自研)
- 🔴 把核心元数据锁进 UC 托管表 → 后续 GCP-native 回迁极其痛苦

### B. Databricks 作为"旁路分析平台"(sidecar)

- 🟡 可行:BigQuery/GCS 数据 → Databricks SQL 给 DE 团队自助查
- 🔴 但 `data4cyber` Phase 0/1 用户 = 算法团队,不需要额外 BI 层
- 结论:**Phase 2+ 再按需评估**,现在不做

### C. 只用 Databricks 的 OSS 组件(Delta / MLflow / UC-OSS)

- ✅ 无托管成本,无锁定
- 🟡 但"用 Delta"和"用 Iceberg"对我们是等价选项 → **选更中立的 Iceberg**
- 🟡 UC-OSS 成熟度仍在快速演进,Phase 2+ 再引入

### D. 维持当前 GCP-native 方案 + 吸收思想(**✅ 选中**)

- 详见 "What to Borrow" 和上面的子问题扫描

---

## What to Borrow(具体 & 可落地)

| 借鉴 | 来源 | 在 data4cyber 里对应的事 | 时机 |
| --- | --- | --- | --- |
| **1. Open table format 做派生集** | Databricks Delta 思想 | 训练集、标注产物用 **Iceberg** 写 GCS,不是随便堆目录 | Phase 2 |
| **2. 统一目录 + 多引擎访问** | Unity Catalog | 派生 Iceberg 表用 **UC-OSS** 做目录,暴露 Iceberg REST Catalog,Dagster/Ray/BigQuery/Trino 都能读 | Phase 2 |
| **3. 自托管实验跟踪** | MLflow | 任一方案都行:**Vertex AI Experiments** 或 **MLflow OSS 自托管**,强制算法 SDK 里埋点 | Phase 1 |
| **4. Medallion 命名** | Databricks Bronze/Silver/Gold | GCS bucket 改名:`grace-raw-*` → 保留;派生层引入 **silver/gold** 语义(QA 通过 / 可交付) | Phase 1 |

> 注意:**不引入 Delta Lake**、**不引入 Databricks-managed UC**、**不引入 Databricks MLflow**。我们只用这些产品背后的"思想 + 同类 OSS"。

---

## Consequences

### Positive

- ✅ 架构维持 GCP-native,所有组件都能用 Workload Identity / VPC-SC / Private Service Connect 串起来
- ✅ 不承担 DBU 2 层计费,成本可预测
- ✅ 团队技术栈(Go / Python / Dagster / Ray)无需迁移
- ✅ 长期保持迁移自由(Iceberg 是中立开放格式,OSS 目录 / OSS MLflow 可换)

### Negative

- ⚠️ 放弃 Databricks 的"开箱即用"一体化体验,要自己攒 Dagster + Ray + MLflow + UC-OSS 的组合
- ⚠️ Phase 0/1 没有"统一元数据治理面板"(但我们有自研 asset 平台,够用)
- ⚠️ 如果公司未来战略转向 Databricks,会有一次迁移成本

### Mitigations

- **演进锚点**:Phase 2 再评估是否引入 UC-OSS / MLflow OSS,届时数据以 Iceberg 落地,迁移代价可控
- **人才储备**:团队内部对 Delta / UC / MLflow 保持跟进(share 频次见 Acceptance)
- **应急通道**:若公司突然强制上 Databricks,走 **C 方案**(只用 OSS 组件),不动核心平台

---

## Acceptance Criteria

- [ ] 项目代码、依赖、部署文件中**零** `databricks-*` / `delta-*` runtime 依赖(Phase 0/1)
- [ ] `07-tech-stack-gcp.md` 明确标注:训练集导出格式 = **Iceberg**(Phase 2+)
- [ ] `08-roadmap.md` 里写明 Phase 2 里程碑包含"评估 UC-OSS 引入"
- [ ] 团队内部完成 1 次 Databricks vs GCP-native 的 share(对齐本 ADR 结论)
- [ ] `06-borrowed-patterns.md` 追加"Medallion / Open table / 自托管实验跟踪"三条(标注来源为 Databricks 生态)

---

## References

- Databricks on GCP limits: https://docs.databricks.com/gcp/en/resources/limits
- Databricks GCP serverless limitations: https://docs.databricks.com/gcp/en/compute/serverless/limitations
- Unity Catalog 开源公告(2024-06): https://www.databricks.com/blog/open-sourcing-unity-catalog
- Unity Catalog OSS: https://github.com/unitycatalog/unitycatalog
- Apache Iceberg: https://iceberg.apache.org/
- MLflow OSS: https://mlflow.org/
- Lakebase(2026-04): https://databricks.com/blog/databricks-lakebase-generally-available
- Agent Bricks: https://databricks.com/blog/introducing-agent-bricks
- 相关 ADR:[ADR-001](ADR-001-wide-table-with-bigtable.md) · [ADR-003](ADR-003-mcap-as-primary-format.md) · [ADR-004](ADR-004-no-fork-datahub-openmetadata.md)
