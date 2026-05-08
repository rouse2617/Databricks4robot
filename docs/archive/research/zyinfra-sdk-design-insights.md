# 卓驭 zyinfra SDK 设计思路借鉴

本文档分析卓驭数据平台 Python SDK（`zyinfra`）中值得借鉴的设计思路，
并给出在 Databricks4robot 项目中的具体落地方案。

> 原则：借鉴设计思路，不照搬代码。卓驭的 SDK 调用的是他们内部的资产/存储/数据计划服务，
> API 和认证体系与我们完全不同，代码层面没有可直接复用的部分。

---

## 1. 资产字段分层：immutable vs mutable

### 卓驭的做法

卓驭把资产元数据分成三层：

| 层 | 字段 | 变更频率 | 示例 |
|----|------|---------|------|
| `typeProp` | 固有属性 | 创建时写入，几乎不变 | storePath, bloodline, startSensorTime |
| `optProp` | 可变属性 | 高频更新 | 处理状态, 抽取状态 |
| `extraData` | 附件引用 | 按需追加 | name/type/category/uri 结构化列表 |

`AssetClient.update_asset_info()` 接收 `opt_prop` 和 `type_prop` 两个独立参数，
服务端分别处理，避免高频更新的 optProp 和低频的 typeProp 互相干扰。

### 我们当前的设计

我们的 `cf_meta` JSONB 里混了两类字段：

**Immutable（创建后不变）：**
- `end_timestamp_ns`, `duration_sec`, `type`, `env`, `task`
- `owner`, `reviewer`

**Mutable（会被更新）：**
- `last_delivered_at`, `delivery_count`, `last_delivered_to`（交付触发器更新）
- `retention_tier`, `archive_after_days`, `total_size_bytes`, `last_accessed_at`（生命周期治理更新）

### 落地方案

**Phase 0（现在）：** 不改代码，在 `models/asset.go` 的注释里标记字段的变更特征：

```go
type Asset struct {
    // cf:meta — immutable fields (written at creation, never updated)
    StartTimestampNs int64       `json:"start_timestamp_ns"`
    EndTimestampNs   int64       `json:"end_timestamp_ns"`
    DurationSec      float64     `json:"duration_sec"`
    Reviewer         string      `json:"reviewer"`
    Owner            string      `json:"owner"`
    SegType          string      `json:"type,omitempty"`
    Env              string      `json:"env,omitempty"`
    Task             string      `json:"task,omitempty"`

    // cf:meta — mutable fields (updated by triggers, lifecycle jobs, etc.)
    LastDeliveredAt *time.Time `json:"last_delivered_at,omitempty"`
    LastDeliveredTo string     `json:"last_delivered_to,omitempty"`
    DeliveryCount   int        `json:"delivery_count"`
}
```

**Phase 1（Bigtable 迁移时）：** 把 mutable 字段拆到独立列族 `cf:state`，
和 immutable 的 `cf:meta` 分开，设置不同的 GC policy：

```
cf:meta  → maxVersions=1, 不设 TTL（永久保留）
cf:state → maxVersions=3, TTL=90d（保留最近 3 个版本，90 天过期）
cf:algo  → maxVersions=1（只保留最新状态）
cf:tag   → maxVersions=1
cf:files → maxVersions=1
```

### 工作量

Phase 0：在 `models/asset.go` 加注释，0 成本。
Phase 1：拆列族时有明确的分界线，不需要临时判断。

---

## 2. 事件订阅泛化：Asset-Is-Subscribed

### 卓驭的做法

`AssetClient.update_asset_info()` 有一个 `subscribed` 参数（默认 True），
通过 HTTP header `Asset-Is-Subscribed` 传递给服务端。
服务端在资产更新后检查这个 header，决定是否触发下游订阅
（自动下发质检任务、自动触发数据计划等）。

```python
def update_asset_info(self, asset_id, updated_by, opt_prop, type_prop, subscribed=True):
    headers['Asset-Is-Subscribed'] = str(subscribed).lower()
```

这让调用方可以控制"这次更新要不要触发下游"——
批量 backfill 时设 `subscribed=False` 避免触发风暴，正常业务流程设 `True`。

### 我们当前的设计

我们的 `tryUnblockDownstream` 在 `finish_algo(ok)` 时自动触发，没有开关。
`asset_algo_events` 表记录了所有状态变更，但没有 consumer 读这张表。

### 落地方案

