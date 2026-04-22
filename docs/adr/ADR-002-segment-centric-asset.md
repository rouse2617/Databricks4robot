# ADR-002:资产最小单元 = Segment(视频有效片段)

- **Status**: Accepted
- **Date**: 2026-04-22
- **Related**: [04-mcap-and-segment.md](../04-mcap-and-segment.md)

---

## Context

用户澄清:**一个 MCAP 文件不是都是有效的**。QA 员会从一个 MCAP 中标出若干**有效区间** `[(t1_start, t1_end), (t2_start, t2_end), ...]`。
每个有效区间是"一段"数据,后续会被单独打 tag、派生产物、交付给客户。

如果把"整个 MCAP"作为资产,会出现:
- 一个 MCAP 需要多套 tag(不同区间不同场景)
- 派生产物(如 sam2_v3)实际只针对某个区间有效,挂在 MCAP 上语义不清
- 客户交付时需要切分,资产 ID 与交付对象不对齐

## Decision

**资产(Asset)的最小粒度是 "Segment"(视频有效片段)**,定义为:
- 指向某个 MCAP 文件(URI + sha256)
- 有明确的 `(start_ns, end_ns)` 时间区间
- 有独立的 `asset_id`(UUID)作为**平台一等公民主键**
- 一个 MCAP → N 个 Asset(segment)

同时采用 **"虚拟 segment"** 作为默认(**零拷贝,只存时间区间引用**),仅在必要场景(交付、热点、归档)才**物化为独立 MCAP**。

## Alternatives Considered

| 方案 | 缺点 |
| --- | --- |
| 整个 MCAP 作为 asset | 多场景混合,tag 语义混乱,下游派生难挂载 |
| 按 topic 切 asset | 破坏了 MCAP 多 topic 联合语义,下游重建复杂 |
| 按固定时间窗口切(如 10s 一段) | 不符合 QA 真实边界,浪费存储 |
| 强制物化切文件 | 存储翻倍;原始文件不可删;编辑标记变低效 |

## Consequences

### Positive
- ✅ 资产语义清晰,一 tag、一派生、一交付都与 asset_id 对齐
- ✅ 同一个 MCAP 可以被 QA 成多个 asset,操作几乎零延迟
- ✅ 默认虚拟 segment,存储无翻倍
- ✅ 与 Foxglove 的 time-range 语义天然契合(URL 参数 `start=` / `end=`)

### Negative
- ⚠️ 虚拟 segment 依赖源 MCAP 不可删,生命周期管理更复杂
- ⚠️ 跨 chunk 边界读取需要服务端正确处理
- ⚠️ 必须提供强"物化"命令(client 交付场景)

### Mitigations
- `cf:lineage.up:mcap` 明确记录依赖关系;删源 MCAP 前扫下游 asset
- `mcap-gateway`(Go)原生支持跨 chunk streaming
- SDK 提供 `asset.materialize()` 一键物化

## Acceptance Criteria

- 一个 MCAP 上传后,QA 标出 3 个 segment,平台能毫秒级创建 3 个 asset
- 3 个 asset 各自读 MCAP streaming 不互相干扰
- 交付 API 能一键把指定 asset 列表物化+打包

## References

- Foxglove URL params: https://docs.foxglove.dev/docs/app/url-parameters
- MCAP spec: https://mcap.dev/spec
