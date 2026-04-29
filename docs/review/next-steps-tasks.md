# 下一步开发任务清单 / Next-Step Task Board

> 状态：**评审基线**，2026-04-29 出。每完成一项把状态改为 `done` + 把 PR 链接补到「落地证据」列。  
> 优先级：P0 = 当前最高优先级（其中 **§1 主表项阻塞 2.0 启用**；§1.1 / §1.2 标注为“同窗口配套”的任务不单独卡 gate）/ P1 = 2.0 关键路径 / P2 = 2.0 打磨 / P3 = Phase 2+ 候选。  
> 维护规则：**不要把 done 的项删除**，留档可追溯；新任务追加到对应 P 表底部。

---

## 0. 现状快照（截至 2026-04-29）

| 维度 | 当前态 | 下一里程碑 |
|------|--------|------------|
| 架构基线 | 1.0（PG + Backend 单进程） | 启用 2.0：Outbox Worker + ES + Iceberg |
| 投影表 | `asset_tags / asset_algo_latest` 已建已用，cf_* 列只读保留 | 字段消费收口 + lifecycle_state 主消费切换 |
| 事件流 | `asset_events` 在线写入，1.0 无下游消费 | Outbox Worker 上线（pure polling, 30s tick） |
| Tag 子域 | 独立 tag CRUD + history API 已上线；详情页 / 批量操作已切专用 tag 接口 | 下一步：tag history UI / `cf_*` 兼容层清理 |
| 检索 | `/api/v1/search/assets` ES path 实现 + PG fallback | ES 真正接到 outbox 后才算"在线" |
| 湖仓 | `docker-compose` 脚手架，未接业务写路径 | PyIceberg CronJob + Polaris/Lakekeeper 上线 |
| Frontend | 资产发现 v2 facets / 详情 / 算法 / 交付齐全 | Phase 2+ 训练数据集 UI（按业务节奏） |

---

## 1. P0 · 1.0 收尾 / 2.0 启用前置（阻塞）

> 这一组完成才能开 2.0 启用闸口（详见 `data-platform-design.md` §9.1.1）。

| ID | 任务 | DoD（验收口径） | 估时 | Owner | 状态 | 落地证据 |
|----|------|----------------|------|-------|------|----------|
| **P0-1** | **Schema 字段提升收口**：核对 `assets / mcap_files` 上 `asset_type / lifecycle_state / end_timestamp_ns / duration_ms / owner / retention_tier / expire_at` 已在 migration / repo / OpenAPI / filter 白名单一致；只补缺口，不重做已上线字段 | 收口 PR 合入；仍缺的字段/索引补齐；`schemas/pg-phase0.sql` 与 `schema-reference.md` 同步；明确留下哪些 legacy 字段仍处兼容期 | M | backend | todo | — |
| **P0-2** | **存量数据 backfill**（P0-1 落地后）：把现有 `assets` 行的新字段从源数据补齐 | backfill 脚本幂等、可重跑；执行后新字段空值率 < 0.1%；对账报告归档 | M | backend + data | todo | — |
| **P0-3** | **`assets.lifecycle_state` 主消费切换**：列表过滤、详情展示、ES facet、查询文档都切到 `lifecycle_state`；老 `status` 进入退役计时（90 天） | 前端列表 / 详情 / facets 默认都用 `lifecycle_state`；API 仍同时返回两个字段，OpenAPI deprecated 标注；监控旧 `status` 读流量曲线 | M | backend + frontend | todo | — |
| **P0-4** | **Outbox Worker 进程内 MVP**（`OUTBOX_WORKER_ENABLED=true`） | 完整覆盖 `outbox-worker-design.md` G1–G5 验收；ES 索引投递端到端 P99 ≤ 60s；持续 24h 投递成功率 > 99.9% | L | backend | todo | — |
| **P0-5** | **`/admin/search/reindex` 全量重建 API**（同步交付） | 给定 dry_run / rebuild_index 参数；扫 `assets` 全表重建 ES doc；不动 outbox cursor；权限走独立 `ADMIN_TOKEN` | S | backend | todo | — |
| **P0-6** | **PgBouncer 入栈**：docker-compose / K8s 加 PgBouncer Deployment（transaction pool） | Backend & Worker 改 `DB_HOST/DB_PORT` 指向 PgBouncer，业务代码 0 改动；本地 + 预发跑 24h 无连接异常 | S | backend + 运维 | todo | — |