**Phase 0（现在）：** 给 `finish_algo` 和 `reset` 的 API 加一个可选的 `suppress_events` 参数：

```json
POST /api/v1/assets/:id/algo/:algo_key/finish
{
  "status": "ok",
  "output_uri": "gs://...",
  "suppress_events": false    // 可选，默认 false
}
```

当 `suppress_events=true` 时：
- 仍然写 `asset_algo_events`（审计记录不能跳过）
- 但跳过 `tryUnblockDownstream`（不触发下游依赖解锁）

使用场景：批量重跑算法时，先把所有 asset 的某个算法 reset + 重跑，
不希望每个 finish 都触发下游解锁（因为下游算法也要重跑），
等全部跑完后再手动触发一次解锁。

**Phase 1（未来）：** 把 `asset_algo_events` 作为通用事件源，
加一个异步 consumer（可以是编排系统的 sensor，也可以是独立的 worker），
读事件表做更复杂的触发逻辑：

```
asset_algo_events 表
    │
    ├→ 依赖解锁 consumer：blocked → pending
    ├→ 通知 consumer：所有算法都 ok → 发通知
    ├→ 自动交付 consumer：QA 通过 → 触发交付
    └→ 指标聚合 consumer：统计算法成功率/耗时
```

### 工作量

Phase 0：`suppress_events` 参数，在 `algo_usecase.go` 加一个 if 判断，1 小时。
Phase 1：consumer 架构，等有需求时再做。

---

## 3. 资产血缘：source → target 双向关联

### 卓驭的做法

`LifecycleClient` 有显式的血缘 API：

```python
# 建立血缘：标注资产 → 训练资产
lifecycle_client.create_asset_lineage(source_id=ann_result_id, target_id=training_asset_id)

# 查询顶层血缘：从任意资产追溯到原始资产
result = HTTPRequestTool.request(GET, '/v1/api/bloodline/assets', params={'assetId': asset_id})
```

`TrainDataAssetClient.create_train_data_asset()` 在创建训练资产时，
自动建立三层血缘：

```
原始资产 (original_data)
    └→ 标注资产 (ann_result_data)
        └→ 训练资产 (train_data)
```

并且在 `typeProp.bloodline` 里记录完整的血缘链：

```json
{
  "bloodline": {
    "originalParentAsset": {
      "assetTypeName": "original_data",
      "parentAssetIds": "xxx",
      "startSensorTime": "...",
      "endSensorTime": "..."
    },
    "annResultParentAsset": {
      "assetTypeName": "ann_result_data",
      "parentAssetIds": ["yyy", "zzz"]
    },
    "topOriginalAsset": {
      "assetTypeName": "original_data",
      "parentAssetId": "aaa"
    }
  }
}
```

### 我们当前的设计

我们的血缘关系是隐式的：
- `mcap_files → assets`：通过 `mcap_file_id` FK
- `assets → algo results`：通过 `cf_algo` 里的 `run_id`
- `assets → deliveries`：通过 `delivery_items` 关联表

缺少 **asset → asset** 的血缘关系。

### 落地方案

**Phase 0（现在）：** 在 `algo_registry.yaml` 里预留 `creates_asset` 标记，
不实现血缘表，但明确哪些算法的产物只是文件、哪些可能产生新 asset：

```yaml
algorithms:
  hand_tracking:
    creates_asset: false    # 产物是 .npz 文件，不创建新 asset
  deface:
    creates_asset: false    # 产物是 .mp4 文件，不创建新 asset
  segment_merge:            # 未来可能有
    creates_asset: true     # 产物是一个新 asset，需要建血缘
```

**Phase 1（需要时）：** 加一张 `asset_lineage` 表：

```sql
CREATE TABLE IF NOT EXISTS asset_lineage (
    source_asset_id  UUID NOT NULL REFERENCES assets(asset_id),
    target_asset_id  UUID NOT NULL REFERENCES assets(asset_id),
    relation_type    TEXT NOT NULL,  -- 'derived_from', 'merged_from', 'split_from'
    algo_key         TEXT,           -- 产生这条血缘的算法，如 'segment_merge@1.0.0'
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (source_asset_id, target_asset_id)
);

CREATE INDEX IF NOT EXISTS idx_lineage_target
    ON asset_lineage (target_asset_id, source_asset_id);
```

