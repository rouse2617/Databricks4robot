# ADR-001: Iceberg REST Catalog 选型

- **状态**: 已决定（2026-05-13 修订）
- **初版日期**: 2025-01-15
- **修订日期**: 2026-05-13
- **决策者**: 平台团队
- **设计依据**: data-platform-design.md §5.6.4, §5.11, §6.2

## 背景

平台 2.0 引入 Apache Iceberg 作为湖仓表格式（§5.6.4），需要一个 REST Catalog 服务管理 Iceberg 表的 metadata pointer、snapshot 推进和跨引擎权限。

核心诉求：

1. 标准 Iceberg REST Catalog 协议 — PyIceberg、Trino、Spark 均可直连
2. 运维简单 — 最小化常驻服务
3. 多云迁出成本可控（数据与协议层不锁定）

2025-01 初版选定 **Apache Polaris** 作为生产 Catalog，理由是协议标准、跨云无锁定（详见末尾"修订记录"）。

2026-05 重新评估时环境出现两个关键变化：

1. **GCP Lakehouse (前身 BigLake) Iceberg REST Catalog GA** — Google 托管的标准 Iceberg REST Catalog 实现，支持 Workload Identity / Credential Vending，BigQuery 与开源引擎（Spark / Trino / Flink）可读写同一张 Iceberg 表
2. **Trino 自托管 REST Catalog 接 GCS + Workload Identity 当前存在已知缺陷** — [trino #29084](https://github.com/trinodb/trino/issues/29084) 强制要求 `gcs.json-key-file-path`，PR #29101 修复中。该缺陷在 POC 阶段直接阻塞了 Trino+Polaris+GCS 的部署

同时本仓库已确定：
- 目标云为 **GCP 单云**（短期）；多云迁出作为风险项管理，而非默认部署形态
- 团队规模小，**最小化常驻服务**是明确的工程目标
- 数据规模处于 POC → 早期生产（≤ 10 GB Iceberg 数据 / 月）

## 候选方案对比

| 维度 | Apache Polaris（初版选定） | GCP Lakehouse REST Catalog（当前选定） |
|---|---|---|
| 部署形态 | 自托管 Pod + PG HA | 完全托管，无 Pod |
| 协议 | Iceberg REST Catalog v1.0（官方参考实现） | Iceberg REST Catalog v1.0 |
| 月固定成本 | GKE 节点 + Cloud SQL ≈ $50–150 | $0（按用量，免费额度足够当前规模） |
| 用量计费 | 0 | metadata 存储 $0.04/GiB/月（首 1 GiB 免费）；Class A 写 $6/百万次（首 5K 免费）；Class B 读 $0.90/百万次（首 50K 免费） |
| GCS + Workload Identity | 受 Trino #29084 阻塞 | 原生支持，无需挂 SA JSON key |
| BigQuery 互操作 | 不支持 | 同一张表 BigQuery 与 Spark/Trino 共享读写 |
| 多云锁定 | 无（自托管） | 中等（见下方"风险与缓解"） |
| 运维负担 | 升级 / 备份 / 监控自管 | 0 |
| RBAC | 内置 catalog/namespace/table 粒度 | 走 GCP IAM；当前不支持行/列级（phase-0 用不上） |
| 限制 | 无 | Parquet only；`metadata.json` ≤ 1 MB |

其他评估过但未入选的方案：

- **Project Nessie**：git-like 分支语义对当前需求过度，同样要承担自托管运维
- **Unity Catalog OSS**：Databricks 偏好过重，独立部署成熟度不足
- **AWS Glue**：本仓库部署目标是 GCP
- **Lakekeeper**：社区项目尚未进入 Apache 孵化，生产案例少

## 决策

**生产 Iceberg REST Catalog 选用 GCP Lakehouse (BigLake) REST Catalog。**

本地开发继续使用 `apache/iceberg-rest-fixture`（协议兼容，零业务改动）。

## 理由

1. **成本与运维全面优势**：当前规模下托管 Catalog 月费近乎为零，省去一个常驻服务 + HA Postgres 的运维负担
2. **解锁 GCS + Workload Identity**：避开 Trino #29084 阻塞，无需挂载 SA JSON 文件，符合密钥最小化原则
3. **协议保持开放**：BigLake 实现的是标准 Iceberg REST Catalog v1.0，应用层（Trino / Spark / PyIceberg）零改动
4. **数据所有权未变**：Parquet 与 metadata.json 全部在 GCS 自有 bucket，BigLake 只负责"表名 → 当前 metadata.json"的指针
5. **互操作红利**：BigQuery 可直接查询同一张 Iceberg 表，未来 BI / ad-hoc 不再需要双写

## 风险与缓解（迁出预案）

接受"中等程度 GCP 绑定"作为代价，通过以下 **4 条防御性规则**控制迁出成本，使其在工程上 **1–2 天可完成**（数据复制时间另算）：

### 防御性规则

1. **数据格式只用 Parquet**
   - BigLake 当前仅支持 Parquet，与其他云的 Iceberg 实现天然合规
   - 不引入 BigLake 特有的非 Parquet 写入路径

2. **不使用 BigLake 特有扩展**
   - ❌ BigQuery 多语句事务（multi-statement transactions）
   - ❌ BigQuery ObjectRefs（多模态特性）
   - ❌ BigLake 自动 table management（compaction / clustering / GC，DCU-Hour 计费）
   - ❌ History-based optimization（Preview 特性）
   - ✅ 自己用 Spark Action / Trino procedure 跑 compaction 与 snapshot expiration

3. **应用层只走 Iceberg REST 协议**
   - 后端 / Spark / Trino / Dagster 不直接调用 BigLake gRPC / Google SDK，统一通过 `iceberg.catalog.type=rest` 接口
   - 即使未来切到 Polaris / Nessie，配置改 endpoint URL 即可

4. **保留 metadata.json 历史**
   - 不开启 BigLake 自动 GC
   - Snapshot expiration 自管，保留最近 N 个 snapshot（建议 N ≥ 30，T+1 维护频率下覆盖 30 天）
   - 迁出时 metadata.json 历史是路径重写脚本的输入

### 迁出工作量估算

参考 2 GB Iceberg 数据规模（当前 POC 量级）：

| 阶段 | 时间 | 工具 |
|---|---|---|
| GCS → 目标对象存储数据复制 | ~10 分钟（数据量决定，TB 级数小时） | `gsutil rsync` / Storage Transfer Service |
| metadata.json 路径重写（`gs://` → `s3://` 等） | 半天（脚本一次性开发） | 自研脚本 + Iceberg `rewrite-manifests` procedure |
| 部署目标 Catalog（Polaris / Nessie / Glue） | 半天 | 标准 K8s 部署 |
| 应用层 catalog endpoint 切换 + 验证 | 半天 | 配置变更 |
| **合计** | **1–2 天 + 数据复制时间** | |

### 应急脚本预案

按规则 4 的要求，仓库中维护一份 **`scripts/iceberg-metadata-rewrite.py`**（即使不使用），用于把 metadata.json 中的 `gs://` 路径重写为目标云路径。半天工作量，提供 100% 迁出能力。

> 跟踪 issue：**TODO**（创建后回填）

## 后续行动

1. **代码与配置**
   - 移除 `deploy/k8s/lakehouse-minio/`（POC 用，保留 1 个版本作历史参考后归档）
   - **Trino + BigLake REST**：已并入 `deploy/k8s/lakehouse-gcs/`（`configmap-trino-catalog.yaml` + Trino `>=480`），不再单独新增 `lakehouse-gcs-biglake/` 目录
   - `backend/.env.example` 增加 BigLake catalog endpoint 配置项
   - 本地开发继续使用 `make iceberg-up`（`apache/iceberg-rest-fixture`），无需变更

2. **运维 job（按防御规则 2、4 自管）**
   - Spark / Trino 定时 job：`OPTIMIZE TABLE ... REWRITE DATA USING BIN_PACK`
   - Snapshot expiration job：保留近 30 个 snapshot
   - Orphan file cleanup job：每周
   - 详见后续 ADR（Iceberg 表维护策略）

3. **文档同步**
   - 更新 `docs/review/data-platform-design.md` §5.11，反映 catalog 选型变更
   - 更新 `backend/README.md` 环境变量与依赖说明
   - 同步更新 `CLAUDE.md` 与 `.cursor/rules/cyber-databrew-claude.mdc` 中的 catalog 提法

4. **迁出预案落地**
   - 创建 Linear Issue：开发并维护 `scripts/iceberg-metadata-rewrite.py`
   - 1.0 上线前完成一次"演练迁出到 MinIO" 的桌面演练，验证脚本可用

## 升级 / 重新评估触发条件

以下任一条件触发本 ADR 重新评估：

- **成本失控**：Class A 操作月费 > $500（写入量级达到每秒级 commit）
- **限制阻塞业务**：需要非 Parquet 格式 / `metadata.json` > 1 MB / 行级权限
- **多云从可选变为必选**：执行迁出预案，回到 Polaris / Nessie 自托管
- **GCP Lakehouse 产品方向变化**：服务降级 / 显著涨价 / 不兼容性变更

## 参考资料

- [GCP Lakehouse (BigLake) 官方文档](https://docs.cloud.google.com/biglake/docs/blms-rest-catalog)
- [GCP Lakehouse 定价](https://cloud.google.com/biglake#pricing)
- [外部 catalog 迁入 Lakehouse 工具](https://docs.cloud.google.com/bigquery/docs/migration/external-metastore-lakehouse-migration)（反向可用）
- [Trino issue #29084 — REST Catalog 强制要求 gcs.json-key-file-path](https://github.com/trinodb/trino/issues/29084)
- [Iceberg REST Catalog spec v1.0](https://iceberg.apache.org/spec/)
- [Polaris vs Nessie vs Unity 2026 对比](https://iotdigitaltwinplm.com/iceberg-catalogs-polaris-vs-nessie-vs-unity-comparison-2026/)

---

## 修订记录

### 2026-05-13 — 由 Apache Polaris 改为 GCP Lakehouse REST Catalog

- **变更**：生产 Catalog 从自托管 Apache Polaris 改为 GCP 托管的 Lakehouse Iceberg REST Catalog
- **触发原因**：
  1. GCP Lakehouse REST Catalog GA，托管化大幅降低运维与成本
  2. Trino #29084 阻塞了自托管 Polaris + GCS + Workload Identity 链路
  3. 仓库目标云收敛到 GCP 单云，跨云零绑定不再是硬约束
- **保留原决策的部分**：本地开发仍使用 `apache/iceberg-rest-fixture`；协议层仍是标准 Iceberg REST Catalog v1.0
- **新增**：4 条防御性规则 + 迁出工作量估算 + 应急脚本预案

### 2025-01-15 — 初版决策：Apache Polaris

初版选定 Apache Polaris，主要理由：

- Apache 顶级项目（Snowflake 捐赠），官方 Iceberg REST Catalog 参考实现
- 跨云零绑定 — 不依赖 Glue / Unity Catalog / BigLake
- 内置 catalog/namespace/table 粒度 RBAC
- Trino / Spark / PyIceberg / Flink 多引擎已验证

未选 Lakekeeper 的原因：社区项目未进入 Apache 孵化、生产案例少、权限模型基础。

详细对比与初版理由已并入上方"候选方案对比"表与"未入选方案"小节。