**关键依赖图**：

```
阻塞 2.0 gate：
P0-1 ──► P0-2 ──► P0-3
P0-4 ──► P0-5
P0-6（独立，可并行）

同窗口配套（不单独阻塞 gate）：
P0-FE-2 blocked-by P0-3
P0-FE-3 blocked-by P0-1
P0-T-1 blocked-by P0-4
P0-T-2 blocked-by P0-T-1
P0-T-3 blocked-by P0-2
P0-FE-1 / P0-FE-4 / P0-T-4 / P0-T-5（独立，可并行）
```

---

## 1.1 P0 · 前端同窗口配套（Frontend，非 gate）

> 触发：1.0 → 2.0 字段提升 + 通用事件 API 已经上线，但前端没跟。**这一组不单独阻塞 2.0 gate**；P0-FE-2/3 等 P0-1 / P0-3 落地后开始。

| ID | 任务 | DoD | 估时 | 状态 | 落地证据 |
|----|------|-----|------|------|----------|
| **P0-FE-1** | **`duration_sec` → `duration_ms` 全面切换**：`AssetsResultsPane` 排序选项 / 时长列、`OverviewTab` / `AssetPreviewHero` 详情卡都改成读 `duration_ms`，**单位统一以毫秒为底，UI 仍按秒展示**（除以 1000） | 全仓 `rg "duration_sec"` 仅剩 SDK / API 兼容层；`AssetsFacetSidebar` 的范围 chip 已经是 `duration_ms`（已完成），其余消费点同步切换；vitest 通过 | S | todo | — |
| **P0-FE-2** | **`lifecycle_state` 主路径切换**（与 P0-3 后端切换配套）：列表过滤、详情 badge、facet 都从 `status` 切到 `lifecycle_state`；老 `status` 仅在 `OverviewTab` 标灰显示并标注 `(legacy)` | 资产列表筛选默认走 `lifecycle_state`；`AssetsFacetSidebar` 的 `状态 (legacy)` chip 在 `lifecycle_state` 同时存在时隐藏；vitest + 手测交叉切换无回归 | M | blocked-by-P0-3 | — |
| **P0-FE-3** | **暴露 P0-1 新增字段到 UI**：`asset_type / retention_tier / expire_at / owner` 进列表列（`ColumnsConfigPopover`）+ 详情概览 + AddFilterPopover | 4 个字段都可勾选展示、可过滤；`expire_at` 用相对时间渲染（"7 天后到期"）；vitest 覆盖 column toggle | M | blocked-by-P0-1 | — |
| **P0-FE-4** | **通用事件时间线 UI**：`AssetDetailPage` 的算法事件 tab 改用 `/events?event_type=algo_*` + cursor 翻页；同时新增"全部事件"tab（不限 type）支持 `asset_created / tag_upserted / lifecycle_changed` 等 | 算法事件 tab 等价回归（已有适配器 `toAlgoEvent`，确保 cursor "load more" 工作）；新 tab 至少展示 5 种 event_type；空态友好 | M | todo | — |

---

## 1.2 P0 · 测试同窗口配套（Test / CI）

> 设计文档已经写明集成测试矩阵但代码里还没写到。**P0-T-1/2/3 属于 gate 配套；P0-T-4/5 是同窗口质量项，不单独阻塞 2.0 gate**。