查询方式：
- "这个 asset 是从哪些 asset 派生的"：`WHERE target_asset_id = $1`
- "这个 asset 派生出了哪些 asset"：`WHERE source_asset_id = $1`
- "追溯到顶层原始 asset"：递归 CTE

### 工作量

Phase 0：`algo_registry.yaml` 加 `creates_asset` 字段，0 成本。
Phase 1：建表 + API + 在 `finish_algo` 里自动建血缘，1-2 天。

---

## 4. 确定性唯一键幂等（idempotency_uri）

### 卓驭的做法

`TrainDataAssetClient.create_train_data_asset()` 用业务语义构造确定性唯一键：

```python
original_data_uri = (
    f"task://task_id/{self.current_task_id}"
    f"/dataplan_id/{self.data_plan_id}"
    f"/ann_result_asset_ids_md5/{md5}"
    f"/original_data_asset_id/{original_data_asset_id}"
)
# 用这个 URI 创建资产，如果 URI 已存在就返回已有的 asset_id
training_data_asset_id = self.asset_client.create_asset(asset_info=asset_body)['assetId']
```

核心思想：**同样的输入永远产生同一个 asset_id**，"创建"操作天然幂等。
不需要"先查再创建"，没有查和创建之间的竞态条件。

然后通过检查 `typeProp` 是否为空来判断是新创建的还是已存在的：

```python
is_new_asset_id = self._is_new_asset_id(training_data_asset_id)
# 新创建的 → typeProp 为空 → 需要填充元数据
# 已存在的 → typeProp 非空 → 跳过或更新
```

### 我们当前的设计

我们的 ingest 幂等方案是"先查再创建"：

```python
# data-platform-patterns.md 里的方案
existing = client.assets.list(
    filter=[f"mcap_file_id:eq:{mcap_file_id}",
            f"start_timestamp_ns:eq:{seg.start_ns}"],
)
if existing.total > 0:
    continue  # 已存在，跳过
client.assets.create(seg)
```

这有竞态条件：两个并发的 ingest job 同时查到"不存在"，然后都创建，产生重复。

### 落地方案

在 `assets` 表加一个 `idempotency_uri` 列，用 `mcap_file_id + start_timestamp_ns` 构造确定性唯一键：

```sql
-- Schema 改动
ALTER TABLE assets ADD COLUMN IF NOT EXISTS
    idempotency_uri TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_idempotency_uri
    ON assets (idempotency_uri)
    WHERE idempotency_uri IS NOT NULL AND is_deleted = FALSE;
```

ingest 时构造唯一键：

```
idempotency_uri = "ingest://<mcap_file_id>/<start_timestamp_ns>"
```

Repository 层用 `INSERT ... ON CONFLICT DO NOTHING`：

```go
// postgres/repos.go — 新增方法
func (r *AssetRepo) CreateIdempotent(ctx context.Context, a *models.Asset) (created bool, err error) {
    now := time.Now().UTC()
    if a.CreatedAt.IsZero() {
        a.CreatedAt = now
    }
    a.UpdatedAt = now
    a.Version = 1
    meta, algo, tag := assetJSON(a)

    const q = `
INSERT INTO assets(
    asset_id, mcap_file_id, start_timestamp_ns, status, is_deleted,
    cf_meta, cf_algo, cf_tag, idempotency_uri,
    created_at, updated_at, version
)
VALUES ($1,$2,$3,$4,FALSE,$5::jsonb,$6::jsonb,$7::jsonb,$8,$9,$10,$11)
ON CONFLICT (idempotency_uri) WHERE idempotency_uri IS NOT NULL AND is_deleted = FALSE
DO NOTHING
RETURNING asset_id`

    var returnedID string
    err = r.c.db.QueryRow(ctx, q,
        a.AssetID, a.McapFileID, a.StartTimestampNs, string(a.Status),
        meta, algo, tag, a.IdempotencyURI,
        a.CreatedAt, a.UpdatedAt, a.Version,
    ).Scan(&returnedID)

    if err != nil {
        if errors.Is(err, errNoRows) {
            // ON CONFLICT DO NOTHING — row already exists
            return false, nil
        }
        return false, fmt.Errorf("postgres AssetRepo.CreateIdempotent: %w", err)
    }
    return true, nil
}
```

Model 改动：

```go
// models/asset.go
type Asset struct {
    // ... 现有字段 ...
    IdempotencyURI string `json:"idempotency_uri,omitempty"`
}
```

SDK 调用方式：

