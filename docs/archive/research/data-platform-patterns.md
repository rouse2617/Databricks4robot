# 数据平台设计模式

本文档总结了从卓驭科技架构文档中提炼的、适用于 cyber-databrew 项目的五个核心设计模式。这些模式不是照搬卓驭的技术栈（他们用 HBase/Lindorm/Gravitino/Trino），而是借鉴其**设计理念**，在你现有的架构上提前埋好口子，避免未来补字段、补索引、补逻辑的痛苦。

---

## 1. 资产和文件引用收敛（cf_files）

### 问题

你有五六个算法（hand_tracking、head_tracking、body_tracking、deface、action_annotation、env_analysis），每个算法的产物 GCS URI 散落在 `cf_algo` 里，键名是 `hand_tracking@1.2.0:gcs_uri`。要拿到一个 asset 的所有文件，得遍历 cf_algo 的所有 key 去找 `:gcs_uri` 后缀。

```json
// 当前：散落
cf_algo = {
  "hand_tracking@1.2.0:status": "success",
  "hand_tracking@1.2.0:gcs_uri": "gs://bucket/results/ht.npz",
  "deface@2.0.0:status": "success",
  "deface@2.0.0:gcs_uri": "gs://bucket/results/deface.mp4",
  "body_tracking@1.0.0:status": "success",
  "body_tracking@1.0.0:gcs_uri": "gs://bucket/results/bt.npz",
  ...
}
```

### 解决方案

加一个 `cf_files` 列族，把所有文件引用收敛到一个地方：

```sql
-- assets 表加一个 JSONB 列
ALTER TABLE assets ADD COLUMN IF NOT EXISTS
    cf_files JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_assets_cf_files_gin
    ON assets USING GIN (cf_files);
```

```json
// cf_files：收敛
{
  "raw_mcap": "gs://grace-raw/mcap/2026-04/abc.mcap",
  "hand_tracking@1.2.0": "gs://grace-algo/ht/abc-seg01.npz",
  "head_tracking@1.0.0": "gs://grace-algo/head/abc-seg01.npz",
  "body_tracking@1.0.0": "gs://grace-algo/body/abc-seg01.npz",
  "deface@2.0.0": "gs://grace-algo/deface/abc-seg01.mp4",
  "action_annotation@1.0.0": "gs://grace-algo/action/abc-seg01.json",
  "env_analysis@1.0.0": null,
  "thumbnail": "gs://grace-thumbs/abc-seg01.jpg"
}
```

### 写入时机

- **ingest 阶段**：`cf_files["raw_mcap"] = mcap_files.cf_meta.gcs_path`
- **finish_algo(success)**：`cf_files[algo_key] = request.gcs_uri`
- **finish_algo(failed) 或 reset**：`cf_files[algo_key] = null`

### Go model 改动

```go
// models/asset.go
type Asset struct {
    // ... 现有字段 ...
    Files map[string]string `json:"files,omitempty"`  // 新增
}
```

### 收益

- **一次查询拿到所有文件**：`SELECT cf_files FROM assets WHERE asset_id = $1`
- **存储治理**：`SELECT asset_id, cf_files FROM assets WHERE cf_files ? 'deface@2.0.0'`
- **统计存储成本**：遍历 cf_files 的 value 就能算出每个 asset 占了多少 GCS 空间
- **Phase 1 Bigtable**：直接映射到 `cf:files` 列族，零改造

### 实施时机

**现在就做**。五六个算法产物已经足够复杂，等后面再补会很痛苦。改动成本：半天。

---

## 2. 字段注册防腐化（tag_registry.yaml）

### 问题

你的 `cf_algo` 已经有 `algo_registry.yaml` 做验证了，但 `cf_tag` 是完全自由的 map。团队扩大后，不同人会往 cf_tag 里写各种东西：

```json
// 张三写的
{"priority": "high", "scene": "indoor"}

// 李四写的  
{"Priority": "HIGH", "Scene": "室内", "prio": "1"}

// 王五写的
{"priority": "h", "indoor": "true"}
```

