# ADR-007:写入路径 & 事件契约(Write Path & Event Contract)

- **Status**: Accepted(Phase 0 冻结版)
- **Date**: 2026-04-22
- **Supersedes**: 校正先前文档中关于 MCE/MCL / BFF / URN 的口径分歧
- **Related**: [ADR-001](ADR-001-wide-table-with-bigtable.md) / [ADR-006](ADR-006-urn-identity.md) / [ADR-009](ADR-009-phase1-migration-plan.md) / [02-architecture.md](../02-architecture.md) / [06-borrowed-patterns.md](../06-borrowed-patterns.md)

---

## Context

先前文档中关于"谁写库、谁发事件、浏览器是否直连后端服务、ID 是 URN 还是 UUID"的描述出现**口径分歧**,典型如:

- 06-borrowed-patterns §模式 3:SDK/UI → **MCE** → asset-service → **MCL**
- 04-mcap §4.2、02-architecture §4.1:服务直接写库 → **MCL**(没 MCE)
- 06-borrowed-patterns §模式 4 Sink:"写 Bigtable + 发 MCE"(方向错乱)
- 05-ui-strategy §4.2:浏览器 → **BFF** → 服务
- 02-architecture §4.2:Web UI → **gRPC** → asset-service(跳 BFF)
- ADR-006:API id 全 URN;其他文档:接口主键 `asset_id` 是 UUID

本 ADR 一次性**定死契约**,所有其他文档以本 ADR 为准。

---

## Decision

### 1. 写入路径:**唯一权威写入者 = asset-service**

- **Phase 0 简化(强制)**:
  - **同步写**:调用方(SDK / BFF / worker) → `asset-service` gRPC →  **同步 commit PG**
  - **异步广播**:commit 成功后,asset-service 通过 **Transactional Outbox** 发 **MCL** 到 Pub/Sub
  - **不走 MCE**。所有意图直接作为同步 RPC 到 asset-service
- **Phase 1 扩展(可选)**:
  - 引入 **MCE topic(`grace-mce`)** 作为**异步/批量写入**的入口(批量 ingest、外部系统集成)
  - MCE → asset-service consumer → 落库 → MCL
  - 点对点 gRPC 继续保留(低延迟、强一致的 UI / SDK 写)

### 2. 幂等 & 重试契约

- 每次写请求带 **`request_id`**(client 生成 UUIDv4),header: `x-request-id`
- asset-service 内部 `idempotency_keys` 表(PG)TTL 24h
- 重试相同 `request_id` → 返回首次结果,不重复落库、**不重复发 MCL**
- 失败补偿:Outbox + 后台 relay 重发,保证 **at-least-once MCL**

### 3. 事件定义

#### MCL(Metadata Change Log)—— 权威事实

```json
{
  "event_id": "uuid",                               // Pub/Sub messageId 之外的应用级 ID
  "subject":  "urn:grace:asset:<uuid>",             // 必 URN
  "event_type": "asset.qa.approved",                // 见 fields.yaml event_types
  "actor":    "urn:grace:user:alice@company.com",
  "occurred_at": "2026-04-22T11:00:00Z",
  "request_id": "uuid",                             // 原 client request_id
  "prev_version": 1,
  "new_version":  2,
  "changed_aspects": ["qa", "event"],
  "payload":  { /* 变更后的关键字段快照 */ },
  "schema_version": 1
}
```

- topic: `grace-mcl`
- 分区键(ordering key):`subject`(保证同一 asset 顺序)
- 订阅者:OpenSearch indexer / BigQuery sink / Dagster sensor / subscription-service

#### MCE(Metadata Change Event)—— Phase 1 启用

```json
{
  "event_id": "uuid",
  "subject":  "urn:grace:asset:<uuid>",
  "intent":   "qa.approve",                         // 意图动词
  "actor":    "urn:grace:user:alice@company.com",
  "proposed": { /* 希望写入的字段 */ },
  "if_match_version": 1,                            // 乐观锁前件
  "request_id": "uuid",
  "schema_version": 1
}
```

