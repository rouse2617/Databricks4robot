# Cyber Databrew — 渐进式开发路线图（v2）

> 版本：2026-05-14 v2  
> 基准：P0-implementation-plan.md v2 + product-review v4 + 已上线链路现状  
> 原则：每个 phase 独立可交付、不依赖后续 phase 的未完成功能
> 评审对齐：详见 [`product-review-2026-05-14.md`](./product-review-2026-05-14.md)

---

## 路线概览

```
P0（本周）      P1（2-4 周）        P2（1-2 月）         P3（季度+）
─────────────────────────────────────────────────────────────────────
Failure Mining  → 日常体验完备化  → 规模化支撑       → 数据飞轮自动化
+ 事件流          + 批量操作        + 治理 & 安全       + 训练集成
+ 当前页重试修复   + 搜索 & 血缘     + 稳定化             + 自动回流
                 + Schema 清理     + SDK 工程化        + 智能标注触发
                 + ES 运维                                      + 模型部署回写
```

每个 phase 结束时平台可独立交付、可演示。

### 执行方法（统一）

- 每个 phase 拆成 3 类任务：`backend` / `frontend` / `ops-docs`
- 每个任务都带 4 个字段：`输入依赖`、`交付物`、`验收命令`、`回滚点`
- 优先串行关键路径（阻塞项），其余并行；默认每天最多 1 个可上线增量
- 所有 runtime 变更遵循当前仓库规则：本地 build → push → deploy → 验证 → 你确认后再 commit/push

### 当前进展快照（2026-05-14）

| 能力 | 状态 | 备注 |
|---|---|---|
| PG → ES 水位观测（API + Prometheus + Settings 可视化） | ✅ 已上线 | `/api/v1/search/sync-progress` + `outbox_*` gauges |
| PG → Bronze 增量入湖（Cloud Run Job + Scheduler） | ✅ 已上线 | `bronze-incremental` 每 5 分钟 |
| Bronze 水位观测（API + Prometheus + Settings 可视化） | ✅ 已上线 | `/api/v1/lakehouse/sync-progress` + `lakehouse_bronze_*` gauges |
| Cloud Monitoring 告警策略落地 | 🟡 未落地 | ADR 已有阈值，策略未 apply |
| P0 Failure Mining（failure_mode + 聚合 + 饼图） | ⏳ 待实现 | 见 P0 章节 |

---

## 部署基线（Tekton PAC，必须遵循）

> 目标：把“代码提交→部署”统一收敛到 PAC，不依赖本地 `gcloud run deploy` 做最终闭环。  
> 权威来源：`.tekton/*.yaml` 与 `.tekton/README.md`；若与本文冲突，以 YAML 为准。

### 触发矩阵（当前生效）

| 场景 | 定义文件 | 触发条件（简写） | 结果 |
|---|---|---|---|
| Dev 后端自动部署 | `.tekton/push-backend-cloudrun-dev.yaml` | `push` 且分支非 `main`，路径命中 `backend/**` or `deploy/cloudrun/**` or 该 YAML | build + push backend 镜像，deploy `cyber-databrew-backend-dev` |
| Dev 前端自动部署 | `.tekton/push-frontend-cloudrun-dev.yaml` | `push` 且分支非 `main`，路径命中 `Frontend/**` or `deploy/cloudrun/**` or 该 YAML | build + push frontend 镜像，deploy `cyber-databrew-frontend-dev` |
| Prod 后端（main） | `.tekton/push-backend-cloudrun-prod.yaml` | `push` 到 `main`，路径命中 `backend/**` / `deploy/cloudrun/**` / 该 YAML | build + push；若 `cyber-databrew-backend-prod` 存在则 deploy，否则仅推镜像 |
| Prod 前端（main） | `.tekton/push-frontend-cloudrun-prod.yaml` | `push` 到 `main`，路径命中 `Frontend/**` / `deploy/cloudrun/**` / 该 YAML | build + push；若 `cyber-databrew-frontend-prod` 存在则 deploy，否则仅推镜像 |
| PR 评论触发（仅后端 dev） | `.tekton/deploy-cloudrun-dev.yaml` | PR 指向 `main`，评论精确 `/deploy-cloudrun-dev`，且路径命中 `backend/**`/`deploy/cloudrun/**` | build + deploy `cyber-databrew-backend-dev` |

### 关键约束（避免误触发）

1. 修改 `deploy/cloudrun/**` 时，通常会**同时触发后端 + 前端 dev** 两条流水线。
2. 只想部署后端时，尽量只改 `backend/**` 或后端 `.tekton` 文件；避免顺手改 `deploy/cloudrun/**`。
3. 只想部署前端时，改动应落在 `Frontend/**`（目录大小写必须是大写 `F`）。
4. PAC 闭环是“`git push` → PipelineRun”；本地 deploy 只能用于临时验证，不作为团队交付基线。

### Agent / 开发者标准动作

