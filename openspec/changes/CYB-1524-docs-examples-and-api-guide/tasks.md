---
change-id: CYB-1524
slug: docs-examples-and-api-guide
---

# Tasks

## 1. Add example values to key schema properties

- [ ] Add `example` to enum-like fields: `lifecycle_state`, `asset_type`, `split_method`, `status`, `retention_tier`
- [ ] Add `example` to ID fields: `asset_id`, `mcap_file_id`, `logical_asset_id`
- [ ] Add `example` to timestamp/number fields: `start_timestamp_ns`, `end_timestamp_ns`, `duration_ms`

## 2. Publish api-guide.md to doc site

- [ ] Copy `docs/review/api-guide.md` → `docs-site/docs/reference/api-guide.md`
- [ ] Add api-guide to `docs-site/sidebars.ts`
- [ ] Verify sidebar renders correctly

## 3. Verify

- [ ] Validate OpenAPI YAML
