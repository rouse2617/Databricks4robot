# Outbox Worker MVP 设计

> **⚠️ 本文档描述的是 2.0 目标态 / MVP 实施方案，不代表 1.0 当前 runtime 已启用 Outbox Worker。**  
> 1.0 已有 `asset_events` 统一事件表在线写入，但无下游消费；本文档定义 2.0 启用 Worker 后的并发模型、cursor 协议与验收标准。

---

## 1. 目标与非目标

### 1.1 目标


| #   | 描述                                                                   | 验收                                            |
| --- | -------------------------------------------------------------------- | --------------------------------------------- |
| G1  | PG `asset_events` 写入后，对应 asset 在 ES 可查时延 P99 < 60 s（tick 上限 + drain） | 集成测试，每秒 50 events 持续 60 s                     |
| G2  | Worker 重启不丢事件                                                        | 杀 worker 后写 1k events 再重启，ES 最终全量到位           |
| G3  | 同一 asset 多次更新合并为最少 ES 写次数                                            | 1k events × 100 asset → ≤ 100 次 BulkIndex doc |
| G4  | ES 索引误删后可从任意 `event_seq` 重放重建                                        | reindex API 从 0 重放，doc 数与 PG `assets` 一致      |
| G5  | Cursor 推进不丢事件（即使并发 worker 标 `published` 顺序乱）                         | 注入乱序 ack，验证 `safe_horizon` 推进单调               |


### 1.2 非目标（MVP 不做）

- Iceberg Bronze staging Sink（二期）
- 多 worker 实例 SKIP LOCKED 并发拉取（先单 worker，预留接口）
- 向量库 Sink（3.x）
- 把 `asset.create / asset.update / tag.set` 业务侧补 `Append`（独立改动，不属于 worker 范畴）
- DLQ 表（先用 `retry_count` + 告警 + 人工 reindex 兜底）
- Web UI 监控（先靠 metrics + Feishu 告警）

### 1.3 两种正交机制：`publish_state` 字段驱动 + `event_seq` 寻址

第一次读容易把这两件事混在一起，先讲清楚——它们**正交**、**各管各的**：


| 维度                | `asset_events.publish_state` 字段                                                        | `asset_events.event_seq`（BIGSERIAL） |
| ----------------- | -------------------------------------------------------------------------------------- | ----------------------------------- |
| **谁用**            | ES Sink worker 自己                                                                      | 别的下游 / reindex / 监控 lag             |
| **干什么**           | 决定"这条事件还要不要投"                                                                          | 给事件**永久编号 + 永久寻址**                  |
| **取值**            | `pending` → `published` 二态                                                             | `1, 2, 3, …` 单调递增，永不复用              |
| **Worker 怎么用它驱动** | `WHERE publish_state='pending' ORDER BY event_seq LIMIT 1000` —— **就是按"pending 字段"拉的** | 仅作为本批拉取的排序键，不影响驱动力                  |
| **是否影响事件能否被消费**   | 是：`pending` 才会被拉                                                                       | 否：行不删，任意 seq 永远可重读                  |


**典型疑问**：

> "我以为是按字段（`pending`）轮询的？"

**对的**——ES Sink worker 就是这么干的，第 77 行就是这意思。`event_seq` 不是 worker 的驱动力，**它是给"水位线"和"重放"用的另一个维度**：


| 角色                            | 怎么知道还有没有要做的事                                                      |
| ----------------------------- | ----------------------------------------------------------------- |
| ES Sink worker（自己）            | 看 `publish_state='pending'` 字段                                    |
| Iceberg CronJob / 其他直读 PG 的下游 | 看 `outbox_sink_cursors.last_published_seq`，拉 `event_seq > cursor` |
| `/admin/search/reindex` API   | 给一个 `[from_seq, to_seq]` 区间，扫这段事件按 asset_id 折叠后重投 ES              |
| 监控 lag                        | `MAX(event_seq) - last_published_seq`                             |


**"可重放"的物理基础**：`asset_events` 行**永不删**（带长 retention），即使 `publish_state` 已是 `published`，行还在，按 `event_seq` 区间扫历史就能重建任意时刻的下游状态。重放**不会**把 `published` 改回 `pending`——而是另起一个 reindex 流程按 seq 区间扫历史。

> 一句话：**worker 用字段干活，下游 / 重放用 seq 寻址**。两者解耦，所以可以新接下游、可以全量重建 ES、可以重放任意区间，都不影响在线 worker 自己跑。

---

## 2. 总体架构

### 2.1 数据流总览

```mermaid
flowchart LR
    TX["业务事务<br/>同事务写业务表<br/>+ asset_events"]:::biz

    AS["PG · assets /<br/>asset_algo_latest"]:::pg
    AE["PG · asset_events<br/>pending → published"]:::pg

    WK["Outbox Worker<br/>30s tick · drain loop"]:::wk

    ES["Elasticsearch<br/>index: assets"]:::sink
    Adm["/admin/search/reindex<br/>全量重建"]:::adm

    TX -->|"写业务态"| AS
    TX -->|"INSERT pending"| AE
    AS -->|"投影读"| WK
    AE -->|"FetchPending"| WK
    WK -->|"BulkIndex doc_id=asset_id"| ES
    WK -. "ack：mark published + advance horizon" .-> AE
    Adm -. "旁路重写" .-> ES

    classDef biz  fill:#eef2ff,stroke:#5b6cff,color:#1f2a6b;
    classDef pg   fill:#e6f5ec,stroke:#3a9b65,color:#173d27;
    classDef wk   fill:#fff7d6,stroke:#c4a233,color:#5a4708;
    classDef sink fill:#eaf4fb,stroke:#3d8ec9,color:#143a5a;
    classDef adm  fill:#fff5e6,stroke:#d4a574,color:#7a5a2e;
```



