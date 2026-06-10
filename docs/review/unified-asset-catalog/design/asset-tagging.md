# 资产标签 / 标注 / 评分 / 评估 4 层设计

| 字段 | 值 |
|------|----|
| 状态 | Active（P1）|
| 关联 | `../README.md` §10 / `../schema.md` §7-§10 |
| 决策来源 | rev.7 §7 tag/label 术语 / rev.10 TT3 algo 双写 / rev.10 W2 tag_registry / rev.10 TP1/TP2 propagation（P1.5）|

---

## 1. 业务问题（Why）

「给资产打标签」是 DataBrew 最高频的操作，但**「标签」业务上有 4 种语义完全不同的形态**：

| 业务需求 | 应该用 | 现网/PRD 设计错位 |
|---------|--------|--------------|
| 资产级离散分类（kitchen / quality=ok）| **tag** | ✓ asset_tags 已落地 |
| action 实体的核心动作分类（grasp）| **action label** | ✓ actions.primary_label 已落地 |
| 数值评分（4.5 分 / confidence=0.93）| **metric** | ✓ asset_metrics 已落地 |
| 评估完整 payload（QA 报告）| **eval result** | ✓ asset_eval_results 已落地 |

**痛点**（非建模痛点，是治理痛点）：
- 新 tag 类型上线要改 schema → 实际不应该，应该走 tag_registry YAML
- 多源 tag（人 + 算法 + 规则都说 quality=ok）怎么并存
- PII / 合规标签如何沿血缘自动传播
- tag / label / metric 4 种业务上经常混用，新人不知道用哪个

---

## 2. 4 种「标签信息」的明确分工

### 2.1 一张表说清

| 你要标的东西 | 用哪个 | 表 | PK | 例子 |
|------------|--------|---|----|------|
| 资产级开放 K-V 分类（治理 / 检索 / 交付资格 facet）| **tag** | `asset_tags` | `(asset_id, tag_key, tag_value, source, source_version)` | scenario=kitchen, customer=cust_X |
| action 实体核心动作分类（带时间窗，每个算法版本/标注员独立一行，用 `revision_of` 串成版本链）| **action label** | `actions.primary_label / labels[]` | action_id | primary_label='grasp', labels=['grasp','manipulation'] |
| 数值标量（可聚合 / 可比较 / 可阈值）| **metric** | `asset_metrics` | `(asset_id, metric_key, source, source_name, source_version, scored_at)` | rating.quality_score=4.5, confidence=0.93 |
| 评估完整 payload（含详细 reasoning / 多维度细则）| **eval result** | `asset_eval_results` | `eval_result_id` (UUID) | QA pipeline 跑出的完整 JSON 报告 |

### 2.2 该放哪张表？四问决策流程

业务方拿来一个新字段需求，按下面四个问题**从上往下问**，命中哪个就放哪张表。不用纠结，秒出答案。

```
新需求：要给资产加个分类字段 X（例：亮度评分 / 场景=kitchen / QA 报告 / 动作=grasp）
│
├─ 问 1：X 是 action 这种实体的「身份字段」吗？
│        （没它实体就不成立 —— 比如 action 不告诉我是 grasp 还是 pickup，这条记录就毫无意义）
│   ✅ 是 → 写进 actions.primary_label / labels[]    例：grasp、pickup
│
├─ 问 2：X 是个「数值」吗？（能算平均、能排序、能比大小）
│   ✅ 是 → 写进 asset_metrics                      例：亮度=87、置信度=0.93、评分=4.5
│
├─ 问 3：X 是「一坨结构化报告」吗？（不是单值，是嵌套 JSON / 多维度评分 + 文字理由）
│   ✅ 是 → 写进 asset_eval_results                 例：QA 完整报告（8 维评分 + reasoning）
│
└─ 问 4：以上都不是，X 就是个普通的「键=值」分类标签？
   ✅ 是 → 写进 asset_tags                          例：scenario=kitchen、quality=ok、customer.cust_X.priority=high
```

**记忆口诀**：身份字段 → action label；能算数 → metric；带报告 → eval；剩下都是 → tag。

### 2.3 直觉判别 3 条

1. **「资产挂了多少 X」可以 0 到 N？** → tag（一个 clip 可以打 0 个或 20 个 tag）
2. **「X 是不是这个表的列定义？」**：列 → action label；K-V 表 → tag
3. **「X 跨多种 asset_type？」**：跨 → tag；只对某一类有意义 → 那一类的 aspect 表 label 字段

### 2.4 边界场景对照

