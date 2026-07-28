# CYB-4323 tasks

## Frontend (纯视觉)

- [ ] `components/dashboard/DurationDistributionCard.tsx` — `HistogramBars`:柱体
      `maxWidth` + 居中;实心色改竖向渐变 `#60a5fa→#2563eb` + 顶部圆角;`minHeight`
      兜底不变。
- [ ] `components/dashboard/DurationDistributionCard.tsx` — 把 `总资产/总时长/均值/
      P50/P90` 的 `Statistic` 行从直方图**下方**移到**上方**(先数后图),加一条细分隔。
- [ ] `pages/DashboardPage.tsx` — `buildGrowthOption`:左轴 `axisLabel.color` 染蓝
      `#2563eb`、右轴染绿 `#16a34a`;刻度字提到 slate `#64748b`。

## Verification (Tier M — 前端多文件视觉改动,无路由/接口变更)

- [ ] `cd Frontend && npm run lint`
- [ ] `npm run test -- --run src/components/dashboard/DurationDistributionCard.test.tsx`
- [ ] `npm run build`

## Ship

- [ ] deploy frontend dev + Chrome DevTools MCP 目视核对:柱变窄有渐变、KPI 在图
      上方、双轴刻度分色、轴标签更清晰(deploy-verification.md §1.3)
- [ ] commit(`fix(frontend): ...` 全小写 subject)+ push(`CI_LOCAL_BASE=origin/dev`)
      + PR base=dev,填 PR 模板,带 CYB-4323 + change-id
- [ ] merge(user auth)后更新 Linear 状态 Done + commit hash