> Cursor 表 `outbox_sink_cursors` 不在总览图里画——它只承载"已投递到哪个 `event_seq`"的记账，详见 §3.2。

> Worker 内部链路：`tick → FetchPending → DedupByAsset → Projector.Build → ESSink.BulkIndex → MarkPublishedAndAdvanceCursor`，drain loop 在同一 tick 内反复拉直到 pending 空。详见 §5.1 伪代码。

> 关键：**没有 PG trigger，没有 LISTEN/NOTIFY，没有任何 PG → backend 的主动通道**。
> 业务事务 commit 后，事件静静躺在 `asset_events.publish_state='pending'`，
> 等下一次 tick（最多 30s）被 worker 拉走。

### 2.2 Worker 单次 Tick 的时序

```mermaid
sequenceDiagram
    autonumber
    participant T as Ticker
    participant W as Worker
    participant PG as PostgreSQL
    participant ES as Elasticsearch

    T->>W: 30s tick
    loop drain until pending empty
        W->>PG: SELECT 1000 pending<br/>FOR UPDATE SKIP LOCKED
        PG-->>W: events
        Note over W: dedup + 投影<br/>(读 assets/asset_algo_latest)
        W->>ES: _bulk index docs
        alt 全成功
            ES-->>W: OK
            W->>PG: events→published<br/>+ advance cursor
        else 部分失败
            ES-->>W: 200 + errors[]
            W->>PG: OK→published / 失败→retry++<br/>cursor 停在 MIN(pending)-1
        else 整批失败
            ES--xW: error
            W->>PG: retry++ · 不动 cursor<br/>退出 drain，下次 tick 再来
        end
    end
    opt panic
        Note over W: recover → MarkUnhealthy<br/>可选 os.Exit(1) 让 K8s 重启
    end
```



---

## 3. 同步触发：纯轮询

### 3.1 为什么不用 LISTEN/NOTIFY

详见 §16.1。一句话：1 分钟延迟 SLA 下，NOTIFY 没有任何收益，反而引入：

- PG 端 `pg_notify_queue_usage` 灌满风险（业务事务可能因 trigger 失败而回滚）
- 持久 listener 连接 + 重连逻辑
- 必须保留 tick 兜底 ⇒ 等于两套代码

### 3.2 Schema 改动：零

`asset_events` 表既有 schema 已足够，本设计不需要新增任何 trigger / function / column。
worker 完全靠"按 tick 拉 `publish_state='pending'`"驱动。

---

## 4. Cursor 推进协议（safe_horizon）

### 4.1 为什么不能用 `MAX(本批 event_seq)`

背景：

- `event_seq BIGSERIAL` 是 INSERT 取号、COMMIT 才可见
- 多 worker 并发处理时，`publish_state='pending' → 'published'` 标记顺序与 `event_seq` 顺序无关
- 如果 worker 用 `MAX(本批 event_seq)` 推进 cursor，会越过尚未 commit / 尚未 ack 的更小 seq，**永久漏事件**

下图直观展示这个 race（关键是 `event_seq` 取号顺序 ≠ commit 顺序）：

```mermaid
sequenceDiagram
    autonumber
    participant TxA as Tx A
    participant TxB as Tx B
    participant PG
    participant W as Worker
    participant DS as 下游 (cron)

    TxA->>PG: INSERT seq=100 (in-flight)
    TxB->>PG: INSERT seq=101 + COMMIT
    Note over PG: 表里只见 101，100 仍未 commit
    W->>PG: FetchPending → [101]，处理成功

    rect rgb(255, 230, 230)
    Note over W,DS: ❌ MAX(processed)=101 → cursor=101
    DS->>PG: SELECT WHERE seq > 101
    TxA->>PG: COMMIT → 100 可见，但 cursor 已越过它<br/>seq=100 永远不会被消费
    end

    rect rgb(230, 255, 230)
    Note over W,DS: ✅ safe_horizon = MIN(pending)-1<br/>100 未 commit 时 cursor 卡在 99；commit 后续推
    end
```



> 注意上图是为了说明协议正确性的极端 race；本设计是单 worker，`SKIP LOCKED` 保证不会同时拉到同 seq。但 BIGSERIAL 的 commit-ordering 风险**与 worker 数无关**——只要存在并发的业务事务，就需要 `safe_horizon`。

### 4.2 协议：`safe_horizon = MIN(pending) - 1`

`asset_events.publish_state` 二态：`pending`（默认）→ `published`（worker 投递成功）。

每次推进 `outbox_sink_cursors.last_published_seq` 时执行：

```sql
-- 同事务内：
UPDATE asset_events
   SET publish_state='published', published_at=now()
 WHERE event_id = ANY(:processed_ids);

-- 推进 cursor 到"连续已发布的最大 seq"
UPDATE outbox_sink_cursors
   SET last_published_seq = COALESCE(
         (SELECT MIN(event_seq) - 1
            FROM asset_events
           WHERE publish_state = 'pending'),
         (SELECT MAX(event_seq) FROM asset_events)
       ),
       updated_at = now()
 WHERE sink_name = 'es_assets'
   AND last_published_seq < <计算值>;  -- 单调守卫
```

