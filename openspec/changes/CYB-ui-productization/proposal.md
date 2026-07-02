# CYB-ui-productization — 全站 UI 产品化打磨

## Why

DataBrew 后台骨架完整，但中英文混用、状态语义不一致、加载/空态弱、布局宽度分散，理解成本偏高。需在不重做视觉的前提下，统一词汇、状态、三态与大数据列表体验。

## Scope

Frontend-only。不涉及 API / 后端变更。

## Phases

1. 产品词汇表 + 状态标签/颜色统一
2. Loading / Empty / Error 三态组件
3. 组件库展开空白修复 + 手工组件 latest 文案
4. 资产页卡片/紧凑表格视图切换
5. 算法矩阵 sticky 首列/表头 + hover 详情
6. 页面容器宽度策略（dashboard / form / table / matrix）

## Out of scope

- 全站 i18n 框架
- 设置页技术详情折叠（后续迭代）
- 概览页信息分组重组

## Approval

User: 「全面推进」in worktree — 2026-06-16