| 场景 | 选 | 理由 |
|------|----|------|
| 这条 segment 是厨房场景 | **tag** `scenario=kitchen` | 资产 facet |
| 这条 segment 客户 cust_X 拥有 | **tag** `customer=cust_X` | 治理 |
| 这条 segment 被 hand_track v2.0 处理过 | **tag** `algo.hand_track.version=2.0`（algo_sdk 双写） | 跨实体可筛选 |
| 这个 action（t=1000-2000）是 grasp | **action label** `actions.primary_label=grasp` | 实体核心字段 |
| 算法人员给资产打分 4.5 | **metric** `rating.quality_score=4.5`（source=human） | 数值、可聚合 |
| 算法人员附带自由文本 review | **eval result**（payload 引同一 reviewer_id） | 完整原档 |
| 评分简单分类（通过/不通过）| **tag** `review.verdict=passed` | 离散枚举，不需数值聚合 |
| LLM 给整个 clip 生成 summary 长文本 | **既不是 tag 也不是 label** → `asset_content.summary_text` 或 metadata | 不是分类，是描述 |
| 客户标记「这条不要交付」 | **tag** `customer.cust_X.exclude=true` | 治理 facet |
| 标注员标这个 clip 内有 3 个 grasp 动作 | **创建 3 个 action 实体**（label=grasp） | 每个 action 是独立实体 |

---

## 3. asset_tags 表设计

### 3.1 表结构

```sql
CREATE TABLE asset_tags (
  asset_id        TEXT NOT NULL REFERENCES assets(asset_id),
  tag_key         TEXT NOT NULL,
  tag_value       TEXT NOT NULL,

  -- 现网兼容列（保留，不 drop）
  tag_value_num   DOUBLE PRECISION,
  tag_value_bool  BOOLEAN,
  tag_type        TEXT NOT NULL DEFAULT 'string',  -- string / number / bool / enum

  -- 多源契约（5+ 类来源）
  source_type     TEXT NOT NULL,
                  -- algo_sdk | rule_engine | human | system | compliance | llm | vendor | ...
  source_name     TEXT,                    -- algo 名 / user_id / rule 名
  source_version  TEXT,                    -- algo 版本 / rule 版本
  run_id          TEXT REFERENCES algo_runs(run_id) ON DELETE SET NULL,  -- 算法 / 规则触发时关联

  -- 系统字段
  tenant_id       TEXT,
  project_id      TEXT,
  applied_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 多源共存：PK 不能直接用 COALESCE（PG 不允许表达式 PK）
  -- 改用 generated column + UNIQUE 约束（功能等价）
  source_version_norm TEXT GENERATED ALWAYS AS (COALESCE(source_version, '')) STORED,
  CONSTRAINT uq_asset_tags_identity
    UNIQUE (asset_id, tag_key, tag_value, source_type, source_version_norm)
);

CREATE INDEX idx_atags_lookup       ON asset_tags(tag_key, tag_value, asset_id);
CREATE INDEX idx_atags_source       ON asset_tags(source_type, source_name, source_version);
CREATE INDEX idx_atags_propagation  ON asset_tags(tag_key) WHERE tag_key LIKE 'compliance.%';
CREATE INDEX idx_atags_run          ON asset_tags(run_id) WHERE run_id IS NOT NULL;
```

> 命名约定：文中用 `source` 描述来源语义；DB 物理列名统一沿用现网 `source_type`。

### 3.2 多源共存示例

```
asset_id   tag_key                     tag_value  source_type   source_name           source_version
─────────  ──────────────────────────  ─────────  ───────────  ────────────────────  ──────────────
clipA_v2   scenario                    kitchen    human         labeler_007            NULL
clipA_v2   scenario                    kitchen    rule_engine   scenario_classifier   1.0           ← 同 key 多源共存
clipA_v2   algo.hand_track.version     2.0        algo_sdk      hand_track             2.0
clipA_v2   quality                     ok         rule_engine   quality_check          1.0
clipA_v2   customer                    cust_alpha system        delivery_pipeline      NULL
clipA_v2   compliance.pii              false      compliance    compliance@2026-Q1     NULL
```

**UI 展示规则：** 同 key 多源时显示「2 个来源都确认是 kitchen」（信任度叠加）。

### 3.3 现网代码 bug 修复

**当前问题**：`AssetTagRepo.Upsert`（`repos.go:1316-1323`）只写 5 列（asset_id, tag_key, tag_value, tag_type, source_type），**漏写** source_name / source_version / run_id —— 这是代码 bug，schema 早就有这些列。