1. 先读本次改动对应的 `.tekton/*.yaml`，确认会触发哪条流水线。  
2. 提交并 `git push` 到目标分支（dev: 非 `main`；prod: `main`）。  
3. 观察 `tekton-pipelines` 命名空间中的 PipelineRun：  
   - `kubectl get pipelinerun -n tekton-pipelines`  
   - 失败时看失败 TaskRun（典型是 `build-and-push` 或 `deploy-cloudrun-*`）。  
4. 以 Cloud Run revision + API 验证结果作为“部署成功”证据。

### IAM / Secret 前置（简）

- ServiceAccount：`tekton-builder`（当前 PipelineRun 使用）。  
- 必需权限：Artifact Registry push、Cloud Run deploy、`iam.serviceAccounts.actAs`。  
- 飞书通知 Secret：`tekton-pipelines/feishu-open-notify`（可选；缺失时通知 task 会 `SKIP` 且不阻塞部署）。

---

## P0：飞轮可观测 + 体验底线（本周）

> 详见 [P0-implementation-plan.md](./P0-implementation-plan.md)

| # | 项 | 解决什么问题 | 代价 |
|---|-----|------------|------|
| 1 | failure_mode 埋点 + PG 聚合 + Dashboard 饼图 | 算法失败从"黑盒"变为可聚类分析 | ~160 行 |
| 2 | 算法矩阵"重试当前页失败"按钮可用 | 不再逐个点击 Reset | ~20 行 |
| 3 | 全局事件流默认展示（token 分页） | 打开 /events 不必先输 Asset ID | ~70 行 |

**P0 交付后状态**：算法团队能看"为什么失败"，运维能看"最近发生了什么事"。

**P0 退出标准（必须全部满足）**：

1. `/api/v1/lakehouse/failure-clusters?days=7` 返回非空时，`affected_assets` 为去重资产数（同一 `asset_id` 多次失败不重复计数）。
2. Dashboard 新卡片"近 7 天失败模式分布"可见，空数据时有明确 empty state，不报红。
3. 算法矩阵按钮文案明确为"重试当前页失败"，点击后仅作用于当前可见结果集。
4. `/events` 默认最近 24h，支持 `next_token` 继续翻页，不出现重复页。
5. 后端/前端分别通过现有项目门禁（`go test`、`npm run build`）并在 dev Cloud Run 验证。

### P0 任务拆分（可直接建 Issue）

| Task ID | 任务 | 类型 | 依赖 | 交付物 | 验收 |
|---|---|---|---|---|---|
| P0-T1 | `algo_usecase` 增加 `failure_mode` 推断并写入 `algo_failed` payload | backend | 无 | `algo_usecase.go` + 相关测试 | `go test ./internal/usecase/asset/...` 通过；手动 finish failed 后 `asset_events.event_payload` 含 `failure_mode` |
| P0-T2 | 新增 `GET /api/v1/lakehouse/failure-clusters`（PG 聚合） | backend | P0-T1 | handler + route + OpenAPI | `curl /lakehouse/failure-clusters?days=7` 返回 `affected_assets`；同 asset 多次失败只计 1 |
| P0-T3 | Dashboard 增加“失败模式分布”饼图卡片 | frontend | P0-T2 | `DashboardPage` + `api/lakehouse.ts` | 前端 build 通过；7 天内有失败数据时饼图展示，不报错 |
| P0-T4 | 算法矩阵按钮修复为“重试当前页失败” | frontend | 无 | `AlgoProcessingPage` | 可见失败>0 时按钮可点击；只作用当前页 |
| P0-T5 | `GET /events` 全局事件流 + `next_token` 分页 | backend | 无 | 新增 handler/usecase/repo 分页逻辑 + OpenAPI | 连续两页无重复，`next_token` 可前进 |
| P0-T6 | EventsPage 默认加载最近 24h + “加载更多” | frontend | P0-T5 | `EventsPage` | 打开即有数据；点击加载更多追加且无重复 |
| P0-T7 | 上线与验证记录（含回滚命令） | ops-docs | P0-T1..T6 | 部署记录、验证截图/日志 | dev 环境 6 个验收项全通过，有回滚步骤 |

**P0 建议排期（3 天）**

- Day 1：P0-T1、P0-T2
- Day 2：P0-T3、P0-T4
- Day 3：P0-T5、P0-T6、P0-T7

---

## P1：日常体验完备化（2-4 周）

P1 的目标是让 **每天使用这个平台的人不感到痛苦**——现在有很多"写脚本才能做的事"，P1 把它们搬进 UI。

### 1.1 资产列表批量操作

**现状**：列表有 checkbox，选中多行后无任何批量操作 UI。需要批量交付/批量修改生命周期时只能写脚本。

**做什么**：

| 子项 | 描述 | 后端 | 前端 |
|------|------|------|------|
| 批量交付 | 选中 N 行 → "批量交付" → 输入 customer_id + note → 创建单个 delivery | 新增 `POST /api/v1/deliveries/batch`（复用现有 commit 逻辑，批量 asset_ids；沿用 Idempotency-Key） | 选中行后浮动操作栏 + 弹窗 |

