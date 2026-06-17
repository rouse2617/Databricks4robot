# Tasks — CYB-2100 Config Management

## Context files
- `Frontend/src/pages/RegistryCenterPage.tsx` # current registry hub layout
- `Frontend/src/components/AppLayout.tsx` # nav label and selected-state behavior
- `Frontend/src/App.tsx` # route wiring for `/registry`
- `Frontend/src/api/registry.ts` # existing read-only registry clients
- `Frontend/src/pages/ComponentManager.tsx` # component edit surface and mount contract
- `Frontend/src/features/pipeline-designer/` # deploy-time component/config selection surface
- `Frontend/src/pages/PipelinePage.tsx` # pipeline authoring surface
- `Frontend/src/components/pipeline/*` # node config UX
- `backend/internal/handlers/registry/handler.go` # read-only registry endpoints
- `backend/routes/routes.go` # route registration
- `backend/internal/models` # config identity / snapshot types
- `backend/internal/postgres` # persistence layer
- `api/openapi.yaml` # public contract sync target
- `docs/review/api-guide.md` # public API examples
- `openspec/changes/CYB-1824-component-releases/design.md` # adjacent component/release separation
- `docs/agents/deploy-verification.md` # route/UI verification baseline

## Implementation
- [x] [Frontend] Split the `/registry` page into two clear areas: immutable reference registries and a standalone user-owned config library.
- [x] [Frontend] Add config list/detail/search UX with filters for name, description, tags, owner, and ownership scope.
- [x] [Frontend] Add config CRUD interaction UX: create config, view detail, edit metadata, deprecate config, and create version.
- [x] [Frontend] Add config version management UX showing current version, version count, and immutable version history.
- [x] [Frontend] Update the page copy and empty states so the registry hub explains that configs are user-owned files, not component children.
- [ ] [Frontend] Add deploy-time config picker UX that only shows the current user's own/shared configs and attaches the selection with one click.
- [ ] [Frontend] Add a component mount-path editor so the component can declare where the selected config file should be mounted in the container.
- [x] [backend] Add config identity, version, file payload metadata, and snapshot models for standalone configuration resources.
- [x] [backend] Add config repository/usecase/handler support for list, detail, create, update, create-version, and deprecate operations.
- [x] [backend] Add ownership-scoped config lookup for deploy-time selection.
- [ ] [backend] Add pipeline-node config reference and immutable snapshot persistence.
- [ ] [backend] Add delete/deprecate protection for configs that are still referenced by saved pipelines.
- [ ] [Frontend] Update pipeline authoring so nodes select `configId` instead of editing low-level config metadata inline.
- [x] [sdk] Expose config client methods if the config resource is public REST surface.
- [x] [sdk] Add unit tests for the new config client methods if the SDK surface changes.

## API Contract Sync
- [x] [backend] Update `api/openapi.yaml` for the new config resource and any pipeline-node config fields.
- [ ] [backend] Update `api/openapi.yaml` for ownership-scoped deploy-time config selection and mount-path contract fields.
- [x] [backend] Update `docs/review/api-guide.md` with success and error examples for config CRUD or lookup.
- [x] [backend] Add or update `scripts/api-guide-smoke.sh` or a dedicated smoke script for happy path + error path.
- [x] [sdk] Wire the new config client into `sdk/src/cyber_databrew_sdk/`.
- [x] [sdk] Add or update `sdk/tests/unit/` coverage for the new config client.

## Verification
- [x] [backend] Run targeted unit tests for the new config repository/usecase/handler path.
- [ ] [Frontend] Run registry-page and config-version tests that cover the new config library flow.
- [x] [all] Run Tier L checks before PR: backend tests, frontend build, and SDK tests if the SDK surface changed.

## Deploy verification
- [x] [deploy] Deploy the updated frontend to dev if the `/registry` route changes.
- [ ] [deploy] Verify `/registry` in Chrome DevTools MCP with screenshots and console check.
- [x] [deploy] Smoke the config API on dev with one success case and one failure case.

### Deploy record — CYB-2100
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:e128910-cyb2100-dirty-cb-20260617211338` | `cyber-databrew-backend-dev-00893-24h` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:e128910-cyb2100-dirty-frontend-cb-20260617210523` | `cyber-databrew-frontend-dev-00376-qcx` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |

### Deploy verification notes
- Backend `/readyz`: healthy with PostgreSQL healthy.
- Config CRUD smoke: list `200`, missing content `400`, create v1, get v1 content, create v2, update metadata, deprecate all passed.
- Frontend `/registry`: HTTP `200`; frontend proxy `/api/v1/pipeline-configs`: HTTP `200`.
- Chrome DevTools MCP browser verification remains blocked; see `decisions.md`.
