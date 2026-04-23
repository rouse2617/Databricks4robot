# 09 · Asset 数据契约(V1)

> 适用范围:Phase 0 / Phase 1  
> 上游约束:[ADR-002](adr/ADR-002-segment-centric-asset.md) / [ADR-007](adr/ADR-007-write-path-and-event-contract.md) / [03-data-model-wide-table.md](03-data-model-wide-table.md)

---

## 1. 核心结论(先看这个)

1. **Asset = Segment**。平台内不存在"整文件资产"和"片段资产"两套模型。
2. **Recording(MCAP 物理文件)不是资产**,只是资产的上游来源。
3. **算法部门只处理 Segment Asset**。默认只消费 `qa.state=approved` 且 `core.status=active` 的资产。
4. 每个可消费 Segment 都必须有独立 `asset_id(UUID)` 与 `asset_urn`。

---

## 2. 实体边界

### 2.1 Recording(非资产)

- 含义:用户上传的原始 MCAP 文件
- 角色:承载原始字节 + topic + time 范围
- 生命周期:可归档,删除前必须检查下游 segment 依赖

### 2.2 Asset(资产,= Segment)

- 含义:MCAP 中一段时间区间 `[start_ns, end_ns)`
- 主键:`asset_id`(UUIDv4)
- 外部标识:`urn:grace:asset:<uuid>`
- 语义:平台的一等公民,所有标注/QA/派生结果都挂在 asset 上

### 2.3 File(资产附件)

- 含义:与 asset 关联的文件引用或派生产物
- 示例:`raw_mcap` / `sam2_mask` / `human_annotation` / `materialized_segment`
- 规范来源:[schemas/fields.yaml](../schemas/fields.yaml)

---

## 3. 资产创建与可消费规则

## 3.1 创建时机

V1 推荐沿用当前链路:

1. ingestion 完成后创建 1 个**种子 segment asset**(覆盖整段,`qa.state=pending`)
2. QA 在种子上切出候选 segment(调用 `split`)
3. 候选 segment 经 QA 审核,`approved` 后进入可消费池

> 这样可同时满足"全链路可追溯"和"算法只处理有效 segment"。

### 3.2 可消费判定(算法入口)

默认判定:

```text
consumable = (qa.state == approved) AND (core.status == active) AND (deleted_at is null)
```

可选扩展(需显式开关):

- 允许 `qa.state=disputed` 进入灰度评估集(默认关闭)

---

## 4. 状态模型(必须区分两层)

### 4.1 QA 状态(`cf:qa.state`)

以 [schemas/fields.yaml](../schemas/fields.yaml) 为准:

```text
pending -> in_review -> approved | rejected | disputed -> archived
```

### 4.2 资产状态(`cf:core.status`)

```text
active | archived | deleted
```

说明:

- `qa.state` 表示质量流程结果
- `core.status` 表示资产生命周期管理状态
- 算法消费必须同时满足两层约束(见 3.2)

---

## 5. 字段契约(V1 最小必填)

一个可消费 Segment Asset 至少包含以下字段:

| 维度 | 字段 | 约束 |
| --- | --- | --- |
| 标识 | `asset_id` | UUIDv4,不可变 |
| 标识 | `urn` | `urn:grace:asset:<uuid>` |
| 来源 | `cf:time.mcap_uri` | 必填,指向源 MCAP |
| 时间 | `cf:time.start_ns/end_ns` | 必填,`start_ns < end_ns` |
| 时间 | `cf:time.duration_ns` | 必填,> 0 |
| 状态 | `cf:qa.state` | 必填 |
| 状态 | `cf:core.status` | 必填 |
| 文件 | `cf:file.raw_mcap.*` | 至少包含 `uri/kind/producer` |
| 血缘 | `cf:lineage.up:mcap` | 必填 |
| 审计 | `cf:event` | 至少有 `asset.created` 与 QA 结果事件 |

---

## 6. API 契约(算法与 QA 关心的部分)

> 写路径 / 幂等 / Outbox 以 [ADR-007](adr/ADR-007-write-path-and-event-contract.md) 为准。  
> 所有写请求都要求 `x-request-id`。

### 6.1 QA 切段

```http
POST /v1/assets/{parent_asset_id}/split
```

请求体(示例):

```json
[
  {
    "start_ns": 1721520012300000000,
    "end_ns": 1721520057800000000,
    "tags": {"Scene.urban": 1, "Weather.rainy": 1}
  }
]
```

返回:

- 新建子 segment 的 `asset_id/urn` 列表
- 子 segment 初始 `qa.state=pending`(或 `in_review`,由队列策略决定)

### 6.2 QA 审批

```http
POST /v1/assets/{asset_id}/qa/approve
POST /v1/assets/{asset_id}/qa/reject
POST /v1/assets/{asset_id}/qa/dispute
```

语义:

- 审批成功后发出 `qa.approved` 相关 MCL
- 仅 `approved` 默认进入算法可消费池

### 6.3 算法消费查询

```http
POST /v1/assets/search
```

默认过滤:

- `qa_state = approved`
- `status = active`

建议请求中显式带出:

- tag 条件
- 时间范围
- tenant/project 范围

### 6.4 算法写回产物

```http
POST /v1/assets/{asset_id}/files
```

要求:

- `kind/version/uri/producer/run_id` 必填
- 写回时同时更新 `cf:file` + `cf:algo` + `cf:event`

---

## 7. 幂等与去重规则

### 7.1 API 幂等

- 写请求统一 `x-request-id` 去重(24h 窗口)
- 相同请求重复提交返回首次结果,不重复发事件

### 7.2 Segment 语义去重

对于同一 parent/recording 的 split,建议去重键:

```text
dedupe_key = recording_id + start_ns + end_ns + qa_round
```

规则:

- 完全相同区间重复提交:返回已存在资产或报 `409`(按实现二选一,需固定)
- 重叠区间:V1 默认允许(不同业务视角可并存),后续可加队列策略限制

---

## 8. 算法部门工作约定(最重要)

算法团队只需要遵循三条:

1. **只认 `asset_id`** 作为输入主键
2. **只消费 approved+active** 的 segment 资产
3. **所有输出都回写到同一 `asset_id`**(不要另起一套主键)

推荐执行模式:

1. 先调用搜索 API 拿到资产清单
2. 固化一次 `snapshot_id`(保存本次资产列表,保证可复现)
3. 对列表逐个 `stream read -> run -> files.add`

---

## 9. 验收标准(Definition of Done)

上线前至少满足:

1. QA 批准一个 segment 后,系统返回独立 `asset_id + urn`
2. 算法查询默认拿不到 `pending/rejected` 资产
3. `asset.files.add` 能把产物稳定挂回原 `asset_id`
4. 任一资产都能从 `cf:lineage` 追溯到源 MCAP
5. 重试写请求不会重复创建资产或重复发 MCL

---

## 10. 与现有文档关系

- 模型定义: [04-mcap-and-segment.md](04-mcap-and-segment.md)
- 宽表字段: [03-data-model-wide-table.md](03-data-model-wide-table.md)
- 写路径与事件: [ADR-007](adr/ADR-007-write-path-and-event-contract.md)
- 状态/事件枚举: [schemas/fields.yaml](../schemas/fields.yaml)
