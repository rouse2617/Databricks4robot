# CYB-3310 — assets 检索 facet 面板收敛 + 文案

## Why

巡检发现 `/assets` facet 面板可无限制同时展开 5 组,叠成横跨整屏、480px+ 的巨型面板,把数据列表挤出首屏(手段喧宾夺主于目的);另有两处小文案问题。均为纯前端体验打磨。

## What Changes(建议范围,checkpoint 请圈定)

### In scope(推荐:低风险高收益,纯前端)

1. **「全部收起」入口**(对应巡检 #1/#3):facet 面板加一个「全部收起」按钮,一键清空 `expandedGroups` → 列表立刻回到首屏。实现:reducer 加 `FACET_GROUPS_COLLAPSE_ALL`(或复用 `onToggleGroup` 遍历),`AssetsFacetSidebar` 加按钮 + `onCollapseAll` prop。
2. **面板整体限高内滚**(#1 补充):给 facet 面板容器 `max-height + overflow-y:auto`,展开多组时面板自身滚动,而不是把下方列表推走。
3. **占位符换人话**(#5):`AssetsFacetSidebar.tsx:791` 场景标签占位符「eq 操作在关键词模式下走 nested」→「按场景标签精确匹配(如 城市街道)」。
4. **空值文案**(#6):环境 facet 里 bucket key 为 `-` / 空 → 渲染为「(未标注)」而非突兀的「-」。

### Optional(收益较小,视 checkpoint 决定)

5. **列等高**(#2):5 组统一 `max-height` + 组内滚,消除"采集组超长、算法组大片留白"的失衡。

### Out of scope(明确不做 + 理由)

- **#4 即时 vs「应用」不一致**:range/date facet 用「应用」是合理的"批量输入后提交"模式(输入 min/max 过程中不该每次击键触发查询),**非 bug**,保留现状。
- **#7 快捷 chip ↔ facet 双入口**:已做联动(点 chip → facet 亮「N 已选」),低优,不动。

## Impact

- 纯前端;无后端 / migration / off-limits。
- 触及 `AssetsFacetSidebar.tsx`、`AssetsPage.tsx`、可能 `assetsDiscoveryReducer.ts`(收起 all 的 action)。
- 无行为回归:筛选逻辑不变,只改展开管理与文案。

## 验证

- 本地 lint / build(组件测试环境坏见 CYB-3316,靠 dev MCP 兜底)。
- dev Chrome DevTools MCP:展开多组后「全部收起」列表回首屏;占位符/空值文案更新;筛选功能不受影响;console 无 error。
