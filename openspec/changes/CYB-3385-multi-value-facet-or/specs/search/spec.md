# search Specification — CYB-3385 delta

## MODIFIED Requirements

### Requirement: Facet multi-value semantics

Assets 页面 facet 侧栏支持多选。同一 facet 内的多个已选值 SHALL 以 OR 语义解释;不同 facet(不同 field)之间 SHALL 以 AND 语义解释。

- 前端 `buildStructuredQueryWhere` 分组 chip 后:
  - 同 `(field, op)` 组 → 编译成 `{or: [pred_v1, pred_v2, ...]}`
  - 多组之间 → 编译成 `{and: [group1, group2, ...]}`
  - 单 chip → 直接 `{pred:...}`
- `eq` 与 `ne` 于同 field 上被视为不同分组:比如 `owner=alice AND owner!=charlie` 结果 = `(alice) AND (NOT charlie)`。
- 后端 IR `{or:...}` / `{and:...}` 结构不变,不新增 op。