含义：

- 如果还有 pending 事件，`safe_horizon = MIN(pending) - 1`，cursor 永远不越过最小未确认事件
- 如果没有 pending，cursor = 当前最大 seq
- WHERE 守卫确保 cursor **单调**——并发 worker / 乱序 ack 都不能让它倒退

实现上抽出独立函数便于测试：

```go
// repo.ComputeSafeHorizon 返回应推进到的 cursor 值
//   - 若有 pending → MIN(event_seq WHERE pending) - 1
//   - 若无 pending → MAX(event_seq) 全表
func (r *OutboxRepo) ComputeSafeHorizon(ctx context.Context) (int64, error)
```

### 4.3 cursor 用途

仅供**直接消费 PG 的下游**（如未来的 PyIceberg CronJob、reindex API）使用。**ES Sink 自己不依赖 cursor**——它靠 `publish_state='pending'` 直接驱动，cursor 只是"对外可见的水位线"。

---

## 5. Worker 主循环

### 5.1 伪代码

```go
func (w *Worker) Run(ctx context.Context) (err error) {
    // 顶层 panic 兜底：goroutine panic 不会带崩主进程，但 outbox 会静悄悄死掉
    // 而 /healthz 仍报健康。recover 后置 health = unhealthy，可选让进程
    // os.Exit(1) 由 K8s liveness 自动重启容器。
    defer func() {
        if r := recover(); r != nil {
            log.Error("outbox worker panic", "recover", r, "stack", debug.Stack())
            w.health.MarkUnhealthy("panic: " + fmt.Sprint(r))
            err = fmt.Errorf("outbox worker panicked: %v", r)
            if w.cfg.FatalOnPanic {
                os.Exit(1)
            }
        }
    }()

    ticker := time.NewTicker(w.cfg.TickInterval) // 默认 30s
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
        }

        // drain loop：同一个 tick 内一直拉到 pending 空再退出，
        // 保证 burst 写入不会被 30s 节奏限流。
        for {
            done, err := w.processBatch(ctx)
            if err != nil {
                log.Warn("processBatch failed", "err", err)
                break // 退到外层等下次 tick 重试
            }
            if done {
                break // 本轮 pending 已空
            }
        }
    }
}

// processBatch 处理一批；返回 (done=true) 表示当前 pending 已空。
func (w *Worker) processBatch(ctx context.Context) (done bool, err error) {
    // 1. 拉一批 pending（FOR UPDATE SKIP LOCKED 保证当前 worker 独占）
    events := w.repo.FetchPending(ctx, w.cfg.BatchSize)
    if len(events) == 0 { return true, nil }

    // 2. 按 asset_id 去重（同一 asset 取最大 event_seq）
    //    docToEvents 反向映射：asset_id → 该 asset 关联的所有 event_id
    assetIDs, docToEvents := dedupByAssetID(events)

    // 3. fan-out 投影
    docs := make([]ESDoc, 0, len(assetIDs))
    failedAssets := map[string]error{}
    for _, aid := range assetIDs {
        doc, err := w.projector.Build(ctx, aid)
        if err != nil {
            failedAssets[aid] = err
            continue
        }
        docs = append(docs, doc)
    }

    // 4. ES BulkIndex（doc_id=asset_id 幂等）
    bulkResult, err := w.esSink.BulkIndex(ctx, docs)
    if err != nil {
        // 整批失败：retry_count++，整批不标 published，下次 tick 重试
        w.repo.IncrementRetry(ctx, eventIDs(events), err.Error())
        return false, err
    }

    // 5. 计算"哪些 events 可以标 published"
    //    - ES bulk 部分失败的 doc → 对应 events 保留 pending 等下次重试
    //    - 投影失败的 asset → 对应 events 也保留 pending（projection 临时故障应自愈）
    //    - 其它 events → 标 published
    publishable, deferred := splitByOutcome(events, docToEvents, bulkResult, failedAssets)
    if len(deferred) > 0 {
        w.repo.IncrementRetry(ctx, deferred, "partial failure or projection error")
        w.metrics.BulkPartialFailureTotal.Inc()
    }

    // 6. 同事务：标 published + 推进 cursor（safe_horizon 自动跳过 deferred）
    if err := w.repo.MarkPublishedAndAdvanceCursor(ctx, publishable, "es_assets"); err != nil {
        return false, err
    }
    // batch 满 → 还可能有更多 pending；batch 未满 → pending 已空可退出 drain
    return len(events) < w.cfg.BatchSize, nil
}
```

### 5.2 关键不变量


| 不变量                  | 实现                                                                                                                               |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| 单 worker 独占 batch    | `SELECT … FOR UPDATE SKIP LOCKED LIMIT N`（即使是单 worker，SKIP LOCKED 也能让"reindex API 临时启起的并行任务"不冲突）                                 |
| 整批原子 ack             | `MarkPublishedAndAdvanceCursor` 在单事务内更新 `asset_events` 与 `outbox_sink_cursors`                                                   |
| **ES 写失败不丢事件**       | 整批失败 → 全部保留 pending，下次 tick 重试；**部分失败 → 失败 doc 对应的 events 保留 pending**，其余正常 ack；`safe_horizon` 协议保证 cursor 不越过这些 deferred events |
| 投影失败不丢事件             | `Projector.Build` 失败 → 该 asset 对应 events 保留 pending，下次 tick 重试；不会因为单条投影错误把整批吞掉                                                   |
| Tick 错过 / 进程暂停不丢事件   | `asset_events` 是持久事件源；下一次 tick 自然补上。即便 worker 死 1 小时，事件全部安全堆在 `pending`                                                          |
| Worker 崩溃不丢事件        | 未 commit 的事务回滚，事件仍 `publish_state='pending'`，重启后重新拉                                                                              |
| Worker panic 不静悄悄死   | 顶层 `recover` → 健康降级 + 可选 `os.Exit(1)` 让 K8s liveness 重启                                                                          |
| Burst 场景不被 tick 节奏限流 | drain loop 在同一 tick 内反复 fetch 直到 pending 空，不靠下一次 tick 推进                                                                         |


