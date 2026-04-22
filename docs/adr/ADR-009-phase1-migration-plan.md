# ADR-009:Phase 0 → Phase 1 迁移计划(PG → Bigtable + 二级索引)

- **Status**: Proposed
- **Date**: 2026-04-22
- **Deciders**: 架构组 / SRE / Asset-service owner
- **Related**: [ADR-001](ADR-001-wide-table-with-bigtable.md) / [ADR-007](ADR-007-write-path-and-event-contract.md) / [03-data-model-wide-table.md](../03-data-model-wide-table.md) / [08-roadmap.md](../08-roadmap.md)

---

## Context

Phase 0 用 Cloud SQL Postgres 单表 + JSONB 模拟 Bigtable 9 CF 宽表。
Phase 1 要切到真 Bigtable,并引入两张二级索引表(见 [03-data-model §2.4](../03-data-model-wide-table.md)):

- `assets`(主表,v1 row key)
- `asset_locator`(`asset_uuid → main_row_key`)
- `assets_by_project`(按 project 扫最近)

迁移涉及三个难点,如果没有明确契约会翻车:

1. **多表写原子性**(见 [ADR-007 §4](ADR-007-write-path-and-event-contract.md)):Bigtable 无跨行事务,三张表的写入必须有补偿通路
2. **回填时机**:新建 Bigtable 主表时**索引副本是空的**,直接切读会导致 `asset_locator` 点查穿透到主表扫描(百亿级不可接受)
3. **验收口径**:ADR-001 新加的 "locator 无孤儿率 < 1/百万" 指标,**从哪一刻开始生效**?切读前还是切读后?

本 ADR 锁定 **5 阶段迁移路径 + 回滚点 + 指标生效时点**,SRE 可以直接按本 ADR 编写 runbook。

---

## Decision

### 1. 5 阶段迁移流程

```
阶段 0        阶段 1        阶段 2        阶段 3        阶段 4        阶段 5
-------       -------       -------       -------       -------       -------
准备        双写开启      索引回填      影子读校验    切读           PG 下线
PG 权威      PG 权威       PG 权威       PG 权威       BT 权威        BT 权威
(BT 建好)   (BT 异步)    (索引追平)   (读两侧对比)  (PG 仅双写)   (PG 归档)
```

| 阶段 | 权威读源 | 权威写源 | 状态 |
| --- | --- | --- | --- |
| 0 准备 | PG | PG(单写) | Bigtable 空 |
| 1 双写 | PG | PG + BT 主表(**异步、best-effort**) | BT 主表追新写入;索引表空 |
| 2 回填 | PG | PG + BT 主表(异步)+ BT 索引(异步) | 启动 backfill 把存量写入 BT 主表 + 两张索引 |
| 3 影子读 | PG(返回给 client) | 双写 | `asset-service` 并行读 BT,diff 落审计表 |
| 4 切读 | **BT**(经 `asset_locator`) | **BT 权威** + PG(兼容双写,不读) | **SLO 指标从此刻生效** |
| 5 下线 | BT | BT(单写) | PG 归档到 GCS,保留 90d |

### 2. 每阶段强制动作

#### 阶段 0:准备(T-14d ~ T-7d)

- [ ] Bigtable instance 就绪,3 张表建好(按 [schemas/column-families.yaml](../../schemas/column-families.yaml))
- [ ] `asset-service` 支持 "write mode" 配置:`pg_only` | `pg_primary_bt_shadow` | `dual` | `bt_primary_pg_shadow` | `bt_only`
- [ ] `asset-service` 支持 "read mode" 配置:`pg` | `bt_shadow_compare` | `bt`
- [ ] Backfill 工具(`grace-migrate`)就绪,支持:断点续跑、按 `updated_at` 窗口、并发度可调
- [ ] 观测:BT 写入延迟 / 错误率 dashboard、PG↔BT diff 审计表(`migration_diff`)

#### 阶段 1:开启双写(T-7d ~ T-5d)

