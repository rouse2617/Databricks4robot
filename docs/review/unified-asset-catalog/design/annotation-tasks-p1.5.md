# 标注任务（annotation_tasks）设计 — P1.5

| 字段 | 值 |
|------|----|
| 状态 | **Draft / 占位**（决策 5 必建，详细设计 P1.5 落地）|
| 关联 | `../README.md` §3（实体清单 + P1.5 启用）/ `asset-tagging.md` / `customers-and-deliveries.md` |
| 决策来源 | rev.12 决策 5（必建 + B 规模 + 业务参考层，但延后细节）|

---

## 1. 业务问题（Why）

### 1.1 当前现网状态

- **现网无** `annotation_tasks` 表
- 人工标注产物通过 `actions.source_type='human'` + `source_name=labeler_id` 表达
- 标注任务的「派单 / 分配 / SLA / 审核 / 批次管理」无平台支持
- 全靠 Excel / Notion / 飞书表格人工管理

### 1.2 业务规模（决策 5 锁定）

- **B 规模**：10-50 标注员（专职标注队）
- 不是 < 10 人偶尔验证（A）
- 不是 100+ 外包 BPO（C，那时该上 Label Studio 等第三方）

### 1.3 痛点

- 「这批 segment 该派给谁标？」 → 手工分配
- 「labeler_007 标了多少？质量怎么样？」 → 无平台 metric
- 「这个客户 cust_alpha 的标注需求 SLA 还有几天？」 → 无追踪
- 「同一资产派给 2 个人标了？」 → 不知道
- 「标注完成后是否要审核？」 → 无平台审核流

---

## 2. 设计原则（How）

### 2.1 annotation_tasks 是「业务参考」类一等实体（而非「执行事件」）

按 `../README.md` §1 4 类一等实体分层：

| 选择 | 含义 | 决策 5 选择 |
|------|------|------------|
| **业务参考层（task definition）** | 任务定义：长期实体，可分配/重分配/审核；执行状态用状态机表达 | ✓ rev.12 决策 5 |
| 执行事件层（task execution）| 任务执行：一次性事件，每次提交是新行 | ✗ |

**特征：**
- 长期实体（pending → assigned → done → 数据保留长期）
- 状态机表达执行历史（不另起 execution 表）
- 类比 GitHub Issue：1 Issue + 状态变更 + 多条 comment，但 Issue 本身是参考实体

**为什么不归「执行事件」（容易误判）：**

> 「有严格状态机 = 执行事件」是直觉误判。判别标准是「**是否可改 / 是否一次性写完只读**」：
>
> | 维度 | annotation_tasks | algo_runs（典型执行事件）|
> |------|----------------|------------------------|
> | 创建后可改字段 | ✓ assigned_to / status / reviewer / due_at 都可改 | ✗ 写完后业务字段不改，仅 status 推进 |
> | 可重派 / 重审 / cancel | ✓ rejected → 自动回 pending → 可换 labeler 重派 | ✗ 一次跑完就结束（retry 是新 run，不复用 run_id） |
> | 数据保留语义 | 长期实体（可能跨月/跨季） | 历史归档（按 retention 清理） |
> | 跨多次执行 | ✓ 单 task 可被多个 labeler 顺次处理（reject 后换人）| ✗ 单 run 只对应一次实际执行 |
>
> → annotation_tasks **是「可变的、长期的、被反复引用」**业务参考实体。状态机只是表达「这个长期实体当前在哪个阶段」的工具，跟「事件」无关。
>
> 状态机推进逻辑详见 §4 API 契约 + §5 状态机推进规则（每次状态变更对应一个 API endpoint）。

### 2.2 task 与 actions 的关系

```
annotation_tasks（业务参考，「派单」）
    │ task definition
    ├─ task_kind: action_label / time_window / quality_check
    ├─ target_asset_id: 哪个 asset 上标
    ├─ assigned_to: labeler_id
    ├─ status: pending → assigned → in_progress → submitted → approved/rejected
    │
    └─► 提交后产出
        │
        ▼
actions（资产，「成果」）
    ├─ source_type='human', source_name=labeler_id
    ├─ task_id 外键 ──► annotation_tasks（反向追溯）
    └─ primary_label=grasp, start_ns/end_ns, ...
```

→ task = **「派单」**（业务参考），action = **「成果」**（资产）。
→ 提交流程**同事务原子**：写 actions 同时 update task.status='submitted'。

### 2.3 1 对 1 分派（不是多人协作）