- topic: `grace-mce`(**Phase 1 启用**)
- 处理方:asset-service MCE consumer → 验证 → 落库 → 发 MCL

### 4. 多表写原子性(Phase 1 Bigtable 起)

> Phase 0 是 PG 单表 + 单事务 + Outbox,无此问题。Phase 1 起,主表 + `asset_locator` + `assets_by_project` 是**三张 Bigtable 表**,Bigtable 无跨行事务,必须在 asset-service 里显式保证一致性。

**写入顺序(强制)**:

```
1. 写主表 row(v0/v1 key)— CheckAndMutate 保证乐观锁
2. 写 asset_locator(asset_uuid → main_row_key)— 条件写,仅当不存在时才建
3. 写 assets_by_project(by_project row key → main_row_key)
4. Outbox commit(主表同一次 mutation 里写 outbox 行),relay 发 MCL
```

**失败补偿**:

| 失败点 | 结果 | 补偿 |
| --- | --- | --- |
| 第 1 步失败 | 全失败,对 client 返错 | client 按 `request_id` 重试 |
| 第 1 步成功 / 第 2 步失败 | **主表有行,locator 缺** | 后台扫描器(janitor)每 10s 扫最近 N 分钟主表,补 locator |
| 第 1/2 步成功 / 第 3 步失败 | by_project 索引缺 | 同上 janitor 补 |
| 第 1 步成功 / Outbox commit 失败 | **不会发生**(outbox 和主表同一行 / 同一次 mutation) | — |
| Outbox 发 MCL 失败 | MCL 未广播 | relay 无限重试(at-least-once) |

**实现要点**:

- `asset_locator` 的写**必须幂等**(幂等键 = `asset_uuid`,已存在则 skip)
- janitor 读主表用 `cf:core.created_at` 做窗口,避免全表扫
- `request_id` 幂等表需要**同时**保护主表 + 两张索引表的写,任一路径重复都视为同一请求

### 5. API 入口边界

| 调用方 | 入口 | 协议 | ID 形态 |
| --- | --- | --- | --- |
| **浏览器 (Web UI)** | **BFF**(唯一) | HTTPS / GraphQL | **URN** |
| **算法用户 (SDK)** | asset-service(直连)/ mcap-gateway(直连) | gRPC(TLS) | **URN**(SDK 外部接口)/ UUID(内部 wire) |
| **CLI / Ops 工具** | asset-service(直连) | gRPC | URN(外部 API)/ UUID(内部) |
| **Dagster / Ray worker** | asset-service(直连) | gRPC | UUID(内部可信) |
| **外部系统 / 批量 ingest**(Phase 1) | Pub/Sub `grace-mce` | JSON | URN |

**规则**:
- **浏览器不直连** asset-service。所有浏览器流量走 BFF,BFF 做鉴权、聚合、URN↔UUID 转换
- **SDK / CLI / worker 直连** asset-service,简化链路
- **对外 API 响应 id 字段 = URN**;gRPC wire 层面字段可以是 UUID(性能),URN 在 gateway / SDK 层包装

### 6. URN ↔ UUID 转换边界

```
  Browser                BFF               asset-service                      Bigtable/PG
  ─────────              ────              ─────────────                      ───────────
  URN(URL/body)  ────▶   URN ──▶ UUID      UUID                      ────▶    UUID 是 row key 的一个字段
                          │                 │                                  (Phase 1 起,UUID 点查先经
                          │                 │                                   asset_locator 解析到 main_row_key)
                          │                 └──▶ MCL subject = URN(重新包装)
                          │
                          └──▶ 响应体 id = URN(重新包装)

  SDK
  ───
  Urn 对象 ──▶ SDK 内部 .uuid 属性 ──▶ gRPC wire UUID ──▶ asset-service(同上)
```

- **URN 解析**:`urn:grace:<type>:<key>` → `(type, key)`,type 受 `fields.yaml` 约束
- **内部 UUID**:asset_id 的 `<key>` 部分就是 UUIDv4,两者一一对应
- **UUID → 物理行**:Phase 0 = PG 表主键直查;Phase 1 = asset_locator 一跳 + 主表 GetRow(详见本 ADR §4)
- **审计 / 日志 / 事件 subject**:**一律用 URN**(跨系统可读)