**接口设计**：

```
POST /api/v1/deliveries/batch
Body: { "asset_ids": ["V4ftKjqX", "kK9qIAqG"], "customer_id": "team-b", "owner": "alice" }
Response: { "delivery_id": "uuid", "item_count": 2 }
```

**工作量**：后端 ~100 行，前端 ~80 行 + 测试与幂等校验。**~1-1.5 天。**

> **幂等约束**：`POST /deliveries/batch` 与 `POST /deliveries` 共享 Idempotency-Key namespace。同一 key 不能在两处复用，防止误判。

---

### 1.2 全局搜索 `Cmd+K`

**现状**：想快速定位某个 asset_id / mcap_id / delivery_id 需要导航到对应页面 → 搜索框输入。

**做什么**：

| 子项 | 描述 |
|------|------|
| 快捷键全局浮层 | `Cmd+K` 弹出搜索面板 |
| 多实体搜索 | 输入 8 位字母数字 → 匹配 asset_id + mcap_file_id；输入 UUID → 匹配 delivery_id |
| 分组结果 | "资产 (3)" / "MCAP 文件 (1)" / "交付 (2)" |
| 选中跳转 | Enter 跳转到详情页 |

**数据源（分两步）**：

1. **先复用现有 API**：`GET /api/v1/search/assets?q=` + `GET /api/v1/mcap-files?search=`
2. 若联调后 p95 > 400ms，再补轻量聚合端点：`GET /api/v1/search/quick?q=`

**工作量**：前端 ~120 行（必做）；后端 quick 端点 ~60 行（按性能决定）。**~1-1.5 天。**

---

### 1.3 数据血缘（Lineage）可视化

**现状**：无法回答"segment V4ftKjqX 来自哪个 MCAP → 被哪些算法处理过 → 进了哪些交付 → 给了哪个客户"。

**做什么**：

| 子项 | 数据源 | 描述 |
|------|--------|------|
| 后端血缘 API | PG 三表 JOIN | `GET /api/v1/assets/:id/lineage` → upstream（MCAP）+ downstream（algo_results + deliveries + customers） |
| 前端血缘 Tab | — | 资产详情页新增"血缘"Tab，展示上下链路（先用简单 🡒 箭头列表，不做 DAG 图） |

**接口设计**：

```
GET /api/v1/assets/V4ftKjqX/lineage
Response:
{
  "asset_id": "V4ftKjqX",
  "upstream": {
    "mcap_file_id": "k7ec1qx9",
    "mcap_uri": "gs://bucket/path/k7ec1qx9.mcap",
    "ingest_state": "summarized"
  },
  "downstream": {
    "algo_results": [
      {"algo_name": "hand_tracking", "algo_version": "1.2.0", "status": "failed", "run_id": "..."},
      {"algo_name": "deface", "algo_version": "2.0.0", "status": "ok", "output_uri": "gs://..."}
    ],
    "deliveries": [
      {"delivery_id": "uuid", "customer_id": "team-b", "delivered_at": "2026-05-14T18:06:34Z"}
    ],
    "eval_results": [
      {"eval_name": "hand_tracking_qc", "metric_key": "blur_ratio", "metric_value": 0.03}
    ]
  }
}
```

**工作量**：后端 ~80 行（纯联表查询），前端 ~100 行。**~1.5 天。**

---

### 1.4 Schema 清理（技术债）

**现状**：`asset_algo_events` 无活跃写入；`assets.status` / `seg_type` / `duration_sec` 与权威字段双轨。

**做什么**（分 3 个独立 PR）：

| PR | 内容 | 文件 | 迁移 |
|----|------|------|------|
| 1 | 删 `asset_algo_events` 表 + 关联代码 | 删 3 文件 + 改 2 处 | `025_drop_asset_algo_events.sql` |
| 2 | 删 `assets.status` 列 → 统一 `lifecycle_state` | 改 5 文件 | `026_drop_assets_status.sql` |
| 3 | 删 `seg_type` / `duration_sec` Go struct 字段 | 改 2 文件 | 无（仅在 Go struct 层） |

> 注意：PR 2 需要前端确认 `getLifecycleState()` 已全覆盖（已确认，v3 产品评审中验证过）。

**工作量**：3 个 PR，每个 0.5-1 小时。**~0.5 天。**

---

### 1.5 ES 索引重建僵尸任务清理

**现状**：设置页底部大量 `paused` 任务堆积。

**做什么**：

| 子项 | 描述 |
|------|------|
| TTL 自动过期 | `paused` 超 7 天 → 状态改为 `expired` |
| 手动废弃 | `paused` 行增加"废弃"按钮 |
| 状态 Tab | `进行中 \| 已完成 \| 已暂停 \| 已废弃` |
| 新建保护 | 创建新 reindex 前检查是否有 `running` / `queued` 任务 |

**工作量**：后端 ~40 行 + 1 迁移，前端 ~30 行。**~0.5 天。**

---

