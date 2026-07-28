# CYB-4338: dashboard 时长分布卡迭代2(10 桶细拆 + ECharts + 移除配额池)

## Context

CYB-4323(#616 已合并)上线了「数据时长分布」卡。一轮实机 review 暴露三类问题,
本迭代一个 PR 统一处理(前端 + 后端):

1. **5 桶太粗** — `<1min / 1-10min / 10-30min / 30-60min / 60min+` 把语料压成
   "三个大方块",掩盖了真实形状。只读 SELECT 于 dev 实证(总和 18209 ✓):
   1-5min 有 5207 条尖峰、25-30min+30min 处约 7900 条尖峰,粗桶全看不见。
2. **flexbox 柱悬空** — `HistogramBars` 每列 `flex-start` 顶对齐,矮柱(<1min 727)
   整列挤顶部、底边不贴基线;还缺 Y 轴/网格线,千位分隔不统一(727 vs 7,208)。
3. **配额池冗余** — dashboard 顶部的 Koordinator 弹性配额池(ElasticQuota)用户不再需要。

**附带实证结论**:P90=30m1s **不是后端 bug**。4873 条 ≥30min 里 4872 条精确=30:00
(1,800,000ms) —— 数据在 30min 被某种上限钉死的尖峰,`percentile_cont` 算得正确。
"30min 数据截断"是数据质量问题,另记,不在本卡。

## Change

### 后端(API 契约变更)
- `models/dashboard.go` `DurationBucketOrder`:5 → 10 桶(方案A "前密后疏")
  `<1m/1-5m/5-10m/10-15m/15-20m/20-25m/25-30m/30-45m/45-60m/60m+`
  高频区(1-30min)5min 等距,长尾(30-60min)15min 等距,短标签防拥挤。
- `postgres/repos.go` bucket SQL CASE:5 → 10 分支,边界与 model 锁死。
- 契约同步(强制):`api/openapi.yaml`(summary/描述/桶枚举)、
  `docs/review/api-guide.md`(示例)。`scripts/api-guide-smoke.sh` 只 GET 查 200,天然兼容。
- 测试:handler/repo 的桶数与标签断言 5 → 10(usecase_test 不涉及桶,零改)。

### 前端(纯视觉,已本地 MCP 验证)
- flexbox `HistogramBars` → **ECharts 柱状图**:柱基线对齐(修悬空)、Y 轴+虚线网格、
  `barMaxWidth:56` 收紧、千位分隔统一。桶自适应 `data.buckets`,10 桶零改。
- KPI 行与图间加 **Divider** 解耦(修 KPI↔柱假关联)。
- 测试:mock `LazyECharts` 把 option 的桶标签/计数渲染成真 DOM,保留 findByText 断言。
- 移除 `ElasticQuotaPanel` + 相关 state/effect/import(`pipelineApi`/`PoolManager`)。

## Non-goals
- **P50/P90 桶高亮**:分类轴无法精确画竖线,折中的"高亮落入桶"作为快速 follow-up,不在本 PR。
- **30min 数据截断**:数据质量问题,另 issue。
- CYB-4294 批量查页仍用 5 桶(dashboard 要细节,单资产查保持粗桶,有意分叉)。
