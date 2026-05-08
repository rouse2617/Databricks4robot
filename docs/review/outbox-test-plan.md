# Outbox E2E / 压测执行单（历史方案）

> ⚠️ **历史方案说明**：本文档用于保留 Outbox Worker 路径的历史测试方法，**不是当前生产默认链路**。  
> 当前上线与值班执行请以 [`pure-cdc-go-live-runbook.md`](./pure-cdc-go-live-runbook.md) 和 [`cdc-rollout-plan.md`](./cdc-rollout-plan.md) 为准。
>
> 目标：沉淀 `asset_events -> outbox worker -> Elasticsearch` 历史链路的冒烟、恢复、压测与验收口径，供回溯或兼容场景参考。
>
> 范围：只覆盖历史 **ES sink** 测试路径，不覆盖当前 Pure CDC 主路径与未来 Iceberg sink 细节。

---

## 1. 验收目标

| 目标 | 含义 | 通过标准 | 主要手段 |
|---|---|---|---|
| G1 | PG 写入后，ES 最终可查 | 端到端延迟 `P99 <= 60s` | E2E 冒烟 + 压测 |
| G2 | Worker 重启不丢事件 | worker 停机期间写入的事件，重启后全部追平 | 集成测试 + 手工注入 |
| G3 | 同一 asset 多次更新被合并 | 同一批次内多次更新同一 asset，不放大 ES doc 写次数 | 集成测试 |
| G4 | ES 状态可重建 | `reindex` 之后 ES 文档数与 PG `assets` 一致 | 手工验证 |
| G5 | Cursor 推进不跳事件 | `last_published_seq` 单调推进，不越过 pending | 集成测试 |

---

## 2. 现有实现与可复用资产

### 2.1 代码内已有测试

| 文件 | 作用 |
|---|---|
| [backend/internal/outbox/e2e_test.go](/Users/rick/Databricks4robot/backend/internal/outbox/e2e_test.go) | Postgres + ES testcontainers 集成测试 |
| [backend/internal/outbox/worker_test.go](/Users/rick/Databricks4robot/backend/internal/outbox/worker_test.go) | worker 主循环、partial failure、panic/backoff |
| [backend/internal/outbox/chaos_test.go](/Users/rick/Databricks4robot/backend/internal/outbox/chaos_test.go) | outbox chaos / 异常路径 |

### 2.2 仓库内已有脚本

| 文件 | 作用 |
|---|---|
| [backend/scripts/test_outbox_e2e.sh](/Users/rick/Databricks4robot/backend/scripts/test_outbox_e2e.sh) | 本地端到端冒烟：创建 asset，等待 ES 可查 |
| [backend/scripts/bench/outbox_perf.sh](/Users/rick/Databricks4robot/backend/scripts/bench/outbox_perf.sh) | Outbox Worker 性能压测（50 eps × 60s） |
| [backend/scripts/verify_outbox_cutover.sh](/Users/rick/Databricks4robot/backend/scripts/verify_outbox_cutover.sh) | 校验 cutover / cursor / pending 状态 |

### 2.3 关键设计依据

- [docs/review/outbox-worker-design.md](/Users/rick/Databricks4robot/docs/review/outbox-worker-design.md)
- `G1–G5` 目标定义见该文档开头
- 指标阈值见该文档 `§11.1`

---

## 3. 环境准备

### 3.1 本地 docker compose

```bash
cd /Users/rick/Databricks4robot/deploy/local
docker compose -f docker-compose.all.yml up -d --build
```

### 3.2 必要环境

- backend 能访问 PostgreSQL
- backend 能访问 Elasticsearch
- `OUTBOX_WORKER_ENABLED=true`
- 测试 token 可用，当前默认走 `X-Grace-Token`

### 3.3 健康检查

```bash
curl -sf http://localhost:8080/healthz
curl -sf http://localhost:8080/healthz/outbox
curl -sf http://localhost:9200/_cluster/health
```

