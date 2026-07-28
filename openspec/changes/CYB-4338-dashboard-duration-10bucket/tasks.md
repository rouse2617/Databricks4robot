# CYB-4338 tasks

## 后端(10 桶细拆 + 契约)
- [x] `models/dashboard.go` — `DurationBucketOrder` 改 10 桶(方案A边界 + 短标签)
- [x] `postgres/repos.go` — bucket SQL CASE 改 10 分支;注释更正
- [x] `api/openapi.yaml` — summary/描述/桶枚举/示例 → 10 桶
- [x] `docs/review/api-guide.md` — 示例 → 10 桶
- [x] `scripts/api-guide-smoke.sh` — 只 GET 查 200,天然兼容(无需改)
- [x] 后端测试 — handler_test / repos_test 桶数&标签断言 5 → 10(2 处 repos_test)

## 前端(ECharts + 移除配额池,已本地验证)
- [x] `DurationDistributionCard.tsx` — flexbox → ECharts(基线/Y轴/网格/千位分隔/收紧柱宽)
- [x] `DurationDistributionCard.tsx` — KPI 行下加 Divider 解耦
- [x] `DurationDistributionCard.test.tsx` — mock LazyECharts 保留 findByText 断言
- [x] `DashboardPage.tsx` — 移除 ElasticQuotaPanel + state/effect/import
- [x] `DashboardPage.tsx` — (CYB-4323 承接)双 Y 轴分色 + X 轴对比度

## 验证
- [x] 后端 `go vet + build + go test ./...` → 0 FAIL
- [x] 前端 biome(改动文件)干净 / vitest 4/4 / `npm run build` ✓
- [x] 前端本地 Chrome DevTools MCP:配额池消失、KPI+Divider、ECharts 渲染、无 console error

## Ship
- [ ] 部署验证:经 owner 指示跳过合并前 dev 部署,顺延合并后 GHA(见 decisions.md)
- [ ] commit(全小写 subject)+ push(`CI_LOCAL_BASE=origin/dev`)+ PR base=dev
- [ ] merge 后更新 Linear Done + hash

## Follow-up(不在本 PR)
- [ ] P50/P90 桶高亮(ECharts markLine/换色);快速项,owner 要则补
- [ ] 30min 数据截断数据质量问题 → 另 issue
