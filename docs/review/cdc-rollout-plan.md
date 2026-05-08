# CDC / WAL Rollout Plan

> 目标：把当前项目从“混合同步链路 + 本地 Iceberg 脚手架”演进到
> “**Iceberg 走 `asset_events` CDC INSERT 流，ES 走 current-state tables CDC 流**”
> 的长期架构，并给出**从现在到上线**的完整执行顺序、测试清单、回滚方案。
>
> 本文档假设：
>
> - **PostgreSQL schema 不能随便改**
> - `asset_events` 继续保留
> - ES 当前线上路径暂不立即下线
> - 先做本地 / staging shadow run，再做生产切换

---

## 1. Final Target

### 1.1 Long-term architecture

- **Iceberg Bronze**
  - source: PostgreSQL WAL / CDC
  - table: `asset_events`
  - only consume: `INSERT`
  - semantic: immutable business event history

- **Elasticsearch**
  - source: PostgreSQL WAL / CDC
  - tables:
    - `assets`
    - `asset_tags`
    - `asset_algo_latest`
    - `mcap_files`
  - semantic: current-state search document rebuild

### 1.2 Explicitly not the target

- ES 从 `asset_events` 回放拼 current-state
- Bronze 继续依赖应用层 SQL 轮询
- 用 `event_seq > cursor + safety lag` 作为长期协议
- 第一步就强推新的 PostgreSQL projection table

---

## 2. Guiding Constraints

### 2.1 Hard constraints

1. PostgreSQL schema 不随意变更
2. 当前 ES 检索能力不能先被打断
3. Bronze 必须保留业务事件语义，适合 replay / audit / training
4. 长期正确性必须来自 WAL / CDC commit order，不来自应用层猜测

### 2.2 Design consequences

- `asset_events` 持续作为事件主线，消费完全由 CDC 路径承担
- Bronze CDC 只吃 `asset_events` 的 `INSERT`
- ES CDC 不依赖 `asset_events`
- ES current-state rebuild 复用现有 `searchindex.Builder`

---

## 3. Scope Split

### 3.1 Iceberg workstream

要交付：

- Debezium/Postgres CDC -> `asset_events`
- Kafka-compatible topic
- Bronze consumer
- staging files
- `bronze_merge.py`
- Bronze verification

### 3.2 Elasticsearch workstream

要交付：

- Debezium/Postgres CDC -> current-state tables
- Kafka-compatible topics
- ES consumer
- affected asset ID resolver
- document rebuild via `searchindex.Builder`
- ES shadow validation

### 3.3 Out of scope for first rollout

- Silver / Gold modeling
- frontend lakehouse UI
- replacing all legacy sync code immediately
- removing legacy sink bookkeeping columns from `asset_events`

---

## 4. Current Status Snapshot

### 4.1 Already available in the long-term branch

Documented / coded already:

- [asset-events-bronze-design.md](./asset-events-bronze-design.md)
- `internal/cdc` package skeleton
- Debezium-style message decoder
- in-memory source adapter
- Kafka source skeleton + kafka-go poller
- Bronze CDC consumer skeleton
- ES current-state CDC consumer skeleton
- local CDC compose scaffold
- connector registration templates
- BronzeSink contract tests

### 4.2 Still missing

- real Debezium/Kafka message flow into app runtime
- robust Kafka consumer lifecycle / batching / offset behavior
- staging -> Bronze operational verification
- current-state CDC shadow validation against baseline ES index output
- production rollout runbook

---

## 5. Component Choices

### 5.1 Recommended local stack

- PostgreSQL with `wal_level=logical`
- Redpanda as Kafka-compatible bus
- Debezium Connect
- existing MinIO / Iceberg REST / Trino stack

### 5.2 Recommended production stack

- PostgreSQL logical decoding
- Debezium PostgreSQL Connector
- Kafka-compatible durable bus
- Iceberg sink consumer (transitional file-based Bronze write is acceptable)
- custom Go ES current-state consumer

### 5.3 Why not generic ES sink

Because ES documents depend on multiple current-state tables and must be rebuilt as one final document.

Use:

- CDC to detect which asset changed
- Go consumer to rebuild the document
- existing `searchindex.Builder` as the current-state projector

