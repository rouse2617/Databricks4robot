# `asset_events` 收敛 / 砍薄方案

## 1. 结论

`asset_events` **不删除**，但必须从“全平台统一业务事件总线”收窄为：

> **资产域 append-only 业务事件流**

当前它继续承担的职责只保留三类：

1. **资产域用户可见历史**：资产
2. 历史
3. **资产域审计 / 回放事实源**：按 `event_seq` 可追溯、可补偿
4. **Bronze 入湖业务事件源**：`asset_events` 的 CDC → Bronze

当前它**不应该**再承担的职责：

1. **Elasticsearch 主同步源**：ES 已经切到 current-state CDC
2. **全平台所有子域统一事件总线**：delivery / training / eval / metrics / dataset 不要默认接入
3. **泛化“任何 PATCH 都记一条事件”**：应优先保留有明确业务语义的事件，而不是噪音事件

一句话：

> `asset_events` 该保留为“资产域事件流”，不该继续膨胀成“平台级事件垃圾桶”。

---

## 2. 为什么要砍薄

当前仓库里，`asset_events` 已经有真实消费者，不是“未来可能会用”：

- 前端资产详情“全部事件”直接展示 `asset_events`
- Tag 历史接口直接筛 `tag_upserted / tag_deleted`
- 算法生命周期历史直接筛 `algo_*`
- Bronze CDC 只消费 `asset_events`

但是文档口径已经把它扩张成“平台所有域的统一业务事件表”，这会带来三个问题：

1. **职责膨胀**
  - 资产、tag、algo 之外的 delivery / training / eval / metrics 都想往里塞
  - 表面统一，实际变成“所有团队都往一张表写”
2. **事件噪音上升**
  - 同一用户动作可能同时产生 `asset_updated`、`asset_lifecycle_changed`、`tag_upserted`
  - 读时间线的人看不懂，做 Bronze 的人也拿到过多重复语义
3. **未来扩展成本反而更高**
  - 一旦把所有域都绑定到 `asset_events`，后续想拆域会很痛
  - payload schema 会快速失控，review / 版本演进成本飙升

所以这里不是“要不要事件表”的问题，而是：

> **事件表的边界必须收窄到当前代码真正成熟、真正被消费的资产域。**

---

## 3. 新边界：什么能进 `asset_events`

只有同时满足以下条件的事件，才允许进入 `asset_events`：

1. **强绑定单个 asset**
  - 事件能自然挂在一个 `asset_id` 上
2. **有明确用户语义**
  - 用户、前端、审计、运维能说清“这条事件是什么意思”
3. **确实需要历史轨迹**
  - 不是只看当前态，而是需要“发生过什么”
4. **写入频率可控**
  - 不会产生高频细粒度事件风暴
5. **Bronze 保留这类业务事实有价值**
  - 不是纯技术同步痕迹

### 不满足条件的域，默认不进 `asset_events`

- Delivery 子域
- Dataset / Training 子域
- Eval / Metrics 子域
- MCAP 文件接入子域

这些域如果需要事实留痕，优先顺序是：

1. 自己的领域事实表
2. 该领域自己的 CDC
3. 真有跨域事件总线需求时，再单独设计 domain event stream

而不是默认塞回 `asset_events`。

---

## 4. 事件清单：保留 / 收窄 / 删减

### 4.1 应保留的事件

这些事件和当前代码、前端、Bronze 消费是对齐的，应继续保留：


| 事件                        | 处理  | 理由                    |
| ------------------------- | --- | --------------------- |
| `asset_created`           | 保留  | 资产主时间线起点；Bronze 需要    |
| `asset_lifecycle_changed` | 保留  | 生命周期切换有明确审计语义         |
| `tag_upserted`            | 保留  | Tag 历史、时间线、Bronze 都需要 |
| `tag_deleted`             | 保留  | Tag 历史、时间线、Bronze 都需要 |
| `algo_started`            | 保留  | 算法生命周期核心事件            |
| `algo_finished`           | 保留  | 算法生命周期核心事件            |
| `algo_failed`             | 保留  | 算法生命周期核心事件            |
| `algo_reset`              | 保留  | 算法生命周期核心事件            |
| `algo_unblocked`          | 保留  | 依赖解锁路径需要保留            |


### 4.2 应立即收窄的事件

#### `asset_updated`

当前实现中，`PATCH /assets/{id}` 会**无条件**追加一条 `asset_updated`，如果同时改了 lifecycle 或 tags，还会再追加更具体的事件。这样会产生重复语义。

当前代码位置：

- [backend/internal/usecase/asset/usecase.go](/Users/rick/Databricks4robot/backend/internal/usecase/asset/usecase.go)

建议收窄为：

