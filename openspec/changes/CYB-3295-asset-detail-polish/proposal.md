# CYB-3295 — 资产详情：空状态引导 + 概览标识符可复制（A 组 2+3）

## Why
UX 走查 A 组第 2、3 项：

- **A2 空状态无引导**：多个 tab 的空态只有一句「暂无 X」，用户无法判断是「还没算完」还是「需要自己操作」。
- **A3 长标识符不可复制**：概览页的 `asset_id / mcap_file_id / segment_locator` 是需要经常拷贝去查日志/ES 的长字符串，但只用 `<code>` 展示，无法一键复制；`segment_locator` 过长时撑破布局。

（A1 跳转算法 tab、A4 tab↔URL 同步已在 CYB-3294 / PR #353 完成。FilesTab 的 URI 复制此前已有，不在本次范围。）

## What Changes
纯前端，无 API / schema 变更。

1. **A2 — 空状态引导文案**（三处 `<Empty>`）：
   - `TagsTab`：`暂无标签` → `暂无标签。可手动添加，或等待上游算法打标完成。`
   - `EvalMetricsTab`：`暂无评测结果或指标` → `暂无评测结果。运行评测算法后在此展示。`
   - `DeliveryHistoryTab`：`该资产尚未交付` → `该资产尚未交付。可在资产列表选中后「创建交付」。`

2. **A3 — 概览标识符可复制**（`OverviewTab`）：
   - `asset_id / mcap_file_id / segment_locator` 改用 antd `Typography.Text` 的 `copyable`；
   - `segment_locator` 追加 `ellipsis={{ tooltip }}` 防止长值撑破 `Descriptions`。

## Impact
- Affected specs: 无（纯 UI 文案 / 交互增强）
- Affected code:
  - `Frontend/src/components/asset-detail/OverviewTab.tsx`
  - `Frontend/src/components/asset-detail/TagsTab.tsx`
  - `Frontend/src/components/asset-detail/EvalMetricsTab.tsx`
  - `Frontend/src/components/asset-detail/DeliveryHistoryTab.tsx`
- 风险：低。文案可后续微调；`copyable`/`ellipsis` 为 antd 内置能力。
