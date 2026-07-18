# Tasks — CYB-3386

> **状态**: 归档 — 实体已在 PR #378 (commit 1c36e570) 合入 dev。本文件 tasks.md 未及时勾选造成 openspec 幽灵漂移,现补齐。

- [x] `AssetsPage.tsx` 移除 `<AssetsKpiRow>` + `import AssetsKpiRow` 引用(可能两处)
- [x] 删 `Frontend/src/components/assets/AssetsKpiRow.tsx`
- [x] 删 `Frontend/src/components/assets/AssetsKpiRow.test.tsx`(若存在)
- [x] grep 确认无其他文件仍 import 该组件
- [x] Tier L:`tsc --noEmit`(与 pre-existing errors 比对)+ `biome check`
- [x] commit + push + PR base=dev
- [x] dev deploy 后 Chrome MCP 确认页面顶部无 KPI Row + 总数在表头仍可见
