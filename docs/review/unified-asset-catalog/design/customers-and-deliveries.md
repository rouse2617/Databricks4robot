# 客户与交付设计（customers + deliveries + delivery_rules）

| 字段 | 值 |
|------|----|
| 状态 | Active（P1）|
| 关联 | `../README.md` §7-§8 / `../schema.md` §6 §14 §15 |
| 决策来源 | rev.12 决策 2（customers 一等实体）+ rev.10 C2 commit / DR5 enforce_mode |

---

## 1. 业务问题（Why）

### 1.1 客户主数据缺失

DataBrew 是 B2B 数据资产平台，但**客户实体完全没建模**：
- `deliveries.customer_id` 是裸 TEXT，无 外键 校验
- 客户 SLA / 优先级 / 合规要求 / 联系人 / 退款政策 → **没地方存**
- 「这个客户买过多少」「续费率」「退过几次货」 → 没数据基础
- 「这客户拒绝 PII 数据」「这客户要 GDPR strict 模式」 → 无法配置

### 1.2 交付契约不清晰

- delivery 是「订单（business event）」还是「数据集快照（data entity）」？
- delivery_items 永远指**具体 asset_id** 还是可以指 logical？
- 「跟随升级」vs「锁版」是订单属性还是 item 属性？
- 客户拿到的「下载清单」生命周期多长？

---

## 2. 设计原则（How）

### 2.1 客户是「业务参考」类一等实体

按 `../README.md` §1 分层：

| 类型 | customers 归类 |
|------|--------------|
| 物理事实 | ✗（非外部输入；平台主导创建/维护）|
| 业务资产 | ✗（不需要多版本演进；customers 没有 v1/v2 概念）|
| 执行事件 | ✗（长期实体可改，不是一次性事件）|
| **业务参考** | ✓（长期存在 + 被 deliveries / annotation_tasks / delivery_rules 反复引用）|

**特征：**
- 长期实体（onboard → 一直在 → 偶尔 offboard）
- 主数据维护（SLA / 合规要求 / 联系人）
- 被多个 deliveries 引用（1 客户 N 交付）

### 2.2 deliveries 是「执行事件」（交付动作）

- delivery = **「我把这一批 asset 在某时间点交付给客户 X」的事件记录**
- 不是「订单」（订单是销售概念，DataBrew 不管销售）
- 不是「数据集快照」（数据集走 datasets / dataset_snapshots，P1.5 ML 平台对接时启用）
- delivery_items = 该次交付的资产清单（**永远指具体 asset_id**，frozen 快照）
- 客户拿到的 manifest 永久有效（GCS r<n>-uuid 不可变路径保证）

### 2.3 customer_id 是 slug 风格

```
customer_id: cust_alpha_robotics    ← 业务可读 slug
            : cust_beta_auto
            : cust_gamma_research
```

- 运营人工分配（创建客户时填）
- 平台校验唯一性 + 格式 regex `^[a-z][a-z0-9_-]{2,31}$`
- 不用 8 位 hash（hash 在 URL / manifest 里出现不友好）

---

## 3. 数据模型（DDL）

### 3.1 customers 表