| ID | 任务 | DoD | 估时 | 状态 | 落地证据 |
|----|------|-----|------|------|----------|
| **P0-T-1** | **Outbox Worker 集成测试**（`outbox-worker-design.md` §12.2 五件套）：testcontainers-go 起 PG + ES 容器，build tag `//go:build integration` | `TestE2E_Notify_HappyPath` / `TestE2E_DedupBatch` / `TestE2E_RestartReplay` / `TestE2E_ConcurrentAck_NoSeqGap` / `TestE2E_ESDown_Backpressure` 五个用例全部通过；`make test-integration` 一键跑 | L | blocked-by-P0-4 | — |
| **P0-T-2** | **CI 接 `make test-integration`**：GitHub Actions 矩阵跑单测 + integration build tag | PR 中两条流水线都绿；失败回报到 PR check；testcontainers 镜像 cache 命中 | S | blocked-by-P0-T-1 | — |
| **P0-T-3** | **字段提升 backfill 对账脚本**（与 P0-2 配套）：扫存量 `assets / mcap_files`，对比新字段空值率、双写一致率 | `scripts/backfill-audit.sh` 输出 JSON 报告；新字段空值率 < 0.1%；双写一致率 100%；纳入 P0-1 闸口验收 | M | blocked-by-P0-2 | — |
| **P0-T-4** | **前端单测覆盖率守门**：vitest coverage gate（`Frontend/`），核心组件（assets / asset-detail）行覆盖 ≥ 70% | `vitest run --coverage` 跑通；CI 上线最低门槛；不达标禁止合 PR | S | todo | — |
| **P0-T-5** | **`/events` 端点回归脚本**（与 P0-FE-4 配套）：`scripts/test_edge_cases_*.sh` 加 `event_type=algo_*` + cursor 翻页用例，验证算法事件子集与完整事件流的一致性 | bash 脚本一次跑通；事件总数与翻页累计一致；`algo_*` 子集与全集合差等于其他类型计数 | S | todo | — |

---

## 2. P1 · 2.0 启用关键路径

| ID | 任务 | DoD | 估时 | Owner | 状态 | 落地证据 |
|----|------|-----|------|-------|------|----------|
| **P1-1** | **ES `assets` 索引正式启用** | mapping = `outbox-worker-design.md` §5 / `init-index.sh`；接入 outbox 后 ES 文档数与 PG `assets` 一致率 > 99.9% | M | backend | todo | — |
| **P1-2** | **Outbox 监控 + 告警**：Prometheus metrics（`fetch_pending_count / batch_size / publish_lag_seconds / retry_count_total`）+ Grafana dashboard | dashboard 上线；端到端延迟 / 失败率 / cursor lag 三个核心 panel；接 PagerDuty 告警阈值 | M | backend + 运维 | todo | — |
| **P1-3** | **Iceberg REST Catalog 选型 + 部署**：Polaris（首选）/ Lakekeeper | 部署完成；从 PyIceberg / Trino 都能 list / read 一张冒烟表；ADR 写明选型理由 | M | data + 运维 | todo | — |
| **P1-4** | **PyIceberg Bronze 入湖 CronJob**：每 5–10 min 扫 staging parquet → `MERGE INTO bronze.asset_events`（按 `event_seq` 去重） | 24h 跑无丢失；事件去重率验证；compact 周期独立 cron 并跑 | L | data | todo | — |
| **P1-5** | **PyIceberg Compact CronJob**：合并小文件、过期 snapshot 清理（独立周期） | 跑 7 天后 bronze 表小文件数稳定；snapshot 历史保留策略生效 | M | data | todo | — |
| **P1-6** | **Trino 查询接入 `/api/v1/lakehouse/*`**：当前 handler 已注册路由，需对接真实 catalog | sync-status / training-assets / quality-distribution 三个核心端点返回真实数据；E2E 通过 | M | backend + data | todo | — |
| **P1-7** | **事件 schema CI 守门**：PR 改 producer 必须改 schema 版本（`payload_schema_version`） | CI workflow 上线；至少 1 次 major bump 演练通过；规则写进 `CLAUDE.md` | S | backend | todo | — |
| **P1-8** | **Outbox Worker 抽离独立进程**（先内嵌 90 天稳定再切） | 独立 K8s Deployment；backend 关 `OUTBOX_WORKER_ENABLED`；切换无丢事件 | M | backend + 运维 | blocked-by-P0-4 | — |

