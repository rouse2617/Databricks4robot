# 07 · 技术栈:GCP 产品映射

## 1. 锁定 GCP 的前置说明

用户已明确:**所有云服务都在 GCP 上**。本文档据此提供"GCP 原生映射方案",避免多云复杂度。

---

## 2. 核心服务映射总表

| 能力 | Phase 0 | Phase 1 目标 | Phase 2+ 扩展 |
| --- | --- | --- | --- |
| **元数据主表** | Cloud SQL for Postgres(JSONB 宽表) | **Cloud Bigtable** | — |
| **文件存储** | GCS Standard | GCS + Lifecycle(Nearline/Coldline) | GCS + Archive |
| **搜索/Tag** | Postgres GIN 索引 | **OpenSearch on GKE** 或 Elastic Cloud | — |
| **向量检索** | — | — | **Vertex AI Vector Search** |
| **OLAP / 报表** | — | **BigQuery**(含 External Table 读 Bigtable) | BigQuery + BigLake(Iceberg) |
| **派生/训练集格式** | 裸 Parquet / MCAP | 裸 Parquet / MCAP | **Apache Iceberg on GCS**(非 Delta,详见 [ADR-008](adr/ADR-008-no-databricks-as-core-platform.md)) |
| **派生数据集目录** | 自研 asset 平台覆盖 | 自研 asset 平台覆盖 | 评估 **Unity Catalog OSS**(仅作 Iceberg REST Catalog,不上托管 Databricks) |
| **实验跟踪 / 模型注册** | — | **Vertex AI Experiments**(首选)/ MLflow OSS 自托管(可选) | 同左 |
| **事件总线** | **Pub/Sub** | Pub/Sub | Pub/Sub + Eventarc |
| **CDC(主表→索引)** | PG logical replication → Datastream(alpha) | **Datastream / Dataflow**(Bigtable Change Streams) | — |
| **编排** | **Dagster on GKE Autopilot** | 同左 | 同左 |
| **分布式算力** | **Ray on GKE**(KubeRay) | 同左 + HPA | 可选 Vertex AI Training |
| **容器** | **GKE Autopilot** | GKE Autopilot + Standard(Ray 用) | 同左 |
| **镜像** | **Artifact Registry** | 同左 | 同左 |
| **密钥** | **Secret Manager** | 同左 | 同左 |
| **身份 / 权限** | **IAM + Workload Identity** | 同左 | 同左 |
| **日志 / 监控** | **Cloud Logging + Monitoring** | + Cloud Trace + Error Reporting | — |
| **网关 / LB** | **GCLB** / Cloud Armor | 同左 | + Cloud CDN(前端资源) |
| **IaC** | **Terraform + gcloud** | 同左 | 同左 |

---

## 3. 存储层详细映射

### 3.1 元数据主表:**Cloud Bigtable**(Phase 1+ 核心)

| 维度 | 规格 |
| --- | --- |
| Instance 类型 | Production(非 dev) |
| Storage type | SSD |
| 初始节点 | 3 节点(单 cluster) |
| Region | asia-east1 或 us-central1(就近) |
| 备份 | 每日 snapshot(保留 7 天) |
| 扩展 | 单 cluster 可扩至数百节点,无需 downtime |

**Phase 0 过渡**:**Cloud SQL for Postgres 15**,`db-custom-4-16384` 起步,JSONB GIN 索引。

**迁移策略**:见 [08-roadmap.md](08-roadmap.md)。

### 3.2 文件存储:**GCS**

```
# Bucket 布局
grace-raw-mcap-<project>-<region>/      # 原始上传,30 天后 Nearline
grace-derived-<project>-<region>/       # 派生产物
grace-annotation-<project>-<region>/    # 标注
grace-materialized-<project>-<region>/  # 物化切段(客户交付)
grace-staging-<project>-<region>/       # 临时(7 天 TTL 自动清)
```

**生命周期规则**:

```yaml
lifecycle:
  rule:
    - action: { type: SetStorageClass, storageClass: NEARLINE }
      condition: { age: 30 }
    - action: { type: SetStorageClass, storageClass: COLDLINE }
      condition: { age: 90 }
    - action: { type: SetStorageClass, storageClass: ARCHIVE }
      condition: { age: 365 }
    - action: { type: Delete }
      condition:
        age: 7
        matchesPrefix: [staging/]
```

**上传**:Resumable Upload(大文件 multipart)。
**读取**:带 Range header 的 signed URL。

### 3.3 搜索:**OpenSearch on GKE**(Phase 1)

- 用 **Elastic Cloud on GCP Marketplace** 或 **OpenSearch Helm chart on GKE**
- 初始:3 master + 3 data(hot) + 2 data(warm)
- 订阅 Bigtable CDC,将 `cf:tag` + `cf:qa` + `cf:event` 同步成文档

**Index 规划**:

```
grace-assets-v1         { asset_id, tenant, tags: [...], qa_state, duration, ... }
grace-events-v1         { asset_id, event_type, created_at, payload }
grace-files-v1          { asset_id, kind, version, producer }
```

### 3.4 向量检索:**Vertex AI Vector Search**(Phase 2)

- 原 "Matching Engine",2024 后更名 Vector Search
- 支持百亿级,ann 延迟 < 50ms
- 费用按 QPS + 索引大小

**为什么不用 Milvus on GKE**:
- Vertex Vector 是 managed,运维 ~ 0
- 与 GCP IAM / Monitoring 原生集成
- Dagster 可通过 Vertex SDK 跑 build index

### 3.5 OLAP:**BigQuery**(Phase 1+)