```sql
CREATE TABLE customers (
  customer_id      TEXT PRIMARY KEY
                   CHECK (customer_id ~ '^[a-z][a-z0-9_-]{2,31}$'),
                   -- 业务可读 slug，例 'cust_alpha_robotics'

  -- 身份
  display_name     TEXT NOT NULL,
  legal_name       TEXT,                            -- 法律实体名

  -- 状态
  status           TEXT NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','trial','suspended','offboarded')),
  region           TEXT,                            -- us | eu | cn | other

  -- 商务关系
  sla_tier         TEXT NOT NULL DEFAULT 'standard'
                   CHECK (sla_tier IN ('standard','premium','enterprise')),
  account_owner    TEXT,                            -- DataBrew 内部负责人 user_id

  -- 合规要求
  compliance_tags  JSONB NOT NULL DEFAULT '[]',
                   -- ["gdpr_strict","no_pii","us_only"]
  exclude_tags     JSONB NOT NULL DEFAULT '[]',
                   -- 客户拒绝的 tag 类型（自动从交付里过滤）
                   -- 例: [{"key":"compliance.pii","value":"true"}]

  -- 扩展
  metadata         JSONB NOT NULL DEFAULT '{}',
                   -- 联系人 / 邮箱 / 合同 ID 等灵活字段
                   -- {"contact_email":"...", "phone":"...", "company_size":"100-500"}
  extra            JSONB NOT NULL DEFAULT '{}',

  -- 时间戳
  onboarded_at     TIMESTAMPTZ,
  offboarded_at    TIMESTAMPTZ,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  row_version      BIGINT NOT NULL DEFAULT 1
);

CREATE INDEX idx_customers_status      ON customers(status);
CREATE INDEX idx_customers_sla         ON customers(sla_tier);
CREATE INDEX idx_customers_account_own ON customers(account_owner) WHERE account_owner IS NOT NULL;
CREATE INDEX idx_customers_region      ON customers(region) WHERE region IS NOT NULL;
```

### 3.2 deliveries 表（沿用现网 + 加 外键）

```sql
-- 沿用 schemas/pg-phase0.sql:464-497，rev.12 加 customer_id 外键
deliveries (
  delivery_id      TEXT PRIMARY KEY,
  customer_id      TEXT NOT NULL REFERENCES customers(customer_id),  -- rev.12 外键
  contract_id      TEXT,                            -- P1 保 TEXT；P1.5 视情况加 contracts 表
  delivery_type    TEXT NOT NULL DEFAULT 'asset_set',
  status           TEXT NOT NULL DEFAULT 'draft',
                   -- draft | resolving | ready_to_commit | committed
  manifest_uri     TEXT,
  replay_manifest_uri TEXT,
  delivered_at     TIMESTAMPTZ,
  metadata         JSONB DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deliveries_customer ON deliveries(customer_id, created_at DESC);
CREATE INDEX idx_deliveries_status   ON deliveries(status);
```

### 3.3 delivery_items 表

```sql
delivery_items (
  delivery_id       TEXT NOT NULL REFERENCES deliveries(delivery_id),
  asset_id          TEXT NOT NULL REFERENCES assets(asset_id),     -- 锁具体版本
  expected_revision BIGINT,                                         -- C2 commit 校验
  payload_mode      TEXT NOT NULL CHECK (payload_mode IN ('materialized','virtual')),
  forced            BOOL NOT NULL DEFAULT false,                    -- C2 commit force_overrides 记录
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (delivery_id, asset_id)
);

CREATE INDEX idx_di_asset ON delivery_items(asset_id);
```

### 3.4 delivery_rules 表（rev.10 含 dsl_version + enforce_mode）

```sql
CREATE TABLE delivery_rules (
  rule_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  owner         TEXT NOT NULL,
  customer_id   TEXT REFERENCES customers(customer_id),  -- NULL = 通用规则；非 NULL = 客户专属
  query_dsl     JSONB NOT NULL,                          -- 与 queries/run 同语法
  dsl_version   TEXT NOT NULL DEFAULT 'v1',              -- 引擎升级前向兼容
  enforce_mode  TEXT NOT NULL DEFAULT 'block'
                CHECK (enforce_mode IN ('block','warn','tag_only')),
  rating_scope  TEXT NOT NULL DEFAULT 'current'
                CHECK (rating_scope IN ('current','logical')),
  is_active     BOOL NOT NULL DEFAULT true,
  version       BIGINT NOT NULL DEFAULT 1,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_drules_customer ON delivery_rules(customer_id, is_active);
CREATE INDEX idx_drules_active   ON delivery_rules(is_active) WHERE is_active = true;
```

### 3.5 query_dsl 形态规范（rev.12 必须）