### 5.3 单条 event 的状态生命周期

```mermaid
stateDiagram-v2
    direction LR
    [*] --> pending: 业务事务 COMMIT<br/>(asset_events INSERT)

    pending --> pending: 投递失败 → retry_count++<br/>(ES 整批/部分失败 · 投影报错 · worker 崩溃回滚)
    pending --> published: 投递成功<br/>(MarkPublished + cursor 推进)

    published --> [*]: 永久保留<br/>(审计 / 回放 / 二期 Bronze 消费)

    note right of pending
        retry_count > 10 → P1 告警
        运维：修 mapping 重试，或 reindex 全量重建
    end note
```



---

## 6. Doc 投影（fan-out）

### 6.1 输入与输出

Projector 输入是一个 `asset_id`，输出是与 ES `assets` 索引 mapping 严格对齐的文档。
为了避免 ES 跨索引 join，Projector 同事务内读取四张 PG 表后做反范式：


| 来源表                 | 用途                                            |
| ------------------- | --------------------------------------------- |
| `assets`            | 主体字段（生命周期、时长、归属、交付汇总）                         |
| `mcap_files`        | 设备 / 场景维度反范式到 `mcap.*` 命名空间                   |
| `asset_tags`        | 投影成 `tags` (nested) + `tags_flat` (flattened) |
| `asset_algo_latest` | 投影成 `algos` (nested)                          |


软删除（`assets.is_deleted=true`）时输出 tombstone，由 ES Sink 调 `DELETE /assets/_doc/<asset_id>`。

### 6.2 字段映射

#### 6.2.1 标识 / 状态


| ES 字段                                               | 来源                                          | 类型                            |
| --------------------------------------------------- | ------------------------------------------- | ----------------------------- |
| `asset_id`                                          | `assets.asset_id`                           | keyword（doc_id）               |
| `mcap_file_id` / `segment_locator`                  | `assets.<col>`                              | keyword                       |
| `asset_type` / `lifecycle_state` / `status`         | `assets.<col>`                              | keyword                       |
| `is_deleted`                                        | `assets.is_deleted`                         | boolean                       |
| `version`                                           | `assets.version`                            | long（reindex 对账 / "新于某次同步"查询） |
| `retention_tier` / `expire_at`                      | `assets.<col>`                              | keyword / date（保留 / 合规过滤）     |
| `tenant_id` / `project_id`                          | `assets.<col>`                              | keyword                       |
| `parent_asset_id` / `root_asset_id` / `asset_level` | `assets.<col>`                              | keyword / int                 |
| `owner` / `reviewer`                                | `assets.<col>`（提升列），多字段：`keyword` + `.text` | 双形态                           |
| `notes`                                             | `assets.metadata->>'notes'` 或提升列            | text                          |
| `metadata`                                          | `assets.metadata` JSONB 整体平铺                | flattened（自定义字段兜底）            |


#### 6.2.2 时间 / 时长


| ES 字段                                     | 来源                                          | 类型                            |
| ----------------------------------------- | ------------------------------------------- | ----------------------------- |
| `start_timestamp_ns` / `end_timestamp_ns` | `assets.<col>`                              | long（纳秒）                      |
| `duration_ms`                             | `assets.duration_ms`                        | long                          |
| `recorded_at`                             | `assets.start_timestamp_ns / 1_000_000`（派生） | date（毫秒，给 date_histogram 聚合用） |
| `created_at` / `updated_at`               | `assets.<col>`                              | date                          |
| `delivery_count`                          | `assets.delivery_count`                     | int                           |
| `last_delivered_at` / `last_delivered_to` | `assets.<col>`                              | date / keyword                |


#### 6.2.3 mcap 反范式（`mcap.*` namespace）


| ES 字段                                              | 来源                                          | 说明                            |
| -------------------------------------------------- | ------------------------------------------- | ----------------------------- |
| `mcap.vendor_id` / `device_id` / `camera_model`    | `mcap_files.<col>`                          | 按设备 / 厂商过滤资产                  |
| `mcap.scene_id` / `location_id` / `environment_id` | `mcap_files.<col>`                          | 按拍摄场景 / 地点过滤                  |
| `mcap.task_id` / `data_source`                     | `mcap_files.<col>`                          | 按采集任务过滤                       |
| `mcap.file_duration_ms`                            | `mcap_files.file_duration_ms`               | 按整段录制时长过滤（"找时长 > 10min 的整段录"） |
| `mcap.recorded_at`                                 | `mcap_files.start_timestamp_ns / 1_000_000` | mcap 维度时间桶聚合                  |