---

## 6. Rollout Phases

## Phase 0: Freeze the target semantics

### Objective

Agree on the two contracts before touching prod runtime:

- Bronze = `asset_events` insert-only business event history
- ES = current-state rebuild from current-state table CDC

### Deliverables

- this rollout plan
- [asset-events-bronze-design.md](./asset-events-bronze-design.md)
- final connector topic/table mapping

### Exit criteria

- no unresolved architectural ambiguity about source semantics

---

## Phase 1: Local CDC stack bring-up

### Objective

Make the long-term branch runnable locally with CDC infrastructure present.

### Tasks

1. Bring up:
   - PostgreSQL
   - Redpanda
   - Debezium Connect
2. Register two connectors:
   - `asset_events`
   - current-state tables
3. Confirm connector health over Connect REST API
4. Produce at least one CDC message per topic

### Artifacts

- `deploy/local/docker-compose.yml`（Compose profiles：`full`、`lakehouse`、`cdc`）
- `deploy/local/cdc/connectors/*.json`
- `deploy/local/cdc/register-connectors.sh`

### Verification

- `curl http://localhost:8084/connectors`
- topics exist
- Connect reports running tasks

### Exit criteria

- local Debezium emits CDC messages from PG

---

## Phase 2: Bronze CDC path (local)

### Objective

Run the full `asset_events INSERT CDC -> staging -> bronze_merge.py -> bronze_asset_events` path locally.

### Tasks

1. Wire the real Kafka source into the app-side CDC runtime
2. Route `asset_events` topic to Bronze consumer
3. Ensure Bronze consumer ignores `UPDATE` messages
4. Materialize staging JSONL with `BronzeSink`
5. Run `bronze_merge.py`
6. Query `bronze_asset_events` through Trino

### Implementation notes

- First production-grade checkpoint remains Kafka/consumer offset
- `event_seq` remains Bronze dedup key
- staging file mode is acceptable as the transitional write mode

### Verification

1. Create asset / update tags / finish algo through API
2. Confirm corresponding `asset_events` INSERT messages appear
3. Confirm staging files are written
4. Confirm `bronze_merge.py` appends rows
5. Confirm no duplicate `event_seq` values in Bronze

### Exit criteria

- local Bronze path is fully functional and replayable

---

## Phase 3: ES current-state CDC path (local)

### Objective

Run the current-state CDC flow into a shadow ES consumer locally.

### Tasks

1. Route current-state table topics into the ES consumer
2. For each CDC event, derive affected `asset_id`
3. Rebuild the final ES doc with `searchindex.Builder`
4. Upsert/delete the ES document

### Special handling

- `assets` -> direct rebuild
- `asset_tags` -> direct rebuild
- `asset_algo_latest` -> direct rebuild
- `mcap_files` -> fan out by `mcap_file_id`

### Verification

1. Mutate one row in each source table
2. Confirm the correct `asset_id` is rebuilt
3. Compare final ES doc with `searchindex.Builder` output

### Exit criteria

- local ES CDC consumer can rebuild correct docs from CDC topics

---

## Phase 4: Staging shadow run

### Objective

Deploy the CDC stack to staging without cutting over search traffic.

### Tasks

1. Run Debezium + bus + CDC consumers in staging
2. Keep CDC ES consumer as the active ES sync path
3. Let Bronze and CDC ES consumer run in shadow mode
4. Collect metrics and compare outputs

### Required comparisons

- PG vs Bronze event count / event_seq continuity
- baseline ES output vs CDC ES consumer output
- document spot checks on changed assets

### Exit criteria

- no silent Bronze loss
- no systematic ES doc divergence

---

## Phase 5: Bronze production enablement

### Objective

Enable `asset_events` CDC -> Bronze in production, while leaving ES on the old path.

### Tasks

1. Deploy CDC stack
2. Start `asset_events` connector
3. Start Bronze consumer
4. Run merge cron
5. Run no-loss verification and Bronze audits

### Verification

- Bronze row growth
- no-loss check
- replay test on a small interval
- Trino queries return real data

### Exit criteria

- Bronze is trusted as business event history

---

## Phase 6: ES shadow mode in production