```python
# ingest job 里
for seg in segments:
    seg.idempotency_uri = f"ingest://{mcap_file_id}/{seg.start_timestamp_ns}"
    created = client.assets.create_idempotent(seg)
    if not created:
        log.info(f"segment at {seg.start_timestamp_ns} already exists, skipping")
```

### 工作量

Schema + Model + Repository 方法 + SDK 方法：半天。

---

## 5. 事务包裹：状态变更 + 审计记录强一致

### 卓驭的做法

`TrainDataAssetClient.create_train_data_asset()` 有完整的回滚机制：

```python
try:
    # 步骤 1：创建资产
    training_data_asset_id = self.asset_client.create_asset(...)
    is_new = self._is_new_asset_id(training_data_asset_id)

    # 步骤 2：拷贝文件
    self.storage_client.copy_dir(src, dst)

    # 步骤 3：更新元数据
    self.asset_client.update_asset_info(...)

    # 步骤 4：建立血缘
    self.lifecycle_client.create_asset_lineage(...)

except Exception as e:
    # 任何步骤失败 → 回滚：删除新资产 + 删除已拷贝的文件
    self._rollback_on_failure(is_new, training_data_asset_id, remote_path)
    raise
```

### 我们当前的设计

我们的 `MergeCfAlgo` 和事件日志写入是两个独立操作：

```go
// algo_usecase.go（当前）
newVersion, err := repo.MergeCfAlgo(ctx, assetID, version, algoKV, filesKV)  // 步骤 1
if err != nil { return err }
err = eventRepo.Insert(ctx, event)  // 步骤 2 — 如果这里失败，状态已变但没有审计记录
```

如果步骤 1 成功但步骤 2 失败，会导致：
- `cf_algo` 里算法状态已经变了
- 但 `asset_algo_events` 里没有对应的审计记录
- 排查问题时会发现"状态变了但不知道什么时候变的"

### 落地方案

在 `pgDB` 接口上加事务支持，把 `MergeCfAlgo` + 事件日志放在同一个事务里：

```go
// postgres/client.go — 新增事务接口
type pgDB interface {
    // ... 现有方法 ...
    BeginTx(ctx context.Context) (pgTx, error)
}

type pgTx interface {
    QueryRow(ctx context.Context, sql string, args ...any) rowScanner
    Exec(ctx context.Context, sql string, args ...any) error
    ExecResult(ctx context.Context, sql string, args ...any) (int64, error)
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}

// realDB 实现
func (r *realDB) BeginTx(ctx context.Context) (pgTx, error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    return &realTx{tx: tx}, nil
}

type realTx struct {
    tx pgx.Tx
}

func (t *realTx) QueryRow(ctx context.Context, sql string, args ...any) rowScanner {
    return t.tx.QueryRow(ctx, sql, args...)
}
func (t *realTx) Exec(ctx context.Context, sql string, args ...any) error {
    _, err := t.tx.Exec(ctx, sql, args...)
    return err
}
func (t *realTx) ExecResult(ctx context.Context, sql string, args ...any) (int64, error) {
    ct, err := t.tx.Exec(ctx, sql, args...)
    if err != nil { return 0, err }
    return ct.RowsAffected(), nil
}
func (t *realTx) Commit(ctx context.Context) error   { return t.tx.Commit(ctx) }
func (t *realTx) Rollback(ctx context.Context) error  { return t.tx.Rollback(ctx) }
```

Repository 层新增事务方法：

```go
// postgres/repos.go — 新增
func (r *AssetRepo) MergeCfAlgoWithEvent(
    ctx context.Context,
    assetID string,
    expectedVersion int64,
    algoKV map[string]interface{},
    filesKV map[string]interface{},
    event *models.AlgoEvent,
) (int64, error) {
    tx, err := r.c.db.BeginTx(ctx)
    if err != nil {
        return 0, fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx) // no-op if already committed

    // 步骤 1：合并 cf_algo + cf_files
    algoJSON, _ := json.Marshal(algoKV)
    filesJSON, _ := json.Marshal(filesKV)
    const mergeQ = `
UPDATE assets
SET cf_algo = cf_algo || $1::jsonb,
    cf_files = cf_files || $2::jsonb,
    version = version + 1,
    updated_at = now()