**rev.12 必修**：
```go
// repos.go AssetTagRepo.Upsert 增加 4 个参数
const q = `
INSERT INTO asset_tags (asset_id, tag_key, tag_value, source_type, source_name, source_version, run_id, applied_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (asset_id, tag_key, tag_value, source_type, source_version_norm)
DO UPDATE SET applied_at = NOW(), updated_at = NOW()
RETURNING ...
`
```

---

## 4. 5+ 类 tag source（开放扩展）

### 4.1 5 类已知 + N 类未来

| source | 写入方 | 用途 | 例 |
|--------|-------|------|------|
| **`algo_sdk`** | 算法 worker | 算法身份标签 | `algo.hand_track.version=2.0`、`algo.hand_track.status=ok` |
| **`rule_engine`** | 规则引擎（定时跑）| 规则派生 | `quality=ok`（规则 `quality_check@1.0` 给的）|
| **`human`** | 标注员 / 运营手工 | 人工标注 | `scenario=kitchen`（labeler_007 贴的）|
| **`system`** | 平台自动 / 运营配置 | 业务 / 客户 | `customer=cust_alpha`、`purpose=training` |
| **`compliance`** | 合规审查员 | 合规审查 | `compliance.pii=true`、`compliance.review_required=true` |
| **`llm`**（未来）| LLM 自动标签 | LLM 派生 | `llm.scene_caption=厨房洗碗场景` |
| **`vendor`**（未来）| 第三方供应商 | 外部输入 | `vendor.alphaco.quality=premium` |
| **`crowdsource`**（未来）| 众包平台 | 众包标注 | `crowd.appen.label=grasp` |

→ **新增 source = `tag_registry.yaml` 加一行**，不动 schema、不动 ES mapping。

### 4.2 算法身份 tag 是「双写」（TT3 决策）

算法 worker 在 finish 时**同事务双写**：

| 写入 | 表 | 目的 |
|------|----|-----|
| `INSERT/UPDATE asset_algo_latest` | algo 当前态 | 算法状态机 / pin / run_inputs（强类型字段查询）|
| `INSERT asset_tags` | algo.* 命名空间 | 统一 facet 检索（与其他 tag 同语法 filter）|

→ algo.* 既能按状态字段强类型查（pinned / failed / 跑过哪些版本），也能在统一 tags[] facet 里 filter。事务原子性由 AssetWriter 兜底。

---

## 5. tag_registry.yaml 治理

### 5.1 配置结构

```yaml
# tag_registry.yaml（业务可扩展，YAML PR 即可加新规则）

tag_sources:
  # 按 source 维度治理（写权限 / 命名空间 / 传播）
  - source: algo_sdk
    prefix: algo.
    writable_by: [algo_sdk]
    immutable: true

  - source: rule_engine
    prefix: rule.
    writable_by: [rule_engine]
    requires_source_version: true
    recomputable: true

  - source: human
    writable_by: [labeler, ops]
    requires_source_name: true                # human 必填 user_id

  - source: system
    writable_by: [ops]

  - source: compliance
    prefix: compliance.
    writable_by: [compliance_officer]
    immutable: true                            # 合规 tag 写后不能改
    propagation: descendants                   # 沿血缘自动传播（P1.5）
    critical: true                             # 失败 → PagerDuty 告警

  # 未来扩展示例（新增 source = YAML PR）
  - source: llm
    prefix: llm.
    writable_by: [llm_pipeline]
    requires_source_name: true                 # LLM 模型名
    requires_source_version: true              # LLM 版本

  - source: vendor
    prefix: vendor.<vendor_id>.
    writable_by: [vendor_api]

tag_keys:
  # 按 key/prefix 维度治理（value 白名单 + 命名约束）
  - prefix: algo.                              # value 自由
  - prefix: rule.
  - prefix: customer.                          # customer.<id>.<attr>
  - prefix: compliance.
  - key: scenario                              # 严格 key + value 枚举
    allowed_values: [kitchen, warehouse, outdoor, lab, office, other]
  - key: purpose
    allowed_values: [training, eval, replay, demo, qa, other]
  - key: quality
    allowed_values: [ok, warn, rejected]
  - key: review.verdict
    allowed_values: [passed, failed, pending]
```

### 5.2 Validator 检查顺序

1. caller 解析 → 命中 `tag_sources[].source` → 校验 `writable_by` / `requires_source_name` / `requires_source_version` / `immutable`
2. `tag_key` → 命中 `tag_keys[].key` 整条 → 校验 `allowed_values`
3. `tag_key` → 命中 `tag_keys[].prefix` → value 自由
4. 都没命中 → **422**

### 5.3 禁字段（Tag1 不变式）

```
tag_key 禁止 ∈ {label, labels, primary_label, action_id}
→ 任何写入直接 422
```

避免与 action label 混淆（强制走 actions 表）。

---

## 6. action 资产的 5 维标签全溯版本

action 既是「action 实体」（带时间窗 + label）又是「一等资产」（享所有 asset 通用机制），所以一个 action 资产可挂的「标签信息」有 **5 个独立维度**，**每一维都带完整 actor / source / version 血缘**。

| # | 维度 | 表 / 字段 | 多版本契约 |
|---|------|---------|-----------|
| 1 | **action 核心 label**（grasp / pickup） | `actions.primary_label` / `labels[]` | 一版一行：同一时间窗被不同算法版本/标注员识别 → 各自独立 `action_id`，用 `revision_of` 串成版本链，`is_current=true` 标当前版；`source_name + source_version` 记算法/标注员身份 |
| 2 | **action 资产开放 tag**（quality / scenario / customer） | `asset_tags`（asset_id = action_id） | `source` + `source_name` + `source_version`；多源同 key 共存 |
| 3 | **后续算法当前态**（hand_track@2.0 在此 action 上跑过） | `asset_algo_latest`（含 `is_pinned` / `run_inputs`） | per (asset_id, algo_name) 一行 + ES `algo_versions_seen` 累积全历史 |
| 4 | **action 标量评分**（confidence / rating） | `asset_metrics` | `source` + `source_name` + `source_version`；多人多次保留全部 |
| 5 | **QA 评估原档** | `asset_eval_results` | `source` + `source_version`；payload 装完整 reasoning |

### 6.1 ES 端复合查询示例

```text
"找所有：被 action_detector@2.0 标为 grasp + 被 hand_track@2.0 处理过 + QA 平均分 ≥ 4 + 标注员 labeler_007 贴过 scenario=kitchen 的 action 资产"

actions.source_name=action_detector AND actions.source_version=2.0 AND actions.label=grasp
AND algos_current.name=hand_track AND algos_current.version=2.0
AND metrics_aggregates.rating.quality_score.avg >= 4
AND tags.key=scenario AND tags.value=kitchen
AND tags.source_name=labeler_007
```

### 6.2 时间线回放（events）

```text
2026-05-10T10:00  algo_sdk:action_detector@1.0  action_upserted  primary_label: null    → pickup
2026-05-15T09:00  algo_sdk:action_detector@2.0  asset_revised    revision: 1            → 2 (label pickup → grasp)
2026-05-15T10:30  algo_sdk:hand_track@2.0       algo_finished    status: ok, confidence=0.93
2026-05-16T14:00  labeler:labeler_007           tag_upserted     scenario: → kitchen
2026-05-17T11:00  reviewer:alg_zhang            metric_upserted  rating.quality_score: → 4.5
2026-05-18T16:00  qa_pipeline:action_compl@1.2  eval_result_added completeness=0.92
2026-05-19T08:00  propagator:TagPropagator      tag_upserted     compliance.pii: → true  (propagated_from: seg_001)
```

→ **任意一条变更都能追溯到「谁、什么版本、哪次 run、改了什么」**。

---

## 7. API 契约

### 7.1 单条 tag 写入

```text
POST /api/v1/asset-tags
Idempotency-Key: <uuid>
{
  "asset_id": "clipA_v2",
  "tag_key": "scenario",
  "tag_value": "kitchen",
  "source": "human",
  "source_name": "labeler_007",
  "source_version": null
}
```

### 7.2 算法 worker 写入（带 run_id）

```text
POST /api/v1/asset-tags
{
  "asset_id": "clipA_v2",
  "tag_key": "algo.hand_track.version",
  "tag_value": "2.0",
  "source": "algo_sdk",
  "source_name": "hand_track",
  "source_version": "2.0",
  "run_id": "R001abc..."
}
```

实际上 worker **不直接调** asset-tags API，而是在 `POST /algo-runs/{id}/finish` 时由平台**同事务双写**（TT3）。

### 7.3 批量

```text
POST /api/v1/asset-tags:batch
Idempotency-Key: <uuid>
{
  "tags": [
    {"asset_id": "...", "tag_key": "...", ...},
    {"asset_id": "...", "tag_key": "...", ...}
  ]
}
```

### 7.4 删除

```text
DELETE /api/v1/asset-tags
{
  "asset_id": "clipA_v2",
  "tag_key": "scenario",
  "tag_value": "kitchen",
  "source": "human",
  "source_name": "labeler_007"
}
```

只删指定 source 的；不动其他 source 写的 tag（多源共存契约）。

### 7.5 规则引擎重算

```sql
-- 规则升级 v1.0 → v1.1
DELETE FROM asset_tags WHERE source='rule_engine' AND source_name='quality_check' AND source_version='1.0';
-- 然后规则引擎重写 source_version='1.1'
```

→ **不影响 human / algo 的 tag**。

---

## 8. TagPropagator 沿血缘传播（P1.5）

### 8.1 业务问题

「segment_X 上贴了 `compliance.pii=true` → segment_X 切出来的 100 个 clip / action / frame 都应该自动也含 PII」 → 平台自动传播，避免漏标。

### 8.2 适用 vs 不适用

| 标签类 | 应传播？ | 理由 |
|--------|--------|------|
| `compliance.pii=true` | ✓ | segment 含 PII，切出的所有 clip / action / frame 都含 PII |
| `customer.cust_X.exclude=true` | ✓ | 父被排除，子必须排除（避免误交付）|
| `scenario=kitchen` | ✓ | 业务场景天然继承 |
| `algo.hand_track.version=2.0` | ✗ | 算法在每个资产上独立运行 |
| `rule.quality_check=ok` | ✗ | 规则在每个资产上独立评估 |
| `quality=rejected` | ✗ | 子资产可能修复了，质量独立判定 |

### 8.3 实现：TagPropagator ActionHandler

```yaml
# tag_sources 中标 propagation
- source: compliance
  prefix: compliance.
  propagation: descendants                # 沿 split_from / derived_from / contains 边自动传播
  propagation_max_depth: 10
  critical: true                          # 失败触发 PagerDuty
```

**处理流程：**

```text
① 运营贴 tag: compliance.pii=true on seg_001
   → INSERT asset_tags (source=compliance, ...)
   → INSERT asset_events (type=tag_upserted, ...)

② TagPropagator ActionHandler 订阅 tag_upserted event
   → 检查 tag_key 是否在 propagation=descendants 名单
   → 命中：递归找下游 asset（asset_relations CTE）
   → 给每个下游 INSERT asset_tags (source=propagated_from:seg_001, key=compliance.pii, value=true)

③ B 路由生新版本时（asset_revised event）
   → TagPropagator 自动补传播父的可传播 tags 到新版（异步，1-3s 窗口）
```

**TP1/TP2 不变式：**

| # | 规则 |
|---|------|
| TP1 | `propagation=descendants` 的 tag 不在 B 路由事务内复制；由 TagPropagator 订阅 `asset_revised / asset_created` 异步补 |
| TP2 | delivery_rules 校验 PII 类 tag 时**按 logical_asset_id 聚合所有版本**（窗口期保护）；commit C2 在 PG 真源再校验一次 |

### 8.4 PII 窗口风险与缓解

| 风险 | 缓解 |
|------|------|
| B 路由刚生 clipA_v2，propagator 还没补 PII tag（1-3s 窗口）| delivery_rules 按 logical 聚合所有版本 tags，任一版本有 PII 即拒；C2 commit PG 真源再校验 |
| TagPropagator handler 失败 / DLQ | PII tag 在 registry 标 critical=true；handler 失败时额外写 `pii_propagation_failed` metric + Slack 告警 oncall |

### 8.5 与 Atlas 的对比

| 维度 | Apache Atlas | DataBrew TagPropagator |
|------|-------------|---------------------|
| 引擎 | JanusGraph + Kafka | PG `asset_relations` + Actions Framework PG polling |
| 性能 | 强（图原生）| 万级 entity 内可控（recursive CTE）|
| 部署成本 | 高（多组件）| 低（无新组件）|
| 能力等价 | ✓ | ✓ |

### 8.6 Tag 继承的边界场景（具体例子）

借鉴 Atlas 的传播规则细节，明确以下边界（避免实施时模糊）：

#### 场景 A：父 tag 删除时，下游传过去的副本怎么办

```
seg_001 [pii=true,  source=compliance]
  ├─ 传播 → clip_A1 [pii=true, source=propagated_from:seg_001]
  └─ 传播 → clip_A2 [pii=true, source=propagated_from:seg_001]

运营改主意：从 seg_001 删 pii tag
  ├─ DELETE asset_tags WHERE asset_id='seg_001' AND source='compliance'
  └─ TagPropagator 订阅 tag_deleted event：
      → 删所有 source='propagated_from:seg_001' 的副本
```

**规则**：父 tag 删 → 自动级联删传播副本（按 `source='propagated_from:<parent_id>'` 标识）。Atlas 的 `removePropagationsOnEntityDelete` 同款机制。

#### 场景 B：父 tag 改值时，下游怎么同步

```
seg_001 [scenario=kitchen]   →   clip_A1 [scenario=kitchen, source=propagated]
                                  clip_A2 [scenario=kitchen, source=propagated]

运营改：seg_001 [scenario=living_room]
  ├─ UPDATE asset_tags SET tag_value='living_room' WHERE asset_id='seg_001'
  └─ TagPropagator 订阅 tag_updated event：
      → 找出所有 source='propagated_from:seg_001' 的副本
      → 逐个 UPDATE tag_value='living_room'
```

**规则**：父 tag 值变 → 异步同步所有传播副本。

#### 场景 C：下游被人手动改了，传播过来的值要不要覆盖

```
seg_001 [scenario=kitchen]   →   clip_A1 [scenario=kitchen, source=propagated]

人工改：clip_A1 上手动加 [scenario=living_room, source=human:rick]
  → 现在 clip_A1 同时有两条：
     {source=propagated, value=kitchen}
     {source=human, value=living_room}

后来父改：seg_001 [scenario=outdoor]
  → propagated 副本更新成 outdoor
  → human 那条不动（多源共存契约）
```

**规则**：传播副本只动 `source='propagated'` 的行，**人工 / 算法的 tag 不被覆盖**。delivery rules 看「这个 asset 当前是什么场景」时按业务逻辑决定优先级（一般 human > propagated）。

#### 场景 D：传播深度限制

```
seg_001 → clip_A1 → frame_F1   (深度 2)
                  → action_X1  (深度 2)
        → clip_A2                (深度 1)
```

`tag_registry.yaml` 里 `propagation_max_depth: 10`：

- 深度 ≤ 10 → 正常传播
- 深度 > 10 → 停止传播 + 记 warning metric（理论上不可能，但兜底防止血缘环）

#### 场景 E：批量传播性能

10000 个下游 asset 一次传播：

| 实现 | 延迟 | 风险 |
|---|---|---|
| 同步事务 INSERT 10000 行 | 几秒 + 锁 asset_tags 表 | ❌ 阻塞主流程 |
| ✅ TagPropagator 异步分批（每批 100）| 1-3s 总延迟 | 业务方接受窗口期 |

→ 这就是 TP1 不变式「不在 B 路由事务内传播」的原因。

---

## 9. 批量打标 API（Mass Tagging）

### 9.1 业务需求

运营 / 合规 / 算法回填经常要给一**批 asset** 同时打或删标签，循环调单条 API 慢且无事务保证。

| 场景 | 影响范围 |
|---|---|
| GDPR 审计：客户 X 历史所有 asset 打 `pii_review_2026q1=true` | 50 万条 |
| 算法回滚：某次 run 的所有产物标 `quarantined=true` | 1 万条 |
| 重新分类：所有 `scenario=kitchen` 改成 `scenario=indoor.kitchen` | 2 万条 |
| 客户合规交付：批量标 `delivered_to:cust_alpha` | 5000 条 |

### 9.2 两种批量 API（参考 Atlas 设计）

#### 9.2.1 按显式 ID 列表批量

```http
POST /api/v1/asset-tags:bulk-by-ids
Idempotency-Key: <uuid>

{
  "asset_ids": ["aaa11111", "bbb22222", ..., "<最多 1000 个>"],
  "tag": {
    "tag_key": "compliance.pii_review_2026q1",
    "tag_value": "reviewed",
    "source_type": "manual",
    "source_name": "ops_audit",
    "source_version": "v1"
  },
  "reason": "GDPR Q1 audit batch 12"
}

← 200 OK
{
  "tagged": 998,
  "skipped": 2,                          // 已存在同 (key, source) 行
  "audit_run_id": "Rbulk_xyz_001",       // 这次批量也登记一条 algo_runs
  "events_published": 998
}
```

#### 9.2.2 按 filter 批量（最常用）

```http
POST /api/v1/asset-tags:bulk-by-filter
Idempotency-Key: <uuid>

{
  "filter": {
    "asset_type": "clip",
    "logical_id_in": ["L_clipA", "L_clipB"],
    "created_at_lt": "2026-01-01T00:00:00Z",
    "tag.scenario": "kitchen"           // 已有 tag 也能筛
  },
  "tag": {
    "tag_key": "compliance.pii",
    "tag_value": "true",
    "source_type": "rule_engine",
    "source_name": "compliance_check",
    "source_version": "1.2"
  },
  "max_affected": 10000,                  // 安全阀，超就拒
  "dry_run": false
}

← 202 Accepted
{
  "audit_run_id": "Rbulk_xyz_002",
  "estimated_count": 4523,
  "status_url": "/api/v1/algo-runs/Rbulk_xyz_002"
}
```

后台异步分批跑（每批 500 行），状态进 `algo_runs`，可查进度。

#### 9.2.3 dry_run 预览

`dry_run=true` 时不写库，只返回 affected count 和样本 10 行预览。运营常用：

```json
{
  "dry_run": true,
  "estimated_count": 4523,
  "sample_assets": ["aaa11111", "bbb22222", ...],
  "warnings": ["3 个 asset 已被 pin，会跳过"]
}
```

### 9.3 与 algo_runs 关联

每次批量打标**也登记一条 `algo_runs`**（`algo_name='bulk_tagging'`），意义：

- 一次批量出问题可一键回滚（按 audit_run_id 反向 DELETE）
- 审计「过去一年所有 GDPR 批量打标操作」一条 SQL
- 跟普通 algo run 共用统计 / 监控基础设施

### 9.4 必须的限速 / 并发保护

| 保护 | 默认值 |
|---|---|
| 单次 max_affected | 10000（超过强制拆分）|
| 全局并发 bulk job | 3（防多个 GDPR 同时跑垮 PG）|
| 每用户 QPS | 10 次 / 小时 |
| dry_run 不计入限额 | ✓ |

---

## 10. 未来演进（不急，业务复杂后再做）

### 10.1 Type Hierarchy（标签类型层级）

**现状**：用点号字符串假装层级（`compliance.pii.email` / `compliance.pii.ssn`），DB 不知道是层级关系。查「所有 PII 标签」靠 `tag_key LIKE 'compliance.pii%'`。

**问题出现时**：标签类型超过 50 种 + 经常重组层级（从 `compliance.pii.email` 移到 `personal.email` 下要批量 update 历史数据）。

**演进方案**：加一张 `tag_type_hierarchy` 元数据表：

```sql
CREATE TABLE tag_type_hierarchy (
  tag_key      TEXT PRIMARY KEY,         -- "compliance.pii.email"
  parent_key   TEXT REFERENCES tag_type_hierarchy(tag_key),
  display_name TEXT,
  description  TEXT,
  owner        TEXT,
  is_deprecated BOOL DEFAULT false
);
```

查「这个 asset 涉及任何 PII 子类型」用 recursive CTE：

```sql
WITH RECURSIVE pii_descendants AS (
  SELECT tag_key FROM tag_type_hierarchy WHERE tag_key = 'compliance.pii'
  UNION
  SELECT h.tag_key FROM tag_type_hierarchy h
  JOIN pii_descendants p ON h.parent_key = p.tag_key
)
SELECT DISTINCT asset_id FROM asset_tags
WHERE tag_key IN (SELECT tag_key FROM pii_descendants);
```

**优点**：层级显式 + 可视化好做 + 重组历史 tag 不必批量 update（只改 hierarchy 表）。

**何时做**：标签类型超 50 种 / 业务方频繁重组层级时。**P1 不做**。

### 10.2 Glossary 业务术语 vs Classification 技术标签拆分

**现状**：`asset_tags` 一张表混了两类标签：

| 类别 | 例 | 谁维护 |
|---|---|---|
| 业务定义（Glossary）| `scenario=kitchen` / `product_type=grasp_sample` | 运营 / PM |
| 技术标签（Classification）| `compliance.pii=true` / `qa_passed=true` | 工程 / 合规 |

**问题出现时**：业务方说「我想加个 `scenario=extreme_weather` 子类，但工程师说要改代码 / 走 tag_registry YAML PR」—— 治理流程跟不上业务变化。

**演进方案**：拆出独立 `glossary_terms` 表（业务术语字典）：

```sql
CREATE TABLE glossary_terms (
  term_id      TEXT PRIMARY KEY,         -- "scenario.kitchen"
  category     TEXT NOT NULL,            -- "scenario" | "product_type" | ...
  display_name TEXT NOT NULL,
  description  TEXT,
  parent_term  TEXT,
  owner        TEXT,
  status       TEXT DEFAULT 'active'     -- active | deprecated
);

CREATE TABLE asset_glossary_terms (
  asset_id     TEXT REFERENCES assets,
  term_id      TEXT REFERENCES glossary_terms,
  added_by     TEXT,
  added_at     TIMESTAMPTZ,
  PRIMARY KEY (asset_id, term_id)
);
```

`asset_tags` 留给技术标签（合规、QA、流程），`glossary_terms` 归业务术语（场景、产品分类）。

**优点**：业务术语让 PM **自助维护**（专门 admin UI），不必改代码 / tag_registry YAML。

**何时做**：业务方明确要求自助添加业务术语 + 治理流程瓶颈出现时。**P1 不做**。

### 10.3 演进优先级

| 借鉴点 | 何时做 | 工程量 |
|---|---|---|
| **Tag 继承 + 边界场景**（§8） | ✅ P1.5 已规划 | 中 |
| **批量打标 API**（§9） | ✅ **P1 强烈建议** | 中（3-5 天）|
| **Type Hierarchy**（§10.1）| 🟡 标签 > 50 种 / 重组频繁时 | 中 |
| **Glossary 拆分**（§10.2）| 🟡 业务方要自助维护时 | 大 |

---

## 11. 业务场景 walkthrough

### 11.1 算法 finish 双写

```python
# worker
db.algo_run.finish(
    run_id="R001",
    status="ok",
    ...
)

# 平台自动（同事务）：
#   asset_algo_latest (asset_id='clipX', algo_name='hand_track', algo_version='2.0', status='ok', run_id='R001', ...)
#   asset_tags        (asset_id='clipX', key='algo.hand_track.version', value='2.0', source='algo_sdk', run_id='R001')
#   asset_events      (type='algo_finished', actor='algo_sdk:hand_track@2.0', system_metadata={run_id:'R001'})
```

### 11.2 多源 tag 协作

```text
T1: 算法 quality_check@1.0 跑出 clipX → tag (quality=ok, source=rule_engine, source_name=quality_check, source_version=1.0)
T2: 标注员 labeler_007 复审 clipX → tag (quality=ok, source=human, source_name=labeler_007)

PG 状态：
  clipA   quality   ok   rule_engine   quality_check   1.0
  clipA   quality   ok   human         labeler_007    NULL

→ UI 显示「✓ 自动 + 人审 双确认：质量 ok」（信任度叠加）
```

### 11.3 客户私有标签

```text
POST /api/v1/asset-tags
{
  asset_id: 'clipA_v2',
  tag_key: 'customer.cust_alpha.priority',
  tag_value: 'high',
  source: 'system'
}

→ 「cust_alpha 标记这个资产是高优先级」
→ tag_key 前缀 customer.<id>.* 隔离不同客户
→ delivery_rules 可写：「customer.cust_alpha.priority=high 的资产优先交付」
```

### 11.4 合规审计反查

```sql
-- 「这个客户的 delivery 含 PII 资产吗」
SELECT DISTINCT d.delivery_id, di.asset_id, t.tag_value
FROM deliveries d
JOIN delivery_items di USING (delivery_id)
JOIN asset_tags t ON t.asset_id = di.asset_id
WHERE d.customer_id = 'cust_alpha'
  AND t.tag_key = 'compliance.pii'
  AND t.tag_value = 'true';
```

---

## 12. 不变式与边界

| # | 规则 | 实现 |
|---|------|------|
| **Tag1** | tag_key 禁止 ∈ `{label, labels, primary_label, action_id}` | DB CHECK / Validator |
| **TT3** | algo_sdk 写 asset_algo_latest 同事务必须双写 asset_tags(source=algo_sdk, key=algo.<name>.version) | AssetWriter |
| **Tag-Source-1** | source 必须在 tag_registry.yaml `tag_sources[].source` | Validator |
| **Tag-Source-2** | `requires_source_name=true` 时 source_name 必填 | Validator |
| **Tag-Source-3** | `requires_source_version=true` 时 source_version 必填 | Validator |
| **Tag-Source-4** | `immutable=true` 的 source 写后不允许 UPDATE/DELETE | Validator + DB 权限 |
| **Tag-Key-1** | tag_key 必须命中 tag_registry.yaml `tag_keys[].key` 或 `prefix` | Validator |
| **Tag-Key-2** | `allowed_values` 限定时，value 必须在列表 | Validator |
| **TP1**（P1.5）| `propagation=descendants` tag 不在 B 路由事务内复制；异步补 | TagPropagator |
| **TP2**（P1.5）| delivery_rules 校验 PII 按 logical 聚合所有版本 | DeliveryRuleEvaluator |

---

## 13. 与其他模块的关系

| 模块 | 关系 |
|------|------|
| **`asset-hierarchy-and-derivatives.md`** | 任何 asset_type 都可挂 tag（含 raw_mcap）|
| **`asset-versioning.md`** | tag 跟随具体 asset_id（不跟 logical）；新版本上线时算法 tag 重新写 |
| **`algo-runs.md`** | algo finish 双写 asset_tags(source=algo_sdk)；tag 的 run_id 外键 到 algo_runs |
| **`customers-and-deliveries.md`** | tag.customer.<id>.* 命名空间隔离不同客户；exclude_tags 用 tag 表达 |
| **`annotation-tasks-p1.5.md`** | 标注完成时通过 actions 表写 label（不写 tag）；元数据可写 tag |

---

## 14. 不做的事（Out of Scope）

| 不做的事 项 | 原因 |
|-------|------|
| **tag_value 类型化**（如 int / bool 列）| 简单 K-V 用 TEXT 就够；数值走 metric |
| **tag 历史版本**（rev / supersede）| append 多源记录 + applied_at 已能表达；不另起 tag history 表 |
| **tag 跨 asset_type 限定**（如 scenario 只能打在 segment）| tag 跨实体设计，业务上灵活；约束放在 ES query 层 |
| **统一 tag/label/metric/eval 为一个表** | 4 种语义不同（rev.7 §7.0.3 反例验证），强行统一会让查询混乱 |
| **TagPropagator 同步在 B 路由事务内复制** | TP1：B 路由事务太重；异步补 + critical 告警平衡 |

---

## 15. 决策追溯

| 决策 | 来源 |
|------|------|
| tag vs label vs metric vs eval 4 层 | rev.7 §7.0 术语澄清 |
| asset_tags 多源（source_type/source_name/source_version；`source` 为语义名）| rev.5 W1 |
| Tag Source Registry YAML | rev.7 W2 + rev.10 G2 |
| TT3 algo_sdk 双写 | rev.10 第二轮 grill |
| Tag1 禁字段（label / primary_label）| rev.10 提升 |
| `customer.<id>.*` 命名空间 | rev.7 设计 |
| TagPropagator 沿血缘传播（P1.5）| rev.10 R2 + Atlas 借鉴 |
| TP1/TP2 PII 窗口缓解 | rev.10 第二轮 grill |
| Upsert 补 source_name/source_version 落地 | rev.12（修代码 bug）|