**为什么反范式**：ES 没有原生 join；这些字段查询频率高（例如"找 vendor=A 拍的 segment"），如果不预先平铺到资产 doc，必须先在 ES 拿 asset，再回 PG 取 mcap，链路立刻烂。代价是 mcap 字段更新时需要重新 fan-out 该 mcap 关联的所有 segment（事件 `mcap_metadata_updated`，下游 sink 收到后批量重投相关 asset doc）。

#### 6.2.4 Tags（双形态）


| ES 字段       | 来源                       | 形态                                                                                                  |
| ----------- | ------------------------ | --------------------------------------------------------------------------------------------------- |
| `tags`      | `asset_tags` 全部行投影       | **nested** array：`[{key, value, value_num, value_bool, source_type, source_name, confidence}, ...]` |
| `tags_flat` | 同上，平铺为 `{key: value}` 对象 | **flattened**：给简单等值过滤（`tags_flat.scene = "highway"`）                                                |


**取舍**：90% 的 tag 查询是简单等值，走 `tags_flat` 性能最好；剩下 10% 复杂查询（按 `source_type` / `confidence` 过滤、按 tag 来源做 facet）走 `tags`。两者由同一份 `asset_tags` 投影，存储约多 30%。

不再用 `object + dynamic:true` —— 它会被首次写入的类型锁死，且无法表达 `source_type` / `confidence` 等元信息。

#### 6.2.5 Algos（nested）


| ES 字段   | 来源                        | 形态                                                                                               |
| ------- | ------------------------- | ------------------------------------------------------------------------------------------------ |
| `algos` | `asset_algo_latest` 全部行投影 | **nested** array：`[{name, version, status, result_tag, result_score, run_id, finished_at}, ...]` |


支持复杂组合查询（"hand_tracking.score>0.8 AND face_blur.status=ok"），每个算法一条 nested doc，相互不串。

### 6.3 Tombstone 处理

软删除（`assets.is_deleted=true`）时，`AssetRepo.Get` 返回 nil（已经过滤）。Projector 输出 tombstone marker → ES Sink 调 `DELETE /assets/_doc/<asset_id>` 而非 BulkIndex。

---

## 7. ES Sink 与 Mapping 自检

### 7.1 启动期幂等 mapping 校验

Worker 启动时执行：

```go
func (s *ESSink) EnsureMapping(ctx context.Context) error {
    exists, err := s.client.IndexExists(ctx, "assets")
    if !exists {
        return s.client.CreateIndex(ctx, "assets", embeddedMappingJSON)
    }
    // 索引已存在：检查 mapping 关键字段
    actual, _ := s.client.GetMapping(ctx, "assets")
    // 强校验：字段存在 + 类型匹配。只有"完整 + 类型对齐"才算可写。
    required := map[string]string{
        "asset_id":           "keyword",
        "lifecycle_state":    "keyword",
        "tenant_id":          "keyword",
        "is_deleted":         "boolean",
        "version":            "long",
        "expire_at":          "date",
        "start_timestamp_ns": "long",
        "end_timestamp_ns":   "long",
        "duration_ms":        "long",
        "updated_at":         "date",
        "metadata":           "flattened",
        "tags":               "nested",
        "tags_flat":          "flattened",
        "algos":              "nested",
    }
    if missing := mappingDiff(actual, required); len(missing) > 0 {
        log.Warn("ES mapping incomplete or type mismatch; skipping auto-fix to avoid data loss",
            "missing_or_wrong", missing)
        s.health.MarkDegraded("es_mapping_mismatch")
        // 不自动 DELETE+PUT，记录告警让运维触发 reindex API
    }
    return nil
}
```

设计选择：**绝不自动 DELETE 索引**——即使 mapping 不全也只告警，由人工触发 `POST /admin/search/reindex?rebuild_index=true` 显式重建。这避免"启动期误删生产索引"的灾难场景。`mappingDiff` 同时校验字段缺失与类型不一致（如生产被人手 PUT 成 `keyword` 而 worker 期望 `date`），避免静默数据破坏。

> Mapping JSON 内嵌在 worker 包内，与本地部署初始化 ES 用的 mapping
> 同源同义。CI 加一个 assertion 校验两份内容字节对齐，避免漂移。

### 7.2 BulkIndex 调用

复用现有 ES 客户端的 BulkIndex 方法，新增以下约束：

- 批大小上限：100 docs / batch（ES `_bulk` body 不超过 5 MB 经验值）
- 超时：15s（已是 `Client.httpClient.Timeout`）
- 部分失败：记录失败 doc_id 列表，整批仍标 published

---

## 8. 数据结构对齐（schema 已存在）

仅为评审方便重述。**不需要改 schema**。

### `asset_events`（schema 见数据库设计文档）


| 列                | 类型                      | worker 用途                                                                                                                        |
| ---------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `event_id`       | UUID PK                 | 标 published 时的目标                                                                                                                 |
| `event_seq`      | BIGSERIAL UNIQUE        | 顺序、cursor、回放                                                                                                                     |
| `event_type`     | TEXT                    | 投影时可选过滤（MVP 不用，全部触发 doc 重建）                                                                                                      |
| `aggregate_type` | TEXT, DEFAULT 'asset'   | **MVP worker 不用**，但是未来 Iceberg / 多 sink 的 fan-out 路由 key（按 aggregate_type 分到不同 Bronze 表）。索引 `idx_asset_events_aggregate_time` 已建 |
| `asset_id`       | UUID                    | dedup 与投影主键                                                                                                                      |
| `publish_state`  | TEXT, DEFAULT 'pending' | 'pending' → 'published'（二态）                                                                                                      |
| `published_at`   | TIMESTAMPTZ             | ack 时间                                                                                                                           |
| `retry_count`    | INT                     | 失败计数                                                                                                                             |
| `last_error`     | TEXT                    | 最近一次失败原因                                                                                                                         |
| `event_payload`  | JSONB                   | MVP 不用（fan-out 直接读 PG），保留给将来 schema replay                                                                                       |


