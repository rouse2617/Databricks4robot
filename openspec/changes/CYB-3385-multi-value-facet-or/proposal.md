# CYB-3385 — Assets facet 多选统一为 OR 语义

## Why
用户 UX 走查发现 facet 多选逻辑违背直觉:
- **asset_type = segment + action(两个都选)→ 列表显示 0 条**(dev 复现:`total=0`)
- 但用户显然是想"看这两种类型的资产,任一即可"(OR)
- `algo_status` 同 bug:选 `failed + succeeded` → 0 条

Live 复现(dev,`/api/v1/queries/run`):
```
where: {and:[{asset_type=segment},{asset_type=action}]}  → total 0
where: {or:[ {asset_type=segment},{asset_type=action}]}  → total 164
```

Root cause(前端 `Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts:52`):
```ts
const predicates = activeFilters.map(chip => ({pred: {...}}));
return {and: predicates};   // ← 所有 chip 一律 AND
```

FACET_TOGGLE(`assetsDiscoveryReducer.ts`)每选一个值创建独立 chip;编译器一视同仁 AND 起来。一个 asset 一个 `asset_type`,`type='segment' AND type='action'` 永远空集。

## What Changes
`buildStructuredQueryWhere` 分组编译:

- 按 `(field, op)` 分组 chips
- 同组多 chip → `{or: [pred_v1, pred_v2, ...]}`(inter-value OR)
- 组间 → `{and: [group1, group2, ...]}`(inter-field AND)
- 单 chip 场景不变

无后端改动、无新 op、无新 API 表面。UI 层面 FACET_TOGGLE / ActiveFilterChipsRow 不需要动 —— chip 仍是每值一个,只是编译时合并语义。

## Impact
- **面向用户**:多选 asset_type / algo_status / owner / env / lifecycle 任一 facet 时返回 OR 结果,与直觉一致。
- **既有行为**:单值 filter 不变;不同 field 之间仍 AND(比如 asset_type=segment + env=warehouse 依然是"两个都满足")。
- **saved views / URL hydration**:chip 集合不变,只是编译方式改;既有 saved view 在新代码下自动获得 OR 语义,视为"隐性升级"。
- **无需后端更改**、无 API 迁移、无 schema 变化。

## 验证
- 单测:`useAssetsDiscoveryReducer.test.ts` 补 3 用例:同 field 双值 OR、跨 field 与同 field 混合、单 chip 不变
- Tier L:前端 `pnpm biome check` + `pnpm test` 全绿
- dev 回归:Chrome MCP 选 segment+action 两个 asset_type → 列表返回 164 条(而非 0)