- 切 `pg_primary_bt_shadow` 写模式:**PG 同步 commit 成功后,异步推 BT**(失败进 DLQ,不阻塞用户请求)
- **不写索引表**(避免回填期半写状态)
- MCL 仍由 PG Outbox 发,Pub/Sub 不变
- **通过标准**:BT 主表新写入跟 PG 对齐率 > 99.99%(24h 滑窗),DLQ 积压清零

#### 阶段 2:索引回填(T-5d ~ T-2d)

**子阶段 2a:存量 backfill**

- 从 `assets.updated_at ASC` 扫 PG,按 1000 条/batch 推:
  1. 写 BT `assets` 主表(用 v1 row key)
  2. 写 `asset_locator`
  3. 写 `assets_by_project`
- 并发度建议 **32**(对齐 salt 桶数),每个 batch 带 `request_id = uuid5(namespace, asset_uuid)` 保证可重入
- **通过标准**:
  - PG 行数 == BT `assets` 行数(`COUNT`)
  - `asset_locator` 行数 == BT `assets` 行数
  - `assets_by_project` 行数 == BT `assets` 行数

**子阶段 2b:增量追平**

- backfill 完成后,打开 `dual` 写模式:**三表同步写**(按 [ADR-007 §4](ADR-007-write-path-and-event-contract.md) 顺序 + janitor)
- janitor 启用,每 10s 扫最近 60s 的主表行,补齐索引
- **通过标准**:janitor 补偿次数 < 10/min(意味着在线双写基本都成功)

#### 阶段 3:影子读校验(T-2d ~ T-1d)

- 切 `bt_shadow_compare` 读模式:**返回 PG 结果**,后台并发查 BT
- 每次 diff 落 `migration_diff` 表,字段:`asset_id, pg_checksum, bt_checksum, missing_in, diff_aspects`
- **通过标准**(至少 24h 持续观测):
  - 点查(GetAsset)diff 率 < 10 ppm
  - 列表(ListRecent)diff 率 < 100 ppm
  - 所有 diff 都能人工归因到"秒级延迟 / 已知边界"

#### 阶段 4:切读(T)

- 切 `bt` 读模式(经 `asset_locator`),PG 仍在双写但**不读**
- 切 `bt_primary_pg_shadow` 写模式:**BT 同步 commit** + PG 异步兜底(24h 过渡)
- **此刻起 ADR-001 的 SLO 正式生效**:
  - locator 一跳 + GetRow 端到端 P99 < 50ms
  - locator 孤儿率 < 1/百万
  - janitor 补偿次数 < 10/min
- 回滚窗口:**48h 内可一键切回 `pg` 读模式**(PG 仍是完整数据)

#### 阶段 5:PG 下线(T+7d ~ T+30d)

- 切 `bt_only` 写模式,PG 不再写
- PG 快照导出到 `gs://grace-archive/pg-phase0/<date>/`,Cloud SQL instance 保留 90d 后删除
- 移除 `asset-service` 里所有 PG 代码路径
- Outbox relay 切到 BT 的 Change Streams + Dataflow

### 3. 回滚矩阵

| 发生在 | 回滚动作 | 回滚时间 |
| --- | --- | --- |
| 阶段 1 失败 | 关闭 BT shadow 写,回 `pg_only` | <1min(配置切换) |
| 阶段 2a 失败 | 暂停 backfill,清空 BT 三张表,重跑 | <30min |
| 阶段 2b failed | 关闭 janitor,回 `pg_primary_bt_shadow` | <1min |
| 阶段 3 diff 超阈 | 停 shadow compare,排查后重来 | <5min |
| 阶段 4 切读后 48h 内异常 | **一键切回 `pg` 读**(PG 仍在双写,数据完整) | <1min(配置切换) |
| 阶段 4 切读 48h 后异常 | **只能前向修复**(PG 开始落后),走应急 backfill | 视 gap 决定 |
| 阶段 5 下线后异常 | **无回滚**,从 GCS 归档恢复或只读 BT | 降级服务 |

**关键窗口**:**阶段 4 切读后的前 48h 是最后的无损回滚窗口**,此后必须前向修复。SRE 排班建议这 48h 专人在线。

### 4. 观测指标(必须全部有 dashboard + alert)