### 2.1 P1 · 前端

| ID | 任务 | DoD | 估时 | 状态 | 落地证据 |
|----|------|-----|------|------|----------|
| **P1-FE-1** | **Lakehouse Dashboard**：把已注册的 `/api/v1/lakehouse/*` 端点（training-assets / quality-distribution / customer-replay / tag-timeline）至少做成 1 个汇总 dashboard 页面（候选挂在 `AnalyticsPage`） | 4 个数据源中 ≥ 2 个有可视化卡片；ES 故障时降级 PG fallback；与 P1-6 后端联调一次 | M | blocked-by-P1-6 | — |
| **P1-FE-2** | **同步状态可视化**：`/api/v1/lakehouse/sync-status` 接到 `SettingsPage` 或独立"同步监控"页 | 显示 `last_sync_at / postgres_count / iceberg_count / diff_pct`；`status != ok` 时红色徽标 | S | todo | — |
| **P1-FE-3** | **Admin Reindex 入口**：`/admin/search/reindex` 在 SettingsPage 给 admin 角色一个"重建 ES 索引"按钮（带二次确认 + dry_run 选项 + 进度回显） | 仅 admin 可见；dry_run 默认开；调用后显示 `reindexed_assets / failed_assets`；失败列表可下载 | S | blocked-by-P0-5 | — |
| **P1-FE-4** | **保留期 / 过期视图**：列表加"30 天内将过期"快捷过滤（`expire_at:between:now,now+30d`），详情显示 retention badge | 快捷 chip 一键应用；`retention_tier` 用颜色区分（hot/warm/cold/archive）；vitest 覆盖 | S | blocked-by-P0-FE-3 | — |

### 2.2 P1 · 测试

| ID | 任务 | DoD | 估时 | 状态 | 落地证据 |
|----|------|-----|------|------|----------|
| **P1-T-1** | **PG ↔ ES 一致性对账脚本**：跑闸口验收（一致率 > 99.9%）；diff 报告输出 missing/stale/extra | `scripts/es-pg-audit.sh` 跑通；CI nightly job；闸口验收时引用其报告 | M | blocked-by-P1-1 | — |
| **P1-T-2** | **Outbox 性能 / 吞吐压测**：50 events/s × 60s 持续 → 端到端 P99 ≤ 60s（G1） | 基线报告归档；规模上扬到 200 events/s 时 worker 行为可观测 | S | blocked-by-P0-T-1 | — |
| **P1-T-3** | **SDK 集成测试**（启 backend container 跑 pytest E2E） | `sdk/tests/e2e/` 全绿；CI 矩阵的 `sdk-e2e` job 上线 | M | todo | — |
| **P1-T-4** | **事件 schema CI 守门测试**（与 P1-7 配套）：模拟 producer 加字段不 bump version 时 CI 必失败 | 至少 1 次 minor / 1 次 major 模拟用例；规则文档落到 `CLAUDE.md` "提交前 checklist" | S | blocked-by-P1-7 | — |
| **P1-T-5** | **前端 E2E 框架**（Playwright，可后置到 P2 视体量） | 跑通 1 条 happy-path：登录 → 资产列表筛选 → 详情 → 看事件时间线；CI 加 e2e job（可选 nightly） | M | todo | — |

---

## 3. P2 · 2.0 打磨 / 已识别但非阻塞

