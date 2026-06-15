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
