# 03 · 数据模型:宽表设计

## 1. 核心原则

> **一个资产 = 一行**。**`asset_id` 是全局主键**。**所有 aspect 通过列族展开**。

- 追求 **点查 < 50ms**,不追求跨行事务
- 追求 **无限加列**,不追求 schema 约束
- 追求 **十亿行扩展**,不追求单机 SQL 强大
- 所有复杂查询走 **索引副本**(OpenSearch / BigQuery / Vector)

---

## 2. Row Key 设计

### 2.1 设计目标

- Row key 里要有**稳定业务属性**(tenant/project),方便按业务边界扫描
- 同时保证**写入打散**(防热点)和**最近数据优先**
- 点查仍以 `asset_id` 为主入口,通过定位表跳转到主表 row key

### 2.2 Phase 0(< 100 万 asset)

```
row_key = v0#<tenant_id>#<project_id>#<asset_uuid>

示例:
  v0#t_cyber_grace#p_robot_fleet_a#a1b2c3d4-5e6f-7890-abcd-ef1234567890
```

- `tenant_id` / `project_id`:稳定业务 ID(不要用可改名展示名)
- `asset_uuid`:UUIDv4,全局唯一
- 数据量小,**无需考虑热点**

### 2.3 Phase 1+(> 100 万 asset,避免写入热点)

```
row_key = v1#<salt2>#<tenant_id>#<reverse_ts_ms_19>#<project_id>#<asset_uuid>

示例:
  v1#07#t_cyber_grace#9223370290106000000#p_robot_fleet_a#a1b2c3d4-5e6f-7890-abcd-ef1234567890

组成:
  v1                  key 版本(便于将来升级 v2)
  salt2               2 位桶号(建议先 00-31,后续可扩 00-63)
  tenant_id           稳定租户 ID
  reverse_ts_ms_19    2^63-1 - created_at_ms,固定 19 位,新资产前排
  project_id          稳定项目 ID
  asset_uuid          UUIDv4,全局唯一
```

**查询模式**:

| 模式 | 做法 |
| --- | --- |
| 点查 `asset_id` | 先查 `asset_locator`(`asset_id -> main_row_key`) → 再 `GetRow` 主表 |
| 扫某 tenant 最近 | `v1#<00..31>#<tenant_id>#` 做 32 路并行 prefix scan,客户端 top-k merge |
| 扫某 project 最近 | 走二级索引表 `assets_by_project`,做 32 路并行 prefix scan |
| 全局最近 | 不建议直接扫主表,走 BigQuery / 搜索索引副本 |

> **Phase 0 先别上 salt**,简单起步。Phase 1 迁移时采用双写 + backfill + 切读。

### 2.4 配套索引表(推荐)

| 表名 | Row key | 用途 |
| --- | --- | --- |
| `asset_locator` | `<asset_uuid>` | 点查入口,把 `asset_id` 定位到主表 row key |
| `assets_by_project` | `v1#<salt2>#<tenant_id>#<project_id>#<reverse_ts_ms_19>#<asset_uuid>` | 高效做 project 维度最近扫描 |

`asset_locator` 推荐字段:

```
main_row_key
key_ver
tenant_id
project_id
created_at_ms
status
```

---

## 3. 列族(Column Family)设计

### 3.1 全貌

| # | 列族 | 用途 | GC policy | 稀疏度 |
| --- | --- | --- | --- | --- |
| 1 | `cf:core` | 核心不变属性 | 版本=3 | 稠密 |
| 2 | `cf:time` | 时间区间(segment 语义) | 版本=3 | 稠密 |
| 3 | `cf:tag` | 业务标签,稀疏宽列 | 版本=5 | 极稀疏 |
| 4 | `cf:file` | 派生产物文件列表 | 版本=3 | 中等 |
| 5 | `cf:qa` | QA 工作流状态 | 版本=10(状态机历史) | 稠密 |
| 6 | `cf:event` | 事件流(依赖多版本) | **版本=∞,TTL=90d** | 高 |
| 7 | `cf:lineage` | 血缘关系 | 版本=3 | 中等 |
| 8 | `cf:algo` | 各算法运行状态 | 版本=5 | 中等 |
| 9 | `cf:emb` | Embedding 指针(不存向量本身) | 版本=3 | 低 |

### 3.2 详细字段

#### `cf:core`

```
type          video_segment | derived_segment | audio_clip | ...
tenant        cyber-grace
project       robot-fleet-a
urn           urn:grace:asset:<uuid>
status        active | archived | deleted
created_at    2026-04-22T10:30:00Z
created_by    alice@company.com
updated_at    2026-04-22T11:00:00Z
schema_ver    1    (本条 asset 遵循的字段 schema 版本)
```

#### `cf:time`

```
start_ns      1721520000000000000   (nanoseconds since epoch)
end_ns        1721520045000000000
duration_ns   45000000000
mcap_uri      gs://grace-raw-mcap/2026/04/22/robot42_log.mcap
mcap_size     204857600
mcap_sha256   a7f3b2c1d4e5f67890...
```

