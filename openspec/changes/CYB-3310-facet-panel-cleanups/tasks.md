# Tasks — CYB-3310(范围以 checkpoint 圈定为准)

## 1. 全部收起入口
- [ ] `assetsDiscoveryReducer.ts`:加 `FACET_GROUPS_COLLAPSE_ALL`(expandedGroups → [])或确认可复用现有 action
- [ ] `AssetsFacetSidebar.tsx`:新增 `onCollapseAll` prop + 「全部收起」按钮(仅当有展开组时可用),放在「筛选」标题行右侧「重置」附近
- [ ] `AssetsPage.tsx`:接线 `onCollapseAll` → dispatch

## 2. 面板限高内滚
- [ ] facet 面板容器加 `max-height`(如 `min(60vh, ...)`)+ `overflow-y:auto`,确认展开多组时列表仍在首屏

## 3. 占位符换人话
- [ ] `AssetsFacetSidebar.tsx:791`:`"eq 操作在关键词模式下走 nested"` → `"按场景标签精确匹配（如 城市街道）"`

## 4. 空值文案
- [ ] `CheckboxFacet` label 渲染:当 bucket 值为 `"-"` / 空串时显示「(未标注)」,count 照常;不影响 toggle 传给后端的原始值

## 5.(Optional)列等高
- [ ] 5 组统一 `max-height` + 组内滚,消除失衡

## 验证
- [ ] `cd Frontend && npm run lint && npm run build`
- [ ] dev MCP:展开 3+ 组 → 点「全部收起」→ 列表回首屏;场景标签占位符更新;环境空值显示「(未标注)」;勾选筛选仍即时生效、总数正确;console 无 error;截图存档
- [ ] PR → dev,填模板,Linear=CYB-3310