- 单 task 只分给 1 个 labeler（`assigned_to` 单值）
- 多人协作 = 派多个 task（每人一个，target 一样）
- 任务粒度细：每个 task 是「一个 segment 内的一个动作类型」，天然 1 人能做完

---

## 3. 数据模型（DDL，P1.5 落地）

### 3.1 annotation_tasks 表

```sql
CREATE TABLE annotation_tasks (
  task_id            TEXT PRIMARY KEY CHECK (task_id ~ '^[0-9A-Za-z]{8}$'),

  -- 任务定义
  task_kind          TEXT NOT NULL,
                     -- action_label | time_window | quality_check | tag_review
  task_description   TEXT,
  instruction_uri    TEXT,                            -- 标注指南文档（OSS / Notion URL）

  -- 目标资产
  target_asset_id    TEXT NOT NULL REFERENCES assets(asset_id),
                     -- 通常是 segment 或 clip；标注产出会作为子资产挂上去
  target_time_range  INT8RANGE,                        -- 可选：限定 segment 内子时间窗（ns）

  -- 分派
  assigned_to        TEXT,                             -- labeler user_id（NULL = 待领取）
  assigned_at        TIMESTAMPTZ,
  assigned_by        TEXT,                             -- 派单人 user_id
  priority           INT NOT NULL DEFAULT 5,           -- 1=最高，10=最低
  due_at             TIMESTAMPTZ,

  -- 状态机
  status             TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','assigned','in_progress','submitted','approved','rejected','cancelled')),

  -- 审核（可选，配置驱动）
  reviewer           TEXT,                             -- QA 审核员 user_id
  reviewed_at        TIMESTAMPTZ,
  review_comment     TEXT,

  -- 批次（可选）
  batch_id           TEXT,                             -- 同批次的 task 共享，便于批量管理

  -- 客户/项目关联
  customer_id        TEXT REFERENCES customers(customer_id),  -- 如果这批数据是给特定客户
  project_id         TEXT,
  tenant_id          TEXT,

  -- 扩展
  metadata           JSONB NOT NULL DEFAULT '{}',
                     -- {"triggered_by_algo_run":"R001", "expected_label_count":3, ...}
  extra              JSONB NOT NULL DEFAULT '{}',

  -- 系统
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         TEXT NOT NULL,                    -- 派单人 user_id
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version        BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX idx_atasks_assigned  ON annotation_tasks(assigned_to, status) WHERE status IN ('assigned','in_progress');
CREATE INDEX idx_atasks_status    ON annotation_tasks(status);
CREATE INDEX idx_atasks_target    ON annotation_tasks(target_asset_id);
CREATE INDEX idx_atasks_due       ON annotation_tasks(due_at) WHERE due_at IS NOT NULL AND status IN ('pending','assigned','in_progress');
CREATE INDEX idx_atasks_batch     ON annotation_tasks(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_atasks_customer  ON annotation_tasks(customer_id) WHERE customer_id IS NOT NULL;
```

### 3.2 actions 加 task_id 外键

```sql
ALTER TABLE actions
  ADD COLUMN task_id TEXT REFERENCES annotation_tasks(task_id) ON DELETE SET NULL;

CREATE INDEX idx_actions_task ON actions(task_id) WHERE task_id IS NOT NULL;
```

---

## 4. 状态机

```
pending → assigned → in_progress → submitted ──► approved（终态）
                                       │
                                       └──────► rejected
                                                   │
                                                   └─► pending（自动回，可重派）

cancelled 可从任意非终态进入（人工取消）
```

| 状态 | 进入条件 | 触发方 |
|------|---------|------|
| `pending` | 创建 / 未分派 | 派单人 |
| `assigned` | 派给某个 labeler（assigned_to 填）| 派单人 / 自动分配 |
| `in_progress` | labeler 开始标注 | labeler 调 start |
| `submitted` | labeler 提交（同事务写 actions）| labeler |
| `approved` | reviewer 审核通过 | reviewer |
| `rejected` | reviewer 审核拒绝（带 review_comment）| reviewer |
| `cancelled` | 人工取消 | 派单人 |

**关键决策（Q5.7-5.8）：**
- **可选审核**：业务可配；不是每个 task 都强制审核（B 规模下成本高）
- **rejected 自动回 pending**：可重新派给同人或其他人（不需要人工手动改）

---

## 5. API 契约（P1.5）

### 5.1 创建 task

