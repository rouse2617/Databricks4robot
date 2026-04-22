# 08 · 路线图

## 1. 总体节奏

```
Phase 0  |  跑通 E2E, PG 模拟宽表       |  0 - 3 月
Phase 1  |  切 Bigtable + OpenSearch    |  3 - 6 月
Phase 2  |  多模态 + BI + 高级治理       |  6 - 12 月
Phase 3  |  多租户 / 多区 / 合规         |  12+ 月
```

指导原则:
1. **Phase 0 一定要能真正用起来**(MVP,不是玩具)
2. **Phase 1 之前不允许上 Bigtable / OpenSearch**(避免过度设计)
3. **每 Phase 末做一次架构复盘**

---

## 2. Phase 0:MVP 打通(0-3 月)

### 目标

让一个真实 MCAP 能**从上传 → 索引 → QA → 算法 → 检索**走通,整条链路不卡。

### 里程碑

| 月 | 产出 |
| --- | --- |
| **M1** | 基础设施:Terraform / GKE / GCS / Cloud SQL / Pub/Sub;CI/CD 跑通 |
| **M2** | 核心服务:asset-service + mcap-gateway + grace-sdk v0.1;Dagster Asset graph 打通 Ray |
| **M3** | Web UI MVP(搜索 + 详情 + Foxglove 嵌入);1 个完整 demo video 从上传到交付 |

### 关键产出

**Infrastructure**
- [ ] Terraform 模块(`infra/terraform/gcp/`),一键拉起 Phase 0 环境
- [ ] GKE Autopilot + GCS buckets + Cloud SQL + Pub/Sub 就绪
- [ ] CI/CD pipeline(Cloud Build)
- [ ] 可观测:结构化日志 + Monitoring dashboard 骨架

**Backend Services**
- [ ] `asset-service`(Go):Create / Get / Search(简版)/ Update(CheckAndMutate 语义)/ Delete
- [ ] `mcap-gateway`(Go):Multipart Upload + Range GET streaming + Summary 提取
- [ ] `indexer-worker`(Python):MCAP footer 解析,写入 asset 主表
- [ ] 事件:MCE/MCL 发到 Pub/Sub,**暂不做完整 CDC**,写同步更新 PG GIN 索引

**Data Model**
- [ ] PG DDL(`schemas/pg-phase0.sql`)部署
- [ ] Classifications + Tags + Glossary 注册表(`schemas/fields.yaml`)初版
- [ ] URN 规范文档

**SDK / UI**
- [ ] `grace-sdk` v0.1:`asset.get/search/upload/download_mcap/iter_messages`
- [ ] `web-ui` v0.1:搜索页(简)+ 详情页(3 个 Tab)+ Foxglove 外链按钮
- [ ] 登录:Google SSO(OIDC)

**Dagster**
- [ ] Asset graph:`mcap_ingest` → `auto_tag` → `sam2_v3_run` → `qc_report`
- [ ] Ray Pipes 集成(参考 tekton-playground `sam2_asset_graph.py`)
- [ ] Sensor:监听 GCS uploaded event

**实验跟踪(Phase 0 末起,跟 SAM2 对齐)**
- [ ] `grace-sdk` 统一 `grace.run(...)` context,底层可切 **Vertex AI Experiments**(首选)或 MLflow OSS
- [ ] SAM2 demo 里强制 log params / metrics / model

**文档 / 流程**
- [ ] 用户手册:算法用户如何用 SDK(3 个 example notebook)
- [ ] Ops 手册:如何 debug / rollback / 扩容
- [ ] ADR 1-6 全部落地

### 非目标(Phase 0 不做)

- ❌ Bigtable(仍用 PG)
- ❌ OpenSearch(用 PG GIN)
- ❌ 向量检索
- ❌ 复杂 Lineage UI
- ❌ Glossary 富文本编辑
- ❌ 多租户 / 跨 region

### 成功准则

1. 算法用户能用 10 行代码访问任何 asset 的 MCAP 数据
2. 1 个完整用户旅程 demo:上传 → QA → 自动 SAM2 → 搜索 → 交付
3. 1 个完整 Dashboard:当前 asset 总数 / 状态分布 / 当周新增
4. 所有服务有 SLI/SLO 监控,oncall runbook 至少 1 份

---

## 3. Phase 1:规模化(3-6 月)

### 目标

从"能跑"升级到"能扛 **1000 万 asset / 10 亿 tag**"。

### 里程碑

| 月 | 产出 |
| --- | --- |
| **M4** | **Bigtable 双写 + backfill**,验证数据一致 |
| **M5** | **OpenSearch 索引**上线,搜索性能 10x 提升 |
| **M6** | PG 下线(只留 Dagster 元数据 + 审计),100% 走 Bigtable |

### 关键产出

**主表迁移**(按 [ADR-009](adr/ADR-009-phase1-migration-plan.md) 的 5 阶段 runbook 执行)
- [ ] 阶段 0 准备:Bigtable instance + 3 张表(主表 / `asset_locator` / `assets_by_project`)就绪,`grace-migrate` 工具就绪
- [ ] 阶段 1 双写:asset-service `pg_primary_bt_shadow` 模式,主表异步追新
- [ ] 阶段 2 回填:存量 backfill(32 并发)+ janitor 增量补偿
- [ ] 阶段 3 影子读:`bt_shadow_compare` 至少 24h,diff 率达标(< 10 ppm 点查)
- [ ] 阶段 4 切读:48h SRE 盯守;**SLO(locator 孤儿率 / 端到端 P99)从此刻起生效**
- [ ] 阶段 5 下线:PG 归档到 `gs://grace-archive/pg-phase0/`,保留 90 天

