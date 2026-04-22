# 04 · MCAP + Segment 模型

## 1. 为什么聚焦 MCAP

- **MCAP 是 Foxglove 推出的开源容器格式**,已成为机器人 / 自动驾驶领域事实标准
- 原生支持多 topic / 多 encoding(ROS1 / ROS2 / Protobuf / JSON / CBOR)
- **支持分块 + 压缩**,footer 含 summary 偏移
- 业内 Tesla、Wayve、Roboto AI、BMW 等广泛使用

> 详见 [ADR-003](adr/ADR-003-mcap-as-primary-format.md)。

---

## 2. 三层实体关系

```
┌──────────────────────────────────────┐
│  MCAP File(物理)                    │
│  gs://grace-raw-mcap/.../robot42.mcap│
│  size: 1.2 GB                        │
│  duration: 300s                      │
│  topics: [/cam/front, /lidar, /gps]  │
└──────────────────────────────────────┘
             │
             │  一个 MCAP → N 个 Asset
             ▼
┌──────────────────────────────────────┐
│  Asset(= Segment,平台一等公民)     │
│  asset_id: a1b2c3-...                │
│  (t=12.3s, t=57.8s)                  │
│  duration: 45.5s                     │
│  qa_state: approved                  │
│  tags: Scene.urban, Weather.rainy    │
└──────────────────────────────────────┘
             │
             │  一个 Asset → N 个 File
             ▼
┌──────────────────────────────────────┐
│  File(关联文件,kind 区分)         │
│  • original   raw_mcap 引用(不切)  │
│  • sam2_v3    派生 mask mcap         │
│  • human_ann  标注 json              │
│  • tracks     tracker 输出 parquet   │
└──────────────────────────────────────┘
```

**关键点**:
- ✅ Asset 是**逻辑概念**,主要是时间区间 + 元数据
- ✅ 派生产物以 File 形式**挂在 asset 下**,不反过来
- ✅ 一个 MCAP 文件可以**不物化切分**,多个 asset 共享一个 MCAP(通过 time range 引用)

---

## 3. Virtual Segment vs Materialized Segment

### 3.1 默认:Virtual Segment(虚拟,零拷贝)

```
Asset cf:time:
    mcap_uri:    gs://grace-raw-mcap/.../robot42.mcap
    start_ns:    1721520012300000000
    end_ns:      1721520057800000000
```

**读取时**:`mcap-gateway` 接到 SDK 请求,做 **Range GET** 读原 MCAP 指定范围,再 streaming 解压缩回来。**不下载整个 MCAP,也不复制一份新的**。

**优点**:
- ✅ 存储零增长(100 MCAP → 1000 asset,还是 100 MCAP 的空间)
- ✅ ingestion 极快,只需提取 topic / time
- ✅ 一个 MCAP 被 QA 成多个 asset,毫秒级完成

**缺点**:
- ❌ 下游 MCAP 原文件**永远不能删**,否则 asset 失效
- ❌ 时间区间不对齐 chunk 边界时,读取需跨 chunk 解压
- ❌ 删除某个 asset 不释放空间

### 3.2 按需:Materialized Segment(物化)

某些情况下真的切出独立小 MCAP:

| 触发条件 | 为什么要物化 |
| --- | --- |
| 交付给外部客户(数据集导出) | 客户不能访问内部 bucket |
| 某 asset 频繁被访问 | 切小文件后读取延迟更低 |
| 原 MCAP 要归档 / 删除 | 解除对原文件的依赖 |

**实现**:

```python
# SDK 示例
asset.materialize(output_bucket="gs://grace-materialized/", 
                  kind="customer_delivery")

# 结果:
#   cf:file 新增:
#     file:customer_delivery.uri  gs://grace-materialized/.../a1b2c3.mcap
#     file:customer_delivery.kind materialized_segment
#     file:customer_delivery.producer  system:materializer
```

物化操作本身是一个 Dagster asset,可追溯。

---

## 4. MCAP 接入流(详细)

### 4.1 上传

