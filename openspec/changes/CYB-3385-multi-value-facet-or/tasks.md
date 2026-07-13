# Tasks — CYB-3385

## 代码
- [ ] `useAssetsDiscoveryReducer.ts` — `buildStructuredQueryWhere` 按 `(field, op)` 分组;同组 OR、跨组 AND
- [ ] 保持单 chip 场景返回原来的 `{pred:...}` 形状(不引入无谓的 or 包裹)

## 测试
- [ ] `useAssetsDiscoveryReducer.test.ts` 或新建 pure test — 补:
  - 单 chip → 原样 `{pred:...}`
  - 同 field 双值(`asset_type=segment`, `asset_type=action`)→ `{or: [...]}`
  - 跨 field(`asset_type=segment`, `env=warehouse`)→ `{and: [pred, pred]}`
  - 混合(2 个 asset_type + 1 个 env)→ `{and: [{or:[a,b]}, env]}`
  - eq + ne 分开分组(`owner=alice`, `owner:ne=charlie`)→ `{and: [alice, ne charlie]}`

## Tier L(worktree 内)
- [ ] `pnpm biome check --write`(如需 fmt)
- [ ] `pnpm test` reducer 相关全绿

## PR
- [ ] commit + push branch(worktree 已在此 branch,新起 branch `fix/CYB-3385-multi-value-facet-or` 好清晰)
- [ ] PR → base dev

## dev 回归
- [ ] Chrome MCP:asset_type 勾 segment+action → 列表返回 164
- [ ] env 勾 warehouse+lab → 列表返回并集
- [ ] 混合:asset_type=segment + env=warehouse → 交集