### `outbox_sink_cursors`


| 列                    | worker 用途            |
| -------------------- | -------------------- |
| `sink_name` PK       | MVP 写死 `'es_assets'` |
| `last_published_seq` | 单调推进的水位线             |
| `updated_at`         | 监控 stale cursor      |


---

## 9. 错误与重试策略


| 失败类型                                                    | 行为                                                                                                      | 上限 / 退避                                              |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| PG `FetchPending` 失败（连接抖动 / 死锁）                         | 指数退避：1s → 2s → 5s → 10s（封顶），不退出循环                                                                       | 持续 > 1 min 触发 P1（`outbox_fetch_deadlock_total`）      |
| 单条 doc 投影失败（PG 中查不到 asset 等异常）                          | log warn，**对应 events 保留 pending**，其它继续；下次 tick 重试                                                       | retry_count > 10 触发 P1 告警                            |
| ES BulkIndex 整批失败                                       | `retry_count++`，整批不标 published，下次 tick 重试                                                               | retry_count > 10 触发 P1 告警                            |
| **ES BulkIndex 部分失败**                                   | **失败 doc 对应 events 保留 pending**（详见 §5.1 步骤 5），成功 doc 对应 events 标 published；safe_horizon 自动停在最早的 pending | 同上 retry_count > 10                                  |
| Tombstone DELETE 失败                                     | 单独计数（`outbox_tombstone_failure_total`）+ 保留 pending；ES 文档仍存在不算业务错误（搜索结果过滤 `is_deleted=true` 仍可兜底）        | WARN                                                 |
| `MarkPublishedAndAdvanceCursor` 事务失败（典型：与 reindex 并发死锁） | 死锁 → 重试 3 次（间隔 200ms）；其他错误 → log + 保持 pending                                                           | `outbox_cursor_deadlock_total` rate > 0.05/s 触发 WARN |
| Worker panic                                            | 顶层 recover + 健康降级 + （可选）`os.Exit(1)` → K8s liveness 重启                                                  | —                                                    |


> MVP **不实现 DLQ 表**。`retry_count > 10` 的事件继续 pending，等待人工
> 介入。`outbox_oldest_pending_age_seconds` 是真正反映滞后的指标，
> 持续超阈值时由运维评估：通常等 worker 自然 drain 即可；极端长 gap
> （比如 ES 宕数天后恢复）下若觉得 pending 太多想加速，直接调一次
> `POST /admin/search/reindex` 全量重建即可，不需要复杂的"跳过状态机"。

---

## 10. 配置


| 环境变量                    | 含义                                                            | MVP 默认                  |
| ----------------------- | ------------------------------------------------------------- | ----------------------- |
| `OUTBOX_WORKER_ENABLED` | 是否启动 worker goroutine                                         | `false`（开发期手动开）         |
| `OUTBOX_BATCH_SIZE`     | 每批拉取事件上限。**默认 1000**——hot asset 场景下一次拉光，避免 drain 多轮重复投影同一 doc | `1000`                  |
| `OUTBOX_TICK_INTERVAL`  | 轮询间隔；端到端延迟上限 ≈ 此值 + drain time                                | `30s`                   |
| `OUTBOX_RETRY_LIMIT`    | 触发告警的 retry_count 阈值                                          | `10`                    |
| `OUTBOX_FATAL_ON_PANIC` | worker panic 时是否 `os.Exit(1)` 让 K8s 重启容器                      | `true`（生产）/ `false`（本地） |
| `ELASTICSEARCH_URL`     | ES 地址                                                         | 现有                      |
| `ELASTICSEARCH_INDEX`   | 索引名                                                           | `assets`                |


环境变量定义随同应用配置一起维护。

---

## 11. 可观测性

### 11.1 Metrics（Prometheus 风格命名，先写日志结构化输出）


| 指标                                  | 类型           | 含义                                                                         | 告警阈值                                  |
| ----------------------------------- | ------------ | -------------------------------------------------------------------------- | ------------------------------------- |
| `outbox_worker_batch_size`          | histogram    | 每轮拉取事件数                                                                    | —                                     |
| `outbox_worker_batch_duration_ms`   | histogram    | 单轮处理耗时                                                                     | P99 > 5 s 持续 5 min                    |
| `outbox_worker_pending_total`       | gauge（每分钟采样） | `count(*) WHERE publish_state='pending'`                                   | WARN > 10 000 / P1 > 100 000 持续 5 min |
| `outbox_oldest_pending_age_seconds` | gauge        | `now() - MIN(occurred_at) WHERE publish_state='pending'`，**这是真正反映"延迟"的指标** | WARN > 5 min / P1 > 1 h / P0 > 24 h   |
| `outbox_worker_retry_max`           | gauge        | `MAX(retry_count) WHERE publish_state='pending'`                           | WARN > 5 持续 10 min                    |
| `outbox_sink_lag_seq`               | gauge        | `MAX(event_seq) - last_published_seq`                                      | WARN > 10 000 持续 5 min                |
| `outbox_es_bulk_failures_total`     | counter      | ES bulk **整批**失败次数                                                         | rate > 0.1/s                          |
| `outbox_bulk_partial_failure_total` | counter      | ES bulk **部分**失败次数（≥ 1 doc 失败）                                             | rate > 0.5/s 持续 5 min                 |
| `outbox_tombstone_failure_total`    | counter      | DELETE doc 失败                                                              | WARN                                  |
| `outbox_fetch_deadlock_total`       | counter      | `FetchPending` 死锁次数                                                        | WARN rate > 0.05/s                    |
| `outbox_cursor_deadlock_total`      | counter      | `MarkPublishedAndAdvanceCursor` 死锁次数                                       | WARN rate > 0.05/s                    |