`query_dsl` 与 **`/queries/run` 查询 DSL 同语法**（共享 query 引擎，避免双轨）。结构骨架：

```jsonc
{
  "where": [                          // AND 条件数组
    {
      "field": "tag.scenario",        // 字段名（支持点号路径，参考 filter.SearchableFields）
      "op": "eq",                     // eq | ne | in | gt | gte | lt | lte | contains | exists | not_exists
      "value": "kitchen"
    },
    {
      "field": "lifecycle_state",
      "op": "in",
      "value": ["ready", "archived"]
    }
  ],
  "asset_types": ["clip", "action"]   // 可选：限定 asset_type；缺省 = 全部
}
```

**示例 1：阻 PII 资产（block 模式）**

```jsonc
{
  "name": "cust_alpha_no_pii",
  "customer_id": "cust_alpha_robotics",
  "enforce_mode": "block",
  "query_dsl": {
    "where": [
      {"field": "tag.compliance.pii", "op": "eq", "value": "true"}
    ]
  }
}
// 含义：cust_alpha 的 delivery 中不允许含 PII 资产；commit 时若命中 → 进 rule_failed 阻 commit
```

**示例 2：要求质量评分 ≥ 4（warn 模式）**

```jsonc
{
  "name": "premium_quality_threshold",
  "customer_id": null,                // 通用规则，适用所有客户
  "enforce_mode": "warn",
  "rating_scope": "current",
  "query_dsl": {
    "where": [
      {"field": "metric.rating.quality_score", "op": "lt", "value": 4.0}
    ]
  }
}
// 含义：质量评分 < 4 的资产仅警告（运营可直接 commit 不需 force）
```

**示例 3：标 delivery_ready 系统 tag（tag_only 模式）**

```jsonc
{
  "name": "ml_training_ready",
  "customer_id": null,
  "enforce_mode": "tag_only",
  "query_dsl": {
    "where": [
      {"field": "tag.algo.hand_track.status", "op": "eq", "value": "ok"},
      {"field": "tag.algo.action_detector.status", "op": "eq", "value": "ok"},
      {"field": "metric.rating.quality_score", "op": "gte", "value": 4.0}
    ]
  }
}
// 含义：满足三个条件的资产自动打 `delivery_ready:ml_training_ready` 系统 tag；
// 不阻 commit，仅作为运营提数时的便利 facet（DeliveryEligibilityProjector P1.5 处理）
```

**JSON Schema**：P1.5 时由 `query_field_registry.yaml` 派生（与 `/queries/run` 公用）；P1 仅靠 query 引擎运行时校验。

---

## 4. API 契约

### 4.1 customers API

```text
POST /api/v1/customers
Idempotency-Key: <uuid>
{
  "customer_id": "cust_alpha_robotics",
  "display_name": "Alpha Robotics Inc.",
  "legal_name": "Alpha Robotics Holdings Ltd.",
  "status": "active",
  "region": "us",
  "sla_tier": "premium",
  "account_owner": "alice@databrew",
  "compliance_tags": ["gdpr_strict"],
  "exclude_tags": [
    {"key": "compliance.pii", "value": "true"}
  ],
  "metadata": {
    "contact_email": "ops@alpha-robotics.com",
    "phone": "+1-...",
    "industry": "autonomous_driving"
  }
}
→ 201 Created

PATCH /api/v1/customers/cust_alpha_robotics
{
  "sla_tier": "enterprise"
}
→ 200 OK

GET /api/v1/customers/cust_alpha_robotics
→ 详情

GET /api/v1/customers?status=active&sla_tier=enterprise
→ 列表

GET /api/v1/customers/cust_alpha_robotics/deliveries
→ 该客户所有 deliveries（反查 deliveries.customer_id 外键）
```

### 4.2 deliveries 状态机

```
draft → resolving → ready_to_commit → committed
                                    ↓
                                  archived (P1.5)
```