### P1 汇总

| # | 项 | 后端 | 前端 | 迁移 | 时间 |
|---|-----|------|------|------|------|
| 1 | 资产列表批量操作 | ~100 行 | ~80 行 | 0 | 1 天 |
| 2 | 全局搜索 Cmd+K | ~60 行 | ~120 行 | 0 | 1.5 天 |
| 3 | 数据血缘 API + Tab | ~80 行 | ~100 行 | 0 | 1.5 天 |
| 4 | Schema 清理 (3 PRs) | ~8 文件 | 0 | 2 | 0.5 天 |
| 5 | ES 僵尸任务清理 | ~40 行 | ~30 行 | 1 | 0.5 天 |
| **合计** | | **~280 行** | **~330 行** | **3** | **~5 天** |

**P1 交付后状态**：平台不需脚本辅助，所有常见操作用 UI 完成。技术债清理完，代码库干净。血缘可追溯。

### P1 任务拆分（按 Epic）

#### Epic P1-E1：高频操作产品化（批量交付 + Cmd+K）

| Task ID | 任务 | 依赖 | 验收 |
|---|---|---|---|
| P1-E1-T1 | `POST /api/v1/deliveries/batch` + 幂等校验 | 无 | 幂等 key 重试返回同一 delivery；批量 item 数正确 |
| P1-E1-T2 | 资产列表批量交付 UI（浮动栏 + 弹窗） | P1-E1-T1 | UI 可选择 N 行并成功创建 delivery |
| P1-E1-T3 | Cmd+K 面板（先复用已有搜索 API） | 无 | 快捷键打开、分组展示、回车跳转 |
| P1-E1-T4 | 仅在性能不达标时补 `/search/quick` | P1-E1-T3 | p95 < 400ms（固定样本） |

#### Epic P1-E2：可追溯性与运维体验（lineage + reindex 清理）

| Task ID | 任务 | 依赖 | 验收 |
|---|---|---|---|
| P1-E2-T1 | `GET /assets/:id/lineage` 后端 | 无 | 返回 upstream/downstream，字段完整 |
| P1-E2-T2 | 资产详情页 Lineage Tab | P1-E2-T1 | 能展示算法、交付、评估链路 |
| P1-E2-T3 | ES 重建任务 TTL + 手动废弃 | 无 | paused 超阈值自动 expired；手动废弃可用 |

#### Epic P1-E3：技术债清理（schema）

| Task ID | 任务 | 依赖 | 验收 |
|---|---|---|---|
| P1-E3-T1 | 移除 `asset_algo_events` 表与代码 | 无 | migration 成功，编译/测试通过 |
| P1-E3-T2 | 移除 `assets.status` 双轨 | P1-E3-T1 | 前后端 lifecycle 显示一致 |
| P1-E3-T3 | 清理 `seg_type` / `duration_sec` struct 残留 | P1-E3-T2 | 无编译告警/无运行影响 |

---

## P2：规模化支撑（1-2 月）

P2 解决"不止一个团队在用"时的问题——权限、多租户、SDK 稳定性。

### 2.1 多租户隔离

**现状**：`tenant_id` 在所有表里预留了，但查询未强制过滤。

**做什么**：

| 子项 | 描述 |
|------|------|
| Middleware 解析 | `X-Grace-Token` → 查 token→tenant 映射表（简单方案，等 OIDC 后再切） |
| Repository 过滤 | 所有 `SELECT` 追加 `AND tenant_id = $ctxTenant` |
| 写路径约束 | `INSERT/UPDATE/DELETE` + outbox 事件写入同样强制 tenant 绑定，禁止跨租户写入 |
| 存量数据迁移 | 所有 `tenant_id IS NULL` 的行 → 单次 migration 写入 `'default'` |
| 索引补齐 | `CREATE INDEX idx_assets_tenant_lifecycle ON assets(tenant_id, lifecycle_state)`，覆盖高频复合查询 |
| 前端租户切换（可选） | 默认先不开放任意切租户；仅 admin debug 场景可见，避免误读跨租户数据 |

**为什么不等 OIDC 一起做**：OIDC 集成需要域名 + IAP 配置 + cookie 迁移，周期长。先用 token→tenant 映射提供隔离能力，后续切 OIDC 时只改 auth 层。

**工作量**：后端 ~300-500 行（middleware + handlers/usecases/repos 全链路过滤）+ 1 迁移，前端 ~50 行。**~5-7 天。**

### P2 任务拆分（分 3 条并行流）

#### Stream P2-S1：租户隔离闭环（最高优先）

| Task ID | 任务 | 验收 |
|---|---|---|
| P2-S1-T1 | Token→tenant 映射 + middleware 注入 ctx | 所有请求日志可见 tenant_id |
| P2-S1-T2 | 读路径 tenant 过滤（repo 层） | 跨租户读请求返回空或 403 |
| P2-S1-T3 | 写路径 tenant 约束（含 outbox） | 无法跨租户写；事件 tenant_id 正确 |
| P2-S1-T4 | 存量迁移 + 索引 | `tenant_id IS NULL` → `'default'`；复合索引生效 |
| P2-S1-T5 | 租户隔离回归测试（最小 e2e） | 两个测试租户互不可见 |

