# 文档口径收口清单（Pure CDC）

> 目标：把 `docs/review` 内与同步链路相关的描述统一到同一口径：  
> **生产主路径 = PostgreSQL WAL -> Debezium/Kafka -> backend CDC consumers -> Elasticsearch + Iceberg**。  
> `reindex` 仅人工恢复；不作为常驻同步机制。

## 1. 收口原则（统一口径）

- 不再使用“Outbox Worker 30s 轮询是主链路”作为当前/目标表述。
- `asset_events` 继续作为业务事件主线，但下游同步语义以 CDC 消费为准。
- “轮询/outbox”相关内容仅可作为历史路径或迁移背景，必须显式标注“非当前生产路径”。
- 对外评审、上线、SRE 文档中，统一使用 `Pure CDC` 表述。

## 2. 优先级与影响范围

| 优先级 | 文件 | 问题类型 | 建议动作 |
|---|---|---|---|
| P0 | `data-platform-design.md` | 主文档仍混有 outbox 作为主链路描述 | 立即收口，避免评审口径分裂 |
| P1 | `eval-metrics-design.md` | eval 同步仍写 outbox worker | 改成 CDC sink/consumer 术语 |
| P1 | `sql.md` / `schema-reference.md` | outbox 术语较多，易误读为运行主路径 | 保留历史字段说明，但补“运行时以 CDC 为准”注释 |
| P2 | `outbox-test-plan.md` / `outbox-worker-design.md` | 历史文档可能被误当现行 runbook | 文档头部加“历史方案”警示，并链接 Pure CDC runbook |

## 3. 逐文件整改清单

## 3.1 `data-platform-design.md`（P0，必须先改）

### 需要替换的口径（示例）

- “`asset_events` outbox + Go worker（30s 轮询）”  
  -> 改为 “`asset_events` 作为事件主线，PG WAL 经 Debezium/Kafka 进入 CDC consumers”
- “不引 Debezium/Kafka，走 Outbox 自写 Worker”  
  -> 改为 “2.0 同步主路径为 Debezium/Kafka + CDC consumers”
- “outbox sink / cursor / publish_state 作为主同步协议”  
  -> 改为 “consumer offset + 幂等写入为主协议；`event_seq` 用于业务去重/回放键”

### 建议新增小节

- 在文档前部新增“术语与历史路径说明”：
  - 当前生产：Pure CDC
  - 历史实现：Outbox Worker（仅参考）
  - 故障恢复：人工 `reindex`

### 验收标准

- 文档中不再出现“Outbox Worker 是当前主路径”的表述。
- 与 `pure-cdc-go-live-runbook.md`、`cdc-rollout-plan.md` 的架构图和术语一致。

## 3.2 `eval-metrics-design.md`（P1）

### 需要修正

- “COMMIT 后由 outbox worker 推送 ES / Lakehouse”  
  -> “COMMIT 后由 CDC consumer 同步至 ES / Lakehouse”
- “metric_upserted outbox lag”  
  -> “metric_upserted CDC lag”

### 验收标准

- Eval/Metrics 章节与主架构统一使用 CDC 语义，无 outbox 主路径描述。

## 3.3 `schema-reference.md` 与 `sql.md`（P1）

### 处理策略

- 保留 `outbox_sink_cursors` / `outbox_dlq` 等历史或兼容字段的“结构说明”。
- 但在相关章节首行加注：  
  “运行时同步主路径以 Pure CDC 为准；本节字段仅用于历史兼容/迁移参考（如适用）。”

### 验收标准

- 不删除历史字段说明，但不会让读者误认为其是当前主路径。

## 3.4 `outbox-test-plan.md` 与 `outbox-worker-design.md`（P2）

### 处理策略

- 在文档首段加醒目说明：
  - 历史方案文档
  - 非当前生产默认路径
  - 当前上线请以 `pure-cdc-go-live-runbook.md` 为准

### 验收标准

- 新同学读到文档时不会误用 outbox 方案作为上线基线。

## 4. 建议执行顺序（两次 PR）

- PR-1（当天可完成，P0）  
  - `data-platform-design.md`
  - 本清单文档
- PR-2（次日完成，P1/P2）  
  - `eval-metrics-design.md`
  - `schema-reference.md`
  - `sql.md`
  - `outbox-test-plan.md`
  - `outbox-worker-design.md`

## 5. 快速自检脚本（提交前）

在仓库根目录执行：

```bash
rg "outbox worker|30s 轮询|不引 Debezium|不引 Kafka" docs/review/*.md
rg "Pure CDC|WAL|Debezium|Kafka|consumer lag" docs/review/*.md
```

预期：

- 第一条命令只在历史文档中命中，并且有“历史方案”标注。
- 第二条命令在主文档和 runbook 中形成稳定主线。