> ⚠️ **现网兼容性提醒**（rev.12 第二轮 review）：现网 `POST /api/v1/deliveries` 是**一步式 commit**（直接落 `committed` 态，无草稿 / 加 item / 校验 / resolve 阶段），现网 `deliveries.status` DEFAULT `'pending'`。要落地 rev.12 四态机，有两条路径：
>
> **路径 A（推荐）**：旧路径 `POST /deliveries`（一步 commit）保留作**简化版降级**：
>   - 内部行为：自动 `INSERT delivery(status='draft')` → `INSERT delivery_items` → `commit`（跳过 `resolve` 阶段）
>   - 若版本化 + delivery_rules 未落地，跳过 `expected_revision` / rule 校验（degrade 成 rev.11 一步 commit）
>   - 用于「我就要简单交付，不关心冲突」的客户场景
>
> **路径 B（新增）**：新 API `POST /deliveries`（status=`draft`）+ `POST /commit`（带 C2 校验）走完整四态机：
>   - 用于需要严格版本快照 + 合规校验的客户
>
> 两条路径**长期并存**（不规划弃用旧路径）。旧路径作为「一步 commit」语义糖，新路径作为「精确 commit」契约。

| 状态 | 含义 | 进入条件 |
|------|------|---------|
| `draft` | 草稿，可加 item | `POST /deliveries` |
| `resolving` | C2 commit 失败 409 后进入，等运营 resolve 冲突 | `POST /commit` 返 409 |
| `ready_to_commit` | 冲突 resolve 完，可再 commit | `POST /resolve-conflicts` |
| `committed` | 提交完成，items 锁版，manifest 生成 | `POST /commit` 返 200 |

### 4.3 deliveries API

```text
① 创建草稿
POST /api/v1/deliveries
{
  "customer_id": "cust_alpha_robotics",
  "contract_id": "CT-2026-Q2-001",
  "delivery_type": "asset_set"
}
→ {delivery_id: "del_xyz789", status: "draft"}

② 加 item
POST /api/v1/deliveries/del_xyz789/items
Idempotency-Key: <uuid>
[
  {"asset_id": "clipA_v3", "expected_revision": 3, "payload_mode": "materialized"},
  {"asset_id": "actionB_v1", "expected_revision": 1, "payload_mode": "virtual"}
]

③ 首次 commit
POST /api/v1/deliveries/del_xyz789/commit
Idempotency-Key: <uuid>

← 200 OK (status=committed)  或
← 409 (status=resolving, body 含冲突列表)
  {
    "delivery_id": "del_xyz789",
    "status": "resolving",
    "stale_revisions": [
      {"item_id": "...", "expected_revision": 3, "actual_asset_id": "clipA_v4", "actual_revision": 4}
    ],
    "deleted": [...],
    "rule_failed": [
      {"item_id": "...", "asset_id": "...", "rule_id": "...", "reason": "compliance.pii=true on this asset"}
    ],
    "not_materialized": [
      {"item_id": "...", "asset_id": "...", "current_materialization": "virtual"}
    ]
  }

④ 运营解决冲突
POST /api/v1/deliveries/del_xyz789/resolve-conflicts
{
  "resolutions": [
    {"item_id": "...", "action": "replace_latest"},      ← 换成最新 revision
    {"item_id": "...", "action": "force"},               ← 强制接受（含 PII，标 forced=true）
    {"item_id": "...", "action": "remove"}                ← 移除该 item
  ]
}
→ status: ready_to_commit

⑤ 二次 commit（必须用新 Idempotency-Key）
POST /api/v1/deliveries/del_xyz789/commit
Idempotency-Key: <new-uuid>
→ 200 OK, status=committed
```

### 4.4 delivery_rules API

```text
POST /api/v1/delivery-rules
{
  "name": "cust_alpha_no_pii",
  "owner": "alice@databrew",
  "customer_id": "cust_alpha_robotics",       ← 客户专属
  "query_dsl": {
    "where": [{"field": "tag.compliance.pii", "op": "eq", "value": "true"}]
  },
  "dsl_version": "v1",
  "enforce_mode": "block",                     ← block | warn | tag_only
  "rating_scope": "current"
}
→ delivery_rules.is_active=true 后参与所有该客户 delivery 的 C2 commit 校验

GET /api/v1/delivery-rules?customer_id=cust_alpha_robotics
→ 列表
```