```text
POST /api/v1/annotation-tasks
Idempotency-Key: <uuid>
{
  "task_kind": "action_label",
  "task_description": "标注 segment_X 内的所有 grasp 动作",
  "target_asset_id": "seg_X001",
  "target_time_range": "[100000000, 5000000000]",
  "priority": 3,
  "due_at": "2026-05-28T18:00:00Z",
  "batch_id": "batch_2026_05_grasp_v2",
  "customer_id": "cust_alpha_robotics",
  "metadata": {
    "triggered_by_algo_run": "R002",        ← 算法低 confidence 触发人审
    "expected_label_count": 3
  }
}
```

### 5.2 分派 task

```text
PATCH /api/v1/annotation-tasks/{task_id}/assign
{
  "assigned_to": "labeler_007"
}
→ status: pending → assigned, assigned_at=now()

POST /api/v1/annotation-tasks/{task_id}/start
→ status: assigned → in_progress
```

### 5.3 提交标注（同事务原子）

```text
POST /api/v1/annotation-tasks/{task_id}/submit
Idempotency-Key: <uuid>
{
  "actions": [
    {"start_ns": 100000000, "end_ns": 200000000, "primary_label": "grasp"},
    {"start_ns": 300000000, "end_ns": 450000000, "primary_label": "pickup"}
  ]
}

→ 同事务原子（Q5.6 决策）：
  INSERT actions x2 (source_type='human', source_name='labeler_007', task_id='T001')
  UPDATE annotation_tasks SET status='submitted', updated_at=now()
  INSERT asset_events (type='task_submitted', actor='labeler:labeler_007', ...)
```

### 5.4 审核

```text
POST /api/v1/annotation-tasks/{task_id}/review
{
  "verdict": "approved" | "rejected",
  "review_comment": "动作时间窗精度不够"
}

→ approved:
   UPDATE annotation_tasks SET status='approved', reviewer=..., reviewed_at=now()

→ rejected:
   UPDATE annotation_tasks SET status='rejected', reviewer=..., reviewed_at=now(), review_comment='...'
   # rev.12 Q5.8 决策：自动回 pending
   UPDATE annotation_tasks SET status='pending', assigned_to=NULL  -- 可重派
   # 同时撤销之前 submitted 时写的 actions（soft delete 或 mark obsolete）
```

### 5.5 查询

```text
GET /api/v1/annotation-tasks/{task_id}                  ← 详情
GET /api/v1/annotation-tasks?assigned_to=labeler_007&status=in_progress  ← labeler 工作清单
GET /api/v1/annotation-tasks?customer_id=cust_alpha&status=pending&priority<=3  ← 派单视图
GET /api/v1/annotation-tasks?batch_id=batch_2026_05_grasp_v2  ← 批次进度
```

---

## 6. 业务场景 walkthrough

### 6.1 算法低 confidence → 触发人审

```text
① algo: action_detector@2.0 跑出一批 action
   其中 47 个 confidence < 0.7

② AlgoUsecase finish 后触发 ActionHandler（QualityGate 类）：
   对 47 个低 confidence action 自动创建 annotation_tasks：
   for action in low_confidence:
       POST /annotation-tasks {
           task_kind: 'quality_check',
           target_asset_id: action.asset_id,
           metadata: {triggered_by_algo_run: 'R002', target_action_id: action.action_id}
       }

③ 派单人 alice 看到 47 个 pending task：
   POST /annotation-tasks/T001/assign {assigned_to: 'labeler_007'}
   ... (批量分派)

④ labeler 标注 → POST /submit → status=submitted

⑤ reviewer 审核（可选）→ approved → 算法重训用 / 客户验收用
```

### 6.2 客户专属标注任务

```text
cust_alpha 要求 100 个特定场景的 grasp 标注：

POST /annotation-tasks（批量创建）
{
  customer_id: 'cust_alpha_robotics',
  batch_id: 'cust_alpha_grasp_v1',
  task_kind: 'action_label',
  target_asset_id: <各 segment>,
  ...
} x100

监控批次进度：
GET /annotation-tasks?batch_id=cust_alpha_grasp_v1
→ {total:100, pending:30, in_progress:20, submitted:30, approved:15, rejected:5}

完成后给客户出 delivery：
POST /deliveries {customer_id: 'cust_alpha_robotics'}
POST /deliveries/{id}/items {asset_id: <100 个新 action>}
POST /commit
```

### 6.3 reject 重派流程

```text
① labeler_007 标了 T001（提交 3 个 action）
② reviewer alice 审核：拒绝（理由：动作分得太粗）
   POST /T001/review {verdict: 'rejected', review_comment: '动作分得太粗'}
   → status: submitted → rejected → auto → pending

③ 自动取消之前的 actions（标 lifecycle='rejected'）

④ 派单人选另一个 labeler 重做：
   POST /T001/assign {assigned_to: 'labeler_010'}

⑤ labeler_010 重新提交 → 重复审核
```