WHERE asset_id = $3 AND version = $4 AND is_deleted = FALSE`

    rowsAffected, err := tx.ExecResult(ctx, mergeQ, algoJSON, filesJSON, assetID, expectedVersion)
    if err != nil {
        return 0, fmt.Errorf("merge cf_algo: %w", err)
    }
    if rowsAffected == 0 {
        return 0, repository.ErrOptimisticLock
    }

    // 步骤 2：插入事件日志
    const eventQ = `
INSERT INTO asset_algo_events(asset_id, algo_key, prev_status, new_status, run_id, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, now())`
    err = tx.Exec(ctx, eventQ,
        event.AssetID, event.AlgoKey, event.PrevStatus, event.NewStatus, event.RunID, event.Reason)
    if err != nil {
        return 0, fmt.Errorf("insert event: %w", err)
    }

    // 提交事务
    if err := tx.Commit(ctx); err != nil {
        return 0, fmt.Errorf("commit tx: %w", err)
    }
    return expectedVersion + 1, nil
}
```

Usecase 层调用变化：

```go
// 之前（两步，非事务）
newVersion, err := repo.MergeCfAlgo(ctx, assetID, version, algoKV, filesKV)
eventRepo.Insert(ctx, event)

// 之后（一步，事务）
newVersion, err := repo.MergeCfAlgoWithEvent(ctx, assetID, version, algoKV, filesKV, event)
```

### 工作量

`pgDB` 接口加 `BeginTx` + `realTx` 实现 + `MergeCfAlgoWithEvent` 方法：半天。

---

## 6. 结构化质量指标上报

### 卓驭的做法

`LifecycleClient` 有一套结构化的指标上报机制：

```python
norm = Norm(identifier="BinocularCamera", index="10", norm_value=95)
report_item = ReportItem(asset_id="xxx")
report_item.add_norm(norm)
report_data = ReportData(stage_id=1, reported_by="alice")
report_data.add_report_item(report_item)
lifecycle_client.asset_stage_report(report_data)
```

每个指标有三个维度：
- `identifier`：指标唯一标识（如 "confidence", "coverage"）
- `index`：指标原始值（如 "0.95"）
- `norm_value`：归一化值 0-100（如 95）

`QualityClient` 更进一步，有完整的质检流程：
注册脚本 → 注册参数 → 上报结果 → 结束上报。

### 我们当前的设计

我们的 `finish_algo` 有 `result_meta` 自由字段，但没有结构化的指标约定：

```json
{
  "result_meta": {
    "frame_count": 3600,
    "confidence": 0.95
  }
}
```

### 落地方案

**Phase 0（现在）：** 不改 API，在文档里约定 `result_meta` 的推荐结构：

```json
{
  "result_meta": {
    "frame_count": 3600,
    "metrics": [
      {
        "id": "confidence",
        "value": 0.95,
        "norm": 95,
        "threshold": 80
      },
      {
        "id": "coverage",
        "value": 0.88,
        "norm": 88,
        "threshold": 70
      }
    ]
  }
}
```

在 `algo_registry.yaml` 里可以声明每个算法期望上报哪些指标：

```yaml
algorithms:
  hand_tracking:
    output:
      required_fields: ["output_uri", "type"]
      uri_required: true
      report_size: true
      expected_metrics: ["confidence", "coverage", "frame_count"]  # 文档性质，不强制校验
```

**Phase 1（做质量看板时）：** 从 `cf_algo` 里提取 `result_meta.metrics`，
写入时序数据库或聚合表，做算法质量趋势分析。

### 工作量

Phase 0：文档约定 + `algo_registry.yaml` 加 `expected_metrics`，0 成本。
Phase 1：提取 + 聚合，等有看板需求时再做。

---

## 总结：优先级排序

| 设计思路 | 落地方案 | 工作量 | 优先级 |
|---------|---------|--------|--------|
| 确定性唯一键幂等 | `idempotency_uri` 列 + `CreateIdempotent` 方法 | 半天 | 高（直接影响 ingest 正确性） |
| 事务包裹状态+审计 | `BeginTx` + `MergeCfAlgoWithEvent` | 半天 | 高（直接影响数据一致性） |
| 事件订阅泛化 | `suppress_events` 参数 | 1 小时 | 中（批量重跑时需要） |
| 字段分层标记 | 注释标记 immutable/mutable | 0 | 低（Phase 1 迁移时用） |
| 血缘预留 | `creates_asset` 标记 | 0 | 低（未来需要时用） |
| 结构化指标 | 文档约定 metrics 结构 | 0 | 低（做看板时用） |
