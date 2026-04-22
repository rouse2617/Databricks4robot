# ADR-005:UI 采用"自研核心 + iframe 嵌入"策略

- **Status**: Accepted
- **Date**: 2026-04-22
- **Related**: [05-ui-strategy.md](../05-ui-strategy.md)

---

## Context

`data4cyber` Web UI 需要:
- 统一的搜索 / 资产详情 / Lineage(核心品牌)
- MCAP 预览播放(专业能力)
- 标注工作台(专业能力)
- Pipeline / Run 状态(Dagster 原生强)
- BI / 自助分析(Phase 2)

这些功能全部自研 = 浪费;全部用现成 = 品牌分散、体验割裂。

## Decision

**分层决策**:

| 功能 | 策略 |
| --- | --- |
| **资产核心页(搜索 / 详情 / Lineage / QA 队列 / Dashboard)** | **自研** React + Ant Design,借鉴 DataHub / OpenMetadata UI 模式 |
| **MCAP 播放器** | **嵌入**:Phase 0/1 用 Foxglove App 外链;Phase 2 评估 Lichtblick 自托管 |
| **标注工作台** | **嵌入**:Phase 1 起 Label Studio / CVAT iframe |
| **Pipeline UI** | **深链** Dagster UI(不 iframe,有 CSP 风险) |
| **BI / 报表** | **嵌入**(Phase 2):Superset iframe |
| **Lineage 图** | **自研**:React Flow |

**所有嵌入 URL 都封装成统一的"打开方式"组件**(`<OpenInExternalTool />`),便于未来切换实现。

## Alternatives Considered

### A. 完全自研(含 MCAP 播放器)

- 🔴 自研 MCAP 播放器需要 6+ 人月,重复造轮子
- 🔴 复制 Foxglove 的能力不现实(5 年积累)

### B. 完全嵌入(外部工具拼凑)

- 🔴 品牌分散,用户切多个系统
- 🔴 核心资产语义(MCAP segment)无处承载
- 🔴 没有 SSO 统一入口

### C. 深度集成 Foxglove 商业 Data Platform

- 🔴 费用高
- 🔴 数据需上传到 Foxglove cloud(或自托管,但授权复杂)
- 🔴 Vendor lock-in

## Consequences

### Positive
- ✅ 自研部分聚焦核心价值(资产管理)
- ✅ 嵌入部分借势成熟工具(MCAP / 标注 / BI)
- ✅ 节省大量重复开发
- ✅ 用户在核心页享有一致体验

### Negative
- ⚠️ iframe 存在 CSP / 跨域挑战
- ⚠️ 外链打开新标签略伤体验
- ⚠️ 依赖外部工具的稳定性(Foxglove App 政策)

### Mitigations
- Foxglove App 以**外链**方式使用(不 iframe),避免 CSP 问题
- 同时保留 **Lichtblick 自托管**方案作为兜底
- iframe 集成时通过 postMessage 做最小通信(如 asset_id 同步)

## Acceptance Criteria

- Web UI MVP 上线时,资产详情页有"在 Foxglove 中打开"按钮,点击后秒级加载
- 核心搜索页、详情页响应时间 < 300ms
- 后续接入 Label Studio / Superset 只需新增一个 `<OpenInExternalTool tool="label-studio" />` 组件

## References

- Foxglove App URL params: https://docs.foxglove.dev/docs/app/url-parameters
- React Flow: https://reactflow.dev
- Label Studio: https://labelstud.io
- UI 策略文档: [05-ui-strategy.md](../05-ui-strategy.md)