预期：

- `/healthz` = `200`
- `/healthz/outbox` = `200`
- `/_cluster/health` = `200`

### 3.4 建议的预清理

为了避免旧数据干扰压测和一致性对账，建议在正式执行前先做一次环境确认：

```bash
curl -s http://localhost:9200/assets/_count
curl -s http://localhost:8080/metrics | rg "^outbox_"
```

如需做“干净环境”压测，建议：

1. 使用独立 compose 环境或独立数据库
2. 为压测流量打固定前缀（例如 `bench-perf` / `outbox-e2e`）
3. 压测结束后用前缀做定向对账，不要依赖全库清空

### 3.5 建议记录的测试参数

每次执行前先写明：

| 参数 | 示例 |
|---|---|
| backend commit | `HEAD` / 某个 SHA |
| worker tick | `OUTBOX_WORKER_TICK_SEC=30` |
| batch size | `OUTBOX_WORKER_BATCH=100` 或 `1000` |
| retry limit | `OUTBOX_WORKER_RETRY_LIMIT=10` |
| ES endpoint | `http://localhost:9200` |
| PG endpoint | `localhost:5432/data4cyber` |
| 负载参数 | `50 eps × 60s` |

---

## 4. 推荐执行顺序

1. `go test -tags=integration` 跑代码级集成测试
2. `test_outbox_e2e.sh` 跑本地冒烟
3. `outbox_perf.sh` 跑吞吐/延迟基线
4. 手工注入 3 个故障场景
5. 汇总日志、指标、延迟结果，给出是否通过

---

## 5. 自动化测试用例

### 5.1 集成测试（代码级）

命令：

```bash
cd /Users/rick/Databricks4robot/backend
make test-integration
```

等价命令：

```bash
cd /Users/rick/Databricks4robot/backend
go test -tags=integration ./...
```

重点关注：

| 用例 | 目标 |
|---|---|
| `TestE2E_Notify_HappyPath` | G1 基础闭环 |
| `TestE2E_DedupBatch` | G3 dedup |
| `TestE2E_RestartReplay` | G2 重启恢复 |
| `TestE2E_ConcurrentAck_NoSeqGap` | G5 safe horizon / cursor |
| `TestE2E_ESDown_Backpressure` | ES 不可用时 backlog 行为 |

通过标准：

- 所有用例通过
- 无 flaky 失败

### 5.2 E2E 冒烟（脚本）

命令：

```bash
cd /Users/rick/Databricks4robot
bash backend/scripts/test_outbox_e2e.sh
```

它会做的事情：

1. 检查 backend / outbox / ES 健康
2. 创建一个新资产
3. 轮询 ES，确认该资产文档在 `<= 60s` 内可查
4. 校验 `owner / reviewer` 等关键字段已投影到 ES

通过标准：

- 创建资产返回 `201`
- ES 文档在 `<= 60s` 内出现
- 关键字段与 PG 写入值一致

失败时优先排查：

- `docker compose logs backend | grep outbox`
- `GET /healthz/outbox`
- `GET /metrics`

### 5.3 压测基线（脚本）

命令：

```bash
cd /Users/rick/Databricks4robot
bash backend/scripts/bench/outbox_perf.sh
```

默认参数：

- `EVENTS_PER_SEC=50`
- `DURATION_SEC=60`
- 总量约 `3000 events`
- 目标 `P99 <= 60s`

可选参数示例：

```bash
EVENTS_PER_SEC=100 DURATION_SEC=60 ES_TIMEOUT_SEC=180 \
bash backend/scripts/bench/outbox_perf.sh
```

通过标准：

- 创建成功率 > 99%
- ES 最终同步数 = 成功创建数
- `P99 <= 60s`
- worker 无 panic
- 无持续 bulk failure

建议记录：

- `P50 / P95 / P99`
- 总事件数
- 成功/失败数
- backlog 清空总耗时

### 5.4 Cutover 校验（脚本）

