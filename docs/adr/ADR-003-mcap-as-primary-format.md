# ADR-003:MCAP 作为唯一原生数据格式

- **Status**: Accepted
- **Date**: 2026-04-22
- **Related**: [04-mcap-and-segment.md](../04-mcap-and-segment.md)

---

## Context

用户明确:**算法都是围绕 MCAP 格式开展的**。
MCAP 是 Foxglove 推出的开源容器格式,**已成为机器人 / 自动驾驶领域的事实标准**(Tesla、Wayve、BMW、Roboto AI、我们自己的算法都基于它)。

特性:
- 多 topic 容器,原生支持 ROS1/ROS2/Protobuf/JSON/CBOR
- 带 chunk 索引、footer summary,支持 Range read
- 自描述,可反射 schema
- 工具链成熟(Foxglove Studio/App、Lichtblick、mcap CLI、Python/Go/Rust SDK)

## Decision

**`data4cyber` 仅以 MCAP 作为一等公民数据格式。** 其他格式(rosbag/Parquet/mp4/jpg)都是:
- 派生产物(作为 File 附属于 Asset),或
- Phase 0 不支持,后续通过 importer 转成 MCAP

## Alternatives Considered

| 方案 | 缺点 |
| --- | --- |
| 多格式平等支持 | 工具链爆炸;测试矩阵大;MCAP 已够用 |
| 只支持 rosbag | 格式老旧;工具链不如 MCAP;未来迁移成本 |
| 自研统一格式 | 不会被业界采纳;工具链从零建 |
| 只支持 Parquet(数据湖风) | 机器人实时 topic 语义丢失;不适配 Foxglove |

## Consequences

### Positive
- ✅ 工具链统一(Foxglove/Lichtblick 零成本播放)
- ✅ SDK 统一(grace-sdk 只需对接一个格式)
- ✅ 未来 AI/ML 社区互通(Ray Data 原生支持 MCAP)
- ✅ 与 ROS 生态无缝

### Negative
- ⚠️ MCAP Python 库不支持真流式,chunk 一次性解压 → mcap-gateway 必须用 Go
- ⚠️ 其他格式用户需要先转 MCAP 才能用
- ⚠️ 与 Foxglove 生态绑定较深,需关注其开源政策

### Mitigations
- `mcap-gateway`(Go)承担流式解压责任,Python 算法用户不感知
- 提供转换工具 / Dagster asset,Phase 1 支持 rosbag → mcap
- 准备 Lichtblick(MPL 2.0 fork)作为 Foxglove App 兜底

## Acceptance Criteria

- 所有上传文件统一是 MCAP;非 MCAP 被拒绝或走转换通道
- 算法用户 SDK 只需要 `asset.iter_messages()` 一个 API
- Web UI MCAP 预览可用(Foxglove 外链或 iframe)

## References

- MCAP: https://mcap.dev
- Foxglove: https://foxglove.dev
- Lichtblick: https://github.com/lichtblick-suite/lichtblick
- Ray Data MCAP support: https://docs.ray.io/en/latest/data/api/doc/ray.data.read_mcap.html
