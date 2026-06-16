# Proposal — CYB-1824

## Why
Pipeline 用户需要安全地选择算法任务版本，但不应该手动处理 repo path、image tag、image digest 这类平台字段。当前组件注册表直接暴露 Docker 镜像信息，容易填错，也无法避免 tag 漂移。

## What Changes

### New Capabilities
- DataBrew 把算法 task 展示为组件，并展示经过校验的 release 版本。
- DataBrew 可以从现有 task 构建流程中导入平台生成的 component release 元数据。
- DataBrew 前端提供“从算法 repo 到 DataBrew”的一条龙同步、校验、展示流程。
- Pipeline 用户可以通过易读的版本标签和状态选择任务版本。
- 算法用户可以为某个 task 从指定 commit 创建测试版本。

### Modified Capabilities
- Pipeline 组件选择从“手填 image/tag”转向“选择 ready 的 component release”。
- Pipeline template 节点会保存所选 release 的不可变 runtime snapshot。
- 算法 repo 和现有 task authoring 方式保持不变：Phase 1 不要求新增 `component.yaml`，已有 task 有什么元数据就先用什么元数据。

## Impact
- **Affected code**: `backend/internal/models`, `backend/internal/repository`, `backend/internal/postgres`, `backend/internal/usecase/pipeline_component`, `backend/internal/handlers/pipeline_component`, `Frontend/src/api`, `Frontend/src/pages/ComponentManager.tsx`, `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/components/pipeline`, `Frontend/src/lib/pipelineContract.ts`
- **New APIs**: 在 pipeline component API 下新增 release 列表、release 同步/导入、测试 release 创建接口
- **Dependencies**: 需要读取 GCP Artifact Registry / Cloud Build 元数据；Phase 1 不要求用户手写新的 repo 元数据

## Scope
- **In scope**: component release 数据模型、从现有 repo/task-config/cloudbuild/registry 同步 release、digest 固化 runtime snapshot、ready/selectable 过滤、从 task + commit 创建测试版本、DataBrew UI 一条龙同步/校验/展示。
- **Out of scope**: 要求算法同学新增 `component.yaml`、完整 JSON Schema contract、根据 params 自动生成 UI 表单、运行时 output schema 校验、跨版本兼容性 diff、替换 Cloud Build trigger。

## Success Criteria
- [ ] 算法同学不需要修改现有 task 文件结构，也不需要新增 `component.yaml`。
- [ ] DataBrew 可以从现有 `tasks/<task>/task-config.yaml`、Cloud Build、Artifact Registry 同步并校验 release。
- [ ] 用户可以选择 task 和 ready 版本，不需要输入 repo、path、image tag、digest。
- [ ] 新保存的 pipeline template snapshot 会保存 release ID 和 digest 固化的 runtime snapshot。
- [ ] DataBrew 在普通选择入口隐藏非 ready release，但组件详情里仍能看到失败原因。
- [ ] 算法用户可以从 commit 创建或导入测试版本，并看到 Building / Ready / Failed 状态。
- [ ] 技术详情能展示 source commit、image digest、build ID、validation 诊断，但这些不是用户必填字段。

## Goals (SLO)
- **Latency**: 500 个 release 以内，component release 列表接口 p95 小于 500 ms。
- **Concurrency**: release sync/import 重复触发时保持幂等。
- **Quality**: Phase 1 对缺失 task identity、entrypoint、resources、digest 的 release 快速失败。
