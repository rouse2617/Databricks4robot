# Decisions — CYB-1020

## 2026-05-22 — User approved implementation

User: 「CYB-1020 开始做吧」— proceed with tracer after OpenSpec artifacts.

## Scope cuts (tracer)

- Only `enforce_mode=block` enforced at `POST /deliveries`; `warn` / `tag_only` stored for later.
- No `POST /delivery_items` (route not implemented).
- SDK deferred; OpenAPI + api-guide + smoke only.
- Evaluator runs in-process on PG asset + `asset_tags` (not full `/queries/run` ES path).
