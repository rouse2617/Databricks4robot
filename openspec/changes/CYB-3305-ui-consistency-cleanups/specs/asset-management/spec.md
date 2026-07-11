# asset-management spec delta — CYB-3305

## MODIFIED — 资产类型 facet

### Given/When/Then

- **Given** 用户打开 `/assets`,后端 `queries/run` 响应的 facets 里 `asset_type_agg` bucket 包含 `raw_mcap`、`action`、`frame` 三类之一
- **When** 前端渲染「资产类型」facet 组
- **Then** 该类型 chip 必须出现在 facet 侧栏且带 count(`raw_mcap (N)` 等);硬编码列表 `["segment","clip","frame_set","derived_asset"]` 仍作为空 counts 时的兜底

### Given/When/Then(单一真相源)

- **Given** 资产类型选项在前端有两个消费点(`AssetsFacetSidebar`、`AddFilterPopover`)
- **When** 修改类型集合
- **Then** 只允许改一处(sidebar 导出常量或 `lib/assets/assetTypes.ts`);`AddFilterPopover` 必须 `import`,不得重复硬编码

## MODIFIED — 资产标签来源标注

- **Given** 资产详情 `/assets/:id?tab=tags` 展示 `tags_detailed[]`,每条含 `source_type ∈ {human, algo, system}`
- **When** 前端渲染每个标签的来源 badge
- **Then** badge 必须与 `source_type` 一致(不再一律 `system`);badge 颜色可作视觉区分,但**不允许**丢弃 `source_type` 值

## MODIFIED — MCAP 文件大小列

- **Given** `/mcap-files` 列表包含已汇总(.mp4)条目,该条目未记录 `size_bytes`
- **When** 前端渲染「大小」列
- **Then** 展示「未记录」而非「—」;`size_bytes > 0` 时展示格式化字节