| ID | 任务 | 触发条件 / 价值 | 估时 | 状态 |
|----|------|----------------|------|------|
| **P2-1** | `POST /api/v1/assets:batch_get`（任意 ID 集合批量详情） | 当出现循环调 `GET /assets/{id}` ≥ 10 次的页面（交付详情 / 工单详情）；详细形态见对话归档「批量查询」分析 | S | todo |
| **P2-2** | ES nested query 同 tag 内多条件捆绑：`tags.scene:source_type:algo` 这类语法 | 算法治理需要"同条 tag 同时满足 key + source"；当前需要后端解析层扩展 | M | todo |
| **P2-3** | ES 写 `tagged_at` 字段，支持「最近一天打的 tag」过滤 | 数据治理 / 抽样需求；mapping + outbox sink projector 都要改 | S | todo |
| **P2-4** | `lifecycle_state` 加 PG `CHECK` 约束（状态机 6 个月无变更后启用） | 防止应用层 bug 写入非法状态；不在 1.0/2.0 关键路径，到达稳定窗口再做 | S | todo |
| **P2-5** | 资产详情 fan-out 单 SQL 化（用 `JSON_AGG`） | 仅当 PG 连接池压力上升时启用；当前 3 条并行已足够 | S | parked |
| **P2-6** | DLQ 表替代 `retry_count` 阈值 | 当 retry 失败事件出现非偶发漏投时启用；MVP 用人工 reindex 兜底 | M | parked |
| **P2-7** | 事件 retention 策略（`asset_events` 滚动归档到 cold tier） | 表行数到达千万级 / 90 天后再评估；不在 2.0 关键路径 | M | parked |
| **P2-FE-1** | tag 高级筛选 UI：基于 `tags.<key>` nested + `source_type / confidence` 的复合 chip 编辑器 | 用户能筛"算法打的、置信度 ≥ 0.9 的 highway tag"；调研后再拍 | M | todo |
| **P2-FE-2** | 通用事件流总览页（跨 asset 的事件流，按 `event_type` 聚合的实时面板） | 仅在出现"运营 / SRE 想看全量事件流"诉求时启动 | M | parked |
| **P2-T-1** | Frontend coverage 提升到 ≥ 85%（核心组件 + 业务页） | P0-T-4 守门稳定 90 天后再加码 | M | parked |
| **P2-T-2** | Outbox chaos 测试（`testcontainers` 注入 PG / ES 故障）| Outbox MVP 跑稳 60 天后再加 | M | parked |
| **P2-8** | **Tag 子域闭环**：补独立 tag 写接口与历史接口（`POST /api/v1/assets/{id}/tags`、`DELETE /api/v1/assets/{id}/tags/{key}`、`GET /api/v1/assets/{id}/tags/history`） | 同事务更新 `asset_tags` + 追加 `tag_upserted / tag_deleted` 到 `asset_events`；历史查询直接走统一事件流；完成后与 `use-cases.md` 的 B1 / B2 / B6 对齐，前端/SDK 不再依赖 `PATCH /assets/{id}` 承担 tag 专项职责 | M | done | `be7f699` (backend API) + `4e4e4f7` (frontend 切专用 tag API) |
| **P2-9** | **彻底移除 `cf_*` 遗留层**：清掉 `cf_meta / cf_algo / cf_tag / cf_files / cf_process` 的运行时依赖与兼容别名，最终 drop 列 / 索引 | 分 4 步完成：1）filter / API / SDK 不再接受 `cf_*` 别名；2）seed / mock / 脚本 / 旧测试改用 typed columns + projection tables；3）确认前端、运维脚本、reindex / backfill 不再读 `cf_*`；4）出 migration 删除 `assets/mcap_files/deliveries` 上的 `cf_*` 列与相关 GIN/表达式索引，并同步 `schemas/pg-phase0.sql`、`schema-reference.md`、`sql.md` | L | todo |

---

## 4. P3 · Phase 2+ 候选（按业务节奏拍板）

