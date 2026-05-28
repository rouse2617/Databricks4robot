# Tasks — CYB-1254

## Context files

- `TASK.md` — component registry acceptance criteria
- `Frontend/src/components/pipeline/ComponentManager.tsx` — existing embedded component management
- `Frontend/src/api/pipelineComponentApi.ts` — existing component API client
- `Frontend/src/pages/PipelinePage.tsx` — existing component mapping and canvas consumption
- `Frontend/src/components/AppLayout.tsx` — sidebar integration
- `Frontend/src/App.tsx` — route registration
- `backend/internal/handlers/pipeline_component/handler.go` — existing backend handlers
- `backend/internal/usecase/pipeline_component/usecase.go` — component usecase behavior
- `backend/internal/repository/pipeline_component_repository.go` — persistence interface
- `backend/routes/routes.go` — API route registration
- `/Users/rick/src/reference/visual-argo-workflows/src/components/modals/template/` — modal form reference

## Existing Pipeline Integration Steps

- [ ] 1. 安装 @xyflow/react 到 Frontend
- [ ] 2. 创建 `api/pipelineApi.ts`（集成 pipeline API 客户端）
- [ ] 3. 创建 `components/pipeline/` 组件目录结构
  - [ ] 3a. PipelineNode.tsx（自定义 React Flow node）
  - [ ] 3b. ComponentPalette.tsx（拖拽面板）
  - [ ] 3c. NodeConfigPanel.tsx（节点配置）
  - [ ] 3d. ComponentManager.tsx（组件注册管理）
  - [ ] 3e. DeployPanel.tsx（部署列表）
- [ ] 4. 创建 `PipelinePage.tsx`（主页面，整合 3 个 tab）
- [ ] 5. 创建 `styles/pipeline.css`（canvas 样式）
- [ ] 6. 修改 `App.tsx`（添加 /pipeline 路由）
- [ ] 7. 修改 `AppLayout.tsx`（内部导航，非外部链接）
- [ ] 8. 验证：`npm run build` 通过
- [ ] 9. 前端 dev deploy → MCP 验证
- [ ] 10. PR

## Component Registry Implementation

- [x] [backend] Verify existing component persistence implementation and route behavior for list/create/update/delete.
- [x] [backend] Add or alias `GET/POST/PUT/DELETE /api/v1/pipeline-components` while preserving existing `/api/v1/components` compatibility if still used by `/pipeline`.
- [x] [backend] Validate required fields (`name`, `image`, `type`) and normalize optional `command`, `args`, `env`, and `description`.
- [x] [backend] Add/update handler tests for list/create/update/delete and one invalid request path.
- [x] [Frontend] Update `pipelineComponentApi` request/response types to match the OpenAPI shape and route path.
- [x] [Frontend] Create `Frontend/src/pages/ComponentListPage.tsx` with search, Ant Design `Table`, loading/error states, and CRUD actions.
- [x] [Frontend] Build create/edit modal with Ant Design `Form`, required `name`/`image`, type select (`container`, `script`, `resource`, `suspend`), command/args arrays, env key-value pairs, and description.
- [x] [Frontend] Add delete `Popconfirm` and refresh list after mutations.
- [x] [Frontend] Add `/components` route in `App.tsx`.
- [x] [Frontend] Add `组件管理` sidebar item near `流水线` and `流水线运行`.
- [x] [Frontend] Ensure existing `/pipeline` canvas component loading continues to work after API client changes.

## API contract sync

See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — document `/api/v1/pipeline-components` CRUD paths and schemas
- [x] `docs/review/api-guide.md` — curl examples for list/create/update/delete and validation error
- [x] `sdk/src/cyber_databrew_sdk/` — add/update public REST resource client if SDK exposes component registry
- [x] `sdk/tests/unit/` — unit tests if SDK client is added/changed
- [x] `scripts/api-guide-smoke.sh` or `scripts/smoke-pipeline-components-dev.sh` — happy path + error path smoke
- [x] `openspec/changes/CYB-1254-pipeline-integration/specs/pipeline-component-registry/spec.md` — behavior delta
- [x] [Frontend] `Frontend/src/api/pipelineComponentApi.ts` — typed client aligned with OpenAPI

## Local verification

- [x] `cd backend && go build ./cmd/server`
- [x] `cd backend && go test ./internal/handlers/pipeline_component/`
- [x] `cd Frontend && npm run lint`
- [x] Run broader Tier L checks if API contract or route changes require it per `docs/agents/AI-RULES.md`

## Deploy verification (before commit)

- [ ] Build backend and frontend images locally with git SHA tags.
- [ ] Push SHA and `cloudrun-dev-latest` tags.
- [ ] Deploy backend dev and frontend dev with SHA-tagged images.
- [ ] Record image tags, Cloud Run revisions, and dev URLs.
- [ ] Backend smoke against dev: create/list/update/delete a test component and assert validation failure for missing required fields.
- [ ] Chrome DevTools MCP on dev: open `/components`, search, open create modal, validate required fields, create/edit/delete a test component, and verify no console errors.
- [ ] User confirms deployed dev behavior is OK before `git add` / `git commit`.

## Backend major bugfix follow-up

- [x] Fix pipeline handler test nil pointer panic and verify `go test ./internal/handlers/pipeline/`.
- [x] Add CSRF hardening for cookie auth and pipeline mutating API calls.
- [x] Make pipeline/backfill migration deltas idempotent and add migration apply tracking.
- [x] Verify final backend build and full test suite.
