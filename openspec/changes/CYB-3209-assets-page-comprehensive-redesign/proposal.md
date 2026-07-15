# CYB-3209：资产页面全面重构（表格列宽 + 常驻双栏 + 筛选分层）

## 背景

基于 CYB-3207/3208 的成功优化，资产发现页面仍存在三个主要 UX 问题：

1. **表格列宽平均分配** — 重要列（asset_id、owner）被压得极窄，次要信息占同样宽度
2. **右侧预览窗格隐藏** — 在带鱼屏上应常驻显示，目前需手动打开/关闭
3. **筛选区视觉混乱** — 所有筛选平铺，无层级感，导致高认知负荷

## 解决方案

### Phase 1：表格列宽优化（信息价值分配）

**原理：** Fitts 定律 + 信息层级

信息价值分布：
```
HIGH   → asset_id (30%)    | owner (20%)      | created_at (15%)
MEDIUM → asset_type (12%)  | status (10%)     | retention_tier (8%)
LOW    → env (5%)
```

实现方式：
- `asset_id` — 固定宽度（UUID 截断 + 复制，CYB-3208 已实现）
- `owner` — 相对宽度，可排序，常见筛选点
- 次列 — flex 或侧滑抽屉
- 总表格 — `max-width: 1600px`（CYB-3208 已设置）

### Phase 2：常驻双栏（带鱼屏优化）

**原理：** 充分利用超宽屏，减少鼠标移动距离

当 `window.innerWidth > 1800px` 时：
```
┌─────────────────────────────────────────────────────────┐
│ 【筛选条】【表格区(60%)】【预览窗(40%)】 │
└─────────────────────────────────────────────────────────┘
```

实现方式：
- 响应式布局检测（类似 CYB-3208 的 `isUltraWide`）
- 右侧预览 sticky 定位、自动展开
- 表格行点击时预览实时刷新

### Phase 3：筛选区分层设计

**原理：** Hick 定律（降低决策复杂度）+ Gestalt 接近律

三层结构：
```
[Layer 1] 快速筛选（Always-Visible）
  ├─ asset_type
  └─ status

[Layer 2] 常用筛选（Collapsible）
  ├─ owner
  ├─ retention_tier
  └─ created_at range

[Layer 3] 高级筛选（Button → Drawer）
  ├─ tags
  ├─ metadata
  └─ custom fields
```

实现方式：
- 快速筛选用网格化布局（CYB-3208 已做）
- 常用筛选用 Collapse 组件，默认展开
- 高级筛选用 Button 触发 Drawer

## 影响范围

| 文件 | 改动类型 | 复杂度 |
|------|---------|--------|
| `Frontend/src/components/assets/AssetsResultsPane.tsx` | 列宽分配 + flex | 中 |
| `Frontend/src/pages/AssetsPage.tsx` | 响应式布局 + 双栏 | 高 |
| `Frontend/src/components/assets/AssetsFacetSidebar.tsx` | 分层结构 + Collapse | 中 |
| `Frontend/src/styles/design-tokens.css` | 栅栏样式 | 低 |

## 验证方式

### Phase 1：列宽优化
- [ ] asset_id 列宽 ≥ 120px（UUID 截断可读）
- [ ] owner 列宽 ≥ 150px（邮箱完整可读）
- [ ] 表格总宽 = max-width 1600px（两侧留白）

### Phase 2：常驻双栏
- [ ] 窗口 > 1800px 时，预览窗自动显示（无需手动点击）
- [ ] 点击表格行，预览实时更新
- [ ] 窗口 < 1800px 时，预览 Drawer 模式恢复

### Phase 3：筛选分层
- [ ] 快速筛选第一行显示（asset_type, status）
- [ ] 常用筛选 Collapse 默认展开
- [ ] 高级筛选按钮可点击打开 Drawer

## 交付物

- 代码实现（三个 Phase 顺序提交）
- 组件文档（Storybook 更新，可选）
- 截图对比（Before/After）