| ID | 任务 | 触发条件 |
|----|------|----------|
| **P3-1** | `datasets / dataset_snapshots / training_runs` 表落地 | 第一个真训练任务接入平台时启动 |
| **P3-2** | `catalog_objects / catalog_object_versions` 中立对象注册 | 出现多 provider 数据对象（云对象 / 跨 catalog） |
| **P3-3** | 多模态向量库选型（pgvector / Milvus / Qdrant） | 多模态检索成为核心需求时；不再绑定 Lance |
| **P3-4** | SDK 启动专项（`Client + Asset + tag/algo CRUD` MVP） | 第一个外部脚本场景需要 SDK 时 |
| **P3-5** | Dagster / Argo / Temporal 引入 | 算法 job 链式 / 多步 DAG / lineage 可视化需求出现时 |
| **P3-6** | 跨地域多活、HLC / Snowflake-id 替换 BIGSERIAL | 单点 PG 容量触达瓶颈时；3.x 跨地域阶段 |
| **P3-7** | Iceberg Silver / Gold 派生层与维度表 | Bronze 跑稳后按 BI / 训练需求拍板 |
| **P3-8** | PII / GDPR 删除链路（软删 + `retention_tier` + 物理清理 SLA 30 天） | 客户外部数据接入前；具体方案见 `data-platform-design.md` §7.1.2 + §10.1 R5；关闭条件：独立合规文档发布 + 链路 PoC 通过 |

---

## 5. 不做（明确排除，避免被 PR 重新提）

详细见 `data-platform-design.md` §5.11.2。摘录关键项：

- ❌ 在 PG 重写血缘 / 权限 / 质量系统（用专业组件）
- ❌ Spark / Dagster / Kafka / Debezium / Flink / Daft / Lance（除非命中升级触发条件）
- ❌ 多租户 / RLS（不在本方案范围）
- ❌ 重新启用 Bigtable 运行时

---

## 6. Phase Gate 提醒

| 闸口 | 触发条件 | 验收文档 |
|------|----------|----------|
| **1.0 → 2.0** | P0 全绿 + P1-1 / P1-2 通过 | `data-platform-design.md` §9.1.1 |
| **2.0 → 3.x** | Iceberg 入湖 90 天稳定 + 第一个 dataset 上线 | `data-platform-design.md` §9.1.2 |

每个闸口表含「验收摘要 / 回滚要点」两列，**不通过禁止进下一阶段**。

---

## 7. 怎么用这份清单

1. **每周站会**：扫 P0 / P1 表，更新状态 + Owner，挑出本周要推的 1–2 项。
2. **新任务**：追加到 P0/P1/P2/P3 对应表底部，**按 ID 自增**（P1-9, P1-10…）；不要插队改 ID。
3. **完成项**：把状态改成 `done`，落地证据列贴 PR 链接，**不删除行**。
4. **争议项**：写进 `data-platform-design.md` §10.1 "未决事项" 表，不堆在这里。
5. **拆 PR**：单个任务超过 600 行 diff 时拆子任务（`P0-4a / P0-4b …`），每个子任务独立 PR。

---

## 8. 相关文档索引

| 主题 | 看哪一篇 |
|------|----------|
| 整体架构 / 演进路线 | [`data-platform-design.md`](./data-platform-design.md) §4 / §9 |
| 字段速查 / 上线优先级 | [`schema-reference.md`](./schema-reference.md) |
| API / curl / 工作流 | [`api-guide.md`](./api-guide.md) |
| 算法生命周期 | [`algo-lifecycle-and-data-model.md`](./algo-lifecycle-and-data-model.md) |
| 用例 × 角色矩阵 | [`use-cases.md`](./use-cases.md) |
| Outbox 工程落地 | [`outbox-worker-design.md`](./outbox-worker-design.md) |
| Grace 迁移 | [`grace-migration-notes.md`](./grace-migration-notes.md) |
| 3.x 多模态 | [`3.0-multimodal-design.md`](./3.0-multimodal-design.md) |