#### Stream P2-S2：稳定性/SLO（与 S1 并行）

| Task ID | 任务 | 验收 |
|---|---|---|
| P2-S2-T1 | ES + Bronze 告警策略落地到 Cloud Monitoring | 触发测试告警可收到通知 |
| P2-S2-T2 | `readyz` 健康探针（PG+BigQuery） | 健康/异常返回码符合预期 |
| P2-S2-T3 | 连接池与 lag 仪表盘标准化 | 有固定 dashboard URL + runbook |

#### Stream P2-S3：开发效率（低风险快收口）

| Task ID | 任务 | 验收 |
|---|---|---|
| P2-S3-T1 | SDK 类型自动生成脚本 | `make sdk-generate` 可重现 |
| P2-S3-T2 | CI 比较 OpenAPI vs SDK types | drift 时 CI 明确失败/警告 |
| P2-S3-T3 | 资产列表高级筛选增强 | 排序/保存筛选/导出全部可用 |

---

### 2.2 资产列表高级筛选增强

> 详见 [Stream P2-S3](#stream-p2-s3开发效率低风险快收口) · Task `P2-S3-T3`

### 2.3 SDK OpenAPI 自动生成

> 详见 [Stream P2-S3](#stream-p2-s3开发效率低风险快收口) · Task `P2-S3-T1` / `P2-S3-T2`

### 2.4 OIDC 认证（可选 — 取决于组织时间线）

**现状**：`StaticTokenAuth` 单 token 明文比对。Google IAP 集成 + RBAC 骨架（admin/operator/viewer）。

**为什么"可选"**：OIDC 取决于组织 IAP 配置是否就绪、域名是否已 CNAME。后端代码改动不大，但环境配置周期不可控。

**工作量**：后端 ~150 行（middleware），前端 ~50 行（login flow）。**~2 天（编码）+ 配置协调。**

### 2.5 稳定性与监控

> 详见 [Stream P2-S2](#stream-p2-s2稳定性slo与-s1-并行) · Task `P2-S2-T1` ~ `P2-S2-T3`

**最小 SLO（P2 收口时）**：

- Bronze ingest freshness（`bronze_stale_seconds`）P95 < 10 分钟
- Search sync consumer lag（`outbox_consumer_lag`）P95 < 1,000
- `/api/v1/search/sync-progress` 与 `/api/v1/lakehouse/sync-progress` 可用性 > 99.9%

---

### P2 汇总

| # | 项 | 任务 | 后端 | 前端 | 迁移 | 时间 |
|---|-----|------|------|------|------|------|
| 1 | 多租户隔离 | P2-S1 | ~300-500 行 | ~50 行 | 1 | 5-7 天 |
| 2 | 稳定性监控 | P2-S2 | ~80 行 | 0 | 0 | 3 天 |
| 3 | SDK + 筛选增强 | P2-S3 | ~100 行 | ~180 行 | 0 | 3 天 |
| 4 | OIDC 认证（可选） | — | ~150 行 | ~50 行 | 0 | 2 天+ |
| **合计** | | | **~630-830 行** | **~280 行** | **1** | **~11-13 天** |

**P2 交付后状态**：多团队可用，权限隔离，SDK 契约有 CI 保护。平台从"内部工具"级别提升到"可对外交付"级别。

---

## P3：数据飞轮自动化（季度+）

P3 不是"加功能"，而是"改行为"——从手动操作变为自动触发。这里只列方向和接口，实现细节需要和算法/标注/采集团队联合设计。

### 3.1 训练集物化

**目标**：让 `datasets` / `dataset_snapshots` 表真正可用。

```
当前：两张空表，无写入路径
目标：
  1. 资产管理页选中 N 个 asset → "创建训练集" → 写入 datasets + dataset_snapshots
  2. snapshot 内含 immutable asset_ids[] + 当时 tag/algo 状态快照
  3. 训练平台调 API 读取 snapshot → 下载 GCS 数据 → 训练
  4. 训练完成后回调 POST /api/v1/training-runs/:id/finish → 写入 artifact_uri + metrics
```

**新增 API**：

```
POST   /api/v1/datasets                       ← 创建训练集
POST   /api/v1/datasets/:id/snapshots         ← Freeze 快照
GET    /api/v1/datasets/:id/snapshots/:sid    ← 读取快照（asset 列表 + 元信息）
POST   /api/v1/training-runs/:id/finish       ← 训练完成回写
GET    /api/v1/training-runs?model_name=...   ← 查询历史训练 run
```

### 3.2 失败→标注自动触发（可选 — 取决于标注团队工具链）

**目标**：`failure_cluster` 出现时，一键发送到标注队列。

```
当前：P0 后 failure_mode 可见，但下游动作靠人
目标：
  1. Dashboard 饼图每个 failure_mode 旁加"处理"按钮
  2. "发送到标注"→ 调用标注团队 API（webhook），传递 affected asset_ids
  3. 标注结果回到 actions 表（已有 API）后，failure_cluster 状态从 open→resolved
```

**本平台只做**：提供 `affected_asset_ids` 列表 + 调用外部 webhook。标注流程本身不在此实现。

### 3.3 算法版本回归检测

**目标**：新版本算法上线后，自动对比旧版本指标，发现退化。

```
数据源：asset_algo_latest（版本间对比）+ asset_metrics（同一 asset 的两次 eval）
接口：GET /api/v1/lakehouse/algo-regression?algo_name=hand_tracking&old_version=1.2.0&new_version=1.3.0
BigQuery 聚合：对比两个版本的 quality 分布 / failure_rate / avg_metrics
```

### 3.4 模型部署回写

**目标**：ML 平台部署新模型后，回写 `training_runs.status=deployed`，触发对应算法的 `asset_algo_latest` 中版本自动更新（`blocked → pending` cascade）。

**新增 API**：`POST /api/v1/training-runs/:id/deploy` → 写 deployment 事件 → 自动 unblock 下游依赖该算法的其他算法。

### 3.5 自动化重采触发（远期）

**目标**：`failure_cluster` 检测到 `sensor_fault` 模式 → 自动创建 `collection_task`。

这个依赖采集团队有 `collection_task` API。本平台只提供 webhook 调用能力。

---

## P3 汇总

| # | 项 | 核心接口 | 依赖 |
|---|-----|---------|------|
| 1 | 训练集物化 | `datasets` + `snapshots` CRUD | P2 多租户 |
| 2 | 失败→标注触发 | webhook 调用 | P0 failure_clusters + 标注团队 API |
| 3 | 算法版本回归 | `/lakehouse/algo-regression` | P0 failure_mode + Lakehouse Silver 层 |
| 4 | 模型部署回写 | `POST /training-runs/:id/deploy` | P3.1 训练集 |
| 5 | 自动重采触发 | webhook → collection_task | P1 血缘 + 采集团队 API |

### P3 任务拆分（里程碑制）

#### M3.1 训练集资产化（必做里程碑）

| Task ID | 任务 | 验收 |
|---|---|---|
| P3-M31-T1 | `datasets` / `snapshots` API + 权限校验 | 可创建、冻结、读取 immutable snapshot |
| P3-M31-T2 | 资产页“创建训练集”UI | N 个 asset 成功进入 snapshot |
| P3-M31-T3 | 训练完成回写 `training-runs/:id/finish` | artifact/metrics 可查询 |

#### M3.2 飞轮动作自动化（可选里程碑）

| Task ID | 任务 | 验收 |
|---|---|---|
| P3-M32-T1 | failure cluster → 标注 webhook | 可发送 `affected_asset_ids` 并记录回执 |
| P3-M32-T2 | algo regression API | old/new 版本差异指标可比对 |
| P3-M32-T3 | deployment 回写 + 依赖解锁 | deploy 回写后相关任务状态推进 |

---

## 总览：Phase × 时间 × 平台能力

```
      P0（本周）       P1（月内）       P2（2月内）        P3（季度+）
      ───────────────────────────────────────────────────────────
上线   可内测           可内部交付       可对外交付          可规模化运营

核心   飞轮有眼看       日常不痛          安全 & 稳定         自动化飞轮
能力   + 事件可见       批量/搜索/血缘     多租户/OIDC        训练/回归/回写
       + 按钮可用       + 技术债清理       + SDK 契约         + 智能触发

资产   268k 在线        ←←← 持续增长 ←←←
日均   ？               ？                ？                 ？
```

### Now / Next / Later（执行优先级）

| 优先级 | 任务 | 原因 |
|---|---|---|
| Now | P0-T1 ~ P0-T7 | 直接补齐 Failure Mining 主链路，业务感知最强 |
| Next | P2-S2-T1 告警落地（提前到 P1 前半） | 已有指标但无自动告警，属于高风险缺口 |
| Next | P1-E1（批量交付 + Cmd+K） | 日常效率收益最高 |
| Later | P2-S1 全量多租户闭环 | 影响面最大，需单独测试窗口 |
| Later | P3 M3.2 自动化动作 | 依赖外部团队接口成熟度 |

---

## 附录：完整 API 路线图

| Phase | 端点 | 状态 |
|-------|------|------|
| **已上线基线** | `GET /search/sync-progress` | 🟢 |
| **已上线基线** | `GET /lakehouse/sync-progress` | 🟢 |
| **P0** | `GET /lakehouse/failure-clusters` | 🔴 |
| **P0** | `GET /events` | 🔴 |
| **P1** | `POST /deliveries/batch` | 🔴 |
| **P1** | `GET /search/quick?q=` | 🔴 |
| **P1** | `GET /assets/:id/lineage` | 🔴 |
| **P1** | Schema 清理 migration × 2 | 🔴 |
| **P2** | tenant_id 强制过滤（所有端点） | 🔴 |
| **P2** | `GET /assets?sort_by=delivery_count&sort_order=desc` | 🔴 |
| **P3** | `POST /datasets` + `/datasets/:id/snapshots` | ⚪ |
| **P3** | `POST /training-runs/:id/finish` + `/deploy` | ⚪ |
| **P3** | `GET /lakehouse/algo-regression` | ⚪ |

---

## 附录：开发约束与测试指南

> 权威来源：[`CLAUDE.md`](../../CLAUDE.md)、Team process（Data Infra 软件开发规范）

### 代码质量门禁（每层独立通过）

| 层 | 命令 | 必须通过 |
|----|------|---------|
| Go backend | `cd backend && make fmt && make vet && go test ./...` | ✅ |
| Frontend | `cd Frontend && npm run lint && npm run build` | ✅ |
| Python SDK | `cd sdk && uv run ruff check src/ && uv run pytest tests/unit/` | ✅ |
| OpenAPI | `api/openapi.yaml` 无语法错误、无未引用 schema | ✅ |

### 开发节奏：deploy-before-commit（推荐）

每个 Task 按以下顺序执行，确保在线验证通过才 commit：

```
1. 本地 build + lint + test（质量门禁全部通过）
2. 运行部署脚本（后端/前端 → Cloud Run dev）
3. 在线 smoke test
   后端：curl -H "X-Grace-Token: dev-token" "<endpoint>"
   前端：Chrome MCP 浏览器走查（navigate → snapshot → 交互验证）
4. 你确认 OK
5. git commit + push（Tekton CI 自动再部署一次，覆盖即可）
```

**为什么先部署再 commit**：
- 在 Git 历史里不会留下一堆"修部署错误"的 follow-up commit
- 在线环境比本地更真实（PG 数据量、BigQuery 状态、前端 API 代理）
- commit 之前的验证是最终验证

**部署命令速查**：

| 层 | 脚本 | 说明 |
|----|------|------|
| Backend | `bash deploy/cloudrun/backend-dev.sh` | 构建 Docker 镜像 → push → 部署到 Cloud Run（带 `--set-env-vars` 会碰到 Secret Manager 权限问题，Tekton 不受影响） |
| Frontend | `bash deploy/cloudrun/frontend-dev.sh` | 同上 |

> **Cloud Run 部署权限提示**：手动跑 `backend-dev.sh` 可能因 Service Account 缺少 `roles/secretmanager.secretAccessor` 而失败（revision 00059 引入的问题）。此时走 Tekton CI 即可——它的部署命令不带 `--set-secrets`，不会触发权限错误。

**在线验证端点**：

```
# 后端 dev
curl -s -H "X-Grace-Token: dev-token" \
  "https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app/api/v1/<path>"

# 前端 dev
https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app/<page>
```

### Git 约定

- **Commit**: [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) — `feat:` / `fix:` / `refactor:` / `docs:` / `test:` / `chore:`
- **Merge**: Squash merge → `main`；PR 合并前刷新 description 以反映全量变更
- **Review**: ≥1 人类 reviewer；高风险变更 ≥2；Gemini/bot `High` 项必须有 reply

### 契约纪律

- **新/改 API → 同步 `api/openapi.yaml`**。SDK types 未自动生成前，手写类型需与 OpenAPI 一致
- **修改 `asset_events` 的事件类型 → 同步 `backend/schemas/events/` JSON Schema + `registry.json`**
  - 加可选字段：直接改现有 `.vN.json`；加必填字段：新建 `.vN+1.json`
- **Postgres DDL → 按序放 `backend/migrations/`**；同步更新 `docs/review/sql.md`

### 新增依赖

- Go: `go mod tidy`
- Frontend: `npm install` → `package-lock.json` 必须提交
- Python SDK: `uv add <pkg>` → `pyproject.toml` + `uv.lock`

### 安全红线

- 仍然用 `X-Grace-Token`（Phase 0），**禁止**新增 bypass auth 的公开端点
- 内部端点放 `/internal/*`，管理端点放 `/admin/*`
- Idempotency-Key：`POST /deliveries` 和 `POST /deliveries/batch` 共享 namespace
- 错误响应统一信封：`{ "code": "...", "message": "...", "request_id": "...", "details": {...} }`

---

### 前端测试：Chrome DevTools MCP 验证流程

每个 Phase 的 Task 完成后，按以下步骤在 dev Cloud Run 做浏览器实测。

#### 前置条件

dev 环境 URL：

```
前端：https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app
后端：https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app
鉴权：X-Grace-Token: <dev GRACE_TOKEN>
```

#### 标准验证流程（单页面）

```
1. mcp__chrome-devtools__navigate_page → url = "{前端URL}/{页面路径}"
2. mcp__chrome-devtools__take_snapshot                          ← 检查页面渲染 + 无障碍树
3. 如页面含表单 → mcp__chrome-devtools__fill / fill_form       ← 输入数据
4. 如需交互    → mcp__chrome-devtools__click / press_key       ← 操作
5. mcp__chrome-devtools__take_snapshot                          ← 验证操作结果
6. 如有错误    → mcp__chrome-devtools__list_console_messages   ← 抓 JS 错误
7. 如需截图    → mcp__chrome-devtools__take_screenshot          ← 留证
```

#### P0 验证脚本

| Task | 验证步骤 |
|------|---------|
| **P0-T3** Dashboard 饼图 | 1. navigate → `/dashboard` 2. take_snapshot → 确认"近 7 天失败模式分布"卡片出现 3. 检查：有数据时饼图展示；无数据时 Empty state 不报红 4. list_console_messages → 无 JS error |
| **P0-T4** 算法矩阵按钮 | 1. navigate → `/algo` 2. take_snapshot → 确认"重试当前页失败"按钮文案正确 3. 通过筛选调到有失败行的页面 → 按钮应显示计数且可点击 4. click 按钮 → 确认弹窗出现 |
| **P0-T6** 事件流默认加载 | 1. navigate → `/events` 2. take_snapshot → 确认无"请先输入 Asset ID"提示，直接展示事件列表 3. 确认"加载更多"按钮存在 4. click → 追加数据，无重复 5. 输入 Asset ID → 确认可切换为单资产视图书签 |
| **P0-T1/T2** 后端 | 用 curl 验证（无法用浏览器直测）：`curl -H "X-Grace-Token: $token" "$BACKEND/api/v1/lakehouse/failure-clusters?days=7"` → 返回 JSON 含 `items[]` |

#### P1 验证脚本

| Task | 验证步骤 |
|------|---------|
| **P1-E1-T2** 批量交付 | 1. navigate → `/assets` 2. 勾选 ≥2 行 3. take_snapshot → 确认浮动操作栏出现"批量交付" 4. click → 弹窗填 customer_id → confirm 5. navigate → `/deliveries` → 确认新 delivery 包含选中资产 |
| **P1-E1-T3** Cmd+K | 1. navigate → 任意页面 2. press_key → `Meta+K` 3. take_snapshot → 确认搜索面板弹出 4. fill input → 输入已知 asset_id 5. take_snapshot → 分组结果出现 6. press_key → `Enter` → 跳转到详情页 |
| **P1-E2-T2** 血缘 Tab | 1. navigate → `/assets/{已知 asset_id}` 2. click → "血缘" Tab 3. take_snapshot → 确认 upstream MCAP + downstream algo/delivery/eval 链路展示 |
| **P1-E2-T3** ES 清理 | 1. navigate → `/settings` 2. scroll → 底部任务列表 3. take_snapshot → 确认状态 Tab 分组；paused 任务可废弃 |

#### P2 验证脚本

| Task | 验证步骤 |
|------|---------|
| **P2-S1-T2** 租户隔离 | 1. 用 token-A 请求 → 确认仅看到 tenant-A 数据 2. 用 token-A 请求跨 tenant 的 asset_id → 404 或 403 3. 用 token-B 请求 → 确认仅看到 tenant-B 数据 |
| **P2-S2-T1** 告警 | 手动触发 outbox lag > 1000 → 确认 Cloud Monitoring 通知到达 |
| **P2-S2-T2** readyz | `curl $BACKEND/readyz` → 200（PG+BigQuery ok）；kill PG → 503 |

#### 截图留证约定

每个 Task 验证通过后截屏一张，存放在 `docs/review/screenshots/{phase}-{task-id}.png`：

```
docs/review/screenshots/P0-T3-dashboard-pie.png
docs/review/screenshots/P0-T4-algo-retry.png
docs/review/screenshots/P0-T6-events-default.png
docs/review/screenshots/P1-E1-T2-batch-delivery.png
...
```

截图命令：`mcp__chrome-devtools__take_screenshot → filePath = "..."`

---

### 环境与部署

| 环境 | 后端 | 前端 | 用途 |
|------|------|------|------|
| **dev** | `cyber-databrew-backend-dev-...` | `cyber-databrew-frontend-dev-...` | 日常开发验证、PR 预览 |
| **prod** | `cyber-databrew-backend-prod-...` | `cyber-databrew-frontend-prod-...` | 生产 |

- 部署方式：Cloud Run via Tekton pipeline（`deploy/cloudrun/`）
- Bronze 增量：Cloud Run Job `bronze-incremental`，Scheduler 每 5 分钟触发
- 本地全栈：`deploy/local` 提供 PG + ES + 后端 + 前端

---

### P0 完成后必须交付的制品

除代码外，每个 Phase 结束需交付：

1. **部署记录**：dev 环境已验证的 commit hash + 部署时间
2. **验证截图**：`docs/review/screenshots/`（至少覆盖该 Phase 所有 Task）
3. **回滚步骤**：每个 Task 的回滚命令（reverse migration / revert commit）
4. **OpenAPI diff**：当前 `api/openapi.yaml` 与上一版本的 diff（如该 Phase 含 API 变更）