- **External Table** 直读 Bigtable,零 ETL
- 用于:
  - 管理员 ad-hoc 查询
  - 审计报表(`asset_events` → BQ)
  - Dataset 构建(`SELECT asset_id WHERE ...` 输出交付)

```sql
-- BigQuery External Table over Bigtable
CREATE EXTERNAL TABLE grace_ods.assets
OPTIONS (
  format = 'BIGTABLE',
  uris = ['https://googleapis.com/bigtable/projects/.../instances/.../appProfiles/default'],
  bigtable_options = (...)
);
```

---

## 4. 计算层详细映射

### 4.1 **GKE Autopilot**(所有 stateless 服务)

- `asset-service` / `mcap-gateway` / `search-service` / `web-ui` 全部跑这里
- 免节点运维,按 Pod CPU/Mem 计费
- 适合**弹性、无状态、小内存**的微服务

### 4.2 **GKE Standard**(Ray Cluster)

- Ray on KubeRay Operator
- 用 Spot VMs 大幅降本(容错 via Ray 自身)
- A100/L4/T4 GPU 按需

### 4.3 **Dagster on GKE Autopilot**

```
components:
  - daemon     (sensor + scheduler,1 副本)
  - webserver  (UI,2 副本)
  - code-locations (用户代码,按业务多副本)
  - user-deployments (job runner,按需)
storage:
  - Cloud SQL for Postgres (Dagster storage)
  - GCS bucket (Dagster IO manager)
```

### 4.4 **Artifact Registry**(镜像 + Python 包)

```
us-docker.pkg.dev/grace-<project>/containers/
  ├── asset-service:v1.2.3
  ├── mcap-gateway:v0.5.0
  ├── web-ui:v0.3.0
  └── ray-worker:v0.1.0

us-python.pkg.dev/grace-<project>/packages/
  └── grace-sdk (0.1.0, 0.2.0, ...)
```

---

## 5. 事件 / 编排映射

### 5.1 Pub/Sub 主要 topic

| Topic | Schema | 生产者 | 消费者 |
| --- | --- | --- | --- |
| `grace-mce` | MCE(意图) | SDK / UI | asset-service |
| `grace-mcl` | MCL(确认) | asset-service | search-indexer / dagster-sensor / subscription-service |
| `grace-mcap-uploaded` | GCS events | GCS | dagster-sensor |
| `grace-algo-events` | AlgoEvent | Dagster jobs | dashboard |
| `grace-audit` | AuditLog | 所有服务 | BigQuery sink |

### 5.2 Cloud Scheduler(定时任务)

- 每小时跑 lineage 索引重建
- 每天 00:00 跑 archiving job
- 每周跑 stale-asset 清理

### 5.3 Eventarc(Phase 2 跨服务事件)

- 当 BigQuery 数据集导出完成 → 触发交付通知
- 当 GCS 归档成功 → 更新 Bigtable `cf:core.storage_tier`

---

## 6. 可观测 / 安全

### 6.1 Cloud Logging / Monitoring / Trace

- 所有服务 OpenTelemetry 输出
- Log Router 把结构化日志导 BigQuery(审计)
- Cloud Monitoring Dashboard + Alert Policy

### 6.2 IAM + Workload Identity

- 每个 GKE ServiceAccount 映射到 GCP SA
- asset-service SA 只有 Bigtable read/write + Pub/Sub publish
- mcap-gateway SA 只有 GCS read/write + Bigtable read
- 禁用 JSON key,全部 Workload Identity

### 6.3 VPC / 网络

- Private GKE cluster
- Cloud NAT 出网
- Private Service Connect 访问 Bigtable / BigQuery(不走公网)
- Cloud Armor DDoS / WAF

---

## 7. CI/CD

| 阶段 | 工具 |
| --- | --- |
| 源码 | GitHub / GitLab |
| CI | Cloud Build / GitHub Actions |
| 镜像 | Artifact Registry |
| CD(GKE) | Config Sync / Argo CD |
| IaC | Terraform Cloud / local + GCS backend |
| Secrets | Secret Manager(不写 Git) |

---

## 8. 成本估算(参考数字,Phase 0)

| 项 | 月成本 |
| --- | --- |
| GKE Autopilot(5 个微服务,小规模) | ~$300 |
| Cloud SQL PG(db-custom-4-16384) | ~$200 |
| GCS(Standard,5 TB) | ~$100 |
| Pub/Sub | ~$20 |
| Cloud Logging / Monitoring | ~$50 |
| Artifact Registry | ~$10 |
| **合计 Phase 0** | **~$700/mo** |

**Phase 1(切 Bigtable + OpenSearch)** 估算:~$2000-3000/mo。
**Phase 2(+ Vector Search + BigQuery 大量扫)**:~$5000+/mo。

---

## 9. 为什么选 / 不选(关键取舍)

| 选择 | 替代 | 原因 |
| --- | --- | --- |
| Bigtable over Spanner | Spanner | Spanner 贵 10x,不需要 SQL |
| Bigtable over Firestore | Firestore | Firestore 不适合宽列 |
| OpenSearch over Firestore search | — | Firestore 搜索能力极弱 |
| Vertex Vector over Milvus | Milvus on GKE | 减少运维;100B+ 前足够 |
| Dagster over Cloud Workflows | Cloud Workflows | Dagster Asset-first |
| Ray over Dataflow | Dataflow | Dataflow 不是通用 Python 任务 |
| Pub/Sub over Kafka | Kafka on GCE/MSK | 原生 managed,免运维 |
| GKE Autopilot over Cloud Run | Cloud Run | 需要长连接 / 状态 ingress |

---

## 10. 下一步

- 演进节奏 → [08-roadmap.md](08-roadmap.md)
- 数据模型 → [03-data-model-wide-table.md](03-data-model-wide-table.md)