### 解决方案

加一个 `config/tag_registry.yaml`：

```yaml
tags:
  priority:
    description: "处理优先级"
    type: enum
    values: ["critical", "high", "medium", "low"]

  quality:
    description: "数据质量评级"
    type: enum
    values: ["excellent", "good", "acceptable", "poor", "unusable"]

  scene:
    description: "采集场景"
    type: enum
    values: ["indoor", "outdoor", "warehouse", "office", "factory"]

  task:
    description: "采集任务标识"
    type: string

  batch:
    description: "采集批次号"
    type: string

  notes:
    description: "自由备注"
    type: string
    max_length: 500
```

Go 加载逻辑：

```go
// internal/config/tag_registry.go
type TagDef struct {
    Description string   `yaml:"description"`
    Type        string   `yaml:"type"`        // "enum" | "string"
    Values      []string `yaml:"values"`       // only for enum
    MaxLength   int      `yaml:"max_length"`   // only for string
}

type TagRegistry struct {
    Tags map[string]TagDef `yaml:"tags"`
}

func (r *TagRegistry) Validate(key, value string) error {
    def, ok := r.Tags[key]
    if !ok {
        return fmt.Errorf("unknown tag key %q; register it in tag_registry.yaml first", key)
    }
    switch def.Type {
    case "enum":
        for _, v := range def.Values {
            if v == value { return nil }
        }
        return fmt.Errorf("tag %q value %q not in allowed values %v", key, value, def.Values)
    case "string":
        if def.MaxLength > 0 && len(value) > def.MaxLength {
            return fmt.Errorf("tag %q value exceeds max length %d", key, def.MaxLength)
        }
    }
    return nil
}
```

### 实施时机

**现在就做**。和 algo_registry 一样的模式，团队已经熟悉了。改动成本：半天。

---

## 3. 生命周期字段预留

### 问题

卓驭踩过的最大坑：数据量涨到几十 PB 后才开始做存储治理，补字段补得很痛苦。MCAP 原始文件很大（几百 MB 到几 GB），算法结果也会累积，GCS 是要钱的。

### 解决方案

在 asset 的 cf_meta 里预留生命周期字段（ingest 时写入）：

```json
{
  "retention_tier": "standard",
  "archive_after_days": 90,
  "delete_after_days": 365,
  "total_size_bytes": 0,
  "last_accessed_at": null
}
```

| 字段 | 写入时机 | 说明 |
|------|---------|------|
| `retention_tier` | ingest 时默认 `standard` | `standard` / `archive` / `frozen`，决定存储类别 |
| `archive_after_days` | ingest 时默认 90 | 多少天后自动降级到 Nearline/Archive |
| `delete_after_days` | ingest 时默认 365 | 多少天后可以删除（需人工确认或自动） |
| `total_size_bytes` | 每次 finish_algo 累加 | 该 asset 所有文件的总大小 |
| `last_accessed_at` | 每次 GET asset 或算法读取时更新 | 最后一次被访问的时间 |

`total_size_bytes` 的更新逻辑：finish_algo 时，如果 request body 里带了 `result_size_bytes`，就累加到 cf_meta：

```sql
-- finish_algo 的 SQL 里加一行
cf_meta = cf_meta || jsonb_build_object(
    'total_size_bytes',
    COALESCE((cf_meta->>'total_size_bytes')::BIGINT, 0) + $size
)
```

### 实施时机

**现在就埋字段**。逻辑以后再写，成本几乎为零，但省去了未来加字段 + backfill 的痛苦。改动成本：2 小时。

---

## 4. Ingest 幂等

### 问题

你已经有了 `raw_hash_md5` 唯一索引防重复文件，但 ingest 流程还有两个幂等缺口：

**缺口 A：ingest job 被重试**

编排系统触发了 ingest job，job 跑到一半挂了（比如 segment 切了 3 个，还有 2 个没切），编排系统自动重试。如果不做幂等，会重复创建 segment。

**缺口 B：start_algo 被重复调用**