### 11.2 关键日志（INFO/WARN）

```
{"level":"info","msg":"outbox batch processed","events":127,"docs":48,"duration_ms":312,"sink":"es_assets"}
{"level":"warn","msg":"projection failed","asset_id":"...","err":"asset not found","action":"skipped"}
{"level":"warn","msg":"es bulk partial failure","failed_ids":["..."],"total":50,"sink":"es_assets"}
{"level":"error","msg":"es bulk total failure","err":"...","retry_count":3,"action":"retry_next_tick"}
```

---

## 12. 测试策略

### 12.1 单元测试


| 文件                          | 覆盖                                          |
| --------------------------- | ------------------------------------------- |
| `outbox/projection_test.go` | `BuildAssetDoc`：含 algo 多行聚合、tombstone、空字段处理 |
| `outbox/cursor_test.go`     | `safe_horizon` 计算：注入乱序 ack 验证 cursor 单调     |
| `outbox/worker_test.go`     | 主循环：mock repo + sink，验证去重 / batch 行为        |


### 12.2 集成测试（Postgres + ES 真实容器）


| 测试                               | 验收                                                |
| -------------------------------- | ------------------------------------------------- |
| `TestE2E_Notify_HappyPath`       | 写 1 个 event → < 2 s 内 ES 可查                       |
| `TestE2E_DedupBatch`             | 同 asset 写 100 events → ES 实际 BulkIndex doc 数 = 1  |
| `TestE2E_RestartReplay`          | 写 100 events → 杀 worker → 再写 100 → 重启 → ES 最终 200 |
| `TestE2E_ConcurrentAck_NoSeqGap` | 模拟乱序 publish 标记 → cursor 永不越过 MIN(pending) - 1    |
| `TestE2E_ESDown_Backpressure`    | ES 容器停掉 30 s → 期间事件累积在 pending → ES 恢复后全量到位       |


用 testcontainers-go 启 PG + ES 容器，build tag `//go:build integration` 与单元测试隔离。

---

## 13. 模块组成（逻辑视图）


| 模块         | 职责                                                                  | 估算代码量  |
| ---------- | ------------------------------------------------------------------- | ------ |
| Worker 主循环 | tick 调度 + drain loop + panic 兜底                                     | ~120 行 |
| Projector  | 从 `assets` + `asset_algo_latest` 构造 ES doc                          | ~80 行  |
| ESSink     | `BulkIndex` / `DeleteDoc` / 启动期 `EnsureMapping`                     | ~80 行  |
| OutboxRepo | `FetchPending` / `MarkPublishedAndAdvanceCursor` / `IncrementRetry` | ~120 行 |
| 配置         | 环境变量解析                                                              | ~30 行  |
| 单测 + 集成测试  | 同模块就近                                                               | —      |


启动接线：

```go
if cfg.OutboxWorkerEnabled {
    worker := NewWorker(pgClient, esClient, assetRepo, algoLatestRepo, cfg.Outbox)
    go func() {
        if err := worker.Run(ctx); err != nil {
            log.Error("outbox worker exited", err)
        }
    }()
}
```

---

## 14. Reindex API（同步交付，~50 行）

为 G4、mapping 漂移、长 gap 恢复三个场景兜底——**全量基于 `assets` 表重建 ES**。

```
POST /admin/search/reindex
Body: { "rebuild_index": false, "dry_run": false }
权限：独立 ADMIN_TOKEN，不与 X-Grace-Token 共享
```

执行步骤：

1. 可选 rebuild：`rebuild_index=true` 时先 `DELETE /assets` + `PUT /assets` with mapping
2. `SELECT asset_id FROM assets WHERE is_deleted=false ORDER BY asset_id` 分页扫
3. 每页调 `projection.Build(asset_id)` + `BulkIndex`，失败累计返回不中断
4. **不动 outbox cursor / publish_state**——这是"基于源表的全量重建"，与事件流并行

返回：

```json
{ "rebuild_index": true, "reindexed_assets": 10248, "failed_assets": ["uuid-..."], "duration_ms": 42510 }
```

> 设计要点：
>
> - reindex 期间 worker 继续正常消费 outbox，doc_id 幂等保证两路写入不冲突
> （后写者赢，最终态都来自同一个 `assets` 源，结果一致）
> - 长 gap 恢复时只需调一次 `reindex` 把 ES 拉回当前态，pending 中"过时事件"
> 被 worker 重新处理时也只是 BulkIndex 同样的最终态，**不会回退数据**
> - 不实现"从 `event_seq` 重放事件流"——保留给二期

---

## 15. 验收清单（合并入 PR 必过）