#### `cf:tag`(稀疏,可任意扩)

```
Scene.urban                  1
Scene.highway                0
Weather.rainy                1
Weather.sunny                0
Vehicle.truck.count          3
Vehicle.car.count            12
Quality.blur                 0.23
Quality.exposure             good
Algo.sam2_v3.labeled         1
user.alice.bookmark          1
user.team_a.priority         high
_auto.cv_detector_v1.ran     2026-04-22T11:05:00Z
```

> 约定:**第一段大写字母**的为官方 Classification,**小写**的为用户私有(如 `user.alice.*`)。

#### `cf:file`

```
file:original.uri              gs://grace-raw-mcap/.../robot42.mcap
file:original.kind             raw_mcap
file:original.size             204857600
file:original.sha256           a7f3b2c1...
file:original.producer         human:upload
file:original.created_at       2026-04-22T10:30:00Z

file:sam2_v3.uri               gs://grace-derived/asset-a1b2c3/sam2_v3.mcap
file:sam2_v3.kind              derived_mask
file:sam2_v3.version           v3.1.2
file:sam2_v3.producer          algo:sam2@v3.1.2
file:sam2_v3.run_id            dagster-run-abc123
file:sam2_v3.created_at        2026-04-22T11:15:00Z
file:sam2_v3.size              314572800

file:human_ann.uri             gs://grace-annotation/asset-a1b2c3/ann.json
file:human_ann.kind            human_annotation
file:human_ann.version         v1
file:human_ann.producer        human:bob@company.com
```

#### `cf:qa`

```
state          approved        ( pending | in_review | approved | rejected | disputed )
qa_by          bob@company.com
qa_at          2026-04-22T11:00:00Z
qa_notes       clear urban scene, light rain
prev_state     in_review
approved_ver   2                (第几次 approve,乐观锁)
disputes       0                (争议次数)
```

> **状态流转**通过 `CheckAndMutate` 保证原子:只有 `prev_state=in_review` 才允许写入 `state=approved`。

#### `cf:event`(多版本是精髓)

每列 = 一类事件,**每个版本 = 一次发生**。

```
e:created          { by: alice, at: ..., source: upload }
e:qa_started       { by: bob,   at: ... }
e:qa_approved      { by: bob,   at: ..., notes: "..." }
e:tag_added        { by: alice, at: ..., tag: "Scene.urban" }
e:tag_added        { by: alice, at: ..., tag: "Weather.rainy" }   ← 新版本,旧版本保留
e:algo_started     { by: system,at: ..., algo: "sam2_v3" }
e:algo_completed   { by: system,at: ..., algo: "sam2_v3", duration: 42.3 }
```

**查询**:

```
# 按时间正序看 asset 一生
scan cf:event for asset a1b2c3 sort by timestamp asc
```

#### `cf:lineage`

```
up:mcap              urn:grace:mcap:gs://grace-raw-mcap/.../robot42.mcap
up:parent_asset      urn:grace:asset:<uuid>    (若是二次切分)
up:ingestion_run     urn:grace:run:<dagster-run-id>

down:dataset_train_2026q1   urn:grace:dataset:train_2026q1
down:derived_asset_1        urn:grace:asset:<uuid>
down:export_2026_04         urn:grace:export:<export-id>
```

#### `cf:algo`

```
sam2_v3.status         done          ( pending | running | done | failed | skipped )
sam2_v3.run_id         dagster-run-abc123
sam2_v3.started_at     2026-04-22T11:10:00Z
sam2_v3.completed_at   2026-04-22T11:15:00Z
sam2_v3.duration_ms    42300
sam2_v3.error          null
sam2_v3.output_file    sam2_v3       (对应 cf:file 里的 key)

sam2_v4.status         pending       (最新版本,还没跑)

tracker_v2.status      failed
tracker_v2.error       OOM at t=23s
tracker_v2.retries     3
```

#### `cf:emb`(向量指针)

**向量本身存在 Vertex AI Vector Search,Bigtable 只存指针和元数据。**

```
clip_vit_l.vector_id      projects/x/locations/us-central1/indexEndpoints/.../datapoint/a1b2c3
clip_vit_l.model          openai/clip-vit-large-patch14
clip_vit_l.dim            512
clip_vit_l.created_at     2026-04-22T11:20:00Z

sam2_mask_emb.vector_id   ...
sam2_mask_emb.model       custom/sam2-pool
sam2_mask_emb.dim         256
```

---

## 4. 原生能力红利(Bigtable 白送)

| 用户需求 | Bigtable 机制 | 自研成本 |
| --- | --- | --- |
| 多版本 / 历史 | Cell-level timestamp + GC policy | 0 |
| 强一致 | Single-row atomic | 0 |
| 状态机原子 | `CheckAndMutate` | 0 |
| 稀疏列 | 存不存都可以,不占空间 | 0 |
| 海量扩展 | 加节点,自动分片 | 运维成本 |
| 快照备份 | 原生 backup/restore | 运维成本 |