### Objective

Run CDC-based ES current-state sync with shadow validation.

### Tasks

1. Start current-state connectors
2. Start ES CDC consumer
3. Keep current CDC ES consumer active
4. Compare outputs continuously

### What to compare

- ES doc count
- selected document fields
- deletion / soft deletion behavior
- lag after updates

### Exit criteria

- CDC ES output is equivalent to or better than baseline ES output

---

## Phase 7: ES cutover

### Objective

Switch Elasticsearch primary sync fully to CDC-based current-state sync.

### Tasks

1. Freeze cutover window
2. Run final PG↔ES audit
3. Verify no legacy app-level sync switch remains enabled
4. Keep reindex available
5. Watch metrics and logs closely

### Exit criteria

- ES remains correct after cutover
- no unacceptable lag
- no missing changed assets

---

## Phase 8: Cleanup and hardening

### Objective

Reduce legacy coupling after the CDC paths are stable.

### Candidate tasks

1. stop treating `asset_events.publish_state` as long-term business event semantics
2. decide whether sink-local state should be fully moved out of `asset_events`
3. harden Kafka offset / retry / restart behavior
4. add stronger no-loss automated checks

### Not required immediately

- removing old columns in the first post-cutover release

---

## 7. Runtime workstreams

### 7.1 Backend application code

Must deliver:

- `internal/cdc` runtime
- Debezium decoder
- real Kafka-compatible source
- Bronze consumer
- ES current-state consumer
- standalone `cdc-demo`

### 7.2 CDC infrastructure

Must deliver:

- logical decoding enabled PostgreSQL
- Debezium Connect
- connector registration
- Kafka-compatible bus

### 7.3 Iceberg side

Must deliver:

- staging file durability
- merge cron
- Trino readability
- no-loss verification

### 7.4 Search side

Must deliver:

- current-state table topic wiring
- affected asset fanout
- final document rebuild
- ES shadow validation

---

## 8. Testing matrix

### 8.1 Local tests

- `go test ./internal/cdc`
- `go test ./cmd/cdc-demo`
- local connector registration
- local topic smoke
- local Bronze merge smoke

### 8.2 Staging tests

- create / update / tag / algo flows
- Bronze event continuity
- ES doc rebuild correctness
- replay test
- soft delete test

### 8.3 Production gates

- Bronze no-loss verification
- PG↔ES audit
- shadow doc comparison
- connector stability

---

## 9. Rollback plan

### 9.1 Bronze path rollback

If Bronze CDC misbehaves:

- stop Bronze consumer
- stop merge cron
- keep `asset_events` source intact
- fix and replay from Kafka/CDC offset or replay interval

ES remains unaffected.

### 9.2 ES CDC rollback

If ES CDC consumer misbehaves:

- stop ES CDC consumer
- keep current ES index and reindex path active
- reindex from current-state tables if needed

Bronze remains unaffected.

### 9.3 Full rollback

Worst case:

- stop CDC consumers
- keep PG as source of truth
- keep CDC offsets and reindex capabilities available
- preserve CDC offsets and Bronze data for later resume

---

## 10. Observability

Minimum metrics to add before cutover:

### Bronze

- CDC consumer lag
- staging files written
- merge lag
- Bronze max `event_seq`
- failed decode count

### ES

- current-state CDC consumer lag
- affected asset rebuild count
- ES bulk failures
- doc delete count
- doc rebuild latency

### Infrastructure

- Debezium connector task status
- Redpanda / Kafka health
- Connect REST health

---

## 11. What must not be forgotten

1. Bronze only consumes `asset_events` INSERT
2. ES does not consume `asset_events`
3. `event_seq` is a business dedup key, not a transport checkpoint
4. WAL/CDC offset is the transport checkpoint
5. current-state ES rebuild is allowed to be application-specific
6. PostgreSQL schema is preserved during the initial rollout

---

## 12. Immediate next coding steps

1. finish the Kafka-compatible source adapter beyond skeleton quality
2. wire local Redpanda/Connect topics into the runtime
3. exercise Bronze path end-to-end locally
4. exercise ES current-state rebuild end-to-end locally
5. only then discuss staging deployment