如果你们要做“进程内 worker -> 独立 worker”切换，切换后建议补跑：

```bash
cd /Users/rick/Databricks4robot
bash backend/scripts/verify_outbox_cutover.sh
```

它会检查：

1. `pending` 是否接近 0
2. PG 活跃资产数 vs ES 文档数是否匹配
3. `outbox_sink_cursors.last_published_seq` 是否追平 `MAX(event_seq)`
4. 是否存在高 `retry_count` 的卡死事件

通过标准：

- `pending = 0` 或接近 0
- `cursor` 与 `max_seq` 差距在容忍范围内
- 无高 retry stuck event

### 5.5 PG ↔ ES 一致性审计（脚本）

如果要作为 staging / nightly 的定期守门，再补跑：

```bash
cd /Users/rick/Databricks4robot
bash backend/scripts/es-pg-audit.sh
```

它会输出：

1. PG 活跃资产数
2. ES 文档数
3. PG 中存在但 ES 缺失的 asset_id
4. ES 中存在但 PG 缺失的孤儿文档
5. 一致率

建议通过标准：

- 一致率 `>= 99.9%`
- 缺失 / 孤儿文档都为 0，或在已知容忍窗口内

---

## 6. 手工故障注入用例

### 6.1 用例 A：Worker 重启不丢事件（G2）

步骤：

1. 启动全栈
2. 创建一批资产事件（建议 100～1000）
3. 停掉 backend / outbox worker
4. 在 worker 停机期间继续创建一批资产事件
5. 重启 backend / worker
6. 等待 backlog drain

断言：

- 所有资产最终都能在 ES 查到
- `outbox_worker_pending_total` 最终回到接近 0
- 无永久丢失

建议取证：

- worker 重启前后的日志
- `pending_total` 曲线
- 最终 ES `_count`

### 6.2 用例 B：ES 下线后恢复（G1 / backlog）

步骤：

1. 启动 backend + worker + PG
2. 暂停或停止 ES
3. 持续创建资产事件 1～5 分钟
4. 观察 backend API 写入仍成功
5. 恢复 ES
6. 等待 worker 自动追平

断言：

- PG 主路径不阻塞
- `outbox_oldest_pending_age_seconds` 上升后最终回落
- backlog 被完全消费

关键指标：

- `outbox_worker_pending_total`
- `outbox_oldest_pending_age_seconds`
- `outbox_es_bulk_failures_total`

### 6.3 用例 C：Reindex 恢复当前态（G4）

步骤：

1. 正常写入一批资产到 PG + ES
2. 人为删除 ES 索引中的部分或全部文档
3. 调用 reindex API

命令：

```bash
curl -sS -X POST "http://localhost:8080/api/v1/admin/search/reindex" \
  -H "X-Grace-Token: dev-token" \
  -H "Content-Type: application/json" \
  -d '{"dry_run": false, "page_size": 200}'
```

断言：

- 返回 `200`
- `failed == 0` 或失败项可解释
- ES `_count` 与 PG `assets` 一致
- reindex 不推进 outbox cursor，不改 `publish_state`

### 6.4 用例 D：ES Partial Bulk Failure（单 doc 失败）

目标：验证部分文档写失败时，成功项能 ack，失败项保留或进入重试/失败路径。

建议方式：

1. 使用测试环境中“容易触发 mapping 冲突”的 payload，或 mock ES bulk 返回单条失败
2. 发送一批事件，其中至少 1 个 asset 会失败
3. 观察 worker 行为

断言：

- 成功 doc 对应事件被 `published`
- 失败 doc 对应事件不会被错误推进 cursor 跳过
- `outbox_bulk_partial_failure_total` 增加
- `last_error` 可追溯

### 6.5 用例 E：高 retry / DLQ 行为

目标：验证反复失败事件不会无限静默卡死。

步骤：

1. 人为制造稳定失败的 ES 写入
2. 连续运行 worker，直到 `retry_count` 逐步增加
3. 若当前环境启用 DLQ sweep，观察其是否迁移到 `outbox_dlq`