---

## 7. 不变式（P1.5 落地时确认）

| # | 规则 | 实现 |
|---|------|------|
| **AT-1** | task_id 格式 `^[0-9A-Za-z]{8}$` | DB CHECK |
| **AT-2** | task_kind ∈ `{action_label, time_window, quality_check, tag_review}` | DB CHECK |
| **AT-3** | status 状态机严格（pending → ...）| Validator |
| **AT-4** | 单 task 只分给 1 个 labeler（assigned_to 单值）| schema |
| **AT-5** | actions.task_id 外键 必须存在于 annotation_tasks（如果非 NULL）| DB 外键 |
| **AT-6** | submit 必须同事务写 actions + update task.status | AssetWriter |
| **AT-7** | rejected 自动回 pending + 清空 assigned_to | Validator |
| **AT-8** | customer_id 外键 必为 customers.customer_id 合法值 | DB 外键 |
| **AT-9** | priority ∈ [1, 10]，1 最高 | DB CHECK |

---

## 8. 与其他模块的关系

| 模块 | 关系 |
|------|------|
| **`asset-hierarchy-and-derivatives.md`** | target_asset_id 外键 到 assets；产物 action 也是 asset |
| **`asset-versioning.md`** | task 提交产生 action 时，新 action 走标准 asset 创建流程（自引用 logical_id）|
| **`asset-tagging.md`** | task 完成后产生的 action 不打 tag（label 走 actions.primary_label）|
| **`algo-runs.md`** | algo 低 confidence 时通过 ActionHandler 自动派 task；task.metadata.triggered_by_algo_run 反向追溯 |
| **`customers-and-deliveries.md`** | task.customer_id 标记任务归属客户；完成后产物可进客户 delivery |

---

## 9. P1.5 落地范围（vs 永远 Out of Scope）

### 9.1 P1.5 必做

- ✓ annotation_tasks 表 + 状态机
- ✓ 基础 API（create / assign / start / submit / review / query）
- ✓ actions.task_id 外键
- ✓ submit 同事务原子（task status + actions 一起）
- ✓ rejected 自动回 pending
- ✓ batch_id 字段 + 批量进度查询

### 9.2 P2 视情况

- annotation_task_batches 独立表（业务上若需要批次级别 SLA / 审计）
- 标注员绩效统计 dashboard
- 自动分派算法（按 labeler 闲忙度 / 历史质量评分自动 assign）
- algo_run.metadata.triggered_by_algo_run 反向 外键

### 9.3 永不做

| 不做的事 项 | 原因 |
|-------|------|
| **多人协作单 task**（assigned_to 数组）| 任务粒度细，1:1 派单足够；多人 = 派多个 task |
| **workflows / pipelines 编排**（task DAG 依赖）| 平台不管编排，外部 k8s / 脚本 |
| **labeler 绩效自动算工资**（payroll）| 业务上不在 DataBrew 范畴 |
| **标注工具 UI 集成**（CVAT / Label Studio）| 第三方工具，提供 API 集成即可 |
| **subscriptions / 标注配额** | 商业模型 不做的事 |

---

## 10. 决策追溯

| 决策 | 来源 |
|------|------|
| annotation_tasks 一等实体（业务参考类）| rev.12 决策 5 |
| B 规模（10-50 标注员）| 用户回答 Q5.1 |
| 业务参考层（单层模型）| 用户回答 Q5.2 |
| workflows 不做（不做的事）| 用户回答 Q5.3 |
| 1 对 1 分派 | rev.12 Q5.4（推荐）|
| batch_id 单字符串（不独立表）| rev.12 Q5.5（推荐）|
| submit 同事务原子 | rev.12 Q5.6（推荐）|
| 可选审核（不强制）| rev.12 Q5.7（推荐）|
| rejected 自动回 pending | rev.12 Q5.8（推荐）|
| P1 不加 algo_runs 外键，留 metadata | rev.12 Q5.9（推荐）|
| **P1.5 落地（不是 P1）**| rev.12 决策 5 锁定 |

---

## 11. 触发 P1.5 落地的条件

```
任一发生：
  - 业务上每天 > 50 个标注任务需要派单
  - 出现「同一 asset 重复派给多人」事故
  - 客户要求按交付质量追责（reviewer trail）
  - SLA 追踪需求出现（due_at + 逾期告警）
→ 启动本 design doc 的 DDL + API 落地
→ 工程量约 2 周（含测试）
```

在此之前，继续用 actions.source_type='human' 模式。
