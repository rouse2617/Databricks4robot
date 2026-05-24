# CYB-1125 tasks

## OpenSpec

- [x] Create proposal and spec deltas for UI QA blockers.
- [x] Record user approval to continue runtime edits.

## Implementation

- [x] Reproduce or unit-test the settings reindex modal cancel behavior.
- [x] Fix `/settings` unsupported admin endpoint handling so deployed pages do not surface 404 errors when capabilities are absent.
- [x] Reproduce or unit-test asset detail delivery-history pagination behavior.
- [x] Fix delivery history to use bounded asset-scoped data and settle loading state.
- [x] Fix low-risk request burst NITs only where local seams are clear.
- [x] Fix search sync status mode so running Outbox ES subscriber is reported to the UI.
- [x] Translate asset query fallback warnings into actionable Chinese copy.
- [x] Clear both committed asset query and visible search draft when returning to default results.
- [x] Surface delivery creation backend failures inline and preserve user input.
- [x] Relabel asset detail `delivery_count` as completed deliveries.

## Verification

- [x] Run targeted frontend tests for changed files.
- [ ] Run `cd Frontend && npm run lint && npm run test -- --run && npm run build`.
  - Targeted tests passed: `npm --prefix Frontend run test -- SettingsPage DeliveryHistoryTab EventsPage --run`.
  - Build passed: `npm --prefix Frontend run build`.
  - Backend passed: `cd backend && go test ./...`.
  - Full frontend lint/test still fail on pre-existing unrelated files (`RunIdLink`, `TagsTab`, `OverviewTab`, `QuickFiltersRow`, `AssetQuickPreviewPane`, `ColumnsConfigPopover`, `assetsDiscoveryReducer`).
- [x] Deploy frontend dev with SHA image tag before commit.
- [x] Verify `/settings` and `/assets/:id` delivery history via Chrome DevTools MCP on deployed dev.
  - Blocked: Chrome DevTools MCP tools are not available in this Claude Code session. See `decisions.md`.
  - Superseded in Codex session with Chrome DevTools MCP available: verified deployed frontend `ved7886c-cyb1125-uiqa2` on Cloud Run dev.
  - `/settings`: sidebar version shows `ved7886c-cyb1125-uiqa2`; search index status shows `搜索索引：Outbox 自动同步`.
  - `/assets/1nSlczDA`: summary label shows `已完成交付: 0`; Delivery History tab still lists cancelled delivery `8fc2d012-3762-403c-af3b-aa1f697e4c46`, so completed-count wording no longer conflicts with historical cancelled records.
- [x] Record deploy revision, image tag, screenshots/console-network evidence.
  - Backend: `cyber-databrew-backend-dev-00167-sf9`, image `cyber-databrew-backend:ed7886c-cyb1125`, URL `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`.
  - Frontend: `cyber-databrew-frontend-dev-00189-svx`, image `cyber-databrew-frontend:ed7886c-cyb1125`, URL `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app`.
  - Browser screenshot/console evidence blocked by missing Chrome DevTools MCP.
- [x] Run targeted tests for the post-deploy QA fixes.
  - Passed: changed-file Biome check for touched frontend files.
  - Passed: `npm --prefix Frontend run test -- CreateDeliveryModal AssetPreviewHero ResultsEmptyState --run`.
  - Passed: `npm --prefix Frontend run test -- src/lib/assets/assetsDiscoveryReducer.test.ts --run -t CLEAR_ALL_FILTERS`.
  - Passed: `npm --prefix Frontend run build`.
  - Passed: `cd backend && go test ./...`.
  - Full `npm --prefix Frontend run lint` still fails on pre-existing unrelated files (`RunIdLink`, `TagsTab`, `VersionControl`, `VersionProvenanceTab`, `AssetDetailPage`, etc.).
  - Full `npm --prefix Frontend run test -- --run` still fails on pre-existing unrelated tests (`OverviewTab`, `QuickFiltersRow`, `AssetQuickPreviewPane`, `ColumnsConfigPopover`, `assetsDiscoveryReducer` group-toggle case).
- [x] Redeploy backend/frontend dev for post-deploy QA fixes.
  - Required again by user instruction: “你一定要部署哈，然后用dev 的网址来测试”.
  - Blocked until gcloud user credentials are refreshed: local backend image was built as `cyber-databrew-backend:ed7886c-cyb1125-uiqa2`, but Artifact Registry push failed because gcloud auth requires interactive reauthentication.
  - Local frontend Cloud Run images were also built: base `cyber-databrew-frontend:dev-latest`, deploy image `cyber-databrew-frontend:ed7886c-cyb1125-uiqa2`.
  - Superseded after user refreshed GCP auth: deployed backend revision `cyber-databrew-backend-dev-00171-28z`, image `us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:ed7886c-cyb1125-uiqa2`.
  - Deployed frontend revision `cyber-databrew-frontend-dev-00191-rf7`, image `us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-frontend:ed7886c-cyb1125-uiqa2`.
  - Backend dev smoke passed: `/api/v1/search/sync-status` returned `search_index_mode:"outbox_es_subscriber"`, `elasticsearch_ok:true`, `outbox_es_subscriber_enabled:true`.
- [x] Verify fixed flows via Chrome DevTools MCP on the dev URL.
  - Cloud Run dev URL: `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app`.
  - Passed local MCP (`http://127.0.0.1:5174` against dev backend): `/assets` no-result search shows Chinese fallback warning, "放宽筛选" clears URL/list and visible search input.
  - Passed local MCP: `/deliveries` invalid manual asset id keeps modal open, preserves asset/customer input, shows field-specific Chinese error and backend request ID.
  - Passed dev MCP: `/assets` no-result search shows `搜索已降级为数据库查询`, "放宽筛选" clears URL back to `/assets?preview_layout_version=2`, restores list, and clears visible search input.
  - Passed dev MCP: `/deliveries` invalid manual asset id keeps modal open, preserves `not-real-asset-id` and `ux-customer-ywjfye`, shows field-specific Chinese error, and displays backend request ID.
  - Passed dev MCP: `/assets/1nSlczDA` summary label shows `已完成交付`; Delivery History tab still lists the cancelled delivery.
  - Console review: one expected 400 from the intentional invalid delivery submit; one pre-existing browser issue from the Ant Design pagination jump input lacking `id/name`.