你的 spec 里 Requirement 5 已经处理了这个：`status == running` 时返回 409。但还有一个边界情况：status 是 `success` 时不应该允许再 start（除非先 reset）。

**缺口 C：finish_algo 被重复调用**

编排系统调了 finish_algo(success)，网络超时没收到响应，重试又调了一次。

### 解决方案

#### 缺口 A：ingest job 幂等

在 ingest job 开始时检查 `mcap_files.cf_process.ingest_state`：

```python
# ingest job 伪代码（编排工具无关）

def ingest_mcap(mcap_file_id: str):
    mcap = client.mcap.get(mcap_file_id)
    
    # 幂等检查：已经 summarized 就跳过
    if mcap.cf_process.get("ingest_state") == "summarized":
        context.log.info(f"mcap {mcap_file_id} already ingested, skipping")
        return
    
    # 幂等检查：正在处理中（另一个 job 在跑）
    if mcap.cf_process.get("ingest_state") == "running":
        context.log.warning(f"mcap {mcap_file_id} ingest already running, skipping")
        return
    
    # 标记为 running（乐观锁防并发）
    client.mcap.update_process_state(mcap_file_id, "ingest_state", "running",
                                      expected_version=mcap.version)
    
    try:
        segments = parse_and_split(mcap)
        for seg in segments:
            # 用 start_timestamp_ns 做幂等：同一个 mcap + 同一个起始时间 = 同一个 segment
            existing = client.assets.list(
                filter=[f"mcap_file_id:eq:{mcap_file_id}",
                        f"start_timestamp_ns:eq:{seg.start_ns}"],
            )
            if existing.total > 0:
                context.log.info(f"segment at {seg.start_ns} already exists, skipping")
                continue
            client.assets.create(seg)
        
        client.mcap.update_process_state(mcap_file_id, "ingest_state", "summarized")
    except Exception as e:
        client.mcap.update_process_state(mcap_file_id, "ingest_state", "failed",
                                          error=str(e))
        raise
```

#### 缺口 B & C：start_algo / finish_algo 幂等

- **start_algo**：`status == running` 时返回 409；`status == success/failed` 时返回 409（必须先 reset）
- **finish_algo**：如果 `run_id` 相同且 status 已经是 success，直接返回 200（幂等成功）而不是 409

```go
// usecase 层 finish_algo
currentStatus := asset.AlgoResults[algoKey+":status"]
currentRunID  := asset.AlgoResults[algoKey+":run_id"]

if currentStatus == "success" && currentRunID == req.RunID {
    // 幂等：同一个 run 已经成功过了，直接返回
    return asset, nil
}
if currentStatus != "running" {
    return nil, ErrInvalidStateTransition
}
```

### 实施时机

**现在就做**。改动成本：半天。

---

## 5. 事件驱动替代轮询

### 问题

你有五六个算法，假设依赖关系是这样的：

```
mcap_ingest
    ├→ hand_tracking@1.2.0  ─┐
    ├→ head_tracking@1.0.0   ├→ action_annotation@1.0.0
    ├→ body_tracking@1.0.0  ─┘
    ├→ deface@2.0.0
    └→ env_analysis@1.0.0
```

纯轮询模式下，你需要 6 个轮询任务，每个 30 秒查一次 pending 状态。action_annotation 的轮询还得查 hand+head+body 三个都 success 才能触发。这能跑，但有两个问题：

1. **延迟**：hand_tracking 完成后，action_annotation 的轮询最多要等 30 秒才发现
2. **查询浪费**：大部分轮询查到的结果是"没有新的 pending"

### 解决方案

用事件驱动替代：在 `finish_algo` 的 API 里，算法完成后检查下游依赖是否满足，自动把下游算法从 `blocked` 推到 `pending`。

先在 algo_registry.yaml 里声明依赖：

