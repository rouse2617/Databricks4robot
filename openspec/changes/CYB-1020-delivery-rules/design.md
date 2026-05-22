# Design — CYB-1020 (tracer)

## Evaluator

- Load active `delivery_rules` for `customer_id` + global (`customer_id IS NULL`).
- Load `customers.exclude_tags` as implicit block rules (`rule_id=customer.exclude_tags`).
- Per asset: `assets` row + `asset_tags` (all sources); AND `where[]` predicates; optional `asset_types` filter.
- `enforce_mode=block` only blocks in this slice; `warn`/`tag_only` stored but ignored at commit.

## query_dsl (v1)

```json
{ "where": [{ "field": "tag.compliance.pii", "op": "eq", "value": "true" }], "asset_types": ["clip"] }
```

Supported fields: `tag.<key>`, `asset_type`, `lifecycle_state`. Ops: `eq`, `ne`, `in`, `exists`, `not_exists`, `contains`, comparisons on strings.

## Error shape

HTTP 422 `DELIVERY_RULE_FAILED` with `violations: [{ asset_id, rule_id, rule_name, reason }]`.
