# CYB-3297 — 搜索/筛选正确性一揽子（分阶段）

父议题 CYB-3297。分阶段各自一个 PR，逐个落地。本 OpenSpec 覆盖整体，PR 逐阶段推进。

## Phase C（本 PR）— builder 回填 mcap.* 子字段
### Why
ES mapping 声明了 `mcap.{vendor_id,device_id,camera_model,scene_id,location_id,environment_id,task_id,data_source,file_duration_ms}`，registry/UI 也广告为可筛/可 facet，但 `builder.go` 从不写入 → dev 实测 `vendor_agg=0 / scene_agg=0`（229 资产），"采集"组筛选恒空。源数据其实已在 `mcap_files` 列（dev 实测 device_id/camera_model/vendor_id/data_source 均有值），`McapFileRepo.Get` 也已 SELECT+scan 全部列。纯粹是 builder 漏写。

### What Changes
`backend/internal/searchindex/builder.go`：在已有 `mcap` 对象里，当对应字段非空时补写 `vendor_id/device_id/camera_model/data_source/location_id/scene_id/environment_id/task_id` 与 `file_duration_ms`（>0 时）。纯 additive，不改任何现有字段。

### Impact
- 代码：`builder.go`（+ `builder_test.go` 断言）
- 无 schema 变更（ES mapping 已有；PG 列已有）。
- **需一次 reindex 回填存量 229 个**（新增/更新的资产由 subscriber 自动带上）。⚠️ `/admin/search/reindex` 会先清空索引再灌，有短暂空窗——dev 上可接受，验证时择机跑。
- 风险：极低（只增不改）。最坏情况某字段源为空 → 不写，行为同今。

## 后续阶段（各自 PR，本 PR 不含）
- **A**：tag.* 三拼法统一（filter 路径规范化 + 前端统一）。
- **B**：env 可发现（facet 白名单 + 前端聚合动态选项）。
- **D**：raw_mcap 实时入索引（mcap handler 事件发射；⚠️ 不动 outbox 内部）。
