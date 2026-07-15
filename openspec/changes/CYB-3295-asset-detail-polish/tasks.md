# Tasks — CYB-3295

## A2 空状态引导
- [x] TagsTab：`<Empty>` 文案补引导（可手动添加 / 等待上游算法打标）
- [x] EvalMetricsTab：`<Empty>` 文案补引导（运行评测算法后展示）
- [x] DeliveryHistoryTab：`<Empty>` 文案补引导（资产列表选中后创建交付）

## A3 概览标识符可复制
- [x] OverviewTab：`asset_id` 改 `Typography.Text copyable`
- [x] OverviewTab：`mcap_file_id` 改 `Typography.Text copyable`
- [x] OverviewTab：`segment_locator` 改 `Typography.Text copyable` + `ellipsis={{ tooltip }}`

## 验证
- [x] `npm run build`（Frontend）通过
- [ ] 部署 dev 后浏览器验证：空 tab 显示引导；概览标识符有复制按钮且点击可复制
