# CYB-4323: dashboard 数据时长分布卡 UI 优化

## Context

CYB-4303 (#611) 上线了 dashboard「数据时长分布」卡
(`Frontend/src/components/dashboard/DurationDistributionCard.tsx`),替换了原
「事件分布 (30 天)」+「最新日事件类型分布」两张卡。功能正确,但一轮 UI review
暴露出几处视觉/信息动线问题(纯前端,不涉及后端接口或数据口径):

1. **大蓝块 / 视觉过载** — `HistogramBars` 每个桶 `flex: 1` + 满宽实心
   `--color-primary`。宽屏 + 桶数少 (5) 时,柱被拉成大面积实心深蓝矩形,厚重
   且缺乏精致感。
2. **信息动线倒置** — `总资产 / 总时长 / 均值 / P50 / P90` 这排 `Statistic`
   放在直方图**下方**。用户习惯「先看汇总数字,再看分布」,现在得先越过大蓝块
   才够到关键数字。
3. **双 Y 轴识别成本** — `DashboardPage.tsx` 的「资产增长」图 (`buildGrowthOption`)
   左轴 = 每日新增(蓝柱)、右轴 = 累计活跃(绿线),但两条轴刻度都是灰字,读者
   要额外思考哪条轴对哪个系列。
4. **轴标签对比度偏低** — 淡灰刻度字在普通分辨率下易读性一般。

## Change

**纯前端、纯视觉。** 不改任何接口、数据口径、组件 props、测试断言语义。

### `DurationDistributionCard.tsx` — `HistogramBars`

- 柱体加 `maxWidth`(如 72px)并在桶容器内居中,宽屏不再拉成满宽实心块。
- 实心 `--color-primary` 换成竖向线性渐变 `#60a5fa → #2563eb`(与「资产增长」
  柱同款),顶部圆角保留;`minHeight` 兜底逻辑不变。
- KPI `Statistic` 行从直方图**下方**移到**上方**(先数后图),并加一条细分隔。
  测试 (`DurationDistributionCard.test.tsx`) 只用 `findByText` 断言文本存在、
  不校验 DOM 顺序,移动不破坏测试。

### `DashboardPage.tsx` — `buildGrowthOption`

- 左轴(新增)`axisLabel.color` 染品牌蓝 `#2563eb`;右轴(累计)染绿 `#16a34a`,
  与对应柱/线系列色映射。
- 轴/刻度淡灰 `#e2e8f0`/默认灰提到可读的 slate(`#64748b`),仅提对比度,不改布局。

## Non-goals

- 07-27 突增柱的「点击下钻看当天资产列表」(要动路由/后端,单独排期)。
- 失败模式面板空状态(已有 `TableEmptyState` 兜底,无需改)。
- 直方图桶边界、数据口径、Segmented 行为 — 全部保持 CYB-4303/CYB-4294 现状。
- 引入图表库替换 flexbox 柱(保持与 CYB-4294 单资产视图同一套视觉语言)。