---

## 5. C2 commit 协议（rev.10 决策）

### 5.1 强一致校验

**commit 时同事务在 PG 层重跑校验**（不依赖 ES 投影，强一致）：

```sql
BEGIN;
  -- ① 校验所有 expected_revision 还匹配
  -- 注意：rev.10 V4 决策 —— commit 只认 expected_revision（业务版本），不认 row_version（任何 A 路由 tag 变更不触发 409）
  FOR item IN delivery_items WHERE delivery_id='del_xyz789' LOOP
    SELECT revision INTO actual FROM assets WHERE asset_id = item.asset_id;
    IF actual != item.expected_revision THEN
      raise_stale_revision(item, actual);
    END IF;
    -- 不检查 row_version；A 路由微调不影响 delivery
  END LOOP;

  -- ② 校验 delivery_rules
  FOR rule IN delivery_rules WHERE customer_id='cust_alpha_robotics' AND is_active LOOP
    -- 跑 query_dsl 校验：该 rule 命中的 asset_id 不能出现在 delivery_items
    -- enforce_mode=block: 命中 → rule_failed
    -- enforce_mode=warn:  命中 → 仅警告
    -- enforce_mode=tag_only: 不阻 commit，仅打 tag
  END LOOP;

  -- ③ 校验 materialization（payload_mode=materialized 时 assets.materialization 必须=materialized）
  -- ...

  -- ④ 所有校验通过：commit
  UPDATE deliveries SET status='committed', delivered_at=now(), manifest_uri='gs://.../manifest.json'
   WHERE delivery_id='del_xyz789';

  -- ⑤ 更新 asset 的 delivery_count / last_delivered_*
  UPDATE assets SET delivery_count=delivery_count+1, last_delivered_at=now(), last_delivered_to='cust_alpha_robotics'
   WHERE asset_id IN (SELECT asset_id FROM delivery_items WHERE delivery_id='del_xyz789');
COMMIT;
```

### 5.2 expected_revision 只拦 B 路由（V4 决策）

| 场景 | commit 行为 |
|------|------------|
| commit 时该 asset 仍是 expected_revision（A 路由微调 tag 但 revision 没变）| ✓ 通过 |
| commit 时该 asset 已经 B 路由生 v4，is_current=true 是 v4 而非 v3 | ✗ 409 stale_revisions |
| commit 时该 asset 已被 soft-delete | ✗ 409 deleted |

→ **A 路由不触发 409**（commit 期间标注员加 tag 不影响）。

### 5.3 enforce_mode 三态语义

| 值 | commit 时行为 | projector 行为 |
|----|------------|---------------|
| `block` | 不通过 → 进 `rule_failed` 阻 commit | 不打标 |
| `warn` | 不通过 → 进 `rule_failed` 但仅警告（运营可直接 commit 不需 force）| 不打标 |
| `tag_only` | 不参与 commit 校验 | DeliveryEligibilityProjector 仅按结果打 / 撤 `delivery_ready:{rule_id}` 系统 tag |

### 5.4 Idempotency-Key 协议（C2-D）

| 阶段 | Idempotency-Key 用法 |
|------|---------------------|
| ② 加 item | 每次 add 用独立 key（重复 add 同 item 幂等）|
| ③ 首次 commit | 用 key K1 |
| ④ resolve-conflicts | 用独立 key（与 commit 隔离）|
| ⑤ 二次 commit | **必须用新 key K2**（与 K1 不同；同 key 不同 payload 会 409）|

---

## 6. 业务场景 walkthrough

### 6.1 客户 onboard 流程