### 7. Phase 推进

| Phase | 写路径 | 事件 | 浏览器入口 |
| --- | --- | --- | --- |
| **Phase 0** | sync gRPC → PG → Outbox → MCL | **只 MCL** | BFF(简化:直接 REST JSON) |
| **Phase 1** | sync gRPC / async MCE → Bigtable → MCL | MCE + MCL | BFF(GraphQL) |
| **Phase 2+** | 同 Phase 1 + 多源写入 | 同 Phase 1 | 同 Phase 1 |

---

## Alternatives Considered

| 方案 | 为什么不选 |
| --- | --- |
| Phase 0 就上 MCE 全流程 | 异步写在低流量 Phase 0 意义不大,反而增加 debug 难度 |
| 浏览器直连 gRPC(with grpc-web) | 鉴权 / CSRF / 聚合都得在客户端做,不适合核心资产平台 |
| API 响应字段内部用 UUID,外部调用方自己拼 URN | 易错,各家 client 拼法不一;统一在 server 层包装更鲁棒 |
| 全链路都用 URN(包括 Bigtable row key) | URN 冗余字节消耗存储 + 写入开销;内部系统不需要 |

---

## Consequences

### Positive
- ✅ 口径统一,后续所有代码、文档、合同按本 ADR 走
- ✅ Phase 0 路径简单:**一次同步写 + 一次 outbox relay**,易测易 debug
- ✅ URN/UUID 责任边界清晰,无人两头踩空
- ✅ 预留 Phase 1 的 MCE 异步入口,不会推翻 Phase 0 设计

### Negative
- ⚠️ asset-service 成为写入瓶颈(但 Phase 0 单实例即可应付,Phase 1 起水平扩展)
- ⚠️ Outbox relay 需要一点运维(但是通用组件,~ 200 行 Go 可搞定)

### Mitigations
- 写热点用 Pub/Sub + worker 预聚合(Phase 1)
- Outbox 用标准表 + cron job,不引入新基础设施

---

## Acceptance Criteria

- [ ] 所有文档中的"写路径图"都是 *caller → asset-service → PG/Bigtable → Outbox → MCL*
- [ ] MCL 事件 schema 在 `schemas/events/mcl.v1.json` 固化
- [ ] BFF 提供 URN↔UUID 转换中间件,浏览器拿到的所有 id 都是 URN
- [ ] SDK 提供 `Urn` 类,封装转换
- [ ] 幂等键实现 + 单测覆盖
- [ ] Outbox + MCL relay 工作的 E2E 测试

---

## 强制修正的文档

本 ADR 生效后,以下文档**以本 ADR 为准**(已同步修订):

| 文档 | 修订项 |
| --- | --- |
| `docs/02-architecture.md` §4.2 | Web UI → BFF → asset-service(删去"gRPC") |
| `docs/04-mcap-and-segment.md` §4.2、§4.3 | 写库后发 MCL 的描述保持(Phase 0 语义) |
| `docs/06-borrowed-patterns.md` §模式 3 | 改为 Phase 0 = 仅 MCL;Phase 1 = MCE+MCL |
| `docs/06-borrowed-patterns.md` §模式 4 Sink | 改"写 Bigtable + 发 MCE" → "写 Bigtable + 发 MCL" |
| `docs/05-ui-strategy.md` §4.2 | 维持 BFF,与本 ADR 一致 |
| `docs/adr/ADR-006-urn-identity.md` | 增加"外部 URN / 内部 UUID"说明 |
| `README.md` TL;DR | "9 CF"口径,并指向 ADR-007 |
| `diagrams/architecture.mmd` | 改目标态(9 CF + MCE/MCL)+ 新增 `phase0-runtime.mmd` |

---

## References

- Transactional Outbox: https://microservices.io/patterns/data/transactional-outbox.html
- GCP Pub/Sub ordering keys: https://cloud.google.com/pubsub/docs/ordering
- DataHub MCE/MCL 原始设计: https://datahubproject.io/docs/architecture/metadata-serving