- **只在“资产元数据字段”变化时产生**
- **不再为以下动作单独追加 `asset_updated`**：
  - 纯 lifecycle 变化
  - 纯 tag 变化
  - 纯 algo 变化

也就是说：

- 改 owner / reviewer / metadata / files / 非生命周期资产字段：可以保留 `asset_updated`
- 改 lifecycle：只记 `asset_lifecycle_changed`
- 改 tag：只记 `tag_upserted` / `tag_deleted`

中期还可以进一步考虑把 `asset_updated` 重命名为 `asset_metadata_updated`，但这不是第一步必须做的事。

### 4.3 应从“规划口径”里马上删减的事件

这些事件当前并不是成熟运行路径，继续保留在主设计文档里，只会扩大 `asset_events` 的职责边界：


| 事件                         | 处理建议                   | 原因                                        |
| -------------------------- | ---------------------- | ----------------------------------------- |
| `mcap_ingested`            | 从 `asset_events` 主合同移除 | MCAP 是文件接入域，不是资产域时间线核心事件                  |
| `delivery_created`         | 从 `asset_events` 主合同移除 | Delivery 有自己表和明细表                         |
| `delivery_item_added`      | 从 `asset_events` 主合同移除 | 高 fan-out，容易制造噪音                          |
| `delivery_completed`       | 从 `asset_events` 主合同移除 | 应由 delivery 子域自己建事实源                      |
| `dataset_snapshot_created` | 从 `asset_events` 主合同移除 | 属于 dataset/training 域                     |
| `training_run_started`     | 从 `asset_events` 主合同移除 | 属于 training 域                             |
| `training_run_finished`    | 从 `asset_events` 主合同移除 | 属于 training 域                             |
| `eval_started`             | 从 `asset_events` 主合同移除 | eval 域应该以原始事实表为主                          |
| `eval_finished`            | 从 `asset_events` 主合同移除 | 同上                                        |
| `eval_failed`              | 从 `asset_events` 主合同移除 | 同上                                        |
| `eval_result_reported`     | 从 `asset_events` 主合同移除 | `asset_eval_results` 才是事实源                |
| `metric_upserted`          | 从 `asset_events` 主合同移除 | `asset_metrics` / `metric_registry` 才是事实源 |
| `metric_deleted`           | 从 `asset_events` 主合同移除 | 同上                                        |
| `metric_rule_triggered`    | 从 `asset_events` 主合同移除 | 同上                                        |


这里的“删减”是指：

> **先从主设计合同里删掉，不再把这些事件视为默认应进入 `asset_events` 的标准做法。**

不是说未来永远不能有这些事件，而是：

- 它们如果要做，必须重新单独评审
- 默认做法不再是“直接接入 `asset_events`”

---

## 5. 代码层面怎么改

### 5.1 第一优先级：收窄 `asset_updated`

当前问题：

- `PATCH /assets/{id}` 先写 `asset_updated`
- 如果 lifecycle 变化，再写 `asset_lifecycle_changed`
- 如果 tags 有变化，再写 `tag_upserted`

这会让一个动作生成 2–N 条语义重叠事件。

#### 建议改法

在 `backend/internal/usecase/asset/usecase.go` 中：

1. 先计算本次 PATCH 的变更集合
2. 仅当存在“通用资产元数据字段变化”时才写 `asset_updated`
3. 如果只有 lifecycle 变化：
  - 只写 `asset_lifecycle_changed`
4. 如果只有 tag 变化：
  - 只写 `tag_upserted` / `tag_deleted`

建议把“哪些字段算 metadata update”写成显式 helper，不要靠散落的 if 判断。

#### 影响文件

- [backend/internal/usecase/asset/usecase.go](/Users/rick/Databricks4robot/backend/internal/usecase/asset/usecase.go)
- [backend/internal/usecase/asset/usecase_projection_test.go](/Users/rick/Databricks4robot/backend/internal/usecase/asset/usecase_projection_test.go)
- [backend/internal/handlers/asset/handler_test.go](/Users/rick/Databricks4robot/backend/internal/handlers/asset/handler_test.go)

### 5.2 第二优先级：保持 tag / algo 专项路径只产专项事件

当前方向是对的，应继续保持：

- tag 专项接口只产 `tag_upserted` / `tag_deleted`
- algo 专项接口只产 `algo_`*

不要为了“统一”再回头补一层 `asset_updated`。

相关文件：

- [backend/internal/usecase/asset/usecase.go](/Users/rick/Databricks4robot/backend/internal/usecase/asset/usecase.go)
- [backend/internal/usecase/asset/algo_usecase.go](/Users/rick/Databricks4robot/backend/internal/usecase/asset/algo_usecase.go)

### 5.3 第三优先级：未来新域默认禁止直接往 `asset_events` 写

如果后续要做：

