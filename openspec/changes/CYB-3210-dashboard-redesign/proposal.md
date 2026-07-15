# CYB-3210：Dashboard 页面重构 + 移除视频预览菜单

## 任务 A：移除"视频预览"菜单项

**当前状态：**
导航栏中有"视频预览"菜单项，但根据产品规划不再需要。

**改动位置：**
- `Frontend/src/components/AppLayout.tsx`（line 43）— 删除菜单定义
- 同步删除相关的路由判断逻辑（line 78, 91）

**预期效果：**
导航栏中移除"视频预览"菜单，用户无法访问 `/preview` 路由。

---

## 任务 B：Dashboard 页面重构（CYB-3210）

**当前问题：**
从截图看，Dashboard 的主要痛点：

1. **信息密度不均** — KPI 卡片和图表混排，无清晰层级
2. **右侧空白浪费** — 超宽屏上没有充分利用空间
3. **响应式不足** — 带鱼屏上可能有布局问题
4. **卡片大小不一** — 视觉混乱

**改进方案（三个 Phase）：**

### Phase 1：KPI 卡片重构（信息价值分层）

**当前：** 所有 KPI 卡片大小相同，重要指标（资产总数、事件数）和次要指标（趋势）混在一起

**目标：** 按重要性分层显示

```
[L1] 关键KPI（2 列）— 资产总数、事件总数
[L2] 辅助KPI（3 列）— 其他统计数据
[L3] 趋势图表（满宽）— 时间序列数据
```

实现：
- KPI 卡片：`minHeight: 140px` → 更大的点击区域
- 卡片间距：使用统一的 16px gap
- 卡片网格：`repeat(auto-fit, minmax(240px, 1fr))`（类似 CYB-3209）

### Phase 2：图表布局优化（带鱼屏适配）

**当前：** 图表宽度不限制，在带鱼屏上可能超过 1600px 导致查看困难

**目标：** max-width 限制 + 常驻双栏

实现：
- 图表容器 `max-width: 1600px` （配合 CYB-3209）
- 大屏上（>1800px）：侧边栏常驻显示（筛选、说明等）

### Phase 3：加载状态和空状态优化

**当前：** PageLoading 可能过于简单，没有骨架屏

**目标：** 更好的加载体验

实现：
- 骨架屏（Skeleton）为各个卡片
- 渐进加载（优先显示 KPI，后加载图表）

---

## 影响范围

| 文件 | 改动 | 优先级 |
|------|------|--------|
| `Frontend/src/components/AppLayout.tsx` | 删除菜单项 | ⭐⭐⭐ |
| `Frontend/src/pages/DashboardPage.tsx` | Phase 1-3 优化 | ⭐⭐⭐ |
| `Frontend/src/styles/design-tokens.css` | 栅栏样式（可选） | ⭐ |

---

## 验证方式

**菜单移除：**
- ✓ 导航栏不显示"视频预览"
- ✓ 直接访问 `/preview` 显示 404 或重定向

**Dashboard 优化：**
- ✓ KPI 卡片分层显示（2 列关键 KPI）
- ✓ 图表区域有 max-width 限制（1600px）
- ✓ 带鱼屏模式下有二列布局
- ✓ 骨架屏加载状态

---

## 交付物

- 菜单移除（AppLayout.tsx）
- Dashboard Phase 1-3 优化（DashboardPage.tsx）
- OpenSpec 文档
- PR 到 dev
