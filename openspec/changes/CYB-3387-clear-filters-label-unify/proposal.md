# CYB-3387 — Assets 页「重置」按钮统一为「清除全部」

## Why
Assets 页同时存在两个 label 不同的按钮,但**代码上是同一个 dispatch**:

| 位置 | Label | 触发 |
|---|---|---|
| chips row 顶部 | 「清除全部」 | `dispatch({type:"CLEAR_ALL_FILTERS"})` |
| 筛选侧栏 header (`AssetsPage.tsx:310` + `:478`) | 「重置」 | `dispatch({type:"CLEAR_ALL_FILTERS"})` + range draft reset |

两个按钮做的事一样,但 label 分裂让用户误以为:
- 「清除全部」= 只清 filter chips
- 「重置」= 更强(清 filter + sort + view mode + columns …)

实际 range draft reset 也是清 filter 相关的,与「清除全部」语义完全对齐。用户 UX 走查明确标注了这个歧义(见 CYB-3382 走查 5 项 followup)。

## What Changes
`AssetsPage.tsx` 两处 button 文案 `重置` → `清除全部`,dispatch 不变。

## Impact
- **用户**:两个入口 label 一致,不再让人产生"哪个更狠"的判断成本
- **无功能改动** — dispatch 完全不变,只是文案统一
- **无 backend / API 改动**
- 单测:无需新增(dispatch action 名称不变);若要严格,可加一条 button label assertion —— 但成本 > 收益,skip

## 验证
- Tier L:tsc + biome
- dev 回归:Chrome MCP 打开 /assets,勾几个 filter,确认侧栏 header 显示「清除全部」而非「重置」