- delivery handler / usecase
- eval handler / usecase
- training / dataset handler

默认策略应改成：

> **先写领域事实表，不默认 append `asset_events`。**

只有当该域满足“单 asset 绑定 + 用户可见历史 + Bronze 业务事实有价值”三条件时，才允许加回到 `asset_events`。

这意味着未来评审应该明确拒绝以下默认模式：

```text
新域上线
  -> 顺手往 asset_events 塞一个 event_type
```

而应改成：

```text
新域上线
  -> 先有自己的领域表 / 领域 CDC
  -> 如确有必要，再单独评审是否进入 asset_events
```

### 5.4 CDC 代码本身不需要大改

当前 CDC 分工是合理的：

- `asset_events` CDC → Bronze
- current-state tables CDC → ES

这部分不建议改。

相关文件保持现状即可：

- [backend/internal/cdc/bronze_consumer.go](/Users/rick/Databricks4robot/backend/internal/cdc/bronze_consumer.go)
- [backend/internal/cdc/es_consumer.go](/Users/rick/Databricks4robot/backend/internal/cdc/es_consumer.go)

---

## 6. 文档口径怎么同步收窄

这次不建议一次性重写所有设计文档，但要明确哪些文档需要同步。

### 6.1 需要优先同步的文档

#### 1) `data-platform-design.md`

重点收窄这些口径：

- 把 `asset_events` 从“统一业务事件 / outbox / 全平台消费入口”收窄成“资产域业务事件流”
- 删除或降级 delivery / training / eval / metrics 事件清单
- 移除“ES 从 asset_events 直接同步”的遗留叙述
- 把 `Outbox Worker` / `publish_state + polling` 的现役口径改成历史路径说明

#### 2) `schema-reference.md`

重点收窄：

- 事件类型列表只保留资产域事件作为主合同
- 明确“其他域事件不再默认进入 `asset_events`”

#### 3) `use-cases.md`

重点收窄：

- “核心写 API → 事件 → 下游消费”图不要再把所有未来域都默认指向 `asset_events`
- 去掉或降级 M5/M6 这类旧 outbox worker 运维项

#### 4) `algo-lifecycle-and-data-model.md`

重点收窄：

- 把“统一事件 / outbox”改成“资产域统一事件流”
- 保留 algo 子域对 `asset_events` 的依赖，因为这是合理且成熟的

#### 5) `eval-metrics-design.md`

重点收窄：

- 不再把 `eval_result_reported / metric_upserted` 视为默认进入 `asset_events`
- 先强调 `asset_eval_results / asset_metrics` 是事实源

#### 6) `api-guide.md`

重点收窄：

- 事件类型示例与事件过滤说明同步更新
- 移除老 outbox worker 指标 / 口径残留

### 6.2 推荐同步方式

推荐顺序：

1. **先出本文件作为裁决文档**
2. `docs/review/README.md` 挂入口
3. 后续分 PR 修各主文档正文

原因：

- 当前 review 文档之间已有较多历史分叉
- 一次性全文重写风险高
- 先有一份明确的“收窄裁决”更利于后续逐篇修正

---

## 7. 推荐执行顺序

### Phase A：先冻结边界

输出：

- 本文档生效
- 评审口径改成“资产域事件流”

### Phase B：最小代码收敛

只做一件关键事：

- 收窄 `asset_updated` 的发射条件

这是当前最立刻、最容易、收益最大的降噪动作。

### Phase C：主文档口径修正

至少修：

- `data-platform-design.md`
- `schema-reference.md`
- `use-cases.md`
- `algo-lifecycle-and-data-model.md`
- `eval-metrics-design.md`
- `api-guide.md`

### Phase D：未来新域接入门禁

形成一个简单规则：

> 新事件类型进 `asset_events` 之前，必须回答：
>
> 1. 这是不是单 asset 绑定事件？
> 2. 这是不是用户可见 / 审计需要的历史？
> 3. 这是不是应该进入 Bronze 的业务事实？

有任一项答不上来，就不要接入 `asset_events`。

---

## 8. 最终建议

我建议按下面这条主线执行：

1. **保留 `asset_events`**
  - 不删表，不删 Bronze 路径
2. **立即砍薄它的职责**
  - 从“全平台统一事件流”收窄为“资产域事件流”
3. **立即收窄 `asset_updated`**
  - 这是当前最明显的噪音源
4. **立即从主设计合同里去掉跨域事件默认接入**
  - delivery / training / eval / metrics 不再默认挂到 `asset_events`
5. **保持当前 CDC 分工不变**
  - current-state CDC → ES
  - `asset_events` CDC → Bronze

最终目标不是“事件越少越好”，而是：

> **只保留那些真正有业务语义、有人消费、值得长期沉淀到 Bronze 的资产域事件。**

