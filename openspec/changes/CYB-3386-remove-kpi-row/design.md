# Design — CYB-3386

## 唯一方案:直接移除

只保留 CYB-3382 里其他 4 项(DSL asset_type 识别 / facet 折叠 / asset_id subline / 预览默认态);第 3 项 KPI 卡整段回退。

**为什么不修复而是移除**
- 3 张卡各有独立 fix 成本(前端 key 修 + 后端加 algo_status facet + 后端加 expiring_30d range agg)
- 用户明确说:「概览界面其实有的」→ Dashboard 承担同类指标是**正确的功能归位**,资产列表页 KPI 是过度设计
- 移除比修复更少代码 / 更少维护面 / 更少歧义

**候选方案 B(不采用)**:修 KPI 2 + 3,保留总数,移除 KPI 4
- 需前端改 aggregation key + facet 请求;后端不动
- ~40 行改动 + 2-3 个测试
- **放弃原因**:用户明确说全部去掉,概览有了

**候选方案 C(不采用)**:全套修复 + 加后端 range aggregation
- 后端改造(ES aggregation) + 前端改造
- **放弃原因**:与"移除"意图相悖

## 影响面

**删除**
- `Frontend/src/components/assets/AssetsKpiRow.tsx`
- `Frontend/src/components/assets/AssetsKpiRow.test.tsx`(如存在)

**修改**
- `Frontend/src/pages/AssetsPage.tsx` — 移除引用

**不改**
- reducer / aggregations state / facet 请求
- 后端 / API / 无 spec 需要变更(KPI Row 不在 spec 里,是 CYB-3382 时提出的 UI 元素)

## Rollback

如果发现用户依赖某张 KPI,再单独实现那一张(比如「总数」在表格头「共 X 条」已有替身,不需要单独卡)。git revert 单 commit 即可。
