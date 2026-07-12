# Tasks — CYB-3309

## 前端删除(精确清单)

### `Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts`
- [ ] 删函数 `assetMatchesAlgoStatusFilter(asset, statuses)`(约 235–250)
- [ ] 删 `activeAlgoStatusFilters` useMemo(约 312–318)
- [ ] results-fetch effect:`const items = data.items ?? []`,移除 `hasAlgoStatusFallback` 判断与 `.filter(assetMatchesAlgoStatusFilter)` 分支(约 340–346)
- [ ] 移除 algo_status 黄条 push;`const warnings = [...(data.warnings ?? [])]` 后不再追加(约 364–369)
- [ ] 从该 effect 依赖数组移除 `activeAlgoStatusFilters`(约 417)
- [ ] 确认 `total`/`cachedAuthoritativeTotal`(facets count)逻辑保持不变

### `Frontend/src/pages/AssetsPage.tsx`
- [ ] 删 `hasAlgoStatusWarning` 定义(约 149）
- [ ] Alert `message` 三元删 `hasAlgoStatusWarning` 分支,保留 `hasSearchFallbackWarning` 与默认「部分筛选可能未完全生效」(约 372–378)
- [ ] 确认 `resultWarnings` / `visibleResultWarnings` / `translateResultWarning` 通用逻辑不动

## 自查
- [ ] `grep -rn "assetMatchesAlgoStatusFilter\|activeAlgoStatusFilters\|hasAlgoStatusFallback\|hasAlgoStatusWarning\|仅在当前页" Frontend/src` 应只剩 0 处(或仅无关命中)
- [ ] 无悬空引用 / 未用 import

## 本地验证(Tier M)
- [ ] `cd Frontend && npm run lint`(对齐 baseline,不新增 error)
- [ ] `npm run test -- --run`(对齐 baseline;algo_status 相关的 URL/chip/facet 测试应仍绿)
- [ ] `npm run build`

## Deploy dev + Chrome DevTools MCP 验收
- [ ] merge 到 dev 自动触发 `deploy-dev.yml`
- [ ] `/assets` 点「算法失败」:列表 + total + 分页三者一致(dev 无 failed → 空态「没有匹配的资产」,**无黄条**)
- [ ] 抓 `/queries/run`:list 与 facets 都带 `algo_status:eq:failed`,response total 一致
- [ ] 换一个 dev 上有数据的算法状态(如 `blocked`/`pending`)验证:筛后列表数 = total = 分页,可正常翻页
- [ ] console 无新增 error;截图存 `openspec/changes/CYB-3309-*/deploy-verify-*.png`

## PR
- [ ] `gh pr create --base dev`,填模板,Linear=CYB-3309
