# asset-management spec delta — CYB-3382

## MODIFIED — Assets 搜索栏 DSL:asset_type 结构化识别

- **Given** 用户在 `/assets` 搜索栏输入 `asset_type:segment`
- **When** 搜索栏解析该输入
- **Then** `asset_type=segment` 必须显示为**结构化 chip**(与 `algo_status`、`lifecycle_state` 等一致),**不得**回退为 `_fulltext ~ asset_type:segment` fulltext 查询
- **Given** 用户在同一输入行给出 `asset_type:segment algo_status:ok`
- **Then** 两个 token 都必须被识别为结构化条件,组合执行

## MODIFIED — Assets 页顶部 facet 面板折叠

- **Given** 用户**首次**访问 `/assets`,`localStorage['assets:facet-collapsed']` 不存在
- **Then** 页面必须默认**折叠** facet 面板,顶部显示筛选摘要行 + 展开按钮
- **Given** 用户点击展开按钮
- **Then** facet 面板恢复完整显示,`localStorage['assets:facet-collapsed']` 写入 `"false"`
- **Given** 用户再次访问 `/assets`
- **Then** 必须按 `localStorage` 记忆的折叠状态恢复(展开则展开、折叠则折叠)

## ADDED — Assets 页 KPI 概览卡片

- **Given** 用户访问 `/assets`
- **Then** 页面顶部必须显示 4 个 KPI card:总数 / ready:created 比例 / 近 30 天算法失败数 / 近 30 天即将过期数
- **Given** 某个 metric 数据不可用(接口返回空 facet 或 metric 不适用)
- **Then** 对应 card 显示 `—`,其他 card 正常显示,列表 rendering 不受影响

## MODIFIED — 资产行 asset_id 显示层次

- **Given** 用户在表格视图或卡片视图看到资产行
- **Then** 每行必须显示 asset_id 主体(保留 8 位可拷贝、URL 友好的原状),并在其旁边显示副标题 `{owner} · {asset_type} · {short_date}`
- **Given** 副标题字段(如 `owner`)为空
- **Then** 副标题该位置显示 `—`,其他字段照常显示

## ADDED — Quick preview 侧栏默认态与最近查看

- **Given** 用户访问 `/assets`,尚未选中任何行/卡片
- **Then** Quick preview 侧栏必须显示:
  1. 顶部 empty state 提示「点击行/卡片查看预览」
  2. 如果 `localStorage['assets:recent-viewed']` 有条目,下方显示「最近查看」列表(最多 5 项),每项点击后触发 preview 该 id
- **Given** 用户选中某个资产(id `X`)
- **Then** `localStorage['assets:recent-viewed']` 必须以 LRU 方式记录 `X`(去重、置顶、最多 5 项);preview pane 显示 `X` 的详情
