# 任务清单 — CYB-3209

## Phase 1：表格列宽优化

- [ ] **修改 AssetsResultsPane.tsx**
  - 调整 COLUMN_BUILDERS 中各列的 width 属性
  - asset_id: 120px (已有，保持)
  - owner: 150px
  - created_at/updated_at: 140px
  - asset_type: 100px
  - status/lifecycle_state: 90px
  - 其他: flex 或 120px
  - 文件：`Frontend/src/components/assets/AssetsResultsPane.tsx`

- [ ] **修改设计令牌**
  - 添加表格列宽命名常量
  - 文件：`Frontend/src/styles/design-tokens.css`

- [ ] **本地验证**
  - 打开 `/assets` 页面
  - 检查各列宽度是否符合信息价值分配
  - 在 1600px viewport 测试（应该有左右留白）

## Phase 2：常驻双栏（带鱼屏）

- [ ] **修改 AssetsPage.tsx**
  - 添加 `isUltraWide` 状态（`window.innerWidth > 1800px`）
  - 修改右侧预览窗容器样式：
    - 默认 width: 320px（展开）当 `isUltraWide=true`
    - 当 `isUltraWide=false` 时保持原来的 Drawer 模式
  - 确保预览窗 sticky 定位（top: 12px）
  - 文件：`Frontend/src/pages/AssetsPage.tsx`

- [ ] **修改 AssetQuickPreviewPane.tsx**
  - 添加响应式样式（如果需要）
  - 确保在分栏模式下的可滚动性
  - 文件：`Frontend/src/components/assets/AssetQuickPreviewPane.tsx`

- [ ] **本地验证**
  - 窗口宽度 > 1800px：预览窗自动显示在右侧
  - 点击表格行：预览内容实时更新
  - 窗口宽度 < 1800px：预览变回 Drawer 模式
  - 检查 sticky 滚动是否正常

## Phase 3：筛选区分层

- [ ] **修改 AssetsFacetSidebar.tsx**
  - 将筛选分成 3 层：
    1. 快速筛选（asset_type, status）— 常驻第一行
    2. 常用筛选（owner, retention_tier, created_at）— Collapse 组件
    3. 高级筛选（其他所有）— 按钮打开 Drawer
  - 使用 Ant Design Collapse 组件
  - 快速筛选用网格化（minmax(200px, 280px)）— 已有（CYB-3208）
  - 常用筛选在 Collapse 内也用网格化
  - 文件：`Frontend/src/components/assets/AssetsFacetSidebar.tsx`

- [ ] **本地验证**
  - 快速筛选在第一行可见
  - Collapse 默认展开，可折叠
  - 高级筛选按钮可点击打开 Drawer
  - 筛选项网格化布局保持（不超宽）

## 集成与测试

- [ ] **全页面集成测试**
  - 窗口 > 1800px：双栏布局，筛选 + 表格 + 预览 并排显示
  - 窗口 < 1800px：单栏布局，预览变 Drawer
  - 所有筛选和搜索功能正常

- [ ] **性能检查**
  - 没有布局抖动（layout shift）
  - 预览窗 sticky 滚动流畅

- [ ] **跨浏览器测试**
  - Chrome (最新)
  - Firefox
  - Safari

## 提交与 PR

- [ ] **Phase 1 提交**
  - Message: `feat(frontend): cyb-3209 assets table column width optimization (phase 1)`

- [ ] **Phase 2 提交**
  - Message: `feat(frontend): cyb-3209 assets page persistent split-view for ultrawide screens (phase 2)`

- [ ] **Phase 3 提交**
  - Message: `feat(frontend): cyb-3209 assets facet sidebar layered design (phase 3)`

- [ ] **创建 PR**
  - Base: `dev`
  - Title: `feat(frontend): cyb-3209 assets page comprehensive redesign (phases 1-3)`
  - 包含 3 个 phase 的说明和截图对比

## 时间估计

- Phase 1：20-30 分钟
- Phase 2：30-40 分钟
- Phase 3：40-50 分钟
- 集成测试：15-20 分钟
- **总计：120-150 分钟**