断言：

- `retry_count` 递增
- `last_error` 被更新
- 启用 DLQ 时，超过阈值的事件进入 `outbox_dlq`
- 未启用 DLQ 时，至少要能从 SQL 明确查出 stuck 事件

建议核对：

```sql
SELECT COUNT(*) FROM outbox_dlq;
SELECT event_seq, retry_count, last_error
FROM asset_events
WHERE publish_state = 'pending'
ORDER BY retry_count DESC
LIMIT 20;
```

### 6.6 用例 F：Cutover 后无丢失

目标：验证从“进程内 worker”切到“独立 worker”不会丢数据。

步骤：

1. 先在进程内 worker 模式跑一批事件
2. 切换到独立 worker
3. 切换期间继续写事件
4. 跑 `verify_outbox_cutover.sh`

断言：

- `pending` 最终为 0 或接近 0
- ES 文档数与 PG 活跃资产数一致
- `last_published_seq` 接近 `MAX(event_seq)`

### 6.7 手工场景矩阵

| 场景 | 目标 | 必跑 | 建议频率 |
|---|---|---|---|
| Happy path | 基础闭环 | 是 | 每次改动 |
| Worker restart | G2 | 是 | 每次上线前 |
| ES down / recover | backlog 恢复 | 是 | 每次上线前 |
| Reindex restore | G4 | 是 | 每次上线前 |
| Partial bulk failure | 单 doc 失败拆分 | 建议 | 大改后 |
| DLQ / high retry | 失败治理 | 建议 | 大改后 |
| Cutover verify | 切换无丢失 | 条件性 | 切换时 |
| 100 eps / 200 eps 压测 | 容量边界 | 条件性 | 容量评估时 |

---

## 7. 观察项与阈值

### 7.1 最少要看的指标

| 指标 | 期望 |
|---|---|
| `outbox_worker_pending_total` | 压测后能回落到接近 0 |
| `outbox_oldest_pending_age_seconds` | 不持续单调上升；恢复后回落 |
| `outbox_worker_batch_duration_ms` | P99 不长期高于 5s |
| `outbox_es_bulk_failures_total` | 瞬时可接受，持续增长不可接受 |
| `outbox_bulk_partial_failure_total` | 低且可解释 |
| `outbox_sink_lag_seq` | backlog 消费后回落 |

### 7.2 最少要看的日志关键词

```text
outbox worker batch failed
outbox worker panic
es bulk partial failure
moved events to DLQ
```

### 7.3 建议同时记录的 PostgreSQL 视图

测试期间建议每 1~5 分钟采样一次：

```sql
SELECT COUNT(*) FROM asset_events WHERE publish_state = 'pending';
SELECT COALESCE(MAX(retry_count), 0) FROM asset_events WHERE publish_state = 'pending';
SELECT COALESCE(MAX(event_seq), 0) FROM asset_events;
SELECT COALESCE(last_published_seq, 0) FROM outbox_sink_cursors WHERE sink_name = 'es_assets';
```

这 4 个数字足够判断：

- backlog 有没有积压
- retry 有没有异常升高
- cursor 有没有追上
- lag 是不是持续扩大

### 7.4 建议同时记录的 Elasticsearch 视图

```bash
curl -s http://localhost:9200/assets/_count
curl -s http://localhost:9200/_cluster/health
curl -s http://localhost:9200/assets/_stats/indexing
```

### 7.5 失败分级建议

| 现象 | 级别 | 建议动作 |
|---|---|---|
| 单次脚本失败，可重跑恢复 | P3 | 记录原因，重跑一次 |
| E2E 冒烟 > 60s 但最终可达 | P2 | 查 worker tick / ES 性能 |
| 压测 `P99 > 60s` | P1 | 阻断上线，查 batch / backlog / ES |
| worker panic | P1 | 阻断上线，先修复 |
| ES 恢复后 backlog 无法清空 | P1 | 阻断上线 |
| reindex 后 ES 与 PG 长时间不一致 | P1 | 阻断上线 |
| cursor 跳跃 / 越过 pending | P0 | 立刻停止放量 |