---

## 5. Bigtable 做不到的 → 索引副本补

| 需求 | 解法 |
| --- | --- |
| 按 Tag 组合过滤(`urban AND rainy AND duration>30s`) | CDC → **OpenSearch** |
| 全文搜索 notes / annotation | CDC → **OpenSearch** |
| 跨行聚合(按 week 统计) | CDC → **BigQuery** |
| 向量 / 多模态检索 | CDC → **Vertex AI Vector Search** |
| 管理员 SQL 探索 | **BigQuery External Table** 直查 Bigtable |

**CDC 实现**:Bigtable Change Streams → Dataflow → 各索引。

---

## 6. Phase 0 的"宽表模拟"(Postgres 版)

Phase 0 还没 Bigtable,用 **Postgres JSONB + 8 个 JSONB 列** 模拟同构语义。

```sql
-- schemas/pg-phase0.sql(完整版另存)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE assets (
    asset_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant        TEXT NOT NULL,
    urn           TEXT GENERATED ALWAYS AS ('urn:grace:asset:' || asset_id::text) STORED,

    -- 8 个 JSONB = Bigtable 9 CF 里的前 8 个(cf:event 单独建 asset_events 表)
    cf_core       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_time       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_tag        JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_file       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_qa         JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_lineage    JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_algo       JSONB NOT NULL DEFAULT '{}'::jsonb,
    cf_emb        JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    version       BIGINT NOT NULL DEFAULT 1         -- 乐观锁
);

-- GIN 索引
CREATE INDEX idx_assets_tag    ON assets USING GIN (cf_tag);
CREATE INDEX idx_assets_qa     ON assets USING GIN (cf_qa);
CREATE INDEX idx_assets_file   ON assets USING GIN (cf_file);
CREATE INDEX idx_assets_tenant ON assets (tenant);

-- 事件表(Bigtable 的 cf:event 多版本 → PG 里独立表)
CREATE TABLE asset_events (
    id            BIGSERIAL PRIMARY KEY,
    asset_id      UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
    event_type    TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_events_asset_time ON asset_events (asset_id, created_at DESC);

-- "asset_id → 外部 URN 映射" (便于 MCAP URI 快速反查 asset)
CREATE TABLE external_refs (
    external_urn  TEXT PRIMARY KEY,    -- e.g. urn:grace:mcap:gs://...
    asset_id      UUID NOT NULL REFERENCES assets(asset_id),
    ref_kind      TEXT NOT NULL,       -- 'mcap' | 'dataset' | ...
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**乐观锁写示例**:

```sql
UPDATE assets
SET cf_qa    = cf_qa || '{"state":"approved","qa_by":"bob","qa_at":"..."}'::jsonb,
    version  = version + 1,
    updated_at = now()
WHERE asset_id = $1
  AND version  = $2            -- 客户端上一次读到的 version
  AND cf_qa->>'state' = 'in_review'    -- 状态机前件
RETURNING version;
```

**Phase 0 → Phase 1 迁移路径**:

1. 业务代码写 SDK,按列族访问(`asset.cf_tag['Scene.urban']`)
2. Phase 1 切 Bigtable 时,SDK 底层换实现,上层不变
3. 双写 + Backfill + 校验 + 切读 + 下线 PG

---

## 7. 命名约定(Schema Registry)

所有"官方"Classification / Tag / File kind / Algo id,通过 `schemas/fields.yaml` 统一注册。详见 [schemas/fields.yaml](../schemas/fields.yaml)。

- ✅ **可以**用户自由打 `user.<someone>.<tag>`(无需注册)
- ❌ **不可以**用户打未注册的 `Scene.xxx`(管控区)

> 理由:Tag 爆炸难运维,官方维度必须受控,私有空间自由用。

---

## 8. 反模式(不要做)

1. ❌ **不要**把同一 asset 的不同版本用不同 `asset_id` 存(用 `version` 字段)
2. ❌ **不要**在 `cf:tag` 里存大 JSON 或长字符串(tag 应该是 label,数据放别处)
3. ❌ **不要**做跨 asset 的 JOIN(用索引副本 / 冗余)
4. ❌ **不要**把高频变更(如 running 状态每秒刷新)写到 Bigtable(用 Redis / Memorystore)
5. ❌ **不要**按 Postgres 关系范式拆小表(Bigtable 思维是"一行一资产")

---

## 9. 下一步

- MCAP / Segment 细节 → [04-mcap-and-segment.md](04-mcap-and-segment.md)
- 字段注册表 → [../schemas/fields.yaml](../schemas/fields.yaml)
- PG DDL 全文 → [../schemas/pg-phase0.sql](../schemas/pg-phase0.sql)
- Bigtable CF 定义 → [../schemas/column-families.yaml](../schemas/column-families.yaml)
- ADR-001 → [adr/ADR-001-wide-table-with-bigtable.md](adr/ADR-001-wide-table-with-bigtable.md)
