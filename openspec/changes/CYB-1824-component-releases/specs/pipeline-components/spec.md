## ADDED Requirements

### Requirement: Phase 1 不要求算法 task 修改文件
The system SHALL ingest existing algorithm task metadata without requiring algorithm users to add new component definition files in Phase 1.

**Priority**: P0 (Critical)
**Rationale**: 当前目标是让 DataBrew 先接住现有 task，避免把平台元数据维护成本转嫁给算法同学。

#### Scenario: 现有 task 可被同步
- **Given** 算法 repo 中存在一个已有 task，且包含现有 `task-config.yaml` 和 `cloudbuild.yaml`
- **When** DataBrew 同步该 task
- **Then** DataBrew 使用现有 task metadata 生成 component release 候选记录
- **And** 不要求该 task 新增 `component.yaml`

#### Scenario: 现有 task 元数据不完整
- **Given** 现有 task 缺少 inputs、outputs、params 或 mounts 声明
- **When** DataBrew 校验该 task release
- **Then** DataBrew 使用 Phase 1 默认值或标记元数据不足
- **And** DataBrew 不让用户手工补填 image digest、source path 或 release ID

### Requirement: Component release 作为正常可选组件版本
The system SHALL present validated component releases as the normal version choices for pipeline authoring.

**Priority**: P0 (Critical)
**Rationale**: 用户需要稳定的 task 版本选择，不应该手动选择 image tag 或 digest。

#### Scenario: 用户选择 ready release
- **Given** 某个 task 至少有一个 ready 且 selectable 的 release
- **When** 用户把该 task 添加到 pipeline
- **Then** 用户可以通过 label 和 status 选择 release
- **And** pipeline node 保存所选 release 引用和不可变 runtime snapshot

#### Scenario: 非 ready release 不出现在普通选择入口
- **Given** 某个 task release validation 失败或仍在 building
- **When** 用户打开普通 pipeline component selector
- **Then** 该 release 不作为普通可选版本出现
- **And** 组件详情里仍可查看 diagnostics

### Requirement: Component release ingest 校验平台生成元数据
The system SHALL validate generated component release metadata before marking a release selectable.

**Priority**: P0 (Critical)
**Rationale**: identity、entrypoint、resource、digest 等元数据缺失会导致不安全或不可复现的 pipeline run。

#### Scenario: 生成 release 包含必要元数据
- **Given** generated release metadata 包含匹配的 component identity、source commit、entrypoint、minimal ports、resource hints、image digest
- **When** DataBrew ingest 该 release
- **Then** 该 release 可以被标记为 ready 且 selectable

#### Scenario: CI 推送 batch source manifest
- **Given** CI 在 build 成功后提交包含 batch `source` 和多个 release `items` 的 manifest
- **When** DataBrew ingest 该 manifest
- **Then** DataBrew 将 batch source 作为每个 item 的默认 repo/ref/commit/build metadata
- **And** item 级字段可以覆盖 batch source 默认值
- **And** CI token 只能用于 release ingest，不扩大普通 API 访问面

#### Scenario: 用户按版本线索搜索 release
- **Given** DataBrew 已 ingest 多个 component releases
- **When** 用户或 SDK 使用 `q` 搜索 release label、source commit、build id、image tag 或 digest
- **Then** DataBrew 只返回匹配的 release records
- **And** 普通选择器仍可叠加 `selectable=true`

#### Scenario: 用户在组件库按 task 或 commit 查找版本
- **Given** DataBrew 已 ingest 某个 task 的多个 component releases
- **When** 用户在组件库按 task 名称搜索
- **Then** 组件库展示该 task，并直接展开可选版本列表
- **When** 用户切换到按 commit 搜索并输入 commit 前缀
- **Then** 组件库只展示匹配该 commit 的 component release
- **And** 用户可以用版本类型下拉框筛选线上版本、测试版本、分支版本或 PR 预览

#### Scenario: 每个镜像版本有短 ID
- **Given** DataBrew 已 ingest 一个带 image digest 的 component release
- **When** 用户在组件库或版本详情查看该 release
- **Then** DataBrew 展示一个 8 位镜像 ID
- **And** 该镜像 ID 优先由 image digest 稳定生成
- **And** 不把 task name 当作镜像 ID

#### Scenario: Git tag 和 commit 版本可区分
- **Given** CI 提交的 release manifest 包含 source ref 或 source ref type
- **When** source 指向 Git tag
- **Then** DataBrew 将 release 标识为线上版本并归入 prod channel
- **When** source 指向 commit hash
- **Then** DataBrew 将 release 标识为测试版本并允许用户按 commit 搜索
- **And** repo、ref type、commit、image digest 仍位于 technical details 中

#### Scenario: 生成 release 缺少 digest
- **Given** generated release metadata 没有不可变 image digest
- **When** DataBrew ingest 该 release
- **Then** 该 release 被标记为 failed 或 unselectable
- **And** validation diagnostics 说明缺少 digest

### Requirement: 算法用户可以从 commit 创建测试 release
The system SHALL allow advanced users to create a test release for a selected task from a commit hash.

**Priority**: P1 (High)
**Rationale**: 算法用户需要测试 task commit，但不应要求普通 pipeline 用户管理 repo、path、digest 字段。

#### Scenario: 为已选 task 创建测试 release
- **Given** 算法用户选择了一个已知 task
- **When** 他们提交 commit hash 创建测试 release
- **Then** DataBrew 记录一个带 generated test label 的 building release
- **And** 该 release 只有在 validation 成功后才变成 selectable

#### Scenario: 未知 commit 不能创建 selectable release
- **Given** 算法用户提交未知或非法 commit hash
- **When** DataBrew 尝试创建测试 release
- **Then** 请求失败，或该 release 被标记为 failed
- **And** 普通 pipeline 用户不能选择它

## MODIFIED Requirements

### Requirement: Component registry page
- **Before**: The system SHALL provide a component registry page where users can create and edit components by entering image and runtime fields.
- **After**: The system SHALL provide a component library page where normal users inspect components and validated releases, while raw image/runtime editing is legacy or advanced-only.
- **Reason**: 普通用户应该选择安全 release，而不是手动维护平台元数据。

**Priority**: P1 (High)
**Rationale**: Component library 是用户选择 task/version 的主要入口，需要逐步从 raw image 表单迁移到 release selector。

#### Scenario: 普通组件库不要求填写 raw image
- **Given** 用户打开 component library
- **When** 用户查看某个 component
- **Then** 用户看到 task name、release labels、status、channel、owner、technical details
- **And** 用户不需要输入 image digest 或 source path 字段