- `go build ./... && go vet ./... && go test ./...` 全过
- 集成测试 5 个 case 全过
- `OUTBOX_WORKER_ENABLED=false` 时 backend 行为与现状完全一致（回滚保险）
- `OUTBOX_WORKER_ENABLED=true` 启动后，对 5 个 seed asset 调一次 `POST /admin/search/reindex` → ES `_count` = 5
- 触发 `algo.start` → ≤ 60 s 内 ES doc 的 `algo_summary` 出现该 algo（一次 tick + drain 余量）
- 杀 worker / 杀 ES 容器 / 重启 backend 三种故障注入手动验证一遍
- 指标说明文档增加 outbox 相关指标条目
- 总体架构设计文档的"异步同步层"章节末尾链接回本文档

---

## 16. 风险与开放问题

### 16.1 为什么不用 LISTEN/NOTIFY（决策记录）


| 维度                            | 纯 polling（本设计） | LISTEN/NOTIFY                                    |
| ----------------------------- | -------------- | ------------------------------------------------ |
| 端到端延迟 P50 / P99               | ~15s / ~30s    | ~100ms / ~2s                                     |
| 延迟是否在 SLA 内（1 min）            | ✅              | ✅                                                |
| PG 端开销                        | 0              | trigger × 每事务 + `pg_notify` 调用                   |
| PG `pg_notify_queue_usage` 风险 | 无              | listener 卡住会灌满 SLRU 队列 → **trigger 报错 → 业务事务回滚** |
| 持久 listener 连接                | 无              | 占用 1 个 PG slot；网络抖断后需重连逻辑                        |
| 必须保留 tick 兜底                  | ✅（本身就是 tick）   | ✅（NOTIFY 不可靠）→ 等于两套机制                            |
| 代码复杂度                         | ~25 行          | ~50 行（含 listener 重连）                             |
| 故障域                           | 1（worker）      | 2（worker + listener）                             |


**结论**：1 分钟 SLA 下，NOTIFY 唯一收益是"trickle 场景首批省 30 秒"——burst 和 hot-asset 场景都被 drain time 吸收，没有可观测的差别。但它引入业务事务被 trigger 拖累的真实风险。**直接砍掉**。

### 16.2 风险登记


| 风险                                                     | 缓解                                                                                                                                                                                          |
| ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MIN(event_seq) WHERE pending` 在 `asset_events` 行数大时变慢 | `idx_asset_events_publish_pending` 部分索引（`WHERE publish_state='pending'`）已存在，EXPLAIN 验证后再上线                                                                                                  |
| ES Sink 部分失败                                           | 失败 doc 对应 events 保留 pending，safe_horizon 自动等待重试                                                                                                                                             |
| 长 gap 恢复（ES 宕数天）期间 pending 累积                          | 不影响 PG 主路径写入；恢复后 worker 自然 drain；若想加速，调一次 `POST /admin/search/reindex` 全量重建即可。指标 `outbox_oldest_pending_age_seconds` 是 SLO 主依据                                                              |
| **Hot asset**（同 asset 短时间被密集更新）                        | dedup by asset_id 已是天然防护：N 行 → 1 doc。`OUTBOX_BATCH_SIZE` 默认 **1000**（不是 200）专门为此场景——一次拉完 hot asset 所有 pending，drain 内只投影 1 次写 1 次。业务侧应在 API 层加 `Idempotency-Key` + 单 asset 限流（建议 10/s）从根上抑制 |
| Worker goroutine panic 不带崩主进程，但 panic 后不再消费            | 顶层 `recover` → `health.MarkUnhealthy` + 可选 `os.Exit(1)`；K8s liveness probe 检查 `/healthz/outbox`，自动重启容器                                                                                      |
| Mapping 漂移（部署脚本里的 mapping vs worker 内嵌 mapping）        | CI assertion 字节比对，diff 必失败                                                                                                                                                                  |
| 事件 payload schema 演进                                   | MVP 不读 payload；二期"从 event_seq 重放"开始才需要 `payload_schema_version` 兼容矩阵                                                                                                                        |


### 16.3 开放问题（评审时定）

1. **Reindex API 鉴权**：内网 IP allowlist vs 独立 admin token？倾向独立 token（`ADMIN_TOKEN` env），便于本地开发。
2. **Worker 启动是否阻塞 backend `/healthz`**：MVP 倾向"非阻塞"——worker 故障不影响 API。但需要单独的 `/healthz/outbox` 端点供 ops 检查。
3. **何时切 Debezium / Kafka**：触发条件登记在路线图，不是现在的事。触发条件至少满足两条：(a) 持续写入 ≥ 5k TPS；(b) sink 数量 ≥ 3。当前仅 1 个 sink，远未到。

---

## 17. 上线节奏


| 阶段      | 内容                                           | 退出条件              |
| ------- | -------------------------------------------- | ----------------- |
| Phase 0 | 本设计文档评审通过                                    | 评审 sign-off       |
| Phase 1 | 代码 + 单元测试 + 集成测试                             | PR 合入 main        |
| Phase 2 | 本地 docker compose 全链路验证                      | 手动 5 min smoke 全过 |
| Phase 3 | 默认 `OUTBOX_WORKER_ENABLED=false` 部署到 staging | 24h 无回退           |
| Phase 4 | staging 切 `true`，观察 24 h                     | metrics 全绿，告警 0   |
| Phase 5 | 生产灰度（10% → 100%）                             | 灰度期间无 P1          |


回滚方案：任何阶段出问题 → `OUTBOX_WORKER_ENABLED=false` 一键关闭，业务事务侧不受影响（事件继续累积在 PG，问题修复后开 worker 自然追上）。