```yaml
algorithms:
  hand_tracking:
    description: "手部追踪"
    versions: ["1.2.0"]
    depends_on: []                    # 无依赖，ingest 完就能跑
    output:
      required_fields: ["gcs_uri", "type"]
      uri_required: true
      report_size: true

  action_annotation:
    description: "自动动作标注"
    versions: ["1.0.0"]
    depends_on: ["hand_tracking@1.2.0", "head_tracking@1.0.0", "body_tracking@1.0.0"]
    output:
      required_fields: ["gcs_uri"]
      uri_required: true
      report_size: true
```

状态机扩展，加一个 `blocked` 状态：

```
blocked → pending → running → success
                            → failed → pending (reset)
```

ingest 创建 asset 时，初始化所有算法状态：

```json
{
  "hand_tracking@1.2.0:status": "pending",
  "head_tracking@1.0.0:status": "pending",
  "body_tracking@1.0.0:status": "pending",
  "deface@2.0.0:status": "pending",
  "action_annotation@1.0.0:status": "blocked",
  "env_analysis@1.0.0:status": "pending"
}
```

`depends_on` 为空的 → `pending`（轮询立刻能捡到）
`depends_on` 非空的 → `blocked`（轮询忽略）

finish_algo 的核心改动 — 完成后检查下游：

```go
// usecase/asset/finish_algo.go

func (uc *UseCase) FinishAlgo(ctx context.Context, assetID, algoKey string, req FinishRequest) error {
    // ... 现有的状态更新逻辑 ...

    // 新增：检查哪些下游算法依赖了刚完成的这个算法
    if req.Status == "success" {
        uc.tryUnblockDownstream(ctx, asset, algoKey)
    }
    return nil
}

func (uc *UseCase) tryUnblockDownstream(ctx context.Context, asset *models.Asset, completedAlgo string) {
    for _, algo := range uc.algoRegistry.Algorithms {
        for _, ver := range algo.Versions {
            downstream := algo.Name + "@" + ver
            if !dependsOn(algo, completedAlgo) {
                continue
            }
            // 检查这个下游算法当前是否 blocked
            if asset.AlgoResults[downstream+":status"] != "blocked" {
                continue
            }
            // 检查它的所有依赖是否都 success
            allMet := true
            for _, dep := range algo.DependsOn {
                if asset.AlgoResults[dep+":status"] != "success" {
                    allMet = false
                    break
                }
            }
            if allMet {
                // 解锁：blocked → pending
                asset.AlgoResults[downstream+":status"] = "pending"
                // 写事件日志
                uc.eventRepo.Insert(ctx, assetID, downstream, "blocked", "pending", "")
            }
        }
    }
    // 批量更新 cf_algo
    uc.assetRepo.Set(ctx, asset)
}
```

### 收益

- **延迟降低**：hand_tracking 完成后，action_annotation 立刻从 blocked 变成 pending，轮询立刻捡到，不需要等 30 秒
- **查询减少**：所有轮询只查 `status = pending`，不需要自己判断依赖关系
- **逻辑简化**：轮询逻辑变得极其简单，所有算法的轮询都一样

### 实施时机

**优先级最高**。直接影响你五六个算法能不能正确跑起来。改动成本：1 天。

---

## 总结：改动清单

| 改动 | 涉及文件 | 工作量 | 优先级 |
|------|---------|--------|--------|
| cf_files 列 | pg-phase0.sql, models/asset.go, bigtable/repos.go, finish_algo usecase | 半天 | 高 |
| tag_registry.yaml + 校验 | config/tag_registry.go, config/tag_registry.yaml, asset usecase | 半天 | 高 |
| 生命周期字段 | ingest job 里写默认值, finish_algo 累加 size | 2 小时 | 中 |
| ingest 幂等 | ingest job, mcap finalize handler | 半天 | 高 |
| finish_algo 幂等（run_id 判重） | asset usecase | 1 小时 | 高 |
| blocked 状态 + depends_on + tryUnblockDownstream | algo_registry.yaml, asset usecase, ingest job | 1 天 | 最高 |

**建议优先级**：**blocked/depends_on > cf_files > ingest 幂等 > tag_registry > 生命周期字段**。前三个直接影响你五六个算法能不能正确跑起来，后两个是防腐化和未来治理。
