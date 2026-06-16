# Tasks — CYB-1824 Component Releases

## Context files
- `backend/internal/models/pipeline_component.go`
- `backend/migrations/040_pipeline_components.sql`
- `backend/internal/postgres/pipeline_component_repo.go`
- `backend/internal/usecase/pipeline_component/usecase.go`
- `backend/internal/handlers/pipeline_component/handler.go`
- `backend/internal/transpiler/pipeline.go`
- `Frontend/src/api/pipelineComponentApi.ts`
- `Frontend/src/pages/ComponentManager.tsx`
- `Frontend/src/pages/PipelinePage.tsx`
- `Frontend/src/components/pipeline/types.ts`
- `Frontend/src/lib/pipelineContract.ts`
- `Frontend/src/pages/pipeline/pipelinePageHelpers.ts`

## Implementation
- [x] [backend] 新增 `PipelineComponentRelease`、runtime snapshot、validation status、technical metadata model。
- [x] [backend] 新增 `pipeline_component_releases` migration，并约束 component/release identity 唯一。
- [x] [backend] 新增 repository 方法：upsert、list、filter、fetch component releases。
- [x] [backend] 新增 Phase 1 validation：task identity、entrypoint、现有 ports 或默认 ports、resource hints、generated metadata、digest presence、selectable status。
- [x] [backend] 新增 release ingest/sync usecase，能消费现有 task-config/build/digest 元数据。
- [x] [backend] 将 release ingest 升级为正式 CI manifest contract，支持 batch `source + items`，并保留旧 `{items}` 兼容。
- [x] [backend] 支持 `sourceRefType`，区分 Git tag 线上版本、commit 测试版本、branch/PR 版本，并在未传时自动推导。
- [x] [backend] 为每个镜像版本生成 8 位 `imageUid`，优先由 image digest 稳定生成。
- [x] [backend] 新增 CI ingest token 入口，限制 CI token 只用于 release sync，同时保留管理员普通鉴权。
- [x] [backend] 扩展 release list 搜索，支持按 commit、build id、image tag/digest、runtime image 搜索。
- [ ] [backend] 新增 test-release usecase：用户选择 task + commit 后，按 task metadata 解析 Cloud Build trigger。
- [x] [backend] 新增鉴权 release list/detail/sync endpoints 和错误响应。
- [x] [backend] 保留 legacy `pipeline_components` CRUD，兼容现有 custom/system rows。
- [x] [Frontend] 新增 release API types 和 client。
- [x] [Frontend] 更新组件库页面，把 legacy `PipelineComponent` 和 generated `ComponentRelease` 融合成单一组件库行；保留 legacy CRUD，并支持同步 release manifest、校验、展示 release label/status/channel/technical details。
- [x] [Frontend] 组件库按 source ref type 展示线上/测试/分支/PR 版本标识，技术详情保留 repo/ref/commit/digest。
- [x] [Frontend] 组件库支持搜索范围下拉框（智能/按任务/按 commit/按版本）和版本类型下拉框；按 task 搜索时自动展开版本，按 commit 搜索时收敛到匹配 release。
- [x] [Frontend] 在组件库行、展开版本表和版本详情展示 8 位镜像 ID。
- [ ] [Frontend] 新增高级 create-test-release 流程，限定为“已选 task + commit hash”。
- [ ] [Frontend] 更新 pipeline palette，默认列出/选择 ready releases，不要求用户理解 raw image。
- [ ] [Frontend] 更新 node data 和 pipeline JSON 转换，保留 `componentReleaseId`、`releaseLabel`、`runtimeSnapshot`。
- [ ] [Frontend] 保持 legacy pipeline template/raw image node 可渲染、可运行。

## API Contract Sync
- [x] 更新 `api/openapi.yaml`，加入 release schemas 和 endpoints。
- [x] 更新 `docs/review/api-guide.md`，加入 release list、sync/import 示例。
- [x] 新增或更新 smoke script，覆盖 release ingest/list 和一个 validation failure path。
- [x] 更新 frontend API types，保持与 OpenAPI 一致。
- [x] 更新 public SDK endpoint/client/tests。

## Verification
- [x] [backend] 跑 pipeline component repository/usecase/handler 相关 targeted tests。
- [x] [backend] 跑 `go test ./...`。
- [x] [Frontend] 跑 component manager 相关 tests。
- [x] [Frontend] 跑 touched component test 对应的 `npm run test -- --run src/pages/ComponentManager.test.tsx`。
- [x] [all] 跑 backend/frontend/sdk Tier L 主要 checks；frontend full suite 存在非本次触碰旧失败，见 `decisions.md`。

## Deploy verification
- [x] [deploy] runtime code 实现后部署 backend preview pod，并用本地 frontend dev 对接。
- [x] [deploy] 在 dev UI 验证 release list/sync/detail；API 验证缺 digest release 不可选。

### Deploy record
| Target | Image / URL | Status |
|--------|-------------|--------|
| backend preview pod | `cyber-databrew-backend:manual-cyb1824-4db0fdf-3ce73f3397a9` | ready |
| preview API | `https://cyber-databrew-dev.cyberorigin.ai/preview/3ce73f3397a9/api` | `GET /healthz` 200 |
| local frontend | `http://127.0.0.1:5179/` | component release UI verified |
| screenshot | `deploy-verify-component-release-detail.png` | saved |
| screenshot | `deploy-verify-unified-component-library.png` | saved |

## Open Questions
- [x] Phase 1 release ingest 是直接读 Cloud Build/Artifact Registry，还是先读 generated manifest artifact？先接 generated manifest / 平台同步记录，自动扫描后续通过 provider 接入。
- [x] 当前所有 task 的 `component_id` 是否直接使用现有 `task_name`？Phase 1 默认使用 `task_name`，允许同步 payload 显式覆盖。
- [x] 缺失 inputs/outputs/params/mounts 的现有 task，是统一默认，还是标记“元数据不足但可测试”？inputs/outputs 默认 asset input/output；缺 digest/entrypoint/resources 不可选。
- [ ] Published release label 来自 Git tag、GitHub release，还是 DataBrew 手动 promotion？
