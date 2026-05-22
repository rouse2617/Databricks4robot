# Tasks — CYB-1016

- [x] OpenSpec proposal + decisions (terminology: `is_current` / `revision`, not Linear `is_latest` / `version`)
- [x] ES builder: `logical_asset_id`, `revision`, `is_current`
- [x] ES mapping: `deploy/local/elasticsearch/init-index.sh`
- [x] Query IR scope: `include_history`; default current-only (PG + ES)
- [x] Handler: `?include_history=true` on run/validate
- [x] Promote: re-index prior revision via `asset_updated` event
- [x] Unit tests (builder, query filter, handler)
- [x] API contract: OpenAPI, api-guide, smoke script
- [ ] Dev: manual deploy + ES reindex + smoke (post-merge / deploy window)