```text
① 创建客户
POST /api/v1/customers
{
  customer_id: 'cust_alpha_robotics',
  display_name: 'Alpha Robotics Inc.',
  sla_tier: 'premium',
  region: 'us',
  compliance_tags: ['gdpr_strict']
}

② 创建客户专属 delivery_rule（拒 PII）
POST /api/v1/delivery-rules
{
  customer_id: 'cust_alpha_robotics',
  query_dsl: {where: [{field: 'tag.compliance.pii', op: 'eq', value: 'true'}]},
  enforce_mode: 'block'
}

③ 客户开始可以收交付
POST /api/v1/deliveries {customer_id: 'cust_alpha_robotics', ...}
```

### 6.2 算法升级后通知客户续交付

```text
① clipA_v1 在 R001 中产生，delivery_42 交付给 cust_alpha (asset_id='clipA_v1', revision=1)
② 半年后 algo upgrade v2.0，R002 跑出 clipA_v2 (logical='L_clipA', revision=2)
③ RevisionNotifier ActionHandler 订阅 asset_revised event：
   SELECT delivery_id, customer_id
   FROM delivery_items di JOIN deliveries d USING (delivery_id)
   WHERE di.asset_id IN (
     SELECT asset_id FROM assets WHERE logical_asset_id='L_clipA' AND revision < 2
   )
   → 命中 delivery_42 / cust_alpha
④ 平台发邮件 / 飞书 / Slack：
   「您之前购买的 clipA（delivery_42）有新版本 v2 可续交付，是否需要续费升级？」
⑤ 客户决定续交付：
   POST /api/v1/deliveries (新 delivery)
   POST /items {asset_id: 'clipA_v2', expected_revision: 2, payload_mode: 'materialized'}
   POST /commit
```

### 6.3 C2 commit 409 + resolve + 二次 commit

```text
① POST /deliveries (draft) → del_001
② POST /items (3 个 asset)
③ POST /commit (Idempotency-Key=K1)
   ← 409
   {
     status: 'resolving',
     stale_revisions: [{item: 1, expected_revision: 3, actual: 4}],
     rule_failed: [{item: 2, rule: 'cust_alpha_no_pii', reason: '..'}]
   }

④ 运营 UI 显示冲突，运营操作：
   - item 1: 「换最新 v4」
   - item 2: 「强制接受（特批）」
   - item 3: 已通过，保留

   POST /resolve-conflicts {
     resolutions: [
       {item_id: 1, action: 'replace_latest'},          ← item 1 改为 asset_id=clipA_v4, expected_revision=4
       {item_id: 2, action: 'force'}                     ← item 2 标 forced=true
     ]
   }
   → status: ready_to_commit

⑤ POST /commit (Idempotency-Key=K2)        ← 必须新 key
   → 200 OK, status=committed
```

### 6.4 合规审计反查

```sql
-- 「这个客户的 delivery 历史里有没有 PII 资产」
SELECT DISTINCT d.delivery_id, di.asset_id, t.tag_value
FROM deliveries d
JOIN delivery_items di USING (delivery_id)
JOIN asset_tags t ON t.asset_id = di.asset_id
WHERE d.customer_id = 'cust_alpha_robotics'
  AND t.tag_key = 'compliance.pii'
  AND t.tag_value = 'true'
  AND d.status = 'committed';
```

---

## 7. 客户与 tag 命名空间关系

`asset_tags.customer.<id>.*` 命名空间用于「资产层级的客户属性」：

```text
asset_tags 中：
  customer.cust_alpha.priority = high              ← cust_alpha 标记该资产高优先级
  customer.cust_alpha.exclude  = true              ← cust_alpha 拒绝该资产
  customer.cust_beta.tier      = premium_only       ← cust_beta 标记该资产专享
```

**Validator 强约束（Tag-Source-customer）：**
- `<id>` 必须是 `customers.customer_id` 合法值（lint 校验）
- 防止 typo（如 `customer.cust_typo.priority`）

---

## 8. 不变式与边界