```
Client (SDK / CLI / Web UI)
    │
    │  1. 调用 mcap-gateway: POST /v1/uploads
    ▼
mcap-gateway
    │
    │  2. 生成 GCS Resumable Upload Session URL
    │     + 返回 upload_id
    ▼
Client
    │
    │  3. 分片上传到 GCS(断点续传)
    ▼
GCS bucket: grace-raw-mcap/<tenant>/<date>/<uuid>.mcap
    │
    │  4. 上传完成,GCS 发 event
    ▼
Pub/Sub topic: grace-mcap-uploaded
    │
    ▼
Dagster Sensor: mcap_upload_sensor
    │
    │  5. 触发 Asset: mcap_ingest(uuid)
```

### 4.2 索引

```
Asset: mcap_ingest  (Python + mcap 库)
    │
    │  1. Range GET: 读 MCAP footer(最后 8 字节 → summary offset)
    │  2. Range GET: 读 summary 区(topic list, message index, statistics)
    │     注意:不读 data chunks,对 1GB 文件也只需 <1s
    │  3. 解析:
    │     - topics: ["/cam/front", "/lidar", ...]
    │     - start_time, end_time
    │     - total duration
    │     - chunk_count, message_count
    │  4. 写 Bigtable:
    │     cf:core.type = "mcap_file_indexed"
    │     cf:time.start_ns, end_ns
    │     cf:file.original.* (uri / size / sha256)
    │  5. 默认先创建 "整段 asset"(一个 MCAP → 一个 pending asset)
    │     调用 asset-service.Create,同步写 PG/Bigtable
    │  6. Outbox relay → Pub/Sub(grace-mcl):asset.created(见 ADR-007)
```

### 4.3 QA 切 segment

```
人工 QA(Web UI):
    │
    │  1. 打开整段 asset,嵌入 Foxglove 预览
    │  2. QA 员听到 / 看到两段有效:(12s-58s), (120s-150s)
    │  3. UI 通过 BFF 调用 asset-service(浏览器→BFF→gRPC,见 ADR-007):
    │     POST /v1/assets/<parent>/split
    │     body: [
    │       {start_ns: ..., end_ns: ..., tags: {...}},
    │       {start_ns: ..., end_ns: ..., tags: {...}}
    │     ]
    ▼
asset-service
    │
    │  4. 创建 2 个新 asset(virtual segment),共享 parent 的 mcap_uri
    │  5. 写 cf:lineage:
    │     new_asset.cf:lineage.up:parent = urn:grace:asset:<parent>
    │  6. 父 asset 状态变 "split"
    │  7. commit 成功 → Outbox relay → Pub/Sub(grace-mcl):
    │     asset.created × 2 + asset.split(见 ADR-007)
```

---

## 5. MCAP 流式读取(关键能力)

### 5.1 为什么关键

算法用户不能接受"下载 1GB MCAP 才能处理 1 分钟片段"。必须流式。

### 5.2 实现方案

**mcap-gateway 提供 HTTP 接口**:

```
GET /v1/assets/<id>/mcap?topics=/cam/front,/lidar&start=12.3&end=57.8

Response:
  Content-Type: application/x-mcap
  Transfer-Encoding: chunked

  ... streaming MCAP bytes(剪裁到指定 topic 和时间范围)
```

**服务端实现**:
- 用 Go 的 `foxglove/mcap` 库(支持**真·流式解压**)
- 按 Range GET 读 source MCAP 的相关 chunks
- 解压缩 → 过滤 topic → 过滤 time → 重新编码为新 MCAP 流返回
- 整个过程 **memory-bound 可控**,不会 OOM

> ⚠️ **不要用 Python 的 mcap 库做流式服务端**。Python 版 MCAP 库的 chunk 解压会一次性读到内存,1GB 文件直接 OOM。Gateway 必须 Go。

### 5.3 SDK 侧

```python
# grace-sdk
asset = grace.asset(id)

# 选项 A: stream 到本地文件(跑算法用)
asset.download_mcap(path="/tmp/a.mcap", topics=["/cam/front"])

# 选项 B: 直接 iter messages(内存流式)
for msg in asset.iter_messages(topics=["/cam/front"]):
    img = msg.decode_as(CompressedImage)
    run_sam2(img)

# 选项 C: 拿到 file-like 对象给第三方库
with asset.open_mcap(topics=["/lidar"]) as f:
    reader = foxglove.Reader(f)
    ...
```