**搜索层**
- [ ] OpenSearch on GKE 部署(或 Elastic Cloud)
- [ ] CDC(Bigtable Change Streams → Dataflow → OpenSearch)
- [ ] 分面搜索 UI(tag / time / qa / duration)
- [ ] 全文搜索(notes, annotation 内容)
- [ ] 搜索 SLI:P99 < 500ms

**Lineage**
- [ ] `cf:lineage` 数据齐全,UI 用 React Flow 画图
- [ ] Upstream/Downstream 可展开
- [ ] Impact analysis(改这个 asset 影响下游哪些 dataset)

**运维升级**
- [ ] Canary 发布 / Blue-Green
- [ ] 自动化容量告警
- [ ] 备份恢复演练(月度)

### 成功准则

1. 500 万 asset,Bigtable 点查 P99 < 50ms
2. 10 亿 tag 条目,OpenSearch 组合过滤 P99 < 500ms
3. CDC 延迟 < 5s
4. 从 Phase 0 零停机迁移完成

---

## 4. Phase 2:高级能力(6-12 月)

### 目标

新增 **多模态检索 / BI / 自动化治理**。

### 关键产出

**多模态检索**
- [ ] Embedding worker:CLIP + 自研 SAM2 pool embedding,生成向量写 Vertex AI Vector Search
- [ ] 搜索 UI 新增"以图搜图"、"自然语言搜视频"
- [ ] 向量索引 + tag 联合查询(先向量召回 1k,再 tag 过滤)

**BI / OLAP**
- [ ] BigQuery External Table 直读 Bigtable
- [ ] `asset_events` 同步到 BigQuery
- [ ] Superset 部署,SSO 接入,预制 3-5 个 Dashboard
- [ ] 交付报表:客户 X 本月收到 N 条 asset,其中 Scene.urban 占 XX%

**治理 / 合规**
- [ ] Soft Delete + Versioning(30 天可恢复)
- [ ] PII 自动扫描(如标注里出现车牌号)
- [ ] Data Contract v1(对外交付定义)
- [ ] RBAC 细粒度(按 tenant / project)

**运营 / 自动化**
- [ ] Subscription service(订阅 MCL 自动触发)
- [ ] 自动标注算法闭环
- [ ] 算法效果 A/B 框架(同 asset 多个 version 的 file 对比)

**派生数据集格式化(来自 [ADR-008](adr/ADR-008-no-databricks-as-core-platform.md))**
- [ ] 训练集 / 标注快照 / 交付包改用 **Apache Iceberg on GCS**(非 Delta)
- [ ] 评估 **Unity Catalog OSS**(Iceberg REST Catalog)为跨团队派生数据目录
- [ ] BigQuery 侧以 **BigLake Iceberg 外表**直读,无需 ETL
- [ ] **Medallion 命名**正式上墙(Bronze / Silver / Gold)

### 成功准则

1. 多模态搜索 top-10 准确率 > 80%
2. Superset 产出 5 个日常使用的报表
3. PII 扫描覆盖 100% 新增 asset

---

## 5. Phase 3:企业化 / 合规(12+ 月,按需)

- 多租户隔离(Bigtable 按 tenant 分 instance 或 row key 分区)
- 多区域(asia-east1 + us-central1)+ 跨区 DR
- SOC2 / GDPR / 国内合规
- 完整数据主权(客户数据按客户区域存)
- 付费 SaaS(如果走出内部)

---

## 6. 依赖风险(跨 Phase)

| 风险 | 缓解 |
| --- | --- |
| Foxglove App 政策变化 | 准备 Lichtblick 自托管兜底 |
| Bigtable 成本超预期 | 定期做热数据 / 冷数据分层 |
| OpenSearch 运维复杂 | 评估 Elastic Cloud 托管替换 |
| Dagster 版本升级破坏性 | 锁主线版本 + 定期 bump |
| SDK 接口频变 | 语义化版本 + deprecate 通知 |
| 团队规模不足 | 优先外部 SaaS 而非自研 |

---

## 7. 团队建议(Phase 0)

| 角色 | 人数 |
| --- | --- |
| Backend(Go) | 2 |
| Data / Python(SDK + Ingestion) | 1-2 |
| Frontend | 1 |
| Infra / SRE | 0.5(可兼) |
| **合计** | **4-5 人** |

Phase 1 扩至 6-8 人(增 SRE / 搜索专家 / 前端)。

---

## 8. 决策 checkpoint

| 时间 | 决策 |
| --- | --- |
| Phase 0 末 | 确认是否如期切 Bigtable(否则推迟到 Phase 1.5) |
| Phase 1 中 | 确认搜索是用 OpenSearch vs Elastic Cloud |
| Phase 2 初 | 确认是否自建 Superset vs 买 Looker |
| Phase 3 初 | 确认是否做 SaaS |

---

## 9. 相关

- ADR 全集 → [adr/](adr/)
- 架构全景 → [02-architecture.md](02-architecture.md)
- 技术栈 → [07-tech-stack-gcp.md](07-tech-stack-gcp.md)