| # | 规则 | 实现 |
|---|------|------|
| **CUST-1** | `customer_id` 格式 `^[a-z][a-z0-9_-]{2,31}$` | DB CHECK |
| **CUST-2** | `status ∈ {active, trial, suspended, offboarded}` | DB CHECK |
| **CUST-3** | `sla_tier ∈ {standard, premium, enterprise}` | DB CHECK |
| **CUST-4** | `deliveries.customer_id` 必 外键 到 customers（rev.12 加）| DB 外键 |
| **CUST-5** | tag `customer.<id>.*` 中 `<id>` 必为 customers.customer_id 合法值 | TagValidator lint |
| **DEL-1** | `delivery_items.asset_id` 永远指**具体 asset_id**（不指 logical_asset_id） | schema 外键 |
| **DEL-2** | commit C2 必须 PG 强一致校验（不读 ES） | Validator |
| **DEL-3** | `expected_revision` 不匹配 → 409 stale_revisions | C2 commit |
| **DEL-4** | A 路由（tag / metric 变化）**不**触发 commit 409 | rev.10 V4 决策 |
| **DEL-5** | 二次 commit 必须用新 Idempotency-Key | API gateway |
| **DR5** | delivery_rules 必带 dsl_version；引擎升级前向兼容 | Validator |
| **DR_ENF** | enforce_mode 三态（block / warn / tag_only）| DB CHECK + projector |

---

## 9. 与其他模块的关系

| 模块 | 关系 |
|------|------|
| **`asset-hierarchy-and-derivatives.md`** | 任何 asset_type 都可进 delivery（含 raw_mcap 原始数据资产）|
| **`asset-versioning.md`** | delivery_items 永远指具体 asset_id（不跟随 logical_asset_id 升级）|
| **`asset-tagging.md`** | customer.<id>.* 命名空间 + delivery_rules 用 tag 表达资格 |
| **`algo-runs.md`** | 「算法升级影响哪些客户」反查路径：algo_run → affected_assets → delivery_items → customers |
| **`annotation-tasks-p1.5.md`** | annotation_tasks 可关联 customer_id（标记任务归属某客户）|

---

## 10. 不做的事（Out of Scope）

| 不做的事 项 | 原因 |
|-------|------|
| **contracts 表**（P1.5+ 视情况）| P1 contract_id 保 TEXT；合同管理复杂度业务真做 P1.5 再加 |
| **subscriptions / billing 表** | 计费模型未来再说；不在本 PRD 范畴 |
| **delivery_items 跟随 logical 升级**（follow_mode=follow）| S3 选项，P1 仅 frozen；客户老链接稳定优先 |
| **delivery 退货 / 撤回**（refund / recall）| P2+ 视业务需求；当前 archived 状态机够用 |
| **跨客户数据共享 / 转售** | 合规 / 商业模型未明，不建模 |
| **客户分级自动调整**（按消费金额自动升 sla_tier）| 业务规则未定；P1 仅手工 |

---

## 11. 决策追溯

| 决策 | 来源 |
|------|------|
| customers 一等实体（业务参考类）| rev.12 决策 2（Q2.1-Q2.6）|
| customer_id slug 风格 | rev.12 Q2.1 |
| 运营填 + 平台校验 | rev.12 Q2.2 |
| 最小字段集（含 region）| rev.12 Q2.3 |
| P1 不上 contracts 表 | rev.12 Q2.4 |
| tag customer.* 命名空间 + lint | rev.12 Q2.5 |
| P1 上线 | rev.12 Q2.6 |
| C2 commit 拆 resolve/commit API（C2-D）| rev.10 第二轮 grill |
| expected_revision 只拦 B 路由（V4）| rev.10 第二轮 grill |
| enforce_mode 三态（DR_ENF）| rev.10 第二轮 grill |
| dsl_version（DR5）| rev.10 第二轮 grill |
| Idempotency-Key 每次 commit 用新 key | rev.10 C2-D |
