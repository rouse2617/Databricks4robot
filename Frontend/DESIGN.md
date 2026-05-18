# DESIGN.md — Cyber Databrew Frontend

本文件按 [awesome-design-md](https://github.com/VoltAgent/awesome-design-md) 的 9 节模板，沉淀
**Linear + PostHog/Sentry inspired data platform** 视觉规格，作为新页面 / 组件的设计参照。

设计方向不是复制任一品牌皮肤，而是明确分工：

- **Linear-inspired product shell**：全局导航、页面骨架、表格、弹窗、表单保持克制、清晰、低噪音。
- **PostHog/Sentry-inspired observability panels**：状态矩阵、事件流、查询检查器、异常提示、分析图表强调可诊断性。
- **Cyber Databrew tokens**：继续使用当前 Trust Blue、深色侧栏、高密度表格与 Ant Design 组件体系。

实现层的 source of truth 仍然是：

- `Frontend/src/index.css`（CSS 变量 + Ant Design 覆写）
- `Frontend/src/main.tsx`（`ConfigProvider` 主题）

如果本文件与上述实现冲突，**以代码为准**，并提交 PR 同步本文件。

---

## 1. Visual Theme & Atmosphere

- **品类**：SaaS Data-Dense Dashboard（机器人 MCAP 视频数据资产管理 / 算法监控 / 交付）。
- **产品骨架**：参考 Linear 的极简工程工具感。导航、按钮、Tab、表格边界要轻，但层级要明确。
- **数据面板**：参考 PostHog/Sentry 的可观测性产品感。事件、失败、同步状态、查询解释要让用户快速定位问题。
- **气质**：可信、安静、信息密度优先；不要营销站感、不要拟物感、不要重渐变。
- **信息密度**：8/10。表格行高 40px、字号 13px，避免无用留白，但保证可读对比度。
- **动效强度**：4/10。仅用 `200ms ease` 的 hover / focus 过渡；不做滚动动画、不做 Hero 视频。
- **明暗策略**：浅色为主（`#F8FAFC`），**侧栏深色**（`#0F172A`）形成功能分区；当前不提供整页 dark mode。

## 2. Color Palette & Roles

| 角色 | Token | 值 | 用途 |
|------|-------|----|------|
| Primary | `--color-primary` | `#2563EB` | 主按钮、选中态、链接、Tab 激活 |
| Primary Hover | `--color-primary-hover` | `#1D4ED8` | 主按钮 hover |
| Secondary | `--color-secondary` | `#3B82F6` | 次级强调（图标、辅助色块） |
| CTA | `--color-cta` | `#F97316` | 关键行动（创建交付 / 导出 等高优 CTA），**全局每屏限 1 个** |
| Background | `--color-bg` | `#F8FAFC` | 页面底色 |
| Sidebar | — | `#0F172A` | 侧栏背景（Slate 900） |
| Text | `--color-text` | `#1E293B` | 正文 |
| Text Secondary | `--color-text-secondary` | `#64748B` | 次要文字 / 表头 / 标签描述 |
| Border | `--color-border` | `#E2E8F0` | 卡片、表格、分割线 |
| Success | `--color-success` | `#16A34A` | `ok` / `approved` |
| Error | `--color-error` | `#DC2626` | `failed` / `rejected` |
| Warning | `--color-warning` | `#D97706` | `running` |
| Info | `--color-info` | `#2563EB` | 信息提示（与 Primary 同色） |

**KPI / 多卡片场景**：默认所有 KPI 卡使用 Primary 单色 + 不同 icon 区分，**不再**为不同 KPI 引入紫 / 青 / 粉等 categorical accent；如未来确实需要分类色板，必须先在本文件登记角色和场景。

**业务状态色映射**（全站一致，不允许在某个页面自定义）：

| 状态 | 颜色 | 备注 |
|------|------|------|
| `ok` / `approved` | Success 绿 | |
| `running` | Warning 黄 | |
| `failed` / `rejected` | Error 红 | |
| `pending` | 中性灰 `#94A3B8` | |
| `blocked` | 中性灰 + 锁图标 | 不要用红色（避免与 failed 混淆） |

**实现来源**（不要再在页面里就地写映射表）：

- 算法状态颜色 → `src/lib/statusColor.ts` 的 `algoStatusTagColor(status)`
- 算法状态提取 → `src/lib/statusColor.ts` 的 `extractAlgoStatuses(asset.algo_results)`
- 资产生命周期颜色 → `src/lib/assetPresentation.ts` 的 `getAssetStateColor(asset)`（也可从 `statusColor.ts` 以 `assetLifecycleTagColor` 名称导入）
- API 错误展示 → `src/lib/apiError.ts` 的 `describeApiError(err)` + `src/components/common/PageError.tsx`

## 3. Typography Rules

- 正文：`Inter`，回退 `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`。
- 数据 / 代码：`Fira Code`，回退 `SF Mono, Cascadia Code, monospace`。**所有 ID、hash、数值列必须等宽**（用 `.font-mono` 或 `font-variant-numeric: tabular-nums`）。

| 用途 | 字号 | Weight | 备注 |
|------|------|--------|------|
| 页面标题 | 20–24px | 600 | 每页只有 1 个 H1 |
| 区块标题 / Card head | 14px | 600 | 与 `.ant-card-head` 对齐 |
| 正文 | 13–14px | 400 | |
| 表格内容 | 13px | 400 | `--table-font-size` |
| 表头 | 12px | 600 | 大写 + 0.03em letter-spacing，颜色用 Text Secondary |
| Tag | 11px | 400 | `.ant-tag` 默认 |
| Statistic 数值 | 大号 Inter | 700 | `tabular-nums` |

中文使用系统字体回退即可，不引入额外中文字体（保持包体）。

## 4. Component Stylings & State Expression

> 优先使用 Ant Design 5 组件，不要为已有组件自绘替代。在 `index.css` 里覆写以贴合本设计。

- **Button**
  - Primary：`#2563EB` 底，`6px` 圆角，hover 转 `#1D4ED8`。
  - Default：白底 + `--color-border`，`6px` 圆角。
  - CTA（关键操作，例如「创建交付」）：使用 `--color-cta` 橙色 Primary 变体，每屏限 1 个。
  - Danger：用 Ant 默认 `danger` 即可（红色），不要用橙色表达「危险」。
- **Card**：`8px` 圆角、`1px solid --color-border`、`elevation-1`，hover 升至 `elevation-2`；`Card-head` 高 44px。
- **Table**：紧凑型；表头浅灰底 `#F8FAFC`、大写小字；行 hover `#F1F5F9`；分割线 `#F1F5F9`。
- **Tabs**：默认 13px / 500，激活态使用 Primary 下划线（Ant 默认）。
- **Tag**：`4px` 圆角、11px、`6px` 横向 padding。状态 Tag 颜色严格按第 2 节映射。
- **Segmented**：浅灰底 `#F1F5F9`，激活态白底 + `elevation-1`。
- **Sidebar 菜单项**：13px、40px 行高、`6px` 圆角、激活态 `rgba(37,99,235,0.2)` 底色 + 白字；不用 Ant 默认右侧蓝条。
- **Scrollbar**：`6px` 宽，thumb `#CBD5E1`，hover `#94A3B8`。
- **Timeline / Event feed**：用于 Asset events、交付状态流转、算法生命周期。每条事件必须包含时间、actor/source、状态变化摘要；失败事件优先展示错误码和 request ID。
- **Inspector panel**：用于 Query inspector、MCAP inspector、调试详情。采用单列 Card + 分组标题 + monospace 原始值；不要把 JSON / SQL / topic 列表塞进普通段落。
- **Alert / Empty / Error**：成功提示轻量，失败提示明确可恢复动作；空状态说明“为什么为空”和“下一步能做什么”。错误状态优先复用 `ErrorBoundary` 和现有空态组件。
- **Status matrix**：用于算法矩阵、同步状态、处理进度。颜色仅表达状态，文本 / icon 表达含义；hover 显示最近更新时间、失败原因、重试入口。

## 5. Layout Principles

- **基本布局**：左侧深色 Sidebar (`220px`) + 右侧内容区。**不使用顶部 Header**（`--header-height: 0px`）。
- **内容内边距**：`24px` (`--content-padding`)；卡片内 `16px` (`--card-padding`)；网格间距 `16px` (`--grid-gap`)。
- **栅格**：使用 Ant `Row/Col` 24 栅格 或 Tailwind `grid-cols-*`。Dashboard KPI 卡通常 4 列；分析图常 2 列；详情多 Tab 单列。
- **空白**：宁可减少空白以提升数据密度，**不要刻意拉大留白**做营销站观感。
- **页面层级**：Page → 1 个 H1 + 可选筛选栏 → 1–N 个 Card / Table。**不允许嵌套 Card 超过 1 层**。

### Page Templates

- **Dashboard / Overview**：顶部 4 个 KPI 卡；中部 2 列图表（状态分布 + 趋势）；底部失败告警和最近更新事件。参考 PostHog/Sentry 的“先异常、再趋势、再明细”信息顺序。
- **Assets table**：页面标题 + 搜索 / Saved query / 快捷筛选；左侧 facet 面板；右侧 Table；底部或顶部批量操作栏。参考 Linear 的紧凑列表与明确 hover 操作。
- **Asset detail**：顶部 preview hero + 关键元信息；下方 Tabs（概览 / 算法 / 标签 / 交付 / 文件 / 事件）。每个 Tab 只解决一个工作流，不混放多个表格。
- **Timeline / Events**：左侧筛选，右侧时间线；失败和状态变更置顶或可快速过滤。每条事件要能跳转到相关 asset / delivery / request。
- **Analytics**：上方查询条件和时间范围；中部图表卡片；下方原始明细表。图表只负责解释趋势，明细表负责审计。
- **Settings / Registry**：偏 Linear 风格，表单分区清楚、解释文案简短，不做重装饰。

## 6. Depth & Elevation

| 层级 | Token | 用途 |
|------|-------|------|
| 0 | `--elevation-0` | 行内、内联控件 |
| 1 | `--elevation-1` | 卡片默认、Segmented 激活态 |
| 2 | `--elevation-2` | 卡片 hover、Popover |
| 3 | `--elevation-3` | Modal、Drawer 顶层 |

不要叠加多层阴影模拟拟物深度；同一视图内最多出现 2 个层级。

## 7. Do's and Don'ts

**Do**

- 用既有 CSS 变量 / Ant token，不要在组件里硬编码 `#2563EB` 这类颜色字面量。
- 表格里 ID、hash、数值列加 `.font-mono` 与 `tabular-nums`，避免列对不齐。
- 状态用第 2 节映射的颜色 + 文字标签，不只用颜色（无障碍）。
- 危险操作（删除、撤销交付）走 Modal 二次确认，文案明确「不可逆」。
- 空状态、错误状态用现有 `ResultsEmptyState` / `ErrorBoundary` 组件，不要每页造一份。
- 页面级 loading 用 `components/common/PageLoading`；页面级错误用 `components/common/PageError`（自动展示后端 `code` / `request_id`）。
- 表格 / 列表里的 Asset ID 链接用 `components/common/AssetIdLink`，不要再就地写 `<button class="link-like-button">id.slice(0,8)…</button>`。

**Don't**

- 不要重渐变背景、毛玻璃 hero、`box-shadow` 大于 elevation-3。
- 不要在一屏里用超过 1 个 CTA（橙色）按钮。
- 不要为「好看」放弃信息密度，例如把表格行高加到 56px。
- 不要把 `failed` 用橙色，`blocked` 用红色 —— 严格遵守状态色映射。
- 不要绕开 Ant `ConfigProvider` 主题用 `<style>` 注入新风格。

## 8. Responsive Behavior

当前面向桌面端（≥1280px）使用，但需保证 ≥1024px 不破版。

- **断点**（与 Tailwind 一致）：`sm 640 / md 768 / lg 1024 / xl 1280 / 2xl 1536`。
- **<1024px**：Sidebar 折叠为图标（`Layout.Sider collapsible`）；KPI 卡 4 列降为 2 列。
- **触控**：交互目标 ≥32×32px（Ant 默认满足），表格操作列保留至少 1 个图标按钮可点击。
- **不做移动端专属布局**：当前业务面向内部数据团队，移动端只需「能看不破版」。

## 9. Data Visualization & Agent Prompt Guide

### Data Visualization Rules

- **KPI card**：数字优先，标题 12px secondary，主值使用 tabular nums；变化率用小号 Tag，不要用大面积背景色。
- **趋势图**：用于资产增长、处理耗时、失败率。默认蓝色主线；异常或阈值用红 / 黄辅助线；坐标轴和网格线保持低对比。
- **分布图**：算法状态、交付状态优先用条形图或 donut；分类超过 6 个时改用水平条形图，避免扇区难读。
- **矩阵图**：算法处理矩阵采用状态色块 + tooltip；色块大小稳定，hover 展示版本、时间、错误摘要。
- **事件流**：按时间倒序；失败、重试、人工操作要有清晰 icon / Tag；不要只显示裸 JSON。
- **Query / inspector 可视化**：保留原始 Query IR / request payload 的折叠视图，同时提供人类可读摘要。
- **图表颜色**：优先使用 Primary / Success / Warning / Error / neutral；不要引入新的彩虹色板，除非分类数据确实需要。

**新页面提示词模板（建议复用）**

> 按 `Frontend/DESIGN.md` 实现 `<页面名>`：
> - 使用 Ant Design 5 组件 + 现有 CSS 变量；不要新增颜色字面量。
> - 状态色严格按 §2 业务状态色映射；CTA 每屏 ≤1 个。
> - 表格使用紧凑型（行高 40px、字号 13px）；ID / 数值列加 `.font-mono`。
> - 布局遵循 §5：Sidebar + 内容区 24px padding，无顶部 Header；优先套用对应 Page Template。
> - 数据面板遵循 §9：KPI / 趋势 / 分布 / 矩阵 / 事件流各自职责清楚。
> - 优先复用 `ResultsEmptyState`、`ErrorBoundary`、`AppLayout` 等已有组件。

**快速色板**（写代码时直接 copy）

```text
Primary  #2563EB    Hover #1D4ED8    CTA    #F97316
BG       #F8FAFC    Sidebar #0F172A  Border #E2E8F0
Text     #1E293B    Sub     #64748B
Success  #16A34A    Warn    #D97706  Error  #DC2626
```

**评审 checklist**

- [ ] 颜色 / 间距 / 圆角全部走 CSS 变量或 Ant token
- [ ] 状态色与 §2 一致
- [ ] 每屏 CTA ≤ 1
- [ ] 表格密度符合 §3 / §4
- [ ] 页面结构匹配 §5 的 Page Template
- [ ] 图表 / 事件 / 矩阵符合 §9 的数据表达规则
- [ ] 不引入新字体、新阴影层级
- [ ] 不破 1024px 宽度
