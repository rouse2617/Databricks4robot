# Tasks — CYB-3386

- [ ] `AssetsPage.tsx` 移除 `<AssetsKpiRow>` + `import AssetsKpiRow` 引用(可能两处)
- [ ] 删 `Frontend/src/components/assets/AssetsKpiRow.tsx`
- [ ] 删 `Frontend/src/components/assets/AssetsKpiRow.test.tsx`(若存在)
- [ ] grep 确认无其他文件仍 import 该组件
- [ ] Tier L:`tsc --noEmit`(与 pre-existing errors 比对)+ `biome check`
- [ ] commit + push + PR base=dev
- [ ] dev deploy 后 Chrome MCP 确认页面顶部无 KPI Row + 总数在表头仍可见