---

## 6. 派生产物(File)规范

### 6.1 File kind 命名约定

约定格式:`<producer_type>_<name>_v<version>`

| 示例 kind | 含义 |
| --- | --- |
| `raw_mcap` | 原始 MCAP(不可变) |
| `sam2_mask` | SAM2 输出的 mask MCAP |
| `tracker_output` | 跟踪算法输出 |
| `human_annotation` | 人工标注 JSON |
| `ground_truth` | 标注后的 ground truth |
| `materialized_segment` | 物化切出的小 MCAP |
| `customer_delivery` | 客户交付包 |
| `embedding_pool` | 特征池(parquet) |

所有 kind 统一在 `schemas/fields.yaml` 的 `file_kinds:` 章节注册。

### 6.2 File 版本语义

同一个 kind **可以有多个版本**(算法迭代):

```
cf:file.sam2_v3.uri  = gs://.../sam2_v3.mcap
cf:file.sam2_v3.version = 3.1.2

cf:file.sam2_v4.uri  = gs://.../sam2_v4.mcap
cf:file.sam2_v4.version = 4.0.0
```

**不覆盖,并存**。下游可以选"我要 sam2 最新版" 或 "我要 sam2_v3"。

### 6.3 File 写入的 SDK 约定

```python
# 算法用户写法
asset = grace.asset(asset_id)

output_uri = run_my_algo(asset)  # 算法把输出写 GCS

asset.files.add(
    kind="my_algo",
    version="1.0.0",
    uri=output_uri,
    producer="algo:my_algo@1.0.0",
    run_id=context.dagster_run_id,
    extras={"loss": 0.23, "latency_ms": 1234}
)
# 底层:asset-service 写 Bigtable cf:file + cf:algo + cf:event
```

---

## 7. 大规模 MCAP 的 Gotcha 清单

| 坑 | 解决 |
| --- | --- |
| Python mcap lib 不支持 streaming | 服务端 Gateway 用 Go 实现 |
| ROS bag v1/v2 兼容 | 上传时做格式检查,非 MCAP 先转 |
| Topic schema 不一致 | 注册 topic schema,接入时校验 |
| 压缩算法不一(zstd/lz4) | mcap-gateway 全部支持 |
| Chunk 太大(> 500MB) | 警告,推荐客户端控制 chunk 大小 |
| Summary 缺失(旧写入器) | Fallback:扫全文件生成 summary 并写回 |
| GCS 读取带宽限制 | 用 Cloud CDN 或自建缓存 Gateway |
| 跨时区时间戳混乱 | 强制 nanoseconds since epoch (UTC) |
| topic rename | 在 cf:time 额外记录 topic aliases |

---

## 8. 与 Foxglove 生态对接

### 8.1 Foxglove App(预览 / QA)

```python
# 给 UI 生成一个签名 URL,让 Foxglove App 能直接播
signed_url = mcap_gateway.generate_signed_url(
    asset_id=asset_id, ttl=timedelta(hours=1))

foxglove_url = f"https://app.foxglove.dev/?ds=remote-file&url={urlencode(signed_url)}"
# UI 用 <iframe src="..."> 或新标签打开
```

### 8.2 Foxglove Data Platform(闭源,不采用)

> 详见 [ADR-003](adr/ADR-003-mcap-as-primary-format.md) 为什么只用 Foxglove 工具不用它的平台。

---

## 9. 下一步

- UI 嵌入 Foxglove → [05-ui-strategy.md](05-ui-strategy.md)
- 字段 / kind 注册表 → [../schemas/fields.yaml](../schemas/fields.yaml)
- ADR-002 Segment-centric → [adr/ADR-002-segment-centric-asset.md](adr/ADR-002-segment-centric-asset.md)
- ADR-003 MCAP 原生 → [adr/ADR-003-mcap-as-primary-format.md](adr/ADR-003-mcap-as-primary-format.md)