---

## 8. 通过 / 不通过判定

### 8.1 通过

以下全部满足才算通过：

- `make test-integration` 通过
- 冒烟脚本通过
- 压测脚本通过
- `P99 <= 60s`
- worker 重启不丢事件
- ES 故障期间 PG 主路径不阻塞
- reindex 能把 ES 拉回当前态

### 8.2 不通过

任意一条成立即判定不通过：

- 集成测试有 1 个关键用例失败
- E2E 冒烟 60s 内无法在 ES 查到
- 压测 `P99 > 60s`
- backlog 无法在合理时间内清空
- 出现 worker panic 且无法自恢复
- reindex 后 ES 文档数与 PG 不一致

---

## 9. 测试结果记录模板

建议每次测试把结果按下面格式填一份：

```md
# Outbox Test Report

Date:
Env:
Commit:

## Integration
- Command:
- Result:

## E2E Smoke
- Command:
- Asset created:
- ES visible latency:
- Result:

## Perf
- Events/s:
- Duration:
- Total events:
- Success:
- Failure:
- P50:
- P95:
- P99:
- Result:

## Fault Injection
- Worker restart:
- ES down / recover:
- Reindex restore:
- Partial bulk failure:
- DLQ / high retry:
- Cutover verify:

## Metrics Snapshot
- pending_total:
- oldest_pending_age_seconds:
- es_bulk_failures_total:
- bulk_partial_failure_total:
- sink_lag_seq:
- retry_max:
- cursor:
- max_event_seq:

## Final Verdict
- Pass / Fail:
- Notes:
```

建议附带 3 份证据：

1. `test_outbox_e2e.sh` 输出
2. `outbox_perf.sh` 输出
3. 一次 SQL / metrics 快照

### 9.1 Nightly / CI 建议

如果要把这份执行单进一步落进自动化，建议最低配是：

#### Nightly

- 跑 `backend/scripts/es-pg-audit.sh`
- 跑 `backend/scripts/test_outbox_e2e.sh`
- 归档日志和一致率报告

#### PR / Merge 阶段

- 跑 `make test-integration`
- 不默认跑长压测

#### 周期性容量检查

- 每周或每个里程碑跑一次 `outbox_perf.sh`
- 每次参数固定保留一份基线报告

### 9.2 常见失败与定位顺序

#### 9.2.1 冒烟查不到 ES 文档

排查顺序：

1. `/healthz/outbox`
2. `docker compose logs backend | grep outbox`
3. `SELECT COUNT(*) FROM asset_events WHERE publish_state='pending'`
4. `curl /metrics | rg outbox_`
5. `curl http://localhost:9200/_cluster/health`

#### 9.2.2 压测 backlog 不下降

优先看：

1. `outbox_worker_pending_total`
2. `outbox_oldest_pending_age_seconds`
3. `outbox_sink_lag_seq`
4. `outbox_worker_batch_duration_ms`
5. ES `_cluster/health` 与 `_stats/indexing`

#### 9.2.3 Reindex 后仍不一致

优先看：

1. `es-pg-audit.sh`
2. 是否存在 `failed > 0`
3. ES 是否有孤儿文档
4. PG 是否存在软删除 / tombstone 逻辑差异
5. 最近是否并发跑着 worker drain

---

## 10. 建议的最小上线闸口

如果你现在只想要一个最小上线口径，我建议至少执行这 4 条：

1. `make test-integration`
2. `bash backend/scripts/test_outbox_e2e.sh`
3. `bash backend/scripts/bench/outbox_perf.sh`
4. 手工跑一次 `Worker 重启不丢事件` + 一次 `ES 下线恢复`

这 4 条都过，再讨论放量。