| 指标 | 阈值 | 何时生效 |
| --- | --- | --- |
| `bt.write.latency.p99` | < 30ms | 阶段 1 起 |
| `bt.write.error_rate` | < 0.1% | 阶段 1 起 |
| `backfill.pg_rows - bt_rows` | = 0 | 阶段 2 末 |
| `asset_locator.orphan_rate` | < 1/百万(60s 窗口未补齐) | **阶段 4 起** |
| `janitor.compensation_per_min` | < 10/min | 阶段 2b 起 |
| `bt.shadow_diff_rate` | < 10 ppm(点查)/ 100 ppm(列表) | 阶段 3 |
| `bt.read.e2e_latency.p99`(含 locator 一跳) | < 50ms | **阶段 4 起** |
| `idempotency.replay_count` | 正常流量 5% 以内 | 阶段 1 起 |

---

## Alternatives Considered

| 方案 | 为什么不选 |
| --- | --- |
| **一次性切换**(停服务 + 导数据 + 切新库) | 生产数据平台不接受停机;数据量大时窗口不够 |
| **只双写,不影子读**(跳过阶段 3) | diff 只能在切读时暴露,出事即事故,无可控回滚 |
| **backfill 完立即切读,不过双写过渡** | 无在线增量兜底,backfill 期间新写入会分叉 |
| **把 locator 建成 Bigtable 里的另一个列族,而不是独立表** | 破坏 row key 设计目标;且 scan ID → row key 的效率反不如独立表 |
| **用 Spanner 做主表绕过跨行事务问题** | 与 [ADR-001](ADR-001-wide-table-with-bigtable.md) 相悖,且 10x 成本 |
| **Phase 0 一开始就上 Bigtable,不走 PG** | 违反 ADR-001 "降低早期风险" 的 mitigation;团队不熟 BT 建模 |

---

## Consequences

### Positive
- ✅ **全程可回滚到阶段 4 切读 + 48h 之前**,风险可控
- ✅ 每个阶段有明确的通过标准,SRE 可以照单 checkbox 执行
- ✅ SLO 生效时点清晰(阶段 4),不会出现"刚切就被考核"的混乱
- ✅ 给 ADR-001 / ADR-007 的抽象契约配上了可执行 runbook

### Negative
- ⚠️ 整体迁移 T-14d ~ T+30d,**跨度 6 周**,期间双写成本上浮(约 PG + BT 费用叠加)
- ⚠️ 阶段 4 切读后 48h 内需要 SRE 排班盯守
- ⚠️ backfill 期间 PG 的 `updated_at` 索引压力会增加,需要提前评估

### Mitigations
- 双写成本可接受(Phase 0 本来流量不大);backfill 完成后 PG 立即进入只写模式,负载下降
- 48h 盯守期可与 BT 团队协同(GCP support 升级 case priority)
- backfill 前提前给 PG `updated_at` 加 btree 索引 + 用 `SET LOCAL statement_timeout` 防长事务

---

## Acceptance Criteria

- [ ] 每个阶段的通过标准都有**自动化检测**(不是人肉 grafana 看)
- [ ] `grace-migrate` 工具覆盖 backfill + diff + 校验三类场景,有单测
- [ ] 阶段 4 切读前,**SRE 和 asset-service owner 双签**(Release Gate)
- [ ] Runbook 真跑过至少一次 **staging 全链路演练**(用脱敏生产流量 shadow)
- [ ] 回滚脚本在每个阶段都**实际演练过一次**,不是纸上谈兵

---

## References

- Google Cloud: [Migrate from Cloud SQL to Bigtable](https://cloud.google.com/architecture/migrating-data-to-bigtable)
- Martin Kleppmann, *Designing Data-Intensive Applications* ch.4 "Evolvability"(双写 + backfill + 影子读 的经典范式)
- [ADR-001 §Mitigations](ADR-001-wide-table-with-bigtable.md) —— 本 ADR 的上游承接
- [ADR-007 §4](ADR-007-write-path-and-event-contract.md#4-多表写原子性phase-1-bigtable-起) —— 本 ADR 依赖的写入契约
