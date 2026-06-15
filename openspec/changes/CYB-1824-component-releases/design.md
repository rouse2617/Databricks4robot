# Design — CYB-1824

## Architecture Context
- **Constraints**: `cyber-databrew` 是编排和控制平面 UI；`CyberOrigin2077/automated-processing-gcloud` 是算法源码 repo。现有 task 构建已经按 task 配置了 Cloud Build trigger，并把镜像推到 Artifact Registry。
- **Goals**: DataBrew 前端可以承接从 repo 到验证的一条龙流程；算法 repo 和 task authoring 在 Phase 1 不变；普通用户安全选择 task release；pipeline 固化不可变 runtime snapshot；复用现有构建基础设施。
- **Non-Goals**: 不替换 Cloud Build；不要求算法用户手写 release ID 或 digest；不要求新增 `component.yaml`；Phase 1 不做详细 schema contract。

## Affected Modules
- `backend/internal/models/pipeline_component.go` — 新增 release 和 runtime snapshot model。
- `backend/migrations/` — 新增 component release 持久化表。
- `backend/internal/postgres/pipeline_component_repo.go` — 保存和查询 releases。
- `backend/internal/usecase/pipeline_component` — 校验、导入、暴露 releases。
- `backend/internal/handlers/pipeline_component` — 新增 release API。
- `Frontend/src/api/pipelineComponentApi.ts` — 新增 release types 和接口。
- `Frontend/src/pages/ComponentManager.tsx` — 正常 UX 从“手填镜像注册表”调整为“组件 release library”。
- `Frontend/src/pages/PipelinePage.tsx` 和 `Frontend/src/components/pipeline` — 选择 release，并把 runtime 字段快照到 node。
- `Frontend/src/lib/pipelineContract.ts` — canvas node 转 pipeline JSON 时保留 release 引用。

## Architecture Decisions

### Decision 1: ComponentRelease 是用户选择的版本单位
- **Approach**: 新增 component release，包含用户可读 label、validation status、digest 固化 runtime snapshot、technical metadata。
- **Alternative**: 继续把 `pipeline_components` 当作可变 image 行，并让用户动态选择 tag。
- **Rationale**: release 能避免 tag 漂移，并给用户一个稳定、经过校验的版本选择。
- **Trade-off**: 需要新增持久化概念和 migration。
- **Rollback**: 在 release 选择稳定前保留 legacy `pipeline_components` 的读写行为。

### Decision 2: Phase 1 复用现有 task-config 和 Cloud Build 流程
- **Approach**: 从现有 task config 和构建元数据中解析 component identity、command、resource hints。
- **Alternative**: 在实现 release 选择前，先要求所有 task 新增 `component.yaml`。
- **Rationale**: 算法 repo 已经有 task config 和 Cloud Build trigger；新增强制 authoring 文件会拖慢落地。
- **Trade-off**: Phase 1 的元数据表达能力不如专门设计的 component definition。
- **Rollback**: 关闭 release ingest，回退到现有 component registry 行。

### Decision 3: DataBrew 前端可以大改，但算法 task 不改
- **Approach**: DataBrew UI 提供 repo/task 同步、校验结果、release 状态、技术详情等一条龙体验；算法 repo 仍使用现有 `task-config.yaml`、`cloudbuild.yaml`、Dockerfile 和 Cloud Build trigger。
- **Alternative**: 要求算法同学先按新规范补齐 `component.yaml`、schema、params、mounts 后再接入。
- **Rationale**: 先吃现有 task 的真实结构，可以最快验证 DataBrew release 模型，并避免让算法同学承担平台元数据维护成本。
- **Trade-off**: 对缺失 inputs/outputs/params/mounts 的 task，Phase 1 只能使用默认值或标记“元数据不足”，不能假装有完整 contract。

### Decision 4: 普通选择入口隐藏平台元数据
- **Approach**: 正常 UI 只展示 task、release label、channel、validation status，技术细节折叠展示。
- **Alternative**: 让用户输入 repo、task path、commit、image digest、build ID。
- **Rationale**: 手写平台元数据容易填错，也很难让非平台用户判断真假。
- **Risk**: 高级算法用户仍然需要基于 commit 做测试。
- **Mitigation**: 提供一个 scoped 到已选 task 的“从 commit 创建测试 release”高级流程。

## Data Flow

```text
automated-processing-gcloud/tasks/<task>/task-config.yaml
        +
Cloud Build trigger / build result / Artifact Registry digest
        ↓
DataBrew 前端触发/查看同步，后端 ingest 或 test-release creation
        ↓
pipeline_component_releases
        ↓
Component Library 和 Pipeline Designer 的 release selector
        ↓
Pipeline template node:
  componentId + componentReleaseId + releaseLabel + runtimeSnapshot
        ↓
Transpiler 使用 runtimeSnapshot.image@sha256
```

## Data Model Changes
- **Table**: `pipeline_component_releases`
- **Change**: 新 release 行，包含 generated release ID、component identity、release label、channel、status、selectable、validation diagnostics、runtime snapshot、technical metadata、timestamps。
- **Migration**: 在 `backend/migrations/` 新增 migration。

## Risks / Trade-offs

| 风险 | 影响 | 缓解 |
|------|------|------|
| 老 build 的元数据不完整 | 一些历史版本无法 selectable | 标记为 failed/unselectable，并展示清晰 diagnostics |
| 现有 template 只保存 raw image 字段 | 新旧 template shape 混用 | 支持 legacy node，新选 release 才强制 snapshot |
| Cloud Build trigger 映射变化 | test-release 创建可能失败 | 按 `_TASK_NAME` 和 `filename` 解析，不只依赖 trigger name |
| 用户误把 test release 当 published release | shared template 可能用错版本 | 明确 channel 标签，默认只展示 ready 且非 deprecated 的 release |